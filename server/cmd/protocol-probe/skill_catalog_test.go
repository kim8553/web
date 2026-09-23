package main

import (
	"testing"
	"time"
)

func hasCompiledEffect(definition combatSkillDefinition, kind skillEffectKind) bool {
	for _, effect := range definition.effects {
		if effect.kind == kind {
			return true
		}
	}
	return false
}

func TestSkillReplacementRulesPreserveClientResourceFields(t *testing.T) {
	table := iniTable{
		"CS_base": {
			{key: "19443", value: "CS_conditional_hide,0"},
			{key: "0", value: "CS_no_condition_hide,1"},
		},
	}
	rules := skillReplacementRules(table)
	if len(rules) != 2 {
		t.Fatalf("replacement rules=%d, want 2", len(rules))
	}
	if got := rules[0]; got.baseID != "CS_base" || got.conditionID != 19443 || got.replacementID != "CS_conditional_hide" || got.flag != 0 {
		t.Fatalf("conditional replacement=%+v", got)
	}
	if got := rules[1]; got.baseID != "CS_base" || got.conditionID != 0 || got.replacementID != "CS_no_condition_hide" || got.flag != 1 {
		t.Fatalf("condition-zero replacement=%+v", got)
	}
}

func TestNoConditionSkillReplacementBasesOnlyIndexesConditionZero(t *testing.T) {
	table := iniTable{
		"CS_base": {
			{key: "19443", value: "CS_conditional_hide,0"},
			{key: "0", value: "CS_no_condition_hide,0"},
		},
	}
	indexed := noConditionSkillReplacementBases(table)
	if len(indexed) != 1 {
		t.Fatalf("condition-zero replacement index=%v", indexed)
	}
	if got := indexed["cs_no_condition_hide"]; got != "CS_base" {
		t.Fatalf("condition-zero replacement base=%q, want CS_base", got)
	}
	if _, exists := indexed["cs_conditional_hide"]; exists {
		t.Fatal("conditional replacement leaked into condition-zero index")
	}
}

func TestNoConditionSkillReplacementBasesDropsAmbiguousTargets(t *testing.T) {
	table := iniTable{
		"CS_base_a": {{key: "0", value: "CS_same_hide,0"}},
		"CS_base_b": {{key: "0", value: "CS_same_hide,1"}},
	}
	indexed := noConditionSkillReplacementBases(table)
	if _, exists := indexed["cs_same_hide"]; exists {
		t.Fatalf("ambiguous replacement target must not be authorized: %v", indexed)
	}
}

func TestInstalledCatalogCompilesEveryLearnedCombatSkill(t *testing.T) {
	if got := len(combatSkills); got < 3000 {
		t.Fatalf("compiled installed level-one combat skills=%d, want full client catalog", got)
	}
	if got := len(installedCombatSkillCatalog.indexed); got < len(combatSkills) {
		t.Fatalf("indexed installed skills=%d, executable level-one=%d", got, len(combatSkills))
	}
	if len(yiHuaWuQueCombatSkills) < 7 || len(taiJiQuanGuPuCombatSkills) < 8 || len(heartBuddhaPalmCombatSkills) < 8 {
		t.Fatalf("compiled families flower=%d taiJi=%d heartBuddha=%d", len(yiHuaWuQueCombatSkills), len(taiJiQuanGuPuCombatSkills), len(heartBuddhaPalmCombatSkills))
	}
	nonCombatStarter := map[string]bool{"zs_default_01": true, "CS_light_rad_81": true, "taolu_zhenfa_wzyx": true, "taolu_zhenfa_cfxz": true}
	for _, learned := range starterSkillViews {
		if nonCombatStarter[learned.configID] {
			continue
		}
		definition, exists := combatSkills[learned.configID]
		if !exists {
			t.Errorf("combat starter %s is absent from compiled catalog", learned.configID)
			continue
		}
		if definition.staticData != learned.staticData || definition.level != learned.level {
			t.Errorf("%s compiled static=%d level=%d, want %d/%d", learned.configID, definition.staticData, definition.level, learned.staticData, learned.level)
		}
		if definition.actionName == "" || definition.totalActionDuration() <= 0 {
			t.Errorf("%s has no compiled action timeline", learned.configID)
		}
	}
}

func TestCatalogCompilesLearnedSkillAtAuthoritativeLevel(t *testing.T) {
	levelTwo, ok := installedCombatSkillCatalog.definition("CS_yhwq_hsqs01", 2)
	if !ok {
		t.Fatal("花神01 level 2 did not compile")
	}
	if levelTwo.level != 2 || levelTwo.mpCost != 27 || levelTwo.baseDamage != 138 {
		t.Fatalf("花神01 level-two response=%+v", levelTwo)
	}
	levelOne := combatSkills["CS_yhwq_hsqs01"]
	if levelOne.mpCost != 21 || levelOne.baseDamage != 113 {
		t.Fatalf("花神01 cached level-one response=%+v", levelOne)
	}
}

