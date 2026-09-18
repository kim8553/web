package main

import (
	"path/filepath"
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
	// Zero sqlmock expectations: a completed purchase is already on MySQL.
	// A stale logout must not DELETE/INSERT a newer wallet from another login.
	if err := persistDeferredShopCurrency(store, role.RoleID(9001), player); err != nil {
		t.Fatalf("already committed wallet: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("duplicate MySQL currency write: %v", err)
	}

	// This guard must not silently lose subsequent local currency changes.
	player.setSilver(69)
	if player.ordinaryShopWalletUnchangedSinceCommit() {
		t.Fatal("locally changed wallet incorrectly treated as committed")
	}
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM role_currency").WithArgs(role.RoleID(9001)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO role_currency").WithArgs(role.RoleID(9001), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := persistDeferredShopCurrency(store, role.RoleID(9001), player); err != nil {
		t.Fatalf("changed wallet persistence: %v", err)
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
