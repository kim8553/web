package main

import (
	"reflect"
	"testing"
)

func TestCurrentExchangeConditionIsNeedShowRule(t *testing.T) {
	got, err := exchangeConditionReferences("10,20,30", "40,20", "20,40")
	if err != nil {
		t.Fatal(err)
	}
	want := []exchangeConditionReference{
		{ConditionID: 10, IsNeedShow: true, Source: exchangeConditionPrimary},
		{ConditionID: 20, IsNeedShow: false, Source: exchangeConditionPrimary},
		{ConditionID: 30, IsNeedShow: true, Source: exchangeConditionPrimary},
		{ConditionID: 40, IsNeedShow: true, Source: exchangeConditionSecondary},
		{ConditionID: 20, IsNeedShow: true, Source: exchangeConditionSecondary},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("refs=%#v want=%#v", got, want)
	}
}

func TestCurrentExchangeConditionListsPreserveAuthoredOrderAndDuplicates(t *testing.T) {
	got, err := exchangeConditionReferences("3,1,3", "2,2", "2")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]int32, 0, len(got))
	for _, entry := range got {
		ids = append(ids, entry.ConditionID)
	}
	if want := []int32{3, 1, 3, 2, 2}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("ids=%v want=%v", ids, want)
	}
}

func TestCurrentExchangeConditionListRejectsEmptyToken(t *testing.T) {
	if _, err := exchangeConditionReferences("1,,2", "", ""); err == nil {
		t.Fatal("malformed Condition unexpectedly accepted")
	}
}

func TestCurrentExchangeConditionDetailsEvaluateRecursiveAndFailClosed(t *testing.T) {
	formulas := map[int32]string{
		10: "1&2",
		20: "3|4",
	}
	resolve := func(id int32) (string, bool) { value, ok := formulas[id]; return value, ok }
	leaves := func(id int32) (bool, bool) {
		switch id {
		case 1, 2, 3:
			return true, true
		case 4:
			return false, false
		default:
			return false, false
		}
	}
	details, supported, err := evaluateExchangeConditionDetails("10", "20", "20", resolve, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if supported {
		t.Fatal("unsupported recursive leaf must mark ExchangeData fail-closed")
	}
	want := []shopConditionDetail{
		{ConditionID: 10, IsNeedShow: true, Satisfied: true},
		{ConditionID: 20, IsNeedShow: true, Satisfied: false},
	}
	if !reflect.DeepEqual(details, want) {
		t.Fatalf("details=%#v want=%#v", details, want)
	}
}

func TestCurrentExchangeConditionDetailsAllSupported(t *testing.T) {
	resolve := func(int32) (string, bool) { return "", false }
	details, supported, err := evaluateExchangeConditionDetails("1,2", "3", "2,3", resolve, func(id int32) (bool, bool) {
		return id != 2, true
	})
	if err != nil {
		t.Fatal(err)
	}
	if !supported {
		t.Fatal("all authoritative leaves unexpectedly marked unsupported")
	}
	want := []shopConditionDetail{
		{ConditionID: 1, IsNeedShow: true, Satisfied: true},
		{ConditionID: 2, IsNeedShow: false, Satisfied: false},
		{ConditionID: 3, IsNeedShow: true, Satisfied: true},
	}
	if !reflect.DeepEqual(details, want) {
		t.Fatalf("details=%#v want=%#v", details, want)
	}
}
