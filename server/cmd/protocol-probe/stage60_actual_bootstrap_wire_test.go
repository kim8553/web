package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	quicklz "github.com/Hiroko103/go-quicklz"
	"github.com/local/9yin-go-server/internal/role"
)

// Exercise the actual player spawn encoder, not a synthetic compressor sample.
// This checks our server-side wire contract only: it cannot assert that a
// proprietary client accepts the frame or reaches ClientReady.
func TestStage60ActualPlayerSpawnUnGatedQuickLZWire(t *testing.T) {
	player := newPlayerActor("offline-fixture", 37)
	loc := role.Position{X: 103.5, Y: 12.25, Z: 27.75, Orient: 1.125}
	visual := resolveRoleVisual(nil)
	conn := &captureMessageConnection{}
	if err := sendPlayerSpawn(conn, player, loc, visual); err != nil {
		t.Fatalf("sendPlayerSpawn: %v", err)
	}
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("un-gated full spawn frame count=%d, want 5", len(frames))
	}
	for i, want := range []byte{0x10, 0x28, 0x10, 0x1f, 0x10} {
		if len(frames[i]) == 0 || frames[i][0] != want {
			t.Fatalf("spawn frame[%d] opcode=%x, want 0x%02x", i, frames[i], want)
		}
	}
	if got := len(frames[3]); got != 25 {
		t.Fatalf("location frame len=%d, want 25", got)
	}
	wire := frames[1][1:]
	if len(wire) < 9 || wire[0]&0x40 == 0 {
		t.Fatalf("snapshot missing actual QuickLZ long header: %x", frames[1][:min(len(frames[1]), 16)])
	}
	if got := int(binary.LittleEndian.Uint32(wire[1:5])); got != len(wire) {
		t.Fatalf("QuickLZ compressed-size header=%d actual=%d", got, len(wire))
	}
	rawLen := int(quicklz.Size_decompressed(&wire))
	if rawLen < 62 || rawLen > 2_000_000 {
		t.Fatalf("QuickLZ raw payload size=%d", rawLen)
	}
	raw := make([]byte, rawLen)
	dec, err := quicklz.New(quicklz.COMPRESSION_LEVEL_1, quicklz.STREAMING_BUFFER_0)
	if err != nil {
		t.Fatal(err)
	}
	n, err := dec.Decompress(&wire, &raw)
	if err != nil || int(n) != rawLen {
		t.Fatalf("snapshot decompression: bytes=%d want=%d err=%v", n, rawLen, err)
	}
	entityID := uint64(playerObjectID) | uint64(playerOwnerID)<<32
	if got := binary.LittleEndian.Uint64(raw[:8]); got != entityID {
		t.Fatalf("snapshot entityID=%#x, want=%#x", got, entityID)
	}
	for i, coord := range []float32{loc.X, loc.Y, loc.Z, loc.Orient} {
		expected := math.Float32bits(coord)
		for _, offset := range []int{8, 24} {
			if got := binary.LittleEndian.Uint32(raw[offset+i*4:]); got != expected {
				t.Fatalf("snapshot transform at %d got=%#x want=%#x", offset+i*4, got, expected)
			}
		}
	}
	if !bytes.Equal(raw[40:60], make([]byte, 20)) {
		t.Fatalf("snapshot reserved transform region not zero: %x", raw[40:60])
	}
	props := latestClientPlayerWireProperties(player.playerBirthProperties(loc, visual))
	if got := int(binary.LittleEndian.Uint16(raw[60:62])); got != len(props) {
		t.Fatalf("snapshot property count=%d want=%d", got, len(props))
	}
	expected := make([]byte, 0, len(raw)-62)
	for _, prop := range props {
		expected = appendNPCProperty(expected, prop)
	}
	if !bytes.Equal(raw[62:], expected) {
		t.Fatalf("snapshot property bytes differ from negotiated player property encoder")
	}
	t.Logf("stage60 full un-gated player spawn: frames=%d snapshot_compressed=%d snapshot_raw=%d properties=%d", len(frames), len(wire), len(raw), len(props))
}
