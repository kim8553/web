package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/shopbuycommit"
	"github.com/local/9yin-go-server/internal/shopbuygate"
)

// currentShopBuyAtomicPersistenceTarget accepts only the exact MySQL topology
// proven in Stage14/17: bag and currency stores backed by the same *sql.DB.
// JSON fallback and split databases remain fail-closed.
func currentShopBuyAtomicPersistenceTarget(roleID role.RoleID, bagStore bagStoreIface, currencyStore currencyStoreIface) (*mysqlBagStore, *mysqlCurrencyStore, bool) {
	if roleID == 0 {
		return nil, nil, false
	}
	mysqlBag, bagOK := bagStore.(*mysqlBagStore)
	mysqlCurrency, currencyOK := currencyStore.(*mysqlCurrencyStore)
	if !bagOK || !currencyOK || mysqlBag == nil || mysqlCurrency == nil || mysqlBag.db == nil || mysqlCurrency.db == nil {
		return nil, nil, false
	}
	if mysqlBag.db != mysqlCurrency.db {
		return nil, nil, false
	}
	return mysqlBag, mysqlCurrency, true
}

// commitShopPurchaseMySQLPersistenceFirst is a dormant durable/live commit
// helper. It does not parse or assign an ordinary-shop wire selector and it
// does not publish client frames. A caller must supply an already-evaluated
// shopbuygate decision. While player.mu is held, it snapshots live state,
// stages the purchase, commits bag+currency through one MySQL transaction, and
// only then applies the staged values to memory. Client publication belongs
// after this helper returns successfully.
func commitShopPurchaseMySQLPersistenceFirst(
	roleID role.RoleID,
	bagStore bagStoreIface,
	currencyStore currencyStoreIface,
	itemCatalog *itemCatalog,
	player *playerActor,
	listing shopCatalogItem,
	count int32,
	gate shopbuygate.Decision,
) (stagedShopPurchase, error) {
	var zero stagedShopPurchase
	if player == nil {
		return zero, fmt.Errorf("shop buy commit: player unavailable")
	}
	if err := gate.Require(); err != nil {
		return zero, fmt.Errorf("shop buy commit: %w", err)
	}
	mysqlBag, mysqlCurrency, ok := currentShopBuyAtomicPersistenceTarget(roleID, bagStore, currencyStore)
	if !ok {
		return zero, fmt.Errorf("shop buy commit: atomic MySQL bag/currency target unavailable")
	}

	return shopbuycommit.PersistenceFirstUnderLock(&player.mu, gate,
		func() (stagedShopPurchase, error) {
			bagBefore := append([]bagItem(nil), player.bagItems...)
			currencyBefore := currencySnapshot{
				Silver:       player.silver,
				Gold:         player.gold,
				SilverCard:   player.silverCard,
				SilverTicket: player.silverTicket,
			}
			return stageShopPurchaseSnapshot(itemCatalog, bagBefore, currencyBefore, listing, count)
		},
		func(staged stagedShopPurchase) error {
			return persistShopPurchaseMySQLAtomic(roleID, mysqlBag, mysqlCurrency, staged.BagAfter, staged.CurrencyAfter)
		},
		func(staged stagedShopPurchase) {
			player.bagItems = append([]bagItem(nil), staged.BagAfter...)
			player.silver = staged.CurrencyAfter.Silver
			player.gold = staged.CurrencyAfter.Gold
			player.silverCard = staged.CurrencyAfter.SilverCard
			player.silverTicket = staged.CurrencyAfter.SilverTicket
		},
	)
}
