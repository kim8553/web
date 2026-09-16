package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentConditionCatalogPreservesAbsentVsZeroRange(t *testing.T) {
	dir := t.TempDir()
	conditionPath := filepath.Join(dir, "condition.ini")
	formulaPath := filepath.Join(dir, "condition_formula.ini")
	condition := "[1]\nType=1\nName=Sex\nPara1===\nmin=0\nNot=0\n\n[2]\nType=1\nName=PowerLevel\nPara1=<>\n\n"
	formula := "[10]\nFormula=1&2\n"
	if err := os.WriteFile(conditionPath, []byte(condition), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(formulaPath, []byte(formula), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := loadShopConditionCatalog(conditionPath, formulaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Definitions[1].HasMin || catalog.Definitions[1].Min != "0" || catalog.Definitions[1].HasMax {
		t.Fatalf("condition 1 range=%+v", catalog.Definitions[1])
	}
	if catalog.Definitions[2].HasMin || catalog.Definitions[2].HasMax {
		t.Fatalf("condition 2 unexpectedly has bounds: %+v", catalog.Definitions[2])
	}
	if got, ok := catalog.resolve(10); !ok || got != "1&2" {
		t.Fatalf("formula 10=(%q,%t)", got, ok)
	}
}

func TestCurrentConditionEvaluatorAuthoritativeSubset(t *testing.T) {
	player := &playerActor{
		faction:            "newschool_gumu",
		sex:                1,
		activeJingMaiOrder: []string{"jm1", "jm2", "jm3"},
		learnedSkills:      map[string]int32{"CS_test": 4},
		progress:           playerProgress{books: []learnedNeiGong{{configID: "ng_test", level: 7}}},
		bagItems:           []bagItem{{ConfigID: "item_a", ViewID: 1, Amount: 2}, {ConfigID: "item_a", ViewID: 3, Amount: 1}},
	}
	definitions := map[int32]shopConditionDefinition{
		1: {ID: 1, Type: 1, Name: "NewSchool", Para: [6]string{"=="}, Min: "newschool_gumu", HasMin: true},
		2: {ID: 2, Type: 1, Name: "Sex", Para: [6]string{"=="}, Min: "1", HasMin: true},
		3: {ID: 3, Type: 1, Name: "jmActCount", Para: [6]string{"=="}, Min: "3", HasMin: true},
		4: {ID: 4, Type: 2, Name: "ToolBox", Para: [6]string{"item_a"}, Min: "3", HasMin: true},
		5: {ID: 5, Type: 3, Name: "SkillContainer", Para: [6]string{"CS_test", "Level"}, Min: "4", HasMin: true},
		6: {ID: 6, Type: 3, Name: "NeiGongContainer", Para: [6]string{"ng_test", "Level"}, Min: "7", HasMin: true},
		7: {ID: 7, Type: 3, Name: "XueWeiContainer", Para: [6]string{"xw_test", "Level"}, Min: "1", HasMin: true},
		8: {ID: 8, Type: 2, Name: "ToolBox", Para: [6]string{"item_missing"}, Max: "0", HasMax: true},
	}
	catalog := &shopConditionCatalog{Definitions: definitions, Formulas: map[int32]string{10: "1&2|3"}}
	evaluator := exactCurrentConditionEvaluator{catalog: catalog, player: player}
	for _, id := range []int32{1, 2, 3, 4, 5, 6, 8} {
		got, supported := evaluator.evaluate(id)
		if !supported || !got {
			t.Fatalf("condition %d got=%t supported=%t", id, got, supported)
		}
	}
	if got, supported := evaluator.evaluate(7); got || supported {
		t.Fatalf("XueWei got=%t supported=%t", got, supported)
	}
	got, supported, err := evaluateConditionRoot(10, catalog.resolve, evaluator.evaluate)
	if err != nil || !supported || !got {
		t.Fatalf("formula got=%t supported=%t err=%v", got, supported, err)
	}
}

func TestCurrentConditionCapabilityClassification(t *testing.T) {
	catalog := &shopConditionCatalog{Definitions: map[int32]shopConditionDefinition{
		1: {ID: 1, Type: 1, Name: "School"},
		2: {ID: 2, Type: 1, Name: "VipStatus"},
		3: {ID: 3, Type: 2, Name: "BufferContainer"},
		4: {ID: 4, Type: 3, Name: "SkillContainer"},
		5: {ID: 5, Type: 19, Name: "Func"},
	}}
	cases := map[int32]conditionCapabilityClass{
		1:     conditionCapabilityExactEvaluator,
		2:     conditionCapabilityAuthoritativeStateMissing,
		3:     conditionCapabilitySemanticsUnresolved,
		4:     conditionCapabilityExactEvaluator,
		5:     conditionCapabilitySemanticsUnresolved,
		26526: conditionCapabilitySemanticsUnresolved,
	}
	for id, want := range cases {
		if got := exactCurrentConditionCapability(catalog, id); got != want {
			t.Fatalf("condition %d class=%d want=%d", id, got, want)
		}
	}
}
