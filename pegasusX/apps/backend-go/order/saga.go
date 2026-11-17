package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
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

var (
	testRecordSagaChildCreatedHook            func(ctx context.Context, parentID, childID string) error
	testCompleteSagaHook                      func(ctx context.Context, parentID, currency string, totalMinor int64, childCount int) error
	testSweepStalledSagasHook                 func(ctx context.Context) ([]StalledSagaRecord, error)
	testReclaimPendingCancelCompensationsHook func(ctx context.Context) (int, error)
)

// RecordSagaChildCreated updates ParentOrders with the newly created child order ID
// and extends the saga lease, ensuring that even if the coordinator crashes, the child order
// is durable and known to the recovery worker.
func (s *Service) RecordSagaChildCreated(ctx context.Context, parentID, childID string) error {
	s.setParentSagaState(parentID, SagaStateReserving)
	if testRecordSagaChildCreatedHook != nil {
		if err := testRecordSagaChildCreatedHook(ctx, parentID, childID); err != nil {
			return err
		}
	}
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
		if !slices.Contains(existingIDs, childID) {
			existingIDs = append(existingIDs, childID)
		}

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
	s.setParentSagaState(parentID, SagaStateCommitted)
	if testCompleteSagaHook != nil {
		if err := testCompleteSagaHook(ctx, parentID, currency, totalMinor, childCount); err != nil {
			return err
		}
	}
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
	s.setParentSagaState(parentID, SagaStateCompensated)
	if s.spannerClient == nil {
		var cancelErrors []string
		for i := range created {
			o := created[i]
			if s.repo != nil {
				if full, ok, err := s.repo.GetOrder(ctx, o.OrderID); err == nil && ok {
					o = full
				}
			}
			if _, err := s.cancelOrderWithReason(ctx, &o, "system", "SYSTEM", "multi_supplier_checkout_abort", reason); err != nil {
				s.log.Warn("compensate cancel failed", "order_id", o.OrderID, "parent_order_id", parentID, "err", err)
				cancelErrors = append(cancelErrors, fmt.Sprintf("child %s: %v", o.OrderID, err))
			}
			s.rollbackOrderDailyUsageMemory(o.OrderID)
		}
		if len(cancelErrors) > 0 {
			return fmt.Errorf("compensate child orders failed (%d errors): %s", len(cancelErrors), strings.Join(cancelErrors, "; "))
		}
		_ = s.updateParentOrderTotals(ctx, parentID, parentStatusCancelled, s.currency, 0, len(created))
		return nil
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}

	// 1. Mark Saga COMPENSATING in Spanner with lease extension
	var retailerID string
	if _, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if row, err := txn.ReadRow(ctx, "ParentOrders", spanner.Key{parentID}, []string{"RetailerId"}); err == nil {
			_ = row.Columns(&retailerID)
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
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
	}); err != nil {
		s.log.Warn("saga compensate: failed to mark saga COMPENSATING", "parent_id", parentID, "err", err)
	}

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
	var cancelErrors []string
	for oid := range childOrderIDs {
		orderToCancel := Order{OrderID: oid}
		if s.repo != nil {
			if full, ok, err := s.repo.GetOrder(ctx, oid); err == nil && ok {
				orderToCancel = full
			}
		}
		if _, err := s.cancelOrderWithReason(ctx, &orderToCancel, "system", "SYSTEM", "multi_supplier_checkout_abort", reason); err != nil {
			s.log.Warn("saga compensate cancel child failed", "order_id", oid, "parent_order_id", parentID, "err", err)
			cancelErrors = append(cancelErrors, fmt.Sprintf("child %s: %v", oid, err))
		}
	}

	if len(cancelErrors) > 0 {
		return fmt.Errorf("compensate child orders failed (%d errors): %s", len(cancelErrors), strings.Join(cancelErrors, "; "))
	}

	// 4. Roll back RetailerSellerDailyUsage deltas and mark Saga COMPENSATED atomically.
	allChildIDs := make([]string, 0, len(childOrderIDs))
	for oid := range childOrderIDs {
		allChildIDs = append(allChildIDs, oid)
	}
	slices.Sort(allChildIDs)
	if _, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := rollbackOrdersUsageInTxn(ctx, txn, allChildIDs, strings.TrimSpace(retailerID)); err != nil {
			return fmt.Errorf("rollback seller daily usage: %w", err)
		}
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
	}); err != nil {
		s.log.Error("saga compensate: failed to mark saga COMPENSATED", "parent_id", parentID, "err", err)
		return fmt.Errorf("mark saga compensated: %w", err)
	}

	return nil
}

