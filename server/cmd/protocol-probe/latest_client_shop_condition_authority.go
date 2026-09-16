package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Exact-current resource fingerprints recovered independently from the current
// share.package payload. The two condition_formula payloads differ only in
// formula ids 122425/122426; neither is reachable from any of the 8,575 current
// mode3 ExchangeData roots. Both produce the identical mode3 reachable graph.
const (
	exactCurrentShopINISHA256          = "f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9"
	exactCurrentExchangeItemSHA256     = "ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6"
	exactCurrentConditionINISHA256     = "b4fcbb6934617658af6f0e75f7e06734e1dd776b72d6fdf4b687643c1b778d3f"
	exactCurrentSkillMaxLevelSHA256    = "d4a35d90b84921f733afd632b15558bebe4ccdf474cc9ef5d5b6bc636bd99829"
	exactCurrentFormulaSHA256A         = "9a4053f5bd4ac0e3895fe42a44db8bbbf44a469e0cf6fe226571dc3e7fcd1b57"
	exactCurrentFormulaSHA256B         = "272bdce8cf161a70dbaea5fb828e7c4905e0ffff3402073d9c1909c895af9862"
	exactCurrentMode3ExchangeDataCount = 8575
	exactCurrentMode3LeafCount         = 872
	exactCurrentExactLeafCount         = 327
	exactCurrentSafeExchangeDataCount  = 2422
)

type currentShopConditionAuthority struct {
	catalog       *shopConditionCatalog
	definitions   map[int32]exchangeConditionSpec
	skillMaxTable iniTable
	audit         shopExchangeConditionCapabilityAudit
}

var (
	currentShopConditionAuthorityOnce  sync.Once
	currentShopConditionAuthorityValue *currentShopConditionAuthority
	currentShopConditionAuthorityErr   error
)

func fileSHA256Hex(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func requireFileSHA256(path string, allowed ...string) error {
	got, err := fileSHA256Hex(path)
	if err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}
	for _, want := range allowed {
		if strings.EqualFold(got, want) {
			return nil
		}
	}
	return fmt.Errorf("resource authority mismatch %s sha256=%s", path, got)
}

func loadCurrentMode3ExchangeDataSet(path string) (map[int32]struct{}, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer file.Close()
	result := make(map[int32]struct{}, exactCurrentMode3ExchangeDataCount)
	mode3Rows, nonzeroRows := 0, 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if _, err := strconv.ParseInt(strings.TrimSpace(key), 10, 32); err != nil {
			continue
		}
		parts := strings.Split(raw, ",")
		if len(parts) < 8 {
			continue
		}
		mode, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 32)
		if err != nil || mode != 3 {
			continue
		}
		mode3Rows++
		exchangeData, err := strconv.ParseInt(strings.TrimSpace(parts[7]), 10, 32)
		if err != nil || exchangeData == 0 {
			continue
		}
		nonzeroRows++
		result[int32(exchangeData)] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, 0, err
	}
	return result, mode3Rows, nonzeroRows, nil
}

func loadExchangeConditionSpecs(path string, used map[int32]struct{}) (map[int32]exchangeConditionSpec, error) {
	table, err := loadINISections(path)
	if err != nil {
		return nil, err
	}
	result := make(map[int32]exchangeConditionSpec, len(used))
	for id := range used {
		fields, ok := table[strconv.FormatInt(int64(id), 10)]
		if !ok {
			return nil, fmt.Errorf("ExchangeItem missing current mode3 ExchangeData %d", id)
		}
		result[id] = exchangeConditionSpec{
			Condition:  strings.TrimSpace(iniValue(fields, "Condition")),
			Condition2: strings.TrimSpace(iniValue(fields, "Condition2")),
			Filters:    strings.TrimSpace(iniValue(fields, "Filters")),
		}
	}
	return result, nil
}

func loadDefaultCurrentShopConditionAuthority() (*currentShopConditionAuthority, error) {
	currentShopConditionAuthorityOnce.Do(func() {
		currentShopConditionAuthorityValue, currentShopConditionAuthorityErr = buildCurrentShopConditionAuthority(
			defaultShopINIPath, defaultExchangeItemINIPath, defaultConditionINIPath, defaultConditionFormulaINIPath,
			filepath.Join(defaultModernShareRoot, "skill", "skill_maxlevel.ini"),
		)
	})
	return currentShopConditionAuthorityValue, currentShopConditionAuthorityErr
}

func buildCurrentShopConditionAuthority(shopPath, exchangePath, conditionPath, formulaPath, skillMaxLevelPath string) (*currentShopConditionAuthority, error) {
	if err := requireFileSHA256(shopPath, exactCurrentShopINISHA256); err != nil {
		return nil, err
	}
	if err := requireFileSHA256(exchangePath, exactCurrentExchangeItemSHA256); err != nil {
		return nil, err
	}
	if err := requireFileSHA256(conditionPath, exactCurrentConditionINISHA256); err != nil {
		return nil, err
	}
	if err := requireFileSHA256(formulaPath, exactCurrentFormulaSHA256A, exactCurrentFormulaSHA256B); err != nil {
		return nil, err
	}
	if err := requireFileSHA256(skillMaxLevelPath, exactCurrentSkillMaxLevelSHA256); err != nil {
		return nil, err
	}
	skillMaxTable, err := loadINISections(skillMaxLevelPath)
	if err != nil {
		return nil, fmt.Errorf("load exact current skill max-level table: %w", err)
	}

	used, mode3Rows, nonzeroRows, err := loadCurrentMode3ExchangeDataSet(shopPath)
	if err != nil {
		return nil, fmt.Errorf("load exact current shop mode3 set: %w", err)
	}
	if mode3Rows != 42985 || nonzeroRows != 42936 || len(used) != exactCurrentMode3ExchangeDataCount {
		return nil, fmt.Errorf("current mode3 invariant mismatch rows=%d nonzero=%d unique=%d", mode3Rows, nonzeroRows, len(used))
	}
	definitions, err := loadExchangeConditionSpecs(exchangePath, used)
	if err != nil {
		return nil, err
	}
	catalog, err := loadShopConditionCatalog(conditionPath, formulaPath)
	if err != nil {
		return nil, err
	}
	audit, err := auditShopExchangeConditionCapabilities(definitions, catalog.resolve, func(id int32) conditionCapabilityClass {
		return exactCurrentConditionCapability(catalog, id)
	})
	if err != nil {
		return nil, err
	}
	if len(audit.UniqueLeaves) != exactCurrentMode3LeafCount {
		return nil, fmt.Errorf("current mode3 leaf invariant mismatch=%d want=%d", len(audit.UniqueLeaves), exactCurrentMode3LeafCount)
	}
	if audit.ClassCounts[conditionCapabilityExactEvaluator] != exactCurrentExactLeafCount {
		return nil, fmt.Errorf("current exact leaf invariant mismatch=%d want=%d", audit.ClassCounts[conditionCapabilityExactEvaluator], exactCurrentExactLeafCount)
	}
	if audit.SafeExchangeData != exactCurrentSafeExchangeDataCount {
		return nil, fmt.Errorf("current SAFE ExchangeData invariant mismatch=%d want=%d", audit.SafeExchangeData, exactCurrentSafeExchangeDataCount)
	}
	return &currentShopConditionAuthority{
		catalog:       catalog,
		definitions:   definitions,
		skillMaxTable: skillMaxTable,
		audit:         audit,
	}, nil
}
