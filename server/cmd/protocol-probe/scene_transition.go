package main

import (
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/role"
	"math"
	"strconv"
	"strings"
	"time"
)

func sendEntryScene(conn sceneMessageConnection, scene role.Scene, name string) error {
	entry := make([]byte, 15, 128)
	entry[0] = 0x0B
	binary.LittleEndian.PutUint32(entry[1:], 0x0001F6A3)
	binary.LittleEndian.PutUint32(entry[5:], 0x11000001)
	binary.LittleEndian.PutUint32(entry[9:], 0x007A5C0B)
	entry = appendWideStringProperty(entry, 36, name)
	entry = binary.LittleEndian.AppendUint16(entry, 99)
	entry = binary.LittleEndian.AppendUint32(entry, 1000000)
	entry = appendStringProperty(entry, 7, scene.Config)
	entry = appendStringProperty(entry, 8, scene.Resource)
	binary.LittleEndian.PutUint16(entry[13:], 4)
	return conn.WriteFrame(entry)
}
func encodeExitScene() []byte {
	return []byte{0x0C}
}

type sceneDestination struct{ location role.Location }

func parseSceneDestination(value string) (sceneDestination, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 6 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return sceneDestination{}, fmt.Errorf("want config,resource,x,y,z,orient")
	}
	values := [4]float32{}
	for i := range values {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[i+2]), 32)
		if err != nil {
			return sceneDestination{}, fmt.Errorf("coordinate %d: %w", i, err)
		}
		values[i] = float32(parsed)
	}
	return sceneDestination{location: role.Location{Scene: role.Scene{Config: strings.TrimSpace(parts[0]), Resource: strings.TrimSpace(parts[1])}, Position: role.Position{X: values[0], Y: values[1], Z: values[2], Orient: values[3]}}}, nil
}
func parseScenePosition(value string) (role.Position, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return role.Position{}, fmt.Errorf("want x,y,z,orient")
	}
	values := [4]float32{}
	for i := range values {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 32)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			if err == nil {
				err = fmt.Errorf("must be finite")
			}
			return role.Position{}, fmt.Errorf("coordinate %d: %w", i, err)
		}
		values[i] = float32(parsed)
	}
	return role.Position{X: values[0], Y: values[1], Z: values[2], Orient: values[3]}, nil
}

type sceneRuntime struct {
	conn                       sceneMessageConnection
	world                      *sceneLifecycle
	machine                    interface{ ReenterScene() error }
	registry                   *sceneNPCRegistry
	staticNPCs                 []npcSpawn
	radius                     float32
	activeNPCs                 map[int]struct{}
	player                     *playerActor
	activeRole                 *role.RoleSnapshot
	npcCatalog                 []npcSpawn
	awaitingStableReentryReady bool
}

func (r *sceneRuntime) catalogForCurrentScene() ([]npcSpawn, error) {
	if r.registry == nil {
		return r.staticNPCs, nil
	}
	catalog, _, err := r.registry.catalogFor(r.activeRole.Location.Scene)
	if err != nil {
		return nil, err
	}
	return catalog, nil
}
func (r *sceneRuntime) SwitchScene(destination sceneDestination) error {
	if r.player == nil || r.activeRole == nil {
		return fmt.Errorf("switch scene without an active player")
	}
	destination.location.Scene = normalizeClientScene(destination.location.Scene)
	previous := r.activeRole.Location
	destination.location.Version = previous.Version
	r.activeRole.Location = destination.location
	catalog, err := r.catalogForCurrentScene()
	if err != nil {
		r.activeRole.Location = previous
		return err
	}
	if err := r.machine.ReenterScene(); err != nil {
		return err
	}
	if err := r.conn.WriteFrame(encodeExitScene()); err != nil {
		r.activeRole.Location = previous
		return fmt.Errorf("write ExitScene: %w", err)
	}
	if err := r.world.begin(destination.location.Scene.Config, destination.location.Scene.Resource); err != nil {
		return fmt.Errorf("begin replacement scene: %w", err)
	}
	time.Sleep(2750 * time.Millisecond) // latest-client REENTRY-EXIT-DRAIN A/B discriminator
	if err := sendEntryScene(r.conn, destination.location.Scene, r.activeRole.Name); err != nil {
		return fmt.Errorf("write EntryScene: %w", err)
	}
	// 2026-09-14 LIVE reached OnEntryScene create new after the 2750ms drain but
	// never reached flow player data ready. Reuse the same already-recovered
	// player spawn sequence that succeeds on initial entry: AddObject, Snapshot608,
	// Appearance, then Location/Vitals. This is an A/B discriminator, not a claim
	// that any one of those post-AddObject frames is independently proven causal.
	if err := sendPlayerSpawn(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {
		return fmt.Errorf("write full player re-entry spawn: %w", err)
	}
	clear(r.activeNPCs)
	if _, _, err := synchronizeNPCViewport(r.world, catalog, r.activeNPCs, destination.location.Position, r.radius); err != nil {
		return fmt.Errorf("register replacement NPC viewport: %w", err)
	}
	r.npcCatalog = catalog
	r.awaitingStableReentryReady = true
	return nil
}
func (r *sceneRuntime) TeleportWithinScene(position role.Position) error {
	if r.player == nil || r.activeRole == nil {
		return fmt.Errorf("teleport without an active player")
	}
	if r.awaitingStableReentryReady {
		return fmt.Errorf("target scene is still loading")
	}
	previous := r.activeRole.Location.Position
	r.activeRole.Location.Position = position
	if err := r.conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {
		r.activeRole.Location.Position = previous
		return fmt.Errorf("write same-scene ServerLocation: %w", err)
	}
	_, changed, err := synchronizeNPCViewport(r.world, r.npcCatalog, r.activeNPCs, position, r.radius)
	if err != nil {
		return fmt.Errorf("synchronize teleported NPC viewport: %w", err)
	}
	if changed {
		if err := r.world.reconcileViewport(); err != nil {
			return fmt.Errorf("reconcile teleported NPC viewport: %w", err)
		}
	}
	return nil
}
