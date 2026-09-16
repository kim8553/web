package main

import (
	"fmt"
	"sort"
)

// conditionCapabilityClass is intentionally about proof/capability, not a
// guessed boolean condition value. Only conditionCapabilityExactEvaluator can
// make an ExchangeData SAFE for production condition evaluation.
type conditionCapabilityClass uint8

const (
	conditionCapabilityExactEvaluator conditionCapabilityClass = iota + 1
	conditionCapabilityStateMapperMissing
	conditionCapabilityAuthoritativeStateMissing
	conditionCapabilitySemanticsUnresolved
)

type conditionCapabilityClassifier func(conditionID int32) conditionCapabilityClass

// exchangeConditionSpec is the condition-only projection of one ExchangeItem
// row. Keeping the audit on this minimal projection makes it wire-inert and
// independent of item/cost/bind transaction fields.
type exchangeConditionSpec struct {
	Condition  string
	Condition2 string
	Filters    string
}

type exchangeConditionCapability struct {
	ExchangeData int32
	Safe         bool
	Leaves       []int32
	Classes      map[int32]conditionCapabilityClass
}

type shopExchangeConditionCapabilityAudit struct {
	ExchangeDataTotal int
	SafeExchangeData  int
	FailClosed        int
	UniqueLeaves      []int32
	ClassCounts       map[conditionCapabilityClass]int
	ByExchangeData    map[int32]exchangeConditionCapability
}

// auditShopExchangeConditionCapabilities expands Condition and Condition2
// recursively through the exact formula resolver, classifies terminal leaves,
// and marks an ExchangeData SAFE only when every terminal leaf has an exact
// authoritative evaluator. Unsupported/missing/unresolved leaves are never
// converted into false-and-safe; their ExchangeData remains FAIL_CLOSED.
func auditShopExchangeConditionCapabilities(definitions map[int32]exchangeConditionSpec, resolve conditionFormulaResolver, classify conditionCapabilityClassifier) (shopExchangeConditionCapabilityAudit, error) {
	if resolve == nil {
		return shopExchangeConditionCapabilityAudit{}, fmt.Errorf("nil condition formula resolver")
	}
	if classify == nil {
		return shopExchangeConditionCapabilityAudit{}, fmt.Errorf("nil condition capability classifier")
	}

	audit := shopExchangeConditionCapabilityAudit{
		ExchangeDataTotal: len(definitions),
		ClassCounts:       make(map[conditionCapabilityClass]int),
		ByExchangeData:    make(map[int32]exchangeConditionCapability, len(definitions)),
	}
	globalLeaves := make(map[int32]conditionCapabilityClass)

	ids := make([]int32, 0, len(definitions))
	for id := range definitions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, exchangeData := range ids {
		definition := definitions[exchangeData]
		references, err := exchangeConditionReferences(definition.Condition, definition.Condition2, definition.Filters)
		if err != nil {
			return shopExchangeConditionCapabilityAudit{}, fmt.Errorf("ExchangeData %d: %w", exchangeData, err)
		}

		entry := exchangeConditionCapability{
			ExchangeData: exchangeData,
			Safe:         true,
			Classes:      make(map[int32]conditionCapabilityClass),
		}
		seen := make(map[int32]struct{})
		for _, reference := range references {
			leaves, err := terminalConditionLeaves(reference.ConditionID, resolve)
			if err != nil {
				return shopExchangeConditionCapabilityAudit{}, fmt.Errorf("ExchangeData %d condition %d: %w", exchangeData, reference.ConditionID, err)
			}
			for _, leaf := range leaves {
				if _, ok := seen[leaf]; ok {
					continue
				}
				seen[leaf] = struct{}{}
				class := classify(leaf)
				if class < conditionCapabilityExactEvaluator || class > conditionCapabilitySemanticsUnresolved {
					return shopExchangeConditionCapabilityAudit{}, fmt.Errorf("condition %d has invalid capability class %d", leaf, class)
				}
				entry.Leaves = append(entry.Leaves, leaf)
				entry.Classes[leaf] = class
				if class != conditionCapabilityExactEvaluator {
					entry.Safe = false
				}
				if prior, ok := globalLeaves[leaf]; ok && prior != class {
					return shopExchangeConditionCapabilityAudit{}, fmt.Errorf("condition %d classified inconsistently: %d then %d", leaf, prior, class)
				}
				globalLeaves[leaf] = class
			}
		}
		if entry.Safe {
			audit.SafeExchangeData++
		} else {
			audit.FailClosed++
		}
		audit.ByExchangeData[exchangeData] = entry
	}

	audit.UniqueLeaves = make([]int32, 0, len(globalLeaves))
	for leaf, class := range globalLeaves {
		audit.UniqueLeaves = append(audit.UniqueLeaves, leaf)
		audit.ClassCounts[class]++
	}
	sort.Slice(audit.UniqueLeaves, func(i, j int) bool { return audit.UniqueLeaves[i] < audit.UniqueLeaves[j] })
	return audit, nil
}
