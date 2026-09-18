package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/local/9yin-go-server/internal/role"
)

// resyncOrdinaryShopWalletAfterConflict is used only when an ordinary NPC
// purchase lost the database wallet comparison. It does not retry a purchase,
// debit currency, or issue an item reward. The currency frame is the existing
// four-property actor update, not a new client protocol message.
func resyncOrdinaryShopWalletAfterConflict(link sceneMessageConnection, player *playerActor, store currencyStoreIface, roleID role.RoleID, expectedActor currencySnapshot) error {
	mysqlStore, ok := store.(*mysqlCurrencyStore)
	if !ok || mysqlStore == nil || mysqlStore.db == nil || roleID == 0 || player == nil || link == nil {
		return errors.New("shop wallet conflict: missing live MySQL store, role, player, or connection")
	}
	var encoded []byte
	if err := mysqlStore.db.QueryRowContext(context.Background(),
		"SELECT snapshot FROM role_currency WHERE role_id = ?", roleID).Scan(&encoded); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("shop wallet conflict: persisted wallet is missing: %w", err)
		}
		return fmt.Errorf("shop wallet conflict: reload latest wallet: %w", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil || object == nil {
		return fmt.Errorf("shop wallet conflict: invalid persisted wallet object: %v", err)
	}
	var latest currencySnapshot
	if err := json.Unmarshal(encoded, &latest); err != nil {
		return fmt.Errorf("shop wallet conflict: decode persisted wallet: %w", err)
	}
	frame, err := ordinaryShopCurrencyFrame(latest)
	if err != nil {
		return fmt.Errorf("shop wallet conflict: encode existing currency update: %w", err)
	}

	// Refuse to discard local currency changes that are not part of this failed
	// purchase. Do not overwrite a wallet changed after the purchase snapshot;
	// do not silently discard changes made since an earlier committed purchase.
	player.mu.Lock()
	current := currencySnapshot{Silver: player.silver, Gold: player.gold,
		SilverCard: player.silverCard, SilverTicket: player.silverTicket}
	if current != expectedActor || (player.shopWallet != nil && current != *player.shopWallet) {
		player.mu.Unlock()
		return errors.New("shop wallet conflict: unpersisted local currency changes; refusing to discard actor wallet")
	}
	player.silver, player.gold = latest.Silver, latest.Gold
	player.silverCard, player.silverTicket = latest.SilverCard, latest.SilverTicket
	// Mark the freshly read database wallet as the disconnect-save baseline:
	// exiting this stale session must not overwrite another session's wallet.
	player.shopWallet = &latest
	player.mu.Unlock()
	if err := link.WriteFrame(frame); err != nil {
		return fmt.Errorf("shop wallet conflict: publish currency refresh: %w", err)
	}
	return nil
}
