package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestLatestClientShopViewMetadataUsesCurrentOrdinals(t *testing.T) {
	props := latestClientShopViewProperties("Shop_test", 7, 3)
	if len(props) != 4 {
		t.Fatalf("shop metadata property count=%d want=4", len(props))
	}
	wantIndex := []uint16{101, 102, 103, 104}
	for i, want := range wantIndex {
		if props[i].index != want {
			t.Fatalf("shop metadata[%d] ordinal=%d want=%d", i, props[i].index, want)
		}
	}
	if props[2].int32 == nil || *props[2].int32 != 3 {
		t.Fatalf("PageCount=%v want=3", props[2].int32)
	}
	if props[3].int32 == nil || *props[3].int32 != 7 {
		t.Fatalf("ShopType=%v want=7", props[3].int32)
	}
}

func TestLatestClientShopItemDoesNotEmitStringAtOrdinal181(t *testing.T) {
	props := latestClientShopItemProperties(shopCatalogItem{configID: "item_test", amount: 2, priceMode: 1, price: 55})
	wantIndex := []uint16{7, 105, 106, 107, 108, 109, 110}
	if len(props) != len(wantIndex) {
		t.Fatalf("shop item property count=%d want=%d", len(props), len(wantIndex))
	}
	for i, want := range wantIndex {
		if props[i].index != want {
			t.Fatalf("shop item[%d] ordinal=%d want=%d", i, props[i].index, want)
		}
		if props[i].index == 181 {
			t.Fatal("shop item emitted invalid ConfigID/string at current MaxPowerValue ordinal 181")
		}
		if int(props[i].index) >= latestClientPlayerWirePropertyTableCount {
			t.Fatalf("shop item ordinal=%d outside current negotiated count=%d", props[i].index, latestClientPlayerWirePropertyTableCount)
		}
	}
	if props[0].text == nil || *props[0].text != "item_test" || props[1].text == nil || *props[1].text != "item_test" {
		t.Fatalf("ConfigID duplicates malformed: %#v", props[:2])
	}
	if props[4].int32 == nil || *props[4].int32 != 55 {
		t.Fatalf("SellPrice1=%v want=55", props[4].int32)
	}
}

func TestLatestClientShopExchangeDataUsesCurrentNegotiatedOrdinal(t *testing.T) {
	props := latestClientShopItemProperties(shopCatalogItem{configID: "exchange_item", amount: 1, priceMode: 3, exchangeData: 13097})
	if len(props) != 8 {
		t.Fatalf("exchange shop prop count=%d, want 8", len(props))
	}
	last := props[len(props)-1]
	if last.index != 233 || last.int32 == nil || *last.int32 != 13097 {
		t.Fatalf("ExchangeData property=%#v", last)
	}
}

func TestShopCatalogUsesOnlyAuthoredPagesAndParsesExchangeData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	const data = `[Shop_test]
PageInfo=Page1,Page2
Type=0
0=item_silver,1,1,8,0,1,0,0,0
1=item_exchange,1,3,0,0,1,3,13097,0
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	items, shopType, pageCount, err := shopCatalogItems(path, "Shop_test")
	if err != nil {
		t.Fatal(err)
	}
	if shopType != 0 || pageCount != 2 {
		t.Fatalf("shop metadata type=%d pages=%d, want 0/2 authored only", shopType, pageCount)
	}
	if len(items) != 2 {
		t.Fatalf("shop item count=%d, want exactly 2 authored rows", len(items))
	}
	if items[0].priceMode != 1 || items[0].price != 8 || items[0].exchangeData != 0 {
		t.Fatalf("ordinary row=%#v", items[0])
	}
	if items[1].priceMode != 3 || items[1].price != 0 || items[1].exchangeData != 13097 {
		t.Fatalf("exchange row=%#v", items[1])
	}
}

func TestCurrentShopAuthoredCoordinatesUseCurrentClientBases(t *testing.T) {
	tests := []struct {
		wirePage, wirePos int32
		wantPage, wantPos int32
		ok                bool
	}{
		{wirePage: 0, wirePos: 1, wantPage: 0, wantPos: 0, ok: true},
		{wirePage: 1, wirePos: 1, wantPage: 1, wantPos: 0, ok: true},
		{wirePage: 2, wirePos: 10, wantPage: 2, wantPos: 9, ok: true},
		{wirePage: -1, wirePos: 1, ok: false},
		{wirePage: 0, wirePos: 0, ok: false},
	}
	for _, tc := range tests {
		page, pos, ok := currentShopAuthoredCoordinates(tc.wirePage, tc.wirePos)
		if ok != tc.ok || page != tc.wantPage || pos != tc.wantPos {
			t.Fatalf("wire page=%d pos=%d -> page=%d pos=%d ok=%v; want page=%d pos=%d ok=%v", tc.wirePage, tc.wirePos, page, pos, ok, tc.wantPage, tc.wantPos, tc.ok)
		}
	}
}

func TestCurrentShopListingDoesNotAcceptLegacyPageBase(t *testing.T) {
	items := []shopCatalogItem{
		{configID: "first", page: 0, position: 0},
		{configID: "second", page: 1, position: 0},
	}
	if got := currentShopListing(items, 0, 1); got == nil || got.configID != "first" {
		t.Fatalf("current first-page wire coordinate resolved %#v, want first", got)
	}
	if got := currentShopListing(items, 1, 1); got == nil || got.configID != "second" {
		t.Fatalf("current second-page wire coordinate resolved %#v, want second", got)
	}
}

func TestLatestClientShopPageNumUsesCurrentShopConfig(t *testing.T) {
	props := latestClientShopViewProperties("Shop_test", 1, 3)
	if props[1].int32 == nil || *props[1].int32 != 500 {
		t.Fatalf("PageNum=%v want=500", props[1].int32)
	}
}

func TestCurrentShopViewObjectIndexMatchesClientPagingFormula(t *testing.T) {
	tests := []struct {
		item shopCatalogItem
		want uint16
	}{
		{item: shopCatalogItem{page: 0, position: 0}, want: 1},
		{item: shopCatalogItem{page: 0, position: 35}, want: 36},
		{item: shopCatalogItem{page: 1, position: 0}, want: 501},
		{item: shopCatalogItem{page: 4, position: 35}, want: 2036},
	}
	for _, tc := range tests {
		got, ok := currentShopViewObjectIndex(tc.item)
		if !ok || got != tc.want {
			t.Fatalf("page=%d pos=%d -> ident=%d ok=%v want=%d", tc.item.page, tc.item.position, got, ok, tc.want)
		}
		ident := int32(got)
		page := (ident - 1) / currentShopPageSize
		grid := ident%currentShopPageSize - 1
		if page != tc.item.page || grid != tc.item.position {
			t.Fatalf("ident=%d decodes page=%d grid=%d want page=%d grid=%d", ident, page, grid, tc.item.page, tc.item.position)
		}
	}
}

func TestShopCatalogPageInfoDrivesPageCountAndNumericKeyDrivesPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	const data = `[Shop_pages]
PageInfo=Page1,Page2,Page3
Type=1
0=first,1,1,8,0,1,0,0,0
1=second,1,1,9,0,1,0,0,0
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	items, _, pageCount, err := shopCatalogItems(path, "Shop_pages")
	if err != nil {
		t.Fatal(err)
	}
	if pageCount != 3 {
		t.Fatalf("PageCount=%d want=3 from PageInfo", pageCount)
	}
	if len(items) != 2 || items[0].page != 0 || items[1].page != 1 {
		t.Fatalf("numeric page keys not retained: %#v", items)
	}
}

