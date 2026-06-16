package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ReleaseStaleReservations releases inventory holds for unpaid orders older than ttl.
func (s *Service) ReleaseStaleReservations(ctx context.Context, ttl time.Duration) (int, error) {
	if s == nil || s.spannerClient == nil || ttl <= 0 {
		return 0, nil
	}
	cutoff := s.now().Add(-ttl)
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId, WarehouseId, LineItemsJson, Status, OrderSource
		      FROM Orders
		      WHERE Status IN ('PENDING','AWAITING_PAYMENT')
		        AND UpdatedAt < @cutoff
		      LIMIT 100`,
		Params: map[string]any{"cutoff": cutoff},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	released := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return released, fmt.Errorf("scan stale orders: %w", err)
		}
		var orderID, supplierID, retailerID, warehouseID, lineItemsRaw, status, source string
		if err := row.Columns(&orderID, &supplierID, &retailerID, &warehouseID, &lineItemsRaw, &status, &source); err != nil {
			continue
		}
		if strings.EqualFold(source, string(OrderSourceBackorder)) || warehouseID == "" {
			continue
		}
		o, ok, err := s.repo.GetOrder(ctx, orderID)
		if err != nil || !ok {
			continue
		}
		if err := s.releaseOrderReservations(ctx, &o); err != nil {
			s.log.Warn("stale reservation release failed", "order_id", orderID, "err", err)
			continue
		}
		released++
	}
	return released, nil
}
