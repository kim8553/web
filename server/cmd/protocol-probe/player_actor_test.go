package main

import "testing"

func TestQingGongMotionPropertiesUseModernRushRangeConfig(t *testing.T) {
	want := map[uint16]float32{0x0369: 5, 0x036C: 1.5, 0x036A: 8, 0x036D: 6.5, 0x036B: 8, 0x036E: 6.5}
	got := map[uint16]float32{}
	for _, property := range qinggongMotionProperties() {
		if _, ok := want[property.Index]; ok {
			got[property.Index] = property.Value.F32
		}
	}
	for index, expected := range want {
		if got[index] != expected {
			t.Fatalf("property %#x=%v want=%v", index, got[index], expected)
		}
	}
}
