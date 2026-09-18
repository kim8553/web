package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectsAmbiguousOriginalShopPagePosition(t *testing.T) {
	file := filepath.Join(t.TempDir(), "shop.ini")
	// These are two distinct authored entries at the same page/position in the
	// first verified share.package shop stream, in its original column order.
	data := "[Shop_school_zhenghe_fc_sl035_zs_01]\nPageInfo=Page1,Page2\nType=1\n" +
		"1=book_CS_slzp_lzs01_cs06,1,3,0,50,1,12,9254,0,\n" +
		"1=book_zhenfa_sl_01,1,3,0,33,1,12,2676,0\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := loadShopCatalogSection(file, "Shop_school_zhenghe_fc_sl035_zs_01")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("ambiguous page-position must be rejected; got %v", err)
	}
}

func TestSameGridPositionOnDifferentAuthoredPagesIsAllowed(t *testing.T) {
	file := filepath.Join(t.TempDir(), "shop.ini")
	data := "[Shop_test]\nPageInfo=Page1,Page2\nType=1\n0=first,1,1,8,0,1,12,0,0\n1=second,1,1,9,0,1,12,0,0\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	items, _, pages, err := loadShopCatalogSection(file, "Shop_test")
	if err != nil || len(items) != 2 || pages != 2 {
		t.Fatalf("valid separate pages: items=%v pages=%d err=%v", items, pages, err)
	}
}
