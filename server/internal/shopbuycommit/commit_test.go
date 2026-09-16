package shopbuycommit

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/local/9yin-go-server/internal/shopbuygate"
)

func allowedGate() shopbuygate.Decision {
	return shopbuygate.Evaluate(shopbuygate.Evidence{WireVerified: true, AtomicPersistenceReady: true})
}

func assertMutexHeld(t *testing.T, mu *sync.Mutex, where string) {
	t.Helper()
	if mu.TryLock() {
		mu.Unlock()
		t.Fatalf("mutex was not held during %s", where)
	}
}

func assertMutexReleased(t *testing.T, mu *sync.Mutex) {
	t.Helper()
	if !mu.TryLock() {
		t.Fatal("mutex remained locked after commit helper returned")
	}
	mu.Unlock()
}

func TestPersistenceFirstUnderLockOrder(t *testing.T) {
	var mu sync.Mutex
	var events []string
	got, err := PersistenceFirstUnderLock(&mu, allowedGate(),
		func() (int, error) {
			assertMutexHeld(t, &mu, "stage")
			events = append(events, "stage")
			return 7, nil
		},
		func(value int) error {
			assertMutexHeld(t, &mu, "persist")
			if value != 7 {
				t.Fatalf("persist value=%d", value)
			}
			events = append(events, "persist")
			return nil
		},
		func(value int) {
			assertMutexHeld(t, &mu, "apply")
			if value != 7 {
				t.Fatalf("apply value=%d", value)
			}
			events = append(events, "apply")
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 {
		t.Fatalf("got=%d", got)
	}
	if !reflect.DeepEqual(events, []string{"stage", "persist", "apply"}) {
		t.Fatalf("events=%v", events)
	}
	assertMutexReleased(t, &mu)
}

func TestPersistenceFirstUnderLockGateBlocksBeforeCallbacks(t *testing.T) {
	var mu sync.Mutex
	called := false
	gate := shopbuygate.Evaluate(shopbuygate.Evidence{AtomicPersistenceReady: true})
	_, err := PersistenceFirstUnderLock(&mu, gate,
		func() (int, error) { called = true; return 1, nil },
		func(int) error { called = true; return nil },
		func(int) { called = true },
	)
	if err == nil || !strings.Contains(err.Error(), "wire_unverified") {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatal("blocked gate reached callbacks")
	}
	assertMutexReleased(t, &mu)
}

func TestPersistenceFirstUnderLockStageFailureSkipsPersistenceAndApply(t *testing.T) {
	var mu sync.Mutex
	persisted, applied := false, false
	_, err := PersistenceFirstUnderLock(&mu, allowedGate(),
		func() (int, error) { assertMutexHeld(t, &mu, "stage"); return 0, errors.New("stage boom") },
		func(int) error { persisted = true; return nil },
		func(int) { applied = true },
	)
	if err == nil || !strings.Contains(err.Error(), "stage boom") {
		t.Fatalf("err=%v", err)
	}
	if persisted || applied {
		t.Fatalf("persisted=%t applied=%t", persisted, applied)
	}
	assertMutexReleased(t, &mu)
}

func TestPersistenceFirstUnderLockPersistenceFailureSkipsApply(t *testing.T) {
	var mu sync.Mutex
	applied := false
	_, err := PersistenceFirstUnderLock(&mu, allowedGate(),
		func() (int, error) { return 9, nil },
		func(int) error { assertMutexHeld(t, &mu, "persist"); return errors.New("db boom") },
		func(int) { applied = true },
	)
	if err == nil || !strings.Contains(err.Error(), "db boom") {
		t.Fatalf("err=%v", err)
	}
	if applied {
		t.Fatal("persistence failure reached live apply")
	}
	assertMutexReleased(t, &mu)
}

func TestPersistenceFirstUnderLockRejectsMissingStructure(t *testing.T) {
	var mu sync.Mutex
	stage := func() (int, error) { return 1, nil }
	persist := func(int) error { return nil }
	apply := func(int) {}
	cases := []struct {
		name    string
		mu      *sync.Mutex
		stage   func() (int, error)
		persist func(int) error
		apply   func(int)
	}{
		{"nil_mutex", nil, stage, persist, apply},
		{"nil_stage", &mu, nil, persist, apply},
		{"nil_persist", &mu, stage, nil, apply},
		{"nil_apply", &mu, stage, persist, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := PersistenceFirstUnderLock(tc.mu, allowedGate(), tc.stage, tc.persist, tc.apply); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
