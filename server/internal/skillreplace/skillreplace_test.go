package skillreplace

import "testing"

func TestParsePreservesConditionReplacementAndFlag(t *testing.T) {
	got, ok := Parse("CS_base", "19443", "CS_hidden,1")
	if !ok {
		t.Fatal("rule did not parse")
	}
	if got.BaseID != "CS_base" || got.ConditionID != 19443 || got.ReplacementID != "CS_hidden" || got.Flag != 1 {
		t.Fatalf("parsed rule=%+v", got)
	}
}

func TestIndexBasesIncludesConditionalAndConditionZeroTargets(t *testing.T) {
	indexed := IndexBases([]Rule{
		{BaseID: "CS_base", ConditionID: 19443, ReplacementID: "CS_conditional_hide", Flag: 0},
		{BaseID: "CS_base", ConditionID: 0, ReplacementID: "CS_no_condition_hide", Flag: 1},
	})
	if len(indexed) != 2 {
		t.Fatalf("replacement index=%v", indexed)
	}
	if indexed["cs_conditional_hide"] != "CS_base" || indexed["cs_no_condition_hide"] != "CS_base" {
		t.Fatalf("replacement index=%v", indexed)
	}
}

func TestIndexBasesDropsAmbiguousTargets(t *testing.T) {
	indexed := IndexBases([]Rule{
		{BaseID: "CS_base_a", ConditionID: 19443, ReplacementID: "CS_same_hide"},
		{BaseID: "CS_base_b", ConditionID: 0, ReplacementID: "CS_same_hide"},
	})
	if _, exists := indexed["cs_same_hide"]; exists {
		t.Fatalf("ambiguous target must fail closed: %v", indexed)
	}
}
