package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/role"

	_ "github.com/go-sql-driver/mysql"
)

// This test exercises the existing ordinary NPC purchase persistence adapter,
// then reads the result through a new SQL connection and the production bag
// loader. It must NEVER run against a user's nineyin database.
func TestOrdinaryShopPurchasePersistenceRealMySQLReentry(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_SHOP_MYSQL_CI") != "1" {
		t.Skip("isolated shop MySQL CI only")
	}
	dsn := os.Getenv("JIUYIN_TEST_SHOP_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("isolated shop MySQL DSN missing")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var databaseName string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&databaseName); err != nil {
		t.Fatal(err)
	}
	if databaseName != "shop_atomic_ci" {
		t.Fatalf("refuse to touch database %q, require disposable shop_atomic_ci", databaseName)
	}
	for _, statement := range []string{
		`CREATE TABLE roles (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (
role_id BIGINT UNSIGNED NOT NULL, seq BIGINT NOT NULL, slot INT NOT NULL,
config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL,
view_id INT NOT NULL, name TEXT NULL, equip_type TEXT NULL, art_pack INT NULL,
hardiness INT NULL, max_hardiness INT NULL, PRIMARY KEY (role_id,seq)
) ENGINE=InnoDB`,
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY, snapshot JSON NOT NULL) ENGINE=InnoDB`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const id role.RoleID = 991001
	startingBag := []bagItem{{ConfigID: "ci_owned_before_purchase", ItemType: 1, Amount: 1, ViewID: 1, Slot: 1}}
	bought := bagItem{ConfigID: "ci_purchased_item", ItemType: 1, Amount: 2, ViewID: 1, Slot: 2}
	nextBag := append(append([]bagItem(nil), startingBag...), bought)
	startingWallet := currencySnapshot{Silver: 100}
	nextWallet := currencySnapshot{Silver: 70}
	startingJSON, err := json.Marshal(startingWallet)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO roles(role_id) VALUES (?)", id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", id, startingJSON); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id)
VALUES (?,0,1,'ci_owned_before_purchase',1,1,1)`, id); err != nil {
		t.Fatal(err)
	}
	bagStore := &mysqlBagStore{db: db}
	currencyStore := &mysqlCurrencyStore{db: db}
	if err := persistOrdinaryShopPurchase(bagStore, currencyStore, id, nextBag, startingBag, startingWallet, nextWallet); err != nil {
		t.Fatalf("ordinary purchase persistence failed: %v", err)
	}
	// A second independent SQL connection is analogous to a new server session;
	// this does not claim a real current-client LIVE/reconnect test.
	reopened, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	checkReentry := func() {
		t.Helper()
		loaded, exists := (&mysqlBagStore{db: reopened}).Load(id)
		if !exists || len(loaded) != 2 {
			t.Fatalf("reloaded bag exists=%t rows=%d, want 2", exists, len(loaded))
		}
		for index, expected := range nextBag {
			got := loaded[index]
			if got.ConfigID != expected.ConfigID || got.ItemType != expected.ItemType || got.Amount != expected.Amount || got.ViewID != expected.ViewID || got.Slot != expected.Slot {
				t.Fatalf("reloaded bag row %d: got=%+v want=%+v", index, got, expected)
			}
		}
		var walletJSON []byte
		if err := reopened.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", id).Scan(&walletJSON); err != nil {
			t.Fatal(err)
		}
		var loadedWallet currencySnapshot
		if err := json.Unmarshal(walletJSON, &loadedWallet); err != nil {
			t.Fatal(err)
		}
		if loadedWallet != nextWallet {
			t.Fatalf("reloaded wallet=%+v, want=%+v", loadedWallet, nextWallet)
		}
	}
	checkReentry()
	// A second purchase with an obsolete starting bag must not erase the item
	// already bought, even when its wallet snapshot is otherwise current.
	extra := bagItem{ConfigID: "ci_stale_attempt", ItemType: 1, Amount: 1, ViewID: 1, Slot: 3}
	if err := persistOrdinaryShopPurchase(bagStore, currencyStore, id,
		append(append([]bagItem(nil), startingBag...), extra), startingBag,
		nextWallet, currencySnapshot{Silver: 60}); err == nil || !strings.Contains(err.Error(), "bag changed") {
		t.Fatalf("obsolete bag purchase must be rejected: %v", err)
	}
	checkReentry()
	// Likewise an obsolete currency snapshot must not overwrite a committed
	// purchase even with an up-to-date bag snapshot.
	if err := persistOrdinaryShopPurchase(bagStore, currencyStore, id,
		append(append([]bagItem(nil), nextBag...), extra), nextBag,
		startingWallet, currencySnapshot{Silver: 60}); err == nil || !strings.Contains(err.Error(), "wallet changed") {
		t.Fatalf("obsolete wallet purchase must be rejected: %v", err)
	}
	checkReentry()
}
