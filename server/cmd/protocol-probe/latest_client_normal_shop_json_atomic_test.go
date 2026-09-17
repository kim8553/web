package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

func readBagStoreFileForTest(t *testing.T, path string) bagStoreFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc bagStoreFile
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func readCurrencyStoreFileForTest(t *testing.T, path string) currencyStoreFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc currencyStoreFile
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestNormalShopCommitJSONSnapshotsSuccess(t *testing.T) {
	dir := t.TempDir()
	bagPath := filepath.Join(dir, "bag_items.json")
	currencyPath := filepath.Join(dir, "currency.json")
	bag := &bagStore{path: bagPath, roles: map[string][]bagItem{
		"7": {{ConfigID: "old_item", ItemType: 1, Amount: 1, ViewID: 1, Slot: 1}},
	}}
	currency := &currencyStore{path: currencyPath, roles: map[string]currencySnapshot{
		"7": {Silver: 100, Gold: 20, SilverCard: 30, SilverTicket: 40},
	}}
	if err := normalShopAtomicWriteFile(bagPath, mustJSON(t, bagStoreFile{Version: bagStoreVersion, Roles: cloneNormalShopBagRoles(bag.roles)})); err != nil {
		t.Fatal(err)
	}
	if err := normalShopAtomicWriteFile(currencyPath, mustJSON(t, currencyStoreFile{Version: currencyStoreVersion, Roles: cloneNormalShopCurrencyRoles(currency.roles)})); err != nil {
		t.Fatal(err)
	}
	afterBag := []bagItem{
		{ConfigID: "old_item", ItemType: 1, Amount: 1, ViewID: 1, Slot: 1},
		{ConfigID: "shop_item", ItemType: 1, Amount: 1, ViewID: 1, Slot: 2},
	}
	afterCurrency := normalShopPersistedCurrency{Silver: 75, Gold: 20, SilverCard: 30, SilverTicket: 40}
	if err := normalShopCommitJSONSnapshots(bag, currency, role.RoleID(7), afterCurrency, afterBag); err != nil {
		t.Fatal(err)
	}
	bagDoc := readBagStoreFileForTest(t, bagPath)
	if got := bagDoc.Roles["7"]; len(got) != 2 || got[1].ConfigID != "shop_item" || got[1].Slot != 2 {
		t.Fatalf("unexpected persisted bag: %#v", got)
	}
	currencyDoc := readCurrencyStoreFileForTest(t, currencyPath)
	if got := currencyDoc.Roles["7"]; got.Silver != 75 || got.Gold != 20 || got.SilverCard != 30 || got.SilverTicket != 40 {
		t.Fatalf("unexpected persisted currency: %#v", got)
	}
	if got := bag.roles["7"]; len(got) != 2 || got[1].ConfigID != "shop_item" {
		t.Fatalf("in-memory bag not committed: %#v", got)
	}
	if got := currency.roles["7"].Silver; got != 75 {
		t.Fatalf("in-memory currency=%d want 75", got)
	}
}

func TestNormalShopCommitJSONSnapshotsRollsBagBackWhenCurrencyReplaceFails(t *testing.T) {
	dir := t.TempDir()
	bagPath := filepath.Join(dir, "bag_items.json")
	currencyPath := filepath.Join(dir, "currency-target-is-directory")
	if err := os.Mkdir(currencyPath, 0o755); err != nil {
		t.Fatal(err)
	}
	beforeBag := []bagItem{{ConfigID: "old_item", ItemType: 1, Amount: 1, ViewID: 1, Slot: 1}}
	bag := &bagStore{path: bagPath, roles: map[string][]bagItem{"7": append([]bagItem(nil), beforeBag...)}}
	currency := &currencyStore{path: currencyPath, roles: map[string]currencySnapshot{"7": {Silver: 100}}}
	beforeBytes := mustJSON(t, bagStoreFile{Version: bagStoreVersion, Roles: cloneNormalShopBagRoles(bag.roles)})
	if err := normalShopAtomicWriteFile(bagPath, beforeBytes); err != nil {
		t.Fatal(err)
	}
	err := normalShopCommitJSONSnapshots(bag, currency, role.RoleID(7), normalShopPersistedCurrency{Silver: 90}, []bagItem{
		{ConfigID: "old_item", ItemType: 1, Amount: 1, ViewID: 1, Slot: 1},
		{ConfigID: "shop_item", ItemType: 1, Amount: 1, ViewID: 1, Slot: 2},
	})
	if err == nil {
		t.Fatal("expected currency replace failure")
	}
	bagDoc := readBagStoreFileForTest(t, bagPath)
	if got := bagDoc.Roles["7"]; len(got) != 1 || got[0].ConfigID != "old_item" {
		t.Fatalf("bag file was not rolled back: %#v", got)
	}
	if got := bag.roles["7"]; len(got) != 1 || got[0].ConfigID != "old_item" {
		t.Fatalf("in-memory bag changed on failure: %#v", got)
	}
	if got := currency.roles["7"].Silver; got != 100 {
		t.Fatalf("in-memory currency changed on failure: %d", got)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return data
}
