package main

import (
	"fmt"
	"log"
)

// Exact current lua64 contract recovered from custom_sender.lua and
// share/client_custom_define.lua. These are CustomSend message selectors, not
// shop price/capital modes.
const (
	clientCustomRequestShopExchangeForm int32 = 64 // 0x40
	clientCustomRequestConditionShop    int32 = 69 // current CLIENT_CUSTOMMSG_REQUEST_CONDITION_SHOP
	clientCustomExchangeFromShop        int32 = 79 // 0x4f
)

type shopExchangeFormRequest struct {
	ViewIdent int32
	BindIndex int32
	ShopID    string
	Page      int32
	Position  int32
}

type shopConditionDetailsRequest struct {
	ExchangeIndex int32
}

type shopExchangeBuyRequest struct {
	ShopID   string
	Page     int32
	Position int32
	Count    int32
}

// currentShopExchangeBuySelection is the server-authoritative lookup result for
// a syntactically valid 0x4f request. It deliberately contains only fields that
// are authored by the exact-current shop.ini row; it does not derive binding,
// consumption order, inventory mutation, or persistence semantics.
type currentShopExchangeBuySelection struct {
	Request shopExchangeBuyRequest
	Item    shopCatalogItem
}

// currentShopExchangeBuyAuthority is the mutation-free, server-authoritative
// preflight result for a current 0x4f exchange purchase. It proves only that
// the request resolves to an exact-current mode3 row, that its ExchangeData is
// present in the authenticated current condition authority and classified SAFE
// for authoritative condition evaluation, and that the full ExchangeItem row
// parses without drifting from that authority projection. It is not permission
// to charge, grant, bind, persist, or replicate an item.
type currentShopExchangeBuyAuthority struct {
	Selection  currentShopExchangeBuySelection
	Definition shopExchangeDefinition
	Capability exchangeConditionCapability
}

func parseShopExchangeFormRequest(custom clientCustomMessage) (shopExchangeFormRequest, bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomRequestShopExchangeForm {
		return shopExchangeFormRequest{}, false, nil
	}
	// current custom_request_shop_exchange_form(viewid, bindindex, shopid, page, pos)
	// emits selector + int + int + string + int + int.
	if len(custom.Values) != 6 {
		return shopExchangeFormRequest{}, true, fmt.Errorf("shop exchange form request has %d values, want 6", len(custom.Values))
	}
	if custom.Values[1].Type != 2 || custom.Values[2].Type != 2 || custom.Values[3].Type != 6 || custom.Values[4].Type != 2 || custom.Values[5].Type != 2 {
		return shopExchangeFormRequest{}, true, fmt.Errorf("shop exchange form request type layout mismatch: %v", custom.Values)
	}
	return shopExchangeFormRequest{
		ViewIdent: custom.Values[1].Int32,
		BindIndex: custom.Values[2].Int32,
		ShopID:    custom.Values[3].Text,
		Page:      custom.Values[4].Int32,
		Position:  custom.Values[5].Int32,
	}, true, nil
}

func parseShopConditionDetailsRequest(custom clientCustomMessage) (shopConditionDetailsRequest, bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomRequestConditionShop {
		return shopConditionDetailsRequest{}, false, nil
	}
	// current custom_condition_details(exchange_index) emits selector + int32.
	if len(custom.Values) != 2 {
		return shopConditionDetailsRequest{}, true, fmt.Errorf("shop condition-details request has %d values, want 2", len(custom.Values))
	}
	if custom.Values[1].Type != 2 {
		return shopConditionDetailsRequest{}, true, fmt.Errorf("shop condition-details exchange index must be int32: %v", custom.Values)
	}
	return shopConditionDetailsRequest{ExchangeIndex: custom.Values[1].Int32}, true, nil
}

func parseShopExchangeBuyRequest(custom clientCustomMessage) (shopExchangeBuyRequest, bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomExchangeFromShop {
		return shopExchangeBuyRequest{}, false, nil
	}
	// current custom_exchange_item(shopid, page, pos, amount) emits selector +
	// string + int + int + int. ExchangeData is deliberately not on this wire.
	if len(custom.Values) != 5 {
		return shopExchangeBuyRequest{}, true, fmt.Errorf("shop exchange buy request has %d values, want 5", len(custom.Values))
	}
	if custom.Values[1].Type != 6 || custom.Values[2].Type != 2 || custom.Values[3].Type != 2 || custom.Values[4].Type != 2 {
		return shopExchangeBuyRequest{}, true, fmt.Errorf("shop exchange buy request type layout mismatch: %v", custom.Values)
	}
	return shopExchangeBuyRequest{
		ShopID:   custom.Values[1].Text,
		Page:     custom.Values[2].Int32,
		Position: custom.Values[3].Int32,
		Count:    custom.Values[4].Int32,
	}, true, nil
}

