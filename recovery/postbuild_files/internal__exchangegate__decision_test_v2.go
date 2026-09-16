package exchangegate

import (
	"reflect"
	"testing"
)

func allReadyEvidence() Evidence {
	return Evidence{
		ConditionSupported: true,
		ConditionSatisfied: true,
		PropertySupported:  true,
		PropertySatisfied:  true,
		MaterialsSatisfied: true,
		CapacitySupported:  true,
		CapacitySatisfied:  true,
		BindingKnown:       true,
		PersistenceReady:   true,
	}
}

func TestDecisionZeroValueFailsClosed(t *testing.T) {
	var d Decision
	if d.Allowed() {
		t.Fatal("zero-value decision must not allow mutation")
	}
	if err := d.Require(); err == nil {
		t.Fatal("zero-value decision must fail Require")
	}
}

func TestEvaluateAllowsOnlyAllReady(t *testing.T) {
	d := Evaluate(allReadyEvidence())
	if !d.Allowed() {
		t.Fatalf("all-ready evidence unexpectedly blocked: %v", d.Blocks())
	}
	if err := d.Require(); err != nil {
		t.Fatal(err)
	}
}

func TestEvaluateCollectsIndependentBlocksWithoutInventingPrecedence(t *testing.T) {
	d := Evaluate(Evidence{})
	want := []Reason{
		ConditionUnsupported,
		PropertyUnsupported,
		MaterialsUnsatisfied,
		CapacityUnsupported,
		BindingUnknown,
		PersistenceUnavailable,
	}
	if d.Allowed() {
		t.Fatal("empty evidence must be blocked")
	}
	if !reflect.DeepEqual(d.Blocks(), want) {
		t.Fatalf("blocks=%v want=%v", d.Blocks(), want)
	}
}

func TestEvaluateDistinguishesUnsupportedFromUnsatisfied(t *testing.T) {
	e := allReadyEvidence()
	e.ConditionSatisfied = false
	e.PropertySatisfied = false
	e.CapacitySatisfied = false
	d := Evaluate(e)
	want := []Reason{ConditionUnsatisfied, PropertyUnsatisfied, CapacityUnsatisfied}
	if !reflect.DeepEqual(d.Blocks(), want) {
		t.Fatalf("blocks=%v want=%v", d.Blocks(), want)
	}
}

func TestEvaluateBlocksUnknownBindingEvenWhenEverythingElseReady(t *testing.T) {
	e := allReadyEvidence()
	e.BindingKnown = false
	d := Evaluate(e)
	if d.Allowed() || !reflect.DeepEqual(d.Blocks(), []Reason{BindingUnknown}) {
		t.Fatalf("decision=%v blocks=%v", d.Allowed(), d.Blocks())
	}
}

func TestEvaluateBlocksUnavailablePersistenceEvenWhenEverythingElseReady(t *testing.T) {
	e := allReadyEvidence()
	e.PersistenceReady = false
	d := Evaluate(e)
	if d.Allowed() || !reflect.DeepEqual(d.Blocks(), []Reason{PersistenceUnavailable}) {
		t.Fatalf("decision=%v blocks=%v", d.Allowed(), d.Blocks())
	}
}

type persistenceTarget struct{}

func TestPersistenceTargetReadyRejectsNilAndTypedNil(t *testing.T) {
	var direct any
	if PersistenceTargetReady(direct) {
		t.Fatal("nil interface must not be ready")
	}
	var typedNil *persistenceTarget
	var wrapped any = typedNil
	if PersistenceTargetReady(wrapped) {
		t.Fatal("typed-nil pointer inside interface must not be ready")
	}
}

func TestPersistenceTargetReadyAcceptsConcreteTarget(t *testing.T) {
	if !PersistenceTargetReady(&persistenceTarget{}) {
		t.Fatal("non-nil pointer target should be ready")
	}
	if !PersistenceTargetReady(persistenceTarget{}) {
		t.Fatal("non-reference concrete target should be ready")
	}
}
