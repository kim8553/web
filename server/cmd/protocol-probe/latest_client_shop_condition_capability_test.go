package main

import (
	"reflect"
	"testing"
)

func TestCurrentExchangeConditionCapabilityAuditSafeOnlyWhenEveryLeafExact(t *testing.T) {
	definitions := map[int32]exchangeConditionSpec{
		10: {Condition: "100", Condition2: "3"},
		20: {Condition: "200"},
		30: {},
	}
	formulas := map[int32]string{
		100: "1&2",
		200: "2|4",
	}
	resolve := func(id int32) (string, bool) { value, ok := formulas[id]; return value, ok }
	classify := func(id int32) conditionCapabilityClass {
		switch id {
		case 1, 2, 3:
			return conditionCapabilityExactEvaluator
		case 4:
			return conditionCapabilityAuthoritativeStateMissing
		default:
			return conditionCapabilitySemanticsUnresolved
		}
	}
	audit, err := auditShopExchangeConditionCapabilities(definitions, resolve, classify)
	if err != nil {
		t.Fatal(err)
	}
	if audit.ExchangeDataTotal != 3 || audit.SafeExchangeData != 2 || audit.FailClosed != 1 {
		t.Fatalf("totals=%d safe=%d failClosed=%d", audit.ExchangeDataTotal, audit.SafeExchangeData, audit.FailClosed)
	}
	if want := []int32{1, 2, 3, 4}; !reflect.DeepEqual(audit.UniqueLeaves, want) {
		t.Fatalf("unique leaves=%v want=%v", audit.UniqueLeaves, want)
	}
	if !audit.ByExchangeData[10].Safe || audit.ByExchangeData[20].Safe || !audit.ByExchangeData[30].Safe {
		t.Fatalf("unexpected safety map: %#v", audit.ByExchangeData)
	}
	if audit.ClassCounts[conditionCapabilityExactEvaluator] != 3 || audit.ClassCounts[conditionCapabilityAuthoritativeStateMissing] != 1 {
		t.Fatalf("class counts=%v", audit.ClassCounts)
	}
}

func TestCurrentExchangeConditionCapabilityAuditDeduplicatesLeafPerExchangeData(t *testing.T) {
	definitions := map[int32]exchangeConditionSpec{
		7: {Condition: "100,1", Condition2: "100"},
	}
	resolve := func(id int32) (string, bool) {
		if id == 100 {
			return "1|1", true
		}
		return "", false
	}
	audit, err := auditShopExchangeConditionCapabilities(definitions, resolve, func(int32) conditionCapabilityClass {
		return conditionCapabilityExactEvaluator
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := audit.ByExchangeData[7].Leaves, []int32{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("leaves=%v want=%v", got, want)
	}
}

func TestCurrentExchangeConditionCapabilityAuditRejectsFormulaCycle(t *testing.T) {
	definitions := map[int32]exchangeConditionSpec{1: {Condition: "100"}}
	resolve := func(id int32) (string, bool) {
		if id == 100 {
			return "200", true
		}
		if id == 200 {
			return "100", true
		}
		return "", false
	}
	if _, err := auditShopExchangeConditionCapabilities(definitions, resolve, func(int32) conditionCapabilityClass {
		return conditionCapabilityExactEvaluator
	}); err == nil {
		t.Fatal("formula cycle unexpectedly accepted")
	}
}
