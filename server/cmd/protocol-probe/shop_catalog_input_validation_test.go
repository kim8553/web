package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShopCatalogRejectsMalformedPurchaseRows(t *testing.T) {
	tests := []struct {
		name string
		row  string
	}{
		{name: "missing item identity", row: "0=,1,1,8,0,1,0,0,0"},
		{name: "zero authored amount", row: "0=item_test,0,1,8,0,1,0,0,0"},
		{name: "negative authored amount", row: "0=item_test,-1,1,8,0,1,0,0,0"},
		{name: "negative price", row: "0=item_test,1,1,-8,0,1,0,0,0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "shop.ini")
			data := "[Shop_test]\nPageInfo=Page1\nType=0\n" + tc.row + "\n"
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			_, _, _, err := loadShopCatalogSection(path, "Shop_test")
			if err == nil || !strings.Contains(err.Error(), "invalid item") {
				t.Fatalf("malformed row %q: error=%v; want invalid item", tc.row, err)
			}
		})
	}
}

func TestShopCatalogAcceptsZeroPriceOrdinaryItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	const data = "[Shop_test]\nPageInfo=Page1\nType=0\n0=item_test,1,1,0,0,1,0,0,0\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	items, _, _, err := loadShopCatalogSection(path, "Shop_test")
	if err != nil || len(items) != 1 || items[0].price != 0 {
		t.Fatalf("valid zero-price row: items=%#v error=%v", items, err)
	}
}
