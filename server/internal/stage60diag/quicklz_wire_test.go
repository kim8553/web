package stage60diag

import (
	"bytes"
	"encoding/binary"
	"testing"

	quicklz "github.com/Hiroko103/go-quicklz"
)

// The actual Stage60 LIVE frame 0x28 began with an uncompressed entity ID
// because _builddeps/quicklz copied source bytes verbatim. This regression
// checks the dependency actually emits a QuickLZ stream, not merely a frame
// with the correct 0x28 opcode. It does not prove the complete client protocol.
func TestSceneSnapshotQuickLZIsRealCompression(t *testing.T) {
	raw := make([]byte, 62, 320)
	binary.LittleEndian.PutUint64(raw, 0x007A5C0B11000001)
	for i := 8; i < len(raw); i++ {
		raw[i] = byte(i % 13)
	}
	raw = append(raw, bytes.Repeat([]byte{0x11, 0x00, 0x42, 0x00, 0x11, 0x00}, 40)...)
	encoder, err := quicklz.New(quicklz.COMPRESSION_LEVEL_1, quicklz.STREAMING_BUFFER_0)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, len(raw)+400)
	n, err := encoder.Compress(&raw, &buffer)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if n < 9 || int(n) > len(buffer) {
		t.Fatalf("missing QuickLZ long header: n=%d", n)
	}
	encoded := append([]byte(nil), buffer[:n]...)
	if encoded[0]&0x40 == 0 {
		t.Fatalf("QuickLZ header flag absent (identity-only shim?): first=%02x", encoded[0])
	}
	if bytes.Equal(encoded, raw) || bytes.Equal(encoded[:8], raw[:8]) {
		t.Fatalf("compression returned raw entity ID instead of a QuickLZ stream")
	}
	if got := quicklz.Size_decompressed(&encoded); got != int64(len(raw)) {
		t.Fatalf("QuickLZ header raw size=%d, want=%d", got, len(raw))
	}
	decoded := make([]byte, len(raw))
	decoder, err := quicklz.New(quicklz.COMPRESSION_LEVEL_1, quicklz.STREAMING_BUFFER_0)
	if err != nil {
		t.Fatal(err)
	}
	m, err := decoder.Decompress(&encoded, &decoded)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if m != int64(len(raw)) || !bytes.Equal(raw, decoded) {
		t.Fatalf("round-trip mismatch: got=%d want=%d", m, len(raw))
	}
}
