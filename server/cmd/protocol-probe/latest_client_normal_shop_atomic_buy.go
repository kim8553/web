package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// normalShopPersistedRows projects precisely the ten fields written by the
// existing mysqlBagStore. Actor snapshots MUST be serialized by player.mu.
func normalShopPersistedRows(items []bagItem) []normalShopPersistedBagRow {
	rows := make([]normalShopPersistedBagRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, normalShopPersistedBagRow{
			ConfigID: item.ConfigID, ItemType: item.ItemType, Amount: item.Amount,
			ViewID: item.ViewID, Slot: item.Slot, Name: item.Name,
			EquipType: item.EquipType, ArtPack: item.ArtPack,
			Hardiness: item.Hardiness, MaxHardiness: item.MaxHardiness,
		})
	}
	return rows
}

// normalShopSingleSlot uses only the advertised initial open-slot count. It
// does not assume all structural view capacity has been unlocked. No stacking
// or sale-price semantics are inferred for this initial integration.
func normalShopSingleSlot(items []bagItem, view uint16) (int32, bool) {
	var capacity int32
	for _, spec := range starterBagViews() {
		if spec.ID == view {
			capacity = int32(spec.BaseCap)
			break
		}
	}
	if capacity <= 0 {
		return 0, false
	}
	used := make(map[int32]struct{}, len(items))
	for _, item := range items {
		if bagViewForViewID(item.ViewID) != view {
			continue
		}
		if item.Slot <= 0 || item.Slot > capacity {
			return 0, false // unverified or inconsistent current container layout
		}
		if _, exists := used[item.Slot]; exists {
			return 0, false
		}
		used[item.Slot] = struct{}{}
	}
	for slot := int32(1); slot <= capacity; slot++ {
		if _, exists := used[slot]; !exists {
			return slot, true
		}
	}
	return 0, false
}

func normalShopFrameForCurrency(value normalShopPersistedCurrency) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 417, Name: "CapitalType1", Value: clientdata.Int64Value(int64(value.Silver))},
		{Index: 416, Name: "CapitalType0", Value: clientdata.Int64Value(int64(value.Gold))},
		{Index: 418, Name: "CapitalType2", Value: clientdata.Int64Value(int64(value.SilverCard))},
		{Index: 420, Name: "CapitalType4", Value: clientdata.Int64Value(int64(value.SilverTicket))},
	})
}

