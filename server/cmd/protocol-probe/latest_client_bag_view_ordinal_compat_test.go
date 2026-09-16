package main

import "testing"

func TestLatestClientStarterBagViewSet(t *testing.T) {
	for _, id := range []uint16{121, 2, 123, 125, 122, 3, 124, 126, 176, 174, 178, 180, 177, 175, 179, 181} {
		if !latestClientStarterBagView(id) {
			t.Fatalf("bag view %d not recognized", id)
		}
	}
	for _, id := range []uint16{40, 41, 43, 45, 46, 47, 48, 61} {
		if latestClientStarterBagView(id) {
			t.Fatalf("non-bag view %d must stay untouched", id)
		}
	}
}

func TestLatestClientBagWirePropertiesUsesRichCurrentObjectContract(t *testing.T) {
	props := []serverViewProperty{
		viewString(7, "Game_item_hp_001"),
		viewInt(0x0761, 1),
		viewInt(0x0763, 1),
		viewString(0x0765, "7100568-001-0000000001-0001"),
		viewInt(0x0766, 5),
		viewInt(0x0767, 30),
		viewInt(0x0779, 6), // unsupported historical field must stay off wire
	}
	got := latestClientBagWireProperties(props)
	if len(got) != 6 {
		t.Fatalf("bag wire property count=%d, want 6", len(got))
	}
	want := []uint16{7, 105, 106, 110, 205, 228}
	for i, index := range want {
		if got[i].index != index {
			t.Fatalf("property[%d].index=%d, want %d", i, got[i].index, index)
		}
	}
	for _, property := range got {
		if property.index == 229 {
			t.Fatal("Ident ordinal 229 must not be emitted")
		}
		if int(property.index) >= latestClientPlayerWirePropertyTableCount {
			t.Fatalf("emitted out-of-range property %d >= %d", property.index, latestClientPlayerWirePropertyTableCount)
		}
	}
}

func TestLatestClientBagWirePropertiesRecoversWeaponNestedConfigID(t *testing.T) {
	got := latestClientBagWireProperties([]serverViewProperty{
		viewNest(0x05A0, "weapon_test"),
		viewInt(0x0766, 1),
		viewInt(0x0767, 1),
	})
	if len(got) != 4 || got[0].text == nil || *got[0].text != "weapon_test" || got[1].text == nil || *got[1].text != "weapon_test" {
		t.Fatalf("nested ConfigID normalization failed: %#v", got)
	}
}
