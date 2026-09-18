package shopbuyatomic

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSavePurchaseCommitAndRollback(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail string
	}{
		{name: "success"},
		{name: "bag insert fails", fail: "bag"},
		{name: "wallet insert fails", fail: "wallet"},
		{name: "commit fails", fail: "commit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM role_bag_items").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
			bagInsert := mock.ExpectExec("INSERT INTO role_bag_items").WithArgs(uint64(7), 0, int32(1), "item_test", int32(100), int32(2), int32(1), nil, nil, nil, nil, nil)
			if tc.fail == "bag" {
				bagInsert.WillReturnError(errors.New("bag failure"))
				mock.ExpectRollback()
			} else {
				bagInsert.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM role_currency").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				walletInsert := mock.ExpectExec("INSERT INTO role_currency").WithArgs(uint64(7), []byte(`{"silver":98}`))
				if tc.fail == "wallet" {
					walletInsert.WillReturnError(errors.New("wallet failure"))
					mock.ExpectRollback()
				} else {
					walletInsert.WillReturnResult(sqlmock.NewResult(0, 1))
					if tc.fail == "commit" {
						mock.ExpectCommit().WillReturnError(errors.New("commit failure"))
					} else {
						mock.ExpectCommit()
					}
				}
			}
			err = Save(db, 7, []Row{{Slot: 1, ConfigID: "item_test", ItemType: 100, Amount: 2, ViewID: 1}}, []byte(`{"silver":98}`))
			if (err != nil) != (tc.fail != "") {
				t.Fatalf("Save error=%v, fail=%q", err, tc.fail)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSavePurchaseInvalidInputDoesNotBegin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, tc := range []struct {
		role uint64
		rows []Row
		wallet []byte
	}{
		{role: 0, wallet: []byte("{}")},
		{role: 7, wallet: nil},
		{role: 7, wallet: []byte("{}"), rows: []Row{{Slot: 0, ConfigID: "item_test", Amount: 1}}},
		{role: 7, wallet: []byte("{}"), rows: []Row{{Slot: 1, ConfigID: "", Amount: 1}}},
		{role: 7, wallet: []byte("{}"), rows: []Row{{Slot: 1, ConfigID: "item_test", Amount: 0}}},
	} {
		if err := Save(db, tc.role, tc.rows, tc.wallet); err == nil {
			t.Fatalf("accepted invalid purchase input: %+v", tc)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
