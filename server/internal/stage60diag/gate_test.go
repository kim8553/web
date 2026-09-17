package stage60diag

import (
	"bytes"
	"testing"
)

type captureWriter struct {
	frames [][]byte
}

func (c *captureWriter) WriteFrame(frame []byte) error {
	c.frames = append(c.frames, append([]byte(nil), frame...))
	return nil
}

func TestParseLimitEnv(t *testing.T) {
	const name = "STAGE60_DIAG_TEST_GATE"
	t.Setenv(name, "3")
	limit, enabled, err := ParseLimitEnv(name, 5)
	if err != nil || !enabled || limit != 3 {
		t.Fatalf("limit=%d enabled=%v err=%v", limit, enabled, err)
	}

	t.Setenv(name, "6")
	_, enabled, err = ParseLimitEnv(name, 5)
	if !enabled || err == nil {
		t.Fatalf("out-of-range gate enabled=%v err=%v", enabled, err)
	}
}

func TestGateWriterSuppressesAfterLimit(t *testing.T) {
	capture := &captureWriter{}
	gate := &GateWriter{Inner: capture, Limit: 2}
	frames := [][]byte{{0x0D, 1}, {0x30, 2, 3}, {0x1F, 4}, {0x10, 5}}
	for _, frame := range frames {
		if err := gate.WriteFrame(frame); err != nil {
			t.Fatal(err)
		}
	}
	attempted, sent := gate.Counts()
	if attempted != 4 || sent != 2 {
		t.Fatalf("attempted=%d sent=%d, want 4/2", attempted, sent)
	}
	if len(capture.frames) != 2 {
		t.Fatalf("captured=%d, want 2", len(capture.frames))
	}
	if !bytes.Equal(capture.frames[0], frames[0]) || !bytes.Equal(capture.frames[1], frames[1]) {
		t.Fatalf("captured frames differ: %#v", capture.frames)
	}
}
