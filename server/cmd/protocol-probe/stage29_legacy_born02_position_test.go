package main

import (
	"github.com/local/9yin-go-server/internal/role"
	"testing"
)

func TestStage29LegacyBorn02OnlyExactLiveCoordinate(t *testing.T) {
	source := role.Position{X: 693.908, Y: 24.694, Z: 404.350, Orient: 3.611}
	got, changed := stage29LegacyBorn02Position(role.Scene{Resource: "born02"}, source)
	if !changed || got != (role.Position{X: 905.069, Y: 10.810, Z: 196.980, Orient: 1.610}) {
		t.Fatalf("LIVE-recovered born02 position changed=%t got=%+v", changed, got)
	}
	if source.X != 693.908 || source.Orient != 3.611 {
		t.Fatal("input position was mutated")
	}
	if again, changed := stage29LegacyBorn02Position(role.Scene{Resource: "born02"}, got); changed || again != got {
		t.Fatalf("remap not idempotent: changed=%t got=%+v", changed, again)
	}
}

func TestStage29LegacyBorn02LeavesOtherScenesAndCoordinatesUntouched(t *testing.T) {
	source := role.Position{X: 693.908, Y: 24.694, Z: 404.350, Orient: 3.611}
	cases := []struct {
		scene    string
		position role.Position
	}{
		{"city05", source},
		{"born03", source},
		{"born02", role.Position{X: 693.920, Y: source.Y, Z: source.Z, Orient: source.Orient}},
		{"born02", role.Position{X: source.X, Y: source.Y, Z: 404.400, Orient: source.Orient}},
	}
	for _, tc := range cases {
		got, changed := stage29LegacyBorn02Position(role.Scene{Resource: tc.scene}, tc.position)
		if changed || got != tc.position {
			t.Fatalf("unproven coordinate changed scene=%s changed=%t got=%+v", tc.scene, changed, got)
		}
	}
}
