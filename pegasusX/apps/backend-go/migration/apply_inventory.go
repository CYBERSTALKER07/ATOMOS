package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func resolveApplyProductAndWarehouse(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) (StoredProduct, StoredWarehouse, error) {
	sku := strings.TrimSpace(fields["sku"])
	if sku == "" {
		return StoredProduct{}, StoredWarehouse{}, fmt.Errorf("%w: sku required at row %d", ErrInvalidInput, rowIdx)
	}
	mappedProdID, _, err := tx.LookupIdentity(ctx, PackProductsPrices, sku)
	if err != nil {
		return StoredProduct{}, StoredWarehouse{}, err
	}
	prod, ok, err := tx.FindProduct(ctx, mappedProdID, sku, sku, sku)
	if err != nil {
		return StoredProduct{}, StoredWarehouse{}, err
	}
	if !ok {
		return StoredProduct{}, StoredWarehouse{}, fmt.Errorf("%w: unresolved sku %q at row %d", ErrInvalidInput, sku, rowIdx)
	}

	whRef := strings.TrimSpace(fields["warehouse_id"])
	mappedWHID := ""
	if whRef != "" {
		if id, ok, err := tx.LookupIdentity(ctx, PackWarehouses, whRef); err != nil {
			return StoredProduct{}, StoredWarehouse{}, err
		} else if ok {
			mappedWHID = id
		}
	} else {
		mappedWHID = rctx.DefaultWarehouseID
	}

	wh, whOK, err := tx.FindWarehouse(ctx, firstNonEmpty(mappedWHID, whRef), whRef)
	if err != nil {
		return StoredProduct{}, StoredWarehouse{}, err
	}
	if !whOK {
		return StoredProduct{}, StoredWarehouse{}, fmt.Errorf("%w: unresolved warehouse %q at row %d", ErrInvalidInput, whRef, rowIdx)
	}
	return prod, wh, nil
}

func applyStockRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	prod, wh, err := resolveApplyProductAndWarehouse(ctx, tx, rctx, rowIdx, fields)
	if err != nil {
		return err
	}
	if rctx.warehouseUsesLots(ctx, wh.WarehouseID) {
		return fmt.Errorf("%w: use lots pack when WMS_LOTS_ENABLED is true for warehouse %s", inventory.ErrLotsEnabledDirectV2, wh.WarehouseID)
	}

	qty, err := ParseQuantityInt(fields["quantity_on_hand"])
	if err != nil || qty < 0 {
		return fmt.Errorf("%w: invalid quantity_on_hand %q at row %d", ErrInvalidInput, fields["quantity_on_hand"], rowIdx)
	}

	existing, found, err := tx.GetInventory(ctx, wh.WarehouseID, prod.ProductID)
	if err != nil {
		return err
	}
	if unit := strings.TrimSpace(fields["unit"]); unit != "" && prod.Unit != "" && !strings.EqualFold(unit, prod.Unit) && existing.QuantityOnHand > 0 {
		return fmt.Errorf("%w: unit %q conflicts with catalog unit %q while QuantityOnHand=%d > 0",
			ErrUnitChangeWithNonZeroStock, unit, prod.Unit, existing.QuantityOnHand)
	}

	var before *StoredStock
	if found {
		cp := existing
		before = &cp
	}

	targetOnHand := existing.QuantityOnHand + qty
	if rctx.ApplyMode == ApplyModeSetOnHand {
		targetOnHand = qty
	}
	if targetOnHand < existing.QuantityReserved {
		return fmt.Errorf("%w: target on_hand %d < live QuantityReserved %d for product %s at warehouse %s",
			ErrOnHandBelowReserved, targetOnHand, existing.QuantityReserved, prod.ProductID, wh.WarehouseID)
	}

	var existingLevel *inventory.Level
	if found {
		existingLevel = &inventory.Level{
			InventoryID:      wh.WarehouseID + "|" + prod.ProductID,
			WarehouseID:      wh.WarehouseID,
			ProductID:        prod.ProductID,
			SupplierID:       rctx.Session.SupplierID,
			QuantityOnHand:   existing.QuantityOnHand,
			QuantityReserved: existing.QuantityReserved,
			ReorderThreshold: existing.ReorderThreshold,
			Version:          1,
		}
	}
	level, isUpdate, errCode := partner.ValidateAndBuildStockLevel(rctx.Session.SupplierID, partner.StockUpsertItem{
		ExternalID:     prod.ProductID,
		WarehouseID:    wh.WarehouseID,
		QuantityOnHand: targetOnHand,
	}, existingLevel, tx.Now())
	if errCode != "" {
		return fmt.Errorf("%w: stock upsert error %s at row %d", ErrInvalidInput, errCode, rowIdx)
	}

	next := StoredStock{
		SupplierID:       rctx.Session.SupplierID,
		WarehouseID:      wh.WarehouseID,
		ProductID:        prod.ProductID,
		QuantityOnHand:   level.QuantityOnHand,
		QuantityReserved: level.QuantityReserved,
		ReorderThreshold: level.ReorderThreshold,
	}
	if err := tx.UpsertStock(ctx, next, before); err != nil {
		return err
	}
	if isUpdate {
		rctx.Result.Updated++
	} else {
		rctx.Result.Created++
	}
	return nil
}

func applyLotRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	if rctx.ApplyMode == ApplyModeSetOnHand {
		return fmt.Errorf("%w: SET_ON_HAND forbidden for lots pack", ErrInvalidInput)
	}
	prod, wh, err := resolveApplyProductAndWarehouse(ctx, tx, rctx, rowIdx, fields)
	if err != nil {
		return err
	}
	if !rctx.warehouseUsesLots(ctx, wh.WarehouseID) {
		return fmt.Errorf("%w: WMS_LOTS_ENABLED must be active to apply lots pack for warehouse %s", ErrInvalidInput, wh.WarehouseID)
	}

	lotCode := strings.TrimSpace(fields["lot_number"])
	if lotCode == "" {
		return fmt.Errorf("%w: lot_number required at row %d", ErrInvalidInput, rowIdx)
	}
	exp, err := ParseFlexibleDate(fields["expiry_date"])
	if err != nil {
		return fmt.Errorf("%w: invalid expiry_date %q at row %d", ErrInvalidInput, fields["expiry_date"], rowIdx)
	}
	mfgTimePtr := parseOptionalDatePtr(fields["production_date"])
	if mfgTimePtr != nil && mfgTimePtr.After(exp) {
		return fmt.Errorf("%w: production_date cannot be after expiry_date at row %d", ErrInvalidInput, rowIdx)
	}

	qty, err := ParseQuantityInt(fields["quantity_on_hand"])
	if err != nil || qty < 0 {
		return fmt.Errorf("%w: invalid lot quantity %q at row %d", ErrInvalidInput, fields["quantity_on_hand"], rowIdx)
	}

	if prod.RequiresColdChain {
		rctx.addWarning(rowIdx, "cold_lot_migrated_without_live_baseline",
			fmt.Sprintf("cold-chain lot %s (%s) put away without physical seal baseline check", lotCode, prod.ProductID))
	}

	existingLot, hasLot, err := tx.FindLot(ctx, wh.WarehouseID, prod.ProductID, lotCode)
	if err != nil {
		return err
	}
	var lotPtr *StoredLot
	if hasLot {
		cp := existingLot
		lotPtr = &cp
		if qty < existingLot.QuantityReserved {
			return fmt.Errorf("%w: lot %s target quantity %d < live QuantityReserved %d",
				ErrOnHandBelowReserved, lotCode, qty, existingLot.QuantityReserved)
		}
	}

	existingStk, hasStk, err := tx.GetInventory(ctx, wh.WarehouseID, prod.ProductID)
	if err != nil {
		return err
	}
	var stkPtr *StoredStock
	if hasStk {
		cp := existingStk
		stkPtr = &cp
	}

	locID := firstNonEmpty(strings.TrimSpace(fields["location_id"]), stocklots.DefaultRecvLocationID)
	op := LotMigrationOp{
		WarehouseID:      wh.WarehouseID,
		ProductID:        prod.ProductID,
		LocationID:       locID,
		LotCode:          lotCode,
		TargetQuantity:   qty,
		ExpiryDate:       &exp,
		ManufacturedDate: mfgTimePtr,
		NewLotID:         deterministicEntityID("lot", rctx.Session.SupplierID, string(PackLots), wh.WarehouseID+"|"+prod.ProductID+"|"+lotCode),
	}

	res, err := tx.PutawayOrAdjustLot(ctx, op, lotPtr, stkPtr)
	if err != nil {
		return err
	}
	switch res.Action {
	case "created":
		rctx.Result.Created++
	case "updated":
		rctx.Result.Updated++
	default:
		rctx.Result.Skipped++
	}
	return nil
}

func parseOptionalDatePtr(raw string) *time.Time {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	t, err := ParseFlexibleDate(s)
	if err != nil {
		return nil
	}
	return &t
}
