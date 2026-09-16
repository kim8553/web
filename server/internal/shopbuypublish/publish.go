package shopbuypublish

import (
	"errors"
	"fmt"
)

var ErrResyncRequired = errors.New("shopbuypublish: client resync required")

type Result struct {
	FramesWritten  int
	ResyncRequired bool
}

// Publish writes an already-built post-commit frame sequence. A write failure
// happens after durable/live state has already succeeded, so this helper never
// attempts rollback. Instead it marks the client as requiring resync/reconnect.
func Publish(frames [][]byte, write func([]byte) error) (Result, error) {
	var result Result
	if write == nil {
		return result, fmt.Errorf("shopbuypublish: nil writer")
	}
	if len(frames) == 0 {
		return result, fmt.Errorf("shopbuypublish: no frames")
	}
	for i, frame := range frames {
		if len(frame) == 0 {
			return result, fmt.Errorf("shopbuypublish: empty frame at index %d", i)
		}
		if err := write(frame); err != nil {
			result.ResyncRequired = true
			return result, fmt.Errorf("%w after %d frame(s): %v", ErrResyncRequired, result.FramesWritten, err)
		}
		result.FramesWritten++
	}
	return result, nil
}
