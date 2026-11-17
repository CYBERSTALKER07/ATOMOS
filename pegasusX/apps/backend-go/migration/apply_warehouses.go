package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
)

func applyWarehouseRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	name := strings.TrimSpace(fields["name"])
	if name == "" {
		return fmt.Errorf("%w: warehouse name required at row %d", ErrInvalidInput, rowIdx)
	}
	extID := strings.TrimSpace(fields["external_warehouse_id"])
	mappedID := ""
	if extID != "" {
		if id, ok, err := tx.LookupIdentity(ctx, PackWarehouses, extID); err != nil {
			return err
		} else if ok {
			mappedID = id
		}
	}

	existing, found, err := tx.FindWarehouse(ctx, firstNonEmpty(mappedID, extID), name)
	if err != nil {
		return err
	}

	whID := existing.WarehouseID
	var before *StoredWarehouse
	if found {
		cp := existing
		before = &cp
	} else {
		whID = deterministicEntityID("wh", rctx.Session.SupplierID, string(PackWarehouses), firstNonEmpty(extID, name))
	}

	next := StoredWarehouse{
		WarehouseID: whID,
		SupplierID:  rctx.Session.SupplierID,
		Name:        name,
		Address:     strings.TrimSpace(fields["address"]),
		IsActive:    true,
		IsOnShift:   true,
	}

	if err := tx.SaveWarehouse(ctx, next, extID, before); err != nil {
		return err
	}
	if extID != "" {
		if err := tx.PutIdentity(ctx, PackWarehouses, extID, whID); err != nil {
			return err
		}
		_, _ = partner.UpsertPlantMappingMemory(partner.TenantSupplier, rctx.Session.SupplierID, extID, whID)
	}
	if parseBoolLike(fields["is_default"]) || rctx.DefaultWarehouseID == "" {
		rctx.DefaultWarehouseID = whID
	}
	if found {
		rctx.Result.Updated++
	} else {
		rctx.Result.Created++
	}
	return nil
}
