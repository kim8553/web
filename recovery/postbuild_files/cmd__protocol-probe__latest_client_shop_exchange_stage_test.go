package main

import (
	"reflect"
	"testing"

	"github.com/local/9yin-go-server/internal/exchangeplan"
)

func TestStageShopExchangeAtomicReplacementIsReadOnlyAndUsesFreedSlot(t *testing.T) {
	snapshot := []bagItem{
		{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1, BindStatus: 1},
		{ConfigID: "keep", ViewID: 1, Slot: 2, Amount: 1, MaxAmount: 1, BindStatus: 0},
	}
	before := append([]bagItem(nil), snapshot...)
	material := exchangeplan.BatchPlan{Satisfied: true, Results: []exchangeplan.ResultPlan{{Deductions: []exchangeplan.Deduction{{StackIndex: 0, ConfigID: "mat", Amount: 1, BindStatus: 1}}}}}
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ViewID: 1, MaxAmount: 1}}}
	got, err := stageShopExchangeAtomicReplacement(catalog, snapshot, shopCatalogItem{configID: "result"}, material, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Fits || len(got.Deductions) != 1 || !got.Deductions[0].Removed || len(got.Adds) != 1 {
		t.Fatalf("stage=%+v", got)
	}
	if got.Adds[0].Slot != 1 || got.Adds[0].BindStatus != 1 {
		t.Fatalf("result stack=%+v", got.Adds[0])
	}
	if !reflect.DeepEqual(snapshot, before) {
		t.Fatalf("staging mutated source snapshot: before=%+v after=%+v", before, snapshot)
	}
}

func TestStageShopExchangeAtomicReplacementRequiresExplicitValidResultBind(t *testing.T) {
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ViewID: 1, MaxAmount: 1}}}
	if _, err := stageShopExchangeAtomicReplacement(catalog, nil, shopCatalogItem{configID: "result"}, exchangeplan.BatchPlan{Satisfied: true}, 1, 2); err == nil {
		t.Fatal("unproven/invalid result binding must not be accepted")
	}
}

func TestStageShopExchangeAtomicReplacementRejectsStaleMaterialIdentity(t *testing.T) {
	snapshot := []bagItem{{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1, BindStatus: 1}}
	material := exchangeplan.BatchPlan{Satisfied: true, Results: []exchangeplan.ResultPlan{{Deductions: []exchangeplan.Deduction{{StackIndex: 0, ConfigID: "mat", Amount: 1, BindStatus: 0}}}}}
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ViewID: 1, MaxAmount: 1}}}
	if _, err := stageShopExchangeAtomicReplacement(catalog, snapshot, shopCatalogItem{configID: "result"}, material, 1, 0); err == nil {
		t.Fatal("stale material binding identity must fail")
	}
}
