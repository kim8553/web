package main

import "testing"

// The exact-current Lua request shape is understood, but the server-side
// exchange-binding and purchase-commit rules are not. A syntactically valid
// request must not accidentally emit S2C 557 or an apparent purchase success.
func TestCurrentShopExchangeFormAndBuyRemainFailClosed(t *testing.T) {
	cases := []struct {
		name   string
		values []clientCustomValue
	}{
		{
			name: "form request 0x40",
			values: []clientCustomValue{
				{Type: 2, Int32: clientCustomRequestShopExchangeForm},
				{Type: 2, Int32: 121},
				{Type: 2, Int32: 7},
				{Type: 6, Text: "Shop_Test"},
				{Type: 2, Int32: 3},
				{Type: 2, Int32: 9},
			},
		},
		{
			name: "buy request 0x4f",
			values: []clientCustomValue{
				{Type: 2, Int32: clientCustomExchangeFromShop},
				{Type: 6, Text: "Shop_Test"},
				{Type: 2, Int32: 3},
				{Type: 2, Int32: 9},
				{Type: 2, Int32: 4},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			link := &captureMessageConnection{}
			matched, err := handleShopExchangeContract(link, nil, clientCustomMessage{Values: tc.values}, "test")
			if err != nil || !matched {
				t.Fatalf("matched=%t err=%v", matched, err)
			}
			if got := len(link.Frames()); got != 0 {
				t.Fatalf("unproven exchange request emitted %d server frames, want zero", got)
			}
		})
	}
}

func TestCurrentShopExchangeFormMessage557TypedFieldSequence(t *testing.T) {
	request := shopExchangeFormRequest{ViewIdent: 121, BindIndex: 7, ShopID: "Shop_Test", Page: 3, Position: 9}
	const config = "2|17|3|item_a,4|4|5|1|1001|2001|2001|CapitalType1,99"
	frame, err := serverShopExchangeFormMessage(request, config)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := parseClientCustomMessage(frame)
	if err != nil {
		t.Fatalf("decode typed response: %v", err)
	}
	if decoded.Opcode != 0x1e || len(decoded.Values) != 7 {
		t.Fatalf("custom response opcode=%#x value count=%d, want 0x1e/7", decoded.Opcode, len(decoded.Values))
	}
	wantTypes := []byte{2, 2, 2, 6, 2, 2, 6}
	wantInts := map[int]int32{0: 557, 1: 121, 2: 7, 4: 3, 5: 9}
	wantStrings := map[int]string{3: "Shop_Test", 6: config}
	for i, value := range decoded.Values {
		if value.Type != wantTypes[i] {
			t.Fatalf("typed response value %d type=%d want=%d", i, value.Type, wantTypes[i])
		}
		if want, ok := wantInts[i]; ok && value.Int32 != want {
			t.Fatalf("typed response value %d int=%d want=%d", i, value.Int32, want)
		}
		if want, ok := wantStrings[i]; ok && value.Text != want {
			t.Fatalf("typed response value %d text=%q want=%q", i, value.Text, want)
		}
	}
}
