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

// handleShopExchangeContract recognizes the exact current-client request
// layouts without pretending that the server-side exchange implementation is
// complete. The exact current InitCurExchangeData 11-field grammar is now
// implemented separately. This handler still blocks before S2C 557 because
// ShowBind and ExchangeBind are runtime response values not authored by current
// ExchangeItem.ini, and their server-authoritative derivation is not yet proven.
func handleShopExchangeContract(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if request, matched, err := parseShopExchangeFormRequest(custom); matched {
		if err != nil {
			return true, err
		}
		log.Printf("%s: current shop exchange form request view=%d bind=%d shop=%s page=%d pos=%d blocked: ShowBind/ExchangeBind server rule unresolved",
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
		selection, selected, selectErr := resolveCurrentShopExchangeBuySelection(defaultShopINIPath, request)
		if selectErr != nil {
			log.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d blocked: exact-current shop authority unavailable: %v",
				remote, request.ShopID, request.Page, request.Position, request.Count, selectErr)
			return true, nil
		}
		if !selected {
			log.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d blocked: request does not resolve to an authored mode3 exchange listing",
				remote, request.ShopID, request.Page, request.Position, request.Count)
			return true, nil
		}
		log.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d re-resolved config=%s exchange_data=%d blocked: exchange cost/condition commit path unresolved",
			remote, request.ShopID, request.Page, request.Position, request.Count, selection.Item.configID, selection.Item.exchangeData)
		return true, nil
	}
	return false, nil
}
