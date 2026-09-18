package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

type gmGrantSaveCapture struct {
	saves int
	saved []bagItem
	err   error
}

func (*gmGrantSaveCapture) Load(role.RoleID) ([]bagItem, bool) { return nil, false }
func (s *gmGrantSaveCapture) Save(_ role.RoleID, items []bagItem) error {
	s.saves++
	s.saved = append([]bagItem(nil), items...)
	return s.err
}

func TestGMGrantSaveFailureDoesNotLeaveFalseSuccessInBag(t *testing.T) {
	for _, kind := range []string{"tool", "equipment"} {
		t.Run(kind, func(t *testing.T) {
			player := newPlayerActor("gm-grant", 0)
			player.addBagItem(bagItem{Slot: 1, ConfigID: "existing", ItemType: 50, Amount: 3, ViewID: 1})
			before := player.bagSnapshot()
			item := bagItem{ConfigID: "grant", ItemType: 50, Amount: 2, ViewID: 1}
			if kind == "equipment" {
				item = bagItem{ConfigID: "grant_equipment", ItemType: 100, Amount: 1, ViewID: 3, EquipType: "weapon", Hardiness: 80, MaxHardiness: 100}
			}
			saveFailure := errors.New("injected bag write failure")
			store := &gmGrantSaveCapture{err: saveFailure}
			err := grantBagItemDurably(player, store, role.RoleID(9), item)
			if !errors.Is(err, saveFailure) {
				t.Fatalf("save failure lost: %v", err)
			}
			if store.saves != 1 || len(store.saved) != 2 {
				t.Fatalf("expected one attempted save with prospective reward, saves=%d rows=%+v", store.saves, store.saved)
			}
			if got := player.bagSnapshot(); !reflect.DeepEqual(got, before) {
				t.Fatalf("failed GM grant changed actor bag: got=%+v before=%+v", got, before)
			}
			store.err = nil
			if err := grantBagItemDurably(player, store, role.RoleID(9), item); err != nil {
				t.Fatalf("retry after recovered persistence: %v", err)
			}
			if store.saves != 2 || !reflect.DeepEqual(player.bagSnapshot(), store.saved) {
				t.Fatalf("successful grant not synchronized with saved rows: actor=%+v saved=%+v", player.bagSnapshot(), store.saved)
			}
		})
	}
}

func TestGMGrantMissingStorageOrRoleDoesNotMutateBag(t *testing.T) {
	player := newPlayerActor("gm-grant", 0)
	item := bagItem{ConfigID: "grant", Amount: 1, ViewID: 1}
	if err := grantBagItemDurably(player, nil, 1, item); err == nil || len(player.bagSnapshot()) != 0 {
		t.Fatalf("nil store should reject without mutation: %v", err)
	}
	store := &gmGrantSaveCapture{}
	if err := grantBagItemDurably(player, store, 0, item); err == nil || store.saves != 0 || len(player.bagSnapshot()) != 0 {
		t.Fatalf("zero role should reject without mutation: %v", err)
	}
}
