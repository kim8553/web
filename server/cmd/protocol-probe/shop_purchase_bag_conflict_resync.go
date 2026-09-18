package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/local/9yin-go-server/internal/role"
)

// loadOrdinaryShopConflictBag deliberately does not use mysqlBagStore.Load:
// its bool result cannot distinguish a legitimately empty bag from a query
// failure. A failed read must never erase an actor's inventory.
func loadOrdinaryShopConflictBag(store *mysqlBagStore, roleID role.RoleID) ([]bagItem, error) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, errors.New("shop bag conflict: missing MySQL store or role")
	}
	rows, err := store.db.QueryContext(context.Background(), `
SELECT config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness, COALESCE(slot, 0)
FROM role_bag_items WHERE role_id = ? ORDER BY seq ASC`, roleID)
	if err != nil {
		return nil, fmt.Errorf("shop bag conflict: reload persisted bag: %w", err)
	}
	defer rows.Close()
	var items []bagItem
	seen := make(map[[2]uint16]bool)
	for rows.Next() {
		var item bagItem
		var name, equipType sql.NullString
		var artPack, hardiness, maxHardiness, slot sql.NullInt64
		if err := rows.Scan(&item.ConfigID, &item.ItemType, &item.Amount, &item.ViewID,
			&name, &equipType, &artPack, &hardiness, &maxHardiness, &slot); err != nil {
			return nil, fmt.Errorf("shop bag conflict: scan persisted item: %w", err)
		}
		view := bagViewForViewID(item.ViewID)
		if view == 0 || !slot.Valid || slot.Int64 < 1 || slot.Int64 > 65535 || item.ConfigID == "" || item.Amount <= 0 {
			return nil, fmt.Errorf("shop bag conflict: invalid persisted item at row %d", len(items))
		}
		key := [2]uint16{view, uint16(slot.Int64)}
		if seen[key] {
			return nil, fmt.Errorf("shop bag conflict: duplicate persisted view/slot %d/%d", key[0], key[1])
		}
		seen[key] = true
		item.Slot = int32(slot.Int64)
		item.Name, item.EquipType = name.String, equipType.String
		item.ArtPack = int32(artPack.Int64)
		item.Hardiness = int32(hardiness.Int64)
		item.MaxHardiness = int32(maxHardiness.Int64)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("shop bag conflict: read persisted bag: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("shop bag conflict: close persisted bag: %w", err)
	}
	return items, nil // nil, nil means a valid empty bag, not a failed read.
}

// resyncOrdinaryShopBagAfterConflict runs ONLY after the checked NPC purchase
// reports ErrBagChanged. It never retries or grants the rejected reward. The
// existing VIEW_REMOVE / current-client VIEW_ADD layouts are reused; we do not
// delete/recreate whole views or invent a packet. If the actor has changed
// since the failed purchase snapshot, refuse to discard local changes.
func resyncOrdinaryShopBagAfterConflict(link sceneMessageConnection, player *playerActor, store bagStoreIface, roleID role.RoleID, expectedActor []bagItem) error {
	mysqlStore, ok := store.(*mysqlBagStore)
	if !ok || mysqlStore == nil || mysqlStore.db == nil || roleID == 0 || player == nil || link == nil {
		return errors.New("shop bag conflict: missing live MySQL store, role, player, or connection")
	}
	latest, err := loadOrdinaryShopConflictBag(mysqlStore, roleID)
	if err != nil {
		return err
	}
	frames := make([][]byte, 0, len(expectedActor)+len(latest))
	for _, item := range expectedActor {
		view := bagViewForViewID(item.ViewID)
		if view == 0 || item.Slot < 1 || item.Slot > 65535 {
			return errors.New("shop bag conflict: cannot encode old bag view or slot")
		}
		frames = append(frames, serverViewRemove(view, uint16(item.Slot)))
	}
	for _, item := range latest {
		view := bagViewForViewID(item.ViewID)
		adds, err := latestClientCurrentBagFrames(view, uint16(item.Slot), item.ViewID, bagItemProps(view, item))
		if err != nil {
			return fmt.Errorf("shop bag conflict: encode existing bag replay: %w", err)
		}
		frames = append(frames, adds...)
	}
	player.mu.Lock()
	if !reflect.DeepEqual(player.bagItems, expectedActor) {
		player.mu.Unlock()
		return errors.New("shop bag conflict: local bag changed; refusing to discard actor inventory")
	}
	player.bagItems = append([]bagItem(nil), latest...)
	player.mu.Unlock()
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			// The DB remains authoritative. The caller closes an incomplete view
			// replay rather than treating this failed purchase as successful.
			return fmt.Errorf("shop bag conflict: publish persisted bag refresh: %w", err)
		}
	}
	return nil
}
