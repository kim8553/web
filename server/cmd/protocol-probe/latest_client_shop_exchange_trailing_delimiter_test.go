package main

import "testing"

// These cases exercise the production validator itself in an isolated Go test.
// They do not emulate a client, consume inventory, or authorize purchase.
func TestCurrentNativeItemFinalSemicolon(t *testing.T) {
	valid := []string{
		"item_20181211jm_fc_001,1;item_20181211_jmdwt,1;item_20181211_xtx,10;",
		"Item_xdm_exchange01,440;item_exc_fc_mml,80;",
		"item_a,1;",
		"item_a,1;item_b,2",
	}
	for _, input := range valid {
		if err := validateExchangePairList(input, 32, "Item"); err != nil {
			t.Errorf("current Item grammar wrongly rejects %q: %v", input, err)
		}
	}
}

func TestCurrentNativeItemInternalGarbageStillRejected(t *testing.T) {
	invalid := []string{
		";", "item_a,1;;", "item_a,1;;item_b,2", ";item_a,1",
		"item_a,1;bad", "item_a,1;item_b,not-a-count", "item_a,1;item_b",
		"item_a,not-a-count;",
	}
	for _, input := range invalid {
		if err := validateExchangePairList(input, 32, "Item"); err == nil {
			t.Errorf("invalid Item grammar unexpectedly accepted %q", input)
		}
	}
}

func TestPropFinalSemicolonIsNotEnabledWithoutNativeEvidence(t *testing.T) {
	if err := validateExchangePairList("CapitalType1,10;", 64, "Prop"); err == nil {
		t.Fatal("unproven Prop trailing separator was enabled")
	}
	if err := validateExchangePairList("CapitalType1,10", 64, "Prop"); err != nil {
		t.Fatal(err)
	}
}
