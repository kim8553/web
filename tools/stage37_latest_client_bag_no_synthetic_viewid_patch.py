#!/usr/bin/env python3
# Stage37 latest-client bag A/B: remove the synthetic appended ViewID property.
#
# Evidence boundary from 2026-09-14 02:09-02:10 LIVE:
# - GM HTTP timeout fix succeeded and scene re-entry stayed successful.
# - The server persisted and replayed the granted bag item, but the client bag stayed invisible.
# - Current ViewAdd frames contain a synthetic property-table entry ViewID/int32 at ordinal 228.
# - That entry was added by our compatibility overlay; it was NOT in the original 228-entry
#   negotiated table. Current FxGameLogic already maps container views 174/176/178/180 to
#   tool/equip/material/task categories, so the container itself carries the bag category.
# - Keep only already-negotiated ConfigID(7/105), Amount(106), MaxAmount(110) in the atomic
#   counted VIEW_ADD. No new semantic property is introduced.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_bag_no_synthetic_viewid_patch.py <buildtree>")
    probe = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    # Remove only the compatibility block that appended ViewID to the negotiated player table.
    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    block = r'''
	// Current-client GoodsGrid requires ViewID as bag category. Append only so
	// all already LIVE-proven player-property ordinals remain unchanged.
	haveCurrentBagViewID := false
	for _, field := range fields {
		if field.Name == "ViewID" && field.Type == clientdata.WireInt32 {
			haveCurrentBagViewID = true
			break
		}
	}
	if !haveCurrentBagViewID {
		fields = append(fields, clientdata.FieldSpec{Index: 0x0763, Name: "ViewID", Type: clientdata.WireInt32})
	}
'''
    if text.count(block) != 1:
        raise SystemExit(f"synthetic ViewID append block count={text.count(block)}")
    main_go.write_text(text.replace(block, "", 1), encoding="utf-8")

    helper = probe / "latest_client_current_bag_wire.go"
    text = helper.read_text(encoding="utf-8")
    old = '''func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, properties []serverViewProperty) ([][]byte, error) {\n\tviewIDOrdinal, err := latestClientCurrentBagOrdinal("ViewID", clientdata.WireInt32)\n\tif err != nil { return nil, err }\n\twireProperties := latestClientBagWireProperties(properties)\n\tif len(wireProperties) == 0 { return nil, fmt.Errorf("latest-client bag object has no negotiated properties") }\n\tall := make([]serverViewProperty, 0, 1+len(wireProperties))\n\tall = append(all, viewInt(viewIDOrdinal, category))\n\tall = append(all, wireProperties...)\n\tadd, err := latestClientViewAddCounted(viewID, objectIndex, all)\n\tif err != nil { return nil, err }\n\treturn [][]byte{add}, nil\n}\n'''
    new = '''func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, properties []serverViewProperty) ([][]byte, error) {\n\t_ = category // category is already encoded by current container view 174/176/178/180\n\twireProperties := latestClientBagWireProperties(properties)\n\tif len(wireProperties) == 0 { return nil, fmt.Errorf("latest-client bag object has no negotiated properties") }\n\tadd, err := latestClientViewAddCounted(viewID, objectIndex, wireProperties)\n\tif err != nil { return nil, err }\n\treturn [][]byte{add}, nil\n}\n'''
    text = replace_once(text, old, new, "remove synthetic ViewID from bag VIEW_ADD")
    helper.write_text(text, encoding="utf-8")

    # Restore the pre-append player property table count assertion.
    ptest = probe / "latest_client_player_property_ordinal_compat_test.go"
    text = ptest.read_text(encoding="utf-8")
    text = replace_once(text,
        '''\tif latestClientPlayerWirePropertyTableCount != 229 {\n\t\tt.Fatalf("negotiated property table count=%d, want 229 after append-only ViewID", latestClientPlayerWirePropertyTableCount)\n\t}\n''',
        '''\tif latestClientPlayerWirePropertyTableCount != 228 {\n\t\tt.Fatalf("negotiated property table count=%d, want original 228", latestClientPlayerWirePropertyTableCount)\n\t}\n''',
        "player table count restore")
    ptest.write_text(text, encoding="utf-8")

    test = probe / "latest_client_current_bag_wire_test.go"
    text = test.read_text(encoding="utf-8")
    old = '''func TestLatestClientCurrentBagViewIDIsAppendOnlyOrdinal228(t *testing.T) {\n\tif latestClientPlayerWirePropertyTableCount != 229 { t.Fatalf("property table=%d, want 229", latestClientPlayerWirePropertyTableCount) }\n\tgot, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"ViewID", typ:clientdata.WireInt32}]\n\tif !ok || got != 228 { t.Fatalf("ViewID/int32 ordinal=(%d,%v), want (228,true)", got, ok) }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"MaxHP", typ:clientdata.WireInt32}]; got != 30 { t.Fatalf("MaxHP ordinal moved to %d", got) }\n}\n'''
    new = '''func TestLatestClientCurrentBagKeepsOriginalNegotiatedTable(t *testing.T) {\n\tif latestClientPlayerWirePropertyTableCount != 228 { t.Fatalf("property table=%d, want original 228", latestClientPlayerWirePropertyTableCount) }\n\tif _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"ViewID", typ:clientdata.WireInt32}]; ok { t.Fatal("synthetic ViewID must not be negotiated") }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"MaxHP", typ:clientdata.WireInt32}]; got != 30 { t.Fatalf("MaxHP ordinal moved to %d", got) }\n}\n'''
    text = replace_once(text, old, new, "current-bag ViewID table test")

    old = '''func TestLatestClientCurrentBagFramesAtomicFullAdd(t *testing.T) {\n\tprops := []serverViewProperty{viewString(7, "Game_item_hp_001"), viewInt(0x0766, 5), viewInt(0x0767, 30)}\n\tframes, err := latestClientCurrentBagFrames(174, 3, 1, props); if err != nil { t.Fatal(err) }\n\tif len(frames) != 1 { t.Fatalf("frame count=%d, want 1", len(frames)) }\n\tf := frames[0]\n\tif len(f) != 71 || f[0] != 0x18 || binary.LittleEndian.Uint16(f[1:3]) != 174 || binary.LittleEndian.Uint16(f[3:5]) != 3 || binary.LittleEndian.Uint16(f[5:7]) != 5 { t.Fatalf("full VIEW_ADD header/len=%x", f) }\n\tif binary.LittleEndian.Uint16(f[7:9]) != 228 || binary.LittleEndian.Uint32(f[9:13]) != 1 { t.Fatalf("ViewID property=%x", f[7:13]) }\n\tif binary.LittleEndian.Uint16(f[13:15]) != 7 || binary.LittleEndian.Uint32(f[15:19]) != 17 || string(f[19:35]) != "Game_item_hp_001" || f[35] != 0 { t.Fatalf("ConfigID[7] property=%x", f[13:36]) }\n\tif binary.LittleEndian.Uint16(f[36:38]) != 105 || binary.LittleEndian.Uint32(f[38:42]) != 17 || string(f[42:58]) != "Game_item_hp_001" || f[58] != 0 { t.Fatalf("ConfigID[105] property=%x", f[36:59]) }\n\tif binary.LittleEndian.Uint16(f[59:61]) != 106 || binary.LittleEndian.Uint32(f[61:65]) != 5 { t.Fatalf("Amount property=%x", f[59:65]) }\n\tif binary.LittleEndian.Uint16(f[65:67]) != 110 || binary.LittleEndian.Uint32(f[67:71]) != 30 { t.Fatalf("MaxAmount property=%x", f[65:71]) }\n}\n'''
    new = '''func TestLatestClientCurrentBagFramesAtomicNegotiatedAdd(t *testing.T) {\n\tprops := []serverViewProperty{viewString(7, "Game_item_hp_001"), viewInt(0x0766, 5), viewInt(0x0767, 30)}\n\tframes, err := latestClientCurrentBagFrames(174, 3, 1, props); if err != nil { t.Fatal(err) }\n\tif len(frames) != 1 { t.Fatalf("frame count=%d, want 1", len(frames)) }\n\tf := frames[0]\n\tif len(f) != 65 || f[0] != 0x18 || binary.LittleEndian.Uint16(f[1:3]) != 174 || binary.LittleEndian.Uint16(f[3:5]) != 3 || binary.LittleEndian.Uint16(f[5:7]) != 4 { t.Fatalf("negotiated VIEW_ADD header/len=%x", f) }\n\tif binary.LittleEndian.Uint16(f[7:9]) != 7 || binary.LittleEndian.Uint32(f[9:13]) != 17 || string(f[13:29]) != "Game_item_hp_001" || f[29] != 0 { t.Fatalf("ConfigID[7] property=%x", f[7:30]) }\n\tif binary.LittleEndian.Uint16(f[30:32]) != 105 || binary.LittleEndian.Uint32(f[32:36]) != 17 || string(f[36:52]) != "Game_item_hp_001" || f[52] != 0 { t.Fatalf("ConfigID[105] property=%x", f[30:53]) }\n\tif binary.LittleEndian.Uint16(f[53:55]) != 106 || binary.LittleEndian.Uint32(f[55:59]) != 5 { t.Fatalf("Amount property=%x", f[53:59]) }\n\tif binary.LittleEndian.Uint16(f[59:61]) != 110 || binary.LittleEndian.Uint32(f[61:65]) != 30 { t.Fatalf("MaxAmount property=%x", f[59:65]) }\n}\n'''
    text = replace_once(text, old, new, "atomic no-ViewID frame test")
    test.write_text(text, encoding="utf-8")

    print(f"patched {main_go}")
    print(f"patched {helper}")
    print(f"patched {ptest}")
    print(f"patched {test}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