// normalShopAtomicBuy accepts the exact-current normal shop request shape and
// persists it through whichever storage backend the 9yin-go-server1 runtime is
// actually using: one SQL transaction for MySQL, or the guarded paired JSON
// commit for the native no-DSN runtime. No actor/client-visible state changes
// occur until persistence succeeds.
func normalShopAtomicBuy(link sceneMessageConnection, player *playerActor, world *sceneLifecycle,
	itemCatalog *itemCatalog, equipCatalog *equipCatalog, bagStore bagStoreIface,
	currencyStore currencyStoreIface, roleID role.RoleID, custom clientCustomMessage,
	remote string) (bool, error) {
	if len(custom.Values) != 5 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x46 {
		return false, nil
	}
	if player == nil || world == nil || roleID == 0 || itemCatalog == nil || equipCatalog == nil {
		log.Printf("%s: normal buy blocked: missing active role, actor, scene or catalog", remote)
		return true, nil
	}
	if (custom.Values[1].Type != 6 && custom.Values[1].Type != 7) || custom.Values[2].Type != 2 || custom.Values[3].Type != 2 || custom.Values[4].Type != 2 {
		log.Printf("%s: normal buy blocked: invalid field types", remote)
		return true, nil
	}
	shopID, page, pos, quantity := custom.Values[1].Text, custom.Values[2].Int32, custom.Values[3].Int32, custom.Values[4].Int32
	if shopID == "" || page < 0 || pos <= 0 || quantity != 1 {
		log.Printf("%s: normal buy blocked: unsupported coordinates or quantity shop=%q page=%d pos=%d count=%d", remote, shopID, page, pos, quantity)
		return true, nil
	}
	if !world.currentShopExchangeSessionAllows(shopID) {
		log.Printf("%s: normal buy blocked: no current matching NPC shop session shop=%q", remote, shopID)
		return true, nil
	}

	bagSQL, bagSQLOK := bagStore.(*mysqlBagStore)
	currencySQL, currencySQLOK := currencyStore.(*mysqlCurrencyStore)
	mysqlMode := bagSQLOK && currencySQLOK && bagSQL != nil && currencySQL != nil && bagSQL.db != nil && bagSQL.db == currencySQL.db
	bagJSON, bagJSONOK := bagStore.(*bagStore)
	currencyJSON, currencyJSONOK := currencyStore.(*currencyStore)
	jsonMode := bagJSONOK && currencyJSONOK && bagJSON != nil && currencyJSON != nil
	if !mysqlMode && !jsonMode {
		log.Printf("%s: normal buy blocked: bag/currency stores are not one supported backend", remote)
		return true, nil
	}

	listings, _, _, err := shopCatalogItems(defaultShopINIPath, shopID)
	if err != nil {
		log.Printf("%s: normal buy blocked: shop catalog: %v", remote, err)
		return true, nil
	}
	listing := currentShopListing(listings, page, pos)
	if listing == nil || listing.priceMode < 0 || listing.priceMode > 2 || listing.configID == "" {
		log.Printf("%s: normal buy blocked: missing/unsupported listing shop=%q page=%d pos=%d", remote, shopID, page, pos)
		return true, nil
	}
	total, safe := normalShopSafeTotal(listing.price, quantity)
	if !safe {
		log.Printf("%s: normal buy blocked: invalid price or total shop=%q item=%q", remote, shopID, listing.configID)
		return true, nil
	}
	_, toolKnown := itemCatalog.byID[listing.configID]
	_, equipKnown := equipCatalog.byID[listing.configID]
	if !toolKnown && !equipKnown {
		log.Printf("%s: normal buy blocked: unknown item %q", remote, listing.configID)
		return true, nil
	}
	reward := enrichBagItem(bagItem{ConfigID: listing.configID, Amount: quantity}, itemCatalog, equipCatalog)
	view := bagViewForViewID(reward.ViewID)
	if view == 0 || reward.ItemType <= 0 || (reward.MaxAmount > 0 && quantity > reward.MaxAmount) {
		log.Printf("%s: normal buy blocked: unsupported item view/type/stack item=%q view=%d", remote, listing.configID, view)
		return true, nil
	}

	// Keep the actor lock across snapshot creation, persistence, and the
	// in-memory update; regular actor mutations also take this mutex. We never
	// call an actor method that re-locks it while held.
	var frames [][]byte
	storageMode := "JSON"
	committed, commitErr := func() (bool, error) {
		player.mu.Lock()
		defer player.mu.Unlock()

		beforeCurrency := normalShopPersistedCurrency{
			Silver: player.silver, Gold: player.gold,
			SilverCard: player.silverCard, SilverTicket: player.silverTicket,
		}
		afterCurrency := beforeCurrency
		switch listing.priceMode {
		case 0:
			if int64(afterCurrency.Gold) < total {
				return false, nil
			}
			afterCurrency.Gold -= int32(total)
		case 1:
			if int64(afterCurrency.Silver) < total {
				return false, nil
			}
			afterCurrency.Silver -= int32(total)
		case 2:
			if int64(afterCurrency.SilverCard) < total {
				return false, nil
			}
			afterCurrency.SilverCard -= int32(total)
		}
		slot, valid := normalShopSingleSlot(player.bagItems, view)
		if !valid {
			log.Printf("%s: normal buy blocked: full or inconsistent bag view=%d", remote, view)
			return false, nil
		}
		reward.Slot = slot
		currencyFrame, frameErr := normalShopFrameForCurrency(afterCurrency)
		if frameErr != nil {
			return false, fmt.Errorf("encode purchase currency before commit: %w", frameErr)
		}
		itemFrames, frameErr := latestClientCurrentBagFrames(view, uint16(slot), reward.ViewID, bagItemProps(view, reward))
		if frameErr != nil {
			return false, fmt.Errorf("encode purchase bag before commit: %w", frameErr)
		}
		frames = make([][]byte, 0, 1+len(itemFrames))
		frames = append(frames, currencyFrame)
		frames = append(frames, itemFrames...)

		afterBagItems := append(append([]bagItem(nil), player.bagItems...), reward)
		if mysqlMode {
			storageMode = "MySQL"
			beforeBag := normalShopPersistedRows(player.bagItems)
			afterBag := normalShopPersistedRows(afterBagItems)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if txErr := normalShopCommitMySQLSnapshots(ctx, bagSQL.db, uint64(roleID), beforeCurrency, afterCurrency, beforeBag, afterBag); txErr != nil {
				return false, fmt.Errorf("normal buy MySQL transaction: %w", txErr)
			}
		} else {
			if txErr := normalShopCommitJSONSnapshots(bagJSON, currencyJSON, roleID, afterCurrency, afterBagItems); txErr != nil {
				return false, fmt.Errorf("normal buy JSON transaction: %w", txErr)
			}
		}

		player.silver, player.gold, player.silverCard, player.silverTicket = afterCurrency.Silver, afterCurrency.Gold, afterCurrency.SilverCard, afterCurrency.SilverTicket
		player.bagItems = afterBagItems
		return true, nil
	}()
	if commitErr != nil {
		log.Printf("%s: normal buy aborted; disconnect for authoritative state reload: %v", remote, commitErr)
		return true, commitErr
	}
	if !committed {
		return true, nil
	}
	if err := writeFrames(link, frames...); err != nil {
		return true, fmt.Errorf("normal buy committed but client update failed; reconnect required: %w", err)
	}
	log.Printf("%s: normal buy %s committed shop=%s item=%s count=1 mode=%d cost=%d view=%d", remote, storageMode, shopID, reward.ConfigID, listing.priceMode, total, view)
	return true, nil
}
