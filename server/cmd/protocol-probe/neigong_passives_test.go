package main

import (
	"strings"
	"testing"
	"time"
)

func TestYihuaPassiveProfileUsesInstalledLevelThreeDescription(t *testing.T) {
	catalog, err := loadInnerPowerPassiveCatalog()
	if err != nil {
		t.Fatalf("load inner-power descriptions: %v", err)
	}
	if len(catalog.descriptions) < 100 {
		t.Fatalf("compiled only %d inner-power descriptions", len(catalog.descriptions))
	}
	t.Logf("inner-power source families=%d executable passive families=%d", len(catalog.descriptions), len(catalog.profiles))
	profile, ok := catalog.profiles["buf_ng_yh_001"][3]
	if !ok {
		t.Fatal("missing 明玉神功 BuffLevel 3 profile")
	}
	if profile.procChance != 50 || profile.dodgeAdd != 20 || profile.procLifetime != 7*time.Second || profile.procCooldown != 32*time.Second {
		t.Fatalf("unexpected 料敌先机 profile: %+v", profile)
	}
	if profile.healPercent != 3 || profile.healCooldown != 4*time.Second {
		t.Fatalf("unexpected 移花接玉 profile: %+v", profile)
	}
	if profile.procBuffID != "buf_ng_yh_001_5" || profile.procStatic != 13107 {
		t.Fatalf("unexpected native proc Buff: %+v", profile)
	}
}

func TestYihuaCurrentMindBuffDoesNotImplicitlyAliasLegacyPassive(t *testing.T) {
	player := newPlayerActor("明玉测试", 0)
	if err := learnAuthorityNeiGongAtLevel(player, "ng_yh_001", 32); err != nil {
		t.Fatal(err)
	}
	if err := player.equipNeiGong("ng_yh_001"); err != nil {
		t.Fatal(err)
	}
	book := player.progress.book("ng_yh_001")
	if book == nil {
		t.Fatal("missing current 移花 book")
	}
	if book.staticData != 1114 || book.buffID != "mind_buf_ng_yh_001" || book.buffLevel != 3 {
		t.Fatalf("unexpected current 移花 route: static=%d buff=%q level=%d", book.staticData, book.buffID, book.buffLevel)
	}
	if !strings.HasPrefix(book.buffID, "mind_") {
		t.Fatalf("current 移花 BufferID lost mind_ prefix: %q", book.buffID)
	}
	catalog, err := loadInnerPowerPassiveCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := catalog.profiles[book.buffID]; exists {
		t.Fatalf("current passive catalog unexpectedly aliases %q", book.buffID)
	}
	if profile, ok := player.equippedInnerPowerPassive(); ok {
		t.Fatalf("current EXE direct-key lookup unexpectedly resolved passive: %+v", profile)
	}
	// At this timestamp the legacy buf_ng_yh_001 level-3 profile would satisfy
	// proc roll 14 < 50 and boosted dodge roll 11 < 30. The current mind_buf key
	// does not alias that legacy profile, so no passive proc is authoritative here.
	now := time.Unix(1_700_000_000, 0).UTC()
	outcome := player.resolveNPCAttack(77, 10, now)
	if outcome.innerPowerTriggered || outcome.innerPowerHeal != 0 {
		t.Fatalf("current mind_buf route fabricated legacy passive: %+v", outcome)
	}
	if got := player.bufferInfo(yihuaPassiveBuffSlot, now); got != "" {
		t.Fatalf("current mind_buf route fabricated passive BufferInfo=%q", got)
	}
}
