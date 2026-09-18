package main

// currentShopPageSize is the exact current share/trade/shopconfig.ini PageNum.
// The current client uses this property both for page membership and for the
// per-page grid index derived from each view object's built-in Ident.
const currentShopPageSize int32 = 500

// latestClientShopViewProperties uses the exact ordinals negotiated in the
// current 0x09 table and the current shopconfig.ini page size.
func latestClientShopViewProperties(shopID string, shopType, pageCount int32) []serverViewProperty {
	return []serverViewProperty{
		viewString(101, shopID),           // ShopID/string
		viewInt(102, currentShopPageSize), // PageNum/int32 (current shopconfig.ini)
		viewInt(103, pageCount),           // PageCount/int32
		viewInt(104, shopType),            // ShopType/int32
	}
}

// latestClientShopItemProperties emits only current negotiated name/type pairs.
// Ordinal 181 is MaxPowerValue/int32 in the current table and must never carry a
// ConfigID string. ConfigID is negotiated as string at ordinals 7 and 105; the
// same value is emitted at both, matching the current bag/view identity contract.
func latestClientShopItemProperties(item shopCatalogItem) []serverViewProperty {
	properties := []serverViewProperty{
		viewString(7, item.configID),
		viewString(105, item.configID),
		viewInt(106, item.amount),
		viewInt(107, 0),
		viewInt(108, 0),
		viewInt(109, 0),
		viewInt(110, 99),
	}
	switch item.priceMode {
	case 0:
		properties[3] = viewInt(107, item.price)
	case 1:
		properties[4] = viewInt(108, item.price)
	case 2:
		properties[5] = viewInt(109, item.price)
	case 3:
		// Current shop.ini mode 3 rows are overwhelmingly exchange rows and
		// carry their exchange definition id in column 7. Exact current
		// FxGameLogic queries ExchangeData as int32; ordinal 233 is appended
		// to our negotiated 0x09 table without moving existing ordinals.
		if item.exchangeData > 0 {
			properties = append(properties, viewInt(233, item.exchangeData))
		}
	}
	return properties
}

// currentShopAuthoredCoordinates converts current form_shop.lua wire
// coordinates to the current shop.ini coordinates. The numeric INI key is the
// zero-based page, while the client sends the selected grid index + 1 as pos.
func currentShopAuthoredCoordinates(wirePage, wirePos int32) (page, position int32, ok bool) {
	if wirePage < 0 || wirePos < 1 {
		return 0, 0, false
	}
	return wirePage, wirePos - 1, true
}

// currentShopViewObjectIndex is the built-in Ident/objectIndex expected by the
// current form_shop.lua paging formula:
//
//	PageNum*curpage <= Ident-1 < PageNum*(curpage+1)
//	grid_index = Ident % PageNum - 1
//
// With current shop.ini's zero-based page key and zero-based position, the
// unique mapping is page*PageNum + position + 1.
func currentShopViewObjectIndex(item shopCatalogItem) (uint16, bool) {
	if item.page < 0 || item.position < 0 || item.position >= currentShopPageSize {
		return 0, false
	}
	index := int64(item.page)*int64(currentShopPageSize) + int64(item.position) + 1
	if index < 1 || index > 0xffff {
		return 0, false
	}
	return uint16(index), true
}

func currentShopListing(items []shopCatalogItem, wirePage, wirePos int32) *shopCatalogItem {
	page, position, ok := currentShopAuthoredCoordinates(wirePage, wirePos)
	if !ok {
		return nil
	}
	for i := range items {
		if items[i].page == page && items[i].position == position {
			// openShopLocked publishes ordinary catalog rows only when their
			// object index fits the current client view. Never accept a buy
			// for a row that the same shop view cannot publish.
			if _, representable := currentShopViewObjectIndex(items[i]); !representable {
				return nil
			}
			return &items[i]
		}
	}
	return nil
}
