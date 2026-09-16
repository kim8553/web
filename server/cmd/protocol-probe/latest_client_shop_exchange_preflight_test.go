package main

import (
	"github.com/local/9yin-go-server/internal/exchangeplan"
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentShopExchangeReadOnlyPreflightPlansBoundFirstWithoutMutation(t *testing.T) {
	dir := t.TempDir()
	shopPath := filepath.Join(dir, "shop.ini")
	exchangePath := filepath.Join(dir, "exchangeitem.ini")
	if err := os.WriteFile(shopPath, []byte(`[Shop_test]
Type=0
0=result_item,2,3,0,0,1,0,1001,0
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exchangePath, []byte(`[1001]
Item=mat_a,2
Prop=CapitalType1,10
`), 0o600); err != nil {
		t.Fatal(err)
	}
	player := &playerActor{
		silver: 100,
		bagItems: []bagItem{
			{ConfigID: "mat_a", ViewID: 1, Slot: 1, Amount: 1, BindStatus: 0},
			{ConfigID: "mat_a", ViewID: 1, Slot: 2, Amount: 1, BindStatus: 1},
			{ConfigID: "mat_a", ViewID: 1, Slot: 3, Amount: 2, BindStatus: 0},
		},
	}
	itemCatalog := &itemCatalog{byID: map[string]depotItem{
		"result_item": {ConfigID: "result_item", ViewID: 1, MaxAmount: 10},
	}}
	before := player.bagSnapshot()
	got, err := runShopExchangeReadOnlyPreflight(shopPath, exchangePath, nil, player, itemCatalog, shopExchangeBuyRequest{ShopID: "Shop_test", Page: 0, Position: 1, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !got.ConditionSupported || !got.ConditionSatisfied {
		t.Fatalf("empty condition must be satisfied/supported: %+v", got)
	}
	if !got.PropertySupported || !got.PropertySatisfied {
		t.Fatalf("known CapitalType1 cost should be supported/satisfied: %+v", got)
	}
	if !got.MaterialPlan.Satisfied || len(got.MaterialPlan.Results) != 2 {
		t.Fatalf("material plan=%+v", got.MaterialPlan)
	}
	if !got.MaterialPlan.Results[0].MaterialDerivedBound || got.MaterialPlan.Results[1].MaterialDerivedBound {
		t.Fatalf("per-result material bind preview=%+v", got.MaterialPlan.Results)
	}
	if got.OutputAmount != 4 {
		t.Fatalf("output amount=%d want=4", got.OutputAmount)
	}
	if !got.CapacitySupported || !got.CapacitySatisfied {
		t.Fatalf("current output capacity preview=%+v", got.CapacityPlan)
	}
	if got.CapacityPlan.Container != 2 || got.CapacityPlan.Capacity != 132 || got.CapacityPlan.RequiredOutputStacks != 1 {
		t.Fatalf("capacity plan=%+v", got.CapacityPlan)
	}
	after := player.bagSnapshot()
	if len(before) != len(after) {
		t.Fatalf("read-only preflight mutated bag length before=%d after=%d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("read-only preflight mutated bag[%d]: before=%+v after=%+v", i, before[i], after[i])
		}
	}
	if silver, _, _, _ := player.currencySnapshot(); silver != 100 {
		t.Fatalf("read-only preflight mutated silver=%d", silver)
	}
}

func TestCurrentShopExchangeConditionPreviewUsesCurrentAndOrDisplayRule(t *testing.T) {
	catalog := &shopConditionCatalog{Definitions: map[int32]shopConditionDefinition{
		10: {ID: 10, Type: 1, Name: "Sex", Para: [6]string{"=="}, Min: "1", HasMin: true},
		20: {ID: 20, Type: 1, Name: "Sex", Para: [6]string{"=="}, Min: "2", HasMin: true},
	}, Formulas: map[int32]string{}}
	authority := &currentShopConditionAuthority{catalog: catalog}
	player := &playerActor{sex: 1}

	supported, satisfied, _, err := evaluateShopExchangeConditionPreview(shopExchangeDefinition{ConditionType: 0, Condition: "10,20"}, authority, player)
	if err != nil {
		t.Fatal(err)
	}
	if !supported || satisfied {
		t.Fatalf("AND preview supported=%v satisfied=%v, want true/false", supported, satisfied)
	}
	supported, satisfied, _, err = evaluateShopExchangeConditionPreview(shopExchangeDefinition{ConditionType: 1, Condition: "10,20"}, authority, player)
	if err != nil {
		t.Fatal(err)
	}
	if !supported || !satisfied {
		t.Fatalf("OR preview supported=%v satisfied=%v, want true/true", supported, satisfied)
	}
	// Current Lua chooses OR for every non-zero ConditionType, including the
	// authored non-boolean values present in the exact current corpus.
	supported, satisfied, _, err = evaluateShopExchangeConditionPreview(shopExchangeDefinition{ConditionType: 17377, Condition: "10,20"}, authority, player)
	if err != nil {
		t.Fatal(err)
	}
	if !supported || !satisfied {
		t.Fatalf("non-zero OR preview supported=%v satisfied=%v, want true/true", supported, satisfied)
	}
}

func TestCurrentShopExchangePropertyPreviewFailsClosedForUnmappedState(t *testing.T) {
	player := &playerActor{silver: 100}
	supported, satisfied, err := evaluateShopExchangePropertyPreview(shopExchangeDefinition{Prop: "RevengeFriendly,10"}, player, 1)
	if err != nil {
		t.Fatal(err)
	}
	if supported || satisfied {
		t.Fatalf("unmapped property supported=%v satisfied=%v, want false/false", supported, satisfied)
	}
	supported, satisfied, err = evaluateShopExchangePropertyPreview(shopExchangeDefinition{Type: 1, AddValue: "5"}, player, 1)
	if err != nil {
		t.Fatal(err)
	}
	if supported || satisfied {
		t.Fatalf("guild currency supported=%v satisfied=%v, want false/false", supported, satisfied)
	}
}

func TestCurrentShopExchangeRequirementsAcceptExactCurrentTrailingSemicolonOnly(t *testing.T) {
	requirements, err := parseShopExchangeRequirements("Item_xdm_exchange01,440;item_exc_fc_mml,80;")
	if err != nil {
		t.Fatal(err)
	}
	if len(requirements) != 2 || requirements[0].ConfigID != "Item_xdm_exchange01" || requirements[0].Amount != 440 || requirements[1].ConfigID != "item_exc_fc_mml" || requirements[1].Amount != 80 {
		t.Fatalf("requirements=%+v", requirements)
	}
	if _, err := parseShopExchangeRequirements("mat_a,1;;mat_b,1"); err == nil {
		t.Fatal("embedded empty Item entry must remain fail-closed")
	}
}

func TestFirstShopExchangeMaterialBindPreview(t *testing.T) {
	if got := firstShopExchangeMaterialBindPreview(exchangeplan.BatchPlan{}); got != 0 {
		t.Fatalf("unsatisfied/empty preview=%d, want 0", got)
	}
	unbound := exchangeplan.BatchPlan{Satisfied: true, Results: []exchangeplan.ResultPlan{{MaterialDerivedBound: false}}}
	if got := firstShopExchangeMaterialBindPreview(unbound); got != 0 {
		t.Fatalf("unbound preview=%d, want 0", got)
	}
	bound := exchangeplan.BatchPlan{Satisfied: true, Results: []exchangeplan.ResultPlan{{MaterialDerivedBound: true}, {MaterialDerivedBound: false}}}
	if got := firstShopExchangeMaterialBindPreview(bound); got != 1 {
		t.Fatalf("first-result bound preview=%d, want 1", got)
	}
}

func TestCurrentShopExchangeCapacityPreviewFailsClosedWithoutStaticOutput(t *testing.T) {
	player := &playerActor{bagItems: []bagItem{{ConfigID: "mat", ViewID: 1, Slot: 1, Amount: 1, MaxAmount: 1}}}
	snapshot := player.bagSnapshot()
	supported, satisfied, plan, err := evaluateShopExchangeCapacityPreview(&itemCatalog{byID: map[string]depotItem{}}, snapshot, shopCatalogItem{configID: "missing"}, exchangeplan.BatchPlan{Satisfied: true}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if supported || satisfied || plan.Fits {
		t.Fatalf("unknown output static data must fail closed: supported=%t satisfied=%t plan=%+v", supported, satisfied, plan)
	}
}

func TestCurrentShopExchangeCapacityPreviewUsesMaterialFreedSlot(t *testing.T) {
	snapshot := make([]bagItem, 132)
	for i := range snapshot {
		snapshot[i] = bagItem{ConfigID: "filler", ViewID: 1, Slot: int32(i + 1), Amount: 1, MaxAmount: 1}
	}
	snapshot[0].ConfigID = "mat"
	material := exchangeplan.BatchPlan{Satisfied: true, Results: []exchangeplan.ResultPlan{{Deductions: []exchangeplan.Deduction{{StackIndex: 0, ConfigID: "mat", Amount: 1, BindStatus: 0}}}}}
	catalog := &itemCatalog{byID: map[string]depotItem{"result": {ConfigID: "result", ViewID: 1, MaxAmount: 1}}}
	supported, satisfied, plan, err := evaluateShopExchangeCapacityPreview(catalog, snapshot, shopCatalogItem{configID: "result"}, material, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !supported || !satisfied || !plan.Fits || plan.FreedByMaterials != 1 || plan.RequiredOutputStacks != 1 {
		t.Fatalf("capacity preview=%+v supported=%t satisfied=%t", plan, supported, satisfied)
	}
}
