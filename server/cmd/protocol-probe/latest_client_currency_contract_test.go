package main

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

func TestLatestClientCurrencyPropertiesUseCurrentInt64Contract(t *testing.T) {
	cases := []struct {
		name string
		want uint16
	}{
		{"CapitalType1", 111},
		{"CapitalType2", 229},
		{"CapitalType4", 230},
		{"CapitalType0", 231},
		{"CapitalType3", 232},
	}
	for _, tc := range cases {
		got, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: tc.name, typ: clientdata.WireInt64}]
		if !ok || got != tc.want {
			t.Fatalf("%s/int64 ordinal=(%d,%v), want (%d,true)", tc.name, got, ok, tc.want)
		}
	}
	if _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "CapitalType1", typ: clientdata.WireInt32}]; ok {
		t.Fatal("CapitalType1/int32 must not remain in the current negotiated table")
	}
}

func TestCurrencyUpdatePublishesCurrentResourceProvenInt64Properties(t *testing.T) {
	p := &playerActor{silver: 101, gold: 202, silverCard: 303, silverTicket: 404}
	frame, err := p.currenciesUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) < 12 || frame[0] != 0x10 || frame[1] != 0 {
		t.Fatalf("unexpected ServerObjectProperty header: %x", frame)
	}
	if got := binary.LittleEndian.Uint16(frame[10:12]); got != 4 {
		t.Fatalf("currency property count=%d, want 4 wallet properties", got)
	}
	off := 12
	want := []struct {
		ordinal uint16
		value   int64
	}{
		{111, 101},
		{231, 202},
		{229, 303},
		{230, 404},
	}
	for i, tc := range want {
		if off+10 > len(frame) {
			t.Fatalf("currency[%d] truncated at %d/%d", i, off, len(frame))
		}
		if got := binary.LittleEndian.Uint16(frame[off : off+2]); got != tc.ordinal {
			t.Fatalf("currency[%d] ordinal=%d, want %d", i, got, tc.ordinal)
		}
		if got := int64(binary.LittleEndian.Uint64(frame[off+2 : off+10])); got != tc.value {
			t.Fatalf("currency[%d] value=%d, want %d", i, got, tc.value)
		}
		off += 10
	}
	if off != len(frame) {
		t.Fatalf("currency frame has %d trailing bytes", len(frame)-off)
	}
}
