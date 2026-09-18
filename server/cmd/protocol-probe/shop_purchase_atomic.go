package main

import (
	"encoding/json"
	"fmt"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuyatomic"
)

// persistOrdinaryShopPurchase uses the actual shared game database. Legacy
// JSON stores persist two separate files and cannot offer this guarantee;
// reject that mode rather than charging or reporting an uncommitted purchase.
func persistOrdinaryShopPurchase(bagStore bagStoreIface, currencyStore currencyStoreIface, roleID role.RoleID, items []bagItem, beforeWallet, afterWallet currencySnapshot) error {
	bag, bagOK := bagStore.(*mysqlBagStore)
	currency, currencyOK := currencyStore.(*mysqlCurrencyStore)
	if !bagOK || !currencyOK || bag == nil || currency == nil || bag.db == nil || currency.db != bag.db {
		return fmt.Errorf("shop purchase requires bag and currency stores on one MySQL database; JSON purchases are not transaction-safe")
	}
	if roleID == 0 {
		return fmt.Errorf("shop purchase has no role id")
	}
	rows := make([]shopbuyatomic.Row, 0, len(items))
	for _, item := range items {
		rows = append(rows, shopbuyatomic.Row{
			Slot: item.Slot, ConfigID: item.ConfigID, ItemType: item.ItemType,
			Amount: item.Amount, ViewID: item.ViewID, Name: nullableString(item.Name),
			EquipType: nullableString(item.EquipType), ArtPack: nullableInt32(item.ArtPack),
			Hardiness: nullableInt32(item.Hardiness), MaxHardiness: nullableInt32(item.MaxHardiness),
		})
	}
	expectedJSON, err := json.Marshal(beforeWallet)
	if err != nil {
		return fmt.Errorf("encode shop starting wallet: %w", err)
	}
	nextJSON, err := json.Marshal(afterWallet)
	if err != nil {
		return fmt.Errorf("encode shop resulting wallet: %w", err)
	}
	return shopbuyatomic.SaveChecked(bag.db, uint64(roleID), rows, expectedJSON, nextJSON)
}

// This is the same four-property currency update already emitted by
// playerActor.currencyProperties, built from the prospective wallet before
// any persisted or in-memory purchase state is changed.
func ordinaryShopCurrencyFrame(value currencySnapshot) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 417, Name: "CapitalType1", Value: clientdata.Int64Value(int64(value.Silver))},
		{Index: 416, Name: "CapitalType0", Value: clientdata.Int64Value(int64(value.Gold))},
		{Index: 418, Name: "CapitalType2", Value: clientdata.Int64Value(int64(value.SilverCard))},
		{Index: 420, Name: "CapitalType4", Value: clientdata.Int64Value(int64(value.SilverTicket))},
	})
}
