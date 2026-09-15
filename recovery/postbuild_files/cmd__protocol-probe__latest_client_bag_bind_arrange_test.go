package main

import "testing"

func TestArrangeBagViewNeverMergesBoundAndUnboundStacks(t *testing.T) {
	player := &playerActor{bagItems: []bagItem{
		{ConfigID: "mat_a", ViewID: 1, Slot: 2, Amount: 2, MaxAmount: 10, BindStatus: 0},
		{ConfigID: "mat_a", ViewID: 1, Slot: 4, Amount: 3, MaxAmount: 10, BindStatus: 1},
	}}
	if _, changed := player.arrangeBagView(2); !changed {
		t.Fatal("arrange should compact sparse slots")
	}
	got := player.bagSnapshot()
	if len(got) != 2 {
		t.Fatalf("bound/unbound stacks merged: len=%d items=%+v", len(got), got)
	}
	if got[0].ConfigID != "mat_a" || got[0].Amount != 2 || got[0].BindStatus != 0 || got[0].Slot != 1 {
		t.Fatalf("unbound stack=%+v", got[0])
	}
	if got[1].ConfigID != "mat_a" || got[1].Amount != 3 || got[1].BindStatus != 1 || got[1].Slot != 2 {
		t.Fatalf("bound stack=%+v", got[1])
	}
}

func TestArrangeBagViewStillMergesSameBindStatusStacks(t *testing.T) {
	player := &playerActor{bagItems: []bagItem{
		{ConfigID: "mat_a", ViewID: 1, Slot: 2, Amount: 2, MaxAmount: 10, BindStatus: 1},
		{ConfigID: "mat_a", ViewID: 1, Slot: 4, Amount: 3, MaxAmount: 10, BindStatus: 1},
	}}
	if _, changed := player.arrangeBagView(2); !changed {
		t.Fatal("arrange should merge compatible stacks")
	}
	got := player.bagSnapshot()
	if len(got) != 1 {
		t.Fatalf("same-bind stacks not merged: len=%d items=%+v", len(got), got)
	}
	if got[0].ConfigID != "mat_a" || got[0].Amount != 5 || got[0].BindStatus != 1 || got[0].Slot != 1 {
		t.Fatalf("merged stack=%+v", got[0])
	}
}
