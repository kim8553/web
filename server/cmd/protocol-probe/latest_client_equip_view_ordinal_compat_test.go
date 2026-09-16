package main

import (
	"encoding/binary"
	"testing"
)

func latestClientViewAddPropertyIndexesForTest(t *testing.T, frame []byte) []uint16 {
	t.Helper()
	if len(frame) < 7 || frame[0] != 0x18 {
		t.Fatalf("not VIEW_ADD: %x", frame)
	}
	count := int(binary.LittleEndian.Uint16(frame[5:7]))
	off := 7
	indexes := make([]uint16, 0, count)
	for i := 0; i < count; i++ {
		if off+2 > len(frame) {
			t.Fatalf("property %d truncated before index", i)
		}
		idx := binary.LittleEndian.Uint16(frame[off : off+2])
		off += 2
		indexes = append(indexes, idx)
		if int(idx) >= len(latestClientViewPropertyFields) {
			t.Fatalf("property ordinal %d outside negotiated table", idx)
		}
		switch latestClientViewPropertyFields[idx].Type {
		case 1: // WireByte
			off++
		case 3: // WireInt32
			off += 4
		case 7: // WireString
			if off+4 > len(frame) {
				t.Fatalf("property %d string length truncated", idx)
			}
			n := int(binary.LittleEndian.Uint32(frame[off : off+4]))
			off += 4 + n
		default:
			t.Fatalf("unexpected type %s in normalized equip row at ordinal %d", latestClientViewPropertyFields[idx].Type, idx)
		}
		if off > len(frame) {
			t.Fatalf("property %d value truncated", idx)
		}
	}
	if off != len(frame) {
		t.Fatalf("VIEW_ADD trailing bytes: parsed=%d size=%d", off, len(frame))
	}
	return indexes
}

func TestLatestClientEquipViewNormalizesHistoricalBodyProperties(t *testing.T) {
	item := bagItem{
		ConfigID:     "equip_test",
		ItemType:     101,
		BindStatus:   1,
		MaxAmount:    1,
		ColorLevel:   4,
		LogicPack:    7,
		ArtPack:      123,
		Hardiness:    900,
		MaxHardiness: 1000,
		EquipType:    "Weapon",
	}
	frame, err := serverViewAdd(1, 22, equipItemProps(item))
	if err != nil {
		t.Fatalf("current equip VIEW_ADD rejected after normalization: %v", err)
	}
	got := latestClientViewAddPropertyIndexesForTest(t, frame)
	want := []uint16{7, 105, 106, 110, 205, 228, 234}
	if len(got) != len(want) {
		t.Fatalf("equip wire indexes=%v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("equip wire indexes=%v, want %v", got, want)
		}
	}
}

func TestLatestClientEquipViewNormalizesWeaponNestedConfigID(t *testing.T) {
	item := bagItem{ConfigID: "weapon_test", ItemType: 101, BindStatus: 1, MaxAmount: 1, EquipType: "Weapon"}
	frame, err := serverViewAdd(1, 22, weaponRowProps(item))
	if err != nil {
		t.Fatalf("current weapon VIEW_ADD rejected after normalization: %v", err)
	}
	got := latestClientViewAddPropertyIndexesForTest(t, frame)
	want := []uint16{7, 105, 106, 110, 205, 228, 234}
	if len(got) != len(want) {
		t.Fatalf("weapon wire indexes=%v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("weapon wire indexes=%v, want %v", got, want)
		}
	}
}
