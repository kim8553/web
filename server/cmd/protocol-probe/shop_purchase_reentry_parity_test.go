package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// These cases exercise only ordinary NPC purchases, never exchange or GM paths.
// The fixtures exercise the already implemented bag categories, not a claim
// that these particular fictional catalog IDs exist in the current client.
func TestOrdinaryShopPurchaseFrameMatchesReconnectReplay(t *testing.T) {
	cases := []struct {
		name string
		item bagItem
	}{
		{"tools", bagItem{ConfigID: "shop_parity_tool", ItemType: 1, ViewID: 1, Amount: 3, MaxAmount: 99, Slot: 4}},
		{"equipment", bagItem{ConfigID: "shop_parity_equip", ItemType: 101, ViewID: 2, Amount: 1, MaxAmount: 1, Slot: 5, EquipType: "Weapon"}},
		{"third_category", bagItem{ConfigID: "shop_parity_third", ItemType: 100, ViewID: 3, Amount: 2, MaxAmount: 99, Slot: 6}},
		{"fourth_category", bagItem{ConfigID: "shop_parity_fourth", ItemType: 100, ViewID: 4, Amount: 2, MaxAmount: 99, Slot: 7}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := bagViewForViewID(tc.item.ViewID)
			if view == 0 {
				t.Fatalf("unmapped bag category %d", tc.item.ViewID)
			}
			props := bagItemProps(view, tc.item)
			immediate, err := serverViewAdd(view, uint16(tc.item.Slot), props)
			if err != nil {
				t.Fatalf("purchase frame: %v", err)
			}
			replay, err := latestClientCurrentBagFrames(view, uint16(tc.item.Slot), tc.item.ViewID, props)
			if err != nil {
				t.Fatalf("reconnect frame: %v", err)
			}
			if len(replay) != 1 || !bytes.Equal(immediate, replay[0]) {
				t.Fatalf("purchase/reconnect frame mismatch: immediate=%x replay=%x", immediate, replay)
			}
			if len(immediate) < 7 || immediate[0] != 0x18 || binary.LittleEndian.Uint16(immediate[1:3]) != view || binary.LittleEndian.Uint16(immediate[3:5]) != uint16(tc.item.Slot) {
				t.Fatalf("unexpected purchase frame header: %x", immediate)
			}
		})
	}
}

func TestOrdinaryShopWalletFrameMatchesActorReload(t *testing.T) {
	wallet := currencySnapshot{Silver: 70, Gold: 10, SilverCard: 20, SilverTicket: 30}
	immediate, err := ordinaryShopCurrencyFrame(wallet)
	if err != nil {
		t.Fatalf("purchase wallet frame: %v", err)
	}
	player := &playerActor{silver: wallet.Silver, gold: wallet.Gold, silverCard: wallet.SilverCard, silverTicket: wallet.SilverTicket}
	replay, err := player.currenciesUpdate()
	if err != nil {
		t.Fatalf("reloaded actor wallet frame: %v", err)
	}
	if !bytes.Equal(immediate, replay) {
		t.Fatalf("purchase/reload wallet frame mismatch: immediate=%x replay=%x", immediate, replay)
	}
}
