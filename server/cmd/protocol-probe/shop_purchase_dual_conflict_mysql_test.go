package main

import (
	"bytes"
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

// Creates tables only in the explicitly named disposable CI database.
func TestOrdinaryShopDualConflictResyncRealMySQL(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_SHOP_MYSQL_CI") != "1" {
		t.Skip("disposable shop MySQL CI not enabled")
	}
	dsn := os.Getenv("JIUYIN_TEST_SHOP_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("missing disposable MySQL DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&name); err != nil || name != "shop_atomic_ci" {
		t.Fatalf("REFUSE non-disposable database %q: %v", name, err)
	}
	for _, statement := range []string{
		`CREATE TABLE roles (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (role_id BIGINT UNSIGNED NOT NULL, seq BIGINT NOT NULL, slot INT NOT NULL, config_id VARCHAR(255) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL, view_id INT NOT NULL, name TEXT NULL, equip_type TEXT NULL, art_pack INT NULL, hardiness INT NULL, max_hardiness INT NULL, PRIMARY KEY(role_id,seq)) ENGINE=InnoDB`,
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY, snapshot JSON NOT NULL) ENGINE=InnoDB`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const roleID uint64 = 991196
	beforeWallet := currencySnapshot{Silver: 100}
	otherWallet := currencySnapshot{Silver: 85}
	failedWallet := currencySnapshot{Silver: 90}
	beforeJSON, _ := json.Marshal(beforeWallet)
	otherJSON, _ := json.Marshal(otherWallet)
	failedJSON, _ := json.Marshal(failedWallet)
	if _, err := db.Exec(`INSERT INTO roles(role_id) VALUES (?)`, roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)`, roleID, beforeJSON); err != nil {
		t.Fatal(err)
	}
	beforeBag := []bagItem{{ConfigID: "original", ItemType: 100, Amount: 2, ViewID: 1, Slot: 4}}
	if _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id) VALUES (?,?,?,?,?,?,?)`, roleID, 0, 4, "original", 100, 2, 1); err != nil {
		t.Fatal(err)
	}
	independent, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer independent.Close()
	otherBag := append(append([]bagItem(nil), beforeBag...), bagItem{ConfigID: "other_session_purchase", ItemType: 100, Amount: 1, ViewID: 1, Slot: 5})
	if err := shopbuyatomic.SaveCheckedBag(independent, roleID, ordinaryShopBagRows(otherBag), ordinaryShopBagRows(beforeBag), beforeJSON, otherJSON); err != nil {
		t.Fatalf("independent purchase: %v", err)
	}
	failedBag := append(append([]bagItem(nil), beforeBag...), bagItem{ConfigID: "failed_purchase_reward", ItemType: 100, Amount: 1, ViewID: 1, Slot: 5})
	if err := shopbuyatomic.SaveCheckedBag(db, roleID, ordinaryShopBagRows(failedBag), ordinaryShopBagRows(beforeBag), beforeJSON, failedJSON); !errors.Is(err, shopbuyatomic.ErrWalletChanged) {
		t.Fatalf("stale wallet must be returned first, got %v", err)
	}
	actor := &playerActor{}
	beforeWallet.toActor(actor)
	actor.restoreBag(beforeBag)
	capture := &ordinaryShopBagConflictCapture{}
	if err := resyncOrdinaryShopPurchaseAfterWalletConflict(capture, actor, &mysqlBagStore{db: db}, &mysqlCurrencyStore{db: db}, role.RoleID(roleID), beforeBag, beforeWallet); err != nil {
		t.Fatalf("dual-conflict resync: %v", err)
	}
	var gotWallet currencySnapshot
	gotWallet.fromActor(actor)
	if gotWallet != otherWallet || !reflect.DeepEqual(actor.bagSnapshot(), otherBag) || !actor.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatalf("stale actor not reconciled: wallet=%+v bag=%+v", gotWallet, actor.bagSnapshot())
	}
	if len(capture.frames) != 4 {
		t.Fatalf("expected currency, remove, and two persisted add frames; got %d", len(capture.frames))
	}
	wantCurrency, err := ordinaryShopCurrencyFrame(otherWallet)
	if err != nil || !bytes.Equal(capture.frames[0], wantCurrency) {
		t.Fatalf("currency frame not latest: %v", err)
	}
	for _, frame := range capture.frames {
		if bytes.Contains(frame, []byte("failed_purchase_reward")) {
			t.Fatal("rejected purchase reward was sent")
		}
	}
	var savedJSON []byte
	if err := independent.QueryRow(`SELECT snapshot FROM role_currency WHERE role_id = ?`, roleID).Scan(&savedJSON); err != nil {
		t.Fatal(err)
	}
	var savedWallet currencySnapshot
	if err := json.Unmarshal(savedJSON, &savedWallet); err != nil || savedWallet != otherWallet {
		t.Fatalf("rejected purchase changed persisted wallet: %+v %v", savedWallet, err)
	}
	persistedBag, ok := (&mysqlBagStore{db: independent}).Load(role.RoleID(roleID))
	if !ok || !reflect.DeepEqual(persistedBag, otherBag) {
		t.Fatalf("rejected purchase changed persisted bag: %+v ok=%t", persistedBag, ok)
	}
	if err := persistDeferredShopCurrency(&mysqlCurrencyStore{db: db}, role.RoleID(roleID), actor); err != nil {
		t.Fatal(err)
	}
	if err := independent.QueryRow(`SELECT snapshot FROM role_currency WHERE role_id = ?`, roleID).Scan(&savedJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(savedJSON, &savedWallet); err != nil || savedWallet != otherWallet {
		t.Fatalf("disconnect overwrote newer wallet: %+v %v", savedWallet, err)
	}
}
