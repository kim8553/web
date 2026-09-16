package main

import (
	"reflect"
	"testing"
)

func TestCurrentConditionFormulaRightDescendingShape(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"1&2|3", "&(1,|(2,3))"},
		{"1|2&3", "|(1,&(2,3))"},
		{"1&2&3", "&(1,&(2,3))"},
		{"(1&2)|3", "|(wrap(&(1,2)),3)"},
		{"1&(2|3)", "&(1,wrap(|(2,3)))"},
	}
	for _, tt := range tests {
		node, err := parseConditionFormula(tt.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.raw, err)
		}
		if got := conditionFormulaTestShape(node); got != tt.want {
			t.Fatalf("parse %q shape=%s want=%s", tt.raw, got, tt.want)
		}
	}
}

func TestCurrentConditionFormulaEvaluationRequiresEveryLeafSupported(t *testing.T) {
	node, err := parseConditionFormula("1|2")
	if err != nil {
		t.Fatal(err)
	}
	calls := make([]int32, 0, 2)
	satisfied, supported, err := evaluateConditionFormula(node, func(id int32) (bool, bool) {
		calls = append(calls, id)
		if id == 1 {
			return true, true
		}
		return false, false
	})
	if err != nil {
		t.Fatal(err)
	}
	if satisfied || supported {
		t.Fatalf("unsupported RHS must fail closed, got satisfied=%t supported=%t", satisfied, supported)
	}
	if !reflect.DeepEqual(calls, []int32{1, 2}) {
		t.Fatalf("all leaves must be audited, calls=%v", calls)
	}
}

func TestCurrentConditionFormulaEvaluationAndLeaves(t *testing.T) {
	node, err := parseConditionFormula("10&(20|30)")
	if err != nil {
		t.Fatal(err)
	}
	gotLeaves, err := conditionFormulaLeafIDs(node)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int32{10, 20, 30}; !reflect.DeepEqual(gotLeaves, want) {
		t.Fatalf("leaf ids=%v want=%v", gotLeaves, want)
	}
	satisfied, supported, err := evaluateConditionFormula(node, func(id int32) (bool, bool) {
		switch id {
		case 10, 30:
			return true, true
		case 20:
			return false, true
		default:
			return false, false
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !supported || !satisfied {
		t.Fatalf("got satisfied=%t supported=%t want true,true", satisfied, supported)
	}
}

func TestCurrentConditionFormulaRejectsMalformedInput(t *testing.T) {
	for _, raw := range []string{"", "1&", "&1", "1||2", "(1|2", "1|2)", "x"} {
		if _, err := parseConditionFormula(raw); err == nil {
			t.Fatalf("parse %q unexpectedly succeeded", raw)
		}
	}
}

func conditionFormulaTestShape(node *conditionFormulaNode) string {
	if node == nil {
		return "nil"
	}
	switch node.Op {
	case conditionFormulaLeafOrWrapper:
		if node.Child != nil {
			return "wrap(" + conditionFormulaTestShape(node.Child) + ")"
		}
		return int32Text(node.ConditionID)
	case conditionFormulaAnd:
		return "&(" + conditionFormulaTestShape(node.Left) + "," + conditionFormulaTestShape(node.Right) + ")"
	case conditionFormulaOr:
		return "|(" + conditionFormulaTestShape(node.Left) + "," + conditionFormulaTestShape(node.Right) + ")"
	default:
		return "?"
	}
}

func int32Text(value int32) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	var n int64 = int64(value)
	if negative {
		n = -n
	}
	var buf [12]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func TestCurrentConditionFormulaRecursiveExpansion(t *testing.T) {
	formulas := map[int32]string{
		100: "1|200",
		200: "2&3",
	}
	resolve := func(id int32) (string, bool) {
		value, ok := formulas[id]
		return value, ok
	}
	leaves, err := terminalConditionLeaves(100, resolve)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int32{1, 2, 3}; !reflect.DeepEqual(leaves, want) {
		t.Fatalf("terminal leaves=%v want=%v", leaves, want)
	}
	value, supported, err := evaluateConditionRoot(100, resolve, func(id int32) (bool, bool) {
		switch id {
		case 1:
			return false, true
		case 2, 3:
			return true, true
		default:
			return false, false
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !supported || !value {
		t.Fatalf("recursive formula got value=%t supported=%t want true,true", value, supported)
	}
}

func TestCurrentConditionFormulaCycleFailsClosedWithError(t *testing.T) {
	formulas := map[int32]string{100: "200", 200: "100"}
	resolve := func(id int32) (string, bool) { value, ok := formulas[id]; return value, ok }
	if _, err := terminalConditionLeaves(100, resolve); err == nil {
		t.Fatal("cycle unexpectedly accepted by leaf expansion")
	}
	if _, _, err := evaluateConditionRoot(100, resolve, func(int32) (bool, bool) { return true, true }); err == nil {
		t.Fatal("cycle unexpectedly accepted by evaluator")
	}
}
