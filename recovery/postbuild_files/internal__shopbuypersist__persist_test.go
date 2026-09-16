package shopbuypersist

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/local/9yin-go-server/internal/role"
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func sampleRow() BagRow {
	return BagRow{
		Seq:        0,
		Slot:       1,
		ConfigID:   "item_test",
		ItemType:   3,
		Amount:     2,
		ViewID:     9001,
		BindStatus: 1,
	}
}

func expectSuccessfulBagInsert(mock sqlmock.Sqlmock, roleID role.RoleID, row BagRow) {
	mock.ExpectExec(insertBagSQL).WithArgs(
		roleID,
		row.Seq,
		row.Slot,
		row.ConfigID,
		row.ItemType,
		row.Amount,
		row.ViewID,
		row.Name,
		row.EquipType,
		row.ArtPack,
		row.Hardiness,
		row.MaxHardiness,
		row.BindStatus,
	).WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestPersistCommitsBagAndCurrencyTogether(t *testing.T) {
	db, mock := newMock(t)
	roleID := role.RoleID(77)
	row := sampleRow()
	currency := []byte(`{"silver":123,"gold":456,"silver_card":789,"silver_ticket":10}`)

	mock.ExpectBegin()
	mock.ExpectExec(deleteBagSQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	expectSuccessfulBagInsert(mock, roleID, row)
	mock.ExpectExec(deleteCurrencySQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(insertCurrencySQL).WithArgs(roleID, currency).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := Persist(context.Background(), db, roleID, []BagRow{row}, currency); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPersistRollsBackOnCurrencyInsertFailure(t *testing.T) {
	db, mock := newMock(t)
	roleID := role.RoleID(78)
	currency := []byte(`{"silver":1}`)

	mock.ExpectBegin()
	mock.ExpectExec(deleteBagSQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(deleteCurrencySQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(insertCurrencySQL).WithArgs(roleID, currency).WillReturnError(errors.New("currency insert failed"))
	mock.ExpectRollback()

	if err := Persist(context.Background(), db, roleID, nil, currency); err == nil {
		t.Fatal("expected currency insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPersistRollsBackOnBagInsertFailure(t *testing.T) {
	db, mock := newMock(t)
	roleID := role.RoleID(79)
	row := sampleRow()

	mock.ExpectBegin()
	mock.ExpectExec(deleteBagSQL).WithArgs(roleID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(insertBagSQL).WithArgs(
		roleID,
		row.Seq,
		row.Slot,
		row.ConfigID,
		row.ItemType,
		row.Amount,
		row.ViewID,
		row.Name,
		row.EquipType,
		row.ArtPack,
		row.Hardiness,
		row.MaxHardiness,
		row.BindStatus,
	).WillReturnError(errors.New("bag insert failed"))
	mock.ExpectRollback()

	if err := Persist(context.Background(), db, roleID, []BagRow{row}, []byte(`{}`)); err == nil {
		t.Fatal("expected bag insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPersistRejectsUnavailableInputsWithoutSQL(t *testing.T) {
	if err := Persist(context.Background(), nil, role.RoleID(1), nil, nil); err == nil {
		t.Fatal("expected nil DB rejection")
	}

	db, mock := newMock(t)
	if err := Persist(context.Background(), db, 0, nil, nil); err == nil {
		t.Fatal("expected zero role rejection")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL: %v", err)
	}
}
