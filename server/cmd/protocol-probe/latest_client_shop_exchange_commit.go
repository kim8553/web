package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/exchangecommit"
	"github.com/local/9yin-go-server/internal/exchangeplan"
	"github.com/local/9yin-go-server/internal/role"
)

// commitShopExchangeReplacementPersistenceFirst is a dormant transaction helper.
// It deliberately has no 0x4F handler wiring and emits no client frames.
//
// The player's bag mutex is held across stale-snapshot validation, pure staging,
// persistence, and the final in-memory swap. This prevents another bag mutation
// from racing between a successful Save and the live swap. The persistence-first
// ordering itself lives in internal/exchangecommit so it can be runtime-tested
// without importing the resource-bound protocol package.
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

	committed, err := exchangecommit.PersistFirstSlice(
		player.bagItems,
		func(snapshot []bagItem) ([]bagItem, error) {
			return applyShopExchangeReplacementToSnapshot(itemCatalog, snapshot, plan)
		},
		func(staged []bagItem) error {
			return bagStore.Save(roleID, staged)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("shop exchange commit: %w", err)
	}

	player.bagItems = committed
	return append([]bagItem(nil), committed...), nil
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
