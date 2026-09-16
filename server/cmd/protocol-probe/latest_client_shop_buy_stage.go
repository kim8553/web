package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/shopbuyplan"
)

type stagedShopPurchase struct {
	BagAfter      []bagItem
	CurrencyAfter currencySnapshot
	Reward        bagItem
	RewardView    uint16
	RewardSlot    int32
	TotalPrice    int64
}

// stageShopPurchaseSnapshot is pure with respect to player live state. It
// reproduces only behavior already present in the recovered ordinary-shop
// handler/addBagItem path: enrich one reward, debit capital type 0/1/2, choose
// the smallest positive free slot in the reward's current bag view when no
// positive slot is pre-authored, and append the reward to a copied bag snapshot.
// It deliberately does not infer stacking, capacity, binding, or a wire selector.
func stageShopPurchaseSnapshot(itemCatalog *itemCatalog, bagBefore []bagItem, currencyBefore currencySnapshot, listing shopCatalogItem, count int32) (stagedShopPurchase, error) {
	var staged stagedShopPurchase
	reward := enrichBagItem(bagItem{ConfigID: listing.configID, Amount: count}, itemCatalog, nil)
	view := bagViewForViewID(reward.ViewID)
	if view == 0 {
		return staged, fmt.Errorf("shop buy stage: reward %q has unsupported ViewID %d", reward.ConfigID, reward.ViewID)
	}

	occupied := make([]shopbuyplan.BagSlot, 0, len(bagBefore))
	for _, existing := range bagBefore {
		occupied = append(occupied, shopbuyplan.BagSlot{View: bagViewForViewID(existing.ViewID), Slot: existing.Slot})
	}
	plan, err := shopbuyplan.Stage(shopbuyplan.Currency{
		Silver:       currencyBefore.Silver,
		Gold:         currencyBefore.Gold,
		SilverCard:   currencyBefore.SilverCard,
		SilverTicket: currencyBefore.SilverTicket,
	}, occupied, shopbuyplan.Input{
		CapitalType:   listing.priceMode,
		UnitPrice:     listing.price,
		Count:         count,
		RewardView:    view,
		RequestedSlot: reward.Slot,
	})
	if err != nil {
		return staged, fmt.Errorf("shop buy stage: %w", err)
	}

	reward.Slot = plan.RewardSlot
	bagAfter := append([]bagItem(nil), bagBefore...)
	bagAfter = append(bagAfter, reward)
	return stagedShopPurchase{
		BagAfter: bagAfter,
		CurrencyAfter: currencySnapshot{
			Silver:       plan.CurrencyAfter.Silver,
			Gold:         plan.CurrencyAfter.Gold,
			SilverCard:   plan.CurrencyAfter.SilverCard,
			SilverTicket: plan.CurrencyAfter.SilverTicket,
		},
		Reward:     reward,
		RewardView: view,
		RewardSlot: plan.RewardSlot,
		TotalPrice: plan.TotalPrice,
	}, nil
}
