package main

import (
	"strconv"
	"strings"
)

// exactCurrentConditionEvaluator implements only leaf families whose current
// native semantics are closed and whose required Stage37 state is authoritative.
// Everything else returns supported=false and is therefore fail-closed by the
// formula/detail layer.
type exactCurrentConditionEvaluator struct {
	catalog       *shopConditionCatalog
	player        *playerActor
	skillMaxTable iniTable
}

func (e exactCurrentConditionEvaluator) evaluate(conditionID int32) (bool, bool) {
	if e.catalog == nil || e.player == nil {
		return false, false
	}
	definition, ok := e.catalog.Definitions[conditionID]
	if !ok {
		return false, false
	}
	var value bool
	switch definition.Type {
	case 1:
		value, ok = e.evaluatePlayerProperty(definition)
	case 2:
		if !strings.EqualFold(definition.Name, "ToolBox") {
			return false, false
		}
		amount := e.toolBoxAmount(definition.Para[0])
		value, ok = conditionNumericRange(amount, definition)
	case 3:
		value, ok = e.evaluateContainerProperty(definition)
	default:
		return false, false
	}
	if !ok {
		return false, false
	}
	if definition.Not > 0 {
		value = !value
	}
	return value, true
}

func (e exactCurrentConditionEvaluator) evaluatePlayerProperty(definition shopConditionDefinition) (bool, bool) {
	e.player.mu.Lock()
	faction := e.player.faction
	sex := int64(e.player.sex)
	jmActCount := int64(len(e.player.activeJingMaiOrder))
	powerLevel := int64(powerLevelFromBooks(e.player.progress.books))
	e.player.mu.Unlock()

	school, force, newSchool := "", "", ""
	if strings.HasPrefix(faction, "force_") {
		force = faction
	} else if strings.HasPrefix(faction, "newschool_") {
		newSchool = faction
	} else {
		school = faction
	}

	switch definition.Name {
	case "School":
		return conditionDirectStringCompare(school, definition)
	case "Force":
		return conditionDirectStringCompare(force, definition)
	case "NewSchool":
		return conditionDirectStringCompare(newSchool, definition)
	case "Sex":
		return conditionDirectNumericCompare(sex, definition)
	case "jmActCount":
		return conditionDirectNumericCompare(jmActCount, definition)
	case "PowerLevel":
		// Current PowerLevel mode3 rows use the numeric-range branch (Para1="<>").
		if definition.Para[0] == "==" || definition.Para[0] == "!=" {
			return conditionDirectNumericCompare(powerLevel, definition)
		}
		return conditionNumericRange(powerLevel, definition)
	default:
		return false, false
	}
}

func conditionDirectStringCompare(value string, definition shopConditionDefinition) (bool, bool) {
	switch definition.Para[0] {
	case "==":
		if !definition.HasMin {
			return false, false
		}
		return value == definition.Min, true
	case "!=":
		if !definition.HasMin {
			return false, false
		}
		return value != definition.Min, true
	default:
		return false, false
	}
}

func conditionDirectNumericCompare(value int64, definition shopConditionDefinition) (bool, bool) {
	if !definition.HasMin {
		return false, false
	}
	compare, err := strconv.ParseInt(strings.TrimSpace(definition.Min), 10, 64)
	if err != nil {
		return false, false
	}
	switch definition.Para[0] {
	case "==":
		return value == compare, true
	case "!=":
		return value != compare, true
	default:
		return false, false
	}
}

func conditionNumericRange(value int64, definition shopConditionDefinition) (bool, bool) {
	// Exact-current common rule: no Min and no Max is false, not "no limit".
	if !definition.HasMin && !definition.HasMax {
		return false, true
	}
	if definition.HasMin {
		minimum, err := strconv.ParseInt(strings.TrimSpace(definition.Min), 10, 64)
		if err != nil {
			return false, false
		}
		if value < minimum {
			return false, true
		}
	}
	if definition.HasMax {
		maximum, err := strconv.ParseInt(strings.TrimSpace(definition.Max), 10, 64)
		if err != nil {
			return false, false
		}
		if value > maximum {
			return false, true
		}
	}
	return true, true
}