func TestInstalledButUnlearnedSkillCannotUseUniversalExecutor(t *testing.T) {
	player := newPlayerActor("tester", 0)
	var unlearned string
	for id := range combatSkills {
		if _, learned := player.learnedSkillLevel(id); !learned {
			unlearned = id
			break
		}
	}
	if unlearned == "" {
		t.Fatal("full catalog has no unlearned skill for permission test")
	}
	before := player.actor.Snapshot()
	conn := &captureMessageConnection{}
	handled, err := handleSkillCustom(conn, player, nil, useSkillMessage(unlearned), "test")
	if err != nil || !handled {
		t.Fatalf("unlearned installed skill handled=%t err=%v", handled, err)
	}
	after := player.actor.Snapshot()
	if after.MP != before.MP || player.skillSP() != 100 || len(conn.Frames()) != 0 {
		t.Fatalf("unlearned skill mutated state before=%+v after=%+v frames=%d", before, after, len(conn.Frames()))
	}
}

func TestUniversalCooldownsAreScopedByInstalledCooldownTeam(t *testing.T) {
	player := newPlayerActor("tester", 0)
	now := time.Now()
	flower := combatSkills["CS_yhwq_hsqs04"]
	taiJi := combatSkills["CS_wd_tjq04"]
	flower.mpCost, taiJi.mpCost = 0, 0
	if ok, reason := player.beginSkillUse(flower, 0, now); !ok {
		t.Fatalf("begin flower team %d: %s", flower.cooldownTeam, reason)
	}
	if ok, reason := player.beginSkillUse(taiJi, 0, now); !ok {
		t.Fatalf("different cooldown team %d was blocked by %d: %s", taiJi.cooldownTeam, flower.cooldownTeam, reason)
	}
}

func TestCatalogClassifiesInstalledTargetShapes(t *testing.T) {
	for _, tc := range []struct {
		id        string
		wantRange float32
	}{
		{"CS_wd_tjq01", 5}, {"CS_wd_tjq08", 5}, {"CS_jh_xfz06", 20},
	} {
		d := combatSkills[tc.id]
		if d.targetMode != skillTargetSelf || d.requiresTarget || d.areaRadius != 0 || d.range_ != tc.wantRange {
			t.Fatalf("%s current target contract=%+v", tc.id, d)
		}
	}
}

func TestCatalogCompilesHeartBuddhaLevelOneResources(t *testing.T) {
	first := combatSkills["CS_jh_xfz01"]
	if first.script != "SkillLock" || first.taoLu != "" || first.mpCost != 16 || first.baseDamage != 119 ||
		first.personalCD != 6*time.Second || first.publicCD != 6*time.Second || first.targetMode != skillTargetSelf ||
		first.actionName != "xfz_01" || first.actionDuration != 59*time.Second/30 {
		t.Fatalf("心佛掌01 metadata=%+v", first)
	}
}

func TestCatalogUsesTypedEffectsForVerifiedBuffAndArmorRoutes(t *testing.T) {
	flower := combatSkills["CS_yhwq_hsqs05"]
	if !hasCompiledEffect(flower, skillEffectPlayerBuff) || !hasCompiledEffect(flower, skillEffectRedArmor) {
		t.Fatalf("花神05 effects=%+v", flower.effects)
	}
	open := combatSkills["CS_wd_tjq08"]
	if !hasCompiledEffect(open, skillEffectDamage) ||
		!hasCompiledEffect(open, skillEffectTargetControl) ||
		!hasCompiledEffect(open, skillEffectRedArmor) {
		t.Fatalf("开太极 effects=%+v", open.effects)
	}
	stance := combatSkills["CS_wd_tjq04"]
	if !hasCompiledEffect(stance, skillEffectStanceCycle) {
		t.Fatalf("太极04 effects=%+v", stance.effects)
	}
}

func TestSkillBuffAllocatorUsesNegotiatedDynamicSlots(t *testing.T) {
	player := newPlayerActor("tester", 0)
	now := time.Now()
	for index := uint32(0); index < 14; index++ {
		buff := skillBuffDefinition{
			configID:   "test_dynamic_" + string(rune('a'+index)),
			staticData: 50000 + index,
			lifetime:   time.Minute,
		}
		slot, _, _, ok := player.activateSkillBuff(buff, now)
		if !ok {
			t.Fatalf("dynamic Buff %d could not allocate", index)
		}
		want := uint16(11 + index)
		if slot != want {
			t.Fatalf("dynamic Buff %d slot=%d, want %d", index, slot, want)
		}
	}
	if _, _, _, ok := player.activateSkillBuff(skillBuffDefinition{
		configID: "overflow", staticData: 60000, lifetime: time.Minute,
	}, now); ok {
		t.Fatal("allocator accepted a 15th dynamic Buff beyond BufferInfo24")
	}
}
