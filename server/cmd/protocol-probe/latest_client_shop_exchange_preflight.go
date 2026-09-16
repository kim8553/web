package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/exchangeplan"
)

// shopExchangeReadOnlyPreflight is deliberately side-effect free. It resolves
// the exact current listing/ExchangeData row, evaluates only condition/property
// state Stage37 can prove, and asks exchangeplan for a concrete bound-first
// material plan. It does not deduct materials, create the result, persist DB
// state, emit View mutations, or decide final output BindStatus.
type shopExchangeReadOnlyPreflight struct {
	Listing            shopCatalogItem
	Definition         shopExchangeDefinition
	ConditionSupported bool
	ConditionSatisfied bool
	ConditionDetails   []shopConditionDetail
	PropertySupported  bool
	PropertySatisfied  bool
	MaterialPlan       exchangeplan.BatchPlan
	CapacitySupported  bool
	CapacitySatisfied  bool
	CapacityPlan       exchangeplan.CapacityPlan
	// ExchangeBindPreview is the exact-current UI value for the first result:
	// 1 when bound-first material consumption makes that first result bound,
	// otherwise 0. It is not the final persisted BindStatus because separately
	// proven static bind-on-acquire can still bind an item after creation.
	ExchangeBindPreview int32
	OutputAmount        int64
}

type shopExchangePropertyCost struct {
	Name   string
	Amount int64
}

func parseShopExchangeRequirements(raw string) ([]exchangeplan.Requirement, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	entries := strings.Split(raw, ";")
	result := make([]exchangeplan.Requirement, 0, len(entries))
	for i, entry := range entries {
		entry = strings.TrimSpace(entry)
		// Exact-current ExchangeItem.ini contains reachable mode3 Item lists
		// ending in a single trailing semicolon (for example ExchangeData
		// 11564, 14087..14089, 14286..14288, 14322..14324). Treat only
		// that final empty token as list termination; embedded empty entries
		// remain malformed and fail closed.
		if entry == "" && i == len(entries)-1 {
			continue
		}
		if entry == "" {
			return nil, fmt.Errorf("exchange Item has empty entry in %q", raw)
		}
		parts := strings.Split(entry, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("exchange Item entry %q has no amount", entry)
		}
		configID := strings.TrimSpace(parts[0])
		if configID == "" {
			return nil, fmt.Errorf("exchange Item entry %q has empty ConfigID", entry)
		}
		amount, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil || amount <= 0 {
			return nil, fmt.Errorf("exchange Item entry %q has invalid positive amount", entry)
		}
		result = append(result, exchangeplan.Requirement{ConfigID: configID, Amount: int32(amount)})
	}
	return result, nil
}

func parseShopExchangePropertyCosts(raw string) ([]shopExchangePropertyCost, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	entries := strings.Split(raw, ";")
	result := make([]shopExchangePropertyCost, 0, len(entries))
	for _, entry := range entries {
		parts := strings.Split(entry, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("exchange Prop entry %q has no amount", entry)
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			return nil, fmt.Errorf("exchange Prop entry %q has empty property name", entry)
		}
		amount, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || amount < 0 {
			return nil, fmt.Errorf("exchange Prop entry %q has invalid non-negative amount", entry)
		}
		result = append(result, shopExchangePropertyCost{Name: name, Amount: amount})
	}
	return result, nil
}

func firstShopExchangeMaterialBindPreview(plan exchangeplan.BatchPlan) int32 {
	if !plan.Satisfied || len(plan.Results) == 0 {
		return 0
	}
	if plan.Results[0].MaterialDerivedBound {
		return 1
	}
	return 0
}

func currentShopExchangeInventoryStacksFromSnapshot(items []bagItem) []exchangeplan.Stack {
	result := make([]exchangeplan.Stack, 0, len(items))
	for index, item := range items {
		if _, ok := latestClientCurrentBagContainer(item.ViewID); !ok {
			continue
		}
		result = append(result, exchangeplan.Stack{
			Index:      index,
			ConfigID:   item.ConfigID,
			Amount:     item.Amount,
			BindStatus: item.BindStatus,
		})
	}
	return result
}

