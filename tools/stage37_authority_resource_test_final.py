from pathlib import Path
import re, sys
root = Path(sys.argv[1] if len(sys.argv) > 1 else '.')

def read(rel): return (root/rel).read_text(encoding='utf-8')
def write(rel,s): (root/rel).write_text(s,encoding='utf-8')

def replace_test(rel,name,body):
    text=read(rel)
    pat=re.compile(rf'func {re.escape(name)}\(t \*testing\.T\) \{{')
    m=pat.search(text)
    if not m: raise SystemExit(f'{rel}: {name} not found')
    i,j,depth,quote,esc=m.start(),m.end(),1,None,False
    while j<len(text) and depth:
        c=text[j]
        if quote:
            if esc: esc=False
            elif c=='\\': esc=True
            elif c==quote: quote=None
        else:
            if c in ('"',"'",'`'): quote=c
            elif c=='{': depth+=1
            elif c=='}': depth-=1
        j+=1
    if depth: raise SystemExit(f'{rel}: unterminated {name}')
    body=body.replace(r'\t','\t')
    new=f'func {name}(t *testing.T) {{\n{body.rstrip()}\n}}'
    write(rel,text[:i]+new+text[j:])
    print(f'patched {rel}: {name}')

replace_test('cmd/protocol-probe/neigong_passives_test.go','TestYihuaPassiveTriggersNativeBuffAndHealsOnDodge',r'''\tstart := time.Unix(1_700_000_000, 0).UTC()
\tfor second := int64(0); second < 1000; second++ {
\t\tplayer := newPlayerActor("明玉测试", 0)
\t\tif err := learnAuthorityNeiGongAtLevel(player, "ng_yh_001", 32); err != nil { t.Fatal(err) }
\t\tif err := player.equipNeiGong("ng_yh_001"); err != nil { t.Fatal(err) }
\t\tstate := player.actor.Snapshot()
\t\t// Equipping current neigong raises MaxHP but does not refill current HP.
\t\t// Damage relative to current HP so the actor stays alive and has heal headroom.
\t\tdamage := state.HP / 2
\t\tif damage <= 0 { t.Fatalf("invalid current HP=%d MaxHP=%d", state.HP, state.MaxHP) }
\t\tplayer.actor.ApplyDamage(damage)
\t\tbefore := player.actor.Snapshot().HP
\t\tnow := start.Add(time.Duration(second) * time.Second)
\t\toutcome := player.resolveNPCAttack(77, 10, now)
\t\tif !outcome.innerPowerTriggered || !outcome.dodged { continue }
\t\tif outcome.innerPowerHeal <= 0 || player.actor.Snapshot().HP <= before { t.Fatalf("passive did not heal: %+v", outcome) }
\t\tif got := player.bufferInfo(yihuaPassiveBuffSlot, now); !strings.HasPrefix(got, "3333,") { t.Fatalf("passive BufferInfo=%q", got) }
\t\tif got := player.bufferListString(now); !strings.Contains(got, "buf_ng_yh_001_5,local-10,3,"+sceneIdent(playerObjectID, playerOwnerID)+",") { t.Fatalf("passive BufferListStr=%q", got) }
\t\treturn
\t}
\tt.Fatal("no deterministic current-client proc+dodge tick")''')

