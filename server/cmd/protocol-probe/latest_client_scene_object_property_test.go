package main

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/entity"
)

func TestLatestClientNPCCombatPropertiesUseNegotiatedOrdinals(t *testing.T) {
	raw := npcCombatProperties(entity.ActorState{
		HP: 750, BaseMaxHP: 1000, MaxHP: 1000, HitHP: 900, LogicState: entity.LogicStateFighting,
	})
	got := latestClientSceneObjectWireProperties(raw)
	want := []uint16{21, 28, 30, 32, 112, 209}
	if len(got) != len(want) {
		t.Fatalf("normalized NPC combat property count=%d, want %d: %#v", len(got), len(want), got)
	}
	for i, ordinal := range want {
		if got[i].Index != ordinal {
			t.Fatalf("normalized NPC combat property[%d]=%d, want %d", i, got[i].Index, ordinal)
		}
		if got[i].Index >= uint16(latestClientPlayerWirePropertyTableCount) {
			t.Fatalf("normalized NPC combat property[%d]=%d outside negotiated table count=%d", i, got[i].Index, latestClientPlayerWirePropertyTableCount)
		}
	}

	frame, err := sceneObjectProperties(0x1234, 1, 0, raw)
	if err != nil {
		t.Fatal(err)
	}
	if count := binary.LittleEndian.Uint16(frame[10:12]); count != uint16(len(want)) {
		t.Fatalf("ServerObjectProperty count=%d, want %d after legacy duplicate removal", count, len(want))
	}
}
