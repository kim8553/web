package main

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/qinggong"
)

func TestQingGongRecordSchemaAndRowLayout(t *testing.T) {
	table, err := serverRecordTable(qingGongRecordSchemas)
	if err != nil {
		t.Fatal(err)
	}
	if table[0] != 0x0A || binary.LittleEndian.Uint16(table[1:]) != uint16(len(qingGongRecordSchemas)) || len(qingGongRecordSchemas) != 10 {
		t.Fatalf("record table header=%x schemas=%d", table[:3], len(qingGongRecordSchemas))
	}
	if qingGongRecordSchemas[recordQingGong].name != "QingGongRec" || qingGongRecordSchemas[recordActiveQingGong].name != "ActiveQGSkillRec" {
		t.Fatalf("current qinggong schemas=%+v", qingGongRecordSchemas)
	}
	row := serverRecordAddString(playerObjectID, playerOwnerID, recordQingGong, "qinggong_8")
	if row[0] != 0x11 || binary.LittleEndian.Uint16(row[10:]) != recordQingGong || !strings.Contains(string(row), "qinggong_8") {
		t.Fatalf("record row=%x", row)
	}
}

func TestFunctionUnlockTaskCompletedRecord(t *testing.T) {
	row, mask := taskCompletedBit(functionUnlockTaskID)
	if row != 42 || mask != 0x4 {
		t.Fatalf("task %d mapping = row %d mask 0x%X, want row 42 mask 0x4", functionUnlockTaskID, row, mask)
	}
	frame := serverRecordAddInt32(1, 1, recordTaskCompleted, mask)
	if len(frame) != 20 || frame[0] != 0x11 {
		t.Fatalf("invalid int record frame: % X", frame)
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != recordTaskCompleted {
		t.Fatalf("record index = %d, want %d", got, recordTaskCompleted)
	}
	if got := binary.LittleEndian.Uint32(frame[16:]); got != mask {
		t.Fatalf("record value = 0x%X, want 0x%X", got, mask)
	}
}

func TestFacultyGuideCompletedTaskRows(t *testing.T) {
	taskIDs := []uint32{functionUnlockTaskID, facultyAltTaskID, facultyActAltTaskID}
	for taskID := facultyGuideTaskFirst; taskID <= facultyGuideTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	for taskID := facultyActTaskFirst; taskID <= facultyActTaskLast; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	rows, finalRow := taskCompletedRows(taskIDs)
	if finalRow != 1593 {
		t.Fatalf("final completed-task row = %d, want 1593", finalRow)
	}
	// 1069-1073 occupy bits 13-17 in row 33.  1344 and 1346 occupy
	// bits 0 and 2 in row 42.
	if got := rows[33]; got != 0x0003E000 {
		t.Fatalf("guide row 33 = 0x%08X, want 0x0003E000", got)
	}
	if got := rows[42]; got != 0x00000005 {
		t.Fatalf("unlock row 42 = 0x%08X, want 0x00000005", got)
	}
	// The complete 23613 branch comprises tasks 1140--1147 (row 35) plus
	// 51002 (row 1593); together they gate the faculty-panel 演武 button.
	if got := rows[35]; got != 0x0FF00000 {
		t.Fatalf("activity-faculty row 35 = 0x%08X, want 0x0FF00000", got)
	}
	if got := rows[1593]; got != 0x04000000 {
		t.Fatalf("activity-faculty row 1593 = 0x%08X, want 0x04000000", got)
	}
}

func TestRecordTableExcludesUnrecoveredReputeRecord(t *testing.T) {
	for _, schema := range qingGongRecordSchemas {
		if schema.name == "Repute_Record" {
			t.Fatalf("unrecovered condition record leaked into current schema: %s", schema.name)
		}
	}
}

func TestQingGongWaterBuffsUseSeparateNativeSlots(t *testing.T) {
	player := newPlayerActor("测试", 0)
	drift := qinggong.Definition{ID: "qinggong_2", BufferID: "buf_qg_drift"}
	jump := qinggong.Definition{ID: "qinggong_3", BufferID: "buf_qg_dst"}
	conn := &captureMessageConnection{}
	now := time.Now()
	already, expires, err := activateQingGongBuff(player, drift, now)
	if err != nil || already || expires.IsZero() {
		t.Fatalf("drift already=%t exp=%v err=%v", already, expires, err)
	}
	ident := sceneIdent(playerObjectID, playerOwnerID)
	if got := player.bufferListString(now); !strings.Contains(got, "buf_qg_drift,local-2,1,"+ident+",") {
		t.Fatalf("drift=%q", got)
	}
	already, _, err = activateQingGongBuff(player, drift, now)
	if err != nil || !already {
		t.Fatalf("duplicate=%t err=%v", already, err)
	}
	if err := endQingGongBuff(conn, player, drift); err != nil || len(conn.frames) != 1 {
		t.Fatalf("end err=%v frames=%x", err, conn.frames)
	}
	conn = &captureMessageConnection{}
	already, expires, err = activateQingGongBuff(player, jump, now)
	if err != nil || already || expires.IsZero() {
		t.Fatalf("jump already=%t exp=%v err=%v", already, expires, err)
	}
	if got := player.bufferListString(now); !strings.Contains(got, "buf_qg_dst,local-3,1,"+ident+",") {
		t.Fatalf("jump=%q", got)
	}
}

func TestRedSuperArmorRulesMatchPropPack9679(t *testing.T) {
	player := newPlayerActor("测试", 0)
	now := time.Now()
	if rules := player.buffRules(now); rules.CantHitEffect || rules.CantBeSkillLocked {
		t.Fatalf("inactive red rules=%+v", rules)
	}
	player.activateRedSuperArmor(now)
	if rules := player.buffRules(now); !rules.CantHitEffect || !rules.CantBeSkillLocked {
		t.Fatalf("active red rules=%+v", rules)
	}
}

func TestActivateQingGongWritesActiveSkillRecord(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	changed, err := activateQingGongRecord(conn, player, "QG_JH_gj_004")
	if err != nil || !changed || len(conn.frames) != 1 {
		t.Fatalf("activate changed=%t err=%v frames=%d", changed, err, len(conn.frames))
	}
	frame := conn.frames[0]
	if frame[0] != 0x11 || binary.LittleEndian.Uint16(frame[10:]) != recordActiveQingGong || !strings.Contains(string(frame), "QG_JH_gj_004") {
		t.Fatalf("active qinggong record frame=%x", frame)
	}
	changed, err = activateQingGongRecord(conn, player, "QG_JH_gj_004")
	if err != nil || changed || len(conn.frames) != 1 {
		t.Fatalf("duplicate activation changed=%t err=%v frames=%d", changed, err, len(conn.frames))
	}
}

func TestGrantStarterQingGongUsesRecordAndView(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := grantStarterQingGong(conn, "测试", nil); err != nil {
		t.Fatal(err)
	}
	if len(conn.frames) <= len(starterQingGongIDs)+1 {
		t.Fatalf("too few grant frames=%d", len(conn.frames))
	}
	create := conn.frames[len(starterQingGongIDs)]
	first := conn.frames[len(starterQingGongIDs)+1]
	if create[0] != 0x15 || first[0] != 0x18 {
		t.Fatalf("grant sequence create=%x first=%x", create, first)
	}
	if got := binary.LittleEndian.Uint16(first[7:]); got != 7 {
		t.Fatalf("QingGong ConfigID wire ordinal=%#x want=0x7", got)
	}
	if !strings.Contains(string(first), starterQingGongIDs[0]) {
		t.Fatalf("first QingGong row=%x", first)
	}
}

func TestGrantRedSuperArmorUsesNativeBufferInfoProperty(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	if _, err := grantRedSuperArmor(conn, player); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 1 {
		t.Fatalf("frames=%d", len(frames))
	}
	frame := frames[0]
	if frame[0] != 0x10 || binary.LittleEndian.Uint16(frame[10:]) < 1 {
		t.Fatalf("header=%x", frame[:12])
	}
	if !strings.Contains(string(frame), "21EA,11000001,7A5C0B,1,") {
		t.Fatalf("current native red BufferInfo missing: %q", string(frame))
	}
	if got := player.bufferInfo(redSuperArmorBuffSlot, time.Now()); got == "" {
		t.Fatal("red super armor BuffInfo slot was not staged")
	}
}
