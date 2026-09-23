package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	modernSkillRoot       = filepath.Join(defaultModernShareRoot, "skill")
	modernSkillActionRoot = defaultModernActionRoot
)
var modernPlayerSkillActionFiles = []string{"zhaoshi_player.ini", "zhaoshi_player_2.ini", "zhaoshi_player_dodge.ini", "zhaoshi_player_parry.ini"}

type iniField struct {
	key   string
	value string
}
type iniTable map[string][]iniField

func loadINISections(path string) (iniTable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	table := make(iniTable)
	section := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\ufeff") {
			line = line[len("\ufeff"):]
		}
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' && line[len(line)-1] == ']' {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if _, exists := table[section]; !exists {
				table[section] = nil
			}
			continue
		}
		if section == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		table[section] = append(table[section], iniField{key: key, value: value})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return table, nil
}
func iniValue(fields []iniField, key string) string {
	for i := len(fields) - 1; i >= 0; i-- {
		if strings.EqualFold(fields[i].key, key) {
			return fields[i].value
		}
	}
	return ""
}
func iniInt(fields []iniField, key string) int32 {
	value, _ := strconv.ParseInt(strings.TrimSpace(iniValue(fields, key)), 10, 32)
	return int32(value)
}
func iniFloat(fields []iniField, key string) float32 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(iniValue(fields, key)), 32)
	return float32(value)
}

type skillResourceTables struct {
	skillNew     iniTable
	skillStatic  iniTable
	skillReplace iniTable
	normalVar    iniTable
	lockVar     iniTable
	consume     iniTable
	damage      iniTable
	hitShape    iniTable
	targetShape iniTable
	actions     iniTable
	buffNew     iniTable
	buffStatic  iniTable
	buffVar     iniTable
}

func loadSkillResourceTables() (skillResourceTables, error) {
	load := func(path string) (iniTable, error) {
		table, err := loadINISections(path)
		if err != nil {
			return nil, fmt.Errorf("load skill resource %s: %w", filepath.Base(path), err)
		}
		return table, nil
	}
	loadOptional := func(path string) (iniTable, error) {
		table, err := loadINISections(path)
		if err == nil {
			return table, nil
		}
		if os.IsNotExist(err) {
			return make(iniTable), nil
		}
		return nil, fmt.Errorf("load optional skill resource %s: %w", filepath.Base(path), err)
	}
	result := skillResourceTables{}
	var err error
	result.skillNew, err = load(filepath.Join(modernSkillRoot, "skill_new.ini"))
	if err != nil {
		return result, err
	}
	result.skillStatic, err = load(filepath.Join(modernSkillRoot, "skill_static.ini"))
	if err != nil {
		return result, err
	}
	result.skillReplace, err = loadOptional(filepath.Join(modernSkillRoot, "skill_replace.ini"))
	if err != nil {
		return result, err
	}
	result.normalVar, err = load(filepath.Join(modernSkillRoot, "skill_normal_varprop.ini"))
	if err != nil {
		return result, err
	}
	result.lockVar, err = load(filepath.Join(modernSkillRoot, "skill_lock_varprop.ini"))
	if err != nil {
		return result, err
	}
	result.consume, err = load(filepath.Join(modernSkillRoot, "skill_consume.ini"))
	if err != nil {
		return result, err
	}
	result.damage, err = load(filepath.Join(modernSkillRoot, "damage_calculate.ini"))
	if err != nil {
		return result, err
	}
	result.hitShape, err = load(filepath.Join(modernSkillRoot, "attack_hitshape.ini"))
	if err != nil {
		return result, err
	}
	result.targetShape, err = load(filepath.Join(modernSkillRoot, "attack_targetshape.ini"))
	if err != nil {
		return result, err
	}
	result.actions = make(iniTable)
	for _, name := range modernPlayerSkillActionFiles {
		actionTable, loadErr := load(filepath.Join(modernSkillActionRoot, name))
		if loadErr != nil {
			return result, loadErr
		}
		for section, fields := range actionTable {
			result.actions[section] = fields
		}
	}
	cloneActions, cloneErr := load(filepath.Join(modernSkillActionRoot, "zhaoshi_clone.ini"))
	if cloneErr != nil {
		return result, cloneErr
	}
	for section, fields := range cloneActions {
		if _, exists := result.actions[section]; exists {
			continue
		}
		result.actions[section] = fields
	}
	result.buffNew, err = load(filepath.Join(modernSkillRoot, "buff_new.ini"))
	if err != nil {
		return result, err
	}
	result.buffStatic, err = load(filepath.Join(modernSkillRoot, "buff_static.ini"))
	if err != nil {
		return result, err
	}
	result.buffVar, err = load(filepath.Join(modernSkillRoot, "buff_varprop.ini"))
	if err != nil {
		return result, err
	}
	return result, nil
}
type skillReplacementRule struct {
	baseID        string
	conditionID   int32
	replacementID string
	flag          int32
}