// resolveCurrentShopExchangeBuySelection rejects untrusted purchase coordinates
// unless they resolve back to one authored exact-current shop.ini row. The
// client does not send ConfigID, ExchangeData, price, or cost details in 0x4f;
// those values therefore must be re-resolved server-side before any future
// purchase implementation is allowed to inspect them.
//
// This helper is intentionally wire-inert and mutation-free. A successful
// lookup is NOT permission to charge, grant, bind, persist, or replicate an
// item; those semantics remain fail-closed until independently proven.
func resolveCurrentShopExchangeBuySelection(shopPath string, request shopExchangeBuyRequest) (currentShopExchangeBuySelection, bool, error) {
	if request.ShopID == "" || request.Count <= 0 {
		return currentShopExchangeBuySelection{}, false, nil
	}
	items, _, _, err := loadShopCatalogSection(shopPath, request.ShopID)
	if err != nil {
		return currentShopExchangeBuySelection{}, false, err
	}
	item := currentShopListing(items, request.Page, request.Position)
	if item == nil {
		return currentShopExchangeBuySelection{}, false, nil
	}
	if item.priceMode != 3 || item.exchangeData <= 0 {
		// 0x4f is the current exchange-purchase request. Do not reinterpret a
		// non-mode3 listing as an exchange purchase merely because the client
		// supplied coordinates that point at it.
		return currentShopExchangeBuySelection{}, false, nil
	}
	return currentShopExchangeBuySelection{Request: request, Item: *item}, true, nil
}

// resolveAuthorizedCurrentShopExchangeBuyAuthority layers the authenticated
// ExchangeData/condition authority over the coordinate re-resolution above.
// The caller is responsible for supplying an authority built from the same
// shop/exchange resources; the production wrapper below enforces that with the
// exact-current SHA256 gates in buildCurrentShopConditionAuthority.
func resolveAuthorizedCurrentShopExchangeBuyAuthority(shopPath, exchangePath string, authority *currentShopConditionAuthority, request shopExchangeBuyRequest) (currentShopExchangeBuyAuthority, bool, error) {
	if authority == nil {
		return currentShopExchangeBuyAuthority{}, false, fmt.Errorf("nil current shop condition authority")
	}
	selection, selected, err := resolveCurrentShopExchangeBuySelection(shopPath, request)
	if err != nil || !selected {
		return currentShopExchangeBuyAuthority{}, selected, err
	}
	exchangeData := selection.Item.exchangeData
	spec, ok := authority.definitions[exchangeData]
	if !ok {
		return currentShopExchangeBuyAuthority{}, false, nil
	}
	capability, ok := authority.audit.ByExchangeData[exchangeData]
	if !ok || !capability.Safe {
		return currentShopExchangeBuyAuthority{}, false, nil
	}
	definition, err := loadShopExchangeDefinition(exchangePath, exchangeData)
	if err != nil {
		return currentShopExchangeBuyAuthority{}, false, err
	}
	if definition.Condition != spec.Condition || definition.Condition2 != spec.Condition2 || definition.Filters != spec.Filters {
		return currentShopExchangeBuyAuthority{}, false, fmt.Errorf("ExchangeData %d authority projection drift", exchangeData)
	}
	return currentShopExchangeBuyAuthority{Selection: selection, Definition: definition, Capability: capability}, true, nil
}

// resolveDefaultCurrentShopExchangeBuyAuthority is the production preflight.
// loadDefaultCurrentShopConditionAuthority first authenticates shop.ini,
// ExchangeItem.ini, Condition.ini, condition_formula.ini and skill_maxlevel.ini
// against the exact-current fingerprints and invariants before any 0x4f row is
// accepted.
func resolveDefaultCurrentShopExchangeBuyAuthority(request shopExchangeBuyRequest) (currentShopExchangeBuyAuthority, bool, error) {
	authority, err := loadDefaultCurrentShopConditionAuthority()
	if err != nil {
		return currentShopExchangeBuyAuthority{}, false, err
	}
	return resolveAuthorizedCurrentShopExchangeBuyAuthority(defaultShopINIPath, defaultExchangeItemINIPath, authority, request)
}

