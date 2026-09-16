package shopbuygate

import (
	"fmt"
	"strings"
)

// Reason identifies an independent fail-closed prerequisite for ordinary NPC
// shop purchase mutation. It does not assign or infer a client message ID.
type Reason string

const (
	WireUnverified                Reason = "wire_unverified"
	AtomicPersistenceUnavailable Reason = "atomic_persistence_unavailable"
)

// Evidence contains only prerequisites that must already be proven by a caller.
// The zero value blocks mutation.
type Evidence struct {
	WireVerified           bool
	AtomicPersistenceReady bool
}

// Decision is deliberately zero-value blocked. Evaluate must be called before
// Allowed can ever return true.
type Decision struct {
	evaluated bool
	blocks    []Reason
}

func Evaluate(e Evidence) Decision {
	blocks := make([]Reason, 0, 2)
	if !e.WireVerified {
		blocks = append(blocks, WireUnverified)
	}
	if !e.AtomicPersistenceReady {
		blocks = append(blocks, AtomicPersistenceUnavailable)
	}
	return Decision{evaluated: true, blocks: blocks}
}

func (d Decision) Allowed() bool {
	return d.evaluated && len(d.blocks) == 0
}

func (d Decision) Blocks() []Reason {
	return append([]Reason(nil), d.blocks...)
}

func (d Decision) Require() error {
	if !d.evaluated {
		return fmt.Errorf("shopbuygate: decision not evaluated")
	}
	if len(d.blocks) == 0 {
		return nil
	}
	parts := make([]string, 0, len(d.blocks))
	for _, block := range d.blocks {
		parts = append(parts, string(block))
	}
	return fmt.Errorf("shopbuygate: blocked: %s", strings.Join(parts, ","))
}
