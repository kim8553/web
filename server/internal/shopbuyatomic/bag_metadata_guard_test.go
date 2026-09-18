package shopbuyatomic

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// Only the five identity fields used to be checked, even though checked saves
// delete and recreate the entire row including these persisted properties.
func TestSameBagIdentityProtectsPersistedMetadata(t *testing.T) {
	base := Row{Slot: 7, ConfigID: "equip_test", ItemType: 100, Amount: 1, ViewID: 1,
		Name: "original", EquipType: "weapon", ArtPack: int32(3), Hardiness: int32(80), MaxHardiness: int32(100)}
	for _, tc := range []struct {
		name string
		edit func(*Row)
	}{
		{"name", func(r *Row) { r.Name = "new-name" }},
		{"equip_type", func(r *Row) { r.EquipType = "armor" }},
		{"art_pack", func(r *Row) { r.ArtPack = int32(4) }},
		{"hardiness", func(r *Row) { r.Hardiness = int32(79) }},
		{"max_hardiness", func(r *Row) { r.MaxHardiness = int32(110) }},
		{"null_to_value", func(r *Row) { r.Name = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := base
			tc.edit(&changed)
			if sameBagIdentity(base, changed) {
				t.Fatal("changed persisted item metadata was treated as an unchanged bag; purchase could overwrite it")
			}
		})
	}
	if !sameBagIdentity(base, base) {
		t.Fatal("identical persisted item must be accepted")
	}
}

// The SQL lock must include the actual persisted metadata, not just the pure
// Row comparator. A stale purchase must abort before any DELETE/INSERT.
func TestCheckedShopRejectsChangedPersistedMetadataBeforeWrite(t *testing.T) {
	for _, mode := range []string{"purchase", "bag_move"} {
		t.Run(mode, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			const id uint64 = 71003
			start := Row{Slot: 7, ConfigID: "equip_test", ItemType: 100, Amount: 1, ViewID: 1,
				Name: "original", EquipType: "weapon", ArtPack: int32(3), Hardiness: int32(80), MaxHardiness: int32(100)}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(id))
			if mode == "purchase" {
				mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(id).WillReturnRows(
					sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":100}`)))
			}
			mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness").WithArgs(id).WillReturnRows(
				sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"}).
					AddRow(7, "equip_test", 100, 1, 1, "original", "weapon", 3, 79, 100))
			mock.ExpectRollback()
			next := []Row{start, {Slot: 8, ConfigID: "new_item", ItemType: 100, Amount: 1, ViewID: 1}}
			if mode == "purchase" {
				err = SaveCheckedBag(db, id, next, []Row{start}, []byte(`{"silver":100}`), []byte(`{"silver":90}`))
			} else {
				err = SaveBagChecked(db, id, next, []Row{start})
			}
			if !errors.Is(err, ErrBagChanged) {
				t.Fatalf("stale hardiness should abort before write: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
