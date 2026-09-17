package main

import (
	"log"

	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/stage60diag"
)

const stage60BootstrapGateEnv = "JIUYIN_STAGE60_BOOTSTRAP_GATE"

// stage60BootstrapGateFromEnv enables the Stage60 LIVE-only scene bootstrap
// diagnostic when JIUYIN_STAGE60_BOOTSTRAP_GATE is explicitly set. A gate of
// 0 sends no sendPlayerSpawn frames (PlayerEntry has already been sent by the
// caller); 1..5 allow that many existing sendPlayerSpawn WriteFrame calls.
// No opcode or field is invented here: the wrapper only observes/suppresses
// frames the current implementation would already emit.
func stage60BootstrapGateFromEnv() (limit int, enabled bool, err error) {
	return stage60diag.ParseLimitEnv(stage60BootstrapGateEnv, 5)
}

func stage60SendPlayerSpawnGated(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual, limit int) (attempted int, sent int, err error) {
	gate := &stage60diag.GateWriter{Inner: conn, Limit: limit, Logf: log.Printf}
	err = sendPlayerSpawn(gate, player, location, visual)
	attempted, sent = gate.Counts()
	return attempted, sent, err
}
