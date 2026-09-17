package main

import (
	"fmt"
	"math"
)

// checkedOrdinaryShopTotal ensures the existing int32 wallet-debit path cannot
// narrow an int64 total to a different (possibly negative) amount.
func checkedOrdinaryShopTotal(unitPrice, quantity int32) (int64, error) {
	if unitPrice < 0 || quantity < 1 || quantity > 99 {
		return 0, fmt.Errorf("invalid shop price=%d quantity=%d", unitPrice, quantity)
	}
	total := int64(unitPrice) * int64(quantity)
	if total > math.MaxInt32 {
		return 0, fmt.Errorf("shop purchase total %d exceeds int32 wallet debit", total)
	}
	return total, nil
}
