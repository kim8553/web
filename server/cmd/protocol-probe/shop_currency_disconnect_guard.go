package main

import "github.com/local/9yin-go-server/internal/role"

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

func (p *playerActor) ordinaryShopWalletUnchangedSinceCommit() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	wallet := p.shopWallet
	return wallet != nil && p.silver == wallet.Silver && p.gold == wallet.Gold &&
		p.silverCard == wallet.SilverCard && p.silverTicket == wallet.SilverTicket
}

// persistDeferredShopCurrency avoids overwriting a newer wallet from another
// session when this session's ordinary purchase has already committed exactly
// the same wallet to MySQL. Explicit currency saves remain unchanged. Any
// further local wallet change continues through the existing save path; it
// requires separate concurrency protection and is not claimed safe here.
func persistDeferredShopCurrency(store currencyStoreIface, roleID role.RoleID, player *playerActor) error {
	if store == nil || player == nil {
		return nil
	}
	if mysqlStore, ok := store.(*mysqlCurrencyStore); ok && mysqlStore != nil &&
		mysqlStore.db != nil && player.ordinaryShopWalletUnchangedSinceCommit() {
		return nil
	}
	var snapshot currencySnapshot
	snapshot.fromActor(player)
	return store.Save(roleID, snapshot)
}
