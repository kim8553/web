package main

import (
	"os"
	"path/filepath"
	"testing"
)

// This is an offline preflight regression only. A successful authority lookup
// must never be mistaken for permission to debit materials or grant an item.
func TestCurrentShopExchangeBuyWireRejectsMalformedTypedRequests(t *testing.T) {
	valid := []clientCustomValue{
		{Type: 2, Int32: clientCustomExchangeFromShop},
		{Type: 6, Text: "shop_exchange"},
		{Type: 2, Int32: 0},
		{Type: 2, Int32: 5},
		{Type: 2, Int32: 1},
	}
	for _, tc := range []struct {
		name   string
		mutate func([]clientCustomValue) []clientCustomValue
	}{
		{"missing-count", func(v []clientCustomValue) []clientCustomValue { return v[:4] }},
		{"extra-exchange-data", func(v []clientCustomValue) []clientCustomValue {
			return append(v, clientCustomValue{Type: 2, Int32: 70123})
		}},
		{"string-selector", func(v []clientCustomValue) []clientCustomValue {
			v[0] = clientCustomValue{Type: 6, Text: "79"}
			return v
		}},
		{"numeric-shop-id", func(v []clientCustomValue) []clientCustomValue {
			v[1] = clientCustomValue{Type: 2, Int32: 70123}
			return v
		}},
		{"float-page", func(v []clientCustomValue) []clientCustomValue {
			v[2] = clientCustomValue{Type: 4, Float32: 0}
			return v
		}},
		{"string-position", func(v []clientCustomValue) []clientCustomValue {
			v[3] = clientCustomValue{Type: 6, Text: "5"}
			return v
		}},
		{"int64-count", func(v []clientCustomValue) []clientCustomValue {
			v[4] = clientCustomValue{Type: 3, Int64: 1}
			return v
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := append([]clientCustomValue(nil), valid...)
			values = tc.mutate(values)
			_, matched, err := parseShopExchangeBuyRequest(clientCustomMessage{Values: values})
			if tc.name == "string-selector" {
				if matched || err != nil {
					t.Fatalf("non-0x4f selector matched=%t err=%v", matched, err)
				}
				return
			}
			if !matched || err == nil {
				t.Fatalf("malformed 0x4f matched=%t err=%v", matched, err)
			}
		})
	}
}

func TestCurrentShopExchangeBuyPreflightRejectsUnauthoredCoordinatesAndDefinitionDrift(t *testing.T) {
	dir := t.TempDir()
	shopPath := filepath.Join(dir, "shop.ini")
	exchangePath := filepath.Join(dir, "exchangeitem.ini")
	shopData := []byte("[shop_exchange]\nType=7\nPageInfo=all\n" +
		"0=item_exchange,2,3,0,0,0,4,70123\n" +
		"0=item_other,1,1,99,0,0,6,0\n" +
		"0=item_missing,1,3,0,0,0,7,0\n")
	exchangeData := []byte("[70123]\nType=2\nBindStatus=1\nItem=material,2\nCondition=10\nCondition2=20\nFilters=20\n")
	if err := os.WriteFile(shopPath, shopData, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exchangePath, exchangeData, 0o600); err != nil {
		t.Fatal(err)
	}
	authority := &currentShopConditionAuthority{
		definitions: map[int32]exchangeConditionSpec{
			70123: {Condition: "10", Condition2: "20", Filters: "20"},
		},
		audit: shopExchangeConditionCapabilityAudit{ByExchangeData: map[int32]exchangeConditionCapability{
			70123: {ExchangeData: 70123, Safe: true, Leaves: []int32{10, 20}},
		}},
	}
	valid := shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 5, Count: 2}
	for _, tc := range []struct {
		name    string
		request shopExchangeBuyRequest
	}{
		{"empty-shop", shopExchangeBuyRequest{Page: 0, Position: 5, Count: 2}},
		{"wrong-page", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 1, Position: 5, Count: 2}},
		{"negative-page", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: -1, Position: 5, Count: 2}},
		{"zero-position", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 0, Count: 2}},
		{"wrong-position", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 4, Count: 2}},
		{"ordinary-listing", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 7, Count: 2}},
		{"missing-exchange-data", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 8, Count: 2}},
		{"zero-count", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 5, Count: 0}},
		{"negative-count", shopExchangeBuyRequest{ShopID: "shop_exchange", Page: 0, Position: 5, Count: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, tc.request); ok || err != nil {
				t.Fatalf("invalid selection authorized=%t err=%v", ok, err)
			}
		})
	}
	for attempt := 0; attempt < 2; attempt++ {
		got, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, valid)
		if err != nil || !ok {
			t.Fatalf("valid preflight attempt=%d authorized=%t err=%v", attempt, ok, err)
		}
		if got.Selection.Item.exchangeData != 70123 || got.Selection.Item.configID != "item_exchange" || got.Selection.Request.Count != 2 {
			t.Fatalf("preflight resolved wrong authored row: %+v", got.Selection)
		}
	}
	if data, err := os.ReadFile(shopPath); err != nil || string(data) != string(shopData) {
		t.Fatalf("preflight changed shop fixture: err=%v", err)
	}
	if data, err := os.ReadFile(exchangePath); err != nil || string(data) != string(exchangeData) {
		t.Fatalf("preflight changed ExchangeItem fixture: err=%v", err)
	}
	authority.audit.ByExchangeData[70123] = exchangeConditionCapability{ExchangeData: 70123, Safe: false}
	if _, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, valid); ok || err != nil {
		t.Fatalf("unsafe capability accepted=%t err=%v", ok, err)
	}
	authority.audit.ByExchangeData[70123] = exchangeConditionCapability{ExchangeData: 70123, Safe: true}
	authority.definitions[70123] = exchangeConditionSpec{Condition: "999", Condition2: "20", Filters: "20"}
	if _, ok, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath, authority, valid); ok || err == nil {
		t.Fatalf("drifted ExchangeItem projection accepted=%t err=%v", ok, err)
	}
}
