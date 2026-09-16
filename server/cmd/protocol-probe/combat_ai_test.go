package main

import (
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func registerTestCombatNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32) {
	t.Helper()
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_test"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.x, npc.y, npc.z = 1, 0, 0
	if err := lifecycle.registerNPC(id, npc); err != nil {
		t.Fatal(err)
	}
	lifecycle.combatActive = true
}

func TestCombatTickAggroesAndDamagesNearbyPlayer(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 81)
	id := worldcore.EntityID(81)
	state := lifecycle.combatStates[id]
	state.threat = 1
	lifecycle.combatStates[id] = state
	player := newPlayerActor("tester", 0)
	before := player.actor.Snapshot().HP
	result, err := lifecycle.combatTick(player, role.Position{X: 1}, time.Unix(1000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("current threat did not enter combat: result=%+v actor=%+v", result, player.actor.Snapshot())
	}
	if after := player.actor.Snapshot().HP; after >= before {
		t.Fatalf("NPC attack HP=%d, want below %d", after, before)
	}
}

func TestCombatTickHonorsCurrentPlayerBlockState(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 82)
	id := worldcore.EntityID(82)
	state := lifecycle.combatStates[id]
	state.threat = 1
	lifecycle.combatStates[id] = state
	player := newPlayerActor("tester", 0)
	now := time.Unix(2000, 0)
	if !player.setPlayerBlock(true, now) {
		t.Fatal("current block state was not engaged")
	}
	before := player.actor.Snapshot().HP
	result, err := lifecycle.combatTick(player, role.Position{X: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	if after := player.actor.Snapshot().HP; after != before {
		t.Fatalf("parry HP=%d, want %d", after, before)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("parry combat result=%+v actor=%+v", result, player.actor.Snapshot())
	}
}

func TestCombatThreatInterruptsSitcrossAndClearsAfterLeash(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	id := worldcore.EntityID(83)
	state := lifecycle.combatStates[id]
	state.threat = 1
	lifecycle.combatStates[id] = state
	player := newPlayerActor("tester", 0)
	player.actor.SetLogicState(entity.LogicStateSitcross)
	now := time.Unix(2500, 0)
	result, err := lifecycle.combatTick(player, role.Position{X: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("threat state result=%+v actor=%+v", result, player.actor.Snapshot())
	}
	result, err = lifecycle.combatTick(player, role.Position{X: 100}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("leash result=%+v actor=%+v", result, player.actor.Snapshot())
	}
	if lifecycle.combatStates[id].threat != 0 {
		t.Fatalf("leash retained threat=%d", lifecycle.combatStates[id].threat)
	}
}

func TestCombatTickRespawnsDeadHostileAtHome(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	id := worldcore.EntityID(83)
	actor := lifecycle.combatActors[id]
	actor.ApplyDamage(999999)
	state := lifecycle.combatStates[id]
	now := time.Unix(3000, 0)
	state.respawnAt = now
	lifecycle.combatStates[id] = state
	player := newPlayerActor("tester", 0)
	if _, err := lifecycle.combatTick(player, role.Position{X: 999, Z: 999}, now); err != nil {
		t.Fatal(err)
	}
	if got := actor.Snapshot(); got.HP != got.MaxHP || got.LogicState != entity.LogicStateNormal {
		t.Fatalf("respawn actor=%+v", got)
	}
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x1F || frames[1][0] != 0x10 {
		t.Fatalf("current respawn frames=%x", frames)
	}
}

func TestCombatMoveDoesNotImmediatelyConfirmLocation(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 84)
	id := worldcore.EntityID(84)
	state := lifecycle.combatStates[id]
	state.threat = 1
	lifecycle.combatStates[id] = state
	player := newPlayerActor("tester", 0)
	if err := lifecycle.combatMoveTick(player, role.Position{X: 10}, time.Unix(4000, 0)); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x21 {
		t.Fatalf("chase frames=%x, want one current ServerMoving opcode 0x21", frames)
	}
	if !lifecycle.combatStates[id].moving {
		t.Fatal("current combatMoveTick did not mark NPC moving")
	}
}

func TestNPCCombatUsesCurrentCreatorWakeupRangeInsteadOfSpringRange(t *testing.T) {
	npc := testNPCSpawn()
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Extensions["creator.WakeupRange"] = "5.5"
	property := npc.resolved.Properties["SpringRange"]
	property.Value.F32 = 60
	npc.resolved.Properties["SpringRange"] = property
	state := newNPCCombatState(npc, worldcore.Transform{})
	if state.aggroRange != 5.5 {
		t.Fatalf("aggroRange=%v, want creator WakeupRange 5.5; SpringRange must not be used as alert radius", state.aggroRange)
	}
}

func TestCombatTickAcquiresThreatInsideCurrentWakeupRange(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_wakeup"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Extensions["creator.WakeupRange"] = "6"
	npc.x, npc.y, npc.z = 1, 0, 0
	if err := lifecycle.registerNPC(85, npc); err != nil {
		t.Fatal(err)
	}
	lifecycle.combatActive = true
	player := newPlayerActor("tester", 0)
	before := player.actor.Snapshot().HP
	result, err := lifecycle.combatTick(player, role.Position{X: 1}, time.Unix(5000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if lifecycle.combatStates[worldcore.EntityID(85)].threat == 0 {
		t.Fatal("hostile did not acquire threat inside current WakeupRange")
	}
	if !result.playerStateChanged || player.actor.Snapshot().HP >= before {
		t.Fatalf("wakeup aggro did not attack: result=%+v hp=%d before=%d", result, player.actor.Snapshot().HP, before)
	}
}

func TestCombatTickDoesNotAcquireThreatOutsideCurrentWakeupRange(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_wakeup_far"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Extensions["creator.WakeupRange"] = "2"
	npc.x, npc.y, npc.z = 10, 0, 0
	if err := lifecycle.registerNPC(86, npc); err != nil {
		t.Fatal(err)
	}
	lifecycle.combatActive = true
	player := newPlayerActor("tester", 0)
	before := player.actor.Snapshot().HP
	result, err := lifecycle.combatTick(player, role.Position{X: 0}, time.Unix(5000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if lifecycle.combatStates[worldcore.EntityID(86)].threat != 0 {
		t.Fatalf("hostile acquired threat outside WakeupRange: %+v", lifecycle.combatStates[worldcore.EntityID(86)])
	}
	if result.playerStateChanged || player.actor.Snapshot().HP != before {
		t.Fatalf("outside WakeupRange changed player: result=%+v hp=%d before=%d", result, player.actor.Snapshot().HP, before)
	}
}