replace_test('cmd/protocol-probe/skill_combat_test.go','TestHandleSelfSkillSpendsMPAndStartsCooldown',r'''\tplayer := newPlayerActor("tester", 0)
\tlearnAuthoritySkill(player, "CS_yhwq_hsqs05", 1)
\tbeforeMP := player.actor.Snapshot().MP
\tconn := &captureMessageConnection{}
\thandled, err := handleSkillCustom(conn, player, nil, useSkillMessage("CS_yhwq_hsqs05"), "test")
\tif err != nil { t.Fatal(err) }
\tif !handled { t.Fatal("skill 211 was not handled") }
\tif got := player.actor.Snapshot().MP; got != beforeMP-11 { t.Fatalf("MP=%d, want %d", got, beforeMP-11) }
\tframes := conn.Frames()
\tif len(frames) < 3 || frames[0][0] != 0x11 { t.Fatalf("current self-skill frames=%x", frames) }
\tif got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordCooldown { t.Fatalf("cooldown record index=%d, want %d", got, recordCooldown) }
\thasVital, hasAction := false, false
\tfor _, frame := range frames {
\t\tif len(frame) == 0 { continue }
\t\tif frame[0] == 0x10 { hasVital = true }
\t\tif frame[0] == 0x1E { hasAction = true }
\t}
\tif !hasVital || !hasAction { t.Fatalf("current self-skill route missing vital/action: %x", frames) }
\tif got := player.currentSkillSnapshot(); got != "CS_yhwq_hsqs05" { t.Fatalf("CurSkillID=%q", got) }
\t_, effectID, level, target := player.currentSkillStateSnapshot()
\tif effectID != "CS_yhwq_hsqs05" || level != 1 { t.Fatalf("skill effect state=%q level=%d", effectID, level) }
\tif want := uint64(playerObjectID) | uint64(playerOwnerID)<<32; target != want { t.Fatalf("CurSkillTarget=%x want=%x", target, want) }
\tif got := player.bufferInfo(yiHuaSkill05BuffSlot, time.Now()); got == "" { t.Fatal("fifth self skill did not stage buf_CS_yhwq_hsqs05") }
\tif got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_CS_yhwq_hsqs05") || !strings.Contains(got, "buf_fixed_red") { t.Fatalf("BufferListStr=%q", got) }
\tif rules := player.buffRules(time.Now()); !rules.CantHitEffect || !rules.CantBeSkillLocked { t.Fatalf("red super armor rules=%+v", rules) }
\tbeforeFrames := len(frames)
\thandled, err = handleSkillCustom(conn, player, nil, useSkillMessage("CS_yhwq_hsqs05"), "test")
\tif err != nil || !handled { t.Fatalf("second handle handled=%t err=%v", handled, err) }
\tif got := player.actor.Snapshot().MP; got != beforeMP-11 { t.Fatalf("cooldown rejection still spent MP: %d", got) }
\tif got := len(conn.Frames()); got != beforeFrames { t.Fatalf("cooldown rejection emitted frames: before=%d after=%d", beforeFrames, got) }''')

replace_test('cmd/protocol-probe/skill_combat_test.go','TestHandleSkillFourStagesNativeVisibleBuff',r'''\tplayer := newPlayerActor("tester", 0)
\tlearnAuthoritySkill(player, "CS_yhwq_hsqs04", 1)
\tconn := &captureMessageConnection{}
\thandled, err := handleSkillCustom(conn, player, nil, useSkillMessage("CS_yhwq_hsqs04"), "test")
\tif err != nil || !handled { t.Fatalf("handled=%t err=%v", handled, err) }
\tif got := player.bufferInfo(yiHuaSkillBuffSlot, time.Now()); got == "" { t.Fatal("self skill did not stage native visible Buff") }
\tif got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_CS_yhwq_hsqs04") { t.Fatalf("BufferListStr=%q", got) }
\tframes := conn.Frames()
\tif len(frames) < 3 || frames[0][0] != 0x11 { t.Fatalf("current skill-four frames=%x", frames) }
\tif got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordCooldown { t.Fatalf("cooldown record index=%d want=%d", got, recordCooldown) }
\thasVital, hasAction := false, false
\tfor _, frame := range frames {
\t\tif len(frame) == 0 { continue }
\t\tif frame[0] == 0x10 { hasVital = true }
\t\tif frame[0] == 0x1E { hasAction = true }
\t}
\tif !hasVital || !hasAction { t.Fatalf("current skill-four route missing vital/action: %x", frames) }''')

replace_test('cmd/protocol-probe/skill_combat_test.go','TestTaiJiQuanGuPuMetadataAndThreeSegmentAction',r'''\tif len(taiJiQuanGuPuCombatSkills) != 8 { t.Fatalf("tai ji skill count=%d, want 8", len(taiJiQuanGuPuCombatSkills)) }
\tdefinition := taiJiQuanGuPuCombatSkills["CS_wd_tjq08"]
\tif definition.spCost != 50 || definition.baseDamage != 1304 || definition.personalCD != 10*time.Second || definition.publicCD != 10*time.Second || definition.targetMode != skillTargetSelf || definition.areaRadius != 0 || definition.range_ != 5 { t.Fatalf("开太极 metadata=%+v", definition) }
\t// Mirror current parser float64->Duration truncation of the exact resource text.
\twantSecond := time.Duration(4_645_416_667)
\twantThird := time.Duration(6_808_083_333)
\tif len(definition.followupActions) != 2 || definition.followupActions[0].action != "Test0001" || definition.followupActions[0].at != wantSecond || definition.followupActions[0].duration != time.Duration(2_166_666_666) || definition.followupActions[1].action != "fight_0h_jb_10" || definition.followupActions[1].at != wantThird || definition.followupActions[1].duration != time.Duration(1_933_333_333) { t.Fatalf("开太极 action sequence=%+v", definition.followupActions) }
\twant := wantThird + time.Duration(1_933_333_333)
\tif got := definition.totalActionDuration(); got != want { t.Fatalf("开太极 total action=%s, want %s", got, want) }
\tif definition.requiresTarget { t.Fatal("current 开太极 resource contract is self-target, not selected-target") }''')
