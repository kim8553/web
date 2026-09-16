package shopbuycommit

import (
	"fmt"
	"sync"

	"github.com/local/9yin-go-server/internal/shopbuygate"
)

// PersistenceFirstUnderLock serializes one ordinary-shop durable/live commit
// against all other state protected by the same player mutex. The database and
// process memory are not one atomic transaction: persistence is committed first,
// then the in-memory state is applied while the mutex is still held. Client
// publication is intentionally outside this helper and must occur only after it
// returns successfully.
func PersistenceFirstUnderLock[T any](
	mu *sync.Mutex,
	gate shopbuygate.Decision,
	stage func() (T, error),
	persist func(T) error,
	apply func(T),
) (T, error) {
	var zero T
	if mu == nil {
		return zero, fmt.Errorf("shopbuycommit: nil mutex")
	}
	if stage == nil {
		return zero, fmt.Errorf("shopbuycommit: nil stage")
	}
	if persist == nil {
		return zero, fmt.Errorf("shopbuycommit: nil persist")
	}
	if apply == nil {
		return zero, fmt.Errorf("shopbuycommit: nil apply")
	}
	if err := gate.Require(); err != nil {
		return zero, err
	}

	mu.Lock()
	defer mu.Unlock()

	staged, err := stage()
	if err != nil {
		return zero, fmt.Errorf("shopbuycommit: stage: %w", err)
	}
	if err := persist(staged); err != nil {
		return zero, fmt.Errorf("shopbuycommit: persist: %w", err)
	}
	apply(staged)
	return staged, nil
}
