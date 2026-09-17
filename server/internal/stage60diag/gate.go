package stage60diag

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FrameWriter interface {
	WriteFrame([]byte) error
}

func ParseLimitEnv(envName string, max int) (limit int, enabled bool, err error) {
	raw, exists := os.LookupEnv(envName)
	if !exists || strings.TrimSpace(raw) == "" {
		return 0, false, nil
	}
	value, parseErr := strconv.Atoi(strings.TrimSpace(raw))
	if parseErr != nil || value < 0 || value > max {
		return 0, true, fmt.Errorf("%s=%q must be an integer in [0,%d]", envName, raw, max)
	}
	return value, true, nil
}

type GateWriter struct {
	Inner FrameWriter
	Limit int
	Logf  func(string, ...any)

	attempted int
	sent      int
}

func (g *GateWriter) Counts() (attempted int, sent int) {
	return g.attempted, g.sent
}

func (g *GateWriter) WriteFrame(frame []byte) error {
	g.attempted++
	opcode := byte(0)
	if len(frame) > 0 {
		opcode = frame[0]
	}
	prefix := frame
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	if g.attempted > g.Limit {
		if g.Logf != nil {
			g.Logf("STAGE60_BOOTSTRAP_FRAME_SUPPRESSED index=%d gate=%d opcode=0x%02X len=%d first16=% X", g.attempted, g.Limit, opcode, len(frame), prefix)
		}
		return nil
	}
	if g.Logf != nil {
		g.Logf("STAGE60_BOOTSTRAP_FRAME_SENT index=%d gate=%d opcode=0x%02X len=%d first16=% X", g.attempted, g.Limit, opcode, len(frame), prefix)
	}
	if g.Inner == nil {
		return fmt.Errorf("stage60diag: nil inner writer at permitted frame %d", g.attempted)
	}
	if err := g.Inner.WriteFrame(frame); err != nil {
		return err
	}
	g.sent++
	return nil
}
