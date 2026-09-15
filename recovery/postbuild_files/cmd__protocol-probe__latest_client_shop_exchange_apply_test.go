package main

import (
	"reflect"
	"testing"

	"github.com/local/9yin-go-server/internal/exchangeplan"
)

func TestApplyShopExchangeReplacementToSnapshotIsPureAndMaterializesOutput(t *testing.T) {
	snapshot := []bagItem{
		{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1, BindStatus: 1},
		{ConfigID: "keep", ViewID: 1, Slot: 2, Amount: 2, MaxAmount: 10, BindStatus: 0},
	}
	before := append([]bagItem(nil), snapshot...)
	plan := exchangeplan.ReplacementPlan{
		Fits:       true,
		Deductions: []exchangeplan.StackMutation{{Index: 0, Container: 2, Slot: 1, ConfigID: "mat", BindStatus: 1, AmountBefore: 1, AmountAfter: 0, Removed: true}},
		Adds:       []exchangeplan.AddedStack{{Container: 2, Slot: 1, ConfigID: "result", Amount: 1, MaxAmount: 10, BindStatus: 1}},
	}
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ItemType: 7, ViewID: 1, Name: "Result", MaxAmount: 10, LogicPack: 9}}}
	got, err := applyShopExchangeReplacementToSnapshot(catalog, snapshot, plan)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot, before) {
		t.Fatalf("input mutated: before=%+v after=%+v", before, snapshot)
	}
	if len(got) != 2 || got[0].ConfigID != "keep" || got[1].ConfigID != "result" {
		t.Fatalf("result=%+v", got)
	}
	if got[1].Slot != 1 || got[1].Amount != 1 || got[1].BindStatus != 1 || got[1].ItemType != 7 || got[1].LogicPack != 9 {
		t.Fatalf("output=%+v", got[1])
	}
}

func TestApplyShopExchangeReplacementToSnapshotRejectsStaleAmount(t *testing.T) {
	snapshot := []bagItem{{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 2, MaxAmount: 10, BindStatus: 1}}
	plan := exchangeplan.ReplacementPlan{Fits: true, Deductions: []exchangeplan.StackMutation{{Index: 0, Container: 2, Slot: 1, ConfigID: "mat", BindStatus: 1, AmountBefore: 1, AmountAfter: 0, Removed: true}}}
	if _, err := applyShopExchangeReplacementToSnapshot(&itemCatalog{byID: map[string]depotItem{}}, snapshot, plan); err == nil {
		t.Fatal("stale amount must fail")
	}
}

func TestApplyShopExchangeReplacementToSnapshotRejectsStaleBinding(t *testing.T) {
	snapshot := []bagItem{{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1, BindStatus: 0}}
	plan := exchangeplan.ReplacementPlan{Fits: true, Deductions: []exchangeplan.StackMutation{{Index: 0, Container: 2, Slot: 1, ConfigID: "mat", BindStatus: 1, AmountBefore: 1, AmountAfter: 0, Removed: true}}}
	if _, err := applyShopExchangeReplacementToSnapshot(&itemCatalog{byID: map[string]depotItem{}}, snapshot, plan); err == nil {
		t.Fatal("stale binding must fail")
	}
}

func TestApplyShopExchangeReplacementToSnapshotRejectsOutputSlotCollision(t *testing.T) {
	snapshot := []bagItem{{ConfigID: "keep", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1, BindStatus: 0}}
	plan := exchangeplan.ReplacementPlan{Fits: true, Adds: []exchangeplan.AddedStack{{Container: 2, Slot: 1, ConfigID: "result", Amount: 1, MaxAmount: 1, BindStatus: 0}}}
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ViewID: 1, MaxAmount: 1}}}
	if _, err := applyShopExchangeReplacementToSnapshot(catalog, snapshot, plan); err == nil {
		t.Fatal("occupied output slot must fail")
	}
}

func TestApplyShopExchangeReplacementToSnapshotRejectsChangedOutputMaxAmount(t *testing.T) {
	plan := exchangeplan.ReplacementPlan{Fits: true, Adds: []exchangeplan.AddedStack{{Container: 2, Slot: 1, ConfigID: "result", Amount: 1, MaxAmount: 10, BindStatus: 0}}}
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ViewID: 1, MaxAmount: 20}}}
	if _, err := applyShopExchangeReplacementToSnapshot(catalog, nil, plan); err == nil {
		t.Fatal("changed output MaxAmount must fail stale-catalog guard")
	}
}
