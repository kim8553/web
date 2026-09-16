package main

import (
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type npcCombatState struct {
	home            worldcore.Transform
	aggroRange      float32
	chaseRange      float32
	attackRange     float32
	moveSpeed       float32
	damage          int32
	attackEvery     time.Duration
	respawnAfter    time.Duration
	attackSkill     string
	damageSkill     string
	name            string
	threat          int32
	nextAttack      time.Time
	respawnAt       time.Time
	controlledUntil time.Time
	moving          bool
	leaveCombatAt   time.Time
}
type combatTickResult struct {
	playerStateChanged bool
	playerRevived      bool
}
type npcMoveUpdate struct {
	id          uint32
	ownerID     uint32
	from        worldcore.Transform
	destination worldcore.Transform
	speed       float32
}
type npcPropertyUpdate struct {
	id         uint32
	ownerID    uint32
	properties []clientdata.IndexedProperty
}

const (
	combatTickMaxAggressors = 4
	defaultChaseRange       = float32(30)
	defaultAttackRange      = float32(3.5)
	defaultNPCMoveSpeed     = float32(4.5)
	defaultAttackEvery      = 1500 * time.Millisecond
	defaultNPCRespawn       = 15 * time.Second
	playerAutoReviveDelay   = 8 * time.Second
)

func newNPCCombatState(npc npcSpawn, home worldcore.Transform) npcCombatState {
	aggro := npcWakeupRange(npc)
	chase := npc.resolved.Properties["ChaseRange"].Value.F32
	if chase < aggro {
		chase = 30
	}
	move := npc.resolved.Properties["RunSpeed"].Value.F32
	if move <= 0 {
		move = npc.resolved.Properties["MoveSpeed"].Value.F32
	}
	if move <= 0 {
		move = 4
	}
	maxHP := npc.resolved.Properties["MaxHP"].Value.I32
	if maxHP <= 0 {
		maxHP = 1000
	}
	damage := maxHP / 100
	if damage < 10 {
		damage = 10
	}
	if npc.resolved.ScriptClass == "BossNpc" {
		damage *= 2
	}
	return npcCombatState{home: home, aggroRange: aggro, chaseRange: chase, attackRange: 2, moveSpeed: move, damage: damage, attackEvery: 1500 * time.Millisecond, respawnAfter: 15 * time.Second, attackSkill: npcAttackSkill(npc), damageSkill: npcDamageSkill(npc), name: npcCombatName(npc)}
}

// npcWakeupRange returns the authored hostile wake/alert radius. The exact
// current share.package npc/npcconfig/creator_config.ini names WakeupRange as
// 警戒半径 (alert radius) and ChaseRange as 追击半径 (chase radius). Creator
// XML can override WakeupRange per spawn; ResolveNPC keeps that non-visible
// attribute in Extensions as creator.WakeupRange. If there is no per-spawn
// value, only defaults explicitly present in the current creator_config.ini are
// used. Unknown classes stay passive until another combat event creates threat.
func npcWakeupRange(npc npcSpawn) float32 {
	if raw := strings.TrimSpace(npc.resolved.Extensions["creator.WakeupRange"]); raw != "" {
		if value, err := strconv.ParseFloat(raw, 32); err == nil && value >= 0 {
			return float32(value)
		}
	}
	switch strings.ToLower(strings.TrimSpace(npc.resolved.ScriptClass)) {
	case "attacknpc":
		return 16
	case "commonnpc", "funcnpc":
		return 12
	case "gathernpc", "other":
		return 0
	default:
		return 0
	}
}

func (n npcSpawn) float32Property(name string) float32 {
	return n.resolved.Properties[name].Value.F32
}
func (s *sceneLifecycle) combatTick(player *playerActor, position role.Position, now time.Time) (combatTickResult, error) {
	if s == nil || player == nil {
		return combatTickResult{}, nil
	}
	result := combatTickResult{}
	result.playerStateChanged = player.expireInnerPowerPassive(now)
	result.playerRevived = player.reviveWhenDue(now)
	playerState := player.actor.Snapshot()
	result.playerStateChanged = result.playerStateChanged || result.playerRevived
	if playerState.HP <= 0 {
		return result, nil
	}
	playerTransform := worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
	s.mu.Lock()
	if s.closed || !s.combatActive || len(s.combatStates) == 0 {
		s.mu.Unlock()
		return result, nil
	}
	ids := make([]int, 0, len(s.combatStates))
	for id := range s.combatStates {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	moves := make([]npcMoveUpdate, 0, 4)
	updates := make([]npcPropertyUpdate, 0, 2)
	locations := make([]struct {
		id        uint32
		ownerID   uint32
		transform worldcore.Transform
	}, 0, 2)
	aggressors := 0
	threatened := false
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		state, exists := s.combatStates[id]
		if !exists {
			continue
		}
		meta, exists := s.entities[id]
		actor := s.combatActors[id]
		if !exists || actor == nil {
			continue
		}
		actorState := actor.Snapshot()
		if actorState.HP <= 0 {
			if !state.respawnAt.IsZero() && !now.Before(state.respawnAt) {
				actor.Revive()
				state.threat = 0
				state.nextAttack = time.Time{}
				state.respawnAt = time.Time{}
				state.leaveCombatAt = time.Time{}
				if err := s.setNPCTransformLocked(id, state.home); err != nil {
					s.mu.Unlock()
					return result, err
				}
				meta = s.entities[id]
				locations = append(locations, struct {
					id        uint32
					ownerID   uint32
					transform worldcore.Transform
				}{id: meta.id, ownerID: meta.ownerID, transform: meta.transform})
				updates = append(updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(actor.Snapshot())})
			} else if !state.leaveCombatAt.IsZero() && now.Before(state.leaveCombatAt) {
				threatened = true
			}
			s.combatStates[id] = state
			continue
		}
		distance := horizontalDistance(meta.transform, playerTransform)
		if state.threat == 0 {
			if state.aggroRange <= 0 || distance > state.aggroRange {
				continue
			}
			state.threat = 1
		}
		if distance > state.chaseRange {
			state.threat = 0
			if horizontalDistance(meta.transform, state.home) > 0.25 {
				if err := s.setNPCTransformLocked(id, state.home); err != nil {
					s.mu.Unlock()
					return result, err
				}
				meta = s.entities[id]
				locations = append(locations, struct {
					id        uint32
					ownerID   uint32
					transform worldcore.Transform
				}{id: meta.id, ownerID: meta.ownerID, transform: meta.transform})
			}
			s.combatStates[id] = state
			continue
		}
		threatened = true
		if now.Before(state.controlledUntil) {
			s.combatStates[id] = state
			continue
		}
		if aggressors >= 4 {
			s.combatStates[id] = state
			continue
		}
		aggressors++
		if distance > state.attackRange {
			s.combatStates[id] = state
			continue
		}
		yDelta := math.Abs(float64(meta.transform.Y - position.Y))
		if yDelta > 0.5 {
			step := minFloat32(float32(yDelta), state.moveSpeed)
			destination := meta.transform
			if position.Y > meta.transform.Y {
				destination.Y += step
			} else {
				destination.Y -= step
			}
			if err := s.setNPCTransformLocked(id, destination); err != nil {
				s.mu.Unlock()
				return result, err
			}
			moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, from: meta.transform, destination: destination, speed: state.moveSpeed})
			s.combatStates[id] = state
			continue
		}
		if now.Before(state.nextAttack) {
			s.combatStates[id] = state
			continue
		}
		state.nextAttack = now.Add(state.attackEvery)
		outcome := player.resolveNPCAttack(meta.id, state.damage, now)
		if outcome.innerPowerTriggered {
			log.Printf("%s: inner-power incoming passive triggered id=%d config=%s", s.remote, meta.id, meta.configID)
		}
		result.playerStateChanged = result.playerStateChanged || outcome.innerPowerTriggered
		if outcome.dodged {
			if outcome.innerPowerHeal > 0 {
				playerState := player.actor.Snapshot()
				log.Printf("%s: NPC attack dodged id=%d config=%s inner-power dodge-heal=%d player_hp=%d/%d", s.remote, meta.id, meta.configID, outcome.innerPowerHeal, playerState.HP, playerState.MaxHP)
			} else {
				log.Printf("%s: NPC attack dodged id=%d config=%s", s.remote, meta.id, meta.configID)
			}
		} else if outcome.stateChanged {
			playerState := player.actor.Snapshot()
			log.Printf("%s: NPC attack id=%d config=%s damage=%d player_hp=%d/%d", s.remote, meta.id, meta.configID, state.damage, playerState.HP, playerState.MaxHP)
		} else if outcome.parried {
			log.Printf("%s: NPC attack parried id=%d config=%s", s.remote, meta.id, meta.configID)
		}
		result.playerStateChanged = result.playerStateChanged || outcome.stateChanged
		if err := s.writeNPCAttackFrames(meta, state, player, !outcome.dodged && !outcome.parried); err != nil {
			s.mu.Unlock()
			return result, fmt.Errorf("send NPC attack frames %d: %w", meta.id, err)
		}
		if outcome.dead {
			if player.grantZhenQi(10) {
				log.Printf("%s: kill zhenqi reward +%d id=%d config=%s", s.remote, 10, meta.id, meta.configID)
			}
			result.playerStateChanged = true
		}
		s.combatStates[id] = state
	}
	result.playerStateChanged = player.setCombatPresence(threatened) || result.playerStateChanged
	s.mu.Unlock()
	for _, move := range moves {
		speed := move.speed
		if speed <= 0 {
			speed = 1.77
		}
		heading := headingTowards(move.from, move.destination)
		if err := s.conn.WriteFrame(serverEntityMove([]entityMove{{id: move.id, position: move.from, movement: makeMovementData(speed, heading)}})); err != nil {
			return result, fmt.Errorf("combat move NPC %d: %w", move.id, err)
		}
		s.scheduleCombatMoveArrival(move, time.Second)
	}
	for _, location := range locations {
		if err := s.conn.WriteFrame(serverLocation(location.id, location.ownerID, location.transform)); err != nil {
			return result, fmt.Errorf("snap NPC %d home: %w", location.id, err)
		}
	}
	for _, update := range updates {
		frame, err := sceneObjectProperties(update.id, update.ownerID, 0, update.properties)
		if err != nil {
			return result, err
		}
		if err := s.conn.WriteFrame(frame); err != nil {
			return result, fmt.Errorf("combat NPC state %d: %w", update.id, err)
		}
	}
	return result, nil
}
func (s *sceneLifecycle) scheduleCombatMoveArrival(move npcMoveUpdate, delay time.Duration) {
	if delay <= 0 {
		delay = time.Second
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	epoch := s.epoch
	timer := time.AfterFunc(delay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.epoch != epoch {
			return
		}
		meta, exists := s.entities[worldcore.EntityID(move.id)]
		if !exists || meta.ownerID != move.ownerID || !sameCombatTransform(meta.transform, move.destination) {
			return
		}
		if err := s.conn.WriteFrame(serverLocation(meta.id, meta.ownerID, meta.transform)); err != nil {
			log.Printf("%s: confirm combat NPC move id=%d: %v", s.remote, meta.id, err)
		}
	})
	s.timers = append(s.timers, timer)
	s.mu.Unlock()
}
func sameCombatTransform(a, b worldcore.Transform) bool {
	return math.Abs(float64(a.X-b.X)) < 0.01 && math.Abs(float64(a.Y-b.Y)) < 0.01 && math.Abs(float64(a.Z-b.Z)) < 0.01 && math.Abs(float64(a.Orient-b.Orient)) < 0.01
}
func (s *sceneLifecycle) setNPCTransformLocked(id worldcore.EntityID, destination worldcore.Transform) error {
	meta, exists := s.entities[id]
	if !exists {
		return fmt.Errorf("set transform for unknown NPC %d", id)
	}
	current, err := s.scene.Get(id)
	if err != nil {
		return err
	}
	current.Entity.Transform = destination
	properties := []struct {
		name  string
		value float32
	}{{name: "PosiX", value: destination.X}, {name: "PosiY", value: destination.Y}, {name: "PosiZ", value: destination.Z}, {name: "Orient", value: destination.Orient}}
	for _, property := range properties {
		if err := current.Entity.Properties.Set(property.name, worldcore.Value{Kind: worldcore.ValueFloat32, Float: property.value}); err != nil {
			return err
		}
	}
	if err := s.scene.Replace(current.Entity); err != nil {
		return err
	}
	meta.transform = destination
	s.entities[id] = meta
	return nil
}
func npcCombatProperties(state entity.ActorState) []clientdata.IndexedProperty {
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	dead := int32(0)
	if state.HP <= 0 {
		dead = 1
	}
	return []clientdata.IndexedProperty{{Index: 21, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)}, {Index: 28, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)}, {Index: 30, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)}, {Index: 32, Name: "HP", Value: clientdata.Int32Value(state.HP)}, {Index: 112, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)}, {Index: 209, Name: "Dead", Value: clientdata.Int32Value(dead)}, {Index: 463, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)}, {Index: 465, Name: "HP", Value: clientdata.Int32Value(state.HP)}, {Index: 469, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)}, {Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)}, {Index: 464, Name: "Dead", Value: clientdata.ByteValue(uint8(dead))}}
}
func horizontalDistance(a, b worldcore.Transform) float32 {
	dx := float64(a.X - b.X)
	dz := float64(a.Z - b.Z)
	return float32(math.Hypot(dx, dz))
}
func stepToward(from, to worldcore.Transform, step float32) worldcore.Transform {
	distance := horizontalDistance(from, to)
	if distance <= 0 || step <= 0 {
		return from
	}
	ratio := min(step, distance) / distance
	result := worldcore.Transform{X: from.X + (to.X-from.X)*ratio, Y: from.Y + (to.Y-from.Y)*ratio, Z: from.Z + (to.Z-from.Z)*ratio}
	result.Orient = float32(math.Atan2(float64(to.Z-from.Z), float64(to.X-from.X)))
	return result
}
func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
