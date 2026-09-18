package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuyatomic"
)

// This test creates tables ONLY inside the disposable shop_atomic_ci database.
func TestOrdinaryShopBagConflictResyncRealMySQL(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_SHOP_MYSQL_CI") != "1" {
		t.Skip("disposable shop MySQL CI not enabled")
	}
	dsn := os.Getenv("JIUYIN_TEST_SHOP_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("disposable shop MySQL DSN missing")
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
		t.Fatalf("refusing non-disposable database %q", databaseName)
	}
	for _, statement := range []string{
		`CREATE TABLE roles (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (
role_id BIGINT UNSIGNED NOT NULL, seq BIGINT NOT NULL, slot INT NOT NULL,
config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL,
view_id INT NOT NULL, name TEXT NULL, equip_type TEXT NULL, art_pack INT NULL,
hardiness INT NULL, max_hardiness INT NULL, PRIMARY KEY(role_id,seq)) ENGINE=InnoDB`,
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY, snapshot JSON NOT NULL) ENGINE=InnoDB`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const roleID uint64 = 991191
	before := currencySnapshot{Silver: 100}
	after := currencySnapshot{Silver: 90}
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO roles(role_id) VALUES (?)`, roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)`, roleID, beforeJSON); err != nil {
		t.Fatal(err)
	}
	original := []bagItem{{ConfigID: "shop_original", ItemType: 100, Amount: 2, ViewID: 1, Slot: 4}}
	originalRows := ordinaryShopBagRows(original)
	for seq, row := range originalRows {
		if _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id) VALUES (?,?,?,?,?,?,?)`, roleID, seq, row.Slot, row.ConfigID, row.ItemType, row.Amount, row.ViewID); err != nil {
			t.Fatal(err)
		}
	}
	independent, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer independent.Close()
	newer := append(append([]bagItem(nil), original...), bagItem{ConfigID: "newer_purchase", ItemType: 100, Amount: 1, ViewID: 1, Slot: 5})
	newerRows := ordinaryShopBagRows(newer)
	if err := shopbuyatomic.SaveBagChecked(independent, roleID, newerRows, originalRows); err != nil {
		t.Fatalf("independent bag writer: %v", err)
	}
	failedReward := bagItem{ConfigID: "failed_reward", ItemType: 100, Amount: 1, ViewID: 1, Slot: 5}
	failedRows := ordinaryShopBagRows(append(append([]bagItem(nil), original...), failedReward))
	if err := shopbuyatomic.SaveCheckedBag(db, roleID, failedRows, originalRows, beforeJSON, afterJSON); !errors.Is(err, shopbuyatomic.ErrBagChanged) {
		t.Fatalf("stale purchase must return bag conflict sentinel, got %v", err)
	}
	actor := &playerActor{}
	actor.restoreBag(original)
	capture := &ordinaryShopBagConflictCapture{}
	if err := resyncOrdinaryShopBagAfterConflict(capture, actor, &mysqlBagStore{db: db}, role.RoleID(roleID), original); err != nil {
		t.Fatalf("reconcile stale purchase bag: %v", err)
	}
	if !reflect.DeepEqual(actor.bagSnapshot(), newer) {
		t.Fatalf("actor bag not DB state: %+v", actor.bagSnapshot())
	}
	if len(capture.frames) != 3 {
		t.Fatalf("view replay contains %d frames, want 3", len(capture.frames))
	}
	var walletJSON []byte
	if err := independent.QueryRow(`SELECT snapshot FROM role_currency WHERE role_id=?`, roleID).Scan(&walletJSON); err != nil {
		t.Fatal(err)
	}
	var persistedWallet currencySnapshot
	if err := json.Unmarshal(walletJSON, &persistedWallet); err != nil {
		t.Fatal(err)
	}
	if persistedWallet != before {
		t.Fatalf("rejected buy charged currency: %+v", persistedWallet)
	}
	persistedBag, ok := (&mysqlBagStore{db: independent}).Load(role.RoleID(roleID))
	if !ok || !reflect.DeepEqual(persistedBag, newer) {
		t.Fatalf("independent reconnect bag changed: %+v ok=%t", persistedBag, ok)
	}
	// A fresh, explicitly requested purchase can proceed with the reloaded
	// state; there was no automatic reward from reconciliation itself.
	freshReward := bagItem{ConfigID: "fresh_explicit_reward", ItemType: 100, Amount: 1, ViewID: 1, Slot: 6}
	fresh := append(append([]bagItem(nil), newer...), freshReward)
	if err := shopbuyatomic.SaveCheckedBag(db, roleID, ordinaryShopBagRows(fresh), newerRows, beforeJSON, afterJSON); err != nil {
		t.Fatalf("fresh explicit purchase: %v", err)
	}
	persistedBag, ok = (&mysqlBagStore{db: independent}).Load(role.RoleID(roleID))
	if !ok || !reflect.DeepEqual(persistedBag, fresh) {
		t.Fatalf("explicit purchase did not persist once: %+v", persistedBag)
	}
	if err := independent.QueryRow(`SELECT snapshot FROM role_currency WHERE role_id=?`, roleID).Scan(&walletJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(walletJSON, &persistedWallet); err != nil {
		t.Fatal(err)
	}
	if persistedWallet != after {
		t.Fatalf("explicit purchase wallet=%+v want=%+v", persistedWallet, after)
	}
}
