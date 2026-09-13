#!/usr/bin/env python3
# Stage37 latest-client item-schema + scene-exit-drain A/B.
#
# Evidence basis from LIVE 2026-09-13 21:30-21:31:
# - The server restored/persisted Game_item_hp_001 and emitted ViewAdd rows.
# - Latest client trace reported ServerViewAdd property errors during the initial
#   post-ready batch, but did NOT emit new ServerViewAdd property errors when the
#   live GM refresh re-sent the current 4-field bag row.
# - Therefore the current four ordinals are syntactically accepted, but they are
#   insufficient for the latest GoodsGrid to materialize a normal tool item.
# - The recovered official item property table and item struct evidence provide
#   the concrete item fields/types used by the existing bagItemProps path.
# - Scene switch trace showed SERVER_EXIT_SCENE, then SERVER_ENTRY_SCENE entered
#   while exit was still active ("wait exit_scene"), and ExecuteReceiveEntryScene
#   happened ~2.34 s after exit began. The current server waits only 750 ms.
#
# This probe:
# 1) extends the negotiated property-name/type table with evidence-backed item
#    fields that the current bag path actually emits;
# 2) remaps those bag properties to negotiated ordinals dynamically, preserving
#    their authored/runtime values instead of dropping them;
# 3) changes only the scene-switch ExitScene->EntryScene A/B delay from 750 ms to
#    2750 ms so EntryScene is sent after the observed latest-client exit drain;
# 4) preserves the prior View 2 delete+recreate live refresh and still requires
#    the real target ClientReady (0x09).
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def insert_before_scene_visible_return(text: str, block: str) -> str:
    start = text.find("func sceneVisiblePropertyFields(")
    if start < 0:
        raise SystemExit("sceneVisiblePropertyFields: function not found")
    ret = text.find("\treturn fields\n}", start)
    if ret < 0:
        raise SystemExit("sceneVisiblePropertyFields: return anchor not found")
    return text[:ret] + block + text[ret:]


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_item_schema_reentry_drain_patch.py <buildtree>")
    root = Path(sys.argv[1])
    probe = root / "cmd" / "protocol-probe"

    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    block = r'''
	// Latest-client item-view A/B. These names/types are taken from the recovered
	// item/item.xml + tool_item.xml property evidence and the official property
	// table used by the current bagItemProps path. Append only missing exact
	// name/type pairs so all existing player/NPC ordinals stay stable.
	itemViewFields := []clientdata.FieldSpec{
		{Index: 0x0762, Name: "TextureType", Type: clientdata.WireInt32},
		{Index: 0x0763, Name: "ViewID", Type: clientdata.WireInt32},
		{Index: 0x0764, Name: "ColorLevel", Type: clientdata.WireInt32},
		{Index: 0x0765, Name: "UniqueID", Type: clientdata.WireString},
		{Index: 0x0768, Name: "MaxCount", Type: clientdata.WireInt32},
		{Index: 0x076A, Name: "BindStatus", Type: clientdata.WireInt32},
		{Index: 0x076D, Name: "SellPrice1", Type: clientdata.WireInt32},
		{Index: 0x0779, Name: "LogicPack", Type: clientdata.WireInt32},
		{Index: 0x077A, Name: "ArtPack", Type: clientdata.WireInt32},
		{Index: 0x0786, Name: "IsCanConsign", Type: clientdata.WireInt32},
		{Index: 0x0787, Name: "IsMarketItem", Type: clientdata.WireInt32},
		{Index: 0x0788, Name: "IsCarryItem", Type: clientdata.WireInt32},
		{Index: 0x078F, Name: "Hardiness", Type: clientdata.WireInt32},
		{Index: 0x0790, Name: "MaxHardiness", Type: clientdata.WireInt32},
		{Index: 0x0791, Name: "NeedSex", Type: clientdata.WireByte},
		{Index: 0x0792, Name: "EquipType", Type: clientdata.WireString},
		{Index: 0x07DA, Name: "FuncPack", Type: clientdata.WireInt32},
		{Index: 0x07E7, Name: "PropModifyPack", Type: clientdata.WireInt32},
	}
	for _, candidate := range itemViewFields {
		found := false
		for _, existing := range fields {
			if existing.Name == candidate.Name && existing.Type == candidate.Type {
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, candidate)
		}
	}
'''
    text = insert_before_scene_visible_return(text, block)
    main_go.write_text(text, encoding="utf-8")

    helper = probe / "latest_client_bag_view_ordinal_compat.go"
    helper.write_text(r'''package main

import "github.com/local/9yin-go-server/internal/clientdata"

func latestClientStarterBagView(viewID uint16) bool {
	switch viewID {
	case 121, 2, 123, 125, 122, 3, 124, 126, 176, 174, 178, 180, 177, 175, 179, 181:
		return true
	default:
		return false
	}
}

func latestClientBagPropertyKey(index uint16) (latestClientPlayerPropertyKey, bool) {
	switch index {
	case 7, 105, 0x05A0:
		return latestClientPlayerPropertyKey{name: "ConfigID", typ: clientdata.WireString}, true
	case 0x0761:
		return latestClientPlayerPropertyKey{name: "ItemType", typ: clientdata.WireInt32}, true
	case 0x0762:
		return latestClientPlayerPropertyKey{name: "TextureType", typ: clientdata.WireInt32}, true
	case 0x0763:
		return latestClientPlayerPropertyKey{name: "ViewID", typ: clientdata.WireInt32}, true
	case 0x0764:
		return latestClientPlayerPropertyKey{name: "ColorLevel", typ: clientdata.WireInt32}, true
	case 0x0765:
		return latestClientPlayerPropertyKey{name: "UniqueID", typ: clientdata.WireString}, true
	case 0x0766:
		return latestClientPlayerPropertyKey{name: "Amount", typ: clientdata.WireInt32}, true
	case 0x0767:
		return latestClientPlayerPropertyKey{name: "MaxAmount", typ: clientdata.WireInt32}, true
	case 0x0768:
		return latestClientPlayerPropertyKey{name: "MaxCount", typ: clientdata.WireInt32}, true
	case 0x076A:
		return latestClientPlayerPropertyKey{name: "BindStatus", typ: clientdata.WireInt32}, true
	case 0x076D:
		return latestClientPlayerPropertyKey{name: "SellPrice1", typ: clientdata.WireInt32}, true
	case 0x0779:
		return latestClientPlayerPropertyKey{name: "LogicPack", typ: clientdata.WireInt32}, true
	case 0x077A:
		return latestClientPlayerPropertyKey{name: "ArtPack", typ: clientdata.WireInt32}, true
	case 0x0786:
		return latestClientPlayerPropertyKey{name: "IsCanConsign", typ: clientdata.WireInt32}, true
	case 0x0787:
		return latestClientPlayerPropertyKey{name: "IsMarketItem", typ: clientdata.WireInt32}, true
	case 0x0788:
		return latestClientPlayerPropertyKey{name: "IsCarryItem", typ: clientdata.WireInt32}, true
	case 0x078F:
		return latestClientPlayerPropertyKey{name: "Hardiness", typ: clientdata.WireInt32}, true
	case 0x0790:
		return latestClientPlayerPropertyKey{name: "MaxHardiness", typ: clientdata.WireInt32}, true
	case 0x0791:
		return latestClientPlayerPropertyKey{name: "NeedSex", typ: clientdata.WireByte}, true
	case 0x0792:
		return latestClientPlayerPropertyKey{name: "EquipType", typ: clientdata.WireString}, true
	case 0x07DA:
		return latestClientPlayerPropertyKey{name: "FuncPack", typ: clientdata.WireInt32}, true
	case 0x07E7:
		return latestClientPlayerPropertyKey{name: "PropModifyPack", typ: clientdata.WireInt32}, true
	default:
		return latestClientPlayerPropertyKey{}, false
	}
}

func latestClientBagWireProperties(properties []serverViewProperty) []serverViewProperty {
	result := make([]serverViewProperty, 0, len(properties))
	seen := make(map[uint16]struct{}, len(properties))
	for _, property := range properties {
		if property.nest != nil && property.nest.subIndex == 0x05A0 && property.nest.text != nil {
			key := latestClientPlayerPropertyKey{name: "ConfigID", typ: clientdata.WireString}
			ordinal, ok := latestClientPlayerWirePropertyOrdinals[key]
			if !ok {
				continue
			}
			if _, duplicate := seen[ordinal]; duplicate {
				continue
			}
			result = append(result, viewString(ordinal, *property.nest.text))
			seen[ordinal] = struct{}{}
			continue
		}
		key, ok := latestClientBagPropertyKey(property.index)
		if !ok {
			continue
		}
		ordinal, ok := latestClientPlayerWirePropertyOrdinals[key]
		if !ok {
			continue
		}
		if _, duplicate := seen[ordinal]; duplicate {
			continue
		}
		property.index = ordinal
		result = append(result, property)
		seen[ordinal] = struct{}{}
	}
	return result
}
''', encoding="utf-8")

    test = probe / "latest_client_bag_view_ordinal_compat_test.go"
    test.write_text(r'''package main

import (
	"testing"
	"github.com/local/9yin-go-server/internal/clientdata"
)

func TestLatestClientItemSchemaContainsCoreGoodsGridFields(t *testing.T) {
	keys := []latestClientPlayerPropertyKey{
		{name: "ConfigID", typ: clientdata.WireString},
		{name: "ItemType", typ: clientdata.WireInt32},
		{name: "TextureType", typ: clientdata.WireInt32},
		{name: "ViewID", typ: clientdata.WireInt32},
		{name: "ColorLevel", typ: clientdata.WireInt32},
		{name: "UniqueID", typ: clientdata.WireString},
		{name: "Amount", typ: clientdata.WireInt32},
		{name: "MaxAmount", typ: clientdata.WireInt32},
		{name: "LogicPack", typ: clientdata.WireInt32},
		{name: "FuncPack", typ: clientdata.WireInt32},
		{name: "PropModifyPack", typ: clientdata.WireInt32},
	}
	for _, key := range keys {
		if _, ok := latestClientPlayerWirePropertyOrdinals[key]; !ok {
			t.Fatalf("missing negotiated item property %s/%s", key.name, key.typ)
		}
	}
	if latestClientPlayerWirePropertyTableCount <= 228 {
		t.Fatalf("item-schema A/B did not extend negotiated table: %d", latestClientPlayerWirePropertyTableCount)
	}
}

func TestLatestClientBagWirePropertiesPreservesCoreToolItemState(t *testing.T) {
	props := []serverViewProperty{
		viewString(7, "Game_item_hp_001"),
		viewInt(0x0761, 1),
		viewInt(0x0762, 0),
		viewInt(0x0763, 1),
		viewInt(0x0764, 1),
		viewString(0x0765, "uid"),
		viewInt(0x0766, 5),
		viewInt(0x0767, 30),
		viewInt(0x0779, 7),
		viewInt(0x07DA, 9),
		viewInt(0x07E7, 11),
	}
	got := latestClientBagWireProperties(props)
	if len(got) < 10 {
		t.Fatalf("core tool-item properties collapsed: got=%d %#v", len(got), got)
	}
	for _, property := range got {
		if int(property.index) >= latestClientPlayerWirePropertyTableCount {
			t.Fatalf("wire property %d outside negotiated count %d", property.index, latestClientPlayerWirePropertyTableCount)
		}
	}
}
''', encoding="utf-8")

    player_test = probe / "latest_client_player_property_ordinal_compat_test.go"
    text = player_test.read_text(encoding="utf-8")
    old = '''\tif latestClientPlayerWirePropertyTableCount != 228 {\n\t\tt.Fatalf("negotiated property table count=%d, want 228", latestClientPlayerWirePropertyTableCount)\n\t}\n'''
    new = '''\tif latestClientPlayerWirePropertyTableCount < 228 {\n\t\tt.Fatalf("negotiated property table count=%d, want at least 228", latestClientPlayerWirePropertyTableCount)\n\t}\n'''
    text = replace_once(text, old, new, "player table-count test")
    player_test.write_text(text, encoding="utf-8")

    trans = probe / "scene_transition.go"
    text = trans.read_text(encoding="utf-8")
    if '"log"' not in text:
        text = replace_once(text, 'import (\n\t"encoding/binary"\n', 'import (\n\t"encoding/binary"\n\t"log"\n', "scene transition log import")
    text = replace_once(
        text,
        '\ttime.Sleep(750 * time.Millisecond)\n',
        '\tlog.Printf("latest-client REENTRY-EXIT-DRAIN waiting 2750ms before EntryScene scene=%s resource=%s", destination.location.Scene.Config, destination.location.Scene.Resource)\n'
        '\ttime.Sleep(2750 * time.Millisecond)\n'
        '\tlog.Printf("latest-client REENTRY-EXIT-DRAIN complete; sending EntryScene scene=%s resource=%s", destination.location.Scene.Config, destination.location.Scene.Resource)\n',
        "scene exit-drain delay",
    )
    trans.write_text(text, encoding="utf-8")

    overlay = probe / "zz_recovered_overlay.go"
    text = overlay.read_text(encoding="utf-8")
    old = '''\t\tlog.Printf("%s: grant bag item=%s view=%d slot=%d frame=%x", remote, item.ConfigID, view, uint16(slot), frame)\n'''
    new = '''\t\tlog.Printf("%s: grant bag item=%s item_type=%d item_view_id=%d texture_type=%d color=%d amount=%d max_amount=%d logic_pack=%d func_pack=%d prop_modify_pack=%d view=%d slot=%d property_table=%d frame=%x", remote, item.ConfigID, item.ItemType, item.ViewID, item.TextureType, item.ColorLevel, item.Amount, item.MaxAmount, item.LogicPack, item.FuncPack, item.PropModifyPack, view, uint16(slot), latestClientPlayerWirePropertyTableCount, frame)\n'''
    text = replace_once(text, old, new, "bag grant diagnostic")
    overlay.write_text(text, encoding="utf-8")

    print(f"patched {main_go}")
    print(f"rewrote {helper}")
    print(f"rewrote {test}")
    print(f"patched {player_test}")
    print(f"patched {trans}")
    print(f"patched {overlay}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
