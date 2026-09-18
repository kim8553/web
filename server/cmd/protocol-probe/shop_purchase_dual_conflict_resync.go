package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/role"
)

// resyncOrdinaryShopPurchaseAfterWalletConflict handles a rejected ordinary
// NPC purchase whose wallet was stale. SaveCheckedBag checks the wallet before
// the bag, so the bag may ALSO be stale even though only ErrWalletChanged was
// returned. Reuse both already-established refresh paths; do not retry the
// purchase or emit the rejected reward. Any failed refresh must terminate the
// caller's session instead of claiming client synchronization.
func resyncOrdinaryShopPurchaseAfterWalletConflict(link sceneMessageConnection, player *playerActor, bagStore bagStoreIface, currencyStore currencyStoreIface, roleID role.RoleID, beforeBag []bagItem, beforeWallet currencySnapshot) error {
	if err := resyncOrdinaryShopWalletAfterConflict(link, player, currencyStore, roleID, beforeWallet); err != nil {
		return fmt.Errorf("shop purchase conflict: refresh wallet: %w", err)
	}
	if err := resyncOrdinaryShopBagAfterConflict(link, player, bagStore, roleID, beforeBag); err != nil {
		return fmt.Errorf("shop purchase conflict: refresh bag: %w", err)
	}
	return nil
}
