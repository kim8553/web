package main

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/entity"
)

func shortcutCustom(messageID int32, values ...clientCustomValue) clientCustomMessage {
	return clientCustomMessage{
		Opcode: 0x0A,
		Values: append([]clientCustomValue{{Type: 2, Int32: messageID}}, values...),
	}
}

func TestSetShortcutAddsAndReplacesRecordRow(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	set := shortcutCustom(clientCustomSetShortcut,
		clientCustomValue{Type: 2, Int32: 3},
		clientCustomValue{Type: 6, Text: "skill"},
		clientCustomValue{Type: 6, Text: "CS_yhwq_hsqs01"},
	)
	handled, err := handleShortcutCustom(conn, player, set, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 1 {
		t.Fatalf("first set handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	add := conn.frames[0]
	if add[0] != 0x11 || binary.LittleEndian.Uint16(add[10:]) != recordShortcut ||
		binary.LittleEndian.Uint32(add[16:]) != 3 || !strings.Contains(string(add), "CS_yhwq_hsqs01") {
		t.Fatalf("shortcut add=%x", add)
	}

	set.Values[3].Text = "CS_yhwq_hsqs02"
	handled, err = handleShortcutCustom(conn, player, set, "test", nil, 0)
	if err != nil || !handled || len(conn.frames) != 3 {
		t.Fatalf("replace handled=%t err=%v frames=%d", handled, err, len(conn.frames))
	}
	if conn.frames[1][0] != 0x12 || binary.LittleEndian.Uint16(conn.frames[1][10:]) != recordShortcut ||
		binary.LittleEndian.Uint16(conn.frames[1][12:]) != 0 {
		t.Fatalf("shortcut delete=%x", conn.frames[1])
	}
	if conn.frames[2][0] != 0x11 || !strings.Contains(string(conn.frames[2]), "CS_yhwq_hsqs02") {
		t.Fatalf("replacement add=%x", conn.frames[2])
	}
}

func TestRemoveShortcutUsesRecordRow(t *testing.T) {
	player := newPlayerActor("测试", 0)
	player.setShortcut(8, "skill", "CS_yhwq_hsqs01")
	player.setShortcut(9, "skill", "CS_yhwq_hsqs02")
	seed := &captureMessageConnection{}
	if err := player.resyncShortcutRecord(seed); err != nil {
		t.Fatal(err)
	}
	conn := &captureMessageConnection{}
	remove := shortcutCustom(clientCustomRemoveShortcut, clientCustomValue{Type: 2, Int32: 9})
	handled, err := handleShortcutCustom(conn, player, remove, "test", nil, 0)
	if err != nil || !handled {
		t.Fatalf("remove handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 3 || frames[0][0] != 0x12 || frames[1][0] != 0x12 || frames[2][0] != 0x11 {
		t.Fatalf("current record resync frames=%x", frames)
	}
	if binary.LittleEndian.Uint16(frames[0][10:]) != recordShortcut || binary.LittleEndian.Uint16(frames[0][12:]) != 1 ||
		binary.LittleEndian.Uint16(frames[1][10:]) != recordShortcut || binary.LittleEndian.Uint16(frames[1][12:]) != 0 {
		t.Fatalf("shortcut delete rows=%x %x", frames[0], frames[1])
	}
	snapshot := player.shortcutSnapshot()
	if len(snapshot) != 1 || snapshot[0].index != 8 {
		t.Fatalf("remaining=%+v", snapshot)
	}
}

func TestGrantShortcutRowsRestoresCurrentRows(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.setShortcut(1, "skill", "CS_yhwq_hsqs01")
	player.setShortcut(2, "skill", "CS_yhwq_hsqs02")
	if err := grantShortcutRows(conn, player); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) != 2 || conn.frames[0][0] != 0x11 || conn.frames[1][0] != 0x11 {
		t.Fatalf("restore frames=%x", conn.frames)
	}
}

func TestGrantSitcrossShortcutAddsLearnedNativeSkillWithoutOverwritingSlot(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	granted, err := grantSitcrossShortcut(conn, player)
	if err != nil || !granted || len(conn.frames) != 1 {
		t.Fatalf("first grant granted=%t err=%v frames=%d", granted, err, len(conn.frames))
	}
	if conn.frames[0][0] != 0x11 || !strings.Contains(string(conn.frames[0]), "zs_default_01") || !strings.Contains(string(conn.frames[0]), "skill") {
		t.Fatalf("sitcross add=%x", conn.frames[0])
	}
	granted, err = grantSitcrossShortcut(conn, player)
	if err != nil || granted || len(conn.frames) != 1 {
		t.Fatalf("second grant granted=%t err=%v frames=%d", granted, err, len(conn.frames))
	}

	other := newPlayerActor("占位", 0)
	other.setShortcut(sitcrossShortcutIndex, "skill", "CS_yhwq_hsqs01")
	granted, err = grantSitcrossShortcut(&captureMessageConnection{}, other)
	if err != nil || granted {
		t.Fatalf("occupied slot grant granted=%t err=%v", granted, err)
	}
}

func TestSitcrossUsesCapturedStartArgumentAndNormalPropertyUpdate(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	start := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 1})
	handled, err := handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled {
		t.Fatalf("start handled=%t err=%v", handled, err)
	}
	if got := player.actor.Snapshot().LogicState; got != entity.LogicStateSitcross {
		t.Fatalf("LogicState=%d, want sitcross=%d", got, entity.LogicStateSitcross)
	}
	if frames := conn.Frames(); len(frames) != 1 || frames[0][0] != 0x10 {
		t.Fatalf("expected one normal 0x10 property update, got %x", frames)
	}

	// Holding the key yields repeated 240,1 packets.  It must stay in state
	// 102 instead of behaving as a toggle.
	handled, err = handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled || len(conn.Frames()) != 1 {
		t.Fatalf("duplicate handled=%t err=%v frames=%d", handled, err, len(conn.Frames()))
	}

	stop := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 0})
	handled, err = handleSitcrossCustom(conn, player, stop, "test")
	if err != nil || !handled || player.actor.Snapshot().LogicState != entity.LogicStateNormal || len(conn.Frames()) != 2 {
		t.Fatalf("stop handled=%t err=%v state=%d frames=%d", handled, err, player.actor.Snapshot().LogicState, len(conn.Frames()))
	}

	// Live client capture: a hostile interruption emits 240,2 rather than the
	// older cancellation value 0. It must stop sitting without closing the
	// connection.
	handled, err = handleSitcrossCustom(conn, player, start, "test")
	if err != nil || !handled || player.actor.Snapshot().LogicState != entity.LogicStateSitcross {
		t.Fatalf("second start handled=%t err=%v state=%d", handled, err, player.actor.Snapshot().LogicState)
	}
	interrupted := shortcutCustom(clientCustomSitCross, clientCustomValue{Type: 2, Int32: 2})
	handled, err = handleSitcrossCustom(conn, player, interrupted, "test")
	if err != nil || !handled || player.actor.Snapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("interruption handled=%t err=%v state=%d", handled, err, player.actor.Snapshot().LogicState)
	}
}

