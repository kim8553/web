#!/usr/bin/env python3
# Apply after stage37_builtin_ident_followup.py.
# Current FxNet2 decodes VIEW_ADD property indexes against the negotiated
# property-table ordinal range. Views 40/41/43/45/46/47/48 were still sending
# historical V2/global property IDs directly; normalize only fields whose
# current-client name/type contract is already proven. Do not invent ordinals
# for legacy NeiGongLevel/WuXing/BufferID metadata absent from the current table.
from pathlib import Path
import sys


def r1(text, old, new, label):
    n = text.count(old)
    if n != 1:
        raise SystemExit(f"{label}: expected 1 anchor, found {n}")
    return text.replace(old, new, 1)


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_nonbag_view_ordinal_patch.py <buildtree>")
    p = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    helper = p / "latest_client_nonbag_view_ordinal_compat.go"
    if helper.exists():
        raise SystemExit(f"refuse overwrite existing {helper}")
    helper.write_text(r'''package main

import "fmt"

// latestClientNonBagViewOrdinalFamily identifies the post-ClientReady learned/
// progression GameViews whose recovered emitters still carry historical V2/
// global property IDs. FxNet2's current VIEW_ADD decoder indexes the negotiated
// table by ordinal, so those historical IDs are not valid wire ordinals.
func latestClientNonBagViewOrdinalFamily(viewID uint16) bool {
\tswitch viewID {
\tcase viewportSkill, viewportNormalAttack, viewportNeiGong, 45, viewportQingGong, 47, 48:
\t\treturn true
\tdefault:
\t\treturn false
\t}
}

// latestClientNonBagViewProperties maps only name/type pairs already present in
// the current 229-entry negotiated table. The three legacy View43 metadata IDs
// below are deliberately omitted: current FxGameLogic references
// NeiGongLevel/WuXing/BufferID through resource/config paths, while no matching
// negotiated fields exist. Adding new ordinals for them would be speculative.
func latestClientNonBagViewProperties(viewID uint16, properties []serverViewProperty) ([]serverViewProperty, error) {
\tif !latestClientNonBagViewOrdinalFamily(viewID) {
\t\treturn properties, nil
\t}
\tresult := make([]serverViewProperty, 0, len(properties))
\tfor _, property := range properties {
\t\tswitch property.index {
\t\tcase 0x05A0: // historical ConfigID -> current ConfigID/string ordinal 7
\t\t\tif property.text == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy ConfigID has non-string value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewString(7, *property.text))
\t\tcase 0x0761: // historical ItemType -> current ItemType/int32 ordinal 205
\t\t\tif property.int32 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy ItemType has non-int32 value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewInt(205, *property.int32))
\t\tcase 0x08BD: // historical StaticData -> current StaticData/int32 ordinal 204
\t\t\tif property.int32 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy StaticData has non-int32 value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewInt(204, *property.int32))
\t\tcase 0x05B9: // historical Level/byte -> current Level/int32 ordinal 6
\t\t\tif property.byte1 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy Level has non-byte value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewInt(6, int32(*property.byte1)))
\t\tcase 0x0823: // historical MaxLevel -> current MaxLevel/int32 ordinal 203
\t\t\tif property.int32 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy MaxLevel has non-int32 value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewInt(203, *property.int32))
\t\tcase 0x02FC: // historical CurFillValue -> current ordinal 176
\t\t\tif property.int32 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy CurFillValue has non-int32 value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewInt(176, *property.int32))
\t\tcase 0x02FD: // historical TotalFillValue -> current ordinal 177
\t\t\tif property.int32 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy TotalFillValue has non-int32 value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewInt(177, *property.int32))
\t\tcase 0x08BC: // historical CanUse/byte -> current CanUse/byte ordinal 225
\t\t\tif property.byte1 == nil {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy CanUse has non-byte value", viewID)
\t\t\t}
\t\t\tresult = append(result, viewByte(225, *property.byte1))
\t\tcase 0x07F2: // recovered skill lease value -> current PauseTime/float32 ordinal 212
\t\t\tif property.int32 != nil {
\t\t\t\tresult = append(result, viewFloat(212, float32(*property.int32)))
\t\t\t} else if property.real32 != nil {
\t\t\t\tresult = append(result, viewFloat(212, *property.real32))
\t\t\t} else {
\t\t\t\treturn nil, fmt.Errorf("view %d legacy PauseTime has unsupported value type", viewID)
\t\t\t}
\t\tcase 0x08EC, 0x08C3, 0x08ED:
\t\t\tif viewID != viewportNeiGong {
\t\t\t\treturn nil, fmt.Errorf("view %d contains unnegotiated legacy metadata property 0x%04X", viewID, property.index)
\t\t\t}
\t\t\t// Current table has no proven matching field; intentionally omit.
\t\tdefault:
\t\t\tif int(property.index) >= latestClientPlayerWirePropertyTableCount {
\t\t\t\treturn nil, fmt.Errorf("view %d contains unmapped property ordinal %d >= negotiated count %d", viewID, property.index, latestClientPlayerWirePropertyTableCount)
\t\t\t}
\t\t\tresult = append(result, property)
\t\t}
\t}
\tfor _, property := range result {
\t\tif int(property.index) >= latestClientPlayerWirePropertyTableCount {
\t\t\treturn nil, fmt.Errorf("view %d normalized property ordinal %d >= negotiated count %d", viewID, property.index, latestClientPlayerWirePropertyTableCount)
\t\t}
\t}
\treturn result, nil
}
'''.replace('\\t','\t'), encoding="utf-8")

    f = p / "messages.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, '''func serverViewAdd(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {
\tif latestClientStarterBagView(viewID) {
\t\tproperties = latestClientBagWireProperties(properties)
\t}
''', '''func serverViewAdd(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {
\tif latestClientStarterBagView(viewID) {
\t\tproperties = latestClientBagWireProperties(properties)
\t} else if latestClientNonBagViewOrdinalFamily(viewID) {
\t\tvar err error
\t\tproperties, err = latestClientNonBagViewProperties(viewID, properties)
\t\tif err != nil {
\t\t\treturn nil, err
\t\t}
\t}
''', 'serverViewAdd normalization hook')
    f.write_text(s, encoding="utf-8")

    test = p / "latest_client_nonbag_view_ordinal_compat_test.go"
    if test.exists():
        raise SystemExit(f"refuse overwrite existing {test}")
    test.write_text(r'''package main

import "testing"

func TestLatestClientNonBagViewLegacyOrdinalsNormalizeWithinNegotiatedTable(t *testing.T) {
\tprops := []serverViewProperty{
\t\tviewInt(0x0761, 1000),
\t\tviewString(0x05A0, "CS_jh_cqgf01"),
\t\tviewByte(0x08BC, 1),
\t\tviewInt(0x08BD, 4401),
\t\tviewByte(0x05B9, 3),
\t\tviewInt(0x0823, 5),
\t\tviewInt(0x07F2, 1000),
\t}
\tgot, err := latestClientNonBagViewProperties(viewportSkill, props)
\tif err != nil { t.Fatal(err) }
\twant := []uint16{205, 7, 225, 204, 6, 203, 212}
\tif len(got) != len(want) { t.Fatalf("len=%d want=%d", len(got), len(want)) }
\tfor i := range got {
\t\tif got[i].index != want[i] { t.Fatalf("property[%d]=%d want=%d", i, got[i].index, want[i]) }
\t\tif int(got[i].index) >= latestClientPlayerWirePropertyTableCount { t.Fatalf("property[%d]=%d outside table=%d", i, got[i].index, latestClientPlayerWirePropertyTableCount) }
\t}
\tif got[4].int32 == nil || *got[4].int32 != 3 { t.Fatalf("Level conversion=%+v", got[4]) }
\tif got[6].real32 == nil || *got[6].real32 != 1000 { t.Fatalf("PauseTime conversion=%+v", got[6]) }
}

func TestLatestClientNeiGongDropsOnlyUnnegotiatedLegacyMetadata(t *testing.T) {
\tprops := []serverViewProperty{
\t\tviewString(0x05A0, "ng_jh_001"),
\t\tviewInt(0x0761, 1002),
\t\tviewInt(0x08BD, 123),
\t\tviewByte(0x05B9, 2),
\t\tviewInt(0x0823, 36),
\t\tviewInt(0x08EC, 2),
\t\tviewInt(0x02FD, 750),
\t\tviewInt(0x08C3, 3),
\t\tviewString(0x08ED, "buff_test"),
\t}
\tgot, err := latestClientNonBagViewProperties(viewportNeiGong, props)
\tif err != nil { t.Fatal(err) }
\twant := []uint16{7, 205, 204, 6, 203, 177}
\tif len(got) != len(want) { t.Fatalf("len=%d want=%d got=%+v", len(got), len(want), got) }
\tfor i := range got {
\t\tif got[i].index != want[i] { t.Fatalf("property[%d]=%d want=%d", i, got[i].index, want[i]) }
\t}
}

func TestLatestClientNonBagViewRejectsUnknownOutOfRangeProperty(t *testing.T) {
\t_, err := latestClientNonBagViewProperties(viewportSkill, []serverViewProperty{viewInt(0x0900, 1)})
\tif err == nil { t.Fatal("expected unmapped out-of-range property rejection") }
}

func TestLatestClientNonBagServerViewAddNormalizesBeforeEncoding(t *testing.T) {
\tframe, err := serverViewAdd(viewportNormalAttack, 1, []serverViewProperty{
\t\tviewInt(0x0761, 1000), viewString(0x05A0, "CS_light_rad_81"), viewByte(0x08BC, 1),
\t\tviewInt(0x08BD, 5266), viewByte(0x05B9, 1), viewInt(0x0823, 1), viewInt(0x07F2, 0),
\t})
\tif err != nil { t.Fatal(err) }
\tif len(frame) < 7 || frame[0] != 0x18 { t.Fatalf("frame=%x", frame) }
}
'''.replace('\\t','\t'), encoding="utf-8")

    print(f"created {helper}")
    print(f"patched {f}")
    print(f"created {test}")


if __name__ == "__main__":
    main()
