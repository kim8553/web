package main

import (
	"bytes"
	"encoding/binary"
	"github.com/local/9yin-go-server/internal/clientdata"
	"testing"
	"time"
)

func TestPlayerActorPublishesYiHuaFactionAsSchool(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	properties := map[string]string{}
	for _, property := range player.stateProperties() {
		if property.Name == "School" || property.Name == "Force" || property.Name == "NewSchool" {
			properties[property.Name] = property.Value.Text
		}
	}
	if properties["School"] != "" || properties["Force"] != "" || properties["NewSchool"] != "" {
		t.Fatalf("raw current actor must not synthesize faction: %v", properties)
	}
}

func TestPlayerActorPublishesNewSchoolSeparately(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	player.faction = "newschool_shenjihui"
	properties := make(map[string]string)
	for _, property := range player.stateProperties() {
		if property.Name == "School" || property.Name == "Force" || property.Name == "NewSchool" {
			properties[property.Name] = property.Value.Text
		}
	}
	if got := properties["NewSchool"]; got != "newschool_shenjihui" {
		t.Fatalf("NewSchool=%q", got)
	}
	if properties["School"] != "" || properties["Force"] != "" {
		t.Fatalf("new school fields=%v", properties)
	}
}

func TestPlayerActorSPAuthorityClampsAndPublishesCurrentValue(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	sp, frame, err := player.setSP(999)
	if err != nil {
		t.Fatal(err)
	}
	if sp != maxPlayerSP || player.skillSP() != maxPlayerSP {
		t.Fatalf("set SP=%d actor=%d, want %d", sp, player.skillSP(), maxPlayerSP)
	}
	spOrdinal, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: "SP", typ: clientdata.WireInt32}]
	if !ok {
		t.Fatal("current negotiated player table has no SP/int32 ordinal")
	}
	if frame[0] != 0x10 || !bytes.Contains(frame, append(binary.LittleEndian.AppendUint16(nil, spOrdinal), 100, 0, 0, 0)) {
		t.Fatalf("set SP frame=%x ordinal=%d", frame, spOrdinal)
	}
	sp, _, err = player.addSP(-150)
	if err != nil {
		t.Fatal(err)
	}
	if sp != 0 || player.skillSP() != 0 {
		t.Fatalf("subtracted SP=%d actor=%d, want 0", sp, player.skillSP())
	}
	sp, _, err = player.addSP(15)
	if err != nil || sp != 15 || player.skillSP() != 15 {
		t.Fatalf("added SP=%d actor=%d err=%v, want 15", sp, player.skillSP(), err)
	}
}

func TestStarterInnerPowerViewUsesNegotiatedBookFields(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	book := player.progress.book("ng_jh_001")
	if book == nil {
		t.Fatal("missing current starter ng_jh_001")
	}
	if book.staticData != 1 || book.itemType != 1002 || book.level != 1 || book.maxLevel != 20 || book.total != 750 {
		t.Fatalf("starter book=%+v", *book)
	}
	state := player.actor.Snapshot()
	hpAdd, mpAdd := player.progress.totalResourceAdds()
	if state.MaxHP != state.BaseMaxHP+hpAdd || state.MaxMP != state.BaseMaxMP+mpAdd {
		t.Fatalf("derived maxima=%+v adds=%d/%d", state, hpAdd, mpAdd)
	}
	frames, err := player.progressViewFrames()
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || frames[0][0] != 0x15 || binary.LittleEndian.Uint16(frames[0][1:]) != viewportNeiGong || binary.LittleEndian.Uint16(frames[0][3:]) != 1 {
		t.Fatalf("current View43 frames=%x", frames)
	}
	if !bytes.Contains(frames[1], []byte("ng_jh_001\x00")) || !bytes.Contains(frames[1], binary.LittleEndian.AppendUint16(nil, 7)) || !bytes.Contains(frames[1], binary.LittleEndian.AppendUint16(nil, 204)) || !bytes.Contains(frames[1], binary.LittleEndian.AppendUint16(nil, 203)) {
		t.Fatalf("starter book frame lacks current View43 ordinals ConfigID=7 StaticData=204 MaxLevel=203: %x", frames[1])
	}
}

func TestTaoHuaInnerPowerUsesClassicVisibleRoute(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	if err := learnAuthorityNeiGongAtLevel(player, "ng_th_001", 30); err != nil {
		t.Fatal(err)
	}
	book := player.progress.book("ng_th_001")
	if book == nil {
		t.Fatal("missing current 桃花岛 book")
	}
	if err := player.equipNeiGong("ng_th_001"); err != nil {
		t.Fatal(err)
	}
	if book.buffID == "" || player.bufferInfo(innerPowerBuffSlot, time.Now()) == "" || !bytes.Contains([]byte(player.bufferListString(time.Now())), []byte(neiGongDisplayBuffID(book.buffID))) {
		t.Fatalf("桃花 current visible route book=%+v list=%q", *book, player.bufferListString(time.Now()))
	}
}

