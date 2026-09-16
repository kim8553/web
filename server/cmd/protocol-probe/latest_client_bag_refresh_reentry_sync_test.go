package main

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestLatestClientGMPanelStillExposesGiveItem(t *testing.T) {
	for _, token := range []string{"物品发放", "Game_item_hp_001", "action:'give_item'", "config_id:config", "container:'bag'"} {
		if !strings.Contains(gmPanelHTML, token) {
			t.Fatalf("GM panel missing %q", token)
		}
	}
}

func TestServerDeleteViewCompatVerifiedLayout(t *testing.T) {
	frame := serverDeleteViewCompat(2)
	if len(frame) != 3 || frame[0] != 0x16 || binary.LittleEndian.Uint16(frame[1:]) != 2 {
		t.Fatalf("delete view frame=%x", frame)
	}
}
