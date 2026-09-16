package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/local/9yin-go-server/internal/exchangeplan"
	"github.com/local/9yin-go-server/internal/role"
)

type shopExchangeCommitStore struct {
	player     *playerActor
	saveErr    error
	saveCalls  int
	saved      []bagItem
	liveAtSave []bagItem
}

func (store *shopExchangeCommitStore) Load(role.RoleID) ([]bagItem, bool) { return nil, false }

func (store *shopExchangeCommitStore) Save(_ role.RoleID, items []bagItem) error {
	store.saveCalls++
	store.saved = append([]bagItem(nil), items...)
	if store.player != nil {
		// Save is invoked while the caller owns player.mu, so do not re-lock here.
		store.liveAtSave = append([]bagItem(nil), store.player.bagItems...)
	}
	return store.saveErr
}

func shopExchangeCommitFixture() (*playerActor, *itemCatalog, []bagItem, exchangeplan.ReplacementPlan) {
	player := newPlayerActor("commit-test", 0)
	original := []bagItem{
		{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1, BindStatus: 1},
		{ConfigID: "keep", ViewID: 1, Slot: 2, Amount: 2, MaxAmount: 10, BindStatus: 0},
	}
	player.bagItems = append([]bagItem(nil), original...)
	catalog := &itemCatalog{byID: map[string]depotItem{
		"result": {ConfigID: "result", ItemType: 7, ViewID: 1, Name: "Result", MaxAmount: 10, LogicPack: 9},
	}}
	plan := exchangeplan.ReplacementPlan{
		Fits: true,
		Deductions: []exchangeplan.StackMutation{{
			Index: 0, Container: 2, Slot: 1, ConfigID: "mat", BindStatus: 1,
			AmountBefore: 1, AmountAfter: 0, Removed: true,
		}},
		Adds: []exchangeplan.AddedStack{{
			Container: 2, Slot: 1, ConfigID: "result", Amount: 1, MaxAmount: 10, BindStatus: 1,
		}},
	}
	return player, catalog, original, plan
}

func TestCommitShopExchangeReplacementPersistenceFirstSavesBeforeLiveSwap(t *testing.T) {
	player, catalog, original, plan := shopExchangeCommitFixture()
	store := &shopExchangeCommitStore{player: player}

	got, err := commitShopExchangeReplacementPersistenceFirst(role.RoleID(7), store, catalog, player, original, plan)
	if err != nil {
		t.Fatal(err)
	}
	if store.saveCalls != 1 {
		t.Fatalf("save calls=%d want=1", store.saveCalls)
	}
	if !reflect.DeepEqual(store.liveAtSave, original) {
		t.Fatalf("live bag changed before persistence: got=%+v want=%+v", store.liveAtSave, original)
	}
	live := player.bagSnapshot()
	if !reflect.DeepEqual(live, got) || !reflect.DeepEqual(store.saved, got) {
		t.Fatalf("commit mismatch: live=%+v saved=%+v got=%+v", live, store.saved, got)
	}
	if len(got) != 2 || got[0].ConfigID != "keep" || got[1].ConfigID != "result" || got[1].BindStatus != 1 {
		t.Fatalf("unexpected committed bag: %+v", got)
	}

	// Prove file-backed stores cannot share the live player's backing array.
	player.mu.Lock()
	player.bagItems[0].Amount = 99
	player.mu.Unlock()
	if store.saved[0].Amount == 99 {
		t.Fatal("persisted slice aliases live player bag")
	}
}

func TestCommitShopExchangeReplacementPersistenceFirstSaveFailureLeavesLiveBagUntouched(t *testing.T) {
	player, catalog, original, plan := shopExchangeCommitFixture()
	store := &shopExchangeCommitStore{player: player, saveErr: errors.New("disk down")}

	if _, err := commitShopExchangeReplacementPersistenceFirst(role.RoleID(7), store, catalog, player, original, plan); err == nil {
		t.Fatal("persistence failure must fail commit")
	}
	if store.saveCalls != 1 {
		t.Fatalf("save calls=%d want=1", store.saveCalls)
	}
	if !reflect.DeepEqual(player.bagSnapshot(), original) {
		t.Fatalf("live bag mutated after persistence failure: %+v", player.bagSnapshot())
	}
}

func TestCommitShopExchangeReplacementPersistenceFirstRejectsStaleSnapshotBeforeSave(t *testing.T) {
	player, catalog, original, plan := shopExchangeCommitFixture()
	store := &shopExchangeCommitStore{player: player}
	expected := append([]bagItem(nil), original...)
	player.bagItems[0].Amount = 2

	if _, err := commitShopExchangeReplacementPersistenceFirst(role.RoleID(7), store, catalog, player, expected, plan); err == nil {
		t.Fatal("stale snapshot must fail commit")
	}
	if store.saveCalls != 0 {
		t.Fatalf("stale snapshot reached persistence: save calls=%d", store.saveCalls)
	}
	if player.bagSnapshot()[0].Amount != 2 {
		t.Fatalf("stale rejection mutated live bag: %+v", player.bagSnapshot())
	}
}
