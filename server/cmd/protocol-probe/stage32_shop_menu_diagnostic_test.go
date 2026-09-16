package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStage32InspectShopMenuSeparatesOrdinaryAndExchange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	content := "[Shop_test]\nType=0\nPageInfo=Page1\n0=regular,1,1,40,0,0,0,0\n0=exchange,1,3,0,0,0,1,1234\n0=unset,1,3,0,0,0,2,0\n0=unknown,1,9,0,0,0,3,0\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got := stage32InspectShopMenu(path, "Shop_test")
	if got.CatalogError != "" || got.AuthoredRows != 4 || got.OrdinaryRows != 1 || got.ExchangeRows != 2 || got.ExchangeData != 1 || got.OtherRows != 1 {
		t.Fatalf("snapshot=%+v", got)
	}
}

func TestStage32InspectShopMenuReportsExchangeOnlyWithoutClaimingVisible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	if err := os.WriteFile(path, []byte("[Shop_exchange]\n0=exchange,1,3,0,0,0,0,678\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := stage32InspectShopMenu(path, "Shop_exchange")
	if got.CatalogError != "" || got.OrdinaryRows != 0 || got.ExchangeRows != 1 || got.ExchangeData != 1 {
		t.Fatalf("snapshot=%+v", got)
	}
}

func TestStage32InspectShopMenuDoesNotGuessSuffixOrInventSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	if err := os.WriteFile(path, []byte("[Shop_guild_1]\n0=thing,1,1,20,0,0,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := stage32InspectShopMenu(path, "Shop_guild")
	if !strings.Contains(got.CatalogError, "no section") || got.AuthoredRows != 0 {
		t.Fatalf("missing exact shop=%+v", got)
	}
}

func TestStage32InspectShopMenuDoesNotClaimMalformedOrEmptyShopReady(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	if err := os.WriteFile(path, []byte("[Empty]\nType=0\n[Broken]\n0=bad,wrong,1,20,0,0,0,0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, shop := range []string{"Empty", "Broken"} {
		got := stage32InspectShopMenu(path, shop)
		if got.CatalogError == "" || got.AuthoredRows != 0 {
			t.Fatalf("%s=%+v", shop, got)
		}
	}
}

// This opt-in regression is run against the user's exact, SHA-checked RAR
// shop.ini during an offline audit; CI without that private file skips it.
func TestStage32RARShopReferenceIsNotSilentlyRewritten(t *testing.T) {
	path := os.Getenv("NINEYIN_STAGE32_RAR_SHOP_INI")
	if path == "" {
		t.Skip("exact RAR shop.ini not supplied")
	}
	base := stage32InspectShopMenu(path, "Shop_GB_Yishiting")
	if !strings.Contains(base.CatalogError, "no section") || base.AuthoredRows != 0 {
		t.Fatalf("base reference unexpectedly exists: %+v", base)
	}
	suffixed := stage32InspectShopMenu(path, "Shop_GB_Yishiting_1")
	if suffixed.CatalogError != "" || suffixed.AuthoredRows != 3 || suffixed.OrdinaryRows != 3 {
		t.Fatalf("authored suffix differs: %+v", suffixed)
	}
	ordinary := stage32InspectShopMenu(path, "Shop_yaopin_00100")
	if ordinary.CatalogError != "" || ordinary.OrdinaryRows == 0 {
		t.Fatalf("known ordinary shop differs: %+v", ordinary)
	}
}
