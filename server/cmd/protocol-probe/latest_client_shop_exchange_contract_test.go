package main

import "testing"

func TestCurrentShopExchangeFormRequestContract(t *testing.T) {
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 64},
		{Type: 2, Int32: 121},
		{Type: 2, Int32: 7},
		{Type: 6, Text: "Shop_Test"},
		{Type: 2, Int32: 3},
		{Type: 2, Int32: 9},
	}}
	got, matched, err := parseShopExchangeFormRequest(custom)
	if err != nil || !matched {
		t.Fatalf("matched=%t err=%v", matched, err)
	}
	if got.ViewIdent != 121 || got.BindIndex != 7 || got.ShopID != "Shop_Test" || got.Page != 3 || got.Position != 9 {
		t.Fatalf("request=%+v", got)
	}
}

func TestCurrentShopExchangeBuyRequestContract(t *testing.T) {
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 79},
		{Type: 6, Text: "Shop_Test"},
		{Type: 2, Int32: 3},
		{Type: 2, Int32: 9},
		{Type: 2, Int32: 4},
	}}
	got, matched, err := parseShopExchangeBuyRequest(custom)
	if err != nil || !matched {
		t.Fatalf("matched=%t err=%v", matched, err)
	}
	if got.ShopID != "Shop_Test" || got.Page != 3 || got.Position != 9 || got.Count != 4 {
		t.Fatalf("request=%+v", got)
	}
}

func TestCurrentShopExchangeBuyDoesNotAcceptExchangeDataOnWire(t *testing.T) {
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 79},
		{Type: 6, Text: "Shop_Test"},
		{Type: 2, Int32: 3},
		{Type: 2, Int32: 9},
		{Type: 2, Int32: 4},
		{Type: 2, Int32: 13097},
	}}
	_, matched, err := parseShopExchangeBuyRequest(custom)
	if !matched || err == nil {
		t.Fatalf("matched=%t err=%v", matched, err)
	}
}

func TestCurrentShopConditionDetailsRequestContract(t *testing.T) {
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 69},
		{Type: 2, Int32: 13097},
	}}
	got, matched, err := parseShopConditionDetailsRequest(custom)
	if err != nil || !matched {
		t.Fatalf("matched=%t err=%v", matched, err)
	}
	if got.ExchangeIndex != 13097 {
		t.Fatalf("request=%+v", got)
	}
}
