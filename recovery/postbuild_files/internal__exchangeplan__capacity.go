package exchangeplan

import "fmt"

// InventorySlot is one concrete occupied inventory slot. Index must use the
// same stable stack identity as Stack.Index in the material plan.
type InventorySlot struct {
	Index     int
	Container uint16
	Slot      int32
	Amount    int32
}

// CapacityPlan is a conservative replacement-capacity check. It accounts for
// slots freed by the already-proven material plan but deliberately assumes the
// result cannot merge into any existing stack. That can reject an exchange
// that the native server might fit via merging, but it cannot authorize an
// exchange merely by guessing merge semantics.
type CapacityPlan struct {
	Container            uint16
	Capacity             int32
	OccupiedBefore       int32
	OccupiedAfterRemoval int32
	FreedByMaterials     int32
	RequiredOutputStacks int32
	Fits                 bool
}

// PlanConservativeCapacity checks whether the full batch can fit after all
// planned material deductions. outputMaxAmount <= 0 is treated as one item per
// stack, matching Stage37's existing fail-safe stack behavior.
func PlanConservativeCapacity(inventory []InventorySlot, materialPlan BatchPlan, outputContainer uint16, capacity int32, outputAmount int64, outputMaxAmount int32) (CapacityPlan, error) {
	result := CapacityPlan{Container: outputContainer, Capacity: capacity}
	if outputContainer == 0 {
		return result, fmt.Errorf("exchangeplan: output container is zero")
	}
	if capacity <= 0 {
		return result, fmt.Errorf("exchangeplan: invalid container capacity %d", capacity)
	}
	if outputAmount <= 0 {
		return result, fmt.Errorf("exchangeplan: invalid output amount %d", outputAmount)
	}
	if outputMaxAmount <= 0 {
		outputMaxAmount = 1
	}
	maxAmount := int64(outputMaxAmount)
	required := (outputAmount + maxAmount - 1) / maxAmount
	if required > int64(^uint32(0)>>1) {
		return result, fmt.Errorf("exchangeplan: output stack count overflows int32: %d", required)
	}
	result.RequiredOutputStacks = int32(required)

	deducted := make(map[int]int64)
	if materialPlan.Satisfied {
		for _, perResult := range materialPlan.Results {
			for _, deduction := range perResult.Deductions {
				if deduction.Amount <= 0 {
					return result, fmt.Errorf("exchangeplan: stack %d has non-positive deduction %d", deduction.StackIndex, deduction.Amount)
				}
				deducted[deduction.StackIndex] += int64(deduction.Amount)
			}
		}
	}

	seenSlots := make(map[int32]int)
	seenIndexes := make(map[int]struct{}, len(inventory))
	for _, stack := range inventory {
		if _, duplicate := seenIndexes[stack.Index]; duplicate {
			return result, fmt.Errorf("exchangeplan: duplicate inventory stack index %d", stack.Index)
		}
		seenIndexes[stack.Index] = struct{}{}
		if stack.Amount <= 0 {
			return result, fmt.Errorf("exchangeplan: stack %d has non-positive amount %d", stack.Index, stack.Amount)
		}
		if stack.Container != outputContainer {
			continue
		}
		if stack.Slot <= 0 || stack.Slot > capacity {
			return result, fmt.Errorf("exchangeplan: stack %d has slot %d outside container %d capacity %d", stack.Index, stack.Slot, outputContainer, capacity)
		}
		if prior, duplicate := seenSlots[stack.Slot]; duplicate {
			return result, fmt.Errorf("exchangeplan: container %d duplicate slot %d at stacks %d and %d", outputContainer, stack.Slot, prior, stack.Index)
		}
		seenSlots[stack.Slot] = stack.Index
		result.OccupiedBefore++
		remaining := int64(stack.Amount) - deducted[stack.Index]
		if remaining < 0 {
			return result, fmt.Errorf("exchangeplan: deductions exceed stack %d amount: have=%d deduct=%d", stack.Index, stack.Amount, deducted[stack.Index])
		}
		if remaining == 0 {
			result.FreedByMaterials++
			continue
		}
		result.OccupiedAfterRemoval++
	}

	for index, amount := range deducted {
		if _, ok := seenIndexes[index]; !ok {
			return result, fmt.Errorf("exchangeplan: deduction references missing inventory stack %d", index)
		}
		if amount <= 0 {
			return result, fmt.Errorf("exchangeplan: invalid aggregated deduction %d for stack %d", amount, index)
		}
	}

	result.Fits = int64(result.OccupiedAfterRemoval)+int64(result.RequiredOutputStacks) <= int64(capacity)
	return result, nil
}
