package main

import "testing"

func TestMissingColumns(t *testing.T) {
	found := map[string]map[string]bool{"role_bag_items": {"role_id": true, "slot": true}}
	got := missingColumns(found, "role_bag_items", []string{"slot", "max_hardiness", "amount", "role_id"})
	if len(got) != 2 || got[0] != "amount" || got[1] != "max_hardiness" {
		t.Fatalf("unexpected missing columns %v", got)
	}
}

func TestChecksumComparison(t *testing.T) {
	var want [32]byte
	want[0] = 12
	got := make([]byte, 32)
	got[0] = 12
	if !equalChecksum(got, want) {
		t.Fatal("matching checksum rejected")
	}
	got[0] = 13
	if equalChecksum(got, want) {
		t.Fatal("mismatch accepted")
	}
}
