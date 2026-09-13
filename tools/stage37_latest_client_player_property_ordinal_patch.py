#!/usr/bin/env python3
# Stage37 latest-client player property ordinal compatibility A/B.
#
# Evidence basis:
# - The currently reconstructed 0x09 property table serializes only name+wire
#   type in slice order; FieldSpec.Index is not serialized.
# - Latest fxnet2 property decode rejects an object-property index outside the
#   negotiated table count.
# - LIVE 2026-09-13 trace showed ServerAddObject/ServerObjectProperty property
#   errors before game_visual:GetPlayer not exist.
#
# This patch is deliberately narrow: only the main player's scene-object and
# snapshot property lists are translated by exact (property name, wire type)
# into the ordinal of the table that Stage37 itself actually negotiates.
# Unknown or type-mismatched player properties are omitted instead of guessed.
# NPC and view-object encoders are unchanged.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_player_property_ordinal_patch.py <buildtree>")

    root = Path(sys.argv[1])
    probe = root / "cmd" / "protocol-probe"

    compat = probe / "latest_client_player_property_ordinal_compat.go"
    compat.write_text(r'''package main

import "github.com/local/9yin-go-server/internal/clientdata"

type latestClientPlayerPropertyKey struct {
	name string
	typ  clientdata.WireType
}

var latestClientPlayerWirePropertyOrdinals, latestClientPlayerWirePropertyTableCount = func() (map[latestClientPlayerPropertyKey]uint16, int) {
	fields := sceneVisiblePropertyFields(clientdata.VisibleNPCModernV1().Fields)
	ordinals := make(map[latestClientPlayerPropertyKey]uint16, len(fields))
	for ordinal, field := range fields {
		key := latestClientPlayerPropertyKey{name: field.Name, typ: field.Type}
		// The 0x09 encoder transmits fields in slice order and does not serialize
		// FieldSpec.Index.  When the same name exists with a different wire type
		// (for example Force), the exact wire type disambiguates it.  For an exact
		// duplicate, keep the first negotiated ordinal.
		if _, exists := ordinals[key]; exists {
			continue
		}
		ordinals[key] = uint16(ordinal)
	}
	return ordinals, len(fields)
}()

// latestClientPlayerWireProperties converts current server/global property IDs
// to the ordinal indexes of the exact table Stage37 sends in opcode 0x09.
// Only exact name+wire-type matches survive.  This is intentionally not a
// guessed global remap and is applied only to the main player object.
func latestClientPlayerWireProperties(properties []clientdata.IndexedProperty) []clientdata.IndexedProperty {
	if len(properties) == 0 {
		return nil
	}
	result := make([]clientdata.IndexedProperty, 0, len(properties))
	seen := make(map[uint16]struct{}, len(properties))
	for _, property := range properties {
		ordinal, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: property.Name, typ: property.Value.Type}]
		if !ok {
			continue
		}
		if _, duplicate := seen[ordinal]; duplicate {
			continue
		}
		property.Index = ordinal
		result = append(result, property)
		seen[ordinal] = struct{}{}
	}
	return result
}
''', encoding="utf-8")

    test = probe / "latest_client_player_property_ordinal_compat_test.go"
    test.write_text(r'''package main

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

func TestLatestClientPlayerPropertyOrdinalMapMatchesNegotiatedTable(t *testing.T) {
	if latestClientPlayerWirePropertyTableCount != 228 {
		t.Fatalf("negotiated property table count=%d, want 228", latestClientPlayerWirePropertyTableCount)
	}
	cases := []struct {
		name string
		typ  clientdata.WireType
		want uint16
	}{
		{"Type", clientdata.WireByte, 0},
		{"MoveSpeed", clientdata.WireFloat32, 23},
		{"MaxHP", clientdata.WireInt32, 30},
		{"HP", clientdata.WireInt32, 32},
		{"NpcType", clientdata.WireInt32, 34},
		{"Name", clientdata.WireWideString, 36},
		{"QingGongPoint", clientdata.WireInt32, 114},
		{"Gravity", clientdata.WireFloat32, 168},
		{"Force", clientdata.WireString, 226},
		{"NewSchool", clientdata.WireString, 227},
	}
	for _, tc := range cases {
		got, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: tc.name, typ: tc.typ}]
		if !ok || got != tc.want {
			t.Fatalf("%s/%s ordinal=(%d,%v), want (%d,true)", tc.name, tc.typ, got, ok, tc.want)
		}
	}
	if _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "Job", typ: clientdata.WireString}]; ok {
		t.Fatal("Job/string must not be guessed onto the negotiated Job/int32 slot")
	}
}

func TestLatestClientPlayerWirePropertiesDropsUnknownMismatchAndDeduplicates(t *testing.T) {
	raw := []clientdata.IndexedProperty{
		{Index: 181, Name: "Type", Value: clientdata.ByteValue(2)},
		{Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(1000)},
		{Index: 1588, Name: "Job", Value: clientdata.StringValue("")},
		{Index: 405, Name: "ActionSet", Value: clientdata.StringValue("set")},
		{Index: 406, Name: "ActionSet", Value: clientdata.StringValue("set")},
		{Index: 9999, Name: "NoSuchProperty", Value: clientdata.Int32Value(1)},
	}
	got := latestClientPlayerWireProperties(raw)
	if len(got) != 3 {
		t.Fatalf("normalized count=%d, want 3: %#v", len(got), got)
	}
	want := []uint16{0, 30, 18}
	for i, index := range want {
		if got[i].Index != index {
			t.Fatalf("normalized[%d].Index=%d, want %d", i, got[i].Index, index)
		}
	}
}

func TestSceneObjectPropertiesNormalizesOnlyMainPlayer(t *testing.T) {
	raw := []clientdata.IndexedProperty{{Index: 578, Name: "MaxHP", Value: clientdata.Int32Value(1000)}}
	playerFrame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(playerFrame[12:14]); got != 30 {
		t.Fatalf("player MaxHP wire index=%d, want 30", got)
	}
	npcFrame, err := sceneObjectProperties(0x1234, 1, 0, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(npcFrame[12:14]); got != 578 {
		t.Fatalf("non-player MaxHP wire index=%d, want original 578", got)
	}
}
''', encoding="utf-8")

    obj = probe / "object_property.go"
    text = obj.read_text(encoding="utf-8")
    old = '''func sceneObjectProperties(objectID, ownerID uint32, isViewObject byte, properties []clientdata.IndexedProperty) ([]byte, error) {\n\tif len(properties) > 0xffff {\n'''
    new = '''func sceneObjectProperties(objectID, ownerID uint32, isViewObject byte, properties []clientdata.IndexedProperty) ([]byte, error) {\n\tif objectID == playerObjectID && ownerID == playerOwnerID && isViewObject == 0 {\n\t\tproperties = latestClientPlayerWireProperties(properties)\n\t}\n\tif len(properties) > 0xffff {\n'''
    text = replace_once(text, old, new, "player scene-object ordinal normalization")
    obj.write_text(text, encoding="utf-8")

    overlay = probe / "zz_recovered_overlay.go"
    text = overlay.read_text(encoding="utf-8")
    old = '''func (p *playerActor) playerSnapshot608(location role.Position, visual roleVisual) ([]byte, error) {\n\tproperties := p.playerBirthProperties(location, visual)\n\treturn buildSnapshot608Frame(snapshot608PlayerSnapshot{EntityID: uint64(playerObjectID) | uint64(playerOwnerID)<<32, X: location.X, Y: location.Y, Z: location.Z, Orient: location.Orient, Properties: properties})\n}\n'''
    new = '''func (p *playerActor) playerSnapshot608(location role.Position, visual roleVisual) ([]byte, error) {\n\tproperties := latestClientPlayerWireProperties(p.playerBirthProperties(location, visual))\n\treturn buildSnapshot608Frame(snapshot608PlayerSnapshot{EntityID: uint64(playerObjectID) | uint64(playerOwnerID)<<32, X: location.X, Y: location.Y, Z: location.Z, Orient: location.Orient, Properties: properties})\n}\n'''
    text = replace_once(text, old, new, "player snapshot608 ordinal normalization")
    overlay.write_text(text, encoding="utf-8")

    actor = probe / "player_actor.go"
    text = actor.read_text(encoding="utf-8")
    old = '''func sendPlayerAddObject(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {\n\tframe, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, player.playerBirthProperties(location, visual))\n\tif err != nil {\n'''
    new = '''func sendPlayerAddObject(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {\n\trawProperties := player.playerBirthProperties(location, visual)\n\twireProperties := latestClientPlayerWireProperties(rawProperties)\n\tlog.Printf("latest-client PLAYER-PROPERTY-ORDINAL birth raw=%d wire=%d dropped=%d table=%d", len(rawProperties), len(wireProperties), len(rawProperties)-len(wireProperties), latestClientPlayerWirePropertyTableCount)\n\tframe, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, wireProperties)\n\tif err != nil {\n'''
    text = replace_once(text, old, new, "player spawn ordinal diagnostic")
    actor.write_text(text, encoding="utf-8")

    print(f"wrote {compat}")
    print(f"wrote {test}")
    print(f"patched {obj}")
    print(f"patched {overlay}")
    print(f"patched {actor}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
