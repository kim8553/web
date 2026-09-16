package main

// latestClientStarterBagView reports only the 16 bag-container views created by
// starterBagViews().  Skill/inner-power/shop-list views are intentionally not
// changed by this A/B.
// latestClientItemObjectView reports views whose object rows use the exact
// current negotiated item-object subset. VIEWPORT_EQUIP=1 shares the same
// named runtime item properties (ConfigID/ItemType/ViewID/BindStatus) as bag
// objects; historical global item IDs are semantic sources only and cannot be
// sent as current ordinals.
func latestClientItemObjectView(viewID uint16) bool {
	return viewID == 1 || latestClientStarterBagView(viewID)
}

func latestClientStarterBagView(viewID uint16) bool {
	switch viewID {
	case 121, 2, 123, 125, 122, 3, 124, 126, 176, 174, 178, 180, 177, 175, 179, 181:
		return true
	default:
		return false
	}
}

// latestClientBagWireProperties publishes only identities/counts that are
// explicitly present in the exact current negotiated table Stage37 negotiates:
//
//	ordinal 7   ConfigID string
//	ordinal 105 ConfigID string (shop/view binding)
//	ordinal 106 Amount int32
//	ordinal 110 MaxAmount int32
//	ordinal 234 BindStatus int32
//
// High/global IDs are used only as semantic sources and are never emitted.
func latestClientBagWireProperties(properties []serverViewProperty) []serverViewProperty {
	var configID string
	var haveConfig bool
	var amount int32
	var haveAmount bool
	var maxAmount int32
	var haveMaxAmount bool
	var itemType int32
	var haveItemType bool
	var itemViewID int32
	var haveViewID bool
	var bindStatus int32
	var haveBindStatus bool

	for _, property := range properties {
		switch property.index {
		case 7, 105, 0x05A0:
			if property.text != nil {
				configID = *property.text
				haveConfig = true
			}
		case 106, 0x0766:
			if property.int32 != nil {
				amount = *property.int32
				haveAmount = true
			}
		case 110, 0x0767:
			if property.int32 != nil {
				maxAmount = *property.int32
				haveMaxAmount = true
			}
		case 205, 0x0761:
			if property.int32 != nil {
				itemType = *property.int32
				haveItemType = true
			}
		case 228, 0x0763:
			if property.int32 != nil {
				itemViewID = *property.int32
				haveViewID = true
			}
		case 234, 0x076A:
			if property.int32 != nil {
				bindStatus = *property.int32
				haveBindStatus = true
			}
		}
		if property.nest != nil && property.nest.subIndex == 0x05A0 && property.nest.text != nil {
			configID = *property.nest.text
			haveConfig = true
		}
	}

	// Emit in ascending negotiated ordinal order. Existing 0..227 positions are
	// unchanged; ViewID and runtime BindStatus are append-only fields. Ident is built-in and
	// derived by FxNet2 from VIEW_ADD.objectIndex.
	result := make([]serverViewProperty, 0, 7)
	if haveConfig {
		result = append(result, viewString(7, configID), viewString(105, configID))
	}
	if haveAmount {
		result = append(result, viewInt(106, amount))
	}
	if haveMaxAmount {
		result = append(result, viewInt(110, maxAmount))
	}
	if haveItemType {
		result = append(result, viewInt(205, itemType))
	}
	if haveViewID {
		result = append(result, viewInt(228, itemViewID))
	}
	if haveBindStatus {
		result = append(result, viewInt(234, bindStatus))
	}
	return result
}
