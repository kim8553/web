package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Opt-in, catalog-only audit. It never tests real client behavior, contacts a
// database, or modifies runtime resources. To retain the per-ID ledger, set
// JIUYIN_COVERAGE_CSV to a path that does not exist yet.
func TestItemSkillCoverageAudit(t *testing.T) {
	if os.Getenv("JIUYIN_COVERAGE_AUDIT") != "1" {
		t.Skip("opt-in resource-backed coverage audit")
	}
	items, err := loadItemCatalog(filepath.Join(defaultModernShareRoot, "item", "tool_item.ini"))
	if err != nil {
		t.Fatal(err)
	}
	equipment, err := loadEquipCatalog(filepath.Join(defaultModernShareRoot, "item", "equipment.ini"))
	if err != nil {
		t.Fatal(err)
	}
	type auditSummary struct {
		ToolCatalogDefinitions      int            `json:"tool_catalog_definitions"`
		EquipCatalogDefinitions     int            `json:"equip_catalog_definitions"`
		ToolUnwiredFuncBuffer       int            `json:"tool_unwired_func_buffer_metadata"`
		ToolSupportedYufengMetadata int            `json:"tool_yufeng_func_buffer_metadata"`
		ToolUseEffectMetadata       int            `json:"tool_use_effect_metadata"`
		SkillNewSectionsTotal       int            `json:"skill_new_sections_total"`
		SkillOtherScriptSections    int            `json:"skill_other_script_sections"`
		SkillIndexed                int            `json:"skill_normal_or_lock_indexed"`
		SkillLevelOneCompiled       int            `json:"skill_level_one_compiled"`
		SkillCompileFailures        map[string]int `json:"skill_compile_failure_reasons"`
		ClientLiveValidated         bool           `json:"client_live_validated"`
		DatabaseValidated           bool           `json:"database_validated"`
	}
	summary := auditSummary{ToolCatalogDefinitions: len(items.byID), EquipCatalogDefinitions: len(equipment.byID), SkillNewSectionsTotal: len(installedCombatSkillCatalog.tables.skillNew), SkillIndexed: len(installedCombatSkillCatalog.indexed), SkillLevelOneCompiled: len(installedCombatSkillCatalog.levelOne), SkillCompileFailures: make(map[string]int)}
	summary.SkillOtherScriptSections = summary.SkillNewSectionsTotal - summary.SkillIndexed
	var ledger *csv.Writer
	if path := os.Getenv("JIUYIN_COVERAGE_CSV"); path != "" {
		file, createErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr != nil {
			t.Fatalf("create new audit ledger %q: %v", path, createErr)
		}
		defer file.Close()
		ledger = csv.NewWriter(file)
		if writeErr := ledger.Write([]string{"kind", "config_id", "definition_loaded", "level_one_compilation", "effect_metadata", "client_live_e2e"}); writeErr != nil {
			t.Fatal(writeErr)
		}
		defer func() {
			ledger.Flush()
			if flushErr := ledger.Error(); flushErr != nil {
				t.Errorf("write audit ledger: %v", flushErr)
			}
		}()
	}
	write := func(kind, id, compiled, metadata string) {
		if ledger == nil {
			return
		}
		if writeErr := ledger.Write([]string{kind, id, "yes", compiled, metadata, "not_tested"}); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	itemIDs := make([]string, 0, len(items.byID))
	for id := range items.byID {
		itemIDs = append(itemIDs, id)
	}
	sort.Strings(itemIDs)
	for _, id := range itemIDs {
		item := items.byID[id]
		metadata := []string{"script=" + item.Script, "type=" + strconv.FormatInt(int64(item.ItemType), 10)}
		if item.FuncBuffer != "" {
			if item.FuncBuffer == "buf_ride_yufeng" {
				summary.ToolSupportedYufengMetadata++
				metadata = append(metadata, "func_buffer=source_handler_yufeng")
			} else {
				summary.ToolUnwiredFuncBuffer++
				metadata = append(metadata, "func_buffer=not_wired:"+item.FuncBuffer)
			}
		}
		if item.ToolUseEffect != "" {
			summary.ToolUseEffectMetadata++
			metadata = append(metadata, "tool_use_effect=present")
		}
		write("tool_item", id, "not_applicable", strings.Join(metadata, ";"))
	}
	equipIDs := make([]string, 0, len(equipment.byID))
	for id := range equipment.byID {
		equipIDs = append(equipIDs, id)
	}
	sort.Strings(equipIDs)
	for _, id := range equipIDs {
		write("equipment", id, "not_applicable", "equipment_definition_only")
	}
	skillIDs := make([]string, 0, len(installedCombatSkillCatalog.indexed))
	for id := range installedCombatSkillCatalog.indexed {
		skillIDs = append(skillIDs, id)
	}
	sort.Strings(skillIDs)
	for _, id := range skillIDs {
		if _, compiled := installedCombatSkillCatalog.levelOne[id]; compiled {
			write("skill", id, "compiled", "source_compiler_only")
			continue
		}
		_, compileErr := compileCombatSkill(id, 1, installedCombatSkillCatalog.tables)
		if compileErr == nil {
			t.Fatalf("%s missing cached skill without compiler error", id)
		}
		reason := skillCompileReason(compileErr)
		summary.SkillCompileFailures[reason]++
		write("skill", id, "not_compiled", reason)
	}
	for reason, count := range installedCombatSkillCatalog.skipReasons {
		if reason != "unsupported_script" && count != summary.SkillCompileFailures[reason] {
			t.Fatalf("source skip reason %s count=%d, individually audited=%d", reason, count, summary.SkillCompileFailures[reason])
		}
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("ITEM_SKILL_CATALOG_AUDIT=%s\n", encoded)
}
