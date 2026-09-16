package shopbuygate

import (
	"strings"
	"testing"
)

func TestZeroDecisionFailsClosed(t *testing.T) {
	var d Decision
	if d.Allowed() {
		t.Fatal("zero decision must not allow mutation")
	}
	if err := d.Require(); err == nil || !strings.Contains(err.Error(), "not evaluated") {
		t.Fatalf("Require()=%v, want not-evaluated error", err)
	}
}

func TestEvaluateRequiresWireAndAtomicPersistence(t *testing.T) {
	tests := []struct {
		name   string
		e      Evidence
		allow  bool
		blocks []Reason
	}{
		{name: "none", e: Evidence{}, blocks: []Reason{WireUnverified, AtomicPersistenceUnavailable}},
		{name: "wire only", e: Evidence{WireVerified: true}, blocks: []Reason{AtomicPersistenceUnavailable}},
		{name: "persistence only", e: Evidence{AtomicPersistenceReady: true}, blocks: []Reason{WireUnverified}},
		{name: "both", e: Evidence{WireVerified: true, AtomicPersistenceReady: true}, allow: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Evaluate(tt.e)
			if d.Allowed() != tt.allow {
				t.Fatalf("Allowed()=%v want %v blocks=%v", d.Allowed(), tt.allow, d.Blocks())
			}
			got := d.Blocks()
			if len(got) != len(tt.blocks) {
				t.Fatalf("blocks=%v want %v", got, tt.blocks)
			}
			for i := range got {
				if got[i] != tt.blocks[i] {
					t.Fatalf("blocks=%v want %v", got, tt.blocks)
				}
			}
			if tt.allow {
				if err := d.Require(); err != nil {
					t.Fatalf("Require()=%v", err)
				}
			} else if err := d.Require(); err == nil {
				t.Fatal("blocked decision Require() unexpectedly succeeded")
			}
		})
	}
}
