package main

// normalShopSafeTotal only validates that the authenticated listing's int32
// price and client purchase count can be represented by the existing actor's
// int32 currency mutation methods. It is not transaction authorization.
func normalShopSafeTotal(price, quantity int32) (int64, bool) {
	if price < 0 || quantity < 1 || quantity > 99 {
		return 0, false
	}
	total := int64(price) * int64(quantity)
	return total, total <= int64(1<<31-1)
}
