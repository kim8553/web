package main

import (
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	"log"
)

const (
	clientCustomSetShortcut          int32 = 200
	clientCustomRemoveShortcut       int32 = 201
	clientCustomSitCross             int32 = 240
	sitcrossShortcutIndex            int32 = 9
	sitcrossShortcutKind                   = "skill"
	sitcrossShortcutID                     = "zs_default_01"
	heartBuddhaPalmShortcutBaseIndex int32 = 10
	heartBuddhaPalmShortcutKind            = "skill"
	taiJiQuanGuPuShortcutBaseIndex   int32 = 20
	taiJiQuanGuPuShortcutKind              = "skill"
)

var heartBuddhaPalmShortcutIDs = []string{"CS_jh_xfz01", "CS_jh_xfz02", "CS_jh_xfz03", "CS_jh_xfz04", "CS_jh_xfz05", "CS_jh_xfz06", "CS_jh_xfz07"}
var taiJiQuanGuPuShortcutIDs = []string{"CS_wd_tjq01", "CS_wd_tjq02", "CS_wd_tjq03", "CS_wd_tjq04", "CS_wd_tjq05", "CS_wd_tjq06", "CS_wd_tjq07", "CS_wd_tjq08"}

type playerShortcut struct {
	index int32
	kind  string
	id    string
}

