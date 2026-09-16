package shopbuypublish

import (
	"errors"
	"testing"
)

func TestPublishWritesAllFramesInOrder(t *testing.T) {
	frames := [][]byte{{1}, {2, 3}, {4}}
	var got [][]byte
	result, err := Publish(frames, func(frame []byte) error {
		got = append(got, append([]byte(nil), frame...))
		return nil
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.FramesWritten != 3 || result.ResyncRequired {
		t.Fatalf("result=%+v", result)
	}
	for i := range frames {
		if string(got[i]) != string(frames[i]) {
			t.Fatalf("frame %d=%v want %v", i, got[i], frames[i])
		}
	}
}

func TestPublishFailureRequiresResyncWithoutRollback(t *testing.T) {
	frames := [][]byte{{1}, {2}, {3}}
	calls := 0
	result, err := Publish(frames, func(frame []byte) error {
		calls++
		if calls == 2 {
			return errors.New("wire down")
		}
		return nil
	})
	if err == nil || !errors.Is(err, ErrResyncRequired) {
		t.Fatalf("err=%v", err)
	}
	if result.FramesWritten != 1 || !result.ResyncRequired {
		t.Fatalf("result=%+v", result)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestPublishRejectsInvalidInputsBeforeWrite(t *testing.T) {
	if _, err := Publish(nil, func([]byte) error { return nil }); err == nil {
		t.Fatal("expected no-frames error")
	}
	calls := 0
	if _, err := Publish([][]byte{{1}, nil}, func([]byte) error { calls++; return nil }); err == nil {
		t.Fatal("expected empty-frame error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
	if _, err := Publish([][]byte{{1}}, nil); err == nil {
		t.Fatal("expected nil-writer error")
	}
}