func parseSkillReplacementRule(baseID string, field iniField) (skillReplacementRule, bool) {
	condition, err := strconv.ParseInt(strings.TrimSpace(field.key), 10, 32)
	if err != nil {
		return skillReplacementRule{}, false
	}
	parts := strings.Split(field.value, ",")
	replacementID := strings.TrimSpace(parts[0])
	if replacementID == "" {
		return skillReplacementRule{}, false
	}
	var flag int32
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		parsed, parseErr := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if parseErr != nil {
			return skillReplacementRule{}, false
		}
		flag = int32(parsed)
	}
	return skillReplacementRule{
		baseID:        strings.TrimSpace(baseID),
		conditionID:   int32(condition),
		replacementID: replacementID,
		flag:          flag,
	}, true
}

func skillReplacementRules(table iniTable) []skillReplacementRule {
	rules := make([]skillReplacementRule, 0)
	bases := make([]string, 0, len(table))
	for baseID := range table {
		bases = append(bases, baseID)
	}
	sort.Strings(bases)
	for _, baseID := range bases {
		for _, field := range table[baseID] {
			rule, ok := parseSkillReplacementRule(baseID, field)
			if !ok {
				continue
			}
			rules = append(rules, rule)
		}
	}
	return rules
}

// conditionID == 0 is kept as a distinct resource sentinel. The current
// client condition tables do not define section 0; runtime replacement
// semantics are deliberately not inferred here.
func noConditionSkillReplacementBases(table iniTable) map[string]string {
	result := make(map[string]string)
	conflicts := make(map[string]struct{})
	for _, rule := range skillReplacementRules(table) {
		if rule.conditionID != 0 {
			continue
		}
		key := strings.ToLower(rule.replacementID)
		if existing, ok := result[key]; ok && !strings.EqualFold(existing, rule.baseID) {
			delete(result, key)
			conflicts[key] = struct{}{}
			continue
		}
		if _, conflict := conflicts[key]; conflict {
			continue
		}
		result[key] = rule.baseID
	}
	return result
}

