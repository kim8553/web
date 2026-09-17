package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// The exact current FxGameLogic.dll exchange_item_manager::InitCurExchangeData
// parser consumes exactly eleven pipe-separated fields in this order. Item and
// Prop keep the authored semicolon/comma sub-grammar; Condition, Condition2 and
// Filters keep the authored comma-separated integer-list grammar.
var defaultExchangeItemINIPath = filepath.Join(defaultModernShareRoot, "item", "exchangeitem.ini")

type shopExchangeDefinition struct {
	Type          int32
	AddValue      string
	BindStatus    int32
	Item          string
	ConditionType int32
	Condition     string
	Condition2    string
	Filters       string
	Prop          string
}

type shopExchangeRuntimeBind struct {
	ShowBind     int32
	ExchangeBind int32
}

// loadShopExchangeDefinition reads only values authored in the current
// share/Item/ExchangeItem.ini schema. ShowBind and ExchangeBind are deliberately
// absent: current ExchangeItem.ini does not author those two response fields.
func loadShopExchangeDefinition(path string, exchangeData int32) (shopExchangeDefinition, error) {
	if exchangeData <= 0 {
		return shopExchangeDefinition{}, fmt.Errorf("exchange data id must be positive, got %d", exchangeData)
	}
	table, err := loadINISections(path)
	if err != nil {
		return shopExchangeDefinition{}, fmt.Errorf("load exchange item table %s: %w", path, err)
	}
	fields, ok := table[strconv.FormatInt(int64(exchangeData), 10)]
	if !ok {
		return shopExchangeDefinition{}, fmt.Errorf("exchange item table %s has no section %d", path, exchangeData)
	}
	definition := shopExchangeDefinition{
		Type:          iniInt(fields, "Type"),
		AddValue:      strings.TrimSpace(iniValue(fields, "AddValue")),
		BindStatus:    iniInt(fields, "BindStatus"),
		Item:          strings.TrimSpace(iniValue(fields, "Item")),
		ConditionType: iniInt(fields, "ConditionType"),
		Condition:     strings.TrimSpace(iniValue(fields, "Condition")),
		Condition2:    strings.TrimSpace(iniValue(fields, "Condition2")),
		Filters:       strings.TrimSpace(iniValue(fields, "Filters")),
		Prop:         strings.TrimSpace(iniValue(fields, "Prop")),
	}
	if err := validateShopExchangeDefinition(definition); err != nil {
		return shopExchangeDefinition{}, fmt.Errorf("exchange item section %d: %w", exchangeData, err)
	}
	return definition, nil
}

func validateShopExchangeDefinition(definition shopExchangeDefinition) error {
	// InitCurExchangeData has no escaping layer for the top-level pipe delimiter.
	for name, value := range map[string]string{
		"AddValue":   definition.AddValue,
		"Item":       definition.Item,
		"Condition":  definition.Condition,
		"Condition2": definition.Condition2,
		"Filters":    definition.Filters,
		"Prop":       definition.Prop,
	} {
		if strings.Contains(value, "|") {
			return fmt.Errorf("%s contains unsupported top-level delimiter", name)
		}
	}
	if err := validateExchangePairList(definition.Item, 32, "Item"); err != nil {
		return err
	}
	if err := validateExchangePairList(definition.Prop, 64, "Prop"); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"Condition":  definition.Condition,
		"Condition2": definition.Condition2,
		"Filters":    definition.Filters,
	} {
		if err := validateExchangeIntList(value, name); err != nil {
			return err
		}
	}
	return nil
}

func validateExchangePairList(value string, bits int, name string) error {
	if value == "" {
		return nil
	}
	entries := strings.Split(value, ";")
	// Exact-current FxGameLogic.dll InitCurExchangeData skips a final empty
	// semicolon token (fewer than two comma-separated fields) for BOTH Item
	// (0x11B1593C..0x11B15947) and Prop (0x11B166DC..0x11B166E7).
	// Keep raw authored strings intact in S2C 557; this only aligns display
	// grammar, and never authorizes exchange debit, grant or persistence.
	if (name == "Item" || name == "Prop") && entries[len(entries)-1] == "" {
		entries = entries[:len(entries)-1]
	}
	for _, entry := range entries {
		parts := strings.Split(entry, ",")
		if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" {
			return fmt.Errorf("%s entry %q does not match current name,value grammar", name, entry)
		}
		if _, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, bits); err != nil {
			return fmt.Errorf("%s entry %q has invalid integer value: %w", name, entry, err)
		}
	}
	return nil
}

func validateExchangeIntList(value, name string) error {
	if value == "" {
		return nil
	}
	for _, token := range strings.Split(value, ",") {
		if _, err := strconv.ParseInt(strings.TrimSpace(token), 10, 32); err != nil {
			return fmt.Errorf("%s token %q is not int32: %w", name, token, err)
		}
	}
	return nil
}

// encodeShopExchangeConfig emits the field order accepted by the exact current
// InitCurExchangeData parser:
// Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop.
//
// The caller must supply ShowBind/ExchangeBind from a separately proven server
// rule. This helper does not synthesize them.
func encodeShopExchangeConfig(definition shopExchangeDefinition, bind shopExchangeRuntimeBind) (string, error) {
	if err := validateShopExchangeDefinition(definition); err != nil {
		return "", err
	}
	fields := []string{
		strconv.FormatInt(int64(definition.Type), 10),
		definition.AddValue,
		strconv.FormatInt(int64(definition.BindStatus), 10),
		definition.Item,
		strconv.FormatInt(int64(bind.ShowBind), 10),
		strconv.FormatInt(int64(bind.ExchangeBind), 10),
		strconv.FormatInt(int64(definition.ConditionType), 10),
		definition.Condition,
		definition.Condition2,
		definition.Filters,
		definition.Prop,
	}
	return strings.Join(fields, "|"), nil
}

// serverShopExchangeFormMessage matches the exact current Lua handler
// on_open_shop_exchange_form(view_ident, bind_index, shop_id, page, pos,
// config_str), registered as numeric server custom message 557.
func serverShopExchangeFormMessage(request shopExchangeFormRequest, config string) ([]byte, error) {
	return serverCustomIntMessage(557,
		customInt(request.ViewIdent),
		customInt(request.BindIndex),
		customString(request.ShopID),
		customInt(request.Page),
		customInt(request.Position),
		customString(config),
	)
}

// serverShopConditionDetailsMessage matches current S2C custom message 508.
// Payload is a flat repetition of (condition_id, isneedshow, satisfied).
func serverShopConditionDetailsMessage(details []shopConditionDetail) ([]byte, error) {
	values := make([]serverCustomValue, 0, len(details)*3)
	for _, detail := range details {
		needShow := int32(0)
		if detail.IsNeedShow {
			needShow = 1
		}
		satisfied := int32(0)
		if detail.Satisfied {
			satisfied = 1
		}
		values = append(values, customInt(detail.ConditionID), customInt(needShow), customInt(satisfied))
	}
	return serverCustomIntMessage(508, values...)
}
