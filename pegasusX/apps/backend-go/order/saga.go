package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

const (
	SagaStatePending      = "PENDING"
	SagaStateReserving    = "RESERVING"
	SagaStateCommitted    = "COMMITTED"
	SagaStateCompensating = "COMPENSATING"
	SagaStateCompensated  = "COMPENSATED"
	SagaStateFailed       = "FAILED"

	SagaLeaseDuration = 45 * time.Second
)

// RecordSagaChildCreated updates ParentOrders with the newly created child order ID
// and extends the saga lease, ensuring that even if the coordinator crashes, the child order
// is durable and known to the recovery worker.
func (s *Service) RecordSagaChildCreated(ctx context.Context, parentID, childID string) error {
	if s.spannerClient == nil {
		return nil
	}
	parentID = strings.TrimSpace(parentID)
	childID = strings.TrimSpace(childID)
	if parentID == "" || childID == "" {
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "ParentOrders", spanner.Key{parentID}, []string{"CreatedChildOrderIds"})
		if err != nil {
			return err
		}
		var existingIDs []string
		_ = row.Columns(&existingIDs)
		existingIDs = append(existingIDs, childID)

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ParentOrders", map[string]any{
				"ParentOrderId":        parentID,
				"SagaState":            SagaStateReserving,
				"CreatedChildOrderIds": existingIDs,
				"LeaseExpiresAt":       time.Now().UTC().Add(SagaLeaseDuration),
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// CompleteSaga transitions ParentOrders to COMMITTED state and finalizes totals.
func (s *Service) CompleteSaga(ctx context.Context, parentID, currency string, totalMinor int64, childCount int) error {
	if s.spannerClient == nil {
		return nil
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return fmt.Errorf("parent order requires parent_id")
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ParentOrders", map[string]any{
				"ParentOrderId":  parentID,
				"Status":         parentStatusPending,
				"SagaState":      SagaStateCommitted,
				"Currency":       currency,
				"TotalMinor":     totalMinor,
				"ChildCount":     int64(childCount),
				"LeaseExpiresAt": nil,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		retailerID := ""
		if row, rerr := txn.ReadRow(ctx, "ParentOrders", spanner.Key{parentID}, []string{"RetailerId"}); rerr == nil {
			_ = row.Columns(&retailerID)
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateParentOrder, parentID, events.TopicMain, events.ParentOrderEvent{
			BaseEvent:     events.BaseEvent{Type: events.EventParentOrderUpdated, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			ParentOrderID: parentID,
			RetailerID:    strings.TrimSpace(retailerID),
			Status:        parentStatusPending,
			Currency:      currency,
			TotalMinor:    totalMinor,
			ChildCount:    childCount,
			SagaState:     SagaStateCommitted,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})
	return err
}

// CompensateSaga transitions ParentOrders to COMPENSATING, cancels all created child orders,
// releases reservations, and marks the saga COMPENSATED.
func (s *Service) CompensateSaga(ctx context.Context, parentID string, created []Order, reason string) error {
	if s.spannerClient == nil {
		s.compensateParentCheckout(ctx, parentID, created)
		return nil
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}

	// 1. Mark Saga COMPENSATING in Spanner with lease extension
	var retailerID string
	_, _ = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if row, err := txn.ReadRow(ctx, "ParentOrders", spanner.Key{parentID}, []string{"RetailerId"}); err == nil {
			_ = row.Columns(&retailerID)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ParentOrders", map[string]any{
				"ParentOrderId":  parentID,
				"Status":         parentStatusCancelled,
				"SagaState":      SagaStateCompensating,
				"LeaseExpiresAt": time.Now().UTC().Add(SagaLeaseDuration),
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		})
	})

	// 2. Discover child orders from Spanner if created list is empty or partial
	childOrderIDs := make(map[string]struct{})
	for _, o := range created {
		if strings.TrimSpace(o.OrderID) != "" {
			childOrderIDs[o.OrderID] = struct{}{}
		}
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId FROM Orders WHERE ParentOrderId = @parentID`,
		Params: map[string]any{
			"parentID": parentID,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var oid string
		if err := row.Columns(&oid); err == nil && oid != "" {
			childOrderIDs[oid] = struct{}{}
		}
	}
	iter.Stop()

	// 3. Compensate each child order
	for oid := range childOrderIDs {
		orderToCancel := Order{OrderID: oid}
		if s.repo != nil {
			if full, ok, err := s.repo.GetOrder(ctx, oid); err == nil && ok {
				orderToCancel = full
			}
		}
		if _, err := s.cancelOrderWithReason(ctx, &orderToCancel, "system", "SYSTEM", "multi_supplier_checkout_abort", reason); err != nil {
			s.log.Warn("saga compensate cancel child failed", "order_id", oid, "parent_order_id", parentID, "err", err)
		}
	}

	// 4. Mark Saga COMPENSATED
	_, _ = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ParentOrders", map[string]any{
				"ParentOrderId":  parentID,
				"Status":         parentStatusCancelled,
				"SagaState":      SagaStateCompensated,
				"LeaseExpiresAt": nil,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregateParentOrder, parentID, events.TopicMain, events.ParentOrderEvent{
			BaseEvent:     events.BaseEvent{Type: events.EventParentOrderUpdated, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
			ParentOrderID: parentID,
			RetailerID:    strings.TrimSpace(retailerID),
			Status:        parentStatusCancelled,
			SagaState:     SagaStateCompensated,
		}); err != nil {
			return err
		}
		return buf.Flush(ctx)
	})

	return nil
}

// StalledSagaRecord describes a parent order whose lease expired mid-flight.
type StalledSagaRecord struct {
	ParentOrderID string
	RetailerID    string
	SagaState     string
	Status        string
}

// SweepStalledSagas finds ParentOrders where the checkout lease has expired and compensates them.
func (s *Service) SweepStalledSagas(ctx context.Context) ([]StalledSagaRecord, error) {
	if s.spannerClient == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT ParentOrderId, RetailerId, SagaState, Status 
		      FROM ParentOrders 
		      WHERE SagaState IN ('PENDING', 'RESERVING', 'COMPENSATING') 
		        AND LeaseExpiresAt < CURRENT_TIMESTAMP()`,
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var stalled []StalledSagaRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return stalled, fmt.Errorf("sweep stalled sagas query: %w", err)
		}
		var rec StalledSagaRecord
		if err := row.Columns(&rec.ParentOrderID, &rec.RetailerID, &rec.SagaState, &rec.Status); err != nil {
			continue
		}
		stalled = append(stalled, rec)
	}

	for _, rec := range stalled {
		s.log.Warn("sweeping stalled checkout saga", "parent_order_id", rec.ParentOrderID, "saga_state", rec.SagaState)
		if err := s.CompensateSaga(ctx, rec.ParentOrderID, nil, "coordinator_lease_expired_recovery"); err != nil {
			s.log.Error("failed to compensate stalled saga", "parent_order_id", rec.ParentOrderID, "err", err)
		}
	}
	return stalled, nil
}

// StartSagaRecoveryWorker periodically runs SweepStalledSagas in the background.
func StartSagaRecoveryWorker(ctx context.Context, s *Service, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SweepStalledSagas(ctx); err != nil {
				s.log.Warn("saga recovery worker sweep failed", "err", err)
			}
		}
	}
}
