package shopbuyplan

import "fmt"

type Currency struct {
	Silver       int32
	Gold         int32
	SilverCard   int32
	SilverTicket int32
}

type BagSlot struct {
	View uint16
	Slot int32
}

type Input struct {
	CapitalType   int32
	UnitPrice     int32
	Count         int32
	RewardView    uint16
	RequestedSlot int32
}

type Plan struct {
	CurrencyAfter Currency
	RewardSlot    int32
	TotalPrice    int64
}

// Stage reproduces only the already-recovered ordinary-shop state transition:
// capital type 0/1/2 debits gold/silver/silver-card respectively, and a reward
// without an authored slot receives the smallest positive free slot in its bag
// view. It does not infer the client selector, stacking, capacity, or binding.
func Stage(currency Currency, occupied []BagSlot, input Input) (Plan, error) {
	var plan Plan
	if input.Count < 1 || input.Count > 99 {
		return plan, fmt.Errorf("shopbuyplan: count %d outside 1..99", input.Count)
	}
	if input.UnitPrice < 0 {
		return plan, fmt.Errorf("shopbuyplan: negative unit price %d", input.UnitPrice)
	}
	total := int64(input.UnitPrice) * int64(input.Count)
	plan.CurrencyAfter = currency
	plan.TotalPrice = total

	debit := func(have int32, name string) (int32, error) {
		if int64(have) < total {
			return 0, fmt.Errorf("shopbuyplan: insufficient %s: need %d have %d", name, total, have)
		}
		return have - int32(total), nil
	}

	var err error
	switch input.CapitalType {
	case 0:
		plan.CurrencyAfter.Gold, err = debit(currency.Gold, "gold")
	case 1:
		plan.CurrencyAfter.Silver, err = debit(currency.Silver, "silver")
	case 2:
		plan.CurrencyAfter.SilverCard, err = debit(currency.SilverCard, "silver_card")
	default:
		return Plan{}, fmt.Errorf("shopbuyplan: unsupported capital type %d", input.CapitalType)
	}
	if err != nil {
		return Plan{}, err
	}

	if input.RequestedSlot > 0 {
		plan.RewardSlot = input.RequestedSlot
		return plan, nil
	}
	used := make(map[int32]struct{})
	for _, entry := range occupied {
		if entry.View == input.RewardView && entry.Slot > 0 {
			used[entry.Slot] = struct{}{}
		}
	}
	for slot := int32(1); ; slot++ {
		if _, exists := used[slot]; !exists {
			plan.RewardSlot = slot
			return plan, nil
		}
	}
}