// currentShopExchangeFormAuthority is display-only authority for one exact-current
// mode3 listing. It authenticates coordinates against shop.ini and reparses the
// referenced ExchangeItem.ini row; it does not evaluate purchase conditions or
// authorize any mutation.
type currentShopExchangeFormAuthority struct {
	Selection  currentShopExchangeBuySelection
	Definition shopExchangeDefinition
}

func resolveCurrentShopExchangeFormAuthority(shopPath, exchangePath string, request shopExchangeFormRequest) (currentShopExchangeFormAuthority, bool, error) {
	selection, ok, err := resolveCurrentShopExchangeBuySelection(shopPath, shopExchangeBuyRequest{
		ShopID: request.ShopID, Page: request.Page, Position: request.Position, Count: 1,
	})
	if err != nil || !ok {
		return currentShopExchangeFormAuthority{}, ok, err
	}
	definition, err := loadShopExchangeDefinition(exchangePath, selection.Item.exchangeData)
	if err != nil {
		return currentShopExchangeFormAuthority{}, false, err
	}
	return currentShopExchangeFormAuthority{Selection: selection, Definition: definition}, true, nil
}

func resolveDefaultCurrentShopExchangeFormAuthority(request shopExchangeFormRequest) (currentShopExchangeFormAuthority, bool, error) {
	if err := requireFileSHA256(defaultShopINIPath, exactCurrentShopINISHA256); err != nil {
		return currentShopExchangeFormAuthority{}, false, err
	}
	if err := requireFileSHA256(defaultExchangeItemINIPath, exactCurrentExchangeItemSHA256); err != nil {
		return currentShopExchangeFormAuthority{}, false, err
	}
	return resolveCurrentShopExchangeFormAuthority(defaultShopINIPath, defaultExchangeItemINIPath, request)
}

// buildAuthorizedNoPreviewExchangeFormFrame opens only the exact-current subset
// whose authored BindStatus disables the binding-preview branch entirely. For
// BindStatus <= 0, exact-current form_exchange.lua bypasses ShowBind and
// ExchangeBind presentation, so zero placeholders are inert on that proven
// client path. BindStatus > 0 remains fail-closed until the authoritative
// runtime derivation of ShowBind/ExchangeBind is proven.
//
// This helper is display-only: it does not authorize an exchange mutation.
func buildAuthorizedNoPreviewExchangeFormFrame(authority currentShopExchangeFormAuthority, request shopExchangeFormRequest) ([]byte, bool, error) {
	if authority.Definition.BindStatus > 0 {
		return nil, false, nil
	}
	config, err := encodeShopExchangeConfig(authority.Definition, shopExchangeRuntimeBind{})
	if err != nil {
		return nil, false, err
	}
	frame, err := serverShopExchangeFormMessage(request, config)
	if err != nil {
		return nil, false, err
	}
	return frame, true, nil
}

