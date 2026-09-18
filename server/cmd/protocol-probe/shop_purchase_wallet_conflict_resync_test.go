package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

type shopWalletFrameCapture struct {
	discardMessageConnection
	frames [][]byte
}

func (capture *shopWalletFrameCapture) WriteFrame(frame []byte) error {
	capture.frames = append(capture.frames, append([]byte(nil), frame...))
	return nil
}

func TestOrdinaryShopWalletConflictResyncMatchesExistingCurrencyFrame(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	before := currencySnapshot{Silver: 70, Gold: 8, SilverCard: 4, SilverTicket: 2}
	latest := currencySnapshot{Silver: 55, Gold: 9, SilverCard: 3, SilverTicket: 1}
	player := &playerActor{}
	before.toActor(player)
	player.markOrdinaryShopWalletCommitted(before)
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(role.RoleID(90)).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":55,"gold":9,"silver_card":3,"silver_ticket":1}`)))
	link := &shopWalletFrameCapture{}
	if err := resyncOrdinaryShopWalletAfterConflict(link, player, &mysqlCurrencyStore{db: db}, role.RoleID(90), before); err != nil {
		t.Fatal(err)
	}
	var actual currencySnapshot
	actual.fromActor(player)
	if actual != latest || !player.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatalf("actor/reconciled baseline=%+v expected=%+v", actual, latest)
	}
	wantFrame, err := ordinaryShopCurrencyFrame(latest)
	if err != nil {
		t.Fatal(err)
	}
	if len(link.frames) != 1 || !bytes.Equal(link.frames[0], wantFrame) {
		t.Fatalf("unexpected refresh frame: got=%x want=%x", link.frames, wantFrame)
	}
	// No duplicate logout save after a successful read/reconcile.
	if err := persistDeferredShopCurrency(&mysqlCurrencyStore{db: db}, role.RoleID(90), player); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopWalletConflictResyncRejectsLocalChangeWithoutFrame(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	committed := currencySnapshot{Silver: 70}
	actor := &playerActor{}
	committed.toActor(actor)
	actor.markOrdinaryShopWalletCommitted(committed)
	actor.setSilver(69) // an unsaved local change must not disappear
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(role.RoleID(91)).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":55}`)))
	link := &shopWalletFrameCapture{}
	err = resyncOrdinaryShopWalletAfterConflict(link, actor, &mysqlCurrencyStore{db: db}, role.RoleID(91), currencySnapshot{Silver: 69})
	if err == nil {
		t.Fatal("resync discarded an unpersisted local wallet")
	}
	var got currencySnapshot
	got.fromActor(actor)
	if got.Silver != 69 || len(link.frames) != 0 {
		t.Fatalf("local wallet changed or frame published: %+v frames=%d", got, len(link.frames))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopWalletConflictResyncMissingRowFailsClosed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wallet := currencySnapshot{Silver: 70}
	actor := &playerActor{}
	wallet.toActor(actor)
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(role.RoleID(92)).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}))
	link := &shopWalletFrameCapture{}
	if err := resyncOrdinaryShopWalletAfterConflict(link, actor, &mysqlCurrencyStore{db: db}, role.RoleID(92), wallet); err == nil {
		t.Fatal("missing persisted wallet accepted")
	}
	var got currencySnapshot
	got.fromActor(actor)
	if got != wallet || len(link.frames) != 0 {
		t.Fatalf("missing row changed actor or published frame: %+v frames=%d", got, len(link.frames))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopWalletConflictResyncWriteFailureIsReturned(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wallet := currencySnapshot{Silver: 70}
	actor := &playerActor{}
	wallet.toActor(actor)
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(role.RoleID(93)).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":65}`)))
	// The production connection must close on a failed refresh write, not
	// report a successful purchase or claim client synchronization.
	if err := resyncOrdinaryShopWalletAfterConflict(&shopWalletWriteError{}, actor, &mysqlCurrencyStore{db: db}, role.RoleID(93), wallet); err == nil {
		t.Fatal("failed client update was treated as success")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type shopWalletWriteError struct{ discardMessageConnection }

func (*shopWalletWriteError) WriteFrame([]byte) error { return errors.New("test frame failure") }
