package shopbuyatomic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// SaveWalletChecked persists a currency change after an already committed
// ordinary NPC purchase only if the stored wallet still equals that purchase.
// It never touches the bag, and does not alter other currency writer paths.
func SaveWalletChecked(db *sql.DB, roleID uint64, expectedJSON, nextJSON []byte) error {
	if db == nil || roleID == 0 || len(expectedJSON) == 0 || len(nextJSON) == 0 {
		return errors.New("checked shop wallet: missing database, role, or snapshot")
	}
	var expected, next map[string]int64
	if err := json.Unmarshal(expectedJSON, &expected); err != nil || expected == nil {
		return fmt.Errorf("checked shop wallet: invalid starting snapshot: %v", err)
	}
	if err := json.Unmarshal(nextJSON, &next); err != nil || next == nil {
		return fmt.Errorf("checked shop wallet: invalid resulting snapshot: %v", err)
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin checked shop wallet: %w", err)
	}
	defer tx.Rollback()
	// Match the ordinary purchase's role -> wallet lock ordering.
	if err := LockRoleForBagWrite(ctx, tx, roleID); err != nil {
		return err
	}
	var storedJSON []byte
	if err := tx.QueryRowContext(ctx, "SELECT snapshot FROM role_currency WHERE role_id = ? FOR UPDATE", roleID).Scan(&storedJSON); err != nil {
		return fmt.Errorf("lock checked shop wallet: %w", err)
	}
	var stored map[string]int64
	if err := json.Unmarshal(storedJSON, &stored); err != nil || stored == nil {
		return fmt.Errorf("decode checked shop wallet: %v", err)
	}
	if !reflect.DeepEqual(stored, expected) {
		return errors.New("checked shop wallet: wallet changed since ordinary purchase; reload before retrying")
	}
	result, err := tx.ExecContext(ctx, "UPDATE role_currency SET snapshot = ? WHERE role_id = ?", nextJSON, roleID)
	if err != nil {
		return fmt.Errorf("update checked shop wallet: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return fmt.Errorf("checked shop wallet: expected one updated row, got %d: %v", count, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit checked shop wallet: %w", err)
	}
	return nil
}
