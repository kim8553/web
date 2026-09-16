package main

import (
	"encoding/binary"
	"github.com/local/9yin-go-server/internal/clientdata"
	"testing"
)

func TestLatestClientCurrentBagContainers(t *testing.T) {
	cases := map[int32]uint16{1: 2, 2: 121, 3: 123, 4: 125}
	for category, want := range cases {
		got, ok := latestClientCurrentBagContainer(category)
		if !ok || got != want {
			t.Fatalf("category %d -> (%d,%v), want (%d,true)", category, got, ok, want)
		}
	}
	if _, ok := latestClientCurrentBagContainer(0); ok {
		t.Fatal("unknown bag category must not be guessed")
	}
}

func TestLatestClientCurrentBagRichObjectOrdinalsAreAppendOnly(t *testing.T) {
	if latestClientPlayerWirePropertyTableCount != 235 {
		t.Fatalf("property table=%d, want 235", latestClientPlayerWirePropertyTableCount)
	}
	if got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "ItemType", typ: clientdata.WireInt32}]; got != 205 {
		t.Fatalf("ItemType/int32 ordinal=%d, want 205", got)
	}
	if got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "ViewID", typ: clientdata.WireInt32}]; got != 228 {
		t.Fatalf("ViewID/int32 ordinal=%d, want 228", got)
	}
	if _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "Ident", typ: clientdata.WireString}]; ok {
		t.Fatal("Ident must be built-in/read-only and not negotiated")
	}
	if got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "MaxHP", typ: clientdata.WireInt32}]; got != 30 {
		t.Fatalf("MaxHP ordinal moved to %d", got)
	}
	if got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "Force", typ: clientdata.WireString}]; got != 226 {
		t.Fatalf("Force ordinal moved to %d", got)
	}
}

func TestLatestClientViewAddSingleExactLayout(t *testing.T) {
	frame, err := latestClientViewAddSingle(2, 1, viewInt(228, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 13 || frame[0] != 0x18 || binary.LittleEndian.Uint16(frame[1:3]) != 2 || binary.LittleEndian.Uint16(frame[3:5]) != 1 || binary.LittleEndian.Uint16(frame[5:7]) != 1 || binary.LittleEndian.Uint16(frame[7:9]) != 228 || binary.LittleEndian.Uint32(frame[9:13]) != 1 {
		t.Fatalf("VIEW_ADD counted single frame=%x", frame)
	}
}

func TestLatestClientViewObjectPropertySingleExactLayout(t *testing.T) {
	frame, err := latestClientViewObjectPropertySingle(2, 1, viewInt(106, 5))
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 18 || frame[0] != 0x10 || frame[1] != 1 || binary.LittleEndian.Uint32(frame[2:6]) != 2 || binary.LittleEndian.Uint32(frame[6:10]) != 1 || binary.LittleEndian.Uint16(frame[10:12]) != 1 || binary.LittleEndian.Uint16(frame[12:14]) != 106 || binary.LittleEndian.Uint32(frame[14:18]) != 5 {
		t.Fatalf("SERVER_OBJECT_PROPERTY counted single frame=%x", frame)
	}
}

func TestLatestClientCurrentBagFramesAtomicRichObject(t *testing.T) {
	props := []serverViewProperty{
		viewString(7, "Game_item_hp_001"),
		viewInt(0x0761, 1),
		viewInt(0x0763, 1),
		viewString(0x0765, "7100568-001-0000000001-0001"), // historical UniqueID source; must stay off wire
		viewInt(0x0766, 5),
		viewInt(0x0767, 30),
		viewInt(0x076A, 1),
	}
	frames, err := latestClientCurrentBagFrames(2, 3, 1, props)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 {
		t.Fatalf("frame count=%d, want 1", len(frames))
	}
	f := frames[0]
	if f[0] != 0x18 || binary.LittleEndian.Uint16(f[1:3]) != 2 || binary.LittleEndian.Uint16(f[3:5]) != 3 || binary.LittleEndian.Uint16(f[5:7]) != 7 {
		t.Fatalf("rich VIEW_ADD header=%x", f[:7])
	}

	want := []uint16{7, 105, 106, 110, 205, 228, 234}
	off := 7
	for i, index := range want {
		if off+2 > len(f) {
			t.Fatalf("property[%d] truncated at %d frame=%x", i, off, f)
		}
		got := binary.LittleEndian.Uint16(f[off : off+2])
		off += 2
		if got != index {
			t.Fatalf("property[%d].ordinal=%d, want %d frame=%x", i, got, index, f)
		}
		switch index {
		case 7, 105:
			if off+4 > len(f) {
				t.Fatalf("string length truncated index=%d", index)
			}
			n := int(binary.LittleEndian.Uint32(f[off : off+4]))
			off += 4
			if n <= 0 || off+n > len(f) || f[off+n-1] != 0 {
				t.Fatalf("bad string property index=%d len=%d", index, n)
			}
			off += n
		default:
			if off+4 > len(f) {
				t.Fatalf("int32 truncated index=%d", index)
			}
			off += 4
		}
	}
	if off != len(f) {
		t.Fatalf("frame tail=%d bytes frame=%x", len(f)-off, f)
	}
}
