package main

import (
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/qinggong"
	"log"
	"time"
)

type recordSchema struct {
	name     string
	colTypes []byte
}
type qinggongBuffSpec struct {
	slot       uint16
	staticData uint32
	duration   time.Duration
	clearOnEnd bool
}

func qinggongBuffFor(definition qinggong.Definition) (qinggongBuffSpec, bool) {
	switch definition.BufferID {
	case "buf_qg_dst":
		return qinggongBuffSpec{slot: 3, staticData: 5786, duration: 3 * time.Second}, true
	case "buf_qg_fast":
		return qinggongBuffSpec{slot: 13, staticData: 217, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_climb":
		return qinggongBuffSpec{slot: 12, staticData: 219, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_drift":
		return qinggongBuffSpec{slot: 2, staticData: 216, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_light":
		return qinggongBuffSpec{slot: 14, staticData: 218, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_DHadd_2rd":
		return qinggongBuffSpec{slot: 18, staticData: 1578, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_DHadd_3rd":
		return qinggongBuffSpec{slot: 19, staticData: 1579, duration: 10 * time.Minute, clearOnEnd: true}, true
	case "buf_qg_airrush":
		return qinggongBuffSpec{slot: 17, staticData: 6894, duration: 10 * time.Minute, clearOnEnd: true}, true
	default:
		return qinggongBuffSpec{}, false
	}
}

const (
	recordColumnInt               = byte(3)
	recordColumnInt64             = byte(4)
	recordColumnString            = byte(7)
	recordQingGong         uint16 = 0
	recordActiveQingGong   uint16 = 1
	recordTaskAccepted     uint16 = 2
	recordTaskCompleted    uint16 = 3
	recordShortcut         uint16 = 4
	recordCooldown         uint16 = 5
	viewportQingGong       uint16 = 46
	qingGongConfigProperty uint16 = 7
)

var qingGongRecordSchemas = []recordSchema{{name: "QingGongRec", colTypes: []byte{7}}, {name: "ActiveQGSkillRec", colTypes: []byte{7}}, {name: "Task_Accepted", colTypes: []byte{3, 3, 3, 3, 7, 7, 3}}, {name: "Task_Completed", colTypes: []byte{3}}, {name: "shortcut_rec", colTypes: []byte{3, 7, 7}}, {name: "cooldown_rec", colTypes: []byte{3, 4, 4, 3}}, {name: "active_jingmai_rec", colTypes: []byte{7}}, {name: "CollectSkillRec", colTypes: []byte{7, 3}}, {name: "CardRec", colTypes: []byte{3}}, {name: "UseCardRec", colTypes: []byte{3, 3}}}
var starterQingGongIDs = []string{"qinggong_1"}
var starterQingGongSkillIDs = []string{"QG_JH_001"}

func serverRecordTable(schemas []recordSchema) ([]byte, error) {
	if len(schemas) > 0xffff {
		return nil, fmt.Errorf("record schema count %d exceeds u16", len(schemas))
	}
	out := make([]byte, 3)
	out[0] = 0x0a
	binary.LittleEndian.PutUint16(out[1:3], uint16(len(schemas)))
	for _, schema := range schemas {
		if schema.name == "" {
			return nil, fmt.Errorf("record schema name is empty")
		}
		if len(schema.colTypes) > 0xffff {
			return nil, fmt.Errorf("record schema %q columns %d exceeds u16", schema.name, len(schema.colTypes))
		}
		out = append(out, schema.name...)
		out = append(out, 0)
		var count [2]byte
		binary.LittleEndian.PutUint16(count[:], uint16(len(schema.colTypes)))
		out = append(out, count[:]...)
		out = append(out, schema.colTypes...)
	}
	return out, nil
}
func serverRecordAddString(objectID uint32, ownerID uint32, recordIndex uint16, value string) []byte {
	out := make([]byte, 20, 21+len(value))
	binary.LittleEndian.PutUint16(out[0:2], 0x11)
	binary.LittleEndian.PutUint32(out[2:6], objectID)
	binary.LittleEndian.PutUint32(out[6:10], ownerID)
	binary.LittleEndian.PutUint16(out[10:12], recordIndex)
	binary.LittleEndian.PutUint16(out[12:14], 0)
	binary.LittleEndian.PutUint16(out[14:16], 1)
	binary.LittleEndian.PutUint32(out[16:20], uint32(len(value)+1))
	out = append(out, value...)
	out = append(out, 0)
	return out
}
func serverRecordAddInt32(objectID uint32, ownerID uint32, recordIndex uint16, value uint32) []byte {
	out := make([]byte, 20)
	binary.LittleEndian.PutUint16(out[0:2], 0x11)
	binary.LittleEndian.PutUint32(out[2:6], objectID)
	binary.LittleEndian.PutUint32(out[6:10], ownerID)
	binary.LittleEndian.PutUint16(out[10:12], recordIndex)
	binary.LittleEndian.PutUint16(out[12:14], 0)
	binary.LittleEndian.PutUint16(out[14:16], 1)
	binary.LittleEndian.PutUint32(out[16:20], value)
	return out
}

type recordCell struct {
	intValue   *uint32
	int64Value *int64
	text       *string
}

func recordInt(value uint32) recordCell {
	return recordCell{intValue: &value}
}
func recordInt64(value int64) recordCell {
	return recordCell{int64Value: &value}
}
func recordString(value string) recordCell {
	return recordCell{text: &value}
}
func serverRecordAddCells(objectID uint32, ownerID uint32, recordIndex uint16, cells []recordCell) ([]byte, error) {
	if int(recordIndex) >= len(qingGongRecordSchemas) {
		return nil, fmt.Errorf("record index %d has no negotiated schema", recordIndex)
	}
	schema := qingGongRecordSchemas[recordIndex]
	if len(cells) != len(schema.colTypes) {
		return nil, fmt.Errorf("record %q has %d cells, want complete row of %d", schema.name, len(cells), len(schema.colTypes))
	}
	out := make([]byte, 16)
	binary.LittleEndian.PutUint16(out[0:2], 0x11)
	binary.LittleEndian.PutUint32(out[2:6], objectID)
	binary.LittleEndian.PutUint32(out[6:10], ownerID)
	binary.LittleEndian.PutUint16(out[10:12], recordIndex)
	binary.LittleEndian.PutUint16(out[12:14], 0)
	binary.LittleEndian.PutUint16(out[14:16], 1)
	for i, typ := range schema.colTypes {
		cell := cells[i]
		switch typ {
		case 3:
			if cell.intValue == nil || cell.int64Value != nil || cell.text != nil {
				return nil, fmt.Errorf("record column %d requires int32", i)
			}
			var value [4]byte
			binary.LittleEndian.PutUint32(value[:], *cell.intValue)
			out = append(out, value[:]...)
		case 4:
			if cell.int64Value == nil || cell.intValue != nil || cell.text != nil {
				return nil, fmt.Errorf("record column %d requires int64", i)
			}
			var value [8]byte
			binary.LittleEndian.PutUint64(value[:], uint64(*cell.int64Value))
			out = append(out, value[:]...)
		case 7:
			if cell.text == nil || cell.intValue != nil || cell.int64Value != nil {
				return nil, fmt.Errorf("record column %d requires string", i)
			}
			var length [4]byte
			binary.LittleEndian.PutUint32(length[:], uint32(len(*cell.text)+1))
			out = append(out, length[:]...)
			out = append(out, (*cell.text)...)
			out = append(out, 0)
		default:
			return nil, fmt.Errorf("record column %d has unsupported negotiated type %d", i, typ)
		}
	}
	return out, nil
}

const (
	functionUnlockTaskID  uint32 = 1346
	facultyGuideTaskFirst uint32 = 1069
	facultyGuideTaskLast  uint32 = 1073
	facultyAltTaskID      uint32 = 1344
	facultyActTaskFirst   uint32 = 1140
	facultyActTaskLast    uint32 = 1147
	facultyActAltTaskID   uint32 = 51002
)

func taskCompletedBit(taskID uint32) (row uint32, mask uint32) {
	serial := taskID + 1
	row = serial >> 5
	bit := serial & 31
	if bit == 0 {
		bit = 32
		row--
	}
	mask = uint32(1) << (bit - 1)
	return row, mask
}
func taskCompletedRows(taskIDs []uint32) (map[uint32]uint32, uint32) {
	rows := make(map[uint32]uint32)
	var finalRow uint32
	for _, taskID := range taskIDs {
		row, mask := taskCompletedBit(taskID)
		rows[row] |= mask
		if row > finalRow {
			finalRow = row
		}
	}
	return rows, finalRow
}
func facultyAcceptedTaskIDs() []uint32 {
	taskIDs := make([]uint32, 0, 15)
	for taskID := uint32(1069); taskID <= 1073; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	taskIDs = append(taskIDs, 1344)
	for taskID := uint32(1140); taskID <= 1147; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	taskIDs = append(taskIDs, 51002)
	return taskIDs
}
func grantFunctionUnlock(conn sceneMessageConnection) error {
	taskIDs := []uint32{1346, 1344, 51002}
	for taskID := uint32(1069); taskID <= 1073; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	for taskID := uint32(1140); taskID <= 1147; taskID++ {
		taskIDs = append(taskIDs, taskID)
	}
	for _, taskID := range facultyAcceptedTaskIDs() {
		state := uint32(9)
		zero0 := uint32(0)
		zero1 := uint32(0)
		empty1 := ""
		empty2 := ""
		frame, err := serverRecordAddCells(0x11000001, 0x007a5c0b, 2, []recordCell{{intValue: &taskID}, {intValue: &state}, {intValue: &zero0}, {intValue: &zero0}, {text: &empty1}, {text: &empty2}, {intValue: &zero1}})
		if err != nil {
			return fmt.Errorf("encode Task_Accepted task %d: %w", taskID, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add Task_Accepted task %d: %w", taskID, err)
		}
	}
	rows, finalRow := taskCompletedRows(taskIDs)
	for currentRow := uint32(0); currentRow <= finalRow; currentRow++ {
		value := rows[currentRow]
		if err := conn.WriteFrame(serverRecordAddInt32(0x11000001, 0x007a5c0b, 3, value)); err != nil {
			return fmt.Errorf("add Task_Completed row %d: %w", currentRow, err)
		}
	}
	log.Printf("Function/faculty unlock staged Task_Accepted tasks=%v and Task_Completed tasks=%v final-row=%d", facultyAcceptedTaskIDs(), taskIDs, finalRow)
	return nil
}
func grantStarterQingGong(conn sceneMessageConnection, roleName string, qgLevels map[string]int32) error {
	categoryIDs := starterQingGongIDs
	skillIDs, skillErr := qingGongSkillIDs()
	if isJianghuQingGongRole(roleName) {
		categoryIDs = allJianghuQingGongCategoryIDs
		skillIDs, skillErr = allJianghuQingGongSkillIDs()
	}
	if skillErr != nil {
		return fmt.Errorf("load qgskill table: %w", skillErr)
	}
	for _, id := range categoryIDs {
		if err := conn.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, 0, id)); err != nil {
			return fmt.Errorf("add QingGongRec %s: %w", id, err)
		}
	}
	viewItems := append([]string{}, categoryIDs...)
	viewItems = append(viewItems, skillIDs...)
	if err := conn.WriteFrame(serverCreateView(serverViewSpec{ID: 46, Capacity: 512})); err != nil {
		return fmt.Errorf("create QingGongContainer: %w", err)
	}
	for slot, id := range viewItems {
		props := []serverViewProperty{viewString(0x05A0, id), viewInt(0x0761, 1005)}
		if static, ok := qgStaticData[id]; ok && static.staticData > 0 {
			level := qgLevels[id]
			if level <= 0 {
				level = 1
			}
			maxLevel := static.maxLevel
			if maxLevel <= 0 {
				maxLevel = 1
			}
			if level > maxLevel {
				level = maxLevel
			}
			props = append(props, viewInt(0x08BD, static.staticData), viewByte(0x05B9, byte(level)), viewInt(0x0823, maxLevel))
		}
		frame, err := serverViewAdd(46, uint16(slot+1), props)
		if err != nil {
			return fmt.Errorf("encode QingGongContainer %s: %w", id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add QingGongContainer %s: %w", id, err)
		}
	}
	if err := grantFunctionUnlock(conn); err != nil {
		return err
	}
	return nil
}
func activateQingGongRecord(conn sceneMessageConnection, player *playerActor, id string) (bool, error) {
	if player == nil {
		return false, fmt.Errorf("character has not entered a scene")
	}
	if id == "" {
		return false, fmt.Errorf("active qinggong id is empty")
	}
	if _, ok := player.activateQingGong(id); !ok {
		return false, nil
	}
	if err := conn.WriteFrame(serverRecordAddString(playerObjectID, playerOwnerID, 1, id)); err != nil {
		return false, fmt.Errorf("add ActiveQGSkillRec %s: %w", id, err)
	}
	return true, nil
}
func grantRedSuperArmor(conn sceneMessageConnection, player *playerActor) (time.Time, error) {
	if player == nil {
		return time.Time{}, fmt.Errorf("character has not entered a scene")
	}
	now := time.Now()
	buffState, expires := player.activateRedSuperArmor(now)
	log.Printf("Buff add state player=%d-%d property=%d value=%q expiry=%d", playerObjectID, playerOwnerID, bufferInfoFirstPropertyIndex, buffState, expires.UnixMilli())
	update, err := player.vitalUpdate()
	if err != nil {
		return time.Time{}, fmt.Errorf("encode red Buff state: %w", err)
	}
	if err := conn.WriteFrame(update); err != nil {
		return time.Time{}, fmt.Errorf("write red Buff state: %w", err)
	}
	return expires, nil
}
func activateQingGongBuff(player *playerActor, definition qinggong.Definition, now time.Time) (alreadyActive bool, expires time.Time, err error) {
	spec, exists := qinggongBuffFor(definition)
	if !exists {
		return false, time.Time{}, nil
	}
	if spec.clearOnEnd && player.hasActiveBuff(spec.slot, spec.staticData, now) {
		return true, time.Time{}, nil
	}
	buffState, expires := player.activateBuff(spec.slot, spec.staticData, spec.duration, now)
	log.Printf("QingGong Buff staged id=%s property=%d value=%q expiry=%d", definition.ID, bufferInfoPropertyIndex(spec.slot), buffState, expires.UnixMilli())
	return false, expires, nil
}
func endQingGongBuff(conn sceneMessageConnection, player *playerActor, definition qinggong.Definition) error {
	spec, exists := qinggongBuffFor(definition)
	if !exists || !spec.clearOnEnd {
		return nil
	}
	if _, changed := player.clearBuff(spec.slot, spec.staticData); !changed {
		return nil
	}
	update, err := player.vitalUpdate()
	if err != nil {
		return fmt.Errorf("encode qinggong Buff removal: %w", err)
	}
	if err := conn.WriteFrame(update); err != nil {
		return fmt.Errorf("write qinggong Buff removal: %w", err)
	}
	log.Printf("QingGong Buff remove id=%s property=%d", definition.ID, bufferInfoPropertyIndex(spec.slot))
	return nil
}
func scheduleRedSuperArmorExpiry(conn sceneMessageConnection, player *playerActor, expires time.Time) {
	schedulePlayerBuffExpiry(conn, player, redSuperArmorBuffSlot, redSuperArmorStaticData, expires)
}
func schedulePlayerBuffExpiry(conn sceneMessageConnection, player *playerActor, slot uint16, staticData uint32, expires time.Time) {
	delay := time.Until(expires)
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		now := time.Now()
		if _, ok := player.expireBuff(slot, staticData, expires, now); !ok {
			return
		}
		frame, err := player.vitalUpdate()
		if err != nil {
			log.Printf("Buff remove encode: %v", err)
			return
		}
		if err := conn.WriteFrame(frame); err != nil {
			log.Printf("Buff remove state: %v", err)
			return
		}
		log.Printf("Buff remove state player=%d-%d property=%d expiry=%d", playerObjectID, playerOwnerID, bufferInfoPropertyIndex(slot), expires.UnixMilli())
	})
}
