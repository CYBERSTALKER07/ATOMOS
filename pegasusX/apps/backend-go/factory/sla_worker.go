package factory

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// RunFactorySLABreachWorker periodically emits FACTORY_SLA_BREACH for open overdue requests (G7.1).
// Idempotent via WarehouseSupplyRequests.SlaBreachNotifiedAt when column present.
func (s *Service) RunFactorySLABreachWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// First pass shortly after start.
	if err := s.ScanAndNotifySLABreaches(ctx); err != nil && s.log != nil {
		s.log.Warn("factory sla breach scan failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ScanAndNotifySLABreaches(ctx); err != nil && s.log != nil {
				s.log.Warn("factory sla breach scan failed", "err", err)
			}
		}
	}
}

// ScanAndNotifySLABreaches finds open BREACHED requests and emits one outbox event each.
func (s *Service) ScanAndNotifySLABreaches(ctx context.Context) error {
	if s == nil || s.spannerClient == nil {
		return nil
	}
	now := s.now()
	// Prefer SQL filter on RequestedDeliveryDate; also catch default-window breaches in Go.
	stmt := spanner.Statement{
		SQL: `SELECT RequestId, SupplierId, WarehouseId, State, CreatedAt, RequestedDeliveryDate,
		             SlaBreachNotifiedAt
		      FROM WarehouseSupplyRequests
		      WHERE State IN UNNEST(@states)
		      LIMIT 500`,
		Params: map[string]any{
			"states": []string{"SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY"},
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	type cand struct {
		RequestID, SupplierID, WarehouseID, State string
		CreatedAt                                 time.Time
		Delivery                                  spanner.NullTime
		Notified                                  spanner.NullTime
	}
	var cands []cand
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Column missing (pre-migration): fall back without notify column.
			if strings.Contains(err.Error(), "SlaBreachNotifiedAt") {
				return s.scanSLABreachesLegacy(ctx, now)
			}
			return err
		}
		var c cand
		if err := row.Columns(&c.RequestID, &c.SupplierID, &c.WarehouseID, &c.State, &c.CreatedAt, &c.Delivery, &c.Notified); err != nil {
			return err
		}
		if c.Notified.Valid {
			continue
		}
		var delivery time.Time
		if c.Delivery.Valid {
			delivery = c.Delivery.Time
		}
		eval := EvaluateSLA(c.State, c.CreatedAt, delivery, now)
		if eval.Status != SLAStatusBreached {
			continue
		}
		cands = append(cands, c)
	}

	for _, c := range cands {
		if err := s.emitSLABreach(ctx, c.RequestID, c.SupplierID, c.WarehouseID, c.State, true); err != nil {
			if s.log != nil {
				s.log.Warn("factory sla breach emit failed", "request_id", c.RequestID, "err", err)
			}
		}
	}
	return nil
}

func (s *Service) scanSLABreachesLegacy(ctx context.Context, now time.Time) error {
	stmt := spanner.Statement{
		SQL: `SELECT RequestId, SupplierId, WarehouseId, State, CreatedAt, RequestedDeliveryDate
		      FROM WarehouseSupplyRequests
		      WHERE State IN UNNEST(@states)
		      LIMIT 200`,
		Params: map[string]any{
			"states": []string{"SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY"},
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var id, sid, wid, state string
		var created time.Time
		var delivery spanner.NullTime
		if err := row.Columns(&id, &sid, &wid, &state, &created, &delivery); err != nil {
			return err
		}
		var d time.Time
		if delivery.Valid {
			d = delivery.Time
		}
		if EvaluateSLA(state, created, d, now).Status != SLAStatusBreached {
			continue
		}
		// Without notify column, emit only (may re-fire) — still better than silent.
		_ = s.emitSLABreach(ctx, id, sid, wid, state, false)
	}
	return nil
}

func (s *Service) emitSLABreach(ctx context.Context, requestID, supplierID, warehouseID, state string, markNotified bool) error {
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &slaTxnBuf{}
		payload := map[string]any{
			"type":         events.EventFactorySLABreach,
			"request_id":   requestID,
			"supplier_id":  supplierID,
			"warehouse_id": warehouseID,
			"state":        state,
			"kind":         "supply_request",
			"timestamp":    s.now().UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, buf, "WarehouseSupplyRequest", requestID, events.TopicMain, payload); err != nil {
			return err
		}
		muts := buf.muts
		if markNotified {
			muts = append(muts, spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
				"RequestId":            requestID,
				"SlaBreachNotifiedAt":  spanner.CommitTimestamp,
				"UpdatedAt":            spanner.CommitTimestamp,
			}))
		}
		return txn.BufferWrite(muts)
	})
	return err
}

type slaTxnBuf struct {
	muts []*spanner.Mutation
}

func (b *slaTxnBuf) BufferOutbox(_ context.Context, e outbox.Event) error {
	createdAt := e.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if e.EventID == "" {
		e.EventID = uuid.NewString()
	}
	row := map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     createdAt,
		"PublishedAt":   nil,
	}
	b.muts = append(b.muts, spanner.InsertOrUpdateMap("OutboxEvents", row))
	return nil
}