func TestSitcrossTickRestoresAuthoritativeHPAndMP(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.actor.Snapshot()
	player.actor.ApplyDamage(100)
	player.actor.SpendMP(100)
	player.actor.SetLogicState(entity.LogicStateSitcross)
	if !player.sitcrossTick() {
		t.Fatal("sitcross tick unchanged")
	}
	state := player.actor.Snapshot()
	wantHP := minInt32(before.MaxHP, before.HP-100+sitcrossRecoveryPerSecond(before.MaxHP))
	wantMP := minInt32(before.MaxMP, before.MP-100+sitcrossRecoveryPerSecond(before.MaxMP))
	if state.HP != wantHP || state.MP != wantMP {
		t.Fatalf("state=%+v want=%d/%d", state, wantHP, wantMP)
	}
	rates := map[uint16]int32{}
	for _, p := range player.stateProperties() {
		if p.Index >= 615 && p.Index <= 622 {
			rates[p.Index] = p.Value.I32
		}
	}
	if rates[617] != sitcrossHeartMillis || rates[618] != sitcrossHeartMillis || rates[615] != sitcrossRecoveryPerSecond(before.MaxHP) || rates[616] != sitcrossRecoveryPerSecond(before.MaxMP) {
		t.Fatalf("current heart rates=%v", rates)
	}
}

func TestSitcrossFullyRestoresAnyDerivedPoolWithinTenTicks(t *testing.T) {
	player := newPlayerActor("测试", 0)
	before := player.actor.Snapshot()
	player.actor.ApplyDamage(before.MaxHP - 1)
	player.actor.SpendMP(before.MaxMP - 1)
	player.actor.SetLogicState(entity.LogicStateSitcross)
	for tick := 0; tick < int(sitcrossFullRecoverySeconds); tick++ {
		player.sitcrossTick()
	}
	state := player.actor.Snapshot()
	if state.HP != before.MaxHP || state.MP != before.MaxMP {
		t.Fatalf("ten-second recovery HP=%d/%d MP=%d/%d", state.HP, before.MaxHP, state.MP, before.MaxMP)
	}
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func TestCombatBruiseTickDoesNotRestoreRealHP(t *testing.T) {
	player := newPlayerActor("测试", 0)
	player.actor.ApplyDamage(100)
	before := player.actor.Snapshot()
	if changed := player.combatBruiseTick(); !changed {
		t.Fatal("combat tick did not settle bruised blood")
	}
	state := player.actor.Snapshot()
	if state.HP != before.HP || state.HitHP != before.HitHP-combatBruiseRecoveryPerSecond {
		t.Fatalf("combat tick HP=%d HitHP=%d, want %d/%d", state.HP, state.HitHP, before.HP, before.HitHP-combatBruiseRecoveryPerSecond)
	}
}
