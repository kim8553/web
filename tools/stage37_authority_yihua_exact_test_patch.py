from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".")
rel = Path("cmd/protocol-probe/neigong_passives_test.go")
path = root / rel
text = path.read_text(encoding="utf-8")
pattern = r"func TestYihuaPassiveTriggersNativeBuffAndHealsOnDodge\(t \*testing\.T\) \{.*?\n\}"
replacement = r'''func TestYihuaCurrentMindBuffDoesNotImplicitlyAliasLegacyPassive(t *testing.T) {
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
}'''
updated, count = re.subn(pattern, replacement, text, count=1, flags=re.S)
if count != 1:
    raise SystemExit(f"{rel}: expected exact YiHua trigger test once, found {count}")
path.write_text(updated, encoding="utf-8")
print(f"patched {rel}: exact current mind_buf/direct-key contract")
