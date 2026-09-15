package exchangeplan

import "testing"

func TestPlanAtomicReplacementStagesFullRemoveAndAdd(t *testing.T) {
	inventory := []InventoryState{
		{Index: 3, Container: 2, Slot: 1, ConfigID: "mat", Amount: 2, MaxAmount: 10, BindStatus: 1},
		{Index: 4, Container: 2, Slot: 2, ConfigID: "keep", Amount: 1, MaxAmount: 1, BindStatus: 0},
	}
	material := BatchPlan{Satisfied: true, Results: []ResultPlan{{Deductions: []Deduction{{StackIndex: 3, ConfigID: "mat", Amount: 2, BindStatus: 1}}}}}
	got, err := PlanAtomicReplacement(inventory, material, OutputSpec{ConfigID: "result", Container: 2, Amount: 11, MaxAmount: 10, BindStatus: 1}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Fits || len(got.Deductions) != 1 || !got.Deductions[0].Removed {
		t.Fatalf("deductions=%+v fits=%t", got.Deductions, got.Fits)
	}
	if len(got.Adds) != 2 || got.Adds[0].Slot != 1 || got.Adds[0].Amount != 10 || got.Adds[1].Slot != 3 || got.Adds[1].Amount != 1 {
		t.Fatalf("adds=%+v", got.Adds)
	}
	if got.Adds[0].BindStatus != 1 || got.Adds[1].BindStatus != 1 {
		t.Fatalf("result binding not preserved: %+v", got.Adds)
	}
}

func TestPlanAtomicReplacementDoesNotReturnCommittablePrefixWhenFullResultCannotFit(t *testing.T) {
	inventory := []InventoryState{
		{Index: 0, Container: 2, Slot: 1, ConfigID: "mat", Amount: 2, BindStatus: 0},
		{Index: 1, Container: 2, Slot: 2, ConfigID: "keep", Amount: 1, BindStatus: 0},
	}
	material := BatchPlan{Satisfied: true, Results: []ResultPlan{{Deductions: []Deduction{{StackIndex: 0, ConfigID: "mat", Amount: 1, BindStatus: 0}}}}}
	got, err := PlanAtomicReplacement(inventory, material, OutputSpec{ConfigID: "result", Container: 2, Amount: 2, MaxAmount: 1, BindStatus: 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fits || len(got.Deductions) != 0 || len(got.Adds) != 0 {
		t.Fatalf("failed replacement exposed committable prefix: %+v", got)
	}
}

func TestPlanAtomicReplacementRejectsStaleDeductionIdentity(t *testing.T) {
	inventory := []InventoryState{{Index: 5, Container: 2, Slot: 1, ConfigID: "mat", Amount: 1, BindStatus: 1}}
	material := BatchPlan{Satisfied: true, Results: []ResultPlan{{Deductions: []Deduction{{StackIndex: 5, ConfigID: "mat", Amount: 1, BindStatus: 0}}}}}
	if _, err := PlanAtomicReplacement(inventory, material, OutputSpec{ConfigID: "result", Container: 2, Amount: 1, MaxAmount: 1, BindStatus: 0}, 2); err == nil {
		t.Fatal("stale BindStatus identity must fail")
	}
}

func TestPlanAtomicReplacementLeavesBindingDecisionExternal(t *testing.T) {
	inventory := []InventoryState{{Index: 1, Container: 2, Slot: 1, ConfigID: "mat", Amount: 1, BindStatus: 1}}
	material := BatchPlan{Satisfied: true, Results: []ResultPlan{{Deductions: []Deduction{{StackIndex: 1, ConfigID: "mat", Amount: 1, BindStatus: 1}}}}}
	for _, bind := range []int32{0, 1} {
		got, err := PlanAtomicReplacement(inventory, material, OutputSpec{ConfigID: "result", Container: 2, Amount: 1, MaxAmount: 1, BindStatus: bind}, 2)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Fits || len(got.Adds) != 1 || got.Adds[0].BindStatus != bind {
			t.Fatalf("bind=%d plan=%+v", bind, got)
		}
	}
}

func TestPlanAtomicReplacementRejectsInvalidOutputBindStatus(t *testing.T) {
	got, err := PlanAtomicReplacement(nil, BatchPlan{Satisfied: true}, OutputSpec{ConfigID: "result", Container: 2, Amount: 1, MaxAmount: 1, BindStatus: 2}, 2)
	if err == nil || got.Fits {
		t.Fatalf("invalid output bind status must fail: plan=%+v err=%v", got, err)
	}
}
