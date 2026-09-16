package main

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

func newShopBuyAtomicMock(t *testing.T) (*mysqlBagStore, *mysqlCurrencyStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &mysqlBagStore{db: db}, &mysqlCurrencyStore{db: db}, mock
}

func TestPersistShopPurchaseMySQLAtomicCommitsBagAndCurrencyTogether(t *testing.T) {
	bagStore, currencyStore, mock := newShopBuyAtomicMock(t)
	roleID := role.RoleID(77)
	items := []bagItem{{
		ConfigID:   "item_test",
		ItemType:   3,
		Amount:     2,
		BindStatus: 1,
		ViewID:     9001,
	}}
	value := currencySnapshot{Silver: 123, Gold: 456, SilverCard: 789, SilverTicket: 10}

	mock.ExpectBegin()
	mock.ExpectExec(shopBuyAtomicDeleteBagSQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(shopBuyAtomicInsertBagSQL).WithArgs(
		roleID,
		0,
		int32(1),
		"item_test",
		int32(3),
		int32(2),
		int32(9001),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		int32(1),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(shopBuyAtomicDeleteCurrencySQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(shopBuyAtomicInsertCurrencySQL).WithArgs(roleID, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := persistShopPurchaseMySQLAtomic(roleID, bagStore, currencyStore, items, value); err != nil {
		t.Fatalf("persistShopPurchaseMySQLAtomic: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPersistShopPurchaseMySQLAtomicRollsBackOnCurrencyInsertFailure(t *testing.T) {
	bagStore, currencyStore, mock := newShopBuyAtomicMock(t)
	roleID := role.RoleID(78)

	mock.ExpectBegin()
	mock.ExpectExec(shopBuyAtomicDeleteBagSQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(shopBuyAtomicDeleteCurrencySQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(shopBuyAtomicInsertCurrencySQL).WithArgs(roleID, sqlmock.AnyArg()).WillReturnError(errors.New("currency insert failed"))
	mock.ExpectRollback()

	if err := persistShopPurchaseMySQLAtomic(roleID, bagStore, currencyStore, nil, currencySnapshot{Silver: 1}); err == nil {
		t.Fatal("expected currency insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPersistShopPurchaseMySQLAtomicRollsBackOnBagInsertFailure(t *testing.T) {
	bagStore, currencyStore, mock := newShopBuyAtomicMock(t)
	roleID := role.RoleID(79)
	items := []bagItem{{ConfigID: "item_fail", ItemType: 4, Amount: 1, ViewID: 9002}}

	mock.ExpectBegin()
	mock.ExpectExec(shopBuyAtomicDeleteBagSQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(shopBuyAtomicInsertBagSQL).WithArgs(
		roleID,
		0,
		int32(1),
		"item_fail",
		int32(4),
		int32(1),
		int32(9002),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		int32(0),
	).WillReturnError(errors.New("bag insert failed"))
	mock.ExpectRollback()

	if err := persistShopPurchaseMySQLAtomic(roleID, bagStore, currencyStore, items, currencySnapshot{}); err == nil {
		t.Fatal("expected bag insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPersistShopPurchaseMySQLAtomicRejectsSplitDB(t *testing.T) {
	db1, mock1, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New db1: %v", err)
	}
	defer db1.Close()
	db2, mock2, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New db2: %v", err)
	}
	defer db2.Close()

	err = persistShopPurchaseMySQLAtomic(role.RoleID(80), &mysqlBagStore{db: db1}, &mysqlCurrencyStore{db: db2}, nil, currencySnapshot{})
	if err == nil {
		t.Fatal("expected split DB rejection")
	}
	if err := mock1.ExpectationsWereMet(); err != nil {
		t.Fatalf("db1 unexpected SQL: %v", err)
	}
	if err := mock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("db2 unexpected SQL: %v", err)
	}
}

func TestPersistShopPurchaseMySQLAtomicRejectsUnavailableInputs(t *testing.T) {
	bagStore, currencyStore, mock := newShopBuyAtomicMock(t)

	cases := []struct {
		name          string
		roleID        role.RoleID
		bagStore      *mysqlBagStore
		currencyStore *mysqlCurrencyStore
	}{
		{name: "zero role", roleID: 0, bagStore: bagStore, currencyStore: currencyStore},
		{name: "nil bag", roleID: 81, bagStore: nil, currencyStore: currencyStore},
		{name: "nil currency", roleID: 81, bagStore: bagStore, currencyStore: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := persistShopPurchaseMySQLAtomic(tc.roleID, tc.bagStore, tc.currencyStore, nil, currencySnapshot{}); err == nil {
				t.Fatal("expected fail-closed input rejection")
			}
		})
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL: %v", err)
	}
}