func currentShopExchangeInventoryStacks(player *playerActor) []exchangeplan.Stack {
	if player == nil {
		return nil
	}
	return currentShopExchangeInventoryStacksFromSnapshot(player.bagSnapshot())
}

func currentShopExchangeInventorySlotsFromSnapshot(items []bagItem) []exchangeplan.InventorySlot {
	result := make([]exchangeplan.InventorySlot, 0, len(items))
	for index, item := range items {
		container, ok := latestClientCurrentBagContainer(item.ViewID)
		if !ok {
			continue
		}
		result = append(result, exchangeplan.InventorySlot{Index: index, Container: container, Slot: item.Slot, Amount: item.Amount})
	}
	return result
}

func currentShopExchangeContainerCapacity(container uint16) (int32, bool) {
	for _, spec := range starterBagViews() {
		if spec.ID != container {
			continue
		}
		switch container {
		case 2, 121, 123, 125:
			return int32(spec.Capacity), true
		default:
			return 0, false
		}
	}
	return 0, false
}

func evaluateShopExchangeCapacityPreview(itemCatalog *itemCatalog, bagSnapshot []bagItem, listing shopCatalogItem, materialPlan exchangeplan.BatchPlan, outputAmount int64) (supported, satisfied bool, plan exchangeplan.CapacityPlan, err error) {
	if itemCatalog == nil {
		return false, false, plan, nil
	}
	staticItem, ok := itemCatalog.Lookup(listing.configID)
	if !ok {
		return false, false, plan, nil
	}
	container, ok := latestClientCurrentBagContainer(staticItem.ViewID)
	if !ok {
		return false, false, plan, nil
	}
	capacity, ok := currentShopExchangeContainerCapacity(container)
	if !ok {
		return false, false, plan, nil
	}
	plan, err = exchangeplan.PlanConservativeCapacity(currentShopExchangeInventorySlotsFromSnapshot(bagSnapshot), materialPlan, container, capacity, outputAmount, staticItem.MaxAmount)
	if err != nil {
		return false, false, plan, err
	}
	return true, plan.Fits, plan, nil
}

// evaluateShopExchangeConditionPreview follows the exact current form UI's
// ConditionType composition: zero selects the AND description; any non-zero
// value selects the OR description. This is a read-only preview/capability
// result, not a claim that the final server mutation authorization has been
// proven. Every root is still evaluated so unsupported leaves fail closed.
func evaluateShopExchangeConditionPreview(definition shopExchangeDefinition, authority *currentShopConditionAuthority, player *playerActor) (supported, satisfied bool, details []shopConditionDetail, err error) {
	references, err := exchangeConditionReferences(definition.Condition, definition.Condition2, definition.Filters)
	if err != nil {
		return false, false, nil, err
	}
	if len(references) == 0 {
		return true, true, nil, nil
	}
	if authority == nil || authority.catalog == nil || player == nil {
		return false, false, nil, nil
	}
	evaluator := exactCurrentConditionEvaluator{catalog: authority.catalog, player: player, skillMaxTable: authority.skillMaxTable}
	details = make([]shopConditionDetail, 0, len(references))
	allSupported := true
	andValue := true
	orValue := false
	for _, reference := range references {
		value, rootSupported, evalErr := evaluateConditionRoot(reference.ConditionID, authority.catalog.resolve, evaluator.evaluate)
		if evalErr != nil {
			return false, false, nil, fmt.Errorf("condition %d: %w", reference.ConditionID, evalErr)
		}
		if !rootSupported {
			allSupported = false
			value = false
		}
		andValue = andValue && value
		orValue = orValue || value
		details = append(details, shopConditionDetail{ConditionID: reference.ConditionID, IsNeedShow: reference.IsNeedShow, Satisfied: value})
	}
	if !allSupported {
		return false, false, details, nil
	}
	if definition.ConditionType == 0 {
		return true, andValue, details, nil
	}
	return true, orValue, details, nil
}

