package main

import (
    "reflect"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/local/9yin-go-server/internal/role"
)

// A stale database snapshot must be rejected before ARANGEITEM writes any
// VIEW_REMOVE/VIEW_ADD frames. A nil link intentionally fails the test if the
// handler attempts to publish despite the rejected transaction.
func TestShopAdjacentArrangeRejectsStaleBagBeforePublishing(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatal(err) }
    defer db.Close()

    initial := []bagItem{
        {Slot: 1, ConfigID: "item_a", ItemType: 1, Amount: 2, MaxAmount: 10, ViewID: 1},
        {Slot: 2, ConfigID: "item_a", ItemType: 1, Amount: 3, MaxAmount: 10, ViewID: 1},
    }
    player := &playerActor{}
    player.restoreBag(initial)

    mock.ExpectBegin()
    mock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(7)).WillReturnRows(
        sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(7)))
    mock.ExpectQuery("SELECT slot, config_id, item_type, amount, view_id").WithArgs(uint64(7)).WillReturnRows(
        sqlmock.NewRows([]string{"slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"}).
            AddRow(1, "item_a", 1, 4, 1, nil, nil, nil, nil, nil).
            AddRow(2, "item_a", 1, 3, 1, nil, nil, nil, nil, nil))
    mock.ExpectRollback()

    request := clientCustomMessage{Values: []clientCustomValue{
        {Type: 2, Int32: 36}, {Type: 2, Int32: 2},
        {Type: 2, Int32: 1}, {Type: 2, Int32: 2},
    }}
    handled, err := handleArrangeItemCustom(nil, player, &mysqlBagStore{db: db}, role.RoleID(7), request, "arrange-guard-test")
    if err != nil || !handled {
        t.Fatalf("stale arrange handled=%v err=%v", handled, err)
    }
    if got := player.bagSnapshot(); !reflect.DeepEqual(got, initial) {
        t.Fatalf("stale arrange actor bag changed: got=%#v want=%#v", got, initial)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
}
