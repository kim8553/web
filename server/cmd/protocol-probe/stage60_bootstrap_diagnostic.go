package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/role"
)

const stage60BootstrapGateEnv = "JIUYIN_STAGE60_BOOTSTRAP_GATE"

// stage60BootstrapGateFromEnv enables the Stage60 LIVE-only scene bootstrap
// diagnostic when JIUYIN_STAGE60_BOOTSTRAP_GATE is explicitly set.  A gate of
// 0 sends no sendPlayerSpawn frames (PlayerEntry has already been sent by the
// caller); 1..5 allow that many existing sendPlayerSpawn WriteFrame calls.
// No opcode or field is invented here: the wrapper only observes/suppresses
// frames the current implementation would already emit.
func stage60BootstrapGateFromEnv() (limit int, enabled bool, err error) {
	raw, exists := os.LookupEnv(stage60BootstrapGateEnv)
	if !exists || strings.TrimSpace(raw) == "" {
		return 0, false, nil
	}
	value, parseErr := strconv.Atoi(strings.TrimSpace(raw))
	if parseErr != nil || value < 0 || value > 5 {
		return 0, true, fmt.Errorf("%s=%q must be an integer in [0,5]", stage60BootstrapGateEnv, raw)
	}
	return value, true, nil
}

type stage60BootstrapGateConnection struct {
	inner     sceneMessageConnection
	limit     int
	attempted int
	sent      int
}

func (g *stage60BootstrapGateConnection) WriteFrame(frame []byte) error {
	g.attempted++
	opcode := byte(0)
	if len(frame) > 0 {
		opcode = frame[0]
	}
	prefix := frame
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	if g.attempted > g.limit {
		log.Printf("STAGE60_BOOTSTRAP_FRAME_SUPPRESSED index=%d gate=%d opcode=0x%02X len=%d first16=% X", g.attempted, g.limit, opcode, len(frame), prefix)
		return nil
	}
	log.Printf("STAGE60_BOOTSTRAP_FRAME_SENT index=%d gate=%d opcode=0x%02X len=%d first16=% X", g.attempted, g.limit, opcode, len(frame), prefix)
	if err := g.inner.WriteFrame(frame); err != nil {
		return err
	}
	g.sent++
	return nil
}

func stage60SendPlayerSpawnGated(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual, limit int) (attempted int, sent int, err error) {
	gate := &stage60BootstrapGateConnection{inner: conn, limit: limit}
	err = sendPlayerSpawn(gate, player, location, visual)
	return gate.attempted, gate.sent, err
}
