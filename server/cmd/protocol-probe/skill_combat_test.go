package main

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"time"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

func useSkillMessage(id string) clientCustomMessage {
	return clientCustomMessage{Opcode: 0x0A, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomUseSkill},
		{Type: 6, Text: id},
	}}
}

func useSkillMessageAt(id string, x, y, z float32) clientCustomMessage {
	message := useSkillMessage(id)
	message.Values = append(message.Values,
		clientCustomValue{Type: 4, Float32: x},
		clientCustomValue{Type: 4, Float32: y},
		clientCustomValue{Type: 4, Float32: z},
	)
	return message
}

func TestCombatSkillRequestDefinitionAcceptsIndexedConditionZeroReplacement(t *testing.T) {
	catalog := &combatSkillCatalog{
		noConditionReplacementBase: map[string]string{"cs_hidden": "CS_base"},
	}
	installed := map[string]combatSkillDefinition{
		"CS_normal": {id: "CS_normal", level: 1},
	}
	if got, ok := combatSkillRequestDefinition("CS_normal", catalog, installed); !ok || got.id != "CS_normal" {
		t.Fatalf("installed request got=%+v ok=%t", got, ok)
	}
	if got, ok := combatSkillRequestDefinition("CS_hidden", catalog, installed); !ok || got.id != "CS_hidden" {
		t.Fatalf("condition-zero replacement request got=%+v ok=%t", got, ok)
	}
	if _, ok := combatSkillRequestDefinition("CS_unknown", catalog, installed); ok {
		t.Fatal("unknown skill request was accepted")
	}
}

func TestRequestedSkillAuthorityInheritsConditionZeroReplacementBaseLevel(t *testing.T) {
	player := newPlayerActor("tester", 0)
	learnAuthoritySkill(player, "CS_base", 3)
	catalog := &combatSkillCatalog{
		noConditionReplacementBase: map[string]string{"cs_hidden": "CS_base"},
	}
	level, baseID, learned := requestedSkillAuthority(player, "CS_hidden", catalog)
	if !learned || level != 3 || baseID != "CS_base" {
		t.Fatalf("replacement authority learned=%t level=%d base=%q", learned, level, baseID)
	}
}

func TestRequestedSkillAuthorityPrefersReplacementOwnLearnedLevel(t *testing.T) {
	player := newPlayerActor("tester", 0)
	learnAuthoritySkill(player, "CS_base", 3)
	learnAuthoritySkill(player, "CS_hidden", 2)
	catalog := &combatSkillCatalog{
		noConditionReplacementBase: map[string]string{"cs_hidden": "CS_base"},
	}
	level, baseID, learned := requestedSkillAuthority(player, "CS_hidden", catalog)
	if !learned || level != 2 || baseID != "" {
		t.Fatalf("replacement own authority learned=%t level=%d base=%q", learned, level, baseID)
	}
}

