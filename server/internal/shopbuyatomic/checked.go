package shopbuyatomic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// SaveChecked persists a purchase only when its starting wallet still matches
// the database. The role_currency primary key serializes competing purchases;
// the initial absent row is inserted once (a concurrent duplicate insert fails).
// This protects overlapping purchases, not unrelated bag-only writes.
func SaveChecked(db *sql.DB, roleID uint64, rows []Row, expectedJSON, nextJSON []byte) error {
	if db == nil || roleID == 0 || len(expectedJSON) == 0 || len(nextJSON) == 0 {
		return errors.New("shop buy: missing database, role, or wallet snapshot")
	}
	var expected, next map[string]int64
	if err := json.Unmarshal(expectedJSON, &expected); err != nil || expected == nil {
		return fmt.Errorf("shop buy: invalid starting wallet: %v", err)
	}
	if err := json.Unmarshal(nextJSON, &next); err != nil || next == nil {
		return fmt.Errorf("shop buy: invalid resulting wallet: %v", err)
	}
	for seq, row := range rows {
		if row.Slot < 1 || row.Slot > 65535 || row.ConfigID == "" || row.Amount <= 0 {
			return fmt.Errorf("shop buy: invalid bag row %d", seq)
		}
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin checked shop purchase: %w", err)
	}
	defer tx.Rollback()
	var storedJSON []byte
	err = tx.QueryRowContext(ctx, "SELECT snapshot FROM role_currency WHERE role_id = ? FOR UPDATE", roleID).Scan(&storedJSON)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lock shop wallet: %w", err)
	}
	if err == nil {
		var stored map[string]int64
		if decodeErr := json.Unmarshal(storedJSON, &stored); decodeErr != nil || stored == nil {
			return fmt.Errorf("decode locked shop wallet: %v", decodeErr)
		}
		if !reflect.DeepEqual(stored, expected) {
			return errors.New("shop buy: wallet changed since purchase began; reload before retrying")
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {
		return fmt.Errorf("clear checked shop bag: %w", err)
	}
	for seq, row := range rows {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, roleID, seq, row.Slot, row.ConfigID, row.ItemType, row.Amount, row.ViewID, row.Name, row.EquipType, row.ArtPack, row.Hardiness, row.MaxHardiness); err != nil {
			return fmt.Errorf("save checked shop bag row %d: %w", seq, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_currency WHERE role_id = ?", roleID); err != nil {
		return fmt.Errorf("clear checked shop wallet: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_currency(role_id, snapshot) VALUES (?, ?)", roleID, nextJSON); err != nil {
		return fmt.Errorf("save checked shop wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit checked shop purchase: %w", err)
	}
	return nil
}
