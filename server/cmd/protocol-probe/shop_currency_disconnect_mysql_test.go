package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuyatomic"
	_ "github.com/go-sql-driver/mysql"
)

// This test may only run against the disposable shop_atomic_ci database.
// It exercises the existing normal purchase transaction, a second independent
// connection's newer wallet, then the original actor's disconnect persistence.
func TestOrdinaryShopPurchaseLogoutPreservesNewerMySQLWallet(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_SHOP_MYSQL_CI") != "1" {
		t.Skip("isolated shop MySQL CI not enabled")
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
		t.Fatalf("refusing to touch non-CI database %q", databaseName)
	}
	for _, statement := range []string{
		`CREATE TABLE roles (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (
role_id BIGINT UNSIGNED NOT NULL, seq BIGINT NOT NULL, slot INT NOT NULL,
config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL,
view_id INT NOT NULL, name TEXT NULL, equip_type TEXT NULL, art_pack INT NULL,
hardiness INT NULL, max_hardiness INT NULL, PRIMARY KEY(role_id,seq)) ENGINE=InnoDB`,
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
snapshot JSON NOT NULL) ENGINE=InnoDB`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const roleID uint64 = 991170
	before := currencySnapshot{Silver: 100}
	purchased := currencySnapshot{Silver: 70}
	newer := currencySnapshot{Silver: 65}
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	purchasedJSON, err := json.Marshal(purchased)
	if err != nil {
		t.Fatal(err)
	}
	newerJSON, err := json.Marshal(newer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO roles(role_id) VALUES (?)`, roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)`, roleID, beforeJSON); err != nil {
		t.Fatal(err)
	}
	purchaseBag := []shopbuyatomic.Row{{Slot: 1, ConfigID: "ordinary_shop_purchase", ItemType: 100, Amount: 1, ViewID: 1}}
	if err := shopbuyatomic.SaveCheckedBag(db, roleID, purchaseBag, nil, beforeJSON, purchasedJSON); err != nil {
		t.Fatalf("persist ordinary purchase: %v", err)
	}

	independent, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer independent.Close()
	if _, err := independent.Exec(`UPDATE role_currency SET snapshot=? WHERE role_id=?`, newerJSON, roleID); err != nil {
		t.Fatalf("independent wallet update: %v", err)
	}
	actor := &playerActor{}
	purchased.toActor(actor)
	actor.markOrdinaryShopWalletCommitted(purchased)
	if err := persistDeferredShopCurrency(&mysqlCurrencyStore{db: db}, role.RoleID(roleID), actor); err != nil {
		t.Fatalf("disconnect after committed purchase: %v", err)
	}
	var raw []byte
	if err := independent.QueryRow(`SELECT snapshot FROM role_currency WHERE role_id=?`, roleID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var saved currencySnapshot
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved != newer {
		t.Fatalf("stale purchase logout clobbered newer wallet: got=%+v want=%+v", saved, newer)
	}
	var configID string
	if err := independent.QueryRow(`SELECT config_id FROM role_bag_items WHERE role_id=?`, roleID).Scan(&configID); err != nil {
		t.Fatal(err)
	}
	if configID != purchaseBag[0].ConfigID {
		t.Fatalf("purchase bag lost on logout: %q", configID)
	}
}
