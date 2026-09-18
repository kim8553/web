package shopbuyatomic

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// The production currencySnapshot JSON always includes all four fields, but
// mysqlCurrencyStore.Load accepts older JSON with zero-valued fields omitted.
// A shop purchase must compare the values actually loaded into the actor,
// without treating an omitted zero as a concurrent currency modification.
func TestSaveCheckedBagAcceptsOmittedZeroWalletFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	const id uint64 = 7
	before := []byte(`{"silver":100,"gold":0,"silver_card":0,"silver_ticket":0}`)
	after := []byte(`{"silver":90,"gold":0,"silver_card":0,"silver_ticket":0}`)
	reward := Row{Slot: 1, ConfigID: "purchased_item", ItemType: 100, Amount: 1, ViewID: 1}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(id))
	mock.ExpectQuery("SELECT snapshot FROM role_currency").WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`{"silver":100}`)))
	mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id").WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"}))
	mock.ExpectExec("DELETE FROM role_bag_items").WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO role_bag_items").WithArgs(id, 0, reward.Slot, reward.ConfigID,
		reward.ItemType, reward.Amount, reward.ViewID, nil, nil, nil, nil, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM role_currency").WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO role_currency").WithArgs(id, after).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := SaveCheckedBag(db, id, []Row{reward}, nil, before, after); err != nil {
		t.Fatalf("purchase with equivalent legacy wallet was rejected: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSameWalletValuesPreservesConflictsAndUnknownKeys(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stored   map[string]int64
		expected map[string]int64
		want     bool
	}{
		{"omitted zero", map[string]int64{"silver": 100}, map[string]int64{"silver": 100, "gold": 0}, true},
		{"missing nonzero", map[string]int64{"silver": 100}, map[string]int64{"silver": 100, "gold": 1}, false},
		{"changed silver", map[string]int64{"silver": 99}, map[string]int64{"silver": 100}, false},
		{"unexpected saved field", map[string]int64{"silver": 100, "legacy_currency": 0}, map[string]int64{"silver": 100}, false},
		{"identical", map[string]int64{"silver": 100, "gold": 0}, map[string]int64{"silver": 100, "gold": 0}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameWalletValues(tc.stored, tc.expected); got != tc.want {
				t.Fatalf("sameWalletValues(%v, %v) = %t; want %t", tc.stored, tc.expected, got, tc.want)
			}
		})
	}
}
