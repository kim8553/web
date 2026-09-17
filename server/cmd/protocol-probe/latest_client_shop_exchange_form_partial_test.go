package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAuthorizedNoPreviewExchangeFormFrameSendsBindStatusZero(t *testing.T) {
	dir := t.TempDir()
	shopPath := filepath.Join(dir, "shop.ini")
	exchangePath := filepath.Join(dir, "exchangeitem.ini")
	if err := os.WriteFile(shopPath, []byte("[shop_mode3]\nType=7\nPageInfo=all\n0=item_exchange,2,3,0,0,0,4,70123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exchangePath, []byte("[70123]\nType=2\nItem=item_cost,2\nCondition=10\nCondition2=20\nFilters=20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := shopExchangeFormRequest{ViewIdent: 61, BindIndex: 5, ShopID: "shop_mode3", Page: 0, Position: 5}
	authority, resolved, err := resolveCurrentShopExchangeFormAuthority(shopPath, exchangePath, request)
	if err != nil || !resolved {
		t.Fatalf("resolve authority: resolved=%t err=%v", resolved, err)
	}
	frame, ok, err := buildAuthorizedNoPreviewExchangeFormFrame(authority, request)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("BindStatus=0 exchange form was not sendable")
	}
	if len(frame) < 3 || frame[0] != 0x1e {
		t.Fatalf("frame prefix=%x", frame)
	}
	// Exact 11-field config includes inert zero placeholders for the runtime
	// preview fields only because BindStatus=0 bypasses that presentation branch.
	if !strings.Contains(string(frame), "2||0|item_cost,2|0|0|0|10|20|20|") {
		t.Fatalf("frame does not contain expected no-preview config: %x", frame)
	}
}

func TestBuildAuthorizedNoPreviewExchangeFormFrameBlocksBindStatusPositive(t *testing.T) {
	dir := t.TempDir()
	shopPath := filepath.Join(dir, "shop.ini")
	exchangePath := filepath.Join(dir, "exchangeitem.ini")
	if err := os.WriteFile(shopPath, []byte("[shop_mode3]\nType=7\nPageInfo=all\n0=item_exchange,2,3,0,0,0,4,70123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exchangePath, []byte("[70123]\nType=2\nBindStatus=1\nItem=item_cost,2\nCondition=10\nCondition2=20\nFilters=20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := shopExchangeFormRequest{ViewIdent: 61, BindIndex: 5, ShopID: "shop_mode3", Page: 0, Position: 5}
	authority, resolved, err := resolveCurrentShopExchangeFormAuthority(shopPath, exchangePath, request)
	if err != nil || !resolved {
		t.Fatalf("resolve authority: resolved=%t err=%v", resolved, err)
	}
	frame, ok, err := buildAuthorizedNoPreviewExchangeFormFrame(authority, request)
	if err != nil {
		t.Fatal(err)
	}
	if ok || len(frame) != 0 {
		t.Fatalf("BindStatus=1 must remain fail-closed: ok=%t frame=%x", ok, frame)
	}
}
