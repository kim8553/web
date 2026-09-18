package main

import (
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

type itemUseSaveCapture struct{ saves int }

func (*itemUseSaveCapture) Load(role.RoleID) ([]bagItem, bool) { return nil, false }
func (capture *itemUseSaveCapture) Save(role.RoleID, []bagItem) error {
	capture.saves++
	return nil
}

// An item must not disappear merely because its effect is named in the
// original data: the server must have a corresponding effect implementation.
func TestUnwiredFuncBufferPreservesInventory(t *testing.T) {
	player := newPlayerActor("effect-guard", 0)
	item := bagItem{ConfigID: "test_unwired_buff", ItemType: 50, ViewID: 1, Amount: 2, FuncBuffer: "test_unwired_effect"}
	item.Slot = int32(player.addBagItem(item))
	conn := &captureMessageConnection{}
	store := &itemUseSaveCapture{}
	handled, err := applyUseBuffItem(conn, player, store, role.RoleID(1), bagViewForViewID(item.ViewID), item.Slot, item, "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	remaining, ok := player.peekBagItem(bagViewForViewID(item.ViewID), item.Slot)
	if !ok || remaining.Amount != item.Amount || player.yufengMode != 0 || len(conn.Frames()) != 0 || store.saves != 0 {
		t.Fatalf("unwired buff was consumed or activated: present=%t remaining=%+v yufeng=%d frames=%d saves=%d", ok, remaining, player.yufengMode, len(conn.Frames()), store.saves)
	}
}

func TestUnwiredToolUseEffectPreservesInventory(t *testing.T) {
	player := newPlayerActor("effect-guard", 0)
	item := bagItem{ConfigID: "test_unwired_tool", ItemType: 50, ViewID: 1, Amount: 2, ToolUseEffect: "test_unwired_tool_effect"}
	item.Slot = int32(player.addBagItem(item))
	conn := &captureMessageConnection{}
	store := &itemUseSaveCapture{}
	handled, err := applyUseConsumable(conn, player, nil, nil, nil, store, nil, role.RoleID(1), bagViewForViewID(item.ViewID), item.Slot, item, "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	remaining, ok := player.peekBagItem(bagViewForViewID(item.ViewID), item.Slot)
	if !ok || remaining.Amount != item.Amount || len(conn.Frames()) != 0 || store.saves != 0 {
		t.Fatalf("unwired tool effect consumed item: present=%t remaining=%+v frames=%d saves=%d", ok, remaining, len(conn.Frames()), store.saves)
	}
}

func TestVerifiedYufengBuffStillConsumesAndActivates(t *testing.T) {
	player := newPlayerActor("yufeng-guard", 0)
	item := bagItem{ConfigID: "test_yufeng", ItemType: 50, ViewID: 1, Amount: 2, FuncBuffer: "buf_ride_yufeng"}
	item.Slot = int32(player.addBagItem(item))
	conn := &captureMessageConnection{}
	store := &itemUseSaveCapture{}
	handled, err := applyUseBuffItem(conn, player, store, role.RoleID(1), bagViewForViewID(item.ViewID), item.Slot, item, "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	remaining, ok := player.peekBagItem(bagViewForViewID(item.ViewID), item.Slot)
	if !ok || remaining.Amount != 1 || player.yufengMode != 1 || len(conn.Frames()) == 0 || store.saves != 1 {
		t.Fatalf("supported buff regression: present=%t remaining=%+v yufeng=%d frames=%d saves=%d", ok, remaining, player.yufengMode, len(conn.Frames()), store.saves)
	}
}

func TestOrdinaryYufengMatchesSourceItemAndReadyBuff(t *testing.T) {
	catalog, err := loadItemCatalog(defaultToolItemINI)
	if err != nil {
		t.Fatal(err)
	}
	item, ok := catalog.Lookup("ride_windrunner_001")
	if !ok || item.FuncBuffer != "buf_ride_yufeng" || item.WindReadyBuff != "buf_ride_yufeng_ready" {
		t.Fatalf("ordinary yufeng source item missing or mismatched: present=%t item=%+v", ok, item)
	}
	if got := iniInt(installedCombatSkillCatalog.tables.buffNew[item.WindReadyBuff], "StaticData"); got != 10532 {
		t.Fatalf("ordinary yufeng source ready-buff static=%d, existing handler uses 10532", got)
	}
}

func TestOrdinaryYufengCurrentUseItemDispatch(t *testing.T) {
	catalog, err := loadItemCatalog(defaultToolItemINI)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayerActor("yufeng-dispatch", 0)
	item := bagItem{ConfigID: "ride_windrunner_001", ViewID: 1, Amount: 2}
	item.Slot = int32(player.addBagItem(item))
	view := bagViewForViewID(item.ViewID)
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 31}, {Type: 2, Int32: int32(view)}, {Type: 2, Int32: item.Slot},
	}}
	store := &itemUseSaveCapture{}
	conn := &captureMessageConnection{}
	handled, err := handleUseItemCustom(conn, player, catalog, nil, nil, store, nil, nil, role.RoleID(1), custom, "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	remaining, exists := player.peekBagItem(view, item.Slot)
	if !exists || remaining.Amount != 1 || player.yufengMode != 1 || len(conn.Frames()) == 0 || store.saves != 1 {
		t.Fatalf("source yufeng USEITEM route: exists=%t remaining=%+v mode=%d frames=%d saves=%d", exists, remaining, player.yufengMode, len(conn.Frames()), store.saves)
	}
}

func TestDifferentJinwuYufengDoesNotUseOrdinaryEffect(t *testing.T) {
	catalog, err := loadItemCatalog(defaultToolItemINI)
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := catalog.Lookup("ride_windrunner_jinwu001")
	if !ok || definition.FuncBuffer != "buf_ride_yufeng_jinwu" {
		t.Fatalf("jinwu source is missing or changed: present=%t definition=%+v", ok, definition)
	}
	player := newPlayerActor("jinwu-guard", 0)
	item := bagItem{ConfigID: definition.ConfigID, ItemType: definition.ItemType, ViewID: 1, Amount: 2, FuncBuffer: definition.FuncBuffer}
	item.Slot = int32(player.addBagItem(item))
	store := &itemUseSaveCapture{}
	conn := &captureMessageConnection{}
	handled, err := applyUseBuffItem(conn, player, store, role.RoleID(1), bagViewForViewID(item.ViewID), item.Slot, item, "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	remaining, exists := player.peekBagItem(bagViewForViewID(item.ViewID), item.Slot)
	if !exists || remaining.Amount != 2 || player.yufengMode != 0 || len(conn.Frames()) != 0 || store.saves != 0 {
		t.Fatalf("jinwu incorrectly used ordinary yufeng effect: remains=%+v exists=%t mode=%d frames=%d saves=%d", remaining, exists, player.yufengMode, len(conn.Frames()), store.saves)
	}
}
