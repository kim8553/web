package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type exactCurrentConditionResourcePaths struct {
	shop          string
	exchange      string
	condition     string
	formula       string
	skillMaxLevel string
}

func exactCurrentConditionTestResourcePaths(t *testing.T) exactCurrentConditionResourcePaths {
	t.Helper()
	paths := exactCurrentConditionResourcePaths{
		shop:          strings.TrimSpace(os.Getenv("JIUYIN_EXACT_CURRENT_SHOP_INI")),
		exchange:      strings.TrimSpace(os.Getenv("JIUYIN_EXACT_CURRENT_EXCHANGEITEM_INI")),
		condition:     strings.TrimSpace(os.Getenv("JIUYIN_EXACT_CURRENT_CONDITION_INI")),
		formula:       strings.TrimSpace(os.Getenv("JIUYIN_EXACT_CURRENT_CONDITION_FORMULA_INI")),
		skillMaxLevel: strings.TrimSpace(os.Getenv("JIUYIN_EXACT_CURRENT_SKILL_MAXLEVEL_INI")),
	}
	if paths.shop == "" || paths.exchange == "" || paths.condition == "" || paths.formula == "" || paths.skillMaxLevel == "" {
		t.Skip("exact-current resource paths are not configured")
	}
	return paths
}

func buildExactCurrentConditionTestAuthority(t *testing.T, paths exactCurrentConditionResourcePaths) *currentShopConditionAuthority {
	t.Helper()
	authority, err := buildCurrentShopConditionAuthority(paths.shop, paths.exchange, paths.condition, paths.formula, paths.skillMaxLevel)
	if err != nil {
		t.Fatalf("build exact-current shop condition authority: %v", err)
	}
	return authority
}

func TestExactCurrentShopConditionExhaustiveS2C508Generation(t *testing.T) {
	paths := exactCurrentConditionTestResourcePaths(t)
	authority := buildExactCurrentConditionTestAuthority(t, paths)

	// Populate only containers needed to force exact MaxLevel/Level branches to
	// execute. Result truth values are irrelevant here; capability/support and
	// response-generation safety are the audited contract.
	player := &playerActor{learnedSkills: make(map[string]int32)}
	for _, leaf := range authority.audit.UniqueLeaves {
		definition, ok := authority.catalog.Definitions[leaf]
		if !ok || definition.Type != 3 {
			continue
		}
		switch definition.Name {
		case "SkillContainer":
			if definition.Para[0] != "" {
				player.learnedSkills[definition.Para[0]] = 1
			}
		case "NeiGongContainer":
			if definition.Para[0] != "" {
				player.progress.books = append(player.progress.books, learnedNeiGong{configID: definition.Para[0], level: 1})
			}
		}
	}

	evaluator := exactCurrentConditionEvaluator{
		catalog:       authority.catalog,
		player:        player,
		skillMaxTable: authority.skillMaxTable,
	}
	ids := make([]int32, 0, len(authority.definitions))
	for id := range authority.definitions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	safe, failClosed := 0, 0
	for _, exchangeData := range ids {
		definition := authority.definitions[exchangeData]
		details, allSupported, err := evaluateExchangeConditionDetails(
			definition.Condition,
			definition.Condition2,
			definition.Filters,
			authority.catalog.resolve,
			evaluator.evaluate,
		)
		if err != nil {
			t.Fatalf("ExchangeData %d condition evaluation: %v", exchangeData, err)
		}
		capability, ok := authority.audit.ByExchangeData[exchangeData]
		if !ok {
			t.Fatalf("ExchangeData %d missing capability audit row", exchangeData)
		}
		if allSupported != capability.Safe {
			t.Fatalf("ExchangeData %d support mismatch all_supported=%t audit_safe=%t leaves=%v classes=%v",
				exchangeData, allSupported, capability.Safe, capability.Leaves, capability.Classes)
		}
		frame, err := serverShopConditionDetailsMessage(details)
		if err != nil {
			t.Fatalf("ExchangeData %d S2C508 encode: %v", exchangeData, err)
		}
		if len(frame) == 0 {
			t.Fatalf("ExchangeData %d S2C508 encoded empty frame", exchangeData)
		}
		if capability.Safe {
			safe++
		} else {
			failClosed++
		}
	}

	if safe != exactCurrentSafeExchangeDataCount {
		t.Fatalf("SAFE ExchangeData=%d want=%d", safe, exactCurrentSafeExchangeDataCount)
	}
	if failClosed != exactCurrentMode3ExchangeDataCount-exactCurrentSafeExchangeDataCount {
		t.Fatalf("FAIL_CLOSED ExchangeData=%d want=%d", failClosed, exactCurrentMode3ExchangeDataCount-exactCurrentSafeExchangeDataCount)
	}
}

func TestExactCurrentShopConditionAuthorityRejectsSkillMaxLevelDrift(t *testing.T) {
	paths := exactCurrentConditionTestResourcePaths(t)
	original, err := os.ReadFile(paths.skillMaxLevel)
	if err != nil {
		t.Fatal(err)
	}
	if len(original) == 0 {
		t.Fatal("exact-current skill_maxlevel.ini is empty")
	}
	mutated := append([]byte(nil), original...)
	mutated[len(mutated)-1] ^= 0x01
	badPath := filepath.Join(t.TempDir(), "skill_maxlevel.ini")
	if err := os.WriteFile(badPath, mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = buildCurrentShopConditionAuthority(paths.shop, paths.exchange, paths.condition, paths.formula, badPath)
	if err == nil || !strings.Contains(err.Error(), "resource authority mismatch") {
		t.Fatalf("skill_maxlevel drift error=%v, want authority mismatch", err)
	}
}
