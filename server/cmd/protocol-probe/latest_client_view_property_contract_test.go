package main

import "testing"

func TestLatestClientCreateViewRejectsLegacyGlobalPropertyID(t *testing.T) {
	_, err := serverCreateViewWithProperties(serverViewSpec{ID: 61, Capacity: 1}, []serverViewProperty{viewInt(0x08A7, 1)})
	if err == nil {
		t.Fatal("CreateView accepted legacy global property 0x08A7")
	}
}

func TestLatestClientCreateViewRejectsWrongOrdinalType(t *testing.T) {
	// ordinal 181 is MaxPowerValue/int32 in the currently negotiated table.
	_, err := serverCreateViewWithProperties(serverViewSpec{ID: 61, Capacity: 1}, []serverViewProperty{viewString(181, "bad")})
	if err == nil {
		t.Fatal("CreateView accepted string at MaxPowerValue/int32 ordinal 181")
	}
}

func TestLatestClientViewAddRejectsWrongOrdinalType(t *testing.T) {
	_, err := serverViewAdd(61, 0, []serverViewProperty{viewString(181, "bad")})
	if err == nil {
		t.Fatal("ViewAdd accepted string at MaxPowerValue/int32 ordinal 181")
	}
}

func TestLatestClientStarterBagCreateDropsUnnegotiatedBaseCapProperties(t *testing.T) {
	frame, err := serverCreateViewWithProperties(serverViewSpec{ID: 2, Capacity: 132, BaseCap: 6}, []serverViewProperty{
		viewInt(0x00E8, 6), viewInt(0x08A7, 6),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 7 || frame[5] != 0 || frame[6] != 0 {
		t.Fatalf("starter bag CreateView still emitted BaseCap properties: %x", frame)
	}
}
