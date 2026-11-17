package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
)

func applyCatalogRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	name := strings.TrimSpace(fields["name"])
	sku := strings.TrimSpace(fields["sku"])
	barcode := strings.TrimSpace(fields["barcode"])
	if name == "" || (sku == "" && barcode == "") {
		return fmt.Errorf("%w: product requires name and sku or barcode at row %d", ErrInvalidInput, rowIdx)
	}

	extKey := firstNonEmpty(sku, barcode)
	mappedID, hasMapped, err := tx.LookupIdentity(ctx, PackProductsPrices, extKey)
	if err != nil {
		return err
	}

	existing, found, err := tx.FindProduct(ctx, mappedID, sku, barcode, name)
	if err != nil {
		return err
	}

	cur := strings.ToUpper(strings.TrimSpace(fields["currency"]))
	if cur != "" && rctx.SupplierCurrency != "" && cur != rctx.SupplierCurrency {
		return fmt.Errorf("%w: currency %s differs from supplier currency %s", ErrInvalidInput, cur, rctx.SupplierCurrency)
	}
	if cur == "" {
		cur = firstNonEmpty(existing.Currency, rctx.SupplierCurrency, "UZS")
	}

	unit := strings.TrimSpace(fields["unit"])
	if found && unit != "" && existing.Unit != "" && !strings.EqualFold(unit, existing.Unit) {
		onHand, err := tx.TotalProductOnHand(ctx, existing.ProductID)
		if err != nil {
			return err
		}
		if onHand > 0 {
			return fmt.Errorf("%w: cannot change unit %q -> %q for product %s while stock=%d > 0",
				ErrUnitChangeWithNonZeroStock, existing.Unit, unit, existing.ProductID, onHand)
		}
	}

	priceMinor := existing.PriceMinor
	if rawP := strings.TrimSpace(fields["price"]); rawP != "" {
		p, err := ParseMoneyMinor(rawP)
		if err != nil || p < 0 {
			return fmt.Errorf("%w: invalid price %q at row %d", ErrInvalidInput, rawP, rowIdx)
		}
		priceMinor = p
	}

	productID := existing.ProductID
	if !found {
		if hasMapped && mappedID != "" {
			productID = mappedID
		} else if len(extKey) <= 36 {
			productID = CleanIDToken(extKey)
		} else {
			productID = deterministicEntityID("prd", rctx.Session.SupplierID, string(PackProductsPrices), extKey)
		}
	}

	var existingCat *catalog.Product
	var before *StoredProduct
	if found {
		cp := existing
		before = &cp
		existingCat = &catalog.Product{
			ProductID:     existing.ProductID,
			SupplierID:    existing.SupplierID,
			CategoryID:    existing.CategoryID,
			Name:          existing.Name,
			Description:   existing.Description,
			PriceMinor:    existing.PriceMinor,
			Currency:      existing.Currency,
			Unit:          existing.Unit,
			Barcode:       existing.Barcode,
			HandlingClass: catalog.HandlingClass(existing.HandlingClass),
			IsPerishable:  existing.IsPerishable,
			IsActive:      existing.IsActive,
			Version:       existing.Version,
		}
	}

	activeTrue := true
	effectiveUnit := firstNonEmpty(unit, existing.Unit, "pcs")
	effectiveBarcode := firstNonEmpty(barcode, existing.Barcode)
	item := partner.ProductUpsertItem{
		ExternalID:  productID,
		Name:        name,
		CategoryID:  firstNonEmpty(strings.TrimSpace(fields["category"]), existing.CategoryID, "general"),
		PriceMinor:  priceMinor,
		Currency:    cur,
		Unit:        effectiveUnit,
		Barcode:     effectiveBarcode,
		Description: firstNonEmpty(strings.TrimSpace(fields["description"]), existing.Description),
		IsActive:    &activeTrue,
	}

	built, isUpdate, errCode := partner.ValidateAndBuildProduct(rctx.Session.SupplierID, item, existingCat, cur, tx.Now())
	if errCode != "" {
		return fmt.Errorf("%w: catalog upsert error %s at row %d", ErrInvalidInput, errCode, rowIdx)
	}

	requiresCold := existing.RequiresColdChain || strings.Contains(strings.ToLower(fields["category"]+fields["description"]), "cold")
	isPerish := existing.IsPerishable
	minShelf := existing.MinShelfLifeDays
	if rawDays := strings.TrimSpace(fields["shelf_life_days"]); rawDays != "" {
		if d, dErr := ParseQuantityInt(rawDays); dErr == nil && d > 0 {
			isPerish = true
			minShelf = d
		}
	}
	nextVer := built.Version
	if isUpdate {
		nextVer = existing.Version + 1
	}

	next := StoredProduct{
		ProductID:         built.ProductID,
		SupplierID:        built.SupplierID,
		CategoryID:        built.CategoryID,
		Name:              built.Name,
		Description:       built.Description,
		PriceMinor:        built.PriceMinor,
		Currency:          built.Currency,
		Unit:              built.Unit,
		Barcode:           built.Barcode,
		HandlingClass:     string(built.HandlingClass),
		IsPerishable:      isPerish,
		RequiresColdChain: requiresCold,
		MinShelfLifeDays:  minShelf,
		IsActive:          built.IsActive,
		Version:           nextVer,
	}

	if err := tx.SaveProduct(ctx, next, before); err != nil {
		return err
	}
	if err := tx.PutIdentity(ctx, PackProductsPrices, extKey, productID); err != nil {
		return err
	}
	if isUpdate {
		rctx.Result.Updated++
	} else {
		rctx.Result.Created++
	}
	return nil
}
