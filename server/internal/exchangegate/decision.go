package exchangegate

import (
	"fmt"
	"reflect"
	"strings"
)

// Reason identifies one independent fail-closed prerequisite for a shop
// exchange mutation. Reasons are safety gates, not gameplay precedence rules.
type Reason string

const (
	ConditionUnsupported   Reason = "condition_unsupported"
	ConditionUnsatisfied   Reason = "condition_unsatisfied"
	PropertyUnsupported    Reason = "property_unsupported"
	PropertyUnsatisfied    Reason = "property_unsatisfied"
	MaterialsUnsatisfied   Reason = "materials_unsatisfied"
	CapacityUnsupported    Reason = "capacity_unsupported"
	CapacityUnsatisfied    Reason = "capacity_unsatisfied"
	BindingUnknown         Reason = "binding_unknown"
	PersistenceUnavailable Reason = "persistence_unavailable"
)

type Evidence struct {
	ConditionSupported bool
	ConditionSatisfied bool
	PropertySupported  bool
	PropertySatisfied  bool
	MaterialsSatisfied bool
	CapacitySupported  bool
	CapacitySatisfied  bool
	BindingKnown       bool
	PersistenceReady   bool
}

type Decision struct {
	evaluated bool
	blocks    []Reason
}

func Evaluate(e Evidence) Decision {
	blocks := make([]Reason, 0, 9)
	if !e.ConditionSupported {
		blocks = append(blocks, ConditionUnsupported)
	} else if !e.ConditionSatisfied {
		blocks = append(blocks, ConditionUnsatisfied)
	}
	if !e.PropertySupported {
		blocks = append(blocks, PropertyUnsupported)
	} else if !e.PropertySatisfied {
		blocks = append(blocks, PropertyUnsatisfied)
	}
	if !e.MaterialsSatisfied {
		blocks = append(blocks, MaterialsUnsatisfied)
	}
	if !e.CapacitySupported {
		blocks = append(blocks, CapacityUnsupported)
	} else if !e.CapacitySatisfied {
		blocks = append(blocks, CapacityUnsatisfied)
	}
	if !e.BindingKnown {
		blocks = append(blocks, BindingUnknown)
	}
	if !e.PersistenceReady {
		blocks = append(blocks, PersistenceUnavailable)
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
		return fmt.Errorf("exchangegate: decision not evaluated")
	}
	if len(d.blocks) == 0 {
		return nil
	}
	parts := make([]string, 0, len(d.blocks))
	for _, block := range d.blocks {
		parts = append(parts, string(block))
	}
	return fmt.Errorf("exchangegate: blocked: %s", strings.Join(parts, ","))
}

// PersistenceTargetReady rejects both a nil interface and an interface that
// contains a typed-nil reference value. This keeps persistence readiness from
// becoming true merely because an interface value itself is non-nil.
func PersistenceTargetReady(target any) bool {
	if target == nil {
		return false
	}
	value := reflect.ValueOf(target)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !value.IsNil()
	default:
		return true
	}
}
