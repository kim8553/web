package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShopCatalogRejectsAuthoredPageOutsidePageInfo(t *testing.T) {
	// This exact Shop_special_001 section occurs in both verified share.package
	// candidate streams. Do not infer which stream the client selects.
	const source = `[Shop_special_001]
NameID=
PageInfo=Page1
Type=0
0=taskitem_01032,1,1,50,0,1,0,0,0
1=taskitem_01060,1,1,50,0,1,0,0,0
`
	path := filepath.Join(t.TempDir(), "shop.ini")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	items, _, count, err := loadShopCatalogSection(path, "Shop_special_001")
	if err == nil || !strings.Contains(err.Error(), "Shop_special_001") || !strings.Contains(err.Error(), "PageInfo") {
		t.Fatalf("out-of-range page accepted: items=%#v count=%d err=%v", items, count, err)
	}
}

func TestShopCatalogPageInfoBoundariesAndMissingPageInfo(t *testing.T) {
	tests := []struct {
		name, pageInfo string
		wantPages      int32
	}{
		{"one declared page", "PageInfo=Page1\n", 1},
		{"two declared pages", "PageInfo=Page1,Page2\n", 2},
		{"no PageInfo uses existing fallback", "", 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := "[Shop_valid]\n" + tc.pageInfo + "Type=0\n0=item_first,1,1,50,0,1,0,0,0\n"
			if tc.wantPages == 2 {
				content += "1=item_second,1,1,50,0,1,1,0,0\n"
			}
			path := filepath.Join(t.TempDir(), "shop.ini")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			items, _, pages, err := loadShopCatalogSection(path, "Shop_valid")
			if err != nil || pages != tc.wantPages || len(items) != int(tc.wantPages) {
				t.Fatalf("valid pages: items=%#v count=%d err=%v, want pages %d", items, pages, err, tc.wantPages)
			}
		})
	}
}