func serverRecordDelRow(objectID uint32, ownerID uint32, recordIndex uint16, row uint16) []byte {
	out := make([]byte, 14)
	binary.LittleEndian.PutUint16(out[0:2], 0x12)
	binary.LittleEndian.PutUint32(out[2:6], objectID)
	binary.LittleEndian.PutUint32(out[6:10], ownerID)
	binary.LittleEndian.PutUint16(out[10:12], recordIndex)
	binary.LittleEndian.PutUint16(out[12:14], row)
	return out
}
func (p *playerActor) setShortcut(index int32, kind, id string) (existed, unchanged bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := 0; i < len(p.shortcuts); i++ {
		if p.shortcuts[i].index == index {
			continue
		}
		if p.shortcuts[i].kind == kind && p.shortcuts[i].id == id {
			p.shortcuts = append(p.shortcuts[:i], p.shortcuts[i+1:]...)
			break
		}
	}
	for row, shortcut := range p.shortcuts {
		if shortcut.index != index {
			continue
		}
		if shortcut.kind == kind && shortcut.id == id {
			return true, true
		}
		p.shortcuts[row] = playerShortcut{index: index, kind: kind, id: id}
		return true, false
	}
	p.shortcuts = append(p.shortcuts, playerShortcut{index: index, kind: kind, id: id})
	return false, false
}
func (p *playerActor) removeShortcutRow(row int) (playerShortcut, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if row < 0 || row >= len(p.shortcuts) {
		return playerShortcut{}, false
	}
	shortcut := p.shortcuts[row]
	p.shortcuts = append(p.shortcuts[:row], p.shortcuts[row+1:]...)
	return shortcut, true
}
func (p *playerActor) shortcutSnapshot() []playerShortcut {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]playerShortcut(nil), p.shortcuts...)
}
func shortcutAddFrame(shortcut playerShortcut) ([]byte, error) {
	if shortcut.index < 0 {
		return nil, fmt.Errorf("shortcut index %d is negative", shortcut.index)
	}
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordShortcut, []recordCell{recordInt(uint32(shortcut.index)), recordString(shortcut.kind), recordString(shortcut.id)})
}
func grantShortcutRows(conn sceneMessageConnection, player *playerActor) error {
	if player == nil {
		return nil
	}
	for _, shortcut := range player.shortcutSnapshot() {
		frame, err := shortcutAddFrame(shortcut)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("restore shortcut index %d: %w", shortcut.index, err)
		}
	}
	return nil
}
func grantSitcrossShortcut(conn sceneMessageConnection, player *playerActor) (bool, error) {
	if player == nil {
		return false, nil
	}
	player.mu.Lock()
	for _, shortcut := range player.shortcuts {
		if shortcut.index == sitcrossShortcutIndex {
			player.mu.Unlock()
			return false, nil
		}
	}
	shortcut := playerShortcut{index: sitcrossShortcutIndex, kind: sitcrossShortcutKind, id: sitcrossShortcutID}
	player.shortcuts = append(player.shortcuts, shortcut)
	player.mu.Unlock()
	frame, err := shortcutAddFrame(shortcut)
	if err != nil {
		return false, err
	}
	if err := conn.WriteFrame(frame); err != nil {
		return false, fmt.Errorf("add sitcross shortcut: %w", err)
	}
	return true, nil
}
func handleSitcrossCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSitCross {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject sitcross before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 2 || custom.Values[1].Type != 2 {
		return true, fmt.Errorf("sitcross expects TVarList(240, int start_or_stop), got %v", custom.Values)
	}
	var next uint8
	switch custom.Values[1].Int32 {
	case 1:
		next = entity.LogicStateSitcross
	case 0, 2:
		next = entity.LogicStateNormal
	default:
		log.Printf("%s: ignore unknown sitcross start_or_stop=%d", remote, custom.Values[1].Int32)
		return true, nil
	}
	current := player.actor.Snapshot().LogicState
	if current == entity.LogicStateDied {
		log.Printf("%s: reject sitcross while dead", remote)
		return true, nil
	}
	if current == next {
		log.Printf("%s: sitcross state already %d; duplicate request ignored", remote, next)
		return true, nil
	}
	player.actor.SetLogicState(next)
	frame, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode sitcross state=%d: %w", next, err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("send sitcross state=%d: %w", next, err)
	}
	log.Printf("%s: sitcross native state %d -> %d via normal 0x10", remote, current, next)
	return true, nil
}
func handleShortcutCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string, store shortcutStoreIface, roleID role.RoleID) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 {
		return false, nil
	}
	messageID := custom.Values[0].Int32
	if messageID != clientCustomSetShortcut && messageID != clientCustomRemoveShortcut {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject shortcut request before player spawn", remote)
		return true, nil
	}
	log.Printf("%s: shortcut request msg_id=%d current_bar=%v", remote, messageID, player.shortcutSnapshot())
	switch messageID {
	case clientCustomSetShortcut:
		if len(custom.Values) != 4 || custom.Values[1].Type != 2 || custom.Values[2].Type != 6 || custom.Values[3].Type != 6 {
			return true, fmt.Errorf("set shortcut expects int index and two strings, got %v", custom.Values)
		}
		shortcut := playerShortcut{index: custom.Values[1].Int32, kind: custom.Values[2].Text, id: custom.Values[3].Text}
		if shortcut.index < 0 || shortcut.index > 4096 {
			return true, fmt.Errorf("shortcut index %d is out of range", shortcut.index)
		}
		if shortcut.kind == "" || shortcut.id == "" || len(shortcut.kind) > 64 || len(shortcut.id) > 256 {
			return true, fmt.Errorf("invalid shortcut kind/id lengths %d/%d", len(shortcut.kind), len(shortcut.id))
		}
		existed, unchanged := player.setShortcut(shortcut.index, shortcut.kind, shortcut.id)
		if unchanged {
			log.Printf("%s: shortcut unchanged index=%d kind=%s id=%s", remote, shortcut.index, shortcut.kind, shortcut.id)
			return true, nil
		}
		if err := player.resyncShortcutRecord(conn); err != nil {
			return true, err
		}
		log.Printf("%s: shortcut set index=%d kind=%s id=%s replaced=%t", remote, shortcut.index, shortcut.kind, shortcut.id, existed)
		persistShortcuts(store, roleID, player, remote, "set")
		return true, nil
	case clientCustomRemoveShortcut:
		if len(custom.Values) != 2 || custom.Values[1].Type != 2 {
			return true, fmt.Errorf("remove shortcut expects int index, got %v", custom.Values)
		}
		index := custom.Values[1].Int32
		shortcut, exists := player.removeShortcutByIndex(index)
		if !exists {
			log.Printf("%s: ignore missing shortcut index=%d", remote, index)
			return true, nil
		}
		if err := player.resyncShortcutRecord(conn); err != nil {
			return true, err
		}
		log.Printf("%s: shortcut removed index=%d kind=%s id=%s", remote, index, shortcut.kind, shortcut.id)
		persistShortcuts(store, roleID, player, remote, "remove")
		return true, nil
	}
	return false, nil
}