func TestCurrentShopItemVisibleInViewKeepsOrdinaryAndPositiveExchangeData(t *testing.T) {
	tests := []struct {
		name string
		item shopCatalogItem
		want bool
	}{
		{name: "mode0", item: shopCatalogItem{priceMode: 0}, want: true},
		{name: "mode1", item: shopCatalogItem{priceMode: 1}, want: true},
		{name: "mode2", item: shopCatalogItem{priceMode: 2}, want: true},
		{name: "mode3-positive-exchange", item: shopCatalogItem{priceMode: 3, exchangeData: 13097}, want: true},
		{name: "mode3-zero-exchange", item: shopCatalogItem{priceMode: 3, exchangeData: 0}, want: false},
		{name: "mode3-negative-exchange", item: shopCatalogItem{priceMode: 3, exchangeData: -1}, want: false},
		{name: "unknown-mode", item: shopCatalogItem{priceMode: 4, exchangeData: 13097}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := currentShopItemVisibleInView(tc.item); got != tc.want {
				t.Fatalf("visible=%v want=%v item=%#v", got, tc.want, tc.item)
			}
		})
	}
}

func TestOpenShopViewIncludesPositiveMode3AndHidesInvalidMode3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shop.ini")
	const data = `[Shop_view_exchange]
PageInfo=Page1
Type=0
0=item_mode0,1,0,10,0,1,0,0,0
0=item_mode1,1,1,11,0,1,1,0,0
0=item_mode2,1,2,12,0,1,2,0,0
0=item_exchange,1,3,0,0,1,3,13097,0
0=item_exchange_invalid,1,3,0,0,1,4,0,0
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := defaultShopINIPath
	defaultShopINIPath = path
	t.Cleanup(func() { defaultShopINIPath = previous })

	conn := &captureMessageConnection{}
	lifecycle := &sceneLifecycle{conn: conn}
	if err := lifecycle.openShopLocked("Shop_view_exchange"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("frame count=%d want=5 (create + 3 ordinary + 1 valid exchange)", len(frames))
	}
	if len(frames[0]) < 7 || frames[0][0] != 0x15 || binary.LittleEndian.Uint16(frames[0][1:3]) != 61 {
		t.Fatalf("shop create frame=%x", frames[0])
	}
	wantObjectIndex := []uint16{1, 2, 3, 4}
	for i, want := range wantObjectIndex {
		frame := frames[i+1]
		if len(frame) < 7 || frame[0] != 0x18 {
			t.Fatalf("item frame[%d]=%x", i, frame)
		}
		if got := binary.LittleEndian.Uint16(frame[1:3]); got != 61 {
			t.Fatalf("item frame[%d] view=%d want=61", i, got)
		}
		if got := binary.LittleEndian.Uint16(frame[3:5]); got != want {
			t.Fatalf("item frame[%d] object index=%d want=%d", i, got, want)
		}
	}

	exchange := frames[4]
	needle := make([]byte, 6)
	binary.LittleEndian.PutUint16(needle[0:2], 233)
	binary.LittleEndian.PutUint32(needle[2:6], 13097)
	if !bytes.Contains(exchange, needle) {
		t.Fatalf("mode3 frame lacks ExchangeData ordinal/value 233/13097: %x", exchange)
	}
}
