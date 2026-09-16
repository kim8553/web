package main

import "fmt"

// latestClientNonBagViewOrdinalFamily identifies the post-ClientReady learned/
// progression GameViews whose recovered emitters still carry historical V2/
// global property IDs. FxNet2's current VIEW_ADD decoder indexes the negotiated
// table by ordinal, so those historical IDs are not valid wire ordinals.
func latestClientNonBagViewOrdinalFamily(viewID uint16) bool {
	switch viewID {
	case viewportSkill, viewportNormalAttack, viewportNeiGong, 45, viewportQingGong, 47, 48:
		return true
	default:
		return false
	}
}

// latestClientNonBagViewProperties maps only name/type pairs already present in
// the current negotiated table. The three legacy View43 metadata IDs
// below are deliberately omitted: current FxGameLogic references
// NeiGongLevel/WuXing/BufferID through resource/config paths, while no matching
// negotiated fields exist. Adding new ordinals for them would be speculative.
func latestClientNonBagViewProperties(viewID uint16, properties []serverViewProperty) ([]serverViewProperty, error) {
	if !latestClientNonBagViewOrdinalFamily(viewID) {
		return properties, nil
	}
	result := make([]serverViewProperty, 0, len(properties))
	for _, property := range properties {
		switch property.index {
		case 0x05A0: // historical ConfigID -> current ConfigID/string ordinal 7
			if property.text == nil {
				return nil, fmt.Errorf("view %d legacy ConfigID has non-string value", viewID)
			}
			result = append(result, viewString(7, *property.text))
		case 0x0761: // historical ItemType -> current ItemType/int32 ordinal 205
			if property.int32 == nil {
				return nil, fmt.Errorf("view %d legacy ItemType has non-int32 value", viewID)
			}
			result = append(result, viewInt(205, *property.int32))
		case 0x08BD: // historical StaticData -> current StaticData/int32 ordinal 204
			if property.int32 == nil {
				return nil, fmt.Errorf("view %d legacy StaticData has non-int32 value", viewID)
			}
			result = append(result, viewInt(204, *property.int32))
		case 0x05B9: // historical Level/byte -> current Level/int32 ordinal 6
			if property.byte1 == nil {
				return nil, fmt.Errorf("view %d legacy Level has non-byte value", viewID)
			}
			result = append(result, viewInt(6, int32(*property.byte1)))
		case 0x0823: // historical MaxLevel -> current MaxLevel/int32 ordinal 203
			if property.int32 == nil {
				return nil, fmt.Errorf("view %d legacy MaxLevel has non-int32 value", viewID)
			}
			result = append(result, viewInt(203, *property.int32))
		case 0x02FC: // historical CurFillValue -> current ordinal 176
			if property.int32 == nil {
				return nil, fmt.Errorf("view %d legacy CurFillValue has non-int32 value", viewID)
			}
			result = append(result, viewInt(176, *property.int32))
		case 0x02FD: // historical TotalFillValue -> current ordinal 177
			if property.int32 == nil {
				return nil, fmt.Errorf("view %d legacy TotalFillValue has non-int32 value", viewID)
			}
			result = append(result, viewInt(177, *property.int32))
		case 0x08BC: // historical CanUse/byte -> current CanUse/byte ordinal 225
			if property.byte1 == nil {
				return nil, fmt.Errorf("view %d legacy CanUse has non-byte value", viewID)
			}
			result = append(result, viewByte(225, *property.byte1))
		case 0x07F2: // recovered skill lease value -> current PauseTime/float32 ordinal 212
			if property.int32 != nil {
				result = append(result, viewFloat(212, float32(*property.int32)))
			} else if property.real32 != nil {
				result = append(result, viewFloat(212, *property.real32))
			} else {
				return nil, fmt.Errorf("view %d legacy PauseTime has unsupported value type", viewID)
			}
		case 0x08EC, 0x08C3, 0x08ED:
			if viewID != viewportNeiGong {
				return nil, fmt.Errorf("view %d contains unnegotiated legacy metadata property 0x%04X", viewID, property.index)
			}
			// Current table has no proven matching field; intentionally omit.
		default:
			if int(property.index) >= latestClientPlayerWirePropertyTableCount {
				return nil, fmt.Errorf("view %d contains unmapped property ordinal %d >= negotiated count %d", viewID, property.index, latestClientPlayerWirePropertyTableCount)
			}
			result = append(result, property)
		}
	}
	for _, property := range result {
		if int(property.index) >= latestClientPlayerWirePropertyTableCount {
			return nil, fmt.Errorf("view %d normalized property ordinal %d >= negotiated count %d", viewID, property.index, latestClientPlayerWirePropertyTableCount)
		}
	}
	return result, nil
}
