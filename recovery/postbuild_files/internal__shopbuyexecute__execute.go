package shopbuyexecute

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/shopbuygate"
)

// PersistFirst is the activation-order coordinator for a future ordinary NPC
// shop purchase mutation. It deliberately knows nothing about the client
// selector or payload. The caller must provide an already-proven wire gate and
// one persistence callback that atomically commits BOTH the staged bag and the
// staged currency snapshot.
//
// Required order:
//
//   gate Require -> pure stage -> atomic persist -> live apply -> client publish
//
// A stage or persistence failure therefore cannot change live state or emit a
// client mutation frame. publish happens after the durable/live commit; a
// publish error is returned for caller-driven resynchronization and is not
// represented as a database rollback.
func PersistFirst[T any](
	gate shopbuygate.Decision,
	stage func() (T, error),
	persistAtomic func(staged T) error,
	apply func(staged T),
	publish func(staged T) error,
) (T, error) {
	var zero T
	if stage == nil {
		return zero, fmt.Errorf("shopbuyexecute: stage callback unavailable")
	}
	if persistAtomic == nil {
		return zero, fmt.Errorf("shopbuyexecute: atomic persist callback unavailable")
	}
	if apply == nil {
		return zero, fmt.Errorf("shopbuyexecute: apply callback unavailable")
	}
	if publish == nil {
		return zero, fmt.Errorf("shopbuyexecute: publish callback unavailable")
	}
	if err := gate.Require(); err != nil {
		return zero, fmt.Errorf("shopbuyexecute: gate: %w", err)
	}
	staged, err := stage()
	if err != nil {
		return zero, fmt.Errorf("shopbuyexecute: stage: %w", err)
	}
	if err := persistAtomic(staged); err != nil {
		return zero, fmt.Errorf("shopbuyexecute: persist: %w", err)
	}
	apply(staged)
	if err := publish(staged); err != nil {
		return staged, fmt.Errorf("shopbuyexecute: publish after durable apply: %w", err)
	}
	return staged, nil
}
