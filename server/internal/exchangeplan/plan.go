// Package exchangeplan implements the evidence-closed, side-effect-free portion
// of current shop-exchange material planning. It deliberately does not decide
// final output BindStatus, ShowBind, ExchangeBind, capacity, or persistence.
package exchangeplan

import (
	"fmt"
	"math"
)

// Stack is one concrete inventory stack. BindStatus is runtime item-instance
// state and is valid only when it is 0 (unbound) or 1 (bound).
type Stack struct {
	Index      int
	ConfigID   string
	Amount     int32
	BindStatus int32
}

// Requirement is the material cost for one exchange result.
type Requirement struct {
	ConfigID string
	Amount   int32
}

// Deduction records the exact stack slice selected by the planner.
type Deduction struct {
	StackIndex int
	ConfigID   string
	Amount     int32
	BindStatus int32
}

// ResultPlan is the concrete material plan for one produced result.
type ResultPlan struct {
	Deductions           []Deduction
	MaterialDerivedBound bool
}

// BatchPlan is atomic at planning time. If Satisfied is false, Results is empty
// and Remaining is an unchanged copy of the input stacks; callers must not
// commit a partially satisfiable prefix.
type BatchPlan struct {
	Satisfied bool
	Results   []ResultPlan
	Remaining []Stack
}

// PlanBatch plans count sequential results. Material selection is exact and
// case-sensitive by ConfigID. Bound stacks are consumed before unbound stacks;
// each ResultPlan independently records whether any bound material was actually
// selected for that result.
func PlanBatch(stacks []Stack, requirements []Requirement, count int32) (BatchPlan, error) {
	original := cloneStacks(stacks)
	if count <= 0 {
		return BatchPlan{}, fmt.Errorf("exchangeplan: result count must be positive, got %d", count)
	}
	seenStackIndexes := make(map[int]struct{}, len(stacks))
	for i, stack := range stacks {
		if _, exists := seenStackIndexes[stack.Index]; exists {
			return BatchPlan{}, fmt.Errorf("exchangeplan: duplicate stack index %d", stack.Index)
		}
		seenStackIndexes[stack.Index] = struct{}{}
		if stack.ConfigID == "" {
			return BatchPlan{}, fmt.Errorf("exchangeplan: stack %d has empty ConfigID", i)
		}
		if stack.Amount < 0 {
			return BatchPlan{}, fmt.Errorf("exchangeplan: stack %d has negative amount %d", i, stack.Amount)
		}
		if stack.BindStatus != 0 && stack.BindStatus != 1 {
			return BatchPlan{}, fmt.Errorf("exchangeplan: stack %d has invalid runtime BindStatus %d", i, stack.BindStatus)
		}
	}

	aggregated, err := aggregateRequirements(requirements)
	if err != nil {
		return BatchPlan{}, err
	}
	remaining := cloneStacks(stacks)
	results := make([]ResultPlan, 0, int(count))
	for resultIndex := int32(0); resultIndex < count; resultIndex++ {
		result, ok := planOne(remaining, aggregated)
		if !ok {
			return BatchPlan{Satisfied: false, Remaining: original}, nil
		}
		applyDeductions(remaining, result.Deductions)
		results = append(results, result)
	}
	return BatchPlan{Satisfied: true, Results: results, Remaining: remaining}, nil
}

func aggregateRequirements(requirements []Requirement) ([]Requirement, error) {
	order := make([]string, 0, len(requirements))
	amounts := make(map[string]int64, len(requirements))
	for i, requirement := range requirements {
		if requirement.ConfigID == "" {
			return nil, fmt.Errorf("exchangeplan: requirement %d has empty ConfigID", i)
		}
		if requirement.Amount <= 0 {
			return nil, fmt.Errorf("exchangeplan: requirement %d has non-positive amount %d", i, requirement.Amount)
		}
		if _, exists := amounts[requirement.ConfigID]; !exists {
			order = append(order, requirement.ConfigID)
		}
		amounts[requirement.ConfigID] += int64(requirement.Amount)
		if amounts[requirement.ConfigID] > math.MaxInt32 {
			return nil, fmt.Errorf("exchangeplan: requirement %q amount overflows int32", requirement.ConfigID)
		}
	}
	result := make([]Requirement, 0, len(order))
	for _, configID := range order {
		result = append(result, Requirement{ConfigID: configID, Amount: int32(amounts[configID])})
	}
	return result, nil
}

func planOne(stacks []Stack, requirements []Requirement) (ResultPlan, bool) {
	var result ResultPlan
	for _, requirement := range requirements {
		need := requirement.Amount
		for _, bindStatus := range []int32{1, 0} {
			for _, stack := range stacks {
				if need == 0 {
					break
				}
				if stack.ConfigID != requirement.ConfigID || stack.BindStatus != bindStatus || stack.Amount <= 0 {
					continue
				}
				available := stack.Amount
				for _, deduction := range result.Deductions {
					if deduction.StackIndex == stack.Index {
						available -= deduction.Amount
					}
				}
				if available <= 0 {
					continue
				}
				take := available
				if take > need {
					take = need
				}
				result.Deductions = append(result.Deductions, Deduction{
					StackIndex: stack.Index,
					ConfigID:   stack.ConfigID,
					Amount:     take,
					BindStatus: bindStatus,
				})
				if bindStatus == 1 {
					result.MaterialDerivedBound = true
				}
				need -= take
			}
		}
		if need != 0 {
			return ResultPlan{}, false
		}
	}
	return result, true
}

func applyDeductions(stacks []Stack, deductions []Deduction) {
	for _, deduction := range deductions {
		for i := range stacks {
			if stacks[i].Index == deduction.StackIndex {
				stacks[i].Amount -= deduction.Amount
				break
			}
		}
	}
}

func cloneStacks(stacks []Stack) []Stack {
	return append([]Stack(nil), stacks...)
}
