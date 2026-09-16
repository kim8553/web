package main

// stage32ShopMenuDiagnostic is a read-only snapshot of the exact shop.ini
// section referenced by the selected NPC. It does not resolve missing IDs to
// similarly named sections or authorize purchases or client rendering.
type stage32ShopMenuDiagnostic struct {
	CatalogError string
	AuthoredRows int
	OrdinaryRows int
	ExchangeRows int
	ExchangeData int
	OtherRows    int
}

func stage32InspectShopMenu(path, shopID string) stage32ShopMenuDiagnostic {
	items, _, _, err := loadShopCatalogSection(path, shopID)
	if err != nil {
		return stage32ShopMenuDiagnostic{CatalogError: err.Error()}
	}
	out := stage32ShopMenuDiagnostic{AuthoredRows: len(items)}
	for _, item := range items {
		switch item.priceMode {
		case 0, 1, 2:
			out.OrdinaryRows++
		case 3:
			out.ExchangeRows++
			if item.exchangeData > 0 {
				out.ExchangeData++
			}
		default:
			out.OtherRows++
		}
	}
	return out
}