func TestYiHuaInnerPowerUsesSharedClassicVisibleRoute(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	if err := learnAuthorityNeiGongAtLevel(player, "ng_yh_001", 32); err != nil {
		t.Fatal(err)
	}
	book := player.progress.book("ng_yh_001")
	if book == nil {
		t.Fatal("missing current 移花 book")
	}
	if err := player.equipNeiGong("ng_yh_001"); err != nil {
		t.Fatal(err)
	}
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Fatal(err)
	}
	effect, err := catalog.effect(book.staticData, book.level)
	if err != nil {
		t.Fatal(err)
	}
	if book.maxHPAdd != effect.stats["MaxHPAdd"] || book.maxMPAdd != effect.stats["MaxMPAdd"] {
		t.Fatalf("移花 effect=%+v expected=%+v", *book, effect)
	}
	if player.bufferInfo(innerPowerBuffSlot, time.Now()) == "" || !bytes.Contains([]byte(player.bufferListString(time.Now())), []byte(neiGongDisplayBuffID(book.buffID))) {
		t.Fatalf("移花 visible route list=%q", player.bufferListString(time.Now()))
	}
}

func TestInnerPowerC2SUpdatesNormalPropertyAndViewState(t *testing.T) {
	player := newPlayerActor("本地角色", 0)
	conn := &captureMessageConnection{}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 215}, {Type: 6, Text: "ng_jh_001"}}})
	if err != nil || !handled {
		t.Fatalf("215 handled=%v err=%v", handled, err)
	}
	before := len(conn.Frames())
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 1}, {Type: 6, Text: "ng_jh_001"}}})
	if err != nil || !handled {
		t.Fatalf("1006/1 handled=%v err=%v", handled, err)
	}
	if player.progress.facultyName != "ng_jh_001" || player.progress.facultyState != facultyStateNone || len(conn.Frames()) <= before {
		t.Fatalf("faculty ready=%+v", player.progress)
	}
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 11}}})
	if err != nil || !handled {
		t.Fatalf("1006/11 %v %v", handled, err)
	}
	if player.progress.facultyState != facultyStateConvert || player.progress.facultyStyle != facultyStyleNormal || player.actor.Snapshot().LogicState != 0 {
		t.Fatalf("normal faculty=%+v actor=%+v", player.progress, player.actor.Snapshot())
	}
	_, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 12}}})
	if err != nil {
		t.Fatal(err)
	}
	if player.progress.facultyState != facultyStateNone {
		t.Fatalf("faculty exit=%+v", player.progress)
	}
	handled, err = handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 21}}})
	if err != nil || !handled {
		t.Fatalf("1006/21 %v %v", handled, err)
	}
	if player.progress.facultyStyle != facultyStyleAct || player.actor.Snapshot().LogicState != uint8(logicStateFaculty) {
		t.Fatalf("active faculty=%+v actor=%+v", player.progress, player.actor.Snapshot())
	}
}

func TestPersistentFacultySettlesElapsedMinutes(t *testing.T) {
	progress := newPlayerProgress()
	start := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	progress.lastAdvance = start
	beforeFill := progress.books[0].fill
	beforeFaculty := progress.faculty
	slot, changed := progress.advanceFaculty(start.Add(2*time.Minute + 30*time.Second))
	if !changed || slot != 1 {
		t.Fatalf("advance changed=%t slot=%d", changed, slot)
	}
	if got := progress.books[0].fill; got != beforeFill+400 {
		t.Fatalf("fill=%d", got)
	}
	if got := progress.faculty; got != beforeFaculty-400 {
		t.Fatalf("faculty=%d", got)
	}
	if !progress.lastAdvance.Equal(start.Add(2 * time.Minute)) {
		t.Fatalf("lastAdvance=%s", progress.lastAdvance)
	}
}

func TestActiveFacultyPaysOnlyAfterCardSelection(t *testing.T) {
	progress := newPlayerProgress()
	start := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	if err := progress.facultyBegin(facultyStyleAct, start); err != nil {
		t.Fatal(err)
	}
	before := progress.books[0].fill
	slot, changed := progress.advanceFaculty(start.Add(time.Minute))
	if changed || slot != 0 || progress.books[0].fill != before {
		t.Fatalf("active timer changed=%t slot=%d fill=%d", changed, slot, progress.books[0].fill)
	}
	slot, changed, levelUp, maxLevel, err := progress.awardActiveFaculty()
	if err != nil || !changed || !levelUp || maxLevel || slot != 1 {
		t.Fatalf("award slot=%d changed=%t levelUp=%t max=%t err=%v", slot, changed, levelUp, maxLevel, err)
	}
	if progress.books[0].level != 12 || progress.books[0].fill != 750 {
		t.Fatalf("current active award book=%+v", progress.books[0])
	}
}

