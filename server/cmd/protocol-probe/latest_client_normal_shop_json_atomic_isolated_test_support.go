//go:build retailjsonisolated

package main

import "sync"

const bagStoreVersion = 1
const currencyStoreVersion = 1

type bagItem struct {
	ConfigID string `json:"config_id"`
	ItemType int32  `json:"item_type"`
	Amount   int32  `json:"amount"`
	ViewID   int32  `json:"view_id"`
	Slot     int32  `json:"slot"`
}

type bagStore struct {
	mu    sync.Mutex
	path  string
	roles map[string][]bagItem
}

type bagStoreFile struct {
	Version int                  `json:"version"`
	Roles   map[string][]bagItem `json:"roles"`
}

type currencySnapshot struct {
	Silver       int32 `json:"silver"`
	Gold         int32 `json:"gold"`
	SilverCard   int32 `json:"silver_card"`
	SilverTicket int32 `json:"silver_ticket"`
}

type currencyStore struct {
	mu    sync.Mutex
	path  string
	roles map[string]currencySnapshot
}

type currencyStoreFile struct {
	Version int                         `json:"version"`
	Roles   map[string]currencySnapshot `json:"roles"`
}

type normalShopPersistedCurrency struct {
	Silver       int32
	Gold         int32
	SilverCard   int32
	SilverTicket int32
}
