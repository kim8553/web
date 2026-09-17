package main

import "testing"

func TestSummarizeCurrentShopExchangeConditionResults(t *testing.T) {
	cases := []struct {
		name string
		roots []bool
		supported bool
		want currentShopExchangeConditionObservation
	}{
		{"all-true", []bool{true, true}, true, currentShopExchangeConditionObservation{2, true, 0}},
		{"unmet", []bool{true, false}, true, currentShopExchangeConditionObservation{2, true, 1}},
		{"unresolved", []bool{false, true}, false, currentShopExchangeConditionObservation{2, false, 1}},
		{"all-unresolved", []bool{false, false}, false, currentShopExchangeConditionObservation{2, false, 2}},
		{"empty-is-not-authorization", nil, true, currentShopExchangeConditionObservation{0, true, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]bool(nil), tc.roots...)
			for i := 0; i < 2; i++ {
				got := summarizeCurrentShopExchangeConditionResults(tc.roots, tc.supported)
				if got != tc.want {
					t.Fatalf("attempt=%d observation=%+v want=%+v", i, got, tc.want)
				}
			}
			if len(tc.roots) != len(before) {
				t.Fatal("observation modified roots")
			}
			for i := range before {
				if before[i] != tc.roots[i] {
					t.Fatalf("observation modified root %d", i)
				}
			}
		})
	}
}
