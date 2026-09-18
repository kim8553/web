// Package shopbuyatomic persists a normal NPC purchase in one MySQL transaction.
// It has no client wire contract and does not create or migrate database tables.
package shopbuyatomic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Row contains only the columns already saved by mysqlBagStore.Save.
// Nullable properties are supplied by the existing server conversion helpers.
type Row struct {
	Slot          int32
	ConfigID      string
	ItemType      int32
	Amount        int32
	ViewID        int32
	Name          any
	EquipType     any
	ArtPack       any
	Hardiness     any
	MaxHardiness  any
}

// Save commits the complete bag snapshot and the currency JSON together.
// Any statement or commit failure is returned; the transaction is rolled back
// when possible. Callers must not publish the purchase before Save succeeds.
func Save(db *sql.DB, roleID uint64, rows []Row, currencyJSON []byte) error {
	if db == nil || roleID == 0 || len(currencyJSON) == 0 {
		return errors.New("shop buy: missing database, role, or currency snapshot")
	}
	for seq, row := range rows {
		if row.Slot < 1 || row.Slot > 65535 || row.ConfigID == "" || row.Amount <= 0 {
			return fmt.Errorf("shop buy: invalid bag row %d", seq)
		}
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin shop purchase: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {
		return fmt.Errorf("clear shop bag: %w", err)
	}
	for seq, row := range rows {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, roleID, seq, row.Slot, row.ConfigID, row.ItemType, row.Amount, row.ViewID, row.Name, row.EquipType, row.ArtPack, row.Hardiness, row.MaxHardiness); err != nil {
			return fmt.Errorf("save shop bag row %d: %w", seq, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_currency WHERE role_id = ?", roleID); err != nil {
		return fmt.Errorf("clear shop wallet: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO role_currency(role_id, snapshot) VALUES (?, ?)", roleID, currencyJSON); err != nil {
		return fmt.Errorf("save shop wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit shop purchase: %w", err)
	}
	return nil
}
