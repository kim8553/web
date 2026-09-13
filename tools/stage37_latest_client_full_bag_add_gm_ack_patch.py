#!/usr/bin/env python3
# Stage37 latest-client full counted bag-add + GM ack timeout A/B.
#
# Evidence basis from 2026-09-14 LIVE and current client disassembly:
# - The counted-wire A/B is accepted far enough to complete scene re-entry, but
#   bag objects still do not become visible.
# - Current A/B creates each bag object with VIEW_ADD containing only ViewID,
#   then sends ConfigID/Amount later through OBJECT_PROPERTY.
# - The previously recovered bag normalization carries four negotiated fields:
#     ConfigID string ordinal 7
#     ConfigID string ordinal 105
#     Amount int32 ordinal 106
#     MaxAmount int32 ordinal 110
#   and Game_item_hp_001 previously encoded MaxAmount=30.
# - Current FxGameLogic.dll reads Amount and MaxAmount, and bag/view add callbacks
#   are creation-time paths; therefore this A/B restores the complete already-
#   recovered property set atomically in one counted VIEW_ADD together with the
#   current ViewID category. No new property is invented.
# - serveGM currently uses time.NewTimer(2*time.Second). Current scene switch has
#   a proven 2750ms exit-drain before EntryScene, so the HTTP timeout can fire
#   before a successful command finishes. Raise only this local GM wait to 20s.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_full_bag_add_gm_ack_patch.py <buildtree>")

    probe = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    helper = probe / "latest_client_current_bag_wire.go"
    text = helper.read_text(encoding="utf-8")
    old = '''func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, configID string, amount int32) ([][]byte, error) {\n\tviewIDOrdinal, err := latestClientCurrentBagOrdinal("ViewID", clientdata.WireInt32)\n\tif err != nil { return nil, err }\n\tconfigOrdinal, err := latestClientCurrentBagOrdinal("ConfigID", clientdata.WireString)\n\tif err != nil { return nil, err }\n\tamountOrdinal, err := latestClientCurrentBagOrdinal("Amount", clientdata.WireInt32)\n\tif err != nil { return nil, err }\n\tadd, err := latestClientViewAddSingle(viewID, objectIndex, viewInt(viewIDOrdinal, category))\n\tif err != nil { return nil, err }\n\tconfig, err := latestClientViewObjectPropertySingle(viewID, objectIndex, viewString(configOrdinal, configID))\n\tif err != nil { return nil, err }\n\tcount, err := latestClientViewObjectPropertySingle(viewID, objectIndex, viewInt(amountOrdinal, amount))\n\tif err != nil { return nil, err }\n\treturn [][]byte{add, config, count}, nil\n}\n'''
    new = '''func latestClientViewAddCounted(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {\n\tif len(properties) > 0xffff { return nil, fmt.Errorf("latest-client VIEW_ADD property count=%d", len(properties)) }\n\tmsg := make([]byte, 7)\n\tmsg[0] = 0x18\n\tbinary.LittleEndian.PutUint16(msg[1:3], viewID)\n\tbinary.LittleEndian.PutUint16(msg[3:5], objectIndex)\n\tbinary.LittleEndian.PutUint16(msg[5:7], uint16(len(properties)))\n\tfor _, property := range properties {\n\t\tif err := appendViewProperty(&msg, property); err != nil { return nil, err }\n\t}\n\treturn msg, nil\n}\n\nfunc latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, properties []serverViewProperty) ([][]byte, error) {\n\tviewIDOrdinal, err := latestClientCurrentBagOrdinal("ViewID", clientdata.WireInt32)\n\tif err != nil { return nil, err }\n\twireProperties := latestClientBagWireProperties(properties)\n\tif len(wireProperties) == 0 { return nil, fmt.Errorf("latest-client bag object has no negotiated properties") }\n\tall := make([]serverViewProperty, 0, 1+len(wireProperties))\n\tall = append(all, viewInt(viewIDOrdinal, category))\n\tall = append(all, wireProperties...)\n\tadd, err := latestClientViewAddCounted(viewID, objectIndex, all)\n\tif err != nil { return nil, err }\n\treturn [][]byte{add}, nil\n}\n'''
    text = replace_once(text, old, new, "atomic counted current-bag VIEW_ADD")
    helper.write_text(text, encoding="utf-8")

    overlay = probe / "zz_recovered_overlay.go"
    text = overlay.read_text(encoding="utf-8")
    old = '''\t\tframes, err := latestClientCurrentBagFrames(view, uint16(slot), item.ViewID, item.ConfigID, item.Amount)\n\t\tif err != nil {\n\t\t\treturn fmt.Errorf("latest-client bag item %s view=%d slot=%d: %w", item.ConfigID, view, slot, err)\n\t\t}\n\t\tlog.Printf("%s: latest-client CURRENT-BAG item=%s category=%d container=%d slot=%d table=%d add=%x config=%x amount=%x", remote, item.ConfigID, item.ViewID, view, uint16(slot), latestClientPlayerWirePropertyTableCount, frames[0], frames[1], frames[2])\n\t\tfor _, frame := range frames {\n\t\t\tif err := link.WriteFrame(frame); err != nil { return err }\n\t\t}\n'''
    new = '''\t\tframes, err := latestClientCurrentBagFrames(view, uint16(slot), item.ViewID, bagItemProps(view, item))\n\t\tif err != nil {\n\t\t\treturn fmt.Errorf("latest-client bag item %s view=%d slot=%d: %w", item.ConfigID, view, slot, err)\n\t\t}\n\t\tlog.Printf("%s: latest-client CURRENT-BAG-FULL item=%s category=%d container=%d slot=%d table=%d frame=%x", remote, item.ConfigID, item.ViewID, view, uint16(slot), latestClientPlayerWirePropertyTableCount, frames[0])\n\t\tfor _, frame := range frames {\n\t\t\tif err := link.WriteFrame(frame); err != nil { return err }\n\t\t}\n'''
    text = replace_once(text, old, new, "publish full bag properties in VIEW_ADD")
    overlay.write_text(text, encoding="utf-8")

    test = probe / "latest_client_current_bag_wire_test.go"
    text = test.read_text(encoding="utf-8")
    old = '''func TestLatestClientCurrentBagFramesStartsWithViewIDThenIdentityCount(t *testing.T) {\n\tframes, err := latestClientCurrentBagFrames(174, 3, 1, "Game_item_hp_001", 5); if err != nil { t.Fatal(err) }\n\tif len(frames) != 3 { t.Fatalf("frame count=%d", len(frames)) }\n\tif frames[0][0] != 0x18 || binary.LittleEndian.Uint16(frames[0][5:7]) != 1 || binary.LittleEndian.Uint16(frames[0][7:9]) != 228 { t.Fatalf("first counted frame=%x", frames[0]) }\n\tif frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][10:12]) != 1 || binary.LittleEndian.Uint16(frames[1][12:14]) != 105 { t.Fatalf("second counted frame=%x", frames[1]) }\n\tif frames[2][0] != 0x10 || binary.LittleEndian.Uint16(frames[2][10:12]) != 1 || binary.LittleEndian.Uint16(frames[2][12:14]) != 106 { t.Fatalf("third counted frame=%x", frames[2]) }\n}\n'''
    new = '''func TestLatestClientCurrentBagFramesAtomicFullAdd(t *testing.T) {\n\tprops := []serverViewProperty{\n\t\tviewString(7, "Game_item_hp_001"),\n\t\tviewInt(0x0766, 5),\n\t\tviewInt(0x0767, 30),\n\t}\n\tframes, err := latestClientCurrentBagFrames(174, 3, 1, props); if err != nil { t.Fatal(err) }\n\tif len(frames) != 1 { t.Fatalf("frame count=%d, want 1", len(frames)) }\n\tf := frames[0]\n\tif len(f) != 71 || f[0] != 0x18 || binary.LittleEndian.Uint16(f[1:3]) != 174 || binary.LittleEndian.Uint16(f[3:5]) != 3 || binary.LittleEndian.Uint16(f[5:7]) != 5 { t.Fatalf("full VIEW_ADD header/len=%x", f) }\n\tif binary.LittleEndian.Uint16(f[7:9]) != 228 || binary.LittleEndian.Uint32(f[9:13]) != 1 { t.Fatalf("ViewID property=%x", f[7:13]) }\n\tif binary.LittleEndian.Uint16(f[13:15]) != 7 || binary.LittleEndian.Uint32(f[15:19]) != 17 || string(f[19:35]) != "Game_item_hp_001" || f[35] != 0 { t.Fatalf("ConfigID[7] property=%x", f[13:36]) }\n\tif binary.LittleEndian.Uint16(f[36:38]) != 105 || binary.LittleEndian.Uint32(f[38:42]) != 17 || string(f[42:58]) != "Game_item_hp_001" || f[58] != 0 { t.Fatalf("ConfigID[105] property=%x", f[36:59]) }\n\tif binary.LittleEndian.Uint16(f[59:61]) != 106 || binary.LittleEndian.Uint32(f[61:65]) != 5 { t.Fatalf("Amount property=%x", f[59:65]) }\n\tif binary.LittleEndian.Uint16(f[65:67]) != 110 || binary.LittleEndian.Uint32(f[67:71]) != 30 { t.Fatalf("MaxAmount property=%x", f[65:71]) }\n}\n'''
    text = replace_once(text, old, new, "atomic full bag frame test")
    test.write_text(text, encoding="utf-8")

    gm = probe / "gm_panel.go"
    text = gm.read_text(encoding="utf-8")
    timer_old = "time.NewTimer(2 * time.Second)"
    if text.count(timer_old) != 1:
        timer_old = "time.NewTimer(2*time.Second)"
    if text.count(timer_old) != 1:
        raise SystemExit(f"GM ack timer: expected one 2s NewTimer anchor, found spaced={text.count('time.NewTimer(2 * time.Second)')} compact={text.count('time.NewTimer(2*time.Second)')}")
    text = text.replace(timer_old, "time.NewTimer(20 * time.Second) // latest-client GM command wait; scene exit-drain alone is 2.75s", 1)
    gm.write_text(text, encoding="utf-8")

    print(f"patched {helper}")
    print(f"patched {overlay}")
    print(f"patched {test}")
    print(f"patched {gm}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
