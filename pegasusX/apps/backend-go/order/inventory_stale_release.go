package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ReleaseStaleReservations cancels unpaid PENDING orders older than ttl through
// the canonical UpdateOrder path, which releases inventory reservations exactly
// once inside the same RW transaction and emits ORDER_STATUS_CHANGED.
//
// AWAITING_PAYMENT orders are intentionally excluded: goods are already at the
// retailer and settlement is owned by the payment webhook/reconciliation flow.
// A direct reservation decrement here (the previous implementation) was not
// idempotent — each sweep re-released the same rows, corrupting the shared
// QuantityReserved aggregate, and a later cancel released a second time.
func (s *Service) ReleaseStaleReservations(ctx context.Context, ttl time.Duration) (int, error) {
	if s == nil || s.spannerClient == nil || ttl <= 0 {
		return 0, nil
	}
	cutoff := s.now().Add(-ttl)
	stmt := spanner.Statement{
		SQL: `SELECT OrderId
		      FROM Orders
		      WHERE Status = 'PENDING'
		        AND UpdatedAt < @cutoff
		      LIMIT 100`,
		Params: map[string]any{"cutoff": cutoff},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var staleIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("scan stale orders: %w", err)
		}
		var orderID string
		if err := row.Columns(&orderID); err != nil {
			continue
		}
		staleIDs = append(staleIDs, orderID)
	}

	released := 0
	for _, orderID := range staleIDs {
		o, ok, err := s.repo.GetOrder(ctx, orderID)
		if err != nil || !ok {
			continue
		}
		if o.Status != StatusPending {
			continue
		}
		if strings.EqualFold(string(o.Source), string(OrderSourceBackorder)) || strings.TrimSpace(o.WarehouseID) == "" {
			continue
		}
		if _, err := s.cancelOrderWithReason(ctx, &o, "system", "SYSTEM", "stale_unpaid_reservation_release", ""); err != nil {
			s.log.Warn("stale reservation release failed", "order_id", orderID, "err", err)
			continue
		}
		released++
	}
	return released, nil
}
