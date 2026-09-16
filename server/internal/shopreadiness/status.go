// Package shopreadiness classifies read-only NPC shop audit results. It does
// not select wire messages, change currency, create items, or authorize buys.
package shopreadiness

// Status distinguishes an implemented NPC shop menu, an available exact
// catalog, and end-to-end purchase readiness. Catalog presence alone cannot
// establish that the current client purchase request has been verified.
type Status struct {
	MenuImplemented bool
	CatalogReady    bool
	PurchaseReady   bool
	Issue           string
}

// FromCatalogError evaluates only the result of attempting to load the exact
// shop ID; callers must not replace a missing ID with a similarly named one.
// The ordinary purchase handler remains explicitly blocked, even on success.
func FromCatalogError(catalogErr error) Status {
	if catalogErr != nil {
		return Status{MenuImplemented: true, Issue: catalogErr.Error()}
	}
	return Status{
		MenuImplemented: true,
		CatalogReady:    true,
		PurchaseReady:   false,
		Issue:           "shop catalog present; ordinary purchase blocked: exact-current purchase wire unverified",
	}
}
