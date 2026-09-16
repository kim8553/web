package main

import "testing"

func TestStage30ExchangeABOffPreservesStage27OrdinaryOnly(t *testing.T) {
	items := []shopCatalogItem{{configID: "first", page: 0, position: 0, priceMode: 1}, {configID: "exchange", page: 0, position: 1, priceMode: 3, exchangeData: 13097}, {configID: "last", page: 0, position: 2, priceMode: 2}}
	got, stats := stage30ShopExchangeDisplayRows(items, false)
	if len(got) != 2 || got[0].configID != "first" || got[1].configID != "last" || stats.Displayed != 0 || stats.Eligible != 1 {
		t.Fatalf("disabled rows=%+v stats=%+v", got, stats)
	}
}
func TestStage30ExchangeABIncludesOnlyAuthoredValidDistinctRows(t *testing.T) {
	items := []shopCatalogItem{{configID: "first", page: 0, position: 0, priceMode: 1}, {configID: "exchange", page: 1, position: 0, priceMode: 3, exchangeData: 13097}, {configID: "missing", page: 1, position: 1, priceMode: 3, exchangeData: 0}, {configID: "other", page: 1, position: 2, priceMode: 3, exchangeData: 13098}}
	got, stats := stage30ShopExchangeDisplayRows(items, true)
	if len(got) != 3 || got[0].configID != "first" || got[1].configID != "exchange" || got[2].configID != "other" || stats.Displayed != 2 || stats.MissingExchangeData != 1 || stats.CapacityBlocked || stats.CollisionBlocked {
		t.Fatalf("enabled rows=%+v stats=%+v", got, stats)
	}
}
func TestStage30ExchangeABRejectsCollisionWithoutDroppingOrdinary(t *testing.T) {
	items := []shopCatalogItem{{configID: "ordinary", page: 0, position: 0, priceMode: 2}, {configID: "duplicate", page: 0, position: 0, priceMode: 3, exchangeData: 13097}}
	got, stats := stage30ShopExchangeDisplayRows(items, true)
	if len(got) != 1 || got[0].configID != "ordinary" || !stats.CollisionBlocked || stats.Displayed != 0 {
		t.Fatalf("collision rows=%+v stats=%+v", got, stats)
	}
}
func TestStage30ExchangeABRejectsOverCapacityWithoutDroppingOrdinary(t *testing.T) {
	items := []shopCatalogItem{{configID: "ordinary", page: 0, position: 0, priceMode: 1}}
	for i := int32(1); i <= 100; i++ {
		items = append(items, shopCatalogItem{configID: "exchange", page: 0, position: i, priceMode: 3, exchangeData: 13097})
	}
	got, stats := stage30ShopExchangeDisplayRows(items, true)
	if len(got) != 1 || got[0].configID != "ordinary" || !stats.CapacityBlocked || stats.Displayed != 0 {
		t.Fatalf("capacity rows=%d stats=%+v", len(got), stats)
	}
}
func TestStage30ExchangeABRejectsInvalidExchangeCoordinate(t *testing.T) {
	items := []shopCatalogItem{{configID: "ordinary", page: 0, position: 0, priceMode: 1}, {configID: "invalid", page: 0, position: 500, priceMode: 3, exchangeData: 13097}}
	got, stats := stage30ShopExchangeDisplayRows(items, true)
	if len(got) != 1 || got[0].configID != "ordinary" || stats.InvalidExchangeSlot != 1 || stats.Displayed != 0 {
		t.Fatalf("invalid rows=%+v stats=%+v", got, stats)
	}
}
