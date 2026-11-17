package warehouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (s *Service) loadWarehouseReturnsFromSpanner(ctx context.Context, warehouseID string) ([]portalReturnItem, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner unavailable")
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return []portalReturnItem{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT sr.ReturnId, sr.OrderId, COALESCE(p.Name, sr.SkuId),
		             COALESCE(sr.ExpectedQty, sr.RejectedQty), sr.PhysicalStatus, sr.CreatedAt
		      FROM SupplierReturns sr
		      LEFT JOIN Orders o ON sr.OrderId = o.OrderId
		      LEFT JOIN Products p ON p.ProductId = sr.SkuId AND p.SupplierId = o.SupplierId
		      WHERE sr.WarehouseId = @wh
		      ORDER BY sr.CreatedAt DESC
		      LIMIT 100`,
		Params: map[string]any{"wh": warehouseID},
	}
	iter := s.spannerClient.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	out := make([]portalReturnItem, 0, 16)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("warehouse returns query: %w", err)
		}
		var retID, ordID, prodName, status string
		var qty int64
		var createdAt time.Time
		if err := row.Columns(&retID, &ordID, &prodName, &qty, &status, &createdAt); err != nil {
			return nil, fmt.Errorf("warehouse returns scan: %w", err)
		}
		out = append(out, portalReturnItem{
			LineItemID:  retID,
			OrderID:     ordID,
			ProductName: strings.TrimSpace(prodName),
			Quantity:    int(qty),
			Status:      strings.TrimSpace(status),
			UpdatedAt:   createdAt.UTC().Format(time.RFC3339Nano),
		})
	}
}
