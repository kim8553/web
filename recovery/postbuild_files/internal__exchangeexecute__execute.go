package exchangeexecute

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/exchangebinding"
	"github.com/local/9yin-go-server/internal/exchangegate"
)

// PersistFirst is the only activation-order coordinator for a future shop
// exchange mutation. It does not know any gameplay binding precedence; callers
// must supply an already-authoritative exchangebinding.Decision.
//
// Required order:
//   gate Require -> binding Require -> pure stage -> persist -> live apply.
// Any failure before apply leaves live state untouched when stage is pure.
func PersistFirst[T any](
	gate exchangegate.Decision,
	binding exchangebinding.Decision,
	stage func(bindStatus int32) (T, error),
	persist func(staged T) error,
	apply func(staged T),
) (T, error) {
	var zero T
	if stage == nil {
		return zero, fmt.Errorf("exchangeexecute: stage callback unavailable")
	}
	if persist == nil {
		return zero, fmt.Errorf("exchangeexecute: persist callback unavailable")
	}
	if apply == nil {
		return zero, fmt.Errorf("exchangeexecute: apply callback unavailable")
	}
	if err := gate.Require(); err != nil {
		return zero, fmt.Errorf("exchangeexecute: gate: %w", err)
	}
	bindStatus, err := binding.Require()
	if err != nil {
		return zero, fmt.Errorf("exchangeexecute: binding: %w", err)
	}
	staged, err := stage(bindStatus)
	if err != nil {
		return zero, fmt.Errorf("exchangeexecute: stage: %w", err)
	}
	if err := persist(staged); err != nil {
		return zero, fmt.Errorf("exchangeexecute: persist: %w", err)
	}
	apply(staged)
	return staged, nil
}
