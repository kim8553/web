package main

import (
	"errors"
	"strings"
	"testing"
)

func TestStage31ShopFramePreflightValidRowsKeepOrder(t *testing.T) {
	items := []shopCatalogItem{
		{configID: "ordinary", page: 0, position: 0, priceMode: 1},
		{configID: "exchange", page: 1, position: 0, priceMode: 3, exchangeData: 13097},
	}
	var indexes []uint16
	frames, err := stage31PrepareShopItemFrames("Shop_fixture", items, func(item shopCatalogItem, index uint16) ([]byte, error) {
		indexes = append(indexes, index)
		return []byte(item.configID), nil
	})
	if err != nil || len(frames) != 2 || string(frames[0]) != "ordinary" || string(frames[1]) != "exchange" || len(indexes) != 2 || indexes[0] != 1 || indexes[1] != 501 {
		t.Fatalf("frames=%q indices=%v error=%v", frames, indexes, err)
	}
}

func TestStage31ShopFramePreflightDoesNotInventCapacityLimit(t *testing.T) {
	items := make([]shopCatalogItem, 103)
	for i := range items {
		items[i] = shopCatalogItem{configID: "item", page: 0, position: int32(i)}
	}
	called := 0
	frames, err := stage31PrepareShopItemFrames("Shop_authored_103", items, func(shopCatalogItem, uint16) ([]byte, error) {
		called++
		return []byte{1}, nil
	})
	if err != nil || len(frames) != len(items) || called != len(items) {
		t.Fatalf("capacity should not be guessed; frames=%d calls=%d err=%v", len(frames), called, err)
	}
}

func TestStage31ShopFramePreflightRejectsDuplicateIndex(t *testing.T) {
	items := []shopCatalogItem{{configID: "one", page: 0, position: 0}, {configID: "two", page: 0, position: 0}}
	frames, err := stage31PrepareShopItemFrames("Shop_collision", items, func(item shopCatalogItem, index uint16) ([]byte, error) { return []byte(item.configID), nil })
	if err == nil || !strings.Contains(err.Error(), "duplicate") || frames != nil {
		t.Fatalf("collision frames=%v err=%v", frames, err)
	}
}

func TestStage31ShopFramePreflightRejectsInvalidIndex(t *testing.T) {
	items := []shopCatalogItem{{configID: "one", page: 0, position: 0}, {configID: "bad", page: 0, position: 500}}
	frames, err := stage31PrepareShopItemFrames("Shop_invalid", items, func(item shopCatalogItem, index uint16) ([]byte, error) { return []byte(item.configID), nil })
	if err == nil || !strings.Contains(err.Error(), "invalid current-client") || frames != nil {
		t.Fatalf("invalid frames=%v err=%v", frames, err)
	}
}

func TestStage31ShopFramePreflightRejectsEncodingFailureWithoutFrames(t *testing.T) {
	items := []shopCatalogItem{{configID: "one", page: 0, position: 0}, {configID: "bad", page: 0, position: 1}}
	sentinel := errors.New("encoder refused")
	frames, err := stage31PrepareShopItemFrames("Shop_error", items, func(item shopCatalogItem, index uint16) ([]byte, error) {
		if item.configID == "bad" {
			return nil, sentinel
		}
		return []byte{1}, nil
	})
	if !errors.Is(err, sentinel) || frames != nil {
		t.Fatalf("encoder frames=%v err=%v", frames, err)
	}
}
