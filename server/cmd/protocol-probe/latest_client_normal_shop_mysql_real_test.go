//go:build mysql_integration

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// This test MUST target the dedicated disposable CI schema, never a player's DB.
// It exercises real InnoDB transactions using the same bag/currency columns as
// the existing mysqlBagStore/mysqlCurrencyStore, not the game migration runner.
func TestNormalShopMySQLRealAtomicity(t *testing.T) {
	dsn := os.Getenv("NINEYIN_STAGE51_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("dedicated disposable MySQL DSN not configured")
	}
	if os.Getenv("NINEYIN_STAGE51_DISPOSABLE") != "YES_DROP_STAGE51_TABLES" {
		t.Fatal("refusing destructive fixture without disposable-test opt-in")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(8)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("real MySQL ping: %v", err)
	}
	var database string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err != nil {
		t.Fatal(err)
	}
	if database != "jiuyin_stage51" {
		t.Fatalf("refusing to modify non-disposable database %q", database)
	}
	ddl := []string{
		"DROP TABLE IF EXISTS role_bag_items",
		"DROP TABLE IF EXISTS role_currency",
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY, snapshot JSON NOT NULL) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (
            role_id BIGINT UNSIGNED NOT NULL, seq INT NOT NULL, slot INT NOT NULL,
            config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL,
            view_id INT NOT NULL, name VARCHAR(255) NULL, equip_type VARCHAR(64) NULL,
            art_pack INT NULL, hardiness INT NULL, max_hardiness INT NULL,
            PRIMARY KEY(role_id,seq), UNIQUE KEY uq_stage51_slot(role_id,slot)
        ) ENGINE=InnoDB`,
	}
	for _, statement := range ddl {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("disposable fixture DDL: %v", err)
		}
	}
	t.Cleanup(func() {
		// Only this named disposable database is allowed; no production tables.
		_, _ = db.Exec("DROP TABLE IF EXISTS role_bag_items")
		_, _ = db.Exec("DROP TABLE IF EXISTS role_currency")
	})
	const roleID uint64 = 51
	before := normalShopPersistedCurrency{Silver: 100, Gold: 10, SilverCard: 5, SilverTicket: 0}
	after := before
	after.Silver = 80
	original := normalShopPersistedBagRow{ConfigID: "stage51_existing", ItemType: 1, Amount: 1, ViewID: 1, Slot: 1}
	reward := normalShopPersistedBagRow{ConfigID: "stage51_reward", ItemType: 1, Amount: 1, ViewID: 1, Slot: 2}
	initialBag := []normalShopPersistedBagRow{original}
	boughtBag := []normalShopPersistedBagRow{original, reward}
	seed := func(t *testing.T, currencyPresent bool) {
		t.Helper()
		if _, err := db.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, "DELETE FROM role_currency WHERE role_id = ?", roleID); err != nil {
			t.Fatal(err)
		}
		if currencyPresent {
			encoded, _ := json.Marshal(before)
			if _, err := db.ExecContext(ctx, "INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", roleID, encoded); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id) VALUES (?,?,?,?,?,?,?)`, roleID, 0, 1, original.ConfigID, original.ItemType, original.Amount, original.ViewID); err != nil {
			t.Fatal(err)
		}
	}
	assertSaved := func(t *testing.T, expectedCurrency *normalShopPersistedCurrency, expectedBag []normalShopPersistedBagRow) {
		t.Helper()
		// New pool = independent post-commit readback, not an actor snapshot.
		readDB, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer readDB.Close()
		var encoded []byte
		err = readDB.QueryRowContext(ctx, "SELECT snapshot FROM role_currency WHERE role_id=?", roleID).Scan(&encoded)
		if expectedCurrency == nil {
			if err != sql.ErrNoRows {
				t.Fatalf("expected missing currency row: %v", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			var actual normalShopPersistedCurrency
			if err := json.Unmarshal(encoded, &actual); err != nil {
				t.Fatal(err)
			}
			if actual != *expectedCurrency {
				t.Fatalf("persisted currency got %+v want %+v", actual, *expectedCurrency)
			}
		}
		rows, err := readDB.QueryContext(ctx, "SELECT config_id,item_type,amount,view_id,COALESCE(slot,0) FROM role_bag_items WHERE role_id=? ORDER BY seq", roleID)
		if err != nil {
			t.Fatal(err)
		}
		var actual []normalShopPersistedBagRow
		for rows.Next() {
			var item normalShopPersistedBagRow
			if err := rows.Scan(&item.ConfigID, &item.ItemType, &item.Amount, &item.ViewID, &item.Slot); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			actual = append(actual, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		rows.Close()
		if !reflect.DeepEqual(actual, expectedBag) {
			t.Fatalf("persisted bag got %+v want %+v", actual, expectedBag)
		}
	}
	commit := func(preCurrency, postCurrency normalShopPersistedCurrency, preBag, postBag []normalShopPersistedBagRow) error {
		return normalShopCommitMySQLSnapshots(ctx, db, roleID, preCurrency, postCurrency, preBag, postBag)
	}
	t.Run("success_independent_readback", func(t *testing.T) {
		seed(t, true)
		if err := commit(before, after, initialBag, boughtBag); err != nil {
			t.Fatal(err)
		}
		assertSaved(t, &after, boughtBag)
	})
	t.Run("stale_currency_rejected_without_write", func(t *testing.T) {
		seed(t, true)
		stale := before
		stale.Silver++
		if err := commit(stale, after, initialBag, boughtBag); err == nil {
			t.Fatal("stale currency accepted")
		}
		assertSaved(t, &before, initialBag)
	})
	t.Run("stale_bag_rejected_without_write", func(t *testing.T) {
		seed(t, true)
		if err := commit(before, after, nil, boughtBag); err == nil {
			t.Fatal("stale bag accepted")
		}
		assertSaved(t, &before, initialBag)
	})
	t.Run("insert_error_rolls_back_both", func(t *testing.T) {
		seed(t, true)
		badReward := reward
		badReward.Slot = 1 // deliberate unique-slot violation AFTER DELETE + first INSERT
		if err := commit(before, after, initialBag, []normalShopPersistedBagRow{original, badReward}); err == nil {
			t.Fatal("duplicate slot accepted")
		}
		assertSaved(t, &before, initialBag)
	})
	t.Run("currency_update_error_rolls_back_bag", func(t *testing.T) {
		seed(t, true)
		_, err := db.ExecContext(ctx, `CREATE TRIGGER stage51_reject_currency BEFORE UPDATE ON role_currency FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'stage51 deliberate currency update failure'`)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _, _ = db.Exec("DROP TRIGGER IF EXISTS stage51_reject_currency") }()
		if err := commit(before, after, initialBag, boughtBag); err == nil {
			t.Fatal("forced currency UPDATE error ignored")
		}
		assertSaved(t, &before, initialBag)
		if _, err := db.ExecContext(ctx, "DROP TRIGGER stage51_reject_currency"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("missing_currency_fails_closed", func(t *testing.T) {
		seed(t, false)
		if err := commit(before, after, initialBag, boughtBag); err == nil {
			t.Fatal("missing currency accepted")
		}
		assertSaved(t, nil, initialBag)
	})
	t.Run("two_competing_transactions_one_commit", func(t *testing.T) {
		seed(t, true)
		var wg sync.WaitGroup
		outcomes := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); outcomes <- commit(before, after, initialBag, boughtBag) }()
		}
		wg.Wait()
		close(outcomes)
		ok, fail := 0, 0
		for err := range outcomes {
			if err == nil {
				ok++
			} else {
				fail++
			}
		}
		if ok != 1 || fail != 1 {
			t.Fatalf("expected one commit, one stale reject; commits=%d rejects=%d", ok, fail)
		}
		assertSaved(t, &after, boughtBag)
	})
	t.Log(fmt.Sprintf("real InnoDB atomicity verified against disposable %s only; game runtime and player DB NOT tested", database))
}
