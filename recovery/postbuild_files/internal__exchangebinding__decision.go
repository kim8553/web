package exchangebinding

import "fmt"

// Decision is deliberately zero-value-unknown. Callers cannot accidentally
// treat the Go zero value as an authoritative unbound result.
type Decision struct {
	known      bool
	bindStatus int32
}

func Bound() Decision   { return Decision{known: true, bindStatus: 1} }
func Unbound() Decision { return Decision{known: true, bindStatus: 0} }

// Require returns a runtime item BindStatus only for an explicit authoritative
// decision. Unknown decisions fail closed.
func (d Decision) Require() (int32, error) {
	if !d.known {
		return 0, fmt.Errorf("exchangebinding: final BindStatus unresolved")
	}
	return d.bindStatus, nil
}

func (d Decision) Known() bool { return d.known }