func levelVarProp(static []iniField, script string, level int32, tables skillResourceTables) []iniField {
	table := tables.normalVar
	if strings.EqualFold(script, "SkillLock") {
		table = tables.lockVar
	}
	minimum := iniInt(static, "MinVarPropNo")
	maximum := iniInt(static, "MaxVarPropNo")
	var fallback []iniField
	for id := minimum; id <= maximum && id > 0; id++ {
		fields := table[strconv.Itoa(int(id))]
		foundLevel := iniInt(fields, "Level")
		if foundLevel == level {
			return fields
		}
		if foundLevel <= 0 || foundLevel >= level {
			continue
		}
		if fallback == nil || iniInt(fallback, "Level") < foundLevel {
			fallback = fields
		}
	}
	return fallback
}
func compileSkillDamage(fields []iniField) int32 {
	total := 0.0
	for _, field := range fields {
		if !strings.EqualFold(field.key, "r") {
			continue
		}
		parts := strings.Split(field.value, ",")
		if len(parts) < 3 {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil || value <= 0 {
			continue
		}
		total += value
	}
	return int32(math.Floor(total))
}
func compileTargetMode(fields []iniField, isDamage bool) (skillTargetMode, float32, float32, float32) {
	if !isDamage {
		return 0, 0, 0, 0
	}
	switch iniInt(fields, "HitShape") {
	case 1:
		return 2, iniFloat(fields, "HitShapePara2"), 0, 0
	case 2, 11:
		return 5, iniFloat(fields, "HitShapePara2"), 0, 0
	case 3, 9, 10:
		return 3, iniFloat(fields, "HitShapePara2"), iniFloat(fields, "HitShapePara1"), 0
	case 4, 12:
		return 4, iniFloat(fields, "HitShapePara2"), 0, iniFloat(fields, "HitShapePara1")
	default:
		return 1, 0, 0, 0
	}
}

type rawActionSegment struct {
	order    int
	action   string
	endFrame float64
	inBlend  float64
}

func compileSkillActions(fields []iniField) (string, time.Duration, []skillActionSegment, []time.Duration) {
	raw := make([]rawActionSegment, 0, 3)
	hitFrames := make([]time.Duration, 0, 4)
	for _, field := range fields {
		if strings.HasPrefix(field.key, "trigger_") {
			parts := strings.Split(field.value, ";")
			if len(parts) < 3 {
				continue
			}
			trigger := strings.TrimSpace(parts[1])
			if trigger != "trigger_hitbreak" && trigger != "trigger_hitback" {
				continue
			}
			frame, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			if err == nil && frame > 0 {
				hitFrames = append(hitFrames, time.Duration(frame/30*float64(time.Second)))
			}
			continue
		}
		order, err := strconv.Atoi(field.key)
		if err != nil {
			continue
		}
		parts := strings.Split(field.value, ";")
		if len(parts) < 5 || strings.TrimSpace(parts[0]) == "" {
			continue
		}
		endFrame, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		inBlend, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		raw = append(raw, rawActionSegment{order: order, action: strings.TrimSpace(parts[0]), endFrame: endFrame, inBlend: inBlend})
	}
	sort.Slice(hitFrames, func(i, j int) bool {
		return hitFrames[i] < hitFrames[j]
	})
	dedup := hitFrames[:0]
	for _, frame := range hitFrames {
		if len(dedup) != 0 && dedup[len(dedup)-1] == frame {
			continue
		}
		dedup = append(dedup, frame)
	}
	sort.Slice(raw, func(i, j int) bool {
		return raw[i].order < raw[j].order
	})
	if len(raw) == 0 {
		return "", 0, nil, dedup
	}
	duration := time.Duration(raw[0].endFrame / 30 * float64(time.Second))
	at := time.Duration(0)
	followups := make([]skillActionSegment, 0, len(raw)-1)
	for i := 1; i < len(raw); i++ {
		overlap := time.Duration(math.Max(raw[i].inBlend, 0) / 30 * float64(time.Second))
		at += time.Duration(raw[i-1].endFrame/30*float64(time.Second)) - overlap
		followups = append(followups, skillActionSegment{action: raw[i].action, at: at, duration: time.Duration(raw[i].endFrame / 30 * float64(time.Second))})
	}
	return raw[0].action, duration, followups, dedup
}
func compileExactSkillBuff(skillID string, level int32, tables skillResourceTables) (skillBuffDefinition, bool, bool) {
	configID := "buf_" + skillID
	buffNew := tables.buffNew[configID]
	staticData := iniInt(buffNew, "StaticData")
	if staticData <= 0 {
		return skillBuffDefinition{}, false, false
	}
	static := tables.buffStatic[strconv.Itoa(int(staticData))]
	lifetime := time.Duration(iniInt(buffNew, "LifeTime")) * time.Millisecond
	minimum := iniInt(static, "MinVarPropNo")
	maximum := iniInt(static, "MaxVarPropNo")
	for id := minimum; id <= maximum && id > 0; id++ {
		fields := tables.buffVar[strconv.Itoa(int(id))]
		if iniInt(fields, "Level") != level {
			continue
		}
		milliseconds := iniInt(fields, "LifeTime")
		if milliseconds > 0 {
			lifetime = time.Duration(milliseconds) * time.Millisecond
		}
		break
	}
	if lifetime <= 0 {
		lifetime = 24 * time.Hour
	}
	definition := skillBuffDefinition{configID: configID, staticData: uint32(staticData), level: level, lifetime: lifetime}
	return definition, iniInt(static, "IsDamage") != 0, true
}

type skillEffectOverlay struct {
	selfBuffLease time.Duration
	preferredSlot uint16
	redArmor      bool
	yellowArmor   bool
	stance        []skillBuffDefinition
}

var verifiedSkillEffectOverlays = map[string]skillEffectOverlay{"CS_yhwq_hsqs04": {selfBuffLease: 35 * time.Second, preferredSlot: 5}, "CS_yhwq_hsqs05": {preferredSlot: 6, redArmor: true}, "CS_wd_tjq05": {preferredSlot: 8}, "CS_wd_tjq07": {preferredSlot: 9}, "CS_wd_tjq08": {redArmor: true}, "CS_jh_chz04": {redArmor: true}, "CS_wd_tjj07": {yellowArmor: true}, "CS_wd_tjq04": {preferredSlot: 7, stance: []skillBuffDefinition{{configID: "buf_CS_wd_tjq04_1", staticData: 0x177d, lifetime: 24 * time.Hour, preferredSlot: 7}, {configID: "buf_CS_wd_tjq04_2", staticData: 0x17a7, lifetime: 24 * time.Hour, preferredSlot: 7}}}}

func effectsHaveKind(effects []combatSkillEffect, kind skillEffectKind) bool {
	for _, e := range effects {
		if e.kind == kind {
			return true
		}
	}
	return false
}

func compileCombatSkill(skillID string, level int32, tables skillResourceTables) (combatSkillDefinition, error) {
	return compileCombatSkillWithActionSource(skillID, level, tables, skillID)
}

func compileCombatSkillWithActionSource(skillID string, level int32, tables skillResourceTables, actionSkillID string) (combatSkillDefinition, error) {
	if level <= 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has invalid level %d", skillID, level)
	}
	newFields := tables.skillNew[skillID]
	staticData := iniInt(newFields, "StaticData")
	if staticData <= 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has no StaticData", skillID)
	}
	script := iniValue(newFields, "script")
	if !strings.EqualFold(script, "SkillNormal") && !strings.EqualFold(script, "SkillLock") {
		return combatSkillDefinition{}, fmt.Errorf("%s uses unsupported script %q", skillID, script)
	}
	static := tables.skillStatic[strconv.Itoa(int(staticData))]
	if len(static) == 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has no skill_static section %d", skillID, staticData)
	}
	varProp := levelVarProp(static, script, level, tables)
	if len(varProp) == 0 {
		return combatSkillDefinition{}, fmt.Errorf("%s has no level-%d varprop", skillID, level)
	}
	consume := tables.consume[strconv.Itoa(int(iniInt(varProp, "Consume")))]
	damage := compileSkillDamage(tables.damage[strconv.Itoa(int(iniInt(varProp, "DamageCalculate")))])
	isDamage := iniInt(varProp, "IsDamage") != 0
	compiledBuff, buffHarmful, hasCompiledBuff := compileExactSkillBuff(skillID, level, tables)
	hitShape := tables.hitShape[strconv.Itoa(int(iniInt(varProp, "HitShapePkg")))]
	targetMode, radius, angle, width := compileTargetMode(hitShape, isDamage)
	targetShape := tables.targetShape[strconv.Itoa(int(iniInt(varProp, "TargetShapePkg")))]
	skillRange := iniFloat(targetShape, "TargetShapePara2")
	if iniInt(targetShape, "TargetShape") == 0 {
		skillRange = 0
	}
	if targetMode == skillTargetSelected && iniInt(hitShape, "HitShape") == 0 && skillRange > 10 && hasCompiledBuff && buffHarmful && iniInt(hitShape, "HitCount") == 0 {
		targetMode = skillTargetSelectedArea
		radius = skillRange
	}
	if strings.TrimSpace(actionSkillID) == "" {
		actionSkillID = skillID
	}
	action, actionDuration, followups, hitFrames := compileSkillActions(tables.actions[actionSkillID])
	if action == "" {
		if actionSkillID == skillID {
			return combatSkillDefinition{}, fmt.Errorf("%s has no player action timeline", skillID)
		}
		return combatSkillDefinition{}, fmt.Errorf("%s has no player action timeline via replacement source %s", skillID, actionSkillID)
	}
	publicCooldown := iniInt(varProp, "AddCoolDownTime")
	if publicCooldown <= 0 {
		publicCooldown = iniInt(static, "AddCoolDownTime")
	}
	definition := combatSkillDefinition{id: skillID, staticData: staticData, script: script, taoLu: iniValue(varProp, "TaoLu"), attribute: skillElementAttribute(skillID), level: level, cooldownCategory: iniInt(varProp, "CoolDownCategory"), cooldownTeam: iniInt(varProp, "CoolDownTeam"), personalCD: time.Duration(iniInt(varProp, "CoolDownTime")) * time.Millisecond, publicCD: time.Duration(publicCooldown) * time.Millisecond, mpCost: iniInt(consume, "AConsumeMP"), spCost: iniInt(consume, "AConsumeSP"), qgCost: iniInt(consume, "AConsumeQGP"), baseDamage: damage, requiresTarget: targetMode == skillTargetSelected || targetMode == skillTargetSelectedArea, targetMode: targetMode, range_: skillRange, areaRadius: radius, areaHeight: iniFloat(varProp, "HeightDiff"), sectorAngle: angle, areaWidth: width, actionSkillID: actionSkillID, actionName: action, followupActions: followups, hitFrames: hitFrames, actionDuration: actionDuration}
	if definition.publicCD <= 0 {
		definition.publicCD = definition.personalCD
	}
	if damage > 0 {
		definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectDamage, amount: damage})
	}
	if hasCompiledBuff {
		buff := compiledBuff
		overlay := verifiedSkillEffectOverlays[skillID]
		if overlay.selfBuffLease > 0 {
			buff.lifetime = overlay.selfBuffLease
		}
		buff.preferredSlot = overlay.preferredSlot
		kind := skillEffectPlayerBuff
		if buffHarmful {
			kind = skillEffectTargetControl
		}
		definition.effects = append(definition.effects, combatSkillEffect{kind: kind, buff: buff})
	}
	if !verifiedSkillEffectOverlays[skillID].redArmor {
		for _, c := range collectSkillBuffCandidates(skillID, level, tables) {
			if c.immunity == 44 {
				armorLife := c.lifetime
				if armorLife <= 0 {
					armorLife = time.Second
				}
				buff := skillBuffDefinition{configID: c.configID, staticData: c.staticData, level: level, lifetime: armorLife}
				definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectPlayerBuff, buff: buff})
				continue
			}
			buff := skillBuffDefinition{configID: c.configID, staticData: c.staticData, level: level, lifetime: c.lifetime}
			if c.isDamage {
				if !effectsHaveKind(definition.effects, skillEffectPlayerBuff) {
					definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectPlayerBuff, buff: buff})
				}
			} else if !effectsHaveKind(definition.effects, skillEffectTargetControl) {
				definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectTargetControl, buff: buff})
			}
		}
	}
	overlay := verifiedSkillEffectOverlays[skillID]
	if len(overlay.stance) > 0 {
		definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectStanceCycle, buffVariants: overlay.stance})
	}
	if overlay.redArmor {
		definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectRedArmor})
	}
	if overlay.yellowArmor {
		definition.effects = append(definition.effects, combatSkillEffect{kind: skillEffectYellowArmor})
	}
	return definition, nil
}

