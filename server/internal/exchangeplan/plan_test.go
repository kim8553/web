package exchangeplan

import (
	"reflect"
	"testing"
)

func TestPlanBatchConsumesBoundBeforeUnboundAndTracksPerResult(t *testing.T) {
	stacks := []Stack{
		{Index: 10, ConfigID: "mat_a", Amount: 1, BindStatus: 0},
		{Index: 11, ConfigID: "mat_a", Amount: 1, BindStatus: 1},
		{Index: 12, ConfigID: "mat_a", Amount: 2, BindStatus: 0},
	}
	plan, err := PlanBatch(stacks, []Requirement{{ConfigID: "mat_a", Amount: 2}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Satisfied || len(plan.Results) != 2 {
		t.Fatalf("plan=%+v", plan)
	}
	if !plan.Results[0].MaterialDerivedBound {
		t.Fatal("first result must be bound because the bound stack is consumed first")
	}
	if plan.Results[1].MaterialDerivedBound {
		t.Fatal("second result must be unbound after the bound stack was exhausted")
	}
	if got := plan.Remaining[0].Amount + plan.Remaining[1].Amount + plan.Remaining[2].Amount; got != 0 {
		t.Fatalf("remaining total=%d, want 0", got)
	}
}

func TestPlanBatchInsufficientIsAtomic(t *testing.T) {
	stacks := []Stack{{Index: 1, ConfigID: "mat", Amount: 3, BindStatus: 1}}
	plan, err := PlanBatch(stacks, []Requirement{{ConfigID: "mat", Amount: 2}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Satisfied || len(plan.Results) != 0 {
		t.Fatalf("insufficient plan must have no committable prefix: %+v", plan)
	}
	if !reflect.DeepEqual(plan.Remaining, stacks) {
		t.Fatalf("remaining=%+v, want original=%+v", plan.Remaining, stacks)
	}
}

func TestPlanBatchAggregatesDuplicateRequirements(t *testing.T) {
	plan, err := PlanBatch(
		[]Stack{{Index: 1, ConfigID: "mat", Amount: 3, BindStatus: 0}},
		[]Requirement{{ConfigID: "mat", Amount: 1}, {ConfigID: "mat", Amount: 2}},
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Satisfied || len(plan.Results) != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	var consumed int32
	for _, deduction := range plan.Results[0].Deductions {
		consumed += deduction.Amount
	}
	if consumed != 3 {
		t.Fatalf("consumed=%d, want 3", consumed)
	}
}

func TestPlanBatchConfigIDIsCaseSensitive(t *testing.T) {
	plan, err := PlanBatch(
		[]Stack{{Index: 1, ConfigID: "Mat_A", Amount: 1, BindStatus: 0}},
		[]Requirement{{ConfigID: "mat_a", Amount: 1}},
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Satisfied {
		t.Fatal("ConfigID matching must remain case-sensitive")
	}
}

func TestPlanBatchRejectsInvalidRuntimeBindStatus(t *testing.T) {
	if _, err := PlanBatch([]Stack{{Index: 1, ConfigID: "mat", Amount: 1, BindStatus: 2}}, []Requirement{{ConfigID: "mat", Amount: 1}}, 1); err == nil {
		t.Fatal("invalid runtime BindStatus must fail closed")
	}
}

func TestPlanBatchPreservesStableStackIndexAfterFiltering(t *testing.T) {
	plan, err := PlanBatch(
		[]Stack{{Index: 7, ConfigID: "other", Amount: 1, BindStatus: 0}, {Index: 42, ConfigID: "mat", Amount: 1, BindStatus: 1}},
		[]Requirement{{ConfigID: "mat", Amount: 1}},
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Satisfied || len(plan.Results) != 1 || len(plan.Results[0].Deductions) != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	if got := plan.Results[0].Deductions[0].StackIndex; got != 42 {
		t.Fatalf("deduction stable stack index=%d want=42", got)
	}
	if plan.Remaining[1].Amount != 0 || plan.Remaining[0].Amount != 1 {
		t.Fatalf("remaining=%+v", plan.Remaining)
	}
}

func TestPlanBatchRejectsDuplicateStableStackIndex(t *testing.T) {
	_, err := PlanBatch(
		[]Stack{{Index: 5, ConfigID: "a", Amount: 1, BindStatus: 0}, {Index: 5, ConfigID: "b", Amount: 1, BindStatus: 0}},
		[]Requirement{{ConfigID: "a", Amount: 1}},
		1,
	)
	if err == nil {
		t.Fatal("duplicate stable stack index must fail closed")
	}
}
