package shopbuyatomic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// LockRoleForBagWrite serializes participating bag writes for an existing
// migrated role. A role row exists even when the bag or wallet is empty.
// This lock alone does not prove a legacy caller's snapshot is fresh.
func LockRoleForBagWrite(ctx context.Context, tx *sql.Tx, roleID uint64) error {
	if tx == nil || roleID == 0 {
		return errors.New("shop bag: missing transaction or role")
	}
	var lockedID uint64
	if err := tx.QueryRowContext(ctx, "SELECT role_id FROM roles WHERE role_id = ? FOR UPDATE", roleID).Scan(&lockedID); err != nil {
		return fmt.Errorf("lock role for bag write: %w", err)
	}
	if lockedID != roleID {
		return errors.New("shop bag: locked wrong role")
	}
	return nil
}

// SaveChecked retains the wallet-only check for existing callers. Purchases
// with a known starting bag must use SaveCheckedBag instead.
func SaveChecked(db *sql.DB, roleID uint64, rows []Row, expectedJSON, nextJSON []byte) error {
	return saveChecked(db, roleID, rows, nil, false, expectedJSON, nextJSON)
}

// SaveCheckedBag compares the current database bag against the actor's
// pre-purchase bag inside the same transaction as the wallet check and writes.
// It detects stale purchases after a completed bag edit. Other bag writers
// must also coordinate to guarantee against every possible interleaving.
func SaveCheckedBag(db *sql.DB, roleID uint64, nextRows, expectedBag []Row, expectedJSON, nextJSON []byte) error {
	return saveChecked(db, roleID, nextRows, expectedBag, true, expectedJSON, nextJSON)
}

func sameBagIdentity(a, b Row) bool {
	return a.Slot == b.Slot && a.ConfigID == b.ConfigID && a.ItemType == b.ItemType && a.Amount == b.Amount && a.ViewID == b.ViewID
}

// The five fields below come directly from the existing mysqlBagStore.Load
// projection and are not enriched by item/equipment catalog lookups.
func checkLockedBag(ctx context.Context, tx *sql.Tx, roleID uint64, expected []Row) error {
	stored, err := tx.QueryContext(ctx, `SELECT slot, config_id, item_type, amount, view_id
FROM role_bag_items WHERE role_id = ? ORDER BY seq ASC FOR UPDATE`, roleID)
	if err != nil {
		return fmt.Errorf("lock shop bag: %w", err)
	}
	defer stored.Close()
	index := 0
	for stored.Next() {
		var actual Row
		if err := stored.Scan(&actual.Slot, &actual.ConfigID, &actual.ItemType, &actual.Amount, &actual.ViewID); err != nil {
			return fmt.Errorf("read locked shop bag row %d: %w", index, err)
		}
		if index >= len(expected) || !sameBagIdentity(actual, expected[index]) {
			return fmt.Errorf("shop buy: bag changed since purchase began at row %d; reload before retrying", index)
		}
		index++
	}
	if err := stored.Err(); err != nil {
		return fmt.Errorf("read locked shop bag: %w", err)
	}
	if err := stored.Close(); err != nil {
		return fmt.Errorf("close locked shop bag: %w", err)
	}
	if index != len(expected) {
		return fmt.Errorf("shop buy: bag changed since purchase began (rows %d, expected %d); reload before retrying", index, len(expected))
	}
	return nil
}

func saveChecked(db *sql.DB, roleID uint64, rows, expectedBag []Row, checkBag bool, expectedJSON, nextJSON []byte) error {
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
	if checkBag {
		if err := LockRoleForBagWrite(ctx, tx, roleID); err != nil {
			return err
		}
	}
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
	if checkBag {
		if err := checkLockedBag(ctx, tx, roleID, expectedBag); err != nil {
			return err
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