func TestActiveFacultyAcceptsLiveDoubleCardSelection(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	if _, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 21}}}); err != nil {
		t.Fatal(err)
	}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 23}, {Type: 5, Float64: 0}, {Type: 5, Float64: 1}}})
	if err != nil || !handled {
		t.Fatalf("card handled=%t err=%v", handled, err)
	}
	if player.progress.books[0].level != 12 || player.progress.books[0].fill != 750 {
		t.Fatalf("current card result=%+v", player.progress.books[0])
	}
	frames := conn.Frames()
	if len(frames) == 0 || !bytes.Contains(frames[len(frames)-1], []byte{2, 23, 0, 0, 0}) {
		t.Fatalf("current level-up card must end active faculty with subcommand 23: %x", frames)
	}
}

func TestStaleActiveFacultyCardSelectionDoesNotDisconnect(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	before := player.progress.books[0].fill
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 1006}, {Type: 2, Int32: 23},
		{Type: 5, Float64: 0}, {Type: 5, Float64: 1},
	}})
	if err != nil || !handled {
		t.Fatalf("stale 1006/23 handled=%t err=%v", handled, err)
	}
	if got := player.progress.books[0].fill; got != before {
		t.Fatalf("stale card selection changed fill=%d want=%d", got, before)
	}
	if frames := conn.Frames(); len(frames) != 2 {
		t.Fatalf("stale card selection replies=%d want state+play", len(frames))
	}
}

func TestActiveFacultyLevelUpNotifiesClientAfterStateAndViewRefresh(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.mu.Lock()
	player.progress.books[0].fill = player.progress.books[0].total - 1
	player.mu.Unlock()
	if _, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 21}}}); err != nil {
		t.Fatal(err)
	}
	handled, err := handlePlayerProgressCustom(conn, player, clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: 1006}, {Type: 2, Int32: 23}, {Type: 5, Float64: 0}, {Type: 5, Float64: 1}}})
	if err != nil || !handled {
		t.Fatalf("levelup handled=%t err=%v", handled, err)
	}
	if player.progress.books[0].level != 12 {
		t.Fatalf("current level=%d want=12", player.progress.books[0].level)
	}
	frames := conn.Frames()
	levelUp := []byte{2, 14, 0, 0, 0, 2, 0, 0, 0, 0}
	exit := []byte{2, 23, 0, 0, 0, 2, 0, 0, 0, 0}
	li, ei := -1, -1
	for i, f := range frames {
		if bytes.Contains(f, levelUp) {
			li = i
		}
		if bytes.Contains(f, exit) {
			ei = i
		}
	}
	if li < 0 || ei < 0 || li >= ei {
		t.Fatalf("level/exit order li=%d ei=%d frames=%x", li, ei, frames)
	}
}

func TestEncodeFacultyActiveBeginUsesRegisteredHandlerAndActBegin(t *testing.T) {
	frame, err := encodeFacultyActiveBegin()
	if err != nil {
		t.Fatalf("encode active faculty begin: %v", err)
	}
	if len(frame) < 3 || frame[0] != 0x1E || binary.LittleEndian.Uint16(frame[1:]) != 3 {
		t.Fatalf("active faculty frame header=%x", frame)
	}
	// The first typed value is the derived current C++ receiver ID.  The next
	// two are Lua faculty.on_msg's arg1=22 (open act form) and arg2=0.
	needleID := []byte{2, byte(S2CFacultyMessage), 0, 0, 0}
	if !bytes.Contains(frame, needleID) {
		t.Fatalf("active faculty frame misses message id %d: %x", S2CFacultyMessage, frame)
	}
	needle := []byte{2, 22, 0, 0, 0, 2, 0, 0, 0, 0}
	if !bytes.Contains(frame, needle) {
		t.Fatalf("active faculty frame misses 22/0 args: %x", frame)
	}
}

func TestFacultyRestoreSettlesOfflineProgress(t *testing.T) {
	progress := newPlayerProgress()
	start := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	saved := progress.snapshot()
	saved.LastAdvanceUTC = start
	saved.Books[0].Fill = 100
	progress.restore(saved, start.Add(3*time.Minute))
	if got, want := progress.books[0].fill, int32(700); got != want {
		t.Fatalf("offline fill=%d want=%d", got, want)
	}
	if progress.facultyState != facultyStateConvert || progress.facultyStyle != facultyStyleNormal {
		t.Fatalf("restored faculty state=%d style=%d", progress.facultyState, progress.facultyStyle)
	}
}
