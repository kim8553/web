package main

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

func TestLatestClientPlayerPropertyOrdinalMapMatchesNegotiatedTable(t *testing.T) {
	if latestClientPlayerWirePropertyTableCount != 234 {
		t.Fatalf("negotiated property table count=%d, want 234 after ViewID plus proven int64 currencies", latestClientPlayerWirePropertyTableCount)
	}
	cases := []struct {
		name string
		typ  clientdata.WireType
		want uint16
	}{
		{"Type", clientdata.WireByte, 0},
		{"MoveSpeed", clientdata.WireFloat32, 23},
		{"MaxHP", clientdata.WireInt32, 30},
		{"HP", clientdata.WireInt32, 32},
		{"NpcType", clientdata.WireInt32, 34},
		{"Name", clientdata.WireWideString, 36},
		{"CapitalType1", clientdata.WireInt64, 111},
		{"QingGongPoint", clientdata.WireInt32, 114},
		{"Gravity", clientdata.WireFloat32, 168},
		{"Force", clientdata.WireString, 226},
		{"NewSchool", clientdata.WireString, 227},
		{"ViewID", clientdata.WireInt32, 228},
		{"CapitalType2", clientdata.WireInt64, 229},
		{"CapitalType4", clientdata.WireInt64, 230},
		{"CapitalType0", clientdata.WireInt64, 231},
		{"CapitalType3", clientdata.WireInt64, 232},
		{"ExchangeData", clientdata.WireInt32, 233},
	}
	for _, tc := range cases {
		got, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: tc.name, typ: tc.typ}]
		if !ok || got != tc.want {
			t.Fatalf("%s/%s ordinal=(%d,%v), want (%d,true)", tc.name, tc.typ, got, ok, tc.want)
		}
	}
	if _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "Job", typ: clientdata.WireString}]; ok {
		t.Fatal("Job/string must not be guessed onto the negotiated Job/int32 slot")
	}
}

func TestLatestClientPlayerWirePropertiesDropsUnknownMismatchAndDeduplicates(t *testing.T) {
	raw := []clientdata.IndexedProperty{
		{Index: 181, Name: "Type", Value: clientdata.ByteValue(2)},
		{Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(1000)},
		{Index: 1588, Name: "Job", Value: clientdata.StringValue("")},
		{Index: 405, Name: "ActionSet", Value: clientdata.StringValue("set")},
		{Index: 406, Name: "ActionSet", Value: clientdata.StringValue("set")},
		{Index: 9999, Name: "NoSuchProperty", Value: clientdata.Int32Value(1)},
	}
	got := latestClientPlayerWireProperties(raw)
	if len(got) != 3 {
		t.Fatalf("normalized count=%d, want 3: %#v", len(got), got)
	}
	want := []uint16{0, 30, 18}
	for i, index := range want {
		if got[i].Index != index {
			t.Fatalf("normalized[%d].Index=%d, want %d", i, got[i].Index, index)
		}
	}
}

func TestSceneObjectPropertiesNormalizesAllNonViewSceneObjects(t *testing.T) {
	raw := []clientdata.IndexedProperty{{Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(1000)}}
	playerFrame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(playerFrame[12:14]); got != 30 {
		t.Fatalf("player MaxHP wire index=%d, want 30", got)
	}
	npcFrame, err := sceneObjectProperties(0x1234, 1, 0, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(npcFrame[12:14]); got != 30 {
		t.Fatalf("NPC MaxHP wire index=%d, want negotiated ordinal 30", got)
	}
}