func (e exactCurrentConditionEvaluator) toolBoxAmount(configID string) int64 {
	e.player.mu.Lock()
	defer e.player.mu.Unlock()
	var total int64
	for _, item := range e.player.bagItems {
		if item.ConfigID != configID {
			continue
		}
		if _, ok := latestClientCurrentBagContainer(item.ViewID); !ok {
			continue
		}
		total += int64(item.Amount)
	}
	return total
}

func (e exactCurrentConditionEvaluator) evaluateContainerProperty(definition shopConditionDefinition) (bool, bool) {
	switch definition.Name {
	case "SkillContainer":
		e.player.mu.Lock()
		level, learned := e.player.learnedSkills[definition.Para[0]]
		e.player.mu.Unlock()
		if !learned || level <= 0 {
			// Native container lookup failed: exact result is unsatisfied, not unsupported.
			return false, true
		}
		var propertyValue int64
		switch definition.Para[1] {
		case "Level":
			propertyValue = int64(level)
		case "MaxLevel":
			// The authority gate hashes and parses the exact-current
			// skill_maxlevel.ini.  Evaluation must use that exact parsed table,
			// not a second global file lookup that could drift after validation.
			table := e.skillMaxTable
			if table == nil {
				return false, false
			}
			fields := table["book_"+definition.Para[0]]
			if len(fields) == 0 {
				// Some skills use numbered/suffix rows; require a >1 recovered
				// value to avoid treating the generic helper fallback=1 as authored.
				maxLevel := skillMaxLevelFromTable(table, definition.Para[0])
				if maxLevel <= 1 {
					return false, false
				}
				propertyValue = int64(maxLevel)
			} else {
				propertyValue = int64(maxCultivationLevel(fields))
			}
		default:
			return false, false
		}
		return conditionNumericRange(propertyValue, definition)

	case "NeiGongContainer":
		if definition.Para[1] != "Level" {
			return false, false
		}
		e.player.mu.Lock()
		var level int32
		found := false
		for i := range e.player.progress.books {
			if e.player.progress.books[i].configID == definition.Para[0] {
				level = e.player.progress.books[i].level
				found = true
				break
			}
		}
		e.player.mu.Unlock()
		if !found {
			return false, true
		}
		return conditionNumericRange(int64(level), definition)

	default:
		return false, false
	}
}

func exactCurrentConditionCapability(catalog *shopConditionCatalog, conditionID int32) conditionCapabilityClass {
	if catalog == nil {
		return conditionCapabilitySemanticsUnresolved
	}
	definition, ok := catalog.Definitions[conditionID]
	if !ok {
		// Includes current dangling id 26526 and id 0 sentinel until exact native
		// semantics for those ids are proven.
		return conditionCapabilitySemanticsUnresolved
	}
	switch definition.Type {
	case 1:
		switch definition.Name {
		case "School", "Force", "NewSchool", "Sex", "jmActCount", "PowerLevel":
			return conditionCapabilityExactEvaluator
		case "LastSchool", "VipStatus", "SuperVipStatus", "LeaveSchoolType", "SchoolMark",
			"Landflag", "IsGuildCaptain", "DongFangHonour", "NanGongHonour", "YanMenHonour",
			"MuRongHonour", "SwornTitleLevel", "XueDaoShaQi":
			return conditionCapabilityAuthoritativeStateMissing
		default:
			return conditionCapabilityAuthoritativeStateMissing
		}
	case 2:
		if definition.Name == "ToolBox" {
			return conditionCapabilityExactEvaluator
		}
		if definition.Name == "BufferContainer" {
			return conditionCapabilitySemanticsUnresolved
		}
		return conditionCapabilitySemanticsUnresolved
	case 3:
		switch definition.Name {
		case "SkillContainer", "NeiGongContainer":
			return conditionCapabilityExactEvaluator
		case "XueWeiContainer":
			return conditionCapabilityAuthoritativeStateMissing
		default:
			return conditionCapabilitySemanticsUnresolved
		}
	case 4:
		// NewRideRec semantics are exact for current mode3, but Stage37 has no
		// authoritative NewRideRec state.
		return conditionCapabilityAuthoritativeStateMissing
	case 5, 8, 10:
		// Native semantics are recovered; the required complete record/bitset
		// state for current mode3 rows is not authoritative in Stage37.
		return conditionCapabilityAuthoritativeStateMissing
	case 16, 19, 21:
		return conditionCapabilitySemanticsUnresolved
	default:
		return conditionCapabilitySemanticsUnresolved
	}
}
