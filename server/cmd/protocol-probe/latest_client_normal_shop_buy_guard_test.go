package main

import "testing"

func TestNormalShopSafeTotal(t *testing.T) {
	cases := []struct {
		name     string
		price    int32
		quantity int32
		want     int64
		ok       bool
	}{
		{"regular", 100, 3, 300, true},
		{"authored_free_price_not_invented", 0, 1, 0, true},
		{"max_count", 1, 99, 99, true},
		{"zero_quantity", 100, 0, 0, false},
		{"over_max_quantity", 100, 100, 0, false},
		{"negative_price", -1, 1, 0, false},
		{"last_valid_30m", 30000000, 71, 2130000000, true},
		{"overflow_30m", 30000000, 72, 2160000000, false},
		{"large_single", 1600000000, 1, 1600000000, true},
		{"large_pair", 1600000000, 2, 3200000000, false},
		{"int32_limit", 2147483647, 1, 2147483647, true},
		{"int32_overflow", 2147483647, 2, 4294967294, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := normalShopSafeTotal(tc.price, tc.quantity)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("normalShopSafeTotal(%d,%d)=(%d,%t); want (%d,%t)", tc.price, tc.quantity, got, ok, tc.want, tc.ok)
			}
		})
	}
}
