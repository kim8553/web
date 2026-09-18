package shopbuyatomic

import (
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveBagCheckedTransaction(t *testing.T) {
	old := Row{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 1, ViewID: 1}
	next := []Row{old, {Slot: 2, ConfigID: "new_item", ItemType: 100, Amount: 1, ViewID: 1}}
	for _, tc := range []struct {
		name string
		stored Row
		bagReadErr error
		insertErr error
		wantErr bool
	}{
		{name: "success", stored: old},
		{name: "stale original amount", stored: Row{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 2, ViewID: 1}, wantErr: true},
		{name: "bag read error", bagReadErr: errors.New("read failed"), wantErr: true},
		{name: "bag insert error rolls back", stored: old, insertErr: errors.New("insert failed"), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(7)))
			bagRead := mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id").WithArgs(uint64(7))
			if tc.bagReadErr != nil {
				bagRead.WillReturnError(tc.bagReadErr)
			} else {
				bagRead.WillReturnRows(sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id"}).AddRow(tc.stored.Slot, tc.stored.ConfigID, tc.stored.ItemType, tc.stored.Amount, tc.stored.ViewID))
			}
			if !tc.wantErr || tc.insertErr != nil {
				mock.ExpectExec("DELETE FROM role_bag_items").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				for seq, item := range next {
					insert := mock.ExpectExec("INSERT INTO role_bag_items").WithArgs(uint64(7), seq, item.Slot, item.ConfigID, item.ItemType, item.Amount, item.ViewID, nil, nil, nil, nil, nil)
					if tc.insertErr != nil {
						insert.WillReturnError(tc.insertErr)
						break
					}
					insert.WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}
			if tc.wantErr {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			err = SaveBagChecked(db, 7, next, []Row{old})
			if (err != nil) != tc.wantErr {
				t.Fatalf("SaveBagChecked err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.name == "stale original amount" && !strings.Contains(err.Error(), "bag changed") {
				t.Fatalf("missing stale bag reason: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSaveBagCheckedInvalidInputs(t *testing.T) {
	if err := SaveBagChecked(nil, 7, nil, nil); err == nil {
		t.Fatal("nil database allowed")
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := SaveBagChecked(db, 0, nil, nil); err == nil {
		t.Fatal("zero role allowed")
	}
	if err := SaveBagChecked(db, 7, []Row{{Slot: 0, ConfigID: "invalid", Amount: 1}}, nil); err == nil {
		t.Fatal("invalid slot allowed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
