package exchangecommit

import "fmt"

// PersistFirstSlice stages against a private copy, persists another private
// copy, and returns a third copy only after persistence succeeds. The caller
// decides when to publish the returned state. This package has no game/resource
// dependencies so the persistence-first ordering can be executed in CI even
// when the protocol package's exact-current resource corpus is unavailable.
func PersistFirstSlice[T any](current []T, stage func([]T) ([]T, error), persist func([]T) error) ([]T, error) {
	if stage == nil {
		return nil, fmt.Errorf("exchangecommit: nil stage function")
	}
	if persist == nil {
		return nil, fmt.Errorf("exchangecommit: nil persist function")
	}

	working := append([]T(nil), current...)
	staged, err := stage(working)
	if err != nil {
		return nil, fmt.Errorf("exchangecommit: stage: %w", err)
	}

	persisted := append([]T(nil), staged...)
	if err := persist(persisted); err != nil {
		return nil, fmt.Errorf("exchangecommit: persist: %w", err)
	}

	return append([]T(nil), staged...), nil
}
