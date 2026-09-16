package main

import "fmt"

// stage31PrepareShopItemFrames validates and encodes the entire, already
// selected shop display before the first view frame is written. This does not
// change item selection, client properties, currency, inventory or purchases.
// The view declares Capacity=100, but its relationship to authored page indices
// or the number of add frames is not verified; do not invent a count limit.
func stage31PrepareShopItemFrames(shopID string, items []shopCatalogItem, encode func(shopCatalogItem, uint16) ([]byte, error)) ([][]byte, error) {
	seen := make(map[uint16]struct{}, len(items))
	frames := make([][]byte, 0, len(items))
	for _, item := range items {
		index, ok := currentShopViewObjectIndex(item)
		if !ok {
			return nil, fmt.Errorf("shop %q item %q has invalid current-client page=%d position=%d; refusing partial view", shopID, item.configID, item.page, item.position)
		}
		if _, duplicate := seen[index]; duplicate {
			return nil, fmt.Errorf("shop %q has duplicate current-client object index=%d item=%q; refusing partial view", shopID, index, item.configID)
		}
		seen[index] = struct{}{}
		frame, err := encode(item, index)
		if err != nil {
			return nil, fmt.Errorf("encode shop %q item %q: %w", shopID, item.configID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
