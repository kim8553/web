package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentShopExchangeConfigFieldOrder(t *testing.T) {
	definition := shopExchangeDefinition{
		Type:          2,
		AddValue:      "17",
		BindStatus:    3,
		Item:          "item_a,4;item_b,5",
		ConditionType: 1,
		Condition:     "1001,1002",
		Condition2:    "2001",
		Filters:       "2001",
		Prop:          "CapitalType1,99;CapitalType2,1234567890123",
	}
	got, err := encodeShopExchangeConfig(definition, shopExchangeRuntimeBind{ShowBind: 4, ExchangeBind: 5})
	if err != nil {
		t.Fatal(err)
	}
	want := "2|17|3|item_a,4;item_b,5|4|5|1|1001,1002|2001|2001|CapitalType1,99;CapitalType2,1234567890123"
	if got != want {
		t.Fatalf("config=%q want=%q", got, want)
	}
	if fields := strings.Split(got, "|"); len(fields) != 11 {
		t.Fatalf("field count=%d want=11", len(fields))
	}
}

func TestCurrentShopExchangeDefinition13097Shape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exchangeitem.ini")
	if err := os.WriteFile(path, []byte("[13097]\nCondition2=121209\nItem=axe_leader_xt_test_000,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadShopExchangeDefinition(path, 13097)
	if err != nil {
		t.Fatal(err)
	}
	if got.Condition2 != "121209" || got.Item != "axe_leader_xt_test_000,1" {
		t.Fatalf("definition=%+v", got)
	}
	if got.Type != 0 || got.BindStatus != 0 || got.ConditionType != 0 || got.AddValue != "" || got.Filters != "" || got.Prop != "" {
		t.Fatalf("unexpected synthetic fields in definition=%+v", got)
	}
}

func TestCurrentShopExchangeDefinitionFiltersRemainConditionIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exchangeitem.ini")
	data := "[3490]\nCondition2=107028\nItem=item_99wuxue_ww001,100\nFilters=107028\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadShopExchangeDefinition(path, 3490)
	if err != nil {
		t.Fatal(err)
	}
	if got.Filters != "107028" || got.Condition2 != "107028" {
		t.Fatalf("definition=%+v", got)
	}
}

func TestCurrentShopExchangeConfigRejectsInvalidCurrentSubgrammar(t *testing.T) {
	if _, err := encodeShopExchangeConfig(shopExchangeDefinition{Item: "item_without_count"}, shopExchangeRuntimeBind{}); err == nil {
		t.Fatal("invalid Item pair unexpectedly accepted")
	}
	if _, err := encodeShopExchangeConfig(shopExchangeDefinition{Prop: "CapitalType1,not-a-number"}, shopExchangeRuntimeBind{}); err == nil {
		t.Fatal("invalid Prop int64 unexpectedly accepted")
	}
	if _, err := encodeShopExchangeConfig(shopExchangeDefinition{Filters: "107028,x"}, shopExchangeRuntimeBind{}); err == nil {
		t.Fatal("invalid Filters int32 unexpectedly accepted")
	}
}

func TestCurrentShopExchangeFormMessage557(t *testing.T) {
	request := shopExchangeFormRequest{ViewIdent: 61, BindIndex: 7, ShopID: "Shop_Test", Page: 2, Position: 9}
	frame, err := serverShopExchangeFormMessage(request, "0||0|item_a,1|0|0|0||||")
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) < 3 || frame[0] != 0x1e {
		t.Fatalf("frame prefix=%x", frame)
	}
	// Message id 557 is the first typed int32 value after the 0x1e/count header.
	if !strings.Contains(string(frame), "Shop_Test\x00") {
		t.Fatalf("frame does not contain shop id: %x", frame)
	}
}

func TestCurrentShopConditionDetailsMessage508(t *testing.T) {
	frame, err := serverShopConditionDetailsMessage([]shopConditionDetail{
		{ConditionID: 121209, IsNeedShow: true, Satisfied: false},
		{ConditionID: 115321, IsNeedShow: false, Satisfied: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) < 3 || frame[0] != 0x1e {
		t.Fatalf("frame prefix=%x", frame)
	}
}

func TestCurrentShopExchangeDefinitionAcceptsExactCurrentItemTrailingTerminator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exchangeitem.ini")
	data := "[14087]\nItem=Item_xdm_exchange01,440;item_exc_fc_mml,80;\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadShopExchangeDefinition(path, 14087)
	if err != nil {
		t.Fatal(err)
	}
	if got.Item != "Item_xdm_exchange01,440;item_exc_fc_mml,80;" {
		t.Fatalf("Item=%q", got.Item)
	}
	if _, err := encodeShopExchangeConfig(shopExchangeDefinition{Item: "mat_a,1;;mat_b,1"}, shopExchangeRuntimeBind{}); err == nil {
		t.Fatal("embedded empty Item pair must remain rejected")
	}
	if _, err := encodeShopExchangeConfig(shopExchangeDefinition{Prop: "CapitalType1,1;"}, shopExchangeRuntimeBind{}); err == nil {
		t.Fatal("Prop trailing terminator is not authored in exact-current corpus and must remain rejected")
	}
}
