package exchangeplan

import "fmt"

// InventoryState is the immutable inventory snapshot consumed by the atomic
// replacement planner. Index must match Stack.Index/Deduction.StackIndex.
type InventoryState struct {
	Index      int
	Container  uint16
	Slot       int32
	ConfigID   string
	Amount     int32
	MaxAmount  int32
	BindStatus int32
}

// OutputSpec describes the already-authorized exchange result. BindStatus is
// intentionally an input: the planner does not infer current-server binding
// precedence.
type OutputSpec struct {
	ConfigID   string
	Container  uint16
	Amount     int64
	MaxAmount  int32
	BindStatus int32
}

type StackMutation struct {
	Index        int
	Container    uint16
	Slot         int32
	ConfigID     string
	BindStatus   int32
	AmountBefore int32
	AmountAfter  int32
	Removed      bool
}

type AddedStack struct {
	Container  uint16
	Slot       int32
	ConfigID   string
	Amount     int32
	MaxAmount  int32
	BindStatus int32
}

// ReplacementPlan is side-effect free. Fits=false means no prefix of the plan
// is committable; callers must not apply Deductions independently of Adds.
type ReplacementPlan struct {
	Fits       bool
	Deductions []StackMutation
	Adds       []AddedStack
}

// PlanAtomicReplacement validates a proven material plan against one immutable
// inventory snapshot and stages the full remove+add replacement. It never
// merges the result into an existing stack; this matches the conservative
// capacity preflight and avoids inventing unproven native merge semantics.
func PlanAtomicReplacement(inventory []InventoryState, materialPlan BatchPlan, output OutputSpec, capacity int32) (ReplacementPlan, error) {
	var result ReplacementPlan
	if !materialPlan.Satisfied {
		return result, nil
	}
	if output.ConfigID == "" {
		return result, fmt.Errorf("exchangeplan: output ConfigID is empty")
	}
	if output.Container == 0 {
		return result, fmt.Errorf("exchangeplan: output container is zero")
	}
	if output.Amount <= 0 {
		return result, fmt.Errorf("exchangeplan: output amount must be positive, got %d", output.Amount)
	}
	if output.BindStatus != 0 && output.BindStatus != 1 {
		return result, fmt.Errorf("exchangeplan: invalid output BindStatus %d", output.BindStatus)
	}
	if capacity <= 0 {
		return result, fmt.Errorf("exchangeplan: invalid output capacity %d", capacity)
	}
	maxAmount := output.MaxAmount
	if maxAmount <= 0 {
		maxAmount = 1
	}

	byIndex := make(map[int]InventoryState, len(inventory))
	usedSlots := make(map[int32]int)
	for _, state := range inventory {
		if _, duplicate := byIndex[state.Index]; duplicate {
			return result, fmt.Errorf("exchangeplan: duplicate inventory index %d", state.Index)
		}
		if state.ConfigID == "" {
			return result, fmt.Errorf("exchangeplan: inventory index %d has empty ConfigID", state.Index)
		}
		if state.Amount <= 0 {
			return result, fmt.Errorf("exchangeplan: inventory index %d has non-positive amount %d", state.Index, state.Amount)
		}
		if state.BindStatus != 0 && state.BindStatus != 1 {
			return result, fmt.Errorf("exchangeplan: inventory index %d has invalid BindStatus %d", state.Index, state.BindStatus)
		}
		byIndex[state.Index] = state
		if state.Container != output.Container {
			continue
		}
		if state.Slot <= 0 || state.Slot > capacity {
			return result, fmt.Errorf("exchangeplan: inventory index %d slot %d outside container %d capacity %d", state.Index, state.Slot, output.Container, capacity)
		}
		if prior, duplicate := usedSlots[state.Slot]; duplicate {
			return result, fmt.Errorf("exchangeplan: duplicate output-container slot %d at indexes %d and %d", state.Slot, prior, state.Index)
		}
		usedSlots[state.Slot] = state.Index
	}

	type aggregate struct {
		amount int64
		config string
		bind   int32
	}
	agg := make(map[int]aggregate)
	order := make([]int, 0)
	for _, perResult := range materialPlan.Results {
		for _, deduction := range perResult.Deductions {
			if deduction.Amount <= 0 {
				return result, fmt.Errorf("exchangeplan: stack %d has non-positive deduction %d", deduction.StackIndex, deduction.Amount)
			}
			state, ok := byIndex[deduction.StackIndex]
			if !ok {
				return result, fmt.Errorf("exchangeplan: deduction references missing inventory index %d", deduction.StackIndex)
			}
			if deduction.ConfigID != state.ConfigID {
				return result, fmt.Errorf("exchangeplan: deduction ConfigID mismatch at index %d: plan=%q snapshot=%q", deduction.StackIndex, deduction.ConfigID, state.ConfigID)
			}
			if deduction.BindStatus != state.BindStatus {
				return result, fmt.Errorf("exchangeplan: deduction BindStatus mismatch at index %d: plan=%d snapshot=%d", deduction.StackIndex, deduction.BindStatus, state.BindStatus)
			}
			entry, exists := agg[deduction.StackIndex]
			if !exists {
				entry.config = deduction.ConfigID
				entry.bind = deduction.BindStatus
				order = append(order, deduction.StackIndex)
			}
			entry.amount += int64(deduction.Amount)
			agg[deduction.StackIndex] = entry
		}
	}

	for _, index := range order {
		state := byIndex[index]
		deduct := agg[index].amount
		if deduct > int64(state.Amount) {
			return result, fmt.Errorf("exchangeplan: deduction exceeds inventory index %d: have=%d deduct=%d", index, state.Amount, deduct)
		}
		after := int64(state.Amount) - deduct
		mutation := StackMutation{Index: index, Container: state.Container, Slot: state.Slot, ConfigID: state.ConfigID, BindStatus: state.BindStatus, AmountBefore: state.Amount, AmountAfter: int32(after), Removed: after == 0}
		result.Deductions = append(result.Deductions, mutation)
		if mutation.Removed && state.Container == output.Container {
			delete(usedSlots, state.Slot)
		}
	}

	required64 := (output.Amount + int64(maxAmount) - 1) / int64(maxAmount)
	if required64 > int64(capacity) {
		return ReplacementPlan{}, nil
	}
	free := make([]int32, 0, int(capacity)-len(usedSlots))
	for slot := int32(1); slot <= capacity; slot++ {
		if _, used := usedSlots[slot]; !used {
			free = append(free, slot)
		}
	}
	if int64(len(free)) < required64 {
		return ReplacementPlan{}, nil
	}

	remaining := output.Amount
	for i := int64(0); i < required64; i++ {
		amount := int64(maxAmount)
		if remaining < amount {
			amount = remaining
		}
		result.Adds = append(result.Adds, AddedStack{
			Container:  output.Container,
			Slot:       free[i],
			ConfigID:   output.ConfigID,
			Amount:     int32(amount),
			MaxAmount:  maxAmount,
			BindStatus: output.BindStatus,
		})
		remaining -= amount
	}
	if remaining != 0 {
		return ReplacementPlan{}, fmt.Errorf("exchangeplan: internal output remainder %d", remaining)
	}
	result.Fits = true
	return result, nil
}
