package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// normalShopPersistedCurrency matches the existing role_currency.snapshot JSON
// names. It does not define a new wallet or an upstream purchase protocol.
type normalShopPersistedCurrency struct {
	Silver       int32 `json:"silver"`
	Gold         int32 `json:"gold"`
	SilverCard   int32 `json:"silver_card"`
	SilverTicket int32 `json:"silver_ticket"`
}

// normalShopPersistedBagRow contains only fields already saved by mysqlBagStore.Save.
type normalShopPersistedBagRow struct {
	ConfigID, Name, EquipType                                        string
	ItemType, Amount, ViewID, ArtPack, Hardiness, MaxHardiness, Slot int32
}

func normalShopSamePersistedBag(a, b []normalShopPersistedBagRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// normalShopCommitMySQLSnapshots commits the existing MySQL bag and currency
// representations together, or neither. The before snapshots MUST correspond
// to the caller's actor state; a concurrent/stale DB change rejects the write.
// It does not update playerActor or emit client frames; those remain caller
// responsibilities after a successful commit. No JSON fallback is attempted.
func normalShopCommitMySQLSnapshots(ctx context.Context, db *sql.DB, roleID uint64, beforeCurrency, afterCurrency normalShopPersistedCurrency, beforeBag, afterBag []normalShopPersistedBagRow) error {
	if ctx == nil || db == nil || roleID == 0 {
		return errors.New("normal shop transaction: missing context, MySQL connection, or role")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("normal shop transaction: begin: %w", err)
	}
	defer tx.Rollback()

	var encoded []byte
	if err := tx.QueryRowContext(ctx, "SELECT snapshot FROM role_currency WHERE role_id = ? FOR UPDATE", roleID).Scan(&encoded); err != nil {
		return fmt.Errorf("normal shop transaction: lock existing currency row: %w", err)
	}
	var storedCurrency normalShopPersistedCurrency
	if err := json.Unmarshal(encoded, &storedCurrency); err != nil {
		return fmt.Errorf("normal shop transaction: decode persisted currency: %w", err)
	}
	if storedCurrency != beforeCurrency {
		return errors.New("normal shop transaction: actor currency differs from persisted currency")
	}

	rows, err := tx.QueryContext(ctx, `SELECT config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness, COALESCE(slot, 0)
FROM role_bag_items WHERE role_id = ? ORDER BY seq ASC FOR UPDATE`, roleID)
	if err != nil {
		return fmt.Errorf("normal shop transaction: lock bag: %w", err)
	}
	storedBag := make([]normalShopPersistedBagRow, 0, len(beforeBag))
	for rows.Next() {
		var item normalShopPersistedBagRow
		var name, equipType sql.NullString
		var artPack, hardiness, maxHardiness, slot sql.NullInt64
		if err := rows.Scan(&item.ConfigID, &item.ItemType, &item.Amount, &item.ViewID, &name, &equipType, &artPack, &hardiness, &maxHardiness, &slot); err != nil {
			rows.Close()
			return fmt.Errorf("normal shop transaction: scan bag: %w", err)
		}
		item.Name, item.EquipType = name.String, equipType.String
		item.ArtPack, item.Hardiness, item.MaxHardiness, item.Slot = int32(artPack.Int64), int32(hardiness.Int64), int32(maxHardiness.Int64), int32(slot.Int64)
		storedBag = append(storedBag, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("normal shop transaction: read bag: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("normal shop transaction: close bag cursor: %w", err)
	}
	if !normalShopSamePersistedBag(storedBag, beforeBag) {
		return errors.New("normal shop transaction: actor bag differs from persisted bag")
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {
		return fmt.Errorf("normal shop transaction: delete prior bag: %w", err)
	}
	for seq, item := range afterBag {
		if item.ConfigID == "" || item.Amount <= 0 || item.Slot <= 0 {
			return fmt.Errorf("normal shop transaction: invalid bag row at index %d", seq)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, roleID, seq, item.Slot, item.ConfigID, item.ItemType, item.Amount, item.ViewID, normalShopNullableText(item.Name), normalShopNullableText(item.EquipType), normalShopNullableInt32(item.ArtPack), normalShopNullableInt32(item.Hardiness), normalShopNullableInt32(item.MaxHardiness)); err != nil {
			return fmt.Errorf("normal shop transaction: insert bag row %d: %w", seq, err)
		}
	}
	encoded, err = json.Marshal(afterCurrency)
	if err != nil {
		return fmt.Errorf("normal shop transaction: encode currency: %w", err)
	}
	result, err := tx.ExecContext(ctx, "UPDATE role_currency SET snapshot = ? WHERE role_id = ?", encoded, roleID)
	if err != nil {
		return fmt.Errorf("normal shop transaction: update currency: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil || changed < 0 || changed > 1 {
		return fmt.Errorf("normal shop transaction: unexpected currency update result %d: %v", changed, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("normal shop transaction: commit: %w", err)
	}
	return nil
}

func normalShopNullableText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func normalShopNullableInt32(n int32) any {
	if n == 0 {
		return nil
	}
	return n
}
