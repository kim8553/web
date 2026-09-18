package shopbuyatomic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
)

// ErrWalletChanged identifies an ordinary NPC purchase rejected by the
// locked database wallet comparison. Callers must not treat other DB
// failures as safe to reconcile.
var ErrWalletChanged = errors.New("shop buy: wallet changed since purchase began")

// ErrBagChanged identifies ONLY a persisted-bag comparison mismatch.
// Other database failures must not trigger a destructive client replay.
var ErrBagChanged = errors.New("shop buy: bag changed since purchase began")

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
	return a.Slot == b.Slot && a.ConfigID == b.ConfigID && a.ItemType == b.ItemType && a.Amount == b.Amount && a.ViewID == b.ViewID &&
		reflect.DeepEqual(a.Name, b.Name) && reflect.DeepEqual(a.EquipType, b.EquipType) &&
		reflect.DeepEqual(a.ArtPack, b.ArtPack) && reflect.DeepEqual(a.Hardiness, b.Hardiness) &&
		reflect.DeepEqual(a.MaxHardiness, b.MaxHardiness)
}

// The actor decodes role_currency.snapshot into a struct: omitted fields load
// as zero and are serialized explicitly in its pre-purchase snapshot. Accept
// that representational difference, but reject changed nonzero values and
// unexpected persisted keys so a purchase cannot silently discard currency.
func sameWalletValues(stored, expected map[string]int64) bool {
	for key, value := range stored {
		want, known := expected[key]
		if !known || want != value {
			return false
		}
	}
	for key, value := range expected {
		if _, present := stored[key]; !present && value != 0 {
			return false
		}
	}
	return true
}

// Check every field that a checked bag rewrite persists. Checking only the
// first five allows a concurrent durability/name update to be overwritten by
// an otherwise valid shop purchase or bag move.
func checkLockedBag(ctx context.Context, tx *sql.Tx, roleID uint64, expected []Row) error {
	stored, err := tx.QueryContext(ctx, `SELECT slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness
FROM role_bag_items WHERE role_id = ? ORDER BY seq ASC FOR UPDATE`, roleID)
	if err != nil {
		return fmt.Errorf("lock shop bag: %w", err)
	}
	defer stored.Close()
	index := 0
	for stored.Next() {
		var actual Row
		var name, equipType sql.NullString
		var artPack, hardiness, maxHardiness sql.NullInt64
		if err := stored.Scan(&actual.Slot, &actual.ConfigID, &actual.ItemType, &actual.Amount, &actual.ViewID,
			&name, &equipType, &artPack, &hardiness, &maxHardiness); err != nil {
			return fmt.Errorf("read locked shop bag row %d: %w", index, err)
		}
		if name.Valid {
			actual.Name = name.String
		}
		if equipType.Valid {
			actual.EquipType = equipType.String
		}
		for _, field := range []struct {
			name  string
			value sql.NullInt64
			dest  *any
		}{
			{"art_pack", artPack, &actual.ArtPack},
			{"hardiness", hardiness, &actual.Hardiness},
			{"max_hardiness", maxHardiness, &actual.MaxHardiness},
		} {
			if !field.value.Valid {
				continue
			}
			if field.value.Int64 < math.MinInt32 || field.value.Int64 > math.MaxInt32 {
				return fmt.Errorf("%w at row %d: %s out of int32 range", ErrBagChanged, index, field.name)
			}
			*field.dest = int32(field.value.Int64)
		}
		if index >= len(expected) || !sameBagIdentity(actual, expected[index]) {
			return fmt.Errorf("%w at row %d; reload before retrying", ErrBagChanged, index)
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
		return fmt.Errorf("%w (rows %d, expected %d); reload before retrying", ErrBagChanged, index, len(expected))
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
		if !sameWalletValues(stored, expected) {
			return fmt.Errorf("%w; reload before retrying", ErrWalletChanged)
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