// handleShopExchangeContract recognizes the exact current-client request
// layouts without pretending that the server-side exchange implementation is
// complete. S2C 557 is enabled only for authenticated mode3 definitions whose
// BindStatus <= 0, where exact-current Lua bypasses the ShowBind/ExchangeBind
// preview branch. BindStatus > 0 remains fail-closed until those runtime values
// have a proven server-authoritative derivation. The 0x4f mutation path remains
// fail-closed independently.
func handleShopExchangeContract(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if request, matched, err := parseShopExchangeFormRequest(custom); matched {
		if err != nil {
			return true, err
		}
		if link == nil {
			return true, fmt.Errorf("shop exchange form requires active scene connection")
		}
		authority, authorized, loadErr := resolveDefaultCurrentShopExchangeFormAuthority(request)
		if loadErr != nil {
			log.Printf("%s: current shop exchange form request view=%d bind=%d shop=%s page=%d pos=%d blocked: exact-current shop/exchange authority unavailable: %v",
				remote, request.ViewIdent, request.BindIndex, request.ShopID, request.Page, request.Position, loadErr)
			return true, nil
		}
		if !authorized {
			log.Printf("%s: current shop exchange form request view=%d bind=%d shop=%s page=%d pos=%d blocked: request is not an authenticated mode3 exchange listing",
				remote, request.ViewIdent, request.BindIndex, request.ShopID, request.Page, request.Position)
			return true, nil
		}
		frame, sendable, frameErr := buildAuthorizedNoPreviewExchangeFormFrame(authority, request)
		if frameErr != nil {
			return true, fmt.Errorf("build shop exchange form shop=%s page=%d pos=%d: %w", request.ShopID, request.Page, request.Position, frameErr)
		}
		if !sendable {
			log.Printf("%s: current shop exchange form request view=%d bind=%d shop=%s page=%d pos=%d blocked: BindStatus preview requires unresolved ShowBind/ExchangeBind",
				remote, request.ViewIdent, request.BindIndex, request.ShopID, request.Page, request.Position)
			return true, nil
		}
		if writeErr := link.WriteFrame(frame); writeErr != nil {
			return true, fmt.Errorf("write shop exchange form shop=%s page=%d pos=%d: %w", request.ShopID, request.Page, request.Position, writeErr)
		}
		log.Printf("%s: current shop exchange form sent view=%d bind=%d shop=%s page=%d pos=%d bind_preview=disabled",
			remote, request.ViewIdent, request.BindIndex, request.ShopID, request.Page, request.Position)
		return true, nil
	}
	if request, matched, err := parseShopConditionDetailsRequest(custom); matched {
		if err != nil {
			return true, err
		}
		if link == nil || player == nil {
			return true, fmt.Errorf("shop condition-details requires active scene/player")
		}
		authority, loadErr := loadDefaultCurrentShopConditionAuthority()
		if loadErr != nil {
			log.Printf("%s: current shop condition-details request exchange_index=%d blocked: exact-current resource authority unavailable: %v",
				remote, request.ExchangeIndex, loadErr)
			return true, nil
		}
		definition, ok := authority.definitions[request.ExchangeIndex]
		if !ok {
			log.Printf("%s: current shop condition-details request exchange_index=%d blocked: not in exact-current mode3 ExchangeData set",
				remote, request.ExchangeIndex)
			return true, nil
		}
		evaluator := exactCurrentConditionEvaluator{
			catalog:       authority.catalog,
			player:        player,
			skillMaxTable: authority.skillMaxTable,
		}
		details, allSupported, evalErr := evaluateExchangeConditionDetails(definition.Condition, definition.Condition2, definition.Filters, authority.catalog.resolve, evaluator.evaluate)
		if evalErr != nil {
			return true, fmt.Errorf("evaluate shop condition-details exchange_index=%d: %w", request.ExchangeIndex, evalErr)
		}
		frame, encodeErr := serverShopConditionDetailsMessage(details)
		if encodeErr != nil {
			return true, fmt.Errorf("encode shop condition-details exchange_index=%d: %w", request.ExchangeIndex, encodeErr)
		}
		if writeErr := link.WriteFrame(frame); writeErr != nil {
			return true, fmt.Errorf("write shop condition-details exchange_index=%d: %w", request.ExchangeIndex, writeErr)
		}
		log.Printf("%s: current shop condition-details exchange_index=%d sent triples=%d all_supported=%t fail_closed=%t",
			remote, request.ExchangeIndex, len(details), allSupported, !allSupported)
		return true, nil
	}
	if request, matched, err := parseShopExchangeBuyRequest(custom); matched {
		if err != nil {
			return true, err
		}
		authorized, selected, selectErr := resolveDefaultCurrentShopExchangeBuyAuthority(request)
		if selectErr != nil {
			log.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d blocked: exact-current shop/exchange authority unavailable: %v",
				remote, request.ShopID, request.Page, request.Position, request.Count, selectErr)
			return true, nil
		}
		if !selected {
			log.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d blocked: request is not an authenticated SAFE mode3 exchange listing",
				remote, request.ShopID, request.Page, request.Position, request.Count)
			return true, nil
		}
		log.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d authenticated config=%s exchange_data=%d leaves=%d blocked: condition acceptance/cost/bind/commit path unresolved",
			remote, request.ShopID, request.Page, request.Position, request.Count, authorized.Selection.Item.configID, authorized.Selection.Item.exchangeData, len(authorized.Capability.Leaves))
		return true, nil
	}
	return false, nil
}
