package exchangeplan

import "testing"

func TestPlanConservativeCapacityCountsFreedMaterialSlots(t *testing.T) {
	inventory := []InventorySlot{
		{Index: 10, Container: 2, Slot: 1, Amount: 2},
		{Index: 11, Container: 2, Slot: 2, Amount: 5},
		{Index: 12, Container: 121, Slot: 1, Amount: 1},
	}
	material := BatchPlan{Satisfied: true, Results: []ResultPlan{{Deductions: []Deduction{{StackIndex: 10, Amount: 2}}}}}
	got, err := PlanConservativeCapacity(inventory, material, 2, 3, 11, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.OccupiedBefore != 2 || got.OccupiedAfterRemoval != 1 || got.FreedByMaterials != 1 || got.RequiredOutputStacks != 2 || !got.Fits {
		t.Fatalf("plan=%+v", got)
	}
}

func TestPlanConservativeCapacityFailsWhenWholeBatchCannotFit(t *testing.T) {
	inventory := []InventorySlot{
		{Index: 1, Container: 2, Slot: 1, Amount: 1},
		{Index: 2, Container: 2, Slot: 2, Amount: 1},
		{Index: 3, Container: 2, Slot: 3, Amount: 1},
	}
	got, err := PlanConservativeCapacity(inventory, BatchPlan{Satisfied: true}, 2, 3, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fits || got.RequiredOutputStacks != 2 || got.OccupiedAfterRemoval != 3 {
		t.Fatalf("plan=%+v", got)
	}
}

func TestPlanConservativeCapacityDoesNotAssumeResultMerge(t *testing.T) {
	inventory := []InventorySlot{{Index: 1, Container: 2, Slot: 1, Amount: 5}}
	got, err := PlanConservativeCapacity(inventory, BatchPlan{Satisfied: true}, 2, 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fits {
		t.Fatalf("existing partial stack must not be assumed merge-compatible: %+v", got)
	}
}

func TestPlanConservativeCapacityRejectsDuplicateSlots(t *testing.T) {
	inventory := []InventorySlot{
		{Index: 1, Container: 2, Slot: 1, Amount: 1},
		{Index: 2, Container: 2, Slot: 1, Amount: 1},
	}
	if _, err := PlanConservativeCapacity(inventory, BatchPlan{Satisfied: true}, 2, 10, 1, 1); err == nil {
		t.Fatal("duplicate current-container slots must fail closed")
	}
}

func TestPlanConservativeCapacityRequiresStableDeductionIndex(t *testing.T) {
	inventory := []InventorySlot{{Index: 42, Container: 2, Slot: 1, Amount: 1}}
	material := BatchPlan{Satisfied: true, Results: []ResultPlan{{Deductions: []Deduction{{StackIndex: 7, Amount: 1}}}}}
	if _, err := PlanConservativeCapacity(inventory, material, 2, 10, 1, 1); err == nil {
		t.Fatal("missing stable stack identity must fail closed")
	}
}
