package shopbuyatomic

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// mysqlBagStore.Load converts SQL NULL and empty/zero values into the same
// bagItem fields, and ordinaryShopBagRows uses nullableString/nullableInt32.
// The locked SQL comparison must accept that exact representation difference.
func TestCheckedBagAcceptsLegacyExplicitEmptyMetadata(t *testing.T) {
	for _, mode := range []string{"purchase", "bag_only"} {
		t.Run(mode, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			const id uint64 = 71009
			existing := Row{Slot: 1, ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 3}
			after := []Row{existing, {Slot: 2, ConfigID: "purchased", ItemType: 100, Amount: 1, ViewID: 3}}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(id).
				WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(id))
			if mode == "purchase" {
				mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(id).
					WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":100}`)))
			}
			mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness").WithArgs(id).
				WillReturnRows(sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"}).
					AddRow(1, "existing", 100, 1, 3, "", "", 0, 0, 0))
			mock.ExpectExec("DELETE FROM role_bag_items").WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))
			for seq, item := range after {
				mock.ExpectExec("INSERT INTO role_bag_items").WithArgs(id, seq, item.Slot, item.ConfigID, item.ItemType, item.Amount, item.ViewID, nil, nil, nil, nil, nil).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}
			if mode == "purchase" {
				mock.ExpectExec("DELETE FROM role_currency").WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO role_currency").WithArgs(id, []byte(`{"silver":90}`)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectCommit()
			if mode == "purchase" {
				err = SaveCheckedBag(db, id, after, []Row{existing}, []byte(`{"silver":100}`), []byte(`{"silver":90}`))
			} else {
				err = SaveBagChecked(db, id, after, []Row{existing})
			}
			if err != nil {
				t.Fatalf("unchanged bag represented with explicit SQL empties rejected: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Explicit nonzero values are not a representation change: a concurrent
// metadata edit must still abort before the DELETE/INSERT sequence.
func TestCheckedBagRejectsLegacyNonzeroMetadataChange(t *testing.T) {
	for _, tc := range []struct {
		name          string
		actualName    any
		actualArtPack any
	}{
		{name: "name", actualName: "new name", actualArtPack: 0},
		{name: "art_pack", actualName: "", actualArtPack: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			const id uint64 = 71010
			old := Row{Slot: 1, ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 3}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(id).
				WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(id))
			mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(id).
				WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":100}`)))
			mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness").WithArgs(id).
				WillReturnRows(sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"}).
					AddRow(1, "existing", 100, 1, 3, tc.actualName, "", tc.actualArtPack, 0, 0))
			mock.ExpectRollback()
			err = SaveCheckedBag(db, id, []Row{old, {Slot: 2, ConfigID: "purchased", ItemType: 100, Amount: 1, ViewID: 3}},
				[]Row{old}, []byte(`{"silver":100}`), []byte(`{"silver":90}`))
			if !errors.Is(err, ErrBagChanged) {
				t.Fatalf("changed nonzero metadata must block purchase: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
