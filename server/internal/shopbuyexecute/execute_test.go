package shopbuyexecute

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/shopbuygate"
)

type stagedPurchase struct {
	bag      int
	currency int
}

func allowedGate() shopbuygate.Decision {
	return shopbuygate.Evaluate(shopbuygate.Evidence{WireVerified: true, AtomicPersistenceReady: true})
}

func TestPersistFirstExactOrder(t *testing.T) {
	var order []string
	got, err := PersistFirst(
		allowedGate(),
		func() (stagedPurchase, error) {
			order = append(order, "stage")
			return stagedPurchase{bag: 2, currency: 90}, nil
		},
		func(v stagedPurchase) error { order = append(order, "persist"); return nil },
		func(v stagedPurchase) { order = append(order, "apply") },
		func(v stagedPurchase) error { order = append(order, "publish"); return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != (stagedPurchase{bag: 2, currency: 90}) {
		t.Fatalf("staged=%+v", got)
	}
	want := []string{"stage", "persist", "apply", "publish"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order=%v want %v", order, want)
	}
}

func TestBlockedGateRunsNothing(t *testing.T) {
	called := false
	_, err := PersistFirst(
		shopbuygate.Evaluate(shopbuygate.Evidence{}),
		func() (int, error) { called = true; return 1, nil },
		func(int) error { called = true; return nil },
		func(int) { called = true },
		func(int) error { called = true; return nil },
	)
	if err == nil || !strings.Contains(err.Error(), "wire_unverified") {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatal("blocked gate must run no mutation callback")
	}
}

func TestStageFailureRunsNoPersistenceOrMutation(t *testing.T) {
	boom := errors.New("stage failed")
	persisted, applied, published := false, false, false
	_, err := PersistFirst(
		allowedGate(),
		func() (int, error) { return 0, boom },
		func(int) error { persisted = true; return nil },
		func(int) { applied = true },
		func(int) error { published = true; return nil },
	)
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if persisted || applied || published {
		t.Fatalf("callbacks after failed stage: persist=%v apply=%v publish=%v", persisted, applied, published)
	}
}

func TestPersistenceFailureLeavesLiveAndClientUntouched(t *testing.T) {
	boom := errors.New("db failed")
	applied, published := false, false
	_, err := PersistFirst(
		allowedGate(),
		func() (int, error) { return 7, nil },
		func(int) error { return boom },
		func(int) { applied = true },
		func(int) error { published = true; return nil },
	)
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if applied || published {
		t.Fatalf("persistence failure mutated state: apply=%v publish=%v", applied, published)
	}
}

func TestPublishFailureOccursAfterDurableApply(t *testing.T) {
	boom := errors.New("socket failed")
	persisted, applied := false, false
	got, err := PersistFirst(
		allowedGate(),
		func() (int, error) { return 11, nil },
		func(int) error { persisted = true; return nil },
		func(int) {
			if !persisted {
				t.Fatal("apply before persist")
			}
			applied = true
		},
		func(int) error {
			if !persisted || !applied {
				t.Fatal("publish before durable live apply")
			}
			return boom
		},
	)
	if got != 11 || !errors.Is(err, boom) {
		t.Fatalf("got=%d err=%v", got, err)
	}
}

func TestNilCallbacksFailClosed(t *testing.T) {
	gate := allowedGate()
	stage := func() (int, error) { return 1, nil }
	persist := func(int) error { return nil }
	apply := func(int) {}
	publish := func(int) error { return nil }

	tests := []struct {
		name string
		run  func() error
	}{
		{"stage", func() error { _, err := PersistFirst[int](gate, nil, persist, apply, publish); return err }},
		{"persist", func() error { _, err := PersistFirst[int](gate, stage, nil, apply, publish); return err }},
		{"apply", func() error { _, err := PersistFirst[int](gate, stage, persist, nil, publish); return err }},
		{"publish", func() error { _, err := PersistFirst[int](gate, stage, persist, apply, nil); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("expected fail-closed error")
			}
		})
	}
}
