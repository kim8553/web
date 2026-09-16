package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/exchangeplan"
)

// applyShopExchangeReplacementToSnapshot applies a fully staged replacement to
// a clone of snapshot. It revalidates every deduction identity before making
// the clone visible to the caller, so a stale plan cannot silently consume a
// changed stack. The input snapshot is never mutated.
func applyShopExchangeReplacementToSnapshot(itemCatalog *itemCatalog, snapshot []bagItem, plan exchangeplan.ReplacementPlan) ([]bagItem, error) {
	if !plan.Fits {
		return nil, fmt.Errorf("shop exchange apply: replacement plan does not fit")
	}
	if itemCatalog == nil {
		return nil, fmt.Errorf("shop exchange apply: item catalog unavailable")
	}

	staged := append([]bagItem(nil), snapshot...)
	removed := make(map[int]struct{}, len(plan.Deductions))
	touched := make(map[int]struct{}, len(plan.Deductions))
	for _, mutation := range plan.Deductions {
		if mutation.Index < 0 || mutation.Index >= len(staged) {
			return nil, fmt.Errorf("shop exchange apply: deduction index %d outside snapshot len %d", mutation.Index, len(staged))
		}
		if _, duplicate := touched[mutation.Index]; duplicate {
			return nil, fmt.Errorf("shop exchange apply: duplicate deduction index %d", mutation.Index)
		}
		touched[mutation.Index] = struct{}{}
		item := staged[mutation.Index]
		container, ok := latestClientCurrentBagContainer(item.ViewID)
		if !ok {
			return nil, fmt.Errorf("shop exchange apply: deduction index %d has unsupported ViewID %d", mutation.Index, item.ViewID)
		}
		if container != mutation.Container || item.Slot != mutation.Slot || item.ConfigID != mutation.ConfigID || item.BindStatus != mutation.BindStatus || item.Amount != mutation.AmountBefore {
			return nil, fmt.Errorf("shop exchange apply: stale deduction index %d", mutation.Index)
		}
		if mutation.AmountAfter < 0 || mutation.AmountAfter >= mutation.AmountBefore || mutation.Removed != (mutation.AmountAfter == 0) {
			return nil, fmt.Errorf("shop exchange apply: invalid staged deduction at index %d", mutation.Index)
		}
		if mutation.Removed {
			removed[mutation.Index] = struct{}{}
			continue
		}
		item.Amount = mutation.AmountAfter
		staged[mutation.Index] = item
	}

	result := make([]bagItem, 0, len(staged)-len(removed)+len(plan.Adds))
	occupied := make(map[uint16]map[int32]struct{})
	for index, item := range staged {
		if _, drop := removed[index]; drop {
			continue
		}
		container, ok := latestClientCurrentBagContainer(item.ViewID)
		if ok && item.Slot > 0 {
			slots := occupied[container]
			if slots == nil {
				slots = make(map[int32]struct{})
				occupied[container] = slots
			}
			if _, duplicate := slots[item.Slot]; duplicate {
				return nil, fmt.Errorf("shop exchange apply: duplicate retained slot container=%d slot=%d", container, item.Slot)
			}
			slots[item.Slot] = struct{}{}
		}
		result = append(result, item)
	}

	for _, add := range plan.Adds {
		if add.ConfigID == "" || add.Amount <= 0 || add.MaxAmount <= 0 || add.Amount > add.MaxAmount {
			return nil, fmt.Errorf("shop exchange apply: invalid staged output %+v", add)
		}
		if add.BindStatus != 0 && add.BindStatus != 1 {
			return nil, fmt.Errorf("shop exchange apply: invalid output BindStatus %d", add.BindStatus)
		}
		staticItem, ok := itemCatalog.Lookup(add.ConfigID)
		if !ok {
			return nil, fmt.Errorf("shop exchange apply: output %q missing from item catalog", add.ConfigID)
		}
		container, ok := latestClientCurrentBagContainer(staticItem.ViewID)
		if !ok || container != add.Container {
			return nil, fmt.Errorf("shop exchange apply: output %q container mismatch", add.ConfigID)
		}
		staticMaxAmount := staticItem.MaxAmount
		if staticMaxAmount <= 0 {
			staticMaxAmount = 1
		}
		if staticMaxAmount != add.MaxAmount {
			return nil, fmt.Errorf("shop exchange apply: output %q MaxAmount changed: staged=%d current=%d", add.ConfigID, add.MaxAmount, staticMaxAmount)
		}
		capacity, ok := currentShopExchangeContainerCapacity(container)
		if !ok || add.Slot <= 0 || add.Slot > capacity {
			return nil, fmt.Errorf("shop exchange apply: output %q slot %d invalid for container %d", add.ConfigID, add.Slot, container)
		}
		slots := occupied[container]
		if slots == nil {
			slots = make(map[int32]struct{})
			occupied[container] = slots
		}
		if _, duplicate := slots[add.Slot]; duplicate {
			return nil, fmt.Errorf("shop exchange apply: output slot already occupied container=%d slot=%d", container, add.Slot)
		}
		slots[add.Slot] = struct{}{}
		result = append(result, bagItem{
			ConfigID:       staticItem.ConfigID,
			ItemType:       staticItem.ItemType,
			Amount:         add.Amount,
			BindStatus:     add.BindStatus,
			ViewID:         staticItem.ViewID,
			Slot:           add.Slot,
			Name:           staticItem.Name,
			MaxAmount:      add.MaxAmount,
			FuncPack:       staticItem.FuncPack,
			LogicPack:      staticItem.LogicPack,
			PropModifyPack: staticItem.PropModifyPack,
			TextureType:    staticItem.TextureType,
			CardID:         staticItem.CardID,
			ToolUseEffect:  staticItem.ToolUseEffect,
			FuncBuffer:     staticItem.FuncBuffer,
		})
	}
	return result, nil
}
