package main

import (
	"fmt"
	"strconv"
	"strings"
)

type exchangeConditionSource uint8

const (
	exchangeConditionPrimary exchangeConditionSource = iota + 1
	exchangeConditionSecondary
)

type exchangeConditionReference struct {
	ConditionID int32
	IsNeedShow  bool
	Source      exchangeConditionSource
}

// shopConditionDetail is one exact-current S2C 508 triple. The client
// interprets IsNeedShow and Satisfied as int32 0/1 values on the wire.
type shopConditionDetail struct {
	ConditionID int32
	IsNeedShow  bool
	Satisfied   bool
}

func parseExchangeConditionIDs(raw string) ([]int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	result := make([]int32, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return nil, fmt.Errorf("empty condition id in %q", raw)
		}
		value, err := strconv.ParseInt(token, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("condition id %q is not int32: %w", token, err)
		}
		result = append(result, int32(value))
	}
	return result, nil
}

// exchangeConditionReferences reproduces the exact-current ExchangeItem
// isNeedShow rule recovered from native/Lua behavior:
//
//	Condition  -> show when the id is NOT in Filters
//	Condition2 -> show when the id IS in Filters
//
// Authored order is preserved and entries are not deduplicated.
func exchangeConditionReferences(condition, condition2, filters string) ([]exchangeConditionReference, error) {
	primary, err := parseExchangeConditionIDs(condition)
	if err != nil {
		return nil, fmt.Errorf("Condition: %w", err)
	}
	secondary, err := parseExchangeConditionIDs(condition2)
	if err != nil {
		return nil, fmt.Errorf("Condition2: %w", err)
	}
	filterIDs, err := parseExchangeConditionIDs(filters)
	if err != nil {
		return nil, fmt.Errorf("Filters: %w", err)
	}
	filterSet := make(map[int32]struct{}, len(filterIDs))
	for _, id := range filterIDs {
		filterSet[id] = struct{}{}
	}
	result := make([]exchangeConditionReference, 0, len(primary)+len(secondary))
	for _, id := range primary {
		_, filtered := filterSet[id]
		result = append(result, exchangeConditionReference{ConditionID: id, IsNeedShow: !filtered, Source: exchangeConditionPrimary})
	}
	for _, id := range secondary {
		_, filtered := filterSet[id]
		result = append(result, exchangeConditionReference{ConditionID: id, IsNeedShow: filtered, Source: exchangeConditionSecondary})
	}
	return result, nil
}

// evaluateExchangeConditionDetails is the wire-inert bridge from one authored
// ExchangeItem definition to the exact S2C 508 triples. Formula expansion and
// leaf evaluation are authoritative-only: if any referenced condition cannot
// be evaluated from proven server state, that entry is emitted unsatisfied and
// allSupported is false. This function does not enable C2S69 dispatch.
func evaluateExchangeConditionDetails(condition, condition2, filters string, resolve conditionFormulaResolver, evaluateLeaf conditionFormulaLeafEvaluator) (details []shopConditionDetail, allSupported bool, err error) {
	references, err := exchangeConditionReferences(condition, condition2, filters)
	if err != nil {
		return nil, false, err
	}
	details = make([]shopConditionDetail, 0, len(references))
	allSupported = true
	for _, reference := range references {
		satisfied, supported, evalErr := evaluateConditionRoot(reference.ConditionID, resolve, evaluateLeaf)
		if evalErr != nil {
			return nil, false, fmt.Errorf("condition %d: %w", reference.ConditionID, evalErr)
		}
		if !supported {
			allSupported = false
			satisfied = false
		}
		details = append(details, shopConditionDetail{
			ConditionID: reference.ConditionID,
			IsNeedShow:  reference.IsNeedShow,
			Satisfied:   satisfied,
		})
	}
	return details, allSupported, nil
}
