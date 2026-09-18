package shopbuyatomic

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// This test intentionally cannot run on the user's nineyin database. CI must
// supply a disposable database named shop_atomic_ci and opt in explicitly.
func TestSaveCheckedRealMySQLReconnectAndRollback(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_SHOP_MYSQL_CI") != "1" {
		t.Skip("disposable MySQL CI not enabled")
	}
	dsn := os.Getenv("JIUYIN_TEST_SHOP_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("disposable MySQL DSN missing")
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
		t.Fatalf("refusing to touch non-CI database %q", databaseName)
	}
	if _, err := db.Exec(`CREATE TABLE roles (
role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY
) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE role_bag_items (
role_id BIGINT UNSIGNED NOT NULL, seq BIGINT NOT NULL, slot INT NOT NULL,
config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL,
view_id INT NOT NULL, name TEXT NULL, equip_type TEXT NULL, art_pack INT NULL,
hardiness INT NULL, max_hardiness INT NULL, PRIMARY KEY (role_id,seq)
) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE role_currency (
role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY, snapshot JSON NOT NULL
) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	const id uint64 = 880001
	if _, err := db.Exec(`INSERT INTO roles(role_id) VALUES (?)`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)`, id, []byte(`{"silver":100,"gold":0}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id) VALUES (?,0,1,'old_item',100,1,1)`, id); err != nil {
		t.Fatal(err)
	}
	before := []byte(`{"gold":0,"silver":100}`)
	after := []byte(`{"silver":70,"gold":0}`)
	rows := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 1, ViewID: 1}, {Slot: 2, ConfigID: "new_item", ItemType: 100, Amount: 2, ViewID: 1}}
	if err := SaveChecked(db, id, rows, before, after); err != nil {
		t.Fatalf("commit purchase: %v", err)
	}
	// A second independent connection verifies durable reloading, not actor RAM.
	reopened, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	assertSaved := func(wantSilver int64) {
		t.Helper()
		var raw []byte
		if err := reopened.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", id).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var wallet map[string]int64
		if err := json.Unmarshal(raw, &wallet); err != nil {
			t.Fatal(err)
		}
		if wallet["silver"] != wantSilver {
			t.Fatalf("saved silver=%d want=%d", wallet["silver"], wantSilver)
		}
		var count int
		if err := reopened.QueryRow("SELECT COUNT(*) FROM role_bag_items WHERE role_id=?", id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("saved bag count=%d want=2", count)
		}
		var item string
		if err := reopened.QueryRow("SELECT config_id FROM role_bag_items WHERE role_id=? AND seq=1", id).Scan(&item); err != nil {
			t.Fatal(err)
		}
		if item != "new_item" {
			t.Fatalf("saved second item=%q", item)
		}
	}
	assertSaved(70)
	if err := SaveChecked(db, id, []Row{{Slot: 1, ConfigID: "stale_item", Amount: 1}}, before, []byte(`{"silver":60,"gold":0}`)); err == nil {
		t.Fatal("stale wallet unexpectedly committed")
	}
	assertSaved(70)
	// A constraint on this disposable test table forces wallet INSERT to fail
	// after the bag DELETE/INSERT, without requiring privileged CREATE TRIGGER.
	if _, err := db.Exec(`ALTER TABLE role_currency ADD CONSTRAINT shopci_block_60
CHECK (CAST(JSON_UNQUOTE(JSON_EXTRACT(snapshot, '$.silver')) AS SIGNED) <> 60)`); err != nil {
		t.Fatal(err)
	}
	if err := SaveChecked(db, id, []Row{{Slot: 1, ConfigID: "rolled_back_item", Amount: 1}}, after, []byte(`{"silver":60,"gold":0}`)); err == nil {
		t.Fatal("wallet CHECK failure unexpectedly committed")
	}
	assertSaved(70)

	// Reproduce a bag-only writer racing a purchase on TWO independent SQL
	// connections. The legacy writer owns the role lock until it commits;
	// the purchase must then see its changed bag and reject stale actor data.
	const concurrentRole uint64 = 880002
	if _, err := db.Exec("INSERT INTO roles(role_id) VALUES (?)", concurrentRole); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", concurrentRole, before); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id)
