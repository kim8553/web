package shopbuypersist

import (
	"context"
	"database/sql"
	"errors"

	"github.com/local/9yin-go-server/internal/role"
)

const (
	deleteBagSQL = "DELETE FROM role_bag_items WHERE role_id = ?"
	insertBagSQL = `
INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness, bind_status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	deleteCurrencySQL = "DELETE FROM role_currency WHERE role_id = ?"
	insertCurrencySQL = "INSERT INTO role_currency(role_id, snapshot) VALUES (?, ?)"
)

// BagRow is the already-staged persistent representation of one bag row.
// Optional SQL values are kept as database/sql nullable values so callers can
// preserve the same NULL semantics as the current mysqlBagStore.Save path.
type BagRow struct {
	Seq          int
	Slot         int32
	ConfigID     string
	ItemType     int32
	Amount       int32
	ViewID       int32
	Name         sql.NullString
	EquipType    sql.NullString
	ArtPack      sql.NullInt64
	Hardiness    sql.NullInt64
	MaxHardiness sql.NullInt64
	BindStatus   int32
}

// Persist commits bag rows and the serialized currency snapshot through one
// transaction on one DB handle. It does not stage gameplay state, authorize a
// protocol selector, mutate live state, or publish client frames.
func Persist(ctx context.Context, db *sql.DB, roleID role.RoleID, rows []BagRow, currencySnapshot []byte) error {
	if db == nil {
		return errors.New("shop buy atomic persist: db unavailable")
	}
	if roleID == 0 {
		return errors.New("shop buy atomic persist: zero role id")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, deleteBagSQL, roleID); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, insertBagSQL,
			roleID,
			row.Seq,
			row.Slot,
			row.ConfigID,
			row.ItemType,
			row.Amount,
			row.ViewID,
			row.Name,
			row.EquipType,
			row.ArtPack,
			row.Hardiness,
			row.MaxHardiness,
			row.BindStatus,
		); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, deleteCurrencySQL, roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, insertCurrencySQL, roleID, currencySnapshot); err != nil {
		return err
	}
	return tx.Commit()
}
