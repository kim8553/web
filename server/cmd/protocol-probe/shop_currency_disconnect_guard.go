package main

import (
	"encoding/json"
	"fmt"

	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuyatomic"
)

// markOrdinaryShopWalletCommitted records only a successful ordinary NPC
// purchase. It does not change the existing currency store or GM handlers.
func (p *playerActor) markOrdinaryShopWalletCommitted(wallet currencySnapshot) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shopWallet = &wallet
}

// Capture the committed reference and current actor wallet under one lock.
func (p *playerActor) ordinaryShopWalletSinceCommit() (currencySnapshot, currencySnapshot, bool) {
	if p == nil {
		return currencySnapshot{}, currencySnapshot{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.shopWallet == nil {
		return currencySnapshot{}, currencySnapshot{}, false
	}
	current := currencySnapshot{
		Silver: p.silver, Gold: p.gold,
		SilverCard: p.silverCard, SilverTicket: p.silverTicket,
	}
	return *p.shopWallet, current, true
}

func (p *playerActor) ordinaryShopWalletUnchangedSinceCommit() bool {
	committed, current, ok := p.ordinaryShopWalletSinceCommit()
	return ok && current == committed
}

// A committed ordinary NPC purchase must not be overwritten by a deferred
// save from a stale session. A subsequent local wallet change is saved only
// if MySQL still contains that session's last committed purchase wallet.
// JSON stores and sessions without a committed purchase keep their existing
// behavior. GM and exchange handlers are not modified.
func persistDeferredShopCurrency(store currencyStoreIface, roleID role.RoleID, player *playerActor) error {
	if store == nil || player == nil {
		return nil
	}
	if mysqlStore, ok := store.(*mysqlCurrencyStore); ok && mysqlStore != nil && mysqlStore.db != nil {
		committed, current, hasPurchase := player.ordinaryShopWalletSinceCommit()
		if hasPurchase {
			if current == committed {
				return nil
			}
			expectedJSON, err := json.Marshal(committed)
			if err != nil {
				return fmt.Errorf("encode ordinary purchase wallet: %w", err)
			}
			nextJSON, err := json.Marshal(current)
			if err != nil {
				return fmt.Errorf("encode ordinary changed wallet: %w", err)
			}
			return shopbuyatomic.SaveWalletChecked(mysqlStore.db, uint64(roleID), expectedJSON, nextJSON)
		}
	}
	var snapshot currencySnapshot
	snapshot.fromActor(player)
	return store.Save(roleID, snapshot)
}
