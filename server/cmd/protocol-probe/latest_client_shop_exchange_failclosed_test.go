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

func TestResolveAuthorizedCurrentShopExchangeBuyAuthorityRequiresSafeAuthenticatedProjection(t *testing.T) {
	dir := t.TempDir()
	shopPath := filepath.Join(dir, "shop.ini")
	exchangePath := filepath.Join(dir, "exchangeitem.ini")
	if err := os.WriteFile(shopPath, []byte("[shop_mode3]\nType=7\nPageInfo=all\n0=item_exchange,2,3,0,0,0,4,70123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exchangePath, []byte("[70123]\nType=2\nBindStatus=1\nItem=item_cost,2\nCondition=10\nCondition2=20\nFilters=20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := shopExchangeBuyRequest{ShopID: "shop_mode3", Page: 0, Position: 5, Count: 2}
	authority := &currentShopConditionAuthority{
		definitions: map[int32]exchangeConditionSpec{
			70123: {Condition: "10", Condition2: "20", Filters: "20"},
		},
		audit: shopExchangeConditionCapabilityAudit{ByExchangeData: map[int32]exchangeConditionCapability{
			70123: {ExchangeData: 70123, Safe: true, Leaves: []int32{10, 20}},
		}},
	}

	got, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, request)
	if err != nil {
		t.Fatalf("authorized resolve: %v", err)
	}
	if !ok {
		t.Fatal("SAFE authenticated mode3 row was rejected")
	}
	if got.Selection.Item.configID != "item_exchange" || got.Selection.Item.exchangeData != 70123 {
		t.Fatalf("selection=%+v", got.Selection.Item)
	}
	if got.Definition.BindStatus != 1 || got.Definition.Item != "item_cost,2" {
		t.Fatalf("definition=%+v", got.Definition)
	}
	if !got.Capability.Safe || len(got.Capability.Leaves) != 2 {
		t.Fatalf("capability=%+v", got.Capability)
	}

	authority.audit.ByExchangeData[70123] = exchangeConditionCapability{ExchangeData: 70123, Safe: false}
	if _, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, request); err != nil || ok {
		t.Fatalf("unsafe ExchangeData accepted: ok=%t err=%v", ok, err)
	}

	authority.audit.ByExchangeData[70123] = exchangeConditionCapability{ExchangeData: 70123, Safe: true}
	authority.definitions[70123] = exchangeConditionSpec{Condition: "999", Condition2: "20", Filters: "20"}
	if _, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, request); err == nil || ok {
		t.Fatalf("projection drift accepted: ok=%t err=%v", ok, err)
	}
}
