package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	defaultConditionINIPath        = filepath.Join(defaultModernShareRoot, "rule", "condition.ini")
	defaultConditionFormulaINIPath = filepath.Join(defaultModernShareRoot, "rule", "condition_formula.ini")
)

type shopConditionDefinition struct {
	ID         int32
	Type       int32
	Name       string
	TargetType int32
	Not        int32
	Para       [6]string
	Min        string
	Max        string
	HasMin     bool
	HasMax     bool
}

type shopConditionCatalog struct {
	Definitions map[int32]shopConditionDefinition
	Formulas    map[int32]string
}

func iniValuePresent(fields []iniField, key string) (string, bool) {
	for i := len(fields) - 1; i >= 0; i-- {
		if strings.EqualFold(fields[i].key, key) {
			return fields[i].value, true
		}
	}
	return "", false
}

func loadShopConditionCatalog(conditionPath, formulaPath string) (*shopConditionCatalog, error) {
	conditionTable, err := loadINISections(conditionPath)
	if err != nil {
		return nil, fmt.Errorf("load current condition table %s: %w", conditionPath, err)
	}
	formulaTable, err := loadINISections(formulaPath)
	if err != nil {
		return nil, fmt.Errorf("load current condition formula table %s: %w", formulaPath, err)
	}
	catalog := &shopConditionCatalog{
		Definitions: make(map[int32]shopConditionDefinition, len(conditionTable)),
		Formulas:    make(map[int32]string, len(formulaTable)),
	}
	for section, fields := range conditionTable {
		rawID := strings.TrimSpace(section)
		id64, parseErr := strconv.ParseInt(rawID, 10, 32)
		if parseErr != nil {
			continue
		}
		id := int32(id64)
		minValue, hasMin := iniValuePresent(fields, "Min")
		maxValue, hasMax := iniValuePresent(fields, "Max")
		definition := shopConditionDefinition{
			ID:         id,
			Type:       iniInt(fields, "Type"),
			Name:       strings.TrimSpace(iniValue(fields, "Name")),
			TargetType: iniInt(fields, "TargetType"),
			Not:        iniInt(fields, "Not"),
			Min:        strings.TrimSpace(minValue),
			Max:        strings.TrimSpace(maxValue),
			HasMin:     hasMin,
			HasMax:     hasMax,
		}
		for i := range definition.Para {
			definition.Para[i] = strings.TrimSpace(iniValue(fields, fmt.Sprintf("Para%d", i+1)))
		}
		catalog.Definitions[id] = definition
	}
	for section, fields := range formulaTable {
		rawID := strings.TrimSpace(section)
		id64, parseErr := strconv.ParseInt(rawID, 10, 32)
		if parseErr != nil {
			continue
		}
		formula, ok := iniValuePresent(fields, "Formula")
		if !ok || strings.TrimSpace(formula) == "" {
			continue
		}
		catalog.Formulas[int32(id64)] = strings.TrimSpace(formula)
	}
	return catalog, nil
}

func (catalog *shopConditionCatalog) resolve(conditionID int32) (string, bool) {
	if catalog == nil {
		return "", false
	}
	formula, ok := catalog.Formulas[conditionID]
	return formula, ok
}
