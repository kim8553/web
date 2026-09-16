package exchangeexecute

import (
	"errors"
	"reflect"
	"testing"

	"github.com/local/9yin-go-server/internal/exchangebinding"
	"github.com/local/9yin-go-server/internal/exchangegate"
)

func readyGate(bindingKnown bool) exchangegate.Decision {
	return exchangegate.Evaluate(exchangegate.Evidence{
		ConditionSupported: true,
		ConditionSatisfied: true,
		PropertySupported:  true,
		PropertySatisfied:  true,
		MaterialsSatisfied: true,
		CapacitySupported:  true,
		CapacitySatisfied:  true,
		BindingKnown:       bindingKnown,
		PersistenceReady:   true,
	})
}

func TestPersistFirstExactOrderAndBoundValue(t *testing.T) {
	var order []string
	got, err := PersistFirst(
		readyGate(true),
		exchangebinding.Bound(),
		func(bindStatus int32) (string, error) {
			order = append(order, "stage")
			if bindStatus != 1 {
				t.Fatalf("bindStatus=%d", bindStatus)
			}
			return "staged", nil
		},
		func(staged string) error {
			order = append(order, "persist")
			if staged != "staged" {
				t.Fatalf("persist staged=%q", staged)
			}
			return nil
		},
		func(staged string) {
			order = append(order, "apply")
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != "staged" || !reflect.DeepEqual(order, []string{"stage", "persist", "apply"}) {
		t.Fatalf("got=%q order=%v", got, order)
	}
}

func TestPersistFirstBlockedGateCallsNothing(t *testing.T) {
	called := false
	_, err := PersistFirst(
		readyGate(false),
		exchangebinding.Bound(),
		func(int32) (int, error) { called = true; return 1, nil },
		func(int) error { called = true; return nil },
		func(int) { called = true },
	)
	if err == nil {
		t.Fatal("blocked gate must fail")
	}
	if called {
		t.Fatal("blocked gate must not call stage/persist/apply")
	}
}

func TestPersistFirstUnknownBindingCallsNothingEvenIfGateClaimsKnown(t *testing.T) {
	called := false
	var unknown exchangebinding.Decision
	_, err := PersistFirst(
		readyGate(true),
		unknown,
		func(int32) (int, error) { called = true; return 1, nil },
		func(int) error { called = true; return nil },
		func(int) { called = true },
	)
	if err == nil {
		t.Fatal("unknown binding must fail")
	}
	if called {
		t.Fatal("unknown binding must not call stage/persist/apply")
	}
}

func TestPersistFirstStageFailureStopsBeforePersistence(t *testing.T) {
	persisted := false
	applied := false
	_, err := PersistFirst(
		readyGate(true),
		exchangebinding.Unbound(),
		func(int32) (int, error) { return 0, errors.New("stage failed") },
		func(int) error { persisted = true; return nil },
		func(int) { applied = true },
	)
	if err == nil || persisted || applied {
		t.Fatalf("err=%v persisted=%t applied=%t", err, persisted, applied)
	}
}

func TestPersistFirstPersistenceFailureNeverAppliesLiveState(t *testing.T) {
	applied := false
	_, err := PersistFirst(
		readyGate(true),
		exchangebinding.Unbound(),
		func(int32) (int, error) { return 7, nil },
		func(int) error { return errors.New("save failed") },
		func(int) { applied = true },
	)
	if err == nil {
		t.Fatal("persistence failure must fail")
	}
	if applied {
		t.Fatal("live apply must not run after persistence failure")
	}
}

func TestPersistFirstRejectsMissingCallbacksBeforeExecution(t *testing.T) {
	if _, err := PersistFirst[int](readyGate(true), exchangebinding.Bound(), nil, func(int) error { return nil }, func(int) {}); err == nil {
		t.Fatal("nil stage must fail")
	}
	if _, err := PersistFirst[int](readyGate(true), exchangebinding.Bound(), func(int32) (int, error) { return 1, nil }, nil, func(int) {}); err == nil {
		t.Fatal("nil persist must fail")
	}
	if _, err := PersistFirst[int](readyGate(true), exchangebinding.Bound(), func(int32) (int, error) { return 1, nil }, func(int) error { return nil }, nil); err == nil {
		t.Fatal("nil apply must fail")
	}
}