// StalledSagaRecord describes a parent order whose lease expired mid-flight.
type StalledSagaRecord struct {
	ParentOrderID string
	RetailerID    string
	SagaState     string
	Status        string
}

// EventOrderCancelReservationReleaseRequired indicates that an order cancel requires credit reservation release.
const EventOrderCancelReservationReleaseRequired = "ORDER_CANCEL_RESERVATION_RELEASE_REQUIRED"

// SweepStalledSagas finds ParentOrders where the checkout lease has expired and compensates them.
func (s *Service) SweepStalledSagas(ctx context.Context) ([]StalledSagaRecord, error) {
	if testSweepStalledSagasHook != nil {
		return testSweepStalledSagasHook(ctx)
	}
	if s.spannerClient == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT ParentOrderId, RetailerId, SagaState, Status 
		      FROM ParentOrders 
		      WHERE SagaState IN ('PENDING', 'RESERVING', 'COMPENSATING') 
		        AND LeaseExpiresAt < CURRENT_TIMESTAMP()
		      LIMIT 200`,
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var stalled []StalledSagaRecord
	for {
		if ctx.Err() != nil {
			return stalled, ctx.Err()
		}
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
		if ctx.Err() != nil {
			return stalled, ctx.Err()
		}
		s.log.Warn("sweeping stalled checkout saga", "parent_order_id", rec.ParentOrderID, "saga_state", rec.SagaState)
		if err := s.CompensateSaga(ctx, rec.ParentOrderID, nil, "coordinator_lease_expired_recovery"); err != nil {
			s.log.Error("failed to compensate stalled saga", "parent_order_id", rec.ParentOrderID, "err", err)
		}
	}
	return stalled, nil
}

// ReclaimPendingCancelCompensations retries credit reservation releases for cancelled orders
// that failed credit release (tracked in OutboxEvents or directly in OrderCreditReservations).
func (s *Service) ReclaimPendingCancelCompensations(ctx context.Context) (int, error) {
	if testReclaimPendingCancelCompensationsHook != nil {
		return testReclaimPendingCancelCompensationsHook(ctx)
	}
	if s.credit == nil || s.spannerClient == nil {
		return 0, nil
	}

	reclaimed := 0
	type pendingComp struct {
		orderID    string
		retailerID string
		supplierID string
		eventIDs   []string
	}
	pendingByOrder := make(map[string]*pendingComp)

	// 1. Query unpublished OutboxEvents for order cancel compensation events
	stmtOutbox := spanner.Statement{
		SQL: `SELECT EventId, AggregateId, SupplierId, Payload
		      FROM OutboxEvents
		      WHERE PublishedAt IS NULL
		        AND AggregateType IN ('order', 'Order')
		      LIMIT 500`,
	}
	iterOutbox := s.spannerClient.Single().Query(ctx, stmtOutbox)
	for {
		if ctx.Err() != nil {
			iterOutbox.Stop()
			return reclaimed, ctx.Err()
		}
		row, err := iterOutbox.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iterOutbox.Stop()
			return reclaimed, fmt.Errorf("query outbox cancel compensations: %w", err)
		}
		var eventID, aggID, supplierID string
		var rawPayload []byte
		if err := row.Columns(&eventID, &aggID, &supplierID, &rawPayload); err != nil {
			continue
		}

		var payload struct {
			OrderID    string `json:"order_id"`
			SupplierID string `json:"supplier_id"`
			RetailerID string `json:"retailer_id"`
			EventType  string `json:"event_type"`
			Reason     string `json:"reason"`
			Error      string `json:"error"`
		}
		_ = json.Unmarshal(rawPayload, &payload)

		oid := strings.TrimSpace(payload.OrderID)
		if oid == "" {
			oid = strings.TrimSpace(aggID)
		}

		isCancelComp := payload.Error != "" || payload.Reason != "" ||
			payload.EventType == EventOrderCancelReservationReleaseRequired ||
			strings.Contains(string(rawPayload), EventOrderCancelReservationReleaseRequired) ||
			strings.Contains(string(rawPayload), "credit")

		if oid != "" && isCancelComp {
			p, ok := pendingByOrder[oid]
			if !ok {
				p = &pendingComp{
					orderID:    oid,
					retailerID: strings.TrimSpace(payload.RetailerID),
					supplierID: strings.TrimSpace(payload.SupplierID),
				}
				if p.supplierID == "" {
					p.supplierID = supplierID
				}
				pendingByOrder[oid] = p
			}
			p.eventIDs = append(p.eventIDs, eventID)
		}
	}
	iterOutbox.Stop()

	// 2. Query cancelled orders that still have Status = 'RESERVED' in OrderCreditReservations
	stmtReservations := spanner.Statement{
		SQL: `SELECT c.OrderId, c.RetailerId, c.SupplierId
		      FROM OrderCreditReservations c
		      JOIN Orders o ON c.OrderId = o.OrderId
		      WHERE o.Status = 'CANCELLED' AND c.Status = 'RESERVED'
		      LIMIT 500`,
	}
	iterRes := s.spannerClient.Single().Query(ctx, stmtReservations)
	for {
		if ctx.Err() != nil {
			iterRes.Stop()
			return reclaimed, ctx.Err()
		}
		row, err := iterRes.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			iterRes.Stop()
			return reclaimed, fmt.Errorf("query cancelled order reservations: %w", err)
		}
		var oid, retID, supID string
		if err := row.Columns(&oid, &retID, &supID); err != nil {
			continue
		}
		oid = strings.TrimSpace(oid)
		if oid != "" {
			p, ok := pendingByOrder[oid]
			if !ok {
				p = &pendingComp{
					orderID:    oid,
					retailerID: retID,
					supplierID: supID,
				}
				pendingByOrder[oid] = p
			}
		}
	}
	iterRes.Stop()

	// 3. Retry ReleaseReserve for each pending compensation
	for oid, comp := range pendingByOrder {
		if ctx.Err() != nil {
			return reclaimed, ctx.Err()
		}

		if err := s.credit.ReleaseReserve(ctx, oid); err != nil {
			s.log.Warn("reclaim cancel compensation failed to release credit reserve",
				"order_id", oid,
				"retailer_id", comp.retailerID,
				"supplier_id", comp.supplierID,
				"err", err,
			)
			continue
		}

		// On successful release, mark corresponding OutboxEvents rows published
		if len(comp.eventIDs) > 0 {
			eventIDs := comp.eventIDs
			_, pubErr := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				muts := make([]*spanner.Mutation, 0, len(eventIDs))
				for _, eid := range eventIDs {
					muts = append(muts, spanner.UpdateMap("OutboxEvents", map[string]any{
						"EventId":       eid,
						"PublishedAt":   spanner.CommitTimestamp,
						"UnpublishedAt": nil,
					}))
				}
				return txn.BufferWrite(muts)
			})
			if pubErr != nil {
				s.log.Warn("reclaim cancel compensation failed to mark outbox published",
					"order_id", oid,
					"event_ids", comp.eventIDs,
					"err", pubErr,
				)
			}
		}

		reclaimed++
	}

	return reclaimed, nil
}

// StartSagaRecoveryWorker periodically runs SweepStalledSagas and ReclaimPendingCancelCompensations in the background.
func StartSagaRecoveryWorker(ctx context.Context, s *Service, interval time.Duration) {
	if s == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sweep and reclaim on startup
	if _, err := s.SweepStalledSagas(ctx); err != nil {
		s.log.Warn("saga recovery worker initial sweep failed", "err", err)
	}
	if _, err := s.ReclaimPendingCancelCompensations(ctx); err != nil {
		s.log.Warn("saga recovery worker initial reclaim failed", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SweepStalledSagas(ctx); err != nil {
				s.log.Warn("saga recovery worker sweep failed", "err", err)
			}
			if _, err := s.ReclaimPendingCancelCompensations(ctx); err != nil {
				s.log.Warn("saga recovery worker reclaim failed", "err", err)
			}
		}
	}
}

func (s *Service) getParentSagaState(ctx context.Context, parentID string) (sagaState string, status string, err error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return "", "", nil
	}
	if s.spannerClient != nil {
		row, err := s.spannerClient.Single().ReadRow(ctx, "ParentOrders", spanner.Key{parentID}, []string{"SagaState", "Status"})
		if err != nil {
			if errors.Is(err, spanner.ErrRowNotFound) {
				return "", "", nil
			}
			return "", "", err
		}
		if err := row.Columns(&sagaState, &status); err != nil {
			return "", "", err
		}
		return sagaState, status, nil
	}
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if s.parentSagaStates != nil {
		return s.parentSagaStates[parentID], "", nil
	}
	return "", "", nil
}

func (s *Service) setParentSagaState(parentID, state string) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return
	}
	s.usageMu.Lock()
	defer s.usageMu.Unlock()
	if s.parentSagaStates == nil {
		s.parentSagaStates = make(map[string]string)
	}
	s.parentSagaStates[parentID] = state
}

func (s *Service) isParentSagaCompensated(ctx context.Context, parentID string) bool {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return false
	}
	st, status, err := s.getParentSagaState(ctx, parentID)
	if err != nil {
		return false
	}
	return st == SagaStateCompensated || st == SagaStateFailed || status == parentStatusCancelled
}
