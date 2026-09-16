package main

import "testing"

func TestActiveParryAcceptsModernIntegralSubtypeAsSettingsCommand(t *testing.T) {
	player := newPlayerActor("招架测试", 0)
	before := player.actor.Snapshot()
	handled, err := handleActiveParryCustom(player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomActiveParry},
		{Type: 5, Float64: 2},
		{Type: 2, Int32: 77},
	}}, "test")
	if err != nil || !handled {
		t.Fatalf("handle active parry handled=%t err=%v", handled, err)
	}
	if got := player.actor.Snapshot(); got != before {
		t.Fatalf("settings-only active parry mutated actor state: before=%+v after=%+v", before, got)
	}
}

func TestActiveParryRejectsFractionalSubtypeWithoutMutatingActorState(t *testing.T) {
	player := newPlayerActor("招架测试", 0)
	before := player.actor.Snapshot()
	handled, err := handleActiveParryCustom(player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomActiveParry},
		{Type: 5, Float64: 1.5},
		{Type: 2, Int32: 77},
	}}, "test")
	if !handled || err == nil {
		t.Fatalf("fractional subtype handled=%t err=%v", handled, err)
	}
	if got := player.actor.Snapshot(); got != before {
		t.Fatalf("fractional subtype mutated actor state: before=%+v after=%+v", before, got)
	}
}
