package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuyatomic"
)

// Runs exclusively on the disposable shop_atomic_ci database. The second
// connection changes the wallet between normal NPC purchases. No user data
// or client packet capture is involved.
func TestOrdinaryShopWalletConflictResyncRealMySQL(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_SHOP_MYSQL_CI") != "1" {
		t.Skip("disposable MySQL shop CI not enabled")
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
	var schema string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&schema); err != nil || schema != "shop_atomic_ci" {
		t.Fatalf("refusing non-disposable database %q: %v", schema, err)
	}
	for _, statement := range []string{
		`CREATE TABLE roles (role_id BIGINT UNSIGNED PRIMARY KEY) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (role_id BIGINT UNSIGNED NOT NULL, seq BIGINT NOT NULL,
slot INT NOT NULL, config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL,
amount INT NOT NULL, view_id INT NOT NULL, name TEXT NULL, equip_type TEXT NULL,
art_pack INT NULL, hardiness INT NULL, max_hardiness INT NULL,
PRIMARY KEY (role_id,seq)) ENGINE=InnoDB`,
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED PRIMARY KEY,
snapshot JSON NOT NULL) ENGINE=InnoDB`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const id uint64 = 994011
	before := currencySnapshot{Silver: 100}
	purchased := currencySnapshot{Silver: 70}
	latest := currencySnapshot{Silver: 55}
	encode := func(value currencySnapshot) []byte {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	if _, err := db.Exec("INSERT INTO roles(role_id) VALUES (?)", id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", id, encode(before)); err != nil {
		t.Fatal(err)
	}
	firstBag := []shopbuyatomic.Row{{Slot: 1, ConfigID: "shop_purchase_kept", ItemType: 100, Amount: 1, ViewID: 1}}
	if err := shopbuyatomic.SaveCheckedBag(db, id, firstBag, nil, encode(before), encode(purchased)); err != nil {
		t.Fatalf("first normal purchase: %v", err)
	}
	other, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err := other.Exec("UPDATE role_currency SET snapshot = ? WHERE role_id = ?", encode(latest), id); err != nil {
		t.Fatal(err)
	}
	failedBag := append(append([]shopbuyatomic.Row(nil), firstBag...), shopbuyatomic.Row{
		Slot: 2, ConfigID: "uncommitted_purchase", ItemType: 100, Amount: 1, ViewID: 1,
	})
	err = shopbuyatomic.SaveCheckedBag(db, id, failedBag, firstBag, encode(purchased), encode(currencySnapshot{Silver: 50}))
	if !errors.Is(err, shopbuyatomic.ErrWalletChanged) {
		t.Fatalf("stale purchase did not return the typed wallet conflict: %v", err)
	}
	player := &playerActor{}
	purchased.toActor(player)
	player.markOrdinaryShopWalletCommitted(purchased)
	capture := &shopWalletFrameCapture{}
	if err := resyncOrdinaryShopWalletAfterConflict(capture, player, &mysqlCurrencyStore{db: db}, role.RoleID(id), purchased); err != nil {
		t.Fatalf("refresh after rejected purchase: %v", err)
	}
	var got currencySnapshot
	got.fromActor(player)
	if got != latest || !player.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatalf("actor wallet was not refreshed: got=%+v want=%+v", got, latest)
	}
	wantFrame, err := ordinaryShopCurrencyFrame(latest)
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.frames) != 1 || !bytes.Equal(capture.frames[0], wantFrame) {
		t.Fatalf("wrong client currency update frame: got=%x want=%x", capture.frames, wantFrame)
	}
	if err := persistDeferredShopCurrency(&mysqlCurrencyStore{db: db}, role.RoleID(id), player); err != nil {
		t.Fatalf("refreshed session logout: %v", err)
	}
	var raw []byte
	if err := other.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var saved currencySnapshot
	if err := json.Unmarshal(raw, &saved); err != nil || saved != latest {
		t.Fatalf("latest wallet was overwritten: %+v %v", saved, err)
	}
	var itemCount int
	var config string
	if err := other.QueryRow("SELECT COUNT(*) FROM role_bag_items WHERE role_id=?", id).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if err := other.QueryRow("SELECT config_id FROM role_bag_items WHERE role_id=?", id).Scan(&config); err != nil {
		t.Fatal(err)
	}
	if itemCount != 1 || config != firstBag[0].ConfigID {
		t.Fatalf("uncommitted reward appeared or purchased item vanished: count=%d item=%q", itemCount, config)
	}
}