type combatSkillCacheKey struct {
	id    string
	level int32
}
type combatSkillCatalog struct {
	tables      skillResourceTables
	levelOne    map[string]combatSkillDefinition
	indexed     map[string]struct{}
	skipReasons map[string]int
	cache                      map[combatSkillCacheKey]combatSkillDefinition
	noConditionReplacementBase map[string]string
	cacheMu                    sync.RWMutex
}

func skillCompileReason(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "unsupported script"):
		return "unsupported_script"
	case strings.Contains(message, "no StaticData"):
		return "missing_static_id"
	case strings.Contains(message, "no skill_static"):
		return "missing_static"
	case strings.Contains(message, "varprop"):
		return "missing_level_varprop"
	case strings.Contains(message, "action timeline"):
		return "missing_player_action"
	default:
		return "other"
	}
}
func loadCombatSkillCatalog() (*combatSkillCatalog, error) {
	tables, err := loadSkillResourceTables()
	if err != nil {
		return nil, err
	}
	catalog := &combatSkillCatalog{
		tables:                      tables,
		levelOne:                   make(map[string]combatSkillDefinition),
		indexed:                    make(map[string]struct{}),
		skipReasons:                make(map[string]int),
		cache:                      make(map[combatSkillCacheKey]combatSkillDefinition),
		noConditionReplacementBase: noConditionSkillReplacementBases(tables.skillReplace),
	}
	ids := make([]string, 0, len(tables.skillNew))
	for id := range tables.skillNew {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		script := iniValue(tables.skillNew[id], "script")
		if !strings.EqualFold(script, "SkillNormal") && !strings.EqualFold(script, "SkillLock") {
			catalog.skipReasons["unsupported_script"]++
			continue
		}
		catalog.indexed[id] = struct{}{}
		definition, compileErr := compileCombatSkill(id, 1, tables)
		if compileErr != nil {
			catalog.skipReasons[skillCompileReason(compileErr)]++
			continue
		}
		catalog.levelOne[id] = definition
		catalog.cache[combatSkillCacheKey{id: id, level: 1}] = definition
	}
	return catalog, nil
}
func (catalog *combatSkillCatalog) definition(id string, level int32) (combatSkillDefinition, bool) {
	if catalog == nil || level <= 0 {
		return combatSkillDefinition{}, false
	}
	key := combatSkillCacheKey{id: id, level: level}
	catalog.cacheMu.RLock()
	definition, ok := catalog.cache[key]
	catalog.cacheMu.RUnlock()
	if ok {
		return definition, true
	}
	if _, ok := catalog.indexed[id]; !ok {
		return combatSkillDefinition{}, false
	}
	definition, err := compileCombatSkill(id, level, catalog.tables)
	if err != nil {
		return combatSkillDefinition{}, false
	}
	catalog.cacheMu.Lock()
	catalog.cache[key] = definition
	catalog.cacheMu.Unlock()
	return definition, true
}
func (catalog *combatSkillCatalog) noConditionReplacementSource(id string) (string, bool) {
	if catalog == nil {
		return "", false
	}
	baseID, ok := catalog.noConditionReplacementBase[strings.ToLower(strings.TrimSpace(id))]
	return baseID, ok
}

