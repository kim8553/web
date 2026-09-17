package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/local/9yin-go-server/internal/role"
)

// normalShopJSONCommitMu serializes the legacy JSON bag/currency pair. The
// original 9yin-go-server1 runtime supports JSON storage when MySQL is not
// configured, so normal NPC retail must not require an unrelated MySQL setup.
var normalShopJSONCommitMu sync.Mutex

func cloneNormalShopBagRoles(src map[string][]bagItem) map[string][]bagItem {
	out := make(map[string][]bagItem, len(src)+1)
	for key, items := range src {
		out[key] = append([]bagItem(nil), items...)
	}
	return out
}

func cloneNormalShopCurrencyRoles(src map[string]currencySnapshot) map[string]currencySnapshot {
	out := make(map[string]currencySnapshot, len(src)+1)
	for key, value := range src {
		out[key] = value
	}
	return out
}

// normalShopAtomicWriteFile writes a complete replacement beside the target,
// fsyncs it, then renames it into place. This matches the existing JSON store
// model while avoiding partially written JSON documents on normal write errors.
func normalShopAtomicWriteFile(path string, data []byte) error {
	if filepath.Clean(path) == "." || path == "" {
		return errors.New("normal shop JSON write: empty path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("normal shop JSON mkdir: %w", err)
	}
	file, err := os.CreateTemp(dir, filepath.Base(path)+".normal-shop-*")
	if err != nil {
		return fmt.Errorf("normal shop JSON temp: %w", err)
	}
	tmp := file.Name()
	removeTmp := true
	defer func() {
		_ = file.Close()
		if removeTmp {
			_ = os.Remove(tmp)
		}
	}()
	if err := file.Chmod(0o644); err != nil {
		return fmt.Errorf("normal shop JSON chmod: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("normal shop JSON write: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("normal shop JSON sync: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("normal shop JSON close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("normal shop JSON replace %s: %w", filepath.Base(path), err)
	}
	removeTmp = false
	return nil
}

// normalShopCommitJSONSnapshots persists bag and currency as one guarded retail
// operation for the JSON-backed 9yin-go-server1 runtime. Both in-memory stores
// are locked together. If the second file replacement fails, the first file is
// restored to its exact pre-operation JSON snapshot before returning an error.
//
// This provides operation-level all-or-rollback behavior for normal process
// errors. It is not advertised as crash/power-loss atomic across two files; a
// database transaction remains stronger for that failure model.
func normalShopCommitJSONSnapshots(bag *bagStore, currency *currencyStore, roleID role.RoleID, afterCurrency normalShopPersistedCurrency, afterBag []bagItem) error {
	if bag == nil || currency == nil {
		return errors.New("normal shop JSON commit: nil store")
	}
	if roleID == 0 {
		return errors.New("normal shop JSON commit: zero role id")
	}
	if bag.path == "" || currency.path == "" || filepath.Clean(bag.path) == filepath.Clean(currency.path) {
		return errors.New("normal shop JSON commit: invalid store paths")
	}

	normalShopJSONCommitMu.Lock()
	defer normalShopJSONCommitMu.Unlock()
	bag.mu.Lock()
	defer bag.mu.Unlock()
	currency.mu.Lock()
	defer currency.mu.Unlock()

	key := strconv.FormatUint(uint64(roleID), 10)
	oldBagRoles := cloneNormalShopBagRoles(bag.roles)
	newBagRoles := cloneNormalShopBagRoles(bag.roles)
	newCurrencyRoles := cloneNormalShopCurrencyRoles(currency.roles)
	newBagRoles[key] = append([]bagItem(nil), afterBag...)
	newCurrencyRoles[key] = currencySnapshot{
		Silver: afterCurrency.Silver, Gold: afterCurrency.Gold,
		SilverCard: afterCurrency.SilverCard, SilverTicket: afterCurrency.SilverTicket,
	}

	oldBagBytes, err := json.MarshalIndent(bagStoreFile{Version: bagStoreVersion, Roles: oldBagRoles}, "", "  ")
	if err != nil {
		return fmt.Errorf("normal shop JSON encode old bag: %w", err)
	}
	newBagBytes, err := json.MarshalIndent(bagStoreFile{Version: bagStoreVersion, Roles: newBagRoles}, "", "  ")
	if err != nil {
		return fmt.Errorf("normal shop JSON encode bag: %w", err)
	}
	newCurrencyBytes, err := json.MarshalIndent(currencyStoreFile{Version: currencyStoreVersion, Roles: newCurrencyRoles}, "", "  ")
	if err != nil {
		return fmt.Errorf("normal shop JSON encode currency: %w", err)
	}

	if err := normalShopAtomicWriteFile(bag.path, newBagBytes); err != nil {
		return err
	}
	if err := normalShopAtomicWriteFile(currency.path, newCurrencyBytes); err != nil {
		rollbackErr := normalShopAtomicWriteFile(bag.path, oldBagBytes)
		if rollbackErr != nil {
			return fmt.Errorf("normal shop JSON currency write failed: %v; bag rollback failed: %w", err, rollbackErr)
		}
		return fmt.Errorf("normal shop JSON currency write failed; bag rolled back: %w", err)
	}

	bag.roles = newBagRoles
	currency.roles = newCurrencyRoles
	return nil
}
