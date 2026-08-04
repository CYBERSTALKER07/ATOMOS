package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

// GetSupplierReturnPolicy implements ReturnPolicyStore.
func (r *SpannerRepository) GetSupplierReturnPolicy(ctx context.Context, supplierID string) (SupplierReturnPolicy, bool, error) {
	supplierID = strings.TrimSpace(supplierID)
	if r == nil || r.client == nil || supplierID == "" {
		return SupplierReturnPolicy{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "SupplierReturnPolicies", spanner.Key{supplierID}, []string{
		"SupplierId", "DefaultWindowHours", "ConcealedDamageWindowHours", "RequirePhoto", "AllowExpiredClaims", "UpdatedAt", "UpdatedByUserId",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return SupplierReturnPolicy{}, false, nil
		}
		return SupplierReturnPolicy{}, false, err
	}
	var (
		p          SupplierReturnPolicy
		concealed  spanner.NullInt64
		updatedBy  spanner.NullString
		requirePh  bool
		allowExp   bool
		updatedAt  time.Time
	)
	if err := row.Columns(&p.SupplierID, &p.DefaultWindowHours, &concealed, &requirePh, &allowExp, &updatedAt, &updatedBy); err != nil {
		return SupplierReturnPolicy{}, false, err
	}
	p.RequirePhoto = requirePh
	p.AllowExpiredClaims = allowExp
	p.UpdatedAt = updatedAt.UTC()
	p.UpdatedByUserID = updatedBy.StringVal
	if concealed.Valid {
		v := concealed.Int64
		p.ConcealedDamageWindowHours = &v
	}
	return p, true, nil
}

// UpsertSupplierReturnPolicy implements ReturnPolicyStore.
func (r *SpannerRepository) UpsertSupplierReturnPolicy(ctx context.Context, p SupplierReturnPolicy) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner return policy: nil client")
	}
	p.SupplierID = strings.TrimSpace(p.SupplierID)
	if p.SupplierID == "" {
		return fmt.Errorf("supplier_id required")
	}
	p.DefaultWindowHours = clampClaimWindowHours(p.DefaultWindowHours)
	m := map[string]any{
		"SupplierId":           p.SupplierID,
		"DefaultWindowHours":   p.DefaultWindowHours,
		"RequirePhoto":         p.RequirePhoto,
		"AllowExpiredClaims":   p.AllowExpiredClaims,
		"UpdatedAt":            spanner.CommitTimestamp,
		"UpdatedByUserId":      nullableString(p.UpdatedByUserID),
	}
	if p.ConcealedDamageWindowHours != nil {
		h := clampClaimWindowHours(*p.ConcealedDamageWindowHours)
		m["ConcealedDamageWindowHours"] = h
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("SupplierReturnPolicies", m),
	})
	return err
}

// GetWarehouseReturnPolicy implements ReturnPolicyStore.
func (r *SpannerRepository) GetWarehouseReturnPolicy(ctx context.Context, warehouseID, supplierID string) (WarehouseReturnPolicy, bool, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	supplierID = strings.TrimSpace(supplierID)
	if r == nil || r.client == nil || warehouseID == "" {
		return WarehouseReturnPolicy{}, false, nil
	}
	row, err := r.client.Single().ReadRow(ctx, "WarehouseReturnPolicies", spanner.Key{warehouseID, supplierID}, []string{
		"WarehouseId", "SupplierId", "ReverseDockSLAHours", "RetailerFileWindowHours", "CanOverrideRetailerWindow", "UpdatedAt", "UpdatedByUserId",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return WarehouseReturnPolicy{}, false, nil
		}
		return WarehouseReturnPolicy{}, false, err
	}
	var (
		p         WarehouseReturnPolicy
		sla       spanner.NullInt64
		fileHours spanner.NullInt64
		canOverride bool
		updatedAt time.Time
		updatedBy spanner.NullString
	)
	if err := row.Columns(&p.WarehouseID, &p.SupplierID, &sla, &fileHours, &canOverride, &updatedAt, &updatedBy); err != nil {
		return WarehouseReturnPolicy{}, false, err
	}
	p.CanOverrideRetailerWindow = canOverride
	p.UpdatedAt = updatedAt.UTC()
	p.UpdatedByUserID = updatedBy.StringVal
	if sla.Valid {
		v := sla.Int64
		p.ReverseDockSLAHours = &v
	}
	if fileHours.Valid {
		v := fileHours.Int64
		p.RetailerFileWindowHours = &v
	}
	return p, true, nil
}

// UpsertWarehouseReturnPolicy implements ReturnPolicyStore.
func (r *SpannerRepository) UpsertWarehouseReturnPolicy(ctx context.Context, p WarehouseReturnPolicy) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner return policy: nil client")
	}
	p.WarehouseID = strings.TrimSpace(p.WarehouseID)
	p.SupplierID = strings.TrimSpace(p.SupplierID)
	if p.WarehouseID == "" {
		return fmt.Errorf("warehouse_id required")
	}
	m := map[string]any{
		"WarehouseId":               p.WarehouseID,
		"SupplierId":                p.SupplierID,
		"CanOverrideRetailerWindow": p.CanOverrideRetailerWindow,
		"UpdatedAt":                 spanner.CommitTimestamp,
		"UpdatedByUserId":           nullableString(p.UpdatedByUserID),
	}
	if p.ReverseDockSLAHours != nil {
		m["ReverseDockSLAHours"] = *p.ReverseDockSLAHours
	}
	if p.RetailerFileWindowHours != nil {
		h := clampClaimWindowHours(*p.RetailerFileWindowHours)
		m["RetailerFileWindowHours"] = h
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("WarehouseReturnPolicies", m),
	})
	return err
}