func (catalog *combatSkillCatalog) noConditionReplacementDefinition(id string, level int32) (combatSkillDefinition, string, bool) {
	if catalog == nil || level <= 0 {
		return combatSkillDefinition{}, "", false
	}
	baseID, ok := catalog.noConditionReplacementSource(id)
	if !ok {
		return combatSkillDefinition{}, "", false
	}
	if _, indexed := catalog.indexed[id]; !indexed {
		return combatSkillDefinition{}, "", false
	}
	definition, err := compileCombatSkillWithActionSource(id, level, catalog.tables, baseID)
	if err != nil {
		return combatSkillDefinition{}, baseID, false
	}
	return definition, baseID, true
}

func mustLoadCombatSkillCatalog() *combatSkillCatalog {
	catalog, err := loadCombatSkillCatalog()
	if err != nil {
		panic(fmt.Sprintf("compile modern combat skill catalog: %v", err))
	}
	log.Printf("modern universal skill catalog ready indexed=%d executable_level1=%d skipped=%v action_files=%d", len(catalog.indexed), len(catalog.levelOne), catalog.skipReasons, len(modernPlayerSkillActionFiles))
	return catalog
}
func skillCatalogFamily(all map[string]combatSkillDefinition, prefix string) map[string]combatSkillDefinition {
	result := make(map[string]combatSkillDefinition)
	for id, definition := range all {
		if strings.HasPrefix(id, prefix) {
			result[id] = definition
		}
	}
	return result
}

var installedCombatSkillCatalog = mustLoadCombatSkillCatalog()
var combatSkills = installedCombatSkillCatalog.levelOne
var yiHuaWuQueCombatSkills = skillCatalogFamily(combatSkills, "CS_yhwq_hsqs")
var taiJiQuanGuPuCombatSkills = skillCatalogFamily(combatSkills, "CS_wd_tjq")
var heartBuddhaPalmCombatSkills = skillCatalogFamily(combatSkills, "CS_jh_xfz")