func shopExchangePropertyValue(player *playerActor, name string) (int64, bool) {
	if player == nil {
		return 0, false
	}
	silver, gold, silverCard, silverTicket := player.currencySnapshot()
	switch name {
	case "CapitalType0":
		return int64(gold), true
	case "CapitalType1":
		return int64(silver), true
	case "CapitalType2":
		return int64(silverCard), true
	case "CapitalType4":
		return int64(silverTicket), true
	default:
		// CapitalType3 and arbitrary player properties are intentionally not
		// guessed: Stage37 does not yet have authoritative mutable state for them.
		return 0, false
	}
}

func evaluateShopExchangePropertyPreview(definition shopExchangeDefinition, player *playerActor, count int32) (supported, satisfied bool, err error) {
	if count <= 0 {
		return false, false, fmt.Errorf("exchange count must be positive, got %d", count)
	}
	costs, err := parseShopExchangePropertyCosts(definition.Prop)
	if err != nil {
		return false, false, err
	}
	// Exact current form_exchange.lua treats Type==1 as guild-building exchange
	// and displays AddValue as an additional guild-currency cost. Stage37 has no
	// proven authoritative guild-currency state, so such rows remain fail-closed.
	if definition.Type == 1 {
		return false, false, nil
	}
	for _, cost := range costs {
		if cost.Amount > math.MaxInt64/int64(count) {
			return false, false, fmt.Errorf("exchange Prop %s amount overflows batch", cost.Name)
		}
		need := cost.Amount * int64(count)
		have, ok := shopExchangePropertyValue(player, cost.Name)
		if !ok {
			return false, false, nil
		}
		if have < need {
			return true, false, nil
		}
	}
	return true, true, nil
}

func runShopExchangeReadOnlyPreflight(shopPath, exchangePath string, authority *currentShopConditionAuthority, player *playerActor, itemCatalog *itemCatalog, request shopExchangeBuyRequest) (shopExchangeReadOnlyPreflight, error) {
	var result shopExchangeReadOnlyPreflight
	if request.Count < 1 || request.Count > 99 {
		return result, fmt.Errorf("exchange count %d outside current client range 1..99", request.Count)
	}
	items, _, _, err := shopCatalogItems(shopPath, request.ShopID)
	if err != nil {
		return result, err
	}
	listing := currentShopListing(items, request.Page, request.Position)
	if listing == nil {
		return result, fmt.Errorf("shop %s has no current listing at page=%d pos=%d", request.ShopID, request.Page, request.Position)
	}
	result.Listing = *listing
	if listing.priceMode != 3 || listing.exchangeData <= 0 {
		return result, fmt.Errorf("shop %s page=%d pos=%d is not a mode3 ExchangeData listing", request.ShopID, request.Page, request.Position)
	}
	definition, err := loadShopExchangeDefinition(exchangePath, listing.exchangeData)
	if err != nil {
		return result, err
	}
	result.Definition = definition
	result.ConditionSupported, result.ConditionSatisfied, result.ConditionDetails, err = evaluateShopExchangeConditionPreview(definition, authority, player)
	if err != nil {
		return result, err
	}
	result.PropertySupported, result.PropertySatisfied, err = evaluateShopExchangePropertyPreview(definition, player, request.Count)
	if err != nil {
		return result, err
	}
	requirements, err := parseShopExchangeRequirements(definition.Item)
	if err != nil {
		return result, err
	}
	bagSnapshot := player.bagSnapshot()
	result.MaterialPlan, err = exchangeplan.PlanBatch(currentShopExchangeInventoryStacksFromSnapshot(bagSnapshot), requirements, request.Count)
	if err != nil {
		return result, err
	}
	result.ExchangeBindPreview = firstShopExchangeMaterialBindPreview(result.MaterialPlan)
	if listing.amount <= 0 || int64(listing.amount) > math.MaxInt64/int64(request.Count) {
		return result, fmt.Errorf("invalid exchange output amount=%d count=%d", listing.amount, request.Count)
	}
	result.OutputAmount = int64(listing.amount) * int64(request.Count)
	result.CapacitySupported, result.CapacitySatisfied, result.CapacityPlan, err = evaluateShopExchangeCapacityPreview(itemCatalog, bagSnapshot, *listing, result.MaterialPlan, result.OutputAmount)
	if err != nil {
		return result, err
	}
	return result, nil
}
