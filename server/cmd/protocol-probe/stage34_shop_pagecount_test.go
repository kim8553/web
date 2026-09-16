package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStage34OnlyAuthoredPageExtendsUnderdeclaredPageInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	content := "[Shop_underdeclared]\nPageInfo=Page1\nType=0\n0=first,1,1,50,0,1,0,0,0\n1=second,1,1,50,0,1,0,0,0\n[Shop_empty]\nPageInfo=Page1\nType=0\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	items, kind, pages, err := loadShopCatalogSection(path, "Shop_underdeclared")
	if err != nil || kind != 0 || pages != 2 || len(items) != 2 || items[0].page != 0 || items[1].page != 1 || items[1].position != 0 {
		t.Fatalf("underdeclared kind=%d pages=%d items=%+v err=%v", kind, pages, items, err)
	}
	if _, _, _, err := loadShopCatalogSection(path, "Shop_empty"); err == nil {
		t.Fatal("empty shop must remain unavailable")
	}
}

func TestStage34PreservesValidPageInfoIncludingNamedEmptyPages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	content := "[Shop_valid]\nPageInfo=Page1,Page2,Page3\nType=1\n0=first,1,2,9,0,1,0,0,0\n1=second,1,1,8,0,1,1,0,0\n[Shop_fallback]\nType=0\n0=first,1,1,50,0,1,0,0,0\n2=third,1,1,50,0,1,0,0,0\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if rows, kind, pages, err := loadShopCatalogSection(path, "Shop_valid"); err != nil || kind != 1 || pages != 3 || len(rows) != 2 {
		t.Fatalf("valid rows=%d kind=%d pages=%d err=%v", len(rows), kind, pages, err)
	}
	if rows, _, pages, err := loadShopCatalogSection(path, "Shop_fallback"); err != nil || pages != 3 || len(rows) != 2 {
		t.Fatalf("fallback rows=%d pages=%d err=%v", len(rows), pages, err)
	}
}

func TestStage34ExactRARUnderdeclaredShopOnly(t *testing.T) {
	path := os.Getenv("NINEYIN_STAGE34_RAR_SHOP_INI")
	if path == "" {
		t.Skip("exact RAR shop.ini not supplied")
	}
	rows, kind, pages, err := loadShopCatalogSection(path, "Shop_special_001")
	if err != nil || kind != 0 || pages != 2 || len(rows) != 2 || rows[0].configID != "taskitem_01032" || rows[1].configID != "taskitem_01060" || rows[1].page != 1 {
		t.Fatalf("RAR pages=%d rows=%+v kind=%d err=%v", pages, rows, kind, err)
	}
	rows, kind, pages, err = loadShopCatalogSection(path, "Shop_yaopin_00100")
	if err != nil || pages < 1 || kind < 0 || len(rows) == 0 {
		t.Fatalf("ordinary shop pages=%d rows=%d kind=%d err=%v", pages, len(rows), kind, err)
	}
}
