package shopbuyatomic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SaveBagChecked persists a bag-only change only if the database still matches
// the caller's pre-change bag. It uses the existing migrated roles row to
// serialize with SaveCheckedBag (ordinary purchases). It is opt-in: callers
// of the legacy unconditional bag Save are NOT automatically protected.
func SaveBagChecked(db *sql.DB, roleID uint64, nextRows, expectedBag []Row) error {
	if db == nil || roleID == 0 {
		return errors.New("checked bag: missing database or role")
	}
	for seq, row := range nextRows {
		if row.Slot < 1 || row.Slot > 65535 || row.ConfigID == "" || row.Amount <= 0 {
			return fmt.Errorf("checked bag: invalid resulting row %d", seq)
		}
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin checked bag save: %w", err)
	}
	defer tx.Rollback()
	if err := LockRoleForBagWrite(ctx, tx, roleID); err != nil {
		return err
	}
	if err := checkLockedBag(ctx, tx, roleID, expectedBag); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {
		return fmt.Errorf("clear checked bag: %w", err)
	}
	for seq, row := range nextRows {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, roleID, seq, row.Slot, row.ConfigID, row.ItemType, row.Amount, row.ViewID, row.Name, row.EquipType, row.ArtPack, row.Hardiness, row.MaxHardiness); err != nil {
			return fmt.Errorf("insert checked bag row %d: %w", seq, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit checked bag save: %w", err)
	}
	return nil
}
