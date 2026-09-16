package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuypersist"
)

// persistShopPurchaseMySQLAtomic is the dormant main-package adapter for a
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
	rows := make([]shopbuypersist.BagRow, 0, len(items))
	for seq, item := range items {
		slot := item.Slot
		if slot <= 0 {
			slot = int32(seq + 1)
		}
		rows = append(rows, shopbuypersist.BagRow{
			Seq:          seq,
			Slot:         slot,
			ConfigID:     item.ConfigID,
			ItemType:     item.ItemType,
			Amount:       item.Amount,
			ViewID:       item.ViewID,
			Name:         sql.NullString{String: item.Name, Valid: item.Name != ""},
			EquipType:    sql.NullString{String: item.EquipType, Valid: item.EquipType != ""},
			ArtPack:      sql.NullInt64{Int64: int64(item.ArtPack), Valid: item.ArtPack != 0},
			Hardiness:    sql.NullInt64{Int64: int64(item.Hardiness), Valid: item.Hardiness != 0},
			MaxHardiness: sql.NullInt64{Int64: int64(item.MaxHardiness), Valid: item.MaxHardiness != 0},
			BindStatus:   item.BindStatus,
		})
	}
	return shopbuypersist.Persist(context.Background(), bagStore.db, roleID, rows, encoded)
}
