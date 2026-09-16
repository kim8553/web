package shopreadiness

import (
	"errors"
	"strings"
	"testing"
)

func TestCatalogPresentDoesNotAuthorizeOrdinaryPurchase(t *testing.T) {
	status := FromCatalogError(nil)
	if !status.MenuImplemented || !status.CatalogReady || status.PurchaseReady {
		t.Fatalf("catalog was mistaken for purchase readiness: %+v", status)
	}
	if !strings.Contains(status.Issue, "purchase blocked") || !strings.Contains(status.Issue, "wire unverified") {
		t.Fatalf("missing exact-current purchase boundary: %+v", status)
	}
}

func TestMissingExactCatalogDoesNotAuthorizeOrdinaryPurchase(t *testing.T) {
	catalogErr := errors.New("shop.ini has no section Shop_exact")
	status := FromCatalogError(catalogErr)
	if !status.MenuImplemented || status.CatalogReady || status.PurchaseReady {
		t.Fatalf("missing catalog was mistaken for purchase readiness: %+v", status)
	}
	if status.Issue != catalogErr.Error() {
		t.Fatalf("exact catalog failure was concealed: %+v", status)
	}
}
