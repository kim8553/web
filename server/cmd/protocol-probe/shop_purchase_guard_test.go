package main

import (
	"math"
	"testing"
)

func TestCheckedOrdinaryShopTotal(t *testing.T) {
	cases := []struct {
		name          string
		price, count  int32
		want          int64
		invalid       bool
	}{
		{name: "one", price: 42, count: 1, want: 42},
		{name: "multiple", price: 200, count: 99, want: 19800},
		{name: "zero price", price: 0, count: 99, want: 0},
		{name: "max int32", price: math.MaxInt32, count: 1, want: math.MaxInt32},
		{name: "overflow", price: math.MaxInt32, count: 2, invalid: true},
		{name: "negative", price: -1, count: 1, invalid: true},
		{name: "zero count", price: 1, count: 0, invalid: true},
		{name: "over limit", price: 1, count: 100, invalid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkedOrdinaryShopTotal(tc.price, tc.count)
			if (err != nil) != tc.invalid || (!tc.invalid && got != tc.want) {
				t.Fatalf("got total=%d err=%v; want total=%d invalid=%t", got, err, tc.want, tc.invalid)
			}
		})
	}
}
