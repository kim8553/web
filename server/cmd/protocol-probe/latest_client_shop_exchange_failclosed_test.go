package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentShopExchangeFormAndBuyRemainFailClosed(t *testing.T) {
	connection := &captureMessageConnection{}
	player := newPlayerActor("test", 0)

	formRequest := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomRequestShopExchangeForm},
		{Type: 2, Int32: 1001},
		{Type: 2, Int32: 2},
		{Type: 6, Text: "shop_test"},
		{Type: 2, Int32: 0},
		{Type: 2, Int32: 1},
	}}
	matched, err := handleShopExchangeContract(connection, player, formRequest, "test")
	if err != nil {
		t.Fatalf("form request: %v", err)
	}
	if !matched {
		t.Fatal("form request was not matched")
	}
	if frames := connection.Frames(); len(frames) != 0 {
		t.Fatalf("form request emitted %d frames while binding rule unresolved", len(frames))
	}

	buyRequest := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomExchangeFromShop},
		{Type: 6, Text: "shop_test"},
		{Type: 2, Int32: 0},
		{Type: 2, Int32: 1},
		{Type: 2, Int32: 1},
	}}
	matched, err = handleShopExchangeContract(connection, player, buyRequest, "test")
	if err != nil {
		t.Fatalf("buy request: %v", err)
	}
	if !matched {
		t.Fatal("buy request was not matched")
	}
	if frames := connection.Frames(); len(frames) != 0 {
		t.Fatalf("buy request emitted %d frames while commit path unresolved", len(frames))
	}
}

func TestResolveCurrentShopExchangeBuySelectionReResolvesAuthoredMode3Row(t *testing.T) {
	dir := t.TempDir()
	shopPath := filepath.Join(dir, "shop.ini")
	data := []byte("[shop_mode3]\nType=7\nPageInfo=all\n0=item_exchange,2,3,0,0,0,4,70123\n" +
		"[shop_normal]\nType=7\nPageInfo=all\n0=item_normal,1,0,99,0,0,5,0\n")
	if err := os.WriteFile(shopPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	selection, ok, err := resolveCurrentShopExchangeBuySelection(shopPath, shopExchangeBuyRequest{
		ShopID: "shop_mode3", Page: 0, Position: 5, Count: 3,
	})
	if err != nil {
		t.Fatalf("resolve mode3: %v", err)
	}
	if !ok {
		t.Fatal("authored mode3 row was not resolved")
	}
	if selection.Item.configID != "item_exchange" || selection.Item.exchangeData != 70123 || selection.Item.priceMode != 3 {
		t.Fatalf("unexpected selection: %+v", selection.Item)
	}
	if selection.Request.Count != 3 {
		t.Fatalf("request count drifted: %d", selection.Request.Count)
	}

	if _, ok, err := resolveCurrentShopExchangeBuySelection(shopPath, shopExchangeBuyRequest{
		ShopID: "shop_mode3", Page: 0, Position: 4, Count: 1,
	}); err != nil || ok {
		t.Fatalf("un-authored coordinate accepted: ok=%t err=%v", ok, err)
	}
	if _, ok, err := resolveCurrentShopExchangeBuySelection(shopPath, shopExchangeBuyRequest{
		ShopID: "shop_normal", Page: 0, Position: 6, Count: 1,
	}); err != nil || ok {
		t.Fatalf("non-mode3 row accepted as exchange purchase: ok=%t err=%v", ok, err)
	}
	if _, ok, err := resolveCurrentShopExchangeBuySelection(shopPath, shopExchangeBuyRequest{
		ShopID: "shop_mode3", Page: 0, Position: 5, Count: 0,
	}); err != nil || ok {
		t.Fatalf("non-positive count accepted: ok=%t err=%v", ok, err)
	}
}
