package exchangebinding

import "testing"

func TestDecisionZeroValueFailsClosed(t *testing.T) {
	var decision Decision
	if decision.Known() {
		t.Fatal("zero-value decision must be unknown")
	}
	if _, err := decision.Require(); err == nil {
		t.Fatal("unknown decision must fail closed")
	}
}

func TestDecisionExplicitBoundAndUnbound(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    Decision
		want int32
	}{
		{name: "unbound", d: Unbound(), want: 0},
		{name: "bound", d: Bound(), want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.d.Known() {
				t.Fatal("explicit decision must be known")
			}
			got, err := tc.d.Require()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("status=%d want=%d", got, tc.want)
			}
		})
	}
}

func TestFromMaterialDerivedBoundTrueIsAuthoritativeBound(t *testing.T) {
	decision := FromMaterialDerivedBound(true)
	if !decision.Known() {
		t.Fatal("bound material consumption must produce a known bound decision")
	}
	got, err := decision.Require()
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("status=%d want=1", got)
	}
}

func TestFromMaterialDerivedBoundFalseStaysUnknown(t *testing.T) {
	decision := FromMaterialDerivedBound(false)
	if decision.Known() {
		t.Fatal("all-unbound consumed materials do not prove final unbound status")
	}
	if _, err := decision.Require(); err == nil {
		t.Fatal("unresolved authored/base binding precedence must fail closed")
	}
}
