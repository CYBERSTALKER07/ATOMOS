package globalproducts

import "context"

// CatalogHook adapts Service to catalog.ProductUpsertHook.
type CatalogHook struct {
	Svc *Service
}

func (h CatalogHook) OnProductUpserted(ctx context.Context, productID, supplierID, name, barcode string, priceMinor int64, currency string, unitsPerPack int64) error {
	if h.Svc == nil {
		return nil
	}
	return h.Svc.OnProductUpserted(ctx, ProductInput{
		ProductID:    productID,
		SupplierID:   supplierID,
		Name:         name,
		Barcode:      barcode,
		PriceMinor:   priceMinor,
		Currency:     currency,
		UnitsPerPack: unitsPerPack,
	})
}
