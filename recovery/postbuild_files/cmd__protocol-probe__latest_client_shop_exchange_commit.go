package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/exchangeplan"
	"github.com/local/9yin-go-server/internal/role"
)

// commitShopExchangeReplacementPersistenceFirst is a dormant transaction helper.
// It deliberately has no 0x4F handler wiring and emits no client frames.
//
// The player's bag mutex is held across stale-snapshot validation, pure staging,
// persistence, and the final in-memory swap. This prevents another bag mutation
// from racing between a successful Save and the live swap. Persistence receives
// its own slice copy so file-backed bagStore implementations cannot alias the
// live player slice after commit.
func commitShopExchangeReplacementPersistenceFirst(
	roleID role.RoleID,
	bagStore bagStoreIface,
	itemCatalog *itemCatalog,
	player *playerActor,
	expectedSnapshot []bagItem,
	plan exchangeplan.ReplacementPlan,
) ([]bagItem, error) {
	if roleID == 0 {
		return nil, fmt.Errorf("shop exchange commit: zero role id")
	}
	if bagStore == nil {
		return nil, fmt.Errorf("shop exchange commit: bag store unavailable")
	}
	if player == nil {
		return nil, fmt.Errorf("shop exchange commit: player unavailable")
	}

	player.mu.Lock()
	defer player.mu.Unlock()

	if !sameShopExchangeBagSnapshot(player.bagItems, expectedSnapshot) {
		return nil, fmt.Errorf("shop exchange commit: live bag changed since staging")
	}

	staged, err := applyShopExchangeReplacementToSnapshot(itemCatalog, player.bagItems, plan)
	if err != nil {
		return nil, err
	}

	// Keep persistence and live memory on distinct backing arrays. bagStore.Save
	// retains the supplied slice in the file-backed implementation.
	persisted := append([]bagItem(nil), staged...)
	if err := bagStore.Save(roleID, persisted); err != nil {
		return nil, fmt.Errorf("shop exchange commit: persist bag: %w", err)
	}

	player.bagItems = append([]bagItem(nil), staged...)
	return append([]bagItem(nil), staged...), nil
}

func sameShopExchangeBagSnapshot(left, right []bagItem) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
