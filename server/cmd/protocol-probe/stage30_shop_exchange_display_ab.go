package main

// stage30ShopExchangeDisplayAB is read-only: it changes which already-authored
// item rows are sent in a diagnostic View61. It never authorizes or executes a
// purchase. The default path preserves Stage27's ordinary-only listing.
type stage30ShopExchangeDisplayStats struct {
	Eligible            int
	MissingExchangeData int
	InvalidExchangeSlot int
	Displayed           int
	CapacityBlocked     bool
	CollisionBlocked    bool
}

// stage30ShopExchangeDisplayRows uses only the existing current-client
// ExchangeData/int32 ordinal and objectIndex mapping. The opt-in A/B path
// declines the entire exchange extension when the authored rows cannot be
// represented unambiguously in the existing 100-capacity shop view.
func stage30ShopExchangeDisplayRows(items []shopCatalogItem, enabled bool) ([]shopCatalogItem, stage30ShopExchangeDisplayStats) {
	ordinary := make([]shopCatalogItem, 0, len(items))
	var stats stage30ShopExchangeDisplayStats
	for _, item := range items {
		switch item.priceMode {
		case 0, 1, 2:
			ordinary = append(ordinary, item)
		case 3:
			if item.exchangeData <= 0 {
				stats.MissingExchangeData++
			} else if _, ok := currentShopViewObjectIndex(item); !ok {
				stats.InvalidExchangeSlot++
			} else {
				stats.Eligible++
			}
		}
	}
	if !enabled || stats.Eligible == 0 {
		return ordinary, stats
	}
	if len(ordinary)+stats.Eligible > 100 {
		stats.CapacityBlocked = true
		return ordinary, stats
	}
	all := make([]shopCatalogItem, 0, len(ordinary)+stats.Eligible)
	seen := make(map[uint16]struct{}, len(ordinary)+stats.Eligible)
	for _, item := range items {
		include := false
		switch item.priceMode {
		case 0, 1, 2:
			include = true
		case 3:
			if item.exchangeData > 0 {
				_, include = currentShopViewObjectIndex(item)
			}
		}
		if !include {
			continue
		}
		index, ok := currentShopViewObjectIndex(item)
		if !ok { // ordinary invalid coordinates remain subject to Stage27's error.
			stats.InvalidExchangeSlot++
			stats.CollisionBlocked = true
			return ordinary, stats
		}
		if _, exists := seen[index]; exists {
			stats.CollisionBlocked = true
			return ordinary, stats
		}
		seen[index] = struct{}{}
		all = append(all, item)
		if item.priceMode == 3 {
			stats.Displayed++
		}
	}
	return all, stats
}