func TestHandleSelfSkillSpendsMPAndStartsCooldown(t *testing.T) {
	player := newPlayerActor("tester", 0)
	learnAuthoritySkill(player, "CS_yhwq_hsqs05", 1)
	beforeMP := player.actor.Snapshot().MP
	conn := &captureMessageConnection{}
	handled, err := handleSkillCustom(conn, player, nil, useSkillMessage("CS_yhwq_hsqs05"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("skill 211 was not handled")
	}
	if got := player.actor.Snapshot().MP; got != beforeMP-11 {
		t.Fatalf("MP=%d, want %d", got, beforeMP-11)
	}
	frames := conn.Frames()
	if len(frames) < 3 || frames[0][0] != 0x11 {
		t.Fatalf("current self-skill frames=%x", frames)
	}
	if got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordCooldown {
		t.Fatalf("cooldown record index=%d, want %d", got, recordCooldown)
	}
	hasVital, hasAction := false, false
	for _, frame := range frames {
		if len(frame) == 0 {
			continue
		}
		if frame[0] == 0x10 {
			hasVital = true
		}
		if frame[0] == 0x1E {
			hasAction = true
		}
	}
	if !hasVital || !hasAction {
		t.Fatalf("current self-skill route missing vital/action: %x", frames)
	}
	if got := player.currentSkillSnapshot(); got != "CS_yhwq_hsqs05" {
		t.Fatalf("CurSkillID=%q", got)
	}
	_, effectID, level, target := player.currentSkillStateSnapshot()
	if effectID != "CS_yhwq_hsqs05" || level != 1 {
		t.Fatalf("skill effect state=%q level=%d", effectID, level)
	}
	if want := uint64(playerObjectID) | uint64(playerOwnerID)<<32; target != want {
		t.Fatalf("CurSkillTarget=%x want=%x", target, want)
	}
	if got := player.bufferInfo(yiHuaSkill05BuffSlot, time.Now()); got == "" {
		t.Fatal("fifth self skill did not stage buf_CS_yhwq_hsqs05")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_CS_yhwq_hsqs05") || !strings.Contains(got, "buf_fixed_red") {
		t.Fatalf("BufferListStr=%q", got)
	}
	if rules := player.buffRules(time.Now()); !rules.CantHitEffect || !rules.CantBeSkillLocked {
		t.Fatalf("red super armor rules=%+v", rules)
	}
	beforeFrames := len(frames)
	handled, err = handleSkillCustom(conn, player, nil, useSkillMessage("CS_yhwq_hsqs05"), "test")
	if err != nil || !handled {
		t.Fatalf("second handle handled=%t err=%v", handled, err)
	}
	if got := player.actor.Snapshot().MP; got != beforeMP-11 {
		t.Fatalf("cooldown rejection still spent MP: %d", got)
	}
	if got := len(conn.Frames()); got != beforeFrames {
		t.Fatalf("cooldown rejection emitted frames: before=%d after=%d", beforeFrames, got)
	}
}

func TestCurrentSkillClearRejectsStaleTimer(t *testing.T) {
	player := newPlayerActor("tester", 0)
	player.currentSkillID = "CS_yhwq_hsqs05"
	if player.clearCurrentSkill("CS_yhwq_hsqs04") {
		t.Fatal("stale skill timer cleared a later skill")
	}
	if got := player.currentSkillSnapshot(); got != "CS_yhwq_hsqs05" {
		t.Fatalf("CurSkillID=%q after stale timer", got)
	}
	if !player.clearCurrentSkill("CS_yhwq_hsqs05") {
		t.Fatal("matching skill timer did not clear current skill")
	}
	if got := player.currentSkillSnapshot(); got != "" {
		t.Fatalf("CurSkillID=%q after matching timer", got)
	}
}

func TestHandleSkillFourStagesNativeVisibleBuff(t *testing.T) {
	player := newPlayerActor("tester", 0)
	learnAuthoritySkill(player, "CS_yhwq_hsqs04", 1)
	conn := &captureMessageConnection{}
	handled, err := handleSkillCustom(conn, player, nil, useSkillMessage("CS_yhwq_hsqs04"), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.bufferInfo(yiHuaSkillBuffSlot, time.Now()); got == "" {
		t.Fatal("self skill did not stage native visible Buff")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_CS_yhwq_hsqs04") {
		t.Fatalf("BufferListStr=%q", got)
	}
	frames := conn.Frames()
	if len(frames) < 3 || frames[0][0] != 0x11 {
		t.Fatalf("current skill-four frames=%x", frames)
	}
	if got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordCooldown {
		t.Fatalf("cooldown record index=%d want=%d", got, recordCooldown)
	}
	hasVital, hasAction := false, false
	for _, frame := range frames {
		if len(frame) == 0 {
			continue
		}
		if frame[0] == 0x10 {
			hasVital = true
		}
		if frame[0] == 0x1E {
			hasAction = true
		}
	}
	if !hasVital || !hasAction {
		t.Fatalf("current skill-four route missing vital/action: %x", frames)
	}
}

func TestNativeSkillActionFrameUsesModernMessage20(t *testing.T) {
	frame, err := nativeSkillActionFrame("CS_yhwq_hsqs05", playerObjectID, playerOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 56 || frame[0] != 0x1E {
		t.Fatalf("native skill action frame=%x", frame)
	}
	if got := binary.LittleEndian.Uint16(frame[1:]); got != 6 {
		t.Fatalf("TVarList count=%d, want 6", got)
	}
	if frame[3] != 2 || int32(binary.LittleEndian.Uint32(frame[4:])) != serverNativeSkillActionMessage {
		t.Fatalf("native message prefix=%x, want int32 message %d", frame[3:8], serverNativeSkillActionMessage)
	}
	if frame[8] != 8 || binary.LittleEndian.Uint32(frame[9:]) != playerObjectID || binary.LittleEndian.Uint32(frame[13:]) != playerOwnerID {
		t.Fatalf("caster object=%x, want %d-%d", frame[8:17], playerObjectID, playerOwnerID)
	}
	if frame[17] != 6 || binary.LittleEndian.Uint32(frame[18:]) != 15 || string(frame[22:36]) != "CS_yhwq_hsqs05" || frame[36] != 0 {
		t.Fatalf("skill action string=%x", frame[17:37])
	}
	if frame[37] != 8 || binary.LittleEndian.Uint32(frame[38:]) != playerObjectID || binary.LittleEndian.Uint32(frame[42:]) != playerOwnerID {
		t.Fatalf("target object=%x, want %d-%d", frame[37:46], playerObjectID, playerOwnerID)
	}
	if frame[46] != 2 || binary.LittleEndian.Uint32(frame[47:]) != 1 || frame[51] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(frame[52:])) != 1 {
		t.Fatalf("optional action fields=%x, want int:1 float:1", frame[46:])
	}
}

func TestStandaloneActionProbeFrames(t *testing.T) {
	stateFrame, err := playerActionStateFrame("interact268")
	if err != nil {
		t.Fatal(err)
	}
	if len(stateFrame) < 16 || stateFrame[0] != 0x10 {
		t.Fatalf("State action frame=%x", stateFrame)
	}

	nativeFrame, err := nativeGenericActionFrame("interact268")
	if err != nil {
		t.Fatal(err)
	}
	if nativeFrame[0] != 0x27 || binary.LittleEndian.Uint16(nativeFrame[1:]) != 4 {
		t.Fatalf("native generic action frame=%x", nativeFrame)
	}
	if nativeFrame[3] != 6 || binary.LittleEndian.Uint32(nativeFrame[4:]) != 7 ||
		string(nativeFrame[8:14]) != "action" || nativeFrame[14] != 0 {
		t.Fatalf("native generic action message=%x, want string action", nativeFrame[3:15])
	}
	if nativeFrame[15] != 8 ||
		binary.LittleEndian.Uint32(nativeFrame[16:]) != playerObjectID ||
		binary.LittleEndian.Uint32(nativeFrame[20:]) != playerOwnerID {
		t.Fatalf("native generic action object=%x", nativeFrame[15:24])
	}
}

func TestFifthSkillTotalActionIncludesFollowupOffset(t *testing.T) {
	definition := yiHuaWuQueCombatSkills["CS_yhwq_hsqs05"]
	wantFollowup := 31*time.Second/30 - time.Duration(0.24/30*float64(time.Second))
	if got := definition.followupActions[0].at; got != wantFollowup {
		t.Fatalf("fifth skill followup=%s, want %s", got, wantFollowup)
	}
	want := wantFollowup + 113*time.Second/30
	if got := definition.totalActionDuration(); got != want {
		t.Fatalf("total action duration=%s, want %s", got, want)
	}
}

func TestTaiJiQuanGuPuMetadataAndThreeSegmentAction(t *testing.T) {
	if len(taiJiQuanGuPuCombatSkills) != 8 {
		t.Fatalf("tai ji skill count=%d, want 8", len(taiJiQuanGuPuCombatSkills))
	}
	definition := taiJiQuanGuPuCombatSkills["CS_wd_tjq08"]
	if definition.spCost != 50 || definition.baseDamage != 1304 || definition.personalCD != 10*time.Second || definition.publicCD != 10*time.Second || definition.targetMode != skillTargetSelf || definition.areaRadius != 0 || definition.range_ != 5 {
		t.Fatalf("开太极 metadata=%+v", definition)
	}
	// Mirror current parser float64->Duration truncation of the exact resource text.
	wantSecond := time.Duration(4_645_416_667)
	wantThird := time.Duration(6_808_083_333)
	if len(definition.followupActions) != 2 || definition.followupActions[0].action != "Test0001" || definition.followupActions[0].at != wantSecond || definition.followupActions[0].duration != time.Duration(2_166_666_666) || definition.followupActions[1].action != "fight_0h_jb_10" || definition.followupActions[1].at != wantThird || definition.followupActions[1].duration != time.Duration(1_933_333_333) {
		t.Fatalf("开太极 action sequence=%+v", definition.followupActions)
	}
	want := wantThird + time.Duration(1_933_333_333)
	if got := definition.totalActionDuration(); got != want {
		t.Fatalf("开太极 total action=%s, want %s", got, want)
	}
	if definition.requiresTarget {
		t.Fatal("current 开太极 resource contract is self-target, not selected-target")
	}
}

func TestOpenTaiJiStartsWithoutSelectedTargetAndStagesActionBoundRedArmor(t *testing.T) {
	player := newPlayerActor("tester", 0)
	learnAuthoritySkill(player, "CS_wd_tjq08", 1)
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	handled, err := handleSkillCustom(conn, player, world, useSkillMessage("CS_wd_tjq08"), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.skillSP(); got != 50 {
		t.Fatalf("SP=%d, want 50 after 开太极", got)
	}
	if got := player.bufferInfo(redSuperArmorBuffSlot, time.Now()); got == "" {
		t.Fatal("开太极 did not stage action-bound red super armor")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_fixed_red") {
		t.Fatalf("BufferListStr=%q, want buf_fixed_red", got)
	}
	frames := conn.Frames()
	if len(frames) < 5 || frames[0][0] != 0x11 || frames[1][0] != 0x10 {
		t.Fatalf("开太极 initial frames=%x", frames)
	}
	if frames[len(frames)-1][0] != 0x1E {
		t.Fatalf("开太极 current native action frame=%x", frames[len(frames)-1])
	}
}

func TestOpenTaiJiDamagesCombatNPCsInsideInstalledFiveMetreCircle(t *testing.T) {
	definition := taiJiQuanGuPuCombatSkills["CS_wd_tjq08"]
	if definition.targetMode != skillTargetSelf || definition.areaRadius != 0 {
		t.Fatalf("current 开太极 target contract=%+v", definition)
	}
	player := newPlayerActor("tester", 0)
	learnAuthoritySkill(player, "CS_wd_tjq08", 1)
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	insideBefore := world.combatActors[91].Snapshot().HP
	handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt("CS_wd_tjq08", 0, 0, 0), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := world.combatActors[91].Snapshot().HP; got != insideBefore {
		t.Fatalf("current self-target 开太极 changed NPC HP=%d want=%d", got, insideBefore)
	}
}

func TestCompiledSectorDamageUsesC2S211CasterOrientation(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 93)
	registerTestCombatNPC(t, world, 94)
	front := world.entities[93]
	front.transform.X = 0
	front.transform.Z = 4
	world.entities[93] = front
	back := world.entities[94]
	back.transform.X = 0
	back.transform.Z = -4
	world.entities[94] = back
	results, err := world.applyShapedSkillDamage(worldcore.Transform{Orient: 0}, 5, 5, 120, 0, 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].id != 93 {
		t.Fatalf("sector results=%+v, want +Z-facing NPC 93", results)
	}
	if got := world.combatActors[93].Snapshot().HP; got != 900 {
		t.Fatalf("front NPC HP=%d, want 900", got)
	}
	if got := world.combatActors[94].Snapshot().HP; got != 1000 {
		t.Fatalf("back NPC HP=%d, want untouched 1000", got)
	}
}

func TestCompiledRectangleDamageUsesLengthWidthAndOrientation(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint32{95, 96, 97} {
		registerTestCombatNPC(t, world, id)
	}
	inside := world.entities[95]
	inside.transform.X, inside.transform.Z = 1, 8
	world.entities[95] = inside
	tooWide := world.entities[96]
	tooWide.transform.X, tooWide.transform.Z = 3, 8
	world.entities[96] = tooWide
	behind := world.entities[97]
	behind.transform.X, behind.transform.Z = 0, -2
	world.entities[97] = behind
	results, err := world.applyShapedSkillDamage(worldcore.Transform{Orient: 0}, 10, 5, 0, 4, 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].id != 95 {
		t.Fatalf("rectangle results=%+v, want +Z-forward NPC 95", results)
	}
}

func TestTaiJiStanceAlternatesAndTimedBuffsUseInstalledRecords(t *testing.T) {
	player := newPlayerActor("tester", 0)
	target := uint64(playerObjectID) | uint64(playerOwnerID)<<32
	now := time.Now()
	definition := taiJiQuanGuPuCombatSkills["CS_wd_tjq04"]
	if ok, reason := player.beginSkillUse(definition, target, now); !ok {
		t.Fatalf("begin 金鸡独立: %s", reason)
	}
	player.activateNamedBuff(taiJiStanceBuffSlot, taiJiDefenceStanceBuffStaticData, "buf_CS_wd_tjq04_1", taiJiStanceBuffLease, now)
	if !player.hasActiveBuff(taiJiStanceBuffSlot, taiJiDefenceStanceBuffStaticData, now) {
		t.Fatal("defence stance not active")
	}
	player.activateNamedBuff(taiJiStanceBuffSlot, taiJiAttackStanceBuffStaticData, "buf_CS_wd_tjq04_2", taiJiStanceBuffLease, now)
	if !player.hasActiveBuff(taiJiStanceBuffSlot, taiJiAttackStanceBuffStaticData, now) {
		t.Fatal("attack stance did not replace defence stance")
	}
}

func TestSkillCompletionClearsStateAndRestoresStand(t *testing.T) {
	player := newPlayerActor("tester", 0)
	player.currentSkillID = "test_skill"
	player.currentSkillEffectID = "test_skill"
	definition := combatSkillDefinition{id: "test_skill", actionDuration: time.Millisecond}
	conn := &captureMessageConnection{}

	scheduleCurrentSkillClear(conn, player, definition, nil)
	deadline := time.Now().Add(time.Second)
	for len(conn.Frames()) < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x10 || frames[1][0] != 0x10 {
		t.Fatalf("completion frames=%d, want CurSkill clear then State stand", len(frames))
	}
	if got := player.currentSkillSnapshot(); got != "" {
		t.Fatalf("CurSkillID=%q after completion", got)
	}
}

func TestSkillFourCooldownRecordMatchesInstalledStaticData(t *testing.T) {
	definition := yiHuaWuQueCombatSkills["CS_yhwq_hsqs04"]
	start := time.UnixMilli(123456)
	frame, err := skillCooldownFrame(definition, start)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 40 || frame[0] != 0x11 {
		t.Fatalf("cooldown frame=%x", frame)
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != recordCooldown {
		t.Fatalf("record index=%d, want %d", got, recordCooldown)
	}
	if got := binary.LittleEndian.Uint32(frame[16:]); got != 0 {
		t.Fatalf("current category=%d, want 0", got)
	}
	begin := int64(binary.LittleEndian.Uint64(frame[20:]))
	end := int64(binary.LittleEndian.Uint64(frame[28:]))
	if begin != start.UnixMilli() || end-begin != 1000 {
		t.Fatalf("cooldown begin=%d end=%d", begin, end)
	}
	if got := binary.LittleEndian.Uint32(frame[36:]); got != 0 {
		t.Fatalf("current team=%d, want 0", got)
	}
}

func TestSkillSevenSpendsSP(t *testing.T) {
	player := newPlayerActor("tester", 0)
	definition := yiHuaWuQueCombatSkills["CS_yhwq_hsqs07"]
	definition.requiresTarget = false
	ok, reason := player.beginSkillUse(definition, uint64(playerObjectID)|uint64(playerOwnerID)<<32, time.Now())
	if !ok {
		t.Fatalf("begin skill seven: %s", reason)
	}
	if got := player.progress.attributes.sp; got != 50 {
		t.Fatalf("SP=%d, want 50", got)
	}
}

func TestAttackSkillRequiresSelectedTarget(t *testing.T) {
	player := newPlayerActor("tester", 0)
	beforeMP := player.actor.Snapshot().MP
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	handled, err := handleSkillCustom(conn, player, world, useSkillMessage("CS_yhwq_hsqs01"), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.actor.Snapshot().MP; got != beforeMP {
		t.Fatalf("target rejection spent MP: %d", got)
	}
	if len(conn.frames) != 0 {
		t.Fatalf("target rejection emitted %d frames", len(conn.frames))
	}
}
