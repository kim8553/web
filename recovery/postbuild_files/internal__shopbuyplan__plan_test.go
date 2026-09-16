package shopbuyplan

import "testing"

func TestStageDebitsRecoveredCapitalTypes(t *testing.T) {
	base := Currency{Silver: 1000, Gold: 900, SilverCard: 800, SilverTicket: 700}
	for _, tc := range []struct {
		name string
		mode int32
		want Currency
	}{
		{"gold", 0, Currency{Silver: 1000, Gold: 870, SilverCard: 800, SilverTicket: 700}},
		{"silver", 1, Currency{Silver: 970, Gold: 900, SilverCard: 800, SilverTicket: 700}},
		{"silver_card", 2, Currency{Silver: 1000, Gold: 900, SilverCard: 770, SilverTicket: 700}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Stage(base, nil, Input{CapitalType: tc.mode, UnitPrice: 10, Count: 3})
			if err != nil {
				t.Fatal(err)
			}
			if got.CurrencyAfter != tc.want || got.TotalPrice != 30 {
				t.Fatalf("got=%+v", got)
			}
		})
	}
}

func TestStageChoosesSmallestPositiveFreeSlotInSameView(t *testing.T) {
	occupied := []BagSlot{{View: 2, Slot: 1}, {View: 121, Slot: 2}, {View: 2, Slot: 3}, {View: 2, Slot: 0}}
	got, err := Stage(Currency{Gold: 100}, occupied, Input{CapitalType: 0, UnitPrice: 1, Count: 1, RewardView: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.RewardSlot != 2 {
		t.Fatalf("slot=%d want=2", got.RewardSlot)
	}
}

func TestStagePreservesPositiveRequestedSlot(t *testing.T) {
	got, err := Stage(Currency{Gold: 100}, []BagSlot{{View: 2, Slot: 7}}, Input{CapitalType: 0, UnitPrice: 1, Count: 1, RewardView: 2, RequestedSlot: 7})
	if err != nil {
		t.Fatal(err)
	}
	if got.RewardSlot != 7 {
		t.Fatalf("slot=%d want=7", got.RewardSlot)
	}
}

func TestStageFailsClosedForInvalidOrInsufficientInput(t *testing.T) {
	for _, input := range []Input{
		{CapitalType: 3, UnitPrice: 1, Count: 1},
		{CapitalType: 0, UnitPrice: -1, Count: 1},
		{CapitalType: 0, UnitPrice: 1, Count: 0},
		{CapitalType: 0, UnitPrice: 1, Count: 100},
		{CapitalType: 0, UnitPrice: 101, Count: 1},
	} {
		if _, err := Stage(Currency{Gold: 100}, nil, input); err == nil {
			t.Fatalf("expected error for %+v", input)
		}
	}
}

func TestStageDoesNotMutateInputs(t *testing.T) {
	currency := Currency{Gold: 100}
	occupied := []BagSlot{{View: 2, Slot: 1}}
	_, err := Stage(currency, occupied, Input{CapitalType: 0, UnitPrice: 5, Count: 2, RewardView: 2})
	if err != nil {
		t.Fatal(err)
	}
	if currency.Gold != 100 || occupied[0] != (BagSlot{View: 2, Slot: 1}) {
		t.Fatalf("inputs mutated currency=%+v occupied=%+v", currency, occupied)
	}
}
