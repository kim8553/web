#!/usr/bin/env python3
# Stage37 latest-client legacy main-bag + rich object A/B.
#
# Evidence basis after the NO_SYNTHETIC_VIEWID LIVE failure:
# - GM grant/persistence succeeds; the client still does not materialize rows.
# - Current FxGameLogic GetToolBoxViewport maps item ViewID categories exactly:
#       1 -> 2, 2 -> 121, 3 -> 123, 4 -> 125.
#   174/176/178/180 are a separate IsNewViewPort family, not the canonical
#   GetToolBoxViewport result used by normal bag routing.
# - Current bag UI reads object ViewID, ItemType, ConfigID, Amount and Ident.
# - Ident is read through the current string-property accessor; ViewID and
#   ItemType are integer properties.
# - The negotiated Stage37 property table is slice-ordinal based. ItemType is
#   already present at ordinal 205. Append only ViewID/int32 then Ident/string,
#   preserving every existing ordinal 0..227.
# - Recovered rich bag rows already carry semantic sources 0x0761(ItemType),
#   0x0763(ViewID), 0x0765(unique identity string), 0x0766(Amount),
#   0x0767(MaxAmount). This A/B maps only those evidence-backed object fields.
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
        raise SystemExit("usage: stage37_latest_client_legacy_bag_rich_object_patch.py <buildtree>")
    probe = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    # Append only object fields that the current bag UI statically reads and
    # that are absent from the original 228-entry negotiated table.
    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    text = insert_before_scene_visible_return(text, r'''
	// Latest-client canonical bag rows require these object properties by name.
	// Append only: preserve all original negotiated ordinals 0..227.
	haveBagViewID := false
	haveBagIdent := false
	for _, field := range fields {
		if field.Name == "ViewID" && field.Type == clientdata.WireInt32 {
			haveBagViewID = true
		}
		if field.Name == "Ident" && field.Type == clientdata.WireString {
			haveBagIdent = true
		}
	}
	if !haveBagViewID {
		fields = append(fields, clientdata.FieldSpec{Index: 0x0763, Name: "ViewID", Type: clientdata.WireInt32})
	}
	if !haveBagIdent {
		fields = append(fields, clientdata.FieldSpec{Index: 0x0765, Name: "Ident", Type: clientdata.WireString})
	}
''')

    old_refresh = '''\t\t\tfor _, bagView := range []uint16{174, 176, 178, 180} {\n\t\t\t\tif err := link.WriteFrame(serverDeleteViewCompat(bagView)); err != nil {\n\t\t\t\t\tgmSession.setAction("刷新背包失败：" + err.Error())\n\t\t\t\t\treturn err\n\t\t\t\t}\n\t\t\t}\n\t\t\tlog.Printf("%s: latest-client CURRENT-BAG refresh deleted main Views=174,176,178,180 before authoritative replay", conn.RemoteAddr())\n'''
    new_refresh = '''\t\t\tfor _, bagView := range []uint16{2, 121, 123, 125} {\n\t\t\t\tif err := link.WriteFrame(serverDeleteViewCompat(bagView)); err != nil {\n\t\t\t\t\tgmSession.setAction("刷新背包失败：" + err.Error())\n\t\t\t\t\treturn err\n\t\t\t\t}\n\t\t\t}\n\t\t\tlog.Printf("%s: latest-client LEGACY-BAG refresh deleted canonical main Views=2,121,123,125 before authoritative replay", conn.RemoteAddr())\n'''
    text = replace_once(text, old_refresh, new_refresh, "GM canonical bag refresh")
    main_go.write_text(text, encoding="utf-8")

    # Restore the exact current GetToolBoxViewport category mapping.
    helper = probe / "latest_client_current_bag_wire.go"
    text = helper.read_text(encoding="utf-8")
    old_mapping = '''func latestClientCurrentBagContainer(category int32) (uint16, bool) {\n\tswitch category {\n\tcase 1:\n\t\treturn 174, true\n\tcase 2:\n\t\treturn 176, true\n\tcase 3:\n\t\treturn 178, true\n\tcase 4:\n\t\treturn 180, true\n\tdefault:\n\t\treturn 0, false\n\t}\n}\n'''
    new_mapping = '''func latestClientCurrentBagContainer(category int32) (uint16, bool) {\n\tswitch category {\n\tcase 1:\n\t\treturn 2, true\n\tcase 2:\n\t\treturn 121, true\n\tcase 3:\n\t\treturn 123, true\n\tcase 4:\n\t\treturn 125, true\n\tdefault:\n\t\treturn 0, false\n\t}\n}\n'''
    text = replace_once(text, old_mapping, new_mapping, "canonical GetToolBoxViewport mapping")

    old_frames = '''func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, properties []serverViewProperty) ([][]byte, error) {\n\t_ = category // category is already encoded by current container view 174/176/178/180\n\twireProperties := latestClientBagWireProperties(properties)\n\tif len(wireProperties) == 0 { return nil, fmt.Errorf("latest-client bag object has no negotiated properties") }\n\tadd, err := latestClientViewAddCounted(viewID, objectIndex, wireProperties)\n\tif err != nil { return nil, err }\n\treturn [][]byte{add}, nil\n}\n'''
    new_frames = '''func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, properties []serverViewProperty) ([][]byte, error) {\n\twireProperties := latestClientBagWireProperties(properties)\n\tif len(wireProperties) == 0 { return nil, fmt.Errorf("latest-client bag object has no negotiated properties") }\n\n\tvar haveItemType, haveViewID, haveIdent bool\n\tfor _, property := range wireProperties {\n\t\tswitch property.index {\n\t\tcase 205: // ItemType/int32 in the original negotiated table\n\t\t\thaveItemType = property.int32 != nil\n\t\tcase 228: // append-only ViewID/int32\n\t\t\tif property.int32 != nil {\n\t\t\t\thaveViewID = true\n\t\t\t\tif *property.int32 != category {\n\t\t\t\t\treturn nil, fmt.Errorf("latest-client bag ViewID=%d does not match category=%d", *property.int32, category)\n\t\t\t\t}\n\t\t\t}\n\t\tcase 229: // append-only Ident/string\n\t\t\thaveIdent = property.text != nil && *property.text != ""\n\t\t}\n\t}\n\tif !haveItemType || !haveViewID || !haveIdent {\n\t\treturn nil, fmt.Errorf("latest-client bag object missing required fields ItemType=%v ViewID=%v Ident=%v", haveItemType, haveViewID, haveIdent)\n\t}\n\n\tadd, err := latestClientViewAddCounted(viewID, objectIndex, wireProperties)\n\tif err != nil { return nil, err }\n\treturn [][]byte{add}, nil\n}\n'''
    text = replace_once(text, old_frames, new_frames, "rich canonical bag frame validation")
    helper.write_text(text, encoding="utf-8")

    # Map the recovered rich bag-row semantic sources onto the current negotiated
    # table. Keep unsupported historical fields off wire.
    compat = probe / "latest_client_bag_view_ordinal_compat.go"
    text = compat.read_text(encoding="utf-8")
    old_normalizer = '''func latestClientBagWireProperties(properties []serverViewProperty) []serverViewProperty {\n\tvar configID string\n\tvar haveConfig bool\n\tvar amount int32\n\tvar haveAmount bool\n\tvar maxAmount int32\n\tvar haveMaxAmount bool\n\n\tfor _, property := range properties {\n\t\tswitch property.index {\n\t\tcase 7, 105, 0x05A0:\n\t\t\tif property.text != nil {\n\t\t\t\tconfigID = *property.text\n\t\t\t\thaveConfig = true\n\t\t\t}\n\t\tcase 106, 0x0766:\n\t\t\tif property.int32 != nil {\n\t\t\t\tamount = *property.int32\n\t\t\t\thaveAmount = true\n\t\t\t}\n\t\tcase 110, 0x0767:\n\t\t\tif property.int32 != nil {\n\t\t\t\tmaxAmount = *property.int32\n\t\t\t\thaveMaxAmount = true\n\t\t\t}\n\t\t}\n\t\tif property.nest != nil && property.nest.subIndex == 0x05A0 && property.nest.text != nil {\n\t\t\tconfigID = *property.nest.text\n\t\t\thaveConfig = true\n\t\t}\n\t}\n\n\tresult := make([]serverViewProperty, 0, 4)\n\tif haveConfig {\n\t\tresult = append(result, viewString(7, configID), viewString(105, configID))\n\t}\n\tif haveAmount {\n\t\tresult = append(result, viewInt(106, amount))\n\t}\n\tif haveMaxAmount {\n\t\tresult = append(result, viewInt(110, maxAmount))\n\t}\n\treturn result\n}\n'''
    new_normalizer = '''func latestClientBagWireProperties(properties []serverViewProperty) []serverViewProperty {\n\tvar configID string\n\tvar haveConfig bool\n\tvar amount int32\n\tvar haveAmount bool\n\tvar maxAmount int32\n\tvar haveMaxAmount bool\n\tvar itemType int32\n\tvar haveItemType bool\n\tvar itemViewID int32\n\tvar haveViewID bool\n\tvar ident string\n\tvar haveIdent bool\n\n\tfor _, property := range properties {\n\t\tswitch property.index {\n\t\tcase 7, 105, 0x05A0:\n\t\t\tif property.text != nil {\n\t\t\t\tconfigID = *property.text\n\t\t\t\thaveConfig = true\n\t\t\t}\n\t\tcase 106, 0x0766:\n\t\t\tif property.int32 != nil {\n\t\t\t\tamount = *property.int32\n\t\t\t\thaveAmount = true\n\t\t\t}\n\t\tcase 110, 0x0767:\n\t\t\tif property.int32 != nil {\n\t\t\t\tmaxAmount = *property.int32\n\t\t\t\thaveMaxAmount = true\n\t\t\t}\n\t\tcase 205, 0x0761:\n\t\t\tif property.int32 != nil {\n\t\t\t\titemType = *property.int32\n\t\t\t\thaveItemType = true\n\t\t\t}\n\t\tcase 228, 0x0763:\n\t\t\tif property.int32 != nil {\n\t\t\t\titemViewID = *property.int32\n\t\t\t\thaveViewID = true\n\t\t\t}\n\t\tcase 229, 0x0765:\n\t\t\tif property.text != nil {\n\t\t\t\tident = *property.text\n\t\t\t\thaveIdent = ident != ""\n\t\t\t}\n\t\t}\n\t\tif property.nest != nil && property.nest.subIndex == 0x05A0 && property.nest.text != nil {\n\t\t\tconfigID = *property.nest.text\n\t\t\thaveConfig = true\n\t\t}\n\t}\n\n\t// Emit in ascending negotiated ordinal order. Existing 0..227 positions are\n\t// unchanged; ViewID and Ident are the two append-only fields.\n\tresult := make([]serverViewProperty, 0, 7)\n\tif haveConfig {\n\t\tresult = append(result, viewString(7, configID), viewString(105, configID))\n\t}\n\tif haveAmount {\n\t\tresult = append(result, viewInt(106, amount))\n\t}\n\tif haveMaxAmount {\n\t\tresult = append(result, viewInt(110, maxAmount))\n\t}\n\tif haveItemType {\n\t\tresult = append(result, viewInt(205, itemType))\n\t}\n\tif haveViewID {\n\t\tresult = append(result, viewInt(228, itemViewID))\n\t}\n\tif haveIdent {\n\t\tresult = append(result, viewString(229, ident))\n\t}\n\treturn result\n}\n'''
    text = replace_once(text, old_normalizer, new_normalizer, "rich bag ordinal normalization")
    compat.write_text(text, encoding="utf-8")

    # Property-table assertions: ItemType already exists at 205; only ViewID
    # and Ident are appended at 228/229.
    ptest = probe / "latest_client_player_property_ordinal_compat_test.go"
    text = ptest.read_text(encoding="utf-8")
    text = replace_once(text,
        '''\tif latestClientPlayerWirePropertyTableCount != 228 {\n\t\tt.Fatalf("negotiated property table count=%d, want original 228", latestClientPlayerWirePropertyTableCount)\n\t}\n''',
        '''\tif latestClientPlayerWirePropertyTableCount != 230 {\n\t\tt.Fatalf("negotiated property table count=%d, want 230 after append-only ViewID+Ident", latestClientPlayerWirePropertyTableCount)\n\t}\n''',
        "player table count 230")
    ptest.write_text(text, encoding="utf-8")

    # Update focused current-bag tests to the canonical viewport and rich row.
    test = probe / "latest_client_current_bag_wire_test.go"
    text = test.read_text(encoding="utf-8")
    text = replace_once(text,
        '''func TestLatestClientCurrentBagContainers(t *testing.T) {\n\tcases := map[int32]uint16{1:174, 2:176, 3:178, 4:180}\n''',
        '''func TestLatestClientCurrentBagContainers(t *testing.T) {\n\tcases := map[int32]uint16{1:2, 2:121, 3:123, 4:125}\n''',
        "canonical bag container test")

    old_table_test = '''func TestLatestClientCurrentBagKeepsOriginalNegotiatedTable(t *testing.T) {\n\tif latestClientPlayerWirePropertyTableCount != 228 { t.Fatalf("property table=%d, want original 228", latestClientPlayerWirePropertyTableCount) }\n\tif _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"ViewID", typ:clientdata.WireInt32}]; ok { t.Fatal("synthetic ViewID must not be negotiated") }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"MaxHP", typ:clientdata.WireInt32}]; got != 30 { t.Fatalf("MaxHP ordinal moved to %d", got) }\n}\n'''
    new_table_test = '''func TestLatestClientCurrentBagRichObjectOrdinalsAreAppendOnly(t *testing.T) {\n\tif latestClientPlayerWirePropertyTableCount != 230 { t.Fatalf("property table=%d, want 230", latestClientPlayerWirePropertyTableCount) }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"ItemType", typ:clientdata.WireInt32}]; got != 205 { t.Fatalf("ItemType/int32 ordinal=%d, want 205", got) }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"ViewID", typ:clientdata.WireInt32}]; got != 228 { t.Fatalf("ViewID/int32 ordinal=%d, want 228", got) }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"Ident", typ:clientdata.WireString}]; got != 229 { t.Fatalf("Ident/string ordinal=%d, want 229", got) }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"MaxHP", typ:clientdata.WireInt32}]; got != 30 { t.Fatalf("MaxHP ordinal moved to %d", got) }\n\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"Force", typ:clientdata.WireString}]; got != 226 { t.Fatalf("Force ordinal moved to %d", got) }\n}\n'''
    text = replace_once(text, old_table_test, new_table_test, "rich bag table test")

    text = text.replace("latestClientViewAddSingle(174, 1, viewInt(228, 1))", "latestClientViewAddSingle(2, 1, viewInt(228, 1))", 1)
    text = text.replace("binary.LittleEndian.Uint16(frame[1:3]) != 174", "binary.LittleEndian.Uint16(frame[1:3]) != 2", 1)
    text = text.replace("latestClientViewObjectPropertySingle(174, 1, viewInt(106, 5))", "latestClientViewObjectPropertySingle(2, 1, viewInt(106, 5))", 1)
    text = text.replace("binary.LittleEndian.Uint32(frame[2:6]) != 174", "binary.LittleEndian.Uint32(frame[2:6]) != 2", 1)

    old_atomic = '''func TestLatestClientCurrentBagFramesAtomicNegotiatedAdd(t *testing.T) {\n\tprops := []serverViewProperty{viewString(7, "Game_item_hp_001"), viewInt(0x0766, 5), viewInt(0x0767, 30)}\n\tframes, err := latestClientCurrentBagFrames(174, 3, 1, props); if err != nil { t.Fatal(err) }\n\tif len(frames) != 1 { t.Fatalf("frame count=%d, want 1", len(frames)) }\n\tf := frames[0]\n\tif len(f) != 65 || f[0] != 0x18 || binary.LittleEndian.Uint16(f[1:3]) != 174 || binary.LittleEndian.Uint16(f[3:5]) != 3 || binary.LittleEndian.Uint16(f[5:7]) != 4 { t.Fatalf("negotiated VIEW_ADD header/len=%x", f) }\n\tif binary.LittleEndian.Uint16(f[7:9]) != 7 || binary.LittleEndian.Uint32(f[9:13]) != 17 || string(f[13:29]) != "Game_item_hp_001" || f[29] != 0 { t.Fatalf("ConfigID[7] property=%x", f[7:30]) }\n\tif binary.LittleEndian.Uint16(f[30:32]) != 105 || binary.LittleEndian.Uint32(f[32:36]) != 17 || string(f[36:52]) != "Game_item_hp_001" || f[52] != 0 { t.Fatalf("ConfigID[105] property=%x", f[30:53]) }\n\tif binary.LittleEndian.Uint16(f[53:55]) != 106 || binary.LittleEndian.Uint32(f[55:59]) != 5 { t.Fatalf("Amount property=%x", f[53:59]) }\n\tif binary.LittleEndian.Uint16(f[59:61]) != 110 || binary.LittleEndian.Uint32(f[61:65]) != 30 { t.Fatalf("MaxAmount property=%x", f[59:65]) }\n}\n'''
    new_atomic = '''func TestLatestClientCurrentBagFramesAtomicRichObject(t *testing.T) {\n\tident := "7100568-001-0000000001-0001"\n\tprops := []serverViewProperty{\n\t\tviewString(7, "Game_item_hp_001"),\n\t\tviewInt(0x0761, 1),\n\t\tviewInt(0x0763, 1),\n\t\tviewString(0x0765, ident),\n\t\tviewInt(0x0766, 5),\n\t\tviewInt(0x0767, 30),\n\t}\n\tframes, err := latestClientCurrentBagFrames(2, 3, 1, props); if err != nil { t.Fatal(err) }\n\tif len(frames) != 1 { t.Fatalf("frame count=%d, want 1", len(frames)) }\n\tf := frames[0]\n\tif f[0] != 0x18 || binary.LittleEndian.Uint16(f[1:3]) != 2 || binary.LittleEndian.Uint16(f[3:5]) != 3 || binary.LittleEndian.Uint16(f[5:7]) != 7 { t.Fatalf("rich VIEW_ADD header=%x", f[:7]) }\n\n\twant := []uint16{7, 105, 106, 110, 205, 228, 229}\n\toff := 7\n\tfor i, index := range want {\n\t\tif off+2 > len(f) { t.Fatalf("property[%d] truncated at %d frame=%x", i, off, f) }\n\t\tgot := binary.LittleEndian.Uint16(f[off:off+2]); off += 2\n\t\tif got != index { t.Fatalf("property[%d].ordinal=%d, want %d frame=%x", i, got, index, f) }\n\t\tswitch index {\n\t\tcase 7, 105, 229:\n\t\t\tif off+4 > len(f) { t.Fatalf("string length truncated index=%d", index) }\n\t\t\tn := int(binary.LittleEndian.Uint32(f[off:off+4])); off += 4\n\t\t\tif n <= 0 || off+n > len(f) || f[off+n-1] != 0 { t.Fatalf("bad string property index=%d len=%d", index, n) }\n\t\t\toff += n\n\t\tdefault:\n\t\t\tif off+4 > len(f) { t.Fatalf("int32 truncated index=%d", index) }\n\t\t\toff += 4\n\t\t}\n\t}\n\tif off != len(f) { t.Fatalf("frame tail=%d bytes frame=%x", len(f)-off, f) }\n}\n'''
    text = replace_once(text, old_atomic, new_atomic, "rich atomic bag frame test")
    test.write_text(text, encoding="utf-8")

    # Update the original bag normalizer test to verify the evidence-backed rich
    # property set and that every emitted ordinal is inside the new table.
    btest = probe / "latest_client_bag_view_ordinal_compat_test.go"
    text = btest.read_text(encoding="utf-8")
    old_btest = '''func TestLatestClientBagWirePropertiesUsesOnlyNegotiatedSlots(t *testing.T) {\n\tprops := []serverViewProperty{\n\t\tviewString(7, "Game_item_hp_001"),\n\t\tviewInt(0x0761, 100),\n\t\tviewInt(0x0766, 5),\n\t\tviewInt(0x0767, 99),\n\t\tviewInt(0x0779, 123),\n\t}\n\tgot := latestClientBagWireProperties(props)\n\tif len(got) != 4 {\n\t\tt.Fatalf("bag wire property count=%d, want 4", len(got))\n\t}\n\twant := []uint16{7, 105, 106, 110}\n\tfor i, index := range want {\n\t\tif got[i].index != index {\n\t\t\tt.Fatalf("property[%d].index=%d, want %d", i, got[i].index, index)\n\t\t}\n\t}\n\tfor _, property := range got {\n\t\tif int(property.index) >= latestClientPlayerWirePropertyTableCount {\n\t\t\tt.Fatalf("emitted out-of-range property %d >= %d", property.index, latestClientPlayerWirePropertyTableCount)\n\t\t}\n\t}\n}\n'''
    new_btest = '''func TestLatestClientBagWirePropertiesUsesRichCurrentObjectContract(t *testing.T) {\n\tprops := []serverViewProperty{\n\t\tviewString(7, "Game_item_hp_001"),\n\t\tviewInt(0x0761, 1),\n\t\tviewInt(0x0763, 1),\n\t\tviewString(0x0765, "7100568-001-0000000001-0001"),\n\t\tviewInt(0x0766, 5),\n\t\tviewInt(0x0767, 30),\n\t\tviewInt(0x0779, 6), // unsupported historical field must stay off wire\n\t}\n\tgot := latestClientBagWireProperties(props)\n\tif len(got) != 7 {\n\t\tt.Fatalf("bag wire property count=%d, want 7", len(got))\n\t}\n\twant := []uint16{7, 105, 106, 110, 205, 228, 229}\n\tfor i, index := range want {\n\t\tif got[i].index != index {\n\t\t\tt.Fatalf("property[%d].index=%d, want %d", i, got[i].index, index)\n\t\t}\n\t}\n\tfor _, property := range got {\n\t\tif int(property.index) >= latestClientPlayerWirePropertyTableCount {\n\t\t\tt.Fatalf("emitted out-of-range property %d >= %d", property.index, latestClientPlayerWirePropertyTableCount)\n\t\t}\n\t}\n}\n'''
    text = replace_once(text, old_btest, new_btest, "rich bag normalizer test")
    btest.write_text(text, encoding="utf-8")

    print(f"patched {main_go}")
    print(f"patched {helper}")
    print(f"patched {compat}")
    print(f"patched {ptest}")
    print(f"patched {test}")
    print(f"patched {btest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
