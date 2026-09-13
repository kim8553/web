#!/usr/bin/env python3
# Stage37 latest-client current GoodsGrid wire + scene exit-drain A/B.
# Current-client evidence: bag containers 174..181; ViewID category 1..4;
# VIEW_ADD and view-object SERVER_OBJECT_PROPERTY carry exactly one property.
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
        raise SystemExit("usage: stage37_latest_client_current_bag_wire_reentry_drain_patch.py <buildtree>")
    probe = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    text = insert_before_scene_visible_return(text, r'''
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
''')
    main_go.write_text(text, encoding="utf-8")

    helper = probe / "latest_client_current_bag_wire.go"
    helper.write_text(r'''package main

import (
	"encoding/binary"
	"fmt"

	"github.com/local/9yin-go-server/internal/clientdata"
)

func latestClientCurrentBagContainer(category int32) (uint16, bool) {
	switch category {
	case 1:
		return 174, true
	case 2:
		return 176, true
	case 3:
		return 178, true
	case 4:
		return 180, true
	default:
		return 0, false
	}
}

func latestClientCurrentBagOrdinal(name string, typ clientdata.WireType) (uint16, error) {
	ordinal, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: name, typ: typ}]
	if !ok {
		return 0, fmt.Errorf("latest-client bag property %s/%s is not negotiated", name, typ)
	}
	return ordinal, nil
}

func latestClientViewAddSingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {
	msg := make([]byte, 5)
	msg[0] = 0x18
	binary.LittleEndian.PutUint16(msg[1:3], viewID)
	binary.LittleEndian.PutUint16(msg[3:5], objectIndex)
	if err := appendViewProperty(&msg, property); err != nil {
		return nil, err
	}
	return msg, nil
}

func latestClientViewObjectPropertySingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {
	msg := make([]byte, 10)
	msg[0] = 0x10
	msg[1] = 0x01
	binary.LittleEndian.PutUint32(msg[2:6], uint32(viewID))
	binary.LittleEndian.PutUint32(msg[6:10], uint32(objectIndex))
	if err := appendViewProperty(&msg, property); err != nil {
		return nil, err
	}
	return msg, nil
}

func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, configID string, amount int32) ([][]byte, error) {
	viewIDOrdinal, err := latestClientCurrentBagOrdinal("ViewID", clientdata.WireInt32)
	if err != nil { return nil, err }
	configOrdinal, err := latestClientCurrentBagOrdinal("ConfigID", clientdata.WireString)
	if err != nil { return nil, err }
	amountOrdinal, err := latestClientCurrentBagOrdinal("Amount", clientdata.WireInt32)
	if err != nil { return nil, err }
	add, err := latestClientViewAddSingle(viewID, objectIndex, viewInt(viewIDOrdinal, category))
	if err != nil { return nil, err }
	config, err := latestClientViewObjectPropertySingle(viewID, objectIndex, viewString(configOrdinal, configID))
	if err != nil { return nil, err }
	count, err := latestClientViewObjectPropertySingle(viewID, objectIndex, viewInt(amountOrdinal, amount))
	if err != nil { return nil, err }
	return [][]byte{add, config, count}, nil
}
''', encoding="utf-8")

    player_test = probe / "latest_client_player_property_ordinal_compat_test.go"
    text = player_test.read_text(encoding="utf-8")
    text = replace_once(text,
        '''\tif latestClientPlayerWirePropertyTableCount != 228 {\n\t\tt.Fatalf("negotiated property table count=%d, want 228", latestClientPlayerWirePropertyTableCount)\n\t}\n''',
        '''\tif latestClientPlayerWirePropertyTableCount != 229 {\n\t\tt.Fatalf("negotiated property table count=%d, want 229 after append-only ViewID", latestClientPlayerWirePropertyTableCount)\n\t}\n''',
        "player property table count")
    player_test.write_text(text, encoding="utf-8")

    overlay = probe / "zz_recovered_overlay.go"
    text = overlay.read_text(encoding="utf-8")
    text = replace_once(text,
        '''func bagViewForViewID(viewID int32) uint16 {\n\tswitch viewID {\n\tcase 2:\n\t\treturn 121\n\tcase 3:\n\t\treturn 123\n\tcase 4:\n\t\treturn 125\n\tdefault:\n\t\treturn 2\n\t}\n}\n''',
        '''func bagViewForViewID(viewID int32) uint16 {\n\tview, ok := latestClientCurrentBagContainer(viewID)\n\tif !ok { return 0 }\n\treturn view\n}\n''',
        "current bag container mapping")
    text = replace_once(text,
        '''\t\tview, slot := assignSlot(item)\n\t\tviews[view] = struct{}{}\n\t\tframe, err := serverViewAdd(view, uint16(slot), bagItemProps(view, item))\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tlog.Printf("%s: grant bag item=%s view=%d slot=%d frame=%x", remote, item.ConfigID, view, uint16(slot), frame)\n\t\tif err := link.WriteFrame(frame); err != nil {\n\t\t\treturn err\n\t\t}\n''',
        '''\t\tview, slot := assignSlot(item)\n\t\tif view == 0 {\n\t\t\treturn fmt.Errorf("latest-client bag item %s has unsupported ViewID category %d", item.ConfigID, item.ViewID)\n\t\t}\n\t\tviews[view] = struct{}{}\n\t\tframes, err := latestClientCurrentBagFrames(view, uint16(slot), item.ViewID, item.ConfigID, item.Amount)\n\t\tif err != nil {\n\t\t\treturn fmt.Errorf("latest-client bag item %s view=%d slot=%d: %w", item.ConfigID, view, slot, err)\n\t\t}\n\t\tlog.Printf("%s: latest-client CURRENT-BAG item=%s category=%d container=%d slot=%d table=%d add=%x config=%x amount=%x", remote, item.ConfigID, item.ViewID, view, uint16(slot), latestClientPlayerWirePropertyTableCount, frames[0], frames[1], frames[2])\n\t\tfor _, frame := range frames {\n\t\t\tif err := link.WriteFrame(frame); err != nil { return err }\n\t\t}\n''',
        "current bag item publication")
    overlay.write_text(text, encoding="utf-8")

    text = main_go.read_text(encoding="utf-8")
    text = replace_once(text,
        '''\t\t\tif err := link.WriteFrame(serverDeleteViewCompat(2)); err != nil {\n\t\t\t\tgmSession.setAction("刷新背包失败：" + err.Error())\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tlog.Printf("%s: latest-client BAG-REFRESH deleted live View=2 before authoritative replay", conn.RemoteAddr())\n''',
        '''\t\t\tfor _, bagView := range []uint16{174, 176, 178, 180} {\n\t\t\t\tif err := link.WriteFrame(serverDeleteViewCompat(bagView)); err != nil {\n\t\t\t\t\tgmSession.setAction("刷新背包失败：" + err.Error())\n\t\t\t\t\treturn err\n\t\t\t\t}\n\t\t\t}\n\t\t\tlog.Printf("%s: latest-client CURRENT-BAG refresh deleted main Views=174,176,178,180 before authoritative replay", conn.RemoteAddr())\n''',
        "GM current bag refresh")
    main_go.write_text(text, encoding="utf-8")

    trans = probe / "scene_transition.go"
    text = trans.read_text(encoding="utf-8")
    text = replace_once(text,
        '''\ttime.Sleep(750 * time.Millisecond)\n''',
        '''\ttime.Sleep(2750 * time.Millisecond) // latest-client REENTRY-EXIT-DRAIN A/B discriminator\n''',
        "scene exit-drain discriminator")
    trans.write_text(text, encoding="utf-8")

    test = probe / "latest_client_current_bag_wire_test.go"
    test.write_text(r'''package main

import (
	"encoding/binary"
	"testing"
	"github.com/local/9yin-go-server/internal/clientdata"
)

func TestLatestClientCurrentBagContainers(t *testing.T) {
	cases := map[int32]uint16{1:174, 2:176, 3:178, 4:180}
	for category, want := range cases {
		got, ok := latestClientCurrentBagContainer(category)
		if !ok || got != want { t.Fatalf("category %d -> (%d,%v), want (%d,true)", category, got, ok, want) }
	}
	if _, ok := latestClientCurrentBagContainer(0); ok { t.Fatal("unknown bag category must not be guessed") }
}

func TestLatestClientCurrentBagViewIDIsAppendOnlyOrdinal228(t *testing.T) {
	if latestClientPlayerWirePropertyTableCount != 229 { t.Fatalf("property table=%d, want 229", latestClientPlayerWirePropertyTableCount) }
	got, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"ViewID", typ:clientdata.WireInt32}]
	if !ok || got != 228 { t.Fatalf("ViewID/int32 ordinal=(%d,%v), want (228,true)", got, ok) }
	if got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"MaxHP", typ:clientdata.WireInt32}]; got != 30 { t.Fatalf("MaxHP ordinal moved to %d", got) }
}

func TestLatestClientViewAddSingleExactLayout(t *testing.T) {
	frame, err := latestClientViewAddSingle(174, 1, viewInt(228, 1)); if err != nil { t.Fatal(err) }
	if len(frame) != 11 || frame[0] != 0x18 || binary.LittleEndian.Uint16(frame[1:3]) != 174 || binary.LittleEndian.Uint16(frame[3:5]) != 1 || binary.LittleEndian.Uint16(frame[5:7]) != 228 || binary.LittleEndian.Uint32(frame[7:11]) != 1 { t.Fatalf("VIEW_ADD single frame=%x", frame) }
}

func TestLatestClientViewObjectPropertySingleExactLayout(t *testing.T) {
	frame, err := latestClientViewObjectPropertySingle(174, 1, viewInt(106, 5)); if err != nil { t.Fatal(err) }
	if len(frame) != 16 || frame[0] != 0x10 || frame[1] != 1 || binary.LittleEndian.Uint32(frame[2:6]) != 174 || binary.LittleEndian.Uint32(frame[6:10]) != 1 || binary.LittleEndian.Uint16(frame[10:12]) != 106 || binary.LittleEndian.Uint32(frame[12:16]) != 5 { t.Fatalf("SERVER_OBJECT_PROPERTY single frame=%x", frame) }
}

func TestLatestClientCurrentBagFramesStartsWithViewIDThenIdentityCount(t *testing.T) {
	frames, err := latestClientCurrentBagFrames(174, 3, 1, "Game_item_hp_001", 5); if err != nil { t.Fatal(err) }
	if len(frames) != 3 { t.Fatalf("frame count=%d", len(frames)) }
	if frames[0][0] != 0x18 || binary.LittleEndian.Uint16(frames[0][5:7]) != 228 { t.Fatalf("first frame=%x", frames[0]) }
	if frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][10:12]) != 105 { t.Fatalf("second frame=%x", frames[1]) }
	if frames[2][0] != 0x10 || binary.LittleEndian.Uint16(frames[2][10:12]) != 106 { t.Fatalf("third frame=%x", frames[2]) }
}
''', encoding="utf-8")

    print(f"patched {main_go}")
    print(f"wrote {helper}")
    print(f"patched {player_test}")
    print(f"patched {overlay}")
    print(f"patched {trans}")
    print(f"wrote {test}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
