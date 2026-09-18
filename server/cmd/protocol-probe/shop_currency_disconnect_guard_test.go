package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

func TestOrdinaryShopCommittedWalletSkipsDeferredMySQLOverwrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := &mysqlCurrencyStore{db: db}
	player := &playerActor{}
	wallet := currencySnapshot{Silver: 70, Gold: 11, SilverCard: 5, SilverTicket: 3}
	wallet.toActor(player)
	if player.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatal("uncommitted actor wallet must not bypass persistence")
	}
	player.markOrdinaryShopWalletCommitted(wallet)
	if !player.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatal("committed ordinary purchase wallet was not recognized")
	}
	// Zero SQL expectations: purchase already committed exactly this wallet.
	if err := persistDeferredShopCurrency(store, role.RoleID(9001), player); err != nil {
		t.Fatalf("already committed wallet: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("duplicate MySQL currency write: %v", err)
	}

	// A local change must compare against the LAST COMMITTED PURCHASE wallet
	// before modifying the database; unlike the old unguarded DELETE/INSERT.
	player.setSilver(69)
	if player.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatal("locally changed wallet incorrectly treated as committed")
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(9001)).WillReturnRows(
		sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(9001)))
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(uint64(9001)).WillReturnRows(
		sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":70,"gold":11,"silver_card":5,"silver_ticket":3}`)))
	mock.ExpectExec("UPDATE role_currency SET snapshot").WithArgs(sqlmock.AnyArg(), uint64(9001)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := persistDeferredShopCurrency(store, role.RoleID(9001), player); err != nil {
		t.Fatalf("changed wallet persistence: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopChangedWalletRejectsStaleLogout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	player := &playerActor{}
	wallet := currencySnapshot{Silver: 70, Gold: 11, SilverCard: 5, SilverTicket: 3}
	wallet.toActor(player)
	player.markOrdinaryShopWalletCommitted(wallet)
	player.setSilver(69)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(9003)).WillReturnRows(
		sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(9003)))
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(uint64(9003)).WillReturnRows(
		sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":65,"gold":11,"silver_card":5,"silver_ticket":3}`)))
	mock.ExpectRollback()
	err = persistDeferredShopCurrency(&mysqlCurrencyStore{db: db}, role.RoleID(9003), player)
	if err == nil || !strings.Contains(err.Error(), "wallet changed") {
		t.Fatalf("expected stale wallet refusal; got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOrdinaryShopDisconnectGuardDoesNotChangeJSONPersistence(t *testing.T) {
	store := &currencyStore{path: filepath.Join(t.TempDir(), "currency.json"), roles: make(map[string]currencySnapshot)}
	player := &playerActor{}
	wallet := currencySnapshot{Silver: 44, Gold: 22}
	wallet.toActor(player)
	player.markOrdinaryShopWalletCommitted(wallet)
	if err := persistDeferredShopCurrency(store, role.RoleID(9002), player); err != nil {
		t.Fatalf("JSON currency save: %v", err)
	}
	got, ok := store.Load(role.RoleID(9002))
	if !ok || got != wallet {
		t.Fatalf("JSON currency persistence changed got=%+v found=%t want=%+v", got, ok, wallet)
	}
}
