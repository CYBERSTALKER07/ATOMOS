package bootstrap

import (
	"context"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// orderClaimsLookup adapts order.Service to claims.OrderLookup without cycles.
type orderClaimsLookup struct {
	svc *order.Service
}

func (l orderClaimsLookup) GetOrder(ctx context.Context, orderID string) (claims.OrderSnapshot, bool, error) {
	if l.svc == nil {
		return claims.OrderSnapshot{}, false, nil
	}
	o, ok, err := l.svc.GetOrder(ctx, orderID)
	if err != nil || !ok {
		return claims.OrderSnapshot{}, ok, err
	}
	lines := make([]claims.OrderLine, 0, len(o.LineItems))
	for _, li := range o.LineItems {
		lines = append(lines, claims.OrderLine{
			SKU:            li.SKU,
			Quantity:       li.Quantity,
			UnitPriceMinor: li.UnitPrice,
		})
	}
	return claims.OrderSnapshot{
		OrderID:            o.OrderID,
		SupplierID:         o.SupplierID,
		RetailerID:         o.RetailerID,
		WarehouseID:        o.WarehouseID,
		Currency:           o.Currency,
		Status:             string(o.Status),
		TotalMinor:         o.TotalMinor,
		OriginalTotalMinor: o.OriginalTotalMinor,
		LineItems:          lines,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}, true, nil
}

// driverClaimsBridge opens Claims rows from driver OS&D exception reports.
type driverClaimsBridge struct {
	svc *claims.Service
}

func (b *driverClaimsBridge) OnDriverException(ctx context.Context, o order.Order, driverID string, items []order.ExceptionReportItem, photos []string, note string) error {
	if b == nil || b.svc == nil {
		return nil
	}
	lines := make([]claims.ClaimLine, 0, len(items))
	claimType := claims.ClaimTypeMissing
	for _, it := range items {
		reason := strings.ToUpper(strings.TrimSpace(it.Reason))
		if reason == "" {
			reason = "MISSING"
		}
		if reason == "DAMAGED" || reason == "WRONG_ITEM" {
			claimType = claims.ClaimTypeDamaged
		}
		lines = append(lines, claims.ClaimLine{
			SKU:      strings.TrimSpace(it.SKU),
			Quantity: it.Quantity,
			Reason:   reason,
		})
	}
	orderLines := make([]claims.OrderLine, 0, len(o.LineItems))
	for _, li := range o.LineItems {
		orderLines = append(orderLines, claims.OrderLine{
			SKU:            li.SKU,
			Quantity:       li.Quantity,
			UnitPriceMinor: li.UnitPrice,
		})
	}
	snap := claims.OrderSnapshot{
		OrderID:            o.OrderID,
		SupplierID:         o.SupplierID,
		RetailerID:         o.RetailerID,
		WarehouseID:        o.WarehouseID,
		Currency:           o.Currency,
		Status:             string(o.Status),
		TotalMinor:         o.TotalMinor,
		OriginalTotalMinor: o.OriginalTotalMinor,
		LineItems:          orderLines,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}
	_, err := b.svc.CreateFromDriverException(ctx, snap, driverID, claimType, lines, photos, note)
	return err
}
