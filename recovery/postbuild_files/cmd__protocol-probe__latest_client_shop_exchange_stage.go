package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/exchangeplan"
)

// currentShopExchangeInventoryState converts one immutable bag snapshot into
// the stable identity model used by the atomic replacement planner.
func currentShopExchangeInventoryState(items []bagItem) []exchangeplan.InventoryState {
	result := make([]exchangeplan.InventoryState, 0, len(items))
	for index, item := range items {
		container, ok := latestClientCurrentBagContainer(item.ViewID)
		if !ok {
			continue
		}
		result = append(result, exchangeplan.InventoryState{
			Index:      index,
			Container:  container,
			Slot:       item.Slot,
			ConfigID:   item.ConfigID,
			Amount:     item.Amount,
			MaxAmount:  item.MaxAmount,
			BindStatus: item.BindStatus,
		})
	}
	return result
}

// stageShopExchangeAtomicReplacement is side-effect free. resultBindStatus is
// deliberately explicit because the exact-current final persisted binding
// precedence is still unresolved; this helper must not invent that rule.
func stageShopExchangeAtomicReplacement(itemCatalog *itemCatalog, bagSnapshot []bagItem, listing shopCatalogItem, materialPlan exchangeplan.BatchPlan, outputAmount int64, resultBindStatus int32) (exchangeplan.ReplacementPlan, error) {
	if itemCatalog == nil {
		return exchangeplan.ReplacementPlan{}, fmt.Errorf("shop exchange stage: item catalog unavailable")
	}
	staticItem, ok := itemCatalog.Lookup(listing.configID)
	if !ok {
		return exchangeplan.ReplacementPlan{}, fmt.Errorf("shop exchange stage: output %q missing from item catalog", listing.configID)
	}
	container, ok := latestClientCurrentBagContainer(staticItem.ViewID)
	if !ok {
		return exchangeplan.ReplacementPlan{}, fmt.Errorf("shop exchange stage: output %q has unsupported ViewID %d", listing.configID, staticItem.ViewID)
	}
	capacity, ok := currentShopExchangeContainerCapacity(container)
	if !ok {
		return exchangeplan.ReplacementPlan{}, fmt.Errorf("shop exchange stage: output container %d capacity unresolved", container)
	}
	return exchangeplan.PlanAtomicReplacement(
		currentShopExchangeInventoryState(bagSnapshot),
		materialPlan,
		exchangeplan.OutputSpec{
			ConfigID:   listing.configID,
			Container:  container,
			Amount:     outputAmount,
			MaxAmount:  staticItem.MaxAmount,
			BindStatus: resultBindStatus,
		},
		capacity,
	)
}
