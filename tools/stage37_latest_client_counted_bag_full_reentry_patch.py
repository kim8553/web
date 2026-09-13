#!/usr/bin/env python3
# Stage37 latest-client counted bag wire + full player re-entry A/B.
#
# Evidence basis from 2026-09-14 LIVE + latest FxNet2.dll disassembly:
# - VIEW_ADD header is 7 bytes: opcode, view u16, object u16, property-count u16.
# - SERVER_OBJECT_PROPERTY header is 12 bytes: 0x10, subtype 1, view u32,
#   object u32, property-count u16.
# - The prior A/B omitted the count field, so property index 228/105/106 was
#   decoded as a property count and the client emitted RecvProperty Out range.
# - 2750ms moved re-entry from wait-exit-scene to OnEntryScene create new, but
#   the client still did not emit flow player data ready / target ClientReady.
# - Working initial entry uses sendPlayerSpawn (AddObject + Snapshot608 +
#   Appearance + Location/Vitals); failed re-entry used only AddObject +
#   Location/Vitals.  Reusing the already-recovered successful spawn sequence is
#   an A/B discriminator only, not a claim that Snapshot608/Appearance are the
#   proven root cause.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_counted_bag_full_reentry_patch.py <buildtree>")

    probe = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    helper = probe / "latest_client_current_bag_wire.go"
    text = helper.read_text(encoding="utf-8")
    text = replace_once(text,
        '''func latestClientViewAddSingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {\n\tmsg := make([]byte, 5)\n\tmsg[0] = 0x18\n\tbinary.LittleEndian.PutUint16(msg[1:3], viewID)\n\tbinary.LittleEndian.PutUint16(msg[3:5], objectIndex)\n\tif err := appendViewProperty(&msg, property); err != nil {\n\t\treturn nil, err\n\t}\n\treturn msg, nil\n}\n''',
        '''func latestClientViewAddSingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {\n\tmsg := make([]byte, 7)\n\tmsg[0] = 0x18\n\tbinary.LittleEndian.PutUint16(msg[1:3], viewID)\n\tbinary.LittleEndian.PutUint16(msg[3:5], objectIndex)\n\tbinary.LittleEndian.PutUint16(msg[5:7], 1)\n\tif err := appendViewProperty(&msg, property); err != nil {\n\t\treturn nil, err\n\t}\n\treturn msg, nil\n}\n''',
        "VIEW_ADD property-count field")
    text = replace_once(text,
        '''func latestClientViewObjectPropertySingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {\n\tmsg := make([]byte, 10)\n\tmsg[0] = 0x10\n\tmsg[1] = 0x01\n\tbinary.LittleEndian.PutUint32(msg[2:6], uint32(viewID))\n\tbinary.LittleEndian.PutUint32(msg[6:10], uint32(objectIndex))\n\tif err := appendViewProperty(&msg, property); err != nil {\n\t\treturn nil, err\n\t}\n\treturn msg, nil\n}\n''',
        '''func latestClientViewObjectPropertySingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {\n\tmsg := make([]byte, 12)\n\tmsg[0] = 0x10\n\tmsg[1] = 0x01\n\tbinary.LittleEndian.PutUint32(msg[2:6], uint32(viewID))\n\tbinary.LittleEndian.PutUint32(msg[6:10], uint32(objectIndex))\n\tbinary.LittleEndian.PutUint16(msg[10:12], 1)\n\tif err := appendViewProperty(&msg, property); err != nil {\n\t\treturn nil, err\n\t}\n\treturn msg, nil\n}\n''',
        "SERVER_OBJECT_PROPERTY property-count field")
    helper.write_text(text, encoding="utf-8")

    test = probe / "latest_client_current_bag_wire_test.go"
    text = test.read_text(encoding="utf-8")
    text = replace_once(text,
        '''func TestLatestClientViewAddSingleExactLayout(t *testing.T) {\n\tframe, err := latestClientViewAddSingle(174, 1, viewInt(228, 1)); if err != nil { t.Fatal(err) }\n\tif len(frame) != 11 || frame[0] != 0x18 || binary.LittleEndian.Uint16(frame[1:3]) != 174 || binary.LittleEndian.Uint16(frame[3:5]) != 1 || binary.LittleEndian.Uint16(frame[5:7]) != 228 || binary.LittleEndian.Uint32(frame[7:11]) != 1 { t.Fatalf("VIEW_ADD single frame=%x", frame) }\n}\n''',
        '''func TestLatestClientViewAddSingleExactLayout(t *testing.T) {\n\tframe, err := latestClientViewAddSingle(174, 1, viewInt(228, 1)); if err != nil { t.Fatal(err) }\n\tif len(frame) != 13 || frame[0] != 0x18 || binary.LittleEndian.Uint16(frame[1:3]) != 174 || binary.LittleEndian.Uint16(frame[3:5]) != 1 || binary.LittleEndian.Uint16(frame[5:7]) != 1 || binary.LittleEndian.Uint16(frame[7:9]) != 228 || binary.LittleEndian.Uint32(frame[9:13]) != 1 { t.Fatalf("VIEW_ADD counted single frame=%x", frame) }\n}\n''',
        "VIEW_ADD counted layout test")
    text = replace_once(text,
        '''func TestLatestClientViewObjectPropertySingleExactLayout(t *testing.T) {\n\tframe, err := latestClientViewObjectPropertySingle(174, 1, viewInt(106, 5)); if err != nil { t.Fatal(err) }\n\tif len(frame) != 16 || frame[0] != 0x10 || frame[1] != 1 || binary.LittleEndian.Uint32(frame[2:6]) != 174 || binary.LittleEndian.Uint32(frame[6:10]) != 1 || binary.LittleEndian.Uint16(frame[10:12]) != 106 || binary.LittleEndian.Uint32(frame[12:16]) != 5 { t.Fatalf("SERVER_OBJECT_PROPERTY single frame=%x", frame) }\n}\n''',
        '''func TestLatestClientViewObjectPropertySingleExactLayout(t *testing.T) {\n\tframe, err := latestClientViewObjectPropertySingle(174, 1, viewInt(106, 5)); if err != nil { t.Fatal(err) }\n\tif len(frame) != 18 || frame[0] != 0x10 || frame[1] != 1 || binary.LittleEndian.Uint32(frame[2:6]) != 174 || binary.LittleEndian.Uint32(frame[6:10]) != 1 || binary.LittleEndian.Uint16(frame[10:12]) != 1 || binary.LittleEndian.Uint16(frame[12:14]) != 106 || binary.LittleEndian.Uint32(frame[14:18]) != 5 { t.Fatalf("SERVER_OBJECT_PROPERTY counted single frame=%x", frame) }\n}\n''',
        "SERVER_OBJECT_PROPERTY counted layout test")
    text = replace_once(text,
        '''\tif frames[0][0] != 0x18 || binary.LittleEndian.Uint16(frames[0][5:7]) != 228 { t.Fatalf("first frame=%x", frames[0]) }\n\tif frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][10:12]) != 105 { t.Fatalf("second frame=%x", frames[1]) }\n\tif frames[2][0] != 0x10 || binary.LittleEndian.Uint16(frames[2][10:12]) != 106 { t.Fatalf("third frame=%x", frames[2]) }\n''',
        '''\tif frames[0][0] != 0x18 || binary.LittleEndian.Uint16(frames[0][5:7]) != 1 || binary.LittleEndian.Uint16(frames[0][7:9]) != 228 { t.Fatalf("first counted frame=%x", frames[0]) }\n\tif frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][10:12]) != 1 || binary.LittleEndian.Uint16(frames[1][12:14]) != 105 { t.Fatalf("second counted frame=%x", frames[1]) }\n\tif frames[2][0] != 0x10 || binary.LittleEndian.Uint16(frames[2][10:12]) != 1 || binary.LittleEndian.Uint16(frames[2][12:14]) != 106 { t.Fatalf("third counted frame=%x", frames[2]) }\n''',
        "current bag counted frame test")
    test.write_text(text, encoding="utf-8")

    trans = probe / "scene_transition.go"
    text = trans.read_text(encoding="utf-8")
    text = replace_once(text,
        '''\tif err := sendPlayerAddObject(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {\n\t\treturn fmt.Errorf("write player re-entry archive: %w", err)\n\t}\n\t// Latest-client re-entry A/B: the working initial-entry chain supplies the\n\t// authoritative player location/vitals before the explicit ClientReady. The\n\t// failed target-scene chain stopped after AddObject and never produced 0x09.\n\t// Replay only already-recovered player state here, then still require the real\n\t// target ClientReady before materializing target NPCs.\n\tif err := sendPlayerLocationAndVitals(r.conn, r.player, destination.location.Position); err != nil {\n\t\treturn fmt.Errorf("write early target player location/vitals: %w", err)\n\t}\n\tclear(r.activeNPCs)\n''',
        '''\t// 2026-09-14 LIVE reached OnEntryScene create new after the 2750ms drain but\n\t// never reached flow player data ready.  Reuse the same already-recovered\n\t// player spawn sequence that succeeds on initial entry: AddObject, Snapshot608,\n\t// Appearance, then Location/Vitals.  This is an A/B discriminator, not a claim\n\t// that any one of those post-AddObject frames is independently proven causal.\n\tif err := sendPlayerSpawn(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {\n\t\treturn fmt.Errorf("write full player re-entry spawn: %w", err)\n\t}\n\tlog.Printf("%s: latest-client REENTRY-FULL-SPAWN replayed initial-entry player spawn sequence", r.remote)\n\tclear(r.activeNPCs)\n''',
        "re-entry full player spawn")
    trans.write_text(text, encoding="utf-8")

    print(f"patched {helper}")
    print(f"patched {test}")
    print(f"patched {trans}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
