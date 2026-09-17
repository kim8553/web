package main

import (
	"bytes"
	"testing"
)

type stage60BootstrapCapture struct {
	frames [][]byte
}

func (c *stage60BootstrapCapture) WriteFrame(frame []byte) error {
	c.frames = append(c.frames, append([]byte(nil), frame...))
	return nil
}

func TestStage60BootstrapGateFromEnv(t *testing.T) {
	t.Setenv(stage60BootstrapGateEnv, "3")
	limit, enabled, err := stage60BootstrapGateFromEnv()
	if err != nil || !enabled || limit != 3 {
		t.Fatalf("limit=%d enabled=%v err=%v", limit, enabled, err)
	}

	t.Setenv(stage60BootstrapGateEnv, "6")
	_, enabled, err = stage60BootstrapGateFromEnv()
	if !enabled || err == nil {
		t.Fatalf("out-of-range gate enabled=%v err=%v", enabled, err)
	}
}

func TestStage60BootstrapGateConnectionSuppressesAfterLimit(t *testing.T) {
	capture := &stage60BootstrapCapture{}
	gate := &stage60BootstrapGateConnection{inner: capture, limit: 2}
	frames := [][]byte{{0x0D, 1}, {0x30, 2, 3}, {0x1F, 4}, {0x10, 5}}
	for _, frame := range frames {
		if err := gate.WriteFrame(frame); err != nil {
			t.Fatal(err)
		}
	}
	if gate.attempted != 4 || gate.sent != 2 {
		t.Fatalf("attempted=%d sent=%d, want 4/2", gate.attempted, gate.sent)
	}
	if len(capture.frames) != 2 {
		t.Fatalf("captured=%d, want 2", len(capture.frames))
	}
	if !bytes.Equal(capture.frames[0], frames[0]) || !bytes.Equal(capture.frames[1], frames[1]) {
		t.Fatalf("captured frames differ: %#v", capture.frames)
	}
}
