package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/local/9yin-go-server/internal/role"
)

const (
	shopBuyAtomicDeleteBagSQL = "DELETE FROM role_bag_items WHERE role_id = ?"
	shopBuyAtomicInsertBagSQL = `
INSERT INTO role_bag_items(role_id, seq, slot, config_id, item_type, amount, view_id, name, equip_type, art_pack, hardiness, max_hardiness, bind_status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	shopBuyAtomicDeleteCurrencySQL = "DELETE FROM role_currency WHERE role_id = ?"
	shopBuyAtomicInsertCurrencySQL = "INSERT INTO role_currency(role_id, snapshot) VALUES (?, ?)"
)

// persistShopPurchaseMySQLAtomic is the dormant atomic persistence target for a
// regular NPC shop purchase. It deliberately accepts only the concrete MySQL
// stores and requires both stores to share the exact same *sql.DB. JSON fallback
// and split-DB configurations therefore fail closed instead of pretending two
// independent saves are atomic.
//
// This function does not authorize any client selector and does not mutate live
// player state. Production wiring remains gated by exact-current wire authority
// and shopbuyexecute.PersistFirst.
func persistShopPurchaseMySQLAtomic(roleID role.RoleID, bagStore *mysqlBagStore, currencyStore *mysqlCurrencyStore, items []bagItem, value currencySnapshot) error {
	if roleID == 0 {
		return errors.New("shop buy atomic persist: zero role id")
	}
	if bagStore == nil || bagStore.db == nil {
		return errors.New("shop buy atomic persist: mysql bag store unavailable")
	}
	if currencyStore == nil || currencyStore.db == nil {
		return errors.New("shop buy atomic persist: mysql currency store unavailable")
	}
	if bagStore.db != currencyStore.db {
		return errors.New("shop buy atomic persist: shared mysql db required")
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := bagStore.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, shopBuyAtomicDeleteBagSQL, roleID); err != nil {
		return err
	}
	for seq, item := range items {
		slot := item.Slot
		if slot <= 0 {
			slot = int32(seq + 1)
		}
		if _, err := tx.ExecContext(ctx, shopBuyAtomicInsertBagSQL,
			roleID,
			seq,
			slot,
			item.ConfigID,
			item.ItemType,
			item.Amount,
			item.ViewID,
			nullableString(item.Name),
			nullableString(item.EquipType),
			nullableInt32(item.ArtPack),
			nullableInt32(item.Hardiness),
			nullableInt32(item.MaxHardiness),
			item.BindStatus,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, shopBuyAtomicDeleteCurrencySQL, roleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, shopBuyAtomicInsertCurrencySQL, roleID, encoded); err != nil {
		return err
	}
	return tx.Commit()
}