VALUES (?,0,1,'old_item',100,1,1)`, concurrentRole); err != nil {
		t.Fatal(err)
	}
	writer, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Rollback()
	if err := LockRoleForBagWrite(context.Background(), writer, concurrentRole); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec("UPDATE role_bag_items SET amount=2 WHERE role_id=?", concurrentRole); err != nil {
		t.Fatal(err)
	}
	purchaseDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer purchaseDB.Close()
	started := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		close(started)
		finished <- SaveCheckedBag(purchaseDB, concurrentRole,
			[]Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 1, ViewID: 1}, {Slot: 2, ConfigID: "new_item", ItemType: 100, Amount: 1, ViewID: 1}},
			[]Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 1, ViewID: 1}},
			before, []byte(`{"silver":90,"gold":0}`))
	}()
	<-started
	select {
	case premature := <-finished:
		t.Fatalf("purchase completed while another writer held role lock: %v", premature)
	case <-time.After(150 * time.Millisecond):
	}
	if err := writer.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case purchaseErr := <-finished:
		if purchaseErr == nil || !strings.Contains(purchaseErr.Error(), "bag changed") {
			t.Fatalf("stale purchase after competing writer: %v", purchaseErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("purchase did not finish after competing writer committed")
	}
	var savedAmount int
	if err := reopened.QueryRow("SELECT amount FROM role_bag_items WHERE role_id=?", concurrentRole).Scan(&savedAmount); err != nil {
		t.Fatal(err)
	}
	if savedAmount != 2 {
		t.Fatalf("competing writer item amount=%d, want 2", savedAmount)
	}
	var savedWallet []byte
	if err := reopened.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", concurrentRole).Scan(&savedWallet); err != nil {
		t.Fatal(err)
	}
	var wallet map[string]int64
	if err := json.Unmarshal(savedWallet, &wallet); err != nil {
		t.Fatal(err)
	}
	if wallet["silver"] != 100 {
		t.Fatalf("rejected purchase altered silver=%d", wallet["silver"])
	}

	// Reverse order: a purchase commits first; an older bag-only snapshot
	// must NOT delete the purchased item or overwrite the committed bag.
	const reverseRole uint64 = 880003
	if _, err := db.Exec("INSERT INTO roles(role_id) VALUES (?)", reverseRole); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", reverseRole, before); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id)
VALUES (?,0,1,'old_item',100,1,1)`, reverseRole); err != nil {
		t.Fatal(err)
	}
	original := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 1, ViewID: 1}}
	purchased := []Row{original[0], {Slot: 2, ConfigID: "new_item", ItemType: 100, Amount: 1, ViewID: 1}}
	if err := SaveCheckedBag(purchaseDB, reverseRole, purchased, original, before, after); err != nil {
		t.Fatalf("reverse purchase commit: %v", err)
	}
	staleWriter := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 2, ViewID: 1}}
	if err := SaveBagChecked(db, reverseRole, staleWriter, original); err == nil || !strings.Contains(err.Error(), "bag changed") {
		t.Fatalf("stale bag-only writer after committed purchase was not rejected: %v", err)
	}
	var persistedCount int
	if err := reopened.QueryRow("SELECT COUNT(*) FROM role_bag_items WHERE role_id=?", reverseRole).Scan(&persistedCount); err != nil {
		t.Fatal(err)
	}
	if persistedCount != 2 {
		t.Fatalf("stale writer erased purchased bag row: count=%d", persistedCount)
	}
	var bought string
	if err := reopened.QueryRow("SELECT config_id FROM role_bag_items WHERE role_id=? AND seq=1", reverseRole).Scan(&bought); err != nil {
		t.Fatal(err)
	}
	if bought != "new_item" {
		t.Fatalf("stale writer erased purchased item: %q", bought)
	}
	var reverseWalletJSON []byte
	if err := reopened.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", reverseRole).Scan(&reverseWalletJSON); err != nil {
		t.Fatal(err)
	}
	var reverseWallet map[string]int64
	if err := json.Unmarshal(reverseWalletJSON, &reverseWallet); err != nil {
		t.Fatal(err)
	}
	if reverseWallet["silver"] != 70 {
		t.Fatalf("bag-only rejection changed committed wallet: %d", reverseWallet["silver"])
	}
	// A writer that actually read the new bag may still update it normally.
	fresh := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 2, ViewID: 1}, purchased[1]}
	if err := SaveBagChecked(db, reverseRole, fresh, purchased); err != nil {
		t.Fatalf("fresh bag-only writer rejected: %v", err)
	}
	var freshAmount int
	if err := reopened.QueryRow("SELECT amount FROM role_bag_items WHERE role_id=? AND seq=0", reverseRole).Scan(&freshAmount); err != nil {
		t.Fatal(err)
	}
	if freshAmount != 2 {
		t.Fatalf("fresh writer amount=%d, want 2", freshAmount)
	}

	// An older wallet snapshot may omit the actor's three zero-valued
	// currencies. Check the actual purchase transaction and reload through
	// another MySQL connection, not just a mocked JSON comparison.
	const sparseWalletRole uint64 = 880004
	if _, err := db.Exec("INSERT INTO roles(role_id) VALUES (?)", sparseWalletRole); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", sparseWalletRole, []byte(`{"silver":100}`)); err != nil {
		t.Fatal(err)
	}
	sparseBefore := []byte(`{"silver":100,"gold":0,"silver_card":0,"silver_ticket":0}`)
	sparseAfter := []byte(`{"silver":90,"gold":0,"silver_card":0,"silver_ticket":0}`)
	sparseItem := Row{Slot: 1, ConfigID: "sparse_wallet_purchase", ItemType: 100, Amount: 1, ViewID: 1}
	if err := SaveCheckedBag(db, sparseWalletRole, []Row{sparseItem}, nil, sparseBefore, sparseAfter); err != nil {
		t.Fatalf("purchase using sparse legacy wallet: %v", err)
	}
	var savedSparseWallet []byte
	if err := reopened.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", sparseWalletRole).Scan(&savedSparseWallet); err != nil {
		t.Fatal(err)
	}
	var reloadedSparseWallet map[string]int64
	if err := json.Unmarshal(savedSparseWallet, &reloadedSparseWallet); err != nil {
		t.Fatal(err)
	}
	if len(reloadedSparseWallet) != 4 || reloadedSparseWallet["silver"] != 90 || reloadedSparseWallet["gold"] != 0 || reloadedSparseWallet["silver_card"] != 0 || reloadedSparseWallet["silver_ticket"] != 0 {
		t.Fatalf("reloaded sparse purchase wallet=%s, want equivalent %s", savedSparseWallet, sparseAfter)
	}
	var sparseItemID string
	if err := reopened.QueryRow("SELECT config_id FROM role_bag_items WHERE role_id=?", sparseWalletRole).Scan(&sparseItemID); err != nil {
		t.Fatal(err)
	}
	if sparseItemID != sparseItem.ConfigID {
		t.Fatalf("reloaded sparse purchase item=%q, want %q", sparseItemID, sparseItem.ConfigID)
	}
}
