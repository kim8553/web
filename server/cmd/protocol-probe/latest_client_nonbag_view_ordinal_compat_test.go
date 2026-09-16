package main

import "testing"

func TestLatestClientNonBagViewLegacyOrdinalsNormalizeWithinNegotiatedTable(t *testing.T) {
	props := []serverViewProperty{
		viewInt(0x0761, 1000),
		viewString(0x05A0, "CS_jh_cqgf01"),
		viewByte(0x08BC, 1),
		viewInt(0x08BD, 4401),
		viewByte(0x05B9, 3),
		viewInt(0x0823, 5),
		viewInt(0x07F2, 1000),
	}
	got, err := latestClientNonBagViewProperties(viewportSkill, props)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{205, 7, 225, 204, 6, 203, 212}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for i := range got {
		if got[i].index != want[i] {
			t.Fatalf("property[%d]=%d want=%d", i, got[i].index, want[i])
		}
		if int(got[i].index) >= latestClientPlayerWirePropertyTableCount {
			t.Fatalf("property[%d]=%d outside table=%d", i, got[i].index, latestClientPlayerWirePropertyTableCount)
		}
	}
	if got[4].int32 == nil || *got[4].int32 != 3 {
		t.Fatalf("Level conversion=%+v", got[4])
	}
	if got[6].real32 == nil || *got[6].real32 != 1000 {
		t.Fatalf("PauseTime conversion=%+v", got[6])
	}
}

func TestLatestClientNeiGongDropsOnlyUnnegotiatedLegacyMetadata(t *testing.T) {
	props := []serverViewProperty{
		viewString(0x05A0, "ng_jh_001"),
		viewInt(0x0761, 1002),
		viewInt(0x08BD, 123),
		viewByte(0x05B9, 2),
		viewInt(0x0823, 36),
		viewInt(0x08EC, 2),
		viewInt(0x02FD, 750),
		viewInt(0x08C3, 3),
		viewString(0x08ED, "buff_test"),
	}
	got, err := latestClientNonBagViewProperties(viewportNeiGong, props)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint16{7, 205, 204, 6, 203, 177}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d got=%+v", len(got), len(want), got)
	}
	for i := range got {
		if got[i].index != want[i] {
			t.Fatalf("property[%d]=%d want=%d", i, got[i].index, want[i])
		}
	}
}

func TestLatestClientNonBagViewRejectsUnknownOutOfRangeProperty(t *testing.T) {
	_, err := latestClientNonBagViewProperties(viewportSkill, []serverViewProperty{viewInt(0x0900, 1)})
	if err == nil {
		t.Fatal("expected unmapped out-of-range property rejection")
	}
}

func TestLatestClientNonBagServerViewAddNormalizesBeforeEncoding(t *testing.T) {
	frame, err := serverViewAdd(viewportNormalAttack, 1, []serverViewProperty{
		viewInt(0x0761, 1000), viewString(0x05A0, "CS_light_rad_81"), viewByte(0x08BC, 1),
		viewInt(0x08BD, 5266), viewByte(0x05B9, 1), viewInt(0x0823, 1), viewInt(0x07F2, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) < 7 || frame[0] != 0x18 {
		t.Fatalf("frame=%x", frame)
	}
}

func TestLatestClientNonBagServerObjectPropertyNormalizesBeforeEncoding(t *testing.T) {
	frame, err := serverObjectProperty(viewportSkill, 7, []serverViewProperty{viewByte(0x08BC, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 15 {
		t.Fatalf("len(frame)=%d frame=%x", len(frame), frame)
	}
	if frame[0] != 0x10 || frame[1] != 0x01 {
		t.Fatalf("opcode/view flag=%x/%x frame=%x", frame[0], frame[1], frame)
	}
	if got := uint16(frame[12]) | uint16(frame[13])<<8; got != 225 {
		t.Fatalf("encoded CanUse ordinal=%d want=225 frame=%x", got, frame)
	}
	if frame[14] != 1 {
		t.Fatalf("encoded CanUse value=%d want=1 frame=%x", frame[14], frame)
	}
}

func TestLatestClientNonBagServerObjectPropertyWidensLegacyLevel(t *testing.T) {
	frame, err := serverObjectProperty(45, 3, []serverViewProperty{viewByte(0x05B9, 4)})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 18 {
		t.Fatalf("len(frame)=%d frame=%x", len(frame), frame)
	}
	if got := uint16(frame[12]) | uint16(frame[13])<<8; got != 6 {
		t.Fatalf("encoded Level ordinal=%d want=6 frame=%x", got, frame)
	}
	if got := uint32(frame[14]) | uint32(frame[15])<<8 | uint32(frame[16])<<16 | uint32(frame[17])<<24; got != 4 {
		t.Fatalf("encoded Level value=%d want=4 frame=%x", got, frame)
	}
}
