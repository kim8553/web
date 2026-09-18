package shopbuyatomic

import (
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveCheckedBagIdentity(t *testing.T) {
	base := Row{Slot: 1, ConfigID: "original", ItemType: 100, Amount: 3, ViewID: 1}
	next := []Row{base, {Slot: 2, ConfigID: "reward", ItemType: 100, Amount: 1, ViewID: 1}}
	for _, tc := range []struct {
		name     string
		stored   []Row
		expected []Row
		queryErr error
		wantErr  bool
	}{
		{name: "matching existing bag", stored: []Row{base}, expected: []Row{base}},
		{name: "changed amount", stored: []Row{{Slot: 1, ConfigID: "original", ItemType: 100, Amount: 2, ViewID: 1}}, expected: []Row{base}, wantErr: true},
		{name: "moved slot", stored: []Row{{Slot: 3, ConfigID: "original", ItemType: 100, Amount: 3, ViewID: 1}}, expected: []Row{base}, wantErr: true},
		{name: "extra bag row", stored: []Row{base, {Slot: 3, ConfigID: "extra", Amount: 1}}, expected: []Row{base}, wantErr: true},
		{name: "deleted bag row", expected: []Row{base}, wantErr: true},
		{name: "bag lock read fails", expected: []Row{base}, queryErr: errors.New("bag read failed"), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(7)))
			mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(uint64(7)).WillReturnRows(
				sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":100}`)))
			bagRead := mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id").WithArgs(uint64(7))
			if tc.queryErr != nil {
				bagRead.WillReturnError(tc.queryErr)
			} else {
				rows := sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"})
				for _, item := range tc.stored {
					rows.AddRow(item.Slot, item.ConfigID, item.ItemType, item.Amount, item.ViewID, item.Name, item.EquipType, item.ArtPack, item.Hardiness, item.MaxHardiness)
				}
				bagRead.WillReturnRows(rows)
			}
			if tc.wantErr {
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("DELETE FROM role_bag_items").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				for seq, item := range next {
					mock.ExpectExec("INSERT INTO role_bag_items").WithArgs(uint64(7), seq, item.Slot, item.ConfigID, item.ItemType, item.Amount, item.ViewID, nil, nil, nil, nil, nil).WillReturnResult(sqlmock.NewResult(0, 1))
				}
				mock.ExpectExec("DELETE FROM role_currency").WithArgs(uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO role_currency").WithArgs(uint64(7), []byte(`{"silver":90}`)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			err = SaveCheckedBag(db, 7, next, tc.expected, []byte(`{"silver":100}`), []byte(`{"silver":90}`))
			if (err != nil) != tc.wantErr {
				t.Fatalf("SaveCheckedBag err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr && tc.queryErr == nil && !strings.Contains(err.Error(), "bag changed") {
				t.Fatalf("missing bag conflict explanation: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
