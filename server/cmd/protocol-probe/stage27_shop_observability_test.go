package main

import "testing"

func TestStage27ShopDisplaySummaryIsReadOnly(t *testing.T) {
	rows := []shopCatalogItem{
		{priceMode: 0, page: 0, position: 0},
		{priceMode: 1, page: 0, position: 2},
		{priceMode: 2, page: 1, position: 0},
		{priceMode: 3, exchangeData: 13097},
		{priceMode: 3},
		{priceMode: 11},
		{priceMode: 1, page: -1, position: 0},
	}
	stats := summarizeStage27ShopDisplayRows(rows)
	if stats.Ordinary != 4 || stats.ExchangeSkipped != 2 || stats.ExchangeMissingData != 1 || stats.UnsupportedSkipped != 1 || stats.InvalidOrdinaryCoordinates != 1 {
		t.Fatalf("display summary mismatch: %+v", stats)
	}
	if len(rows) != 7 || rows[3].exchangeData != 13097 || rows[6].page != -1 {
		t.Fatalf("summary modified catalog: %#v", rows)
	}
}

func TestStage27ShopDisplaySummaryEmpty(t *testing.T) {
	if got := summarizeStage27ShopDisplayRows(nil); got != (stage27ShopDisplayRows{}) {
		t.Fatalf("empty summary=%+v", got)
	}
}
