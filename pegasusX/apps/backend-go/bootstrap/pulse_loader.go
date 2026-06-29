package bootstrap

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pulse"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

type supplierPulseLoader struct {
	svc *supplier.Service
}

func (l *supplierPulseLoader) ListRecentSupplierOrders(ctx context.Context, supplierID string, limit int) ([]pulse.SupplierActivityOrder, error) {
	if l == nil || l.svc == nil {
		return nil, nil
	}
	orders, err := l.svc.ListOrdersForPulse(ctx, supplierID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]pulse.SupplierActivityOrder, 0, len(orders))
	for _, o := range orders {
		updated := time.Now().UTC()
		if parsed, err := time.Parse(time.RFC3339Nano, o.UpdatedAt); err == nil {
			updated = parsed.UTC()
		} else if parsed, err := time.Parse(time.RFC3339, o.UpdatedAt); err == nil {
			updated = parsed.UTC()
		}
		out = append(out, pulse.SupplierActivityOrder{
			OrderID:    o.OrderID,
			ManifestID: o.ManifestID,
			Status:     o.Status,
			UpdatedAt:  updated,
		})
	}
	return out, nil
}
