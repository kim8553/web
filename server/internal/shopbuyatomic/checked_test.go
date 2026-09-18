package shopbuyatomic

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveCheckedWalletLock(t *testing.T) {
	for _, tc := range []struct {
		name string
		stored string
		queryErr error
		fail string
	}{
		{name: "matching wallet reordered JSON", stored: `{"gold":0,"silver":100}`},
		{name: "first purchase without wallet row", fail: "missing"},
		{name: "stale wallet rejected", stored: `{"silver":90,"gold":0}`, fail: "stale"},
		{name: "malformed stored wallet rejected", stored: `{"silver":"broken"}`, fail: "decode"},
		{name: "locked read fails", queryErr: errors.New("read failed"), fail: "query"},
		{name: "wallet insert fails rolls bag back", stored: `{"gold":0,"silver":100}`, fail: "insert"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil { t.Fatal(err) }
			defer db.Close()
			mock.ExpectBegin()
			query := mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(uint64(7))
			if tc.queryErr != nil {
				query.WillReturnError(tc.queryErr)
			} else if tc.fail == "missing" {
				query.WillReturnRows(sqlmock.NewRows([]string{"snapshot"}))
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(tc.stored)))
			}
			if tc.fail == "stale" || tc.fail == "decode" || tc.fail == "query" {
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("DELETE FROM role_bag_items").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO role_bag_items").WithArgs(uint64(7), 0, int32(1), "item_test", int32(100), int32(2), int32(1), nil, nil, nil, nil, nil).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM role_currency").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				insert := mock.ExpectExec("INSERT INTO role_currency").WithArgs(uint64(7), []byte(`{"silver":98,"gold":0}`))
				if tc.fail == "insert" {
					insert.WillReturnError(errors.New("insert failed"))
					mock.ExpectRollback()
				} else {
					insert.WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
				}
			}
			err = SaveChecked(db, 7, []Row{{Slot:1, ConfigID:"item_test", ItemType:100, Amount:2, ViewID:1}}, []byte(`{"silver":100,"gold":0}`), []byte(`{"silver":98,"gold":0}`))
			if (err != nil) != (tc.fail == "stale" || tc.fail == "decode" || tc.fail == "query" || tc.fail == "insert") {
				t.Fatalf("SaveChecked error=%v fail=%s", err, tc.fail)
			}
			if tc.fail == "stale" && (err == nil || !strings.Contains(err.Error(), "wallet changed")) {
				t.Fatalf("stale wallet error=%v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
		})
	}
}

func TestSaveCheckedBadInputsDoNotBegin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	rows := []Row{{Slot:1,ConfigID:"item_test", Amount:1}}
	for _, tc := range []struct{ role uint64; before, after []byte; rows []Row }{
		{7, nil, []byte(`{"silver":99}`), rows},
		{7, []byte(`{"silver":100}`), []byte(`{"silver":"invalid"}`), rows},
		{7, []byte(`{"silver":100}`), []byte(`{"silver":99}`), []Row{{Slot:0,ConfigID:"item_test", Amount:1}}},
		{0, []byte(`{"silver":100}`), []byte(`{"silver":99}`), rows},
	} {
		if err := SaveChecked(db, tc.role, tc.rows, tc.before, tc.after); err == nil { t.Fatalf("accepted invalid input %+v", tc) }
	}
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
	_ = sql.ErrNoRows
}
