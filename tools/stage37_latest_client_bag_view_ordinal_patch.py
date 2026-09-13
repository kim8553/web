#!/usr/bin/env python3
# Stage37 latest-client bag/view compatibility A/B.
#
# Evidence basis:
# - LIVE player-property ordinal A/B removed the Lua popup and restored movement.
# - Stage37 negotiates a 228-entry 0x09 property table by slice ordinal.
# - Current bag rows still publish high/global property IDs such as 0x0761,
#   0x0766, 0x0767 and create-view properties 0x00E8/0x08A7, all outside
#   the negotiated range. The latest fxnet2 decoder rejects out-of-range
#   property indexes.
# - The currently negotiated table explicitly contains ConfigID at ordinals
#   7 and 105, Amount at 106, and MaxAmount at 110 with matching wire types.
#
# This patch is intentionally narrow: starter bag views only. It strips
# unsupported CreateView properties and converts bag row identity/count fields
# to the exact negotiated ordinals above. Other view types remain unchanged.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_bag_view_ordinal_patch.py <buildtree>")

    root = Path(sys.argv[1])
    probe = root / "cmd" / "protocol-probe"

    helper = probe / "latest_client_bag_view_ordinal_compat.go"
    helper.write_text(r'''package main

// latestClientStarterBagView reports only the 16 bag-container views created by
// starterBagViews().  Skill/inner-power/shop-list views are intentionally not
// changed by this A/B.
func latestClientStarterBagView(viewID uint16) bool {
	switch viewID {
	case 121, 2, 123, 125, 122, 3, 124, 126, 176, 174, 178, 180, 177, 175, 179, 181:
		return true
	default:
		return false
	}
}

// latestClientBagWireProperties publishes only identities/counts that are
// explicitly present in the exact 228-entry table Stage37 negotiates:
//   ordinal 7   ConfigID string
//   ordinal 105 ConfigID string (shop/view binding)
//   ordinal 106 Amount int32
//   ordinal 110 MaxAmount int32
// High/global IDs are used only as semantic sources and are never emitted.
func latestClientBagWireProperties(properties []serverViewProperty) []serverViewProperty {
	var configID string
	var haveConfig bool
	var amount int32
	var haveAmount bool
	var maxAmount int32
	var haveMaxAmount bool

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
		}
		if property.nest != nil && property.nest.subIndex == 0x05A0 && property.nest.text != nil {
			configID = *property.nest.text
			haveConfig = true
		}
	}

	result := make([]serverViewProperty, 0, 4)
	if haveConfig {
		// Both ConfigID ordinals are explicitly negotiated as strings.  The
		// second slot is used by current shop/view Lua while ordinal 7 is the
		// existing compact object identity used by Stage37.
		result = append(result, viewString(7, configID), viewString(105, configID))
	}
	if haveAmount {
		result = append(result, viewInt(106, amount))
	}
	if haveMaxAmount {
		result = append(result, viewInt(110, maxAmount))
	}
	return result
}
''', encoding="utf-8")

    test = probe / "latest_client_bag_view_ordinal_compat_test.go"
    test.write_text(r'''package main

import "testing"

func TestLatestClientStarterBagViewSet(t *testing.T) {
	for _, id := range []uint16{121, 2, 123, 125, 122, 3, 124, 126, 176, 174, 178, 180, 177, 175, 179, 181} {
		if !latestClientStarterBagView(id) {
			t.Fatalf("bag view %d not recognized", id)
		}
	}
	for _, id := range []uint16{40, 41, 43, 45, 46, 47, 48, 61} {
		if latestClientStarterBagView(id) {
			t.Fatalf("non-bag view %d must stay untouched", id)
		}
	}
}

func TestLatestClientBagWirePropertiesUsesOnlyNegotiatedSlots(t *testing.T) {
	props := []serverViewProperty{
		viewString(7, "Game_item_hp_001"),
		viewInt(0x0761, 100),
		viewInt(0x0766, 5),
		viewInt(0x0767, 99),
		viewInt(0x0779, 123),
	}
	got := latestClientBagWireProperties(props)
	if len(got) != 4 {
		t.Fatalf("bag wire property count=%d, want 4", len(got))
	}
	want := []uint16{7, 105, 106, 110}
	for i, index := range want {
		if got[i].index != index {
			t.Fatalf("property[%d].index=%d, want %d", i, got[i].index, index)
		}
	}
	for _, property := range got {
		if property.index >= latestClientPlayerWirePropertyTableCount {
			t.Fatalf("emitted out-of-range property %d >= %d", property.index, latestClientPlayerWirePropertyTableCount)
		}
	}
}

func TestLatestClientBagWirePropertiesRecoversWeaponNestedConfigID(t *testing.T) {
	got := latestClientBagWireProperties([]serverViewProperty{
		viewNest(0x05A0, "weapon_test"),
		viewInt(0x0766, 1),
		viewInt(0x0767, 1),
	})
	if len(got) != 4 || got[0].text == nil || *got[0].text != "weapon_test" || got[1].text == nil || *got[1].text != "weapon_test" {
		t.Fatalf("nested ConfigID normalization failed: %#v", got)
	}
}
''', encoding="utf-8")

    messages = probe / "messages.go"
    text = messages.read_text(encoding="utf-8")
    old = '''func serverCreateViewWithProperties(spec serverViewSpec, properties []serverViewProperty) ([]byte, error) {\n\tif len(properties) > math.MaxUint16 {\n'''
    new = '''func serverCreateViewWithProperties(spec serverViewSpec, properties []serverViewProperty) ([]byte, error) {\n\tif latestClientStarterBagView(spec.ID) {\n\t\t// Capacity is already carried in the CreateView header.  The previous\n\t\t// BaseCap properties (0x00E8/0x08A7) are outside the negotiated 228\n\t\t// property slots and are omitted in this latest-client A/B.\n\t\tproperties = nil\n\t}\n\tif len(properties) > math.MaxUint16 {\n'''
    text = replace_once(text, old, new, "bag CreateView property filter")

    old = '''func serverViewAdd(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {\n\tif len(properties) > math.MaxUint16 {\n'''
    new = '''func serverViewAdd(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {\n\tif latestClientStarterBagView(viewID) {\n\t\tproperties = latestClientBagWireProperties(properties)\n\t}\n\tif len(properties) > math.MaxUint16 {\n'''
    text = replace_once(text, old, new, "bag ViewAdd ordinal normalization")
    messages.write_text(text, encoding="utf-8")

    overlay = probe / "zz_recovered_overlay.go"
    text = overlay.read_text(encoding="utf-8")
    old = '''func serverObjectProperty(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {\n\tif len(properties) > math.MaxUint16 {\n'''
    new = '''func serverObjectProperty(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {\n\tif latestClientStarterBagView(viewID) {\n\t\tproperties = latestClientBagWireProperties(properties)\n\t}\n\tif len(properties) > math.MaxUint16 {\n'''
    text = replace_once(text, old, new, "bag ObjectProperty ordinal normalization")
    overlay.write_text(text, encoding="utf-8")

    print(f"wrote {helper}")
    print(f"wrote {test}")
    print(f"patched {messages}")
    print(f"patched {overlay}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
