package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

func TestOrdinaryShopDualConflictResyncRefreshesBothWithoutReward(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	const roleID role.RoleID = 94
	beforeWallet := currencySnapshot{Silver: 100}
	latestWallet := currencySnapshot{Silver: 82}
	beforeBag := []bagItem{{ConfigID: "existing", ItemType: 100, Amount: 2, ViewID: 1, Slot: 4}}
	latestBag := []bagItem{beforeBag[0], {ConfigID: "other_session_purchase", ItemType: 100, Amount: 1, ViewID: 1, Slot: 5}}
	actor := &playerActor{}
	beforeWallet.toActor(actor)
	actor.markOrdinaryShopWalletCommitted(beforeWallet)
	actor.restoreBag(beforeBag)
	encoded, err := json.Marshal(latestWallet)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(roleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow(encoded))
	mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(roleID).
		WillReturnRows(sqlmock.NewRows([]string{"config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness", "slot"}).
			AddRow("existing", 100, 2, 1, nil, nil, nil, nil, nil, 4).
			AddRow("other_session_purchase", 100, 1, 1, nil, nil, nil, nil, nil, 5))
	capture := &ordinaryShopBagConflictCapture{}
	if err := resyncOrdinaryShopPurchaseAfterWalletConflict(capture, actor, &mysqlBagStore{db: db}, &mysqlCurrencyStore{db: db}, roleID, beforeBag, beforeWallet); err != nil {
		t.Fatal(err)
	}
	var gotWallet currencySnapshot
	gotWallet.fromActor(actor)
	if gotWallet != latestWallet || !actor.ordinaryShopWalletUnchangedSinceCommit() || !reflect.DeepEqual(actor.bagSnapshot(), latestBag) {
		t.Fatalf("reconciled actor wallet=%+v bag=%+v", gotWallet, actor.bagSnapshot())
	}
	if len(capture.frames) != 4 {
		t.Fatalf("got %d frames; need currency + old removal + two persisted items", len(capture.frames))
	}
	wantCurrency, err := ordinaryShopCurrencyFrame(latestWallet)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(capture.frames[0], wantCurrency) || !bytes.Equal(capture.frames[1], serverViewRemove(2, 4)) {
		t.Fatalf("unexpected wallet/remove frames: %x / %x", capture.frames[0], capture.frames[1])
	}
	for i, item := range latestBag {
		view := bagViewForViewID(item.ViewID)
		want, err := latestClientCurrentBagFrames(view, uint16(item.Slot), item.ViewID, bagItemProps(view, item))
		if err != nil || len(want) != 1 || !bytes.Equal(capture.frames[i+2], want[0]) {
			t.Fatalf("persisted bag frame %d mismatch: %v", i, err)
		}
		if bytes.Contains(capture.frames[i+2], []byte("failed_purchase_reward")) {
			t.Fatal("rejected reward appeared in refresh")
		}
	}
	if err := persistDeferredShopCurrency(&mysqlCurrencyStore{db: db}, roleID, actor); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopDualConflictResyncWalletReadFailureDoesNotTouchBag(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wallet := currencySnapshot{Silver: 100}
	bag := []bagItem{{ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 1, Slot: 1}}
	actor := &playerActor{}
	wallet.toActor(actor)
	actor.restoreBag(bag)
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(role.RoleID(95)).WillReturnError(sql.ErrConnDone)
	capture := &ordinaryShopBagConflictCapture{}
	err = resyncOrdinaryShopPurchaseAfterWalletConflict(capture, actor, &mysqlBagStore{db: db}, &mysqlCurrencyStore{db: db}, role.RoleID(95), bag, wallet)
	if err == nil || !strings.Contains(err.Error(), "refresh wallet") || len(capture.frames) != 0 || !reflect.DeepEqual(actor.bagSnapshot(), bag) {
		t.Fatalf("read failure lost bag or sent frames: err=%v bag=%+v frames=%d", err, actor.bagSnapshot(), len(capture.frames))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopDualConflictResyncBagReadFailureIsFatal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wallet := currencySnapshot{Silver: 100}
	bag := []bagItem{{ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 1, Slot: 1}}
	actor := &playerActor{}
	wallet.toActor(actor)
	actor.restoreBag(bag)
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(role.RoleID(96)).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":90}`)))
	mock.ExpectQuery("SELECT config_id, item_type, amount, view_id").WithArgs(role.RoleID(96)).WillReturnError(errors.New("bag DB unavailable"))
	capture := &ordinaryShopBagConflictCapture{}
	err = resyncOrdinaryShopPurchaseAfterWalletConflict(capture, actor, &mysqlBagStore{db: db}, &mysqlCurrencyStore{db: db}, role.RoleID(96), bag, wallet)
	if err == nil || !strings.Contains(err.Error(), "refresh bag") || len(capture.frames) != 1 || !reflect.DeepEqual(actor.bagSnapshot(), bag) {
		t.Fatalf("bag read failure should be fatal without bag replay: err=%v bag=%+v frames=%d", err, actor.bagSnapshot(), len(capture.frames))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
