package order

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// StartNegotiationSweeper starts a background loop to auto-reject expired negotiations.
// No-op while quantity negotiation is product-disabled (see negotiation_disabled.go).
func (s *Service) StartNegotiationSweeper(ctx context.Context) {
	if quantityNegotiationDisabled() {
		if s.log != nil {
			s.log.InfoContext(ctx, "negotiation sweeper not started: feature_disabled")
		}
		return
	}
	if s.spannerClient == nil {
		if s.log != nil {
			s.log.InfoContext(ctx, "negotiation sweeper not started: spanner unavailable")
		}
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			if err := s.SweepExpiredNegotiations(sweepCtx); err != nil {
				s.log.ErrorContext(sweepCtx, "negotiation sweeper failed", "err", err)
			}
			cancel()
		}
	}
}

// NegotiationFeatureEnabled reports whether quantity negotiation APIs are live.
func NegotiationFeatureEnabled() bool {
	return !quantityNegotiationDisabled()
}

// SweepExpiredNegotiations finds PENDING negotiations that have expired and AUTO_REJECTs them.
func (s *Service) SweepExpiredNegotiations(ctx context.Context) error {
	now := s.now()
	stmt := spanner.Statement{
		SQL: `SELECT ProposalId FROM NegotiationProposals 
		      WHERE Status = 'PENDING' AND ExpiresAt <= @now
		      LIMIT 100`,
		Params: map[string]interface{}{"now": now},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var proposals []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var pid string
		if err := row.Columns(&pid); err != nil {
			return err
		}
		proposals = append(proposals, pid)
	}

	for _, pid := range proposals {
		if err := s.autoRejectNegotiation(ctx, pid, now); err != nil {
			s.log.WarnContext(ctx, "failed to auto-reject negotiation", "proposal_id", pid, "err", err)
		}
	}
	return nil
}

func (s *Service) autoRejectNegotiation(ctx context.Context, proposalID string, now time.Time) error {
	var supplierID, retailerID, driverID, orderID string

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "NegotiationProposals", spanner.Key{proposalID}, []string{"OrderId", "DriverId", "Status"})
		if err != nil {
			return err
		}
		var status string
		if err := row.Columns(&orderID, &driverID, &status); err != nil {
			return err
		}
		if status != "PENDING" {
			return nil
		}

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"SupplierId", "RetailerId"})
		if err != nil {
			return err
		}
		var supplierCol, retailerCol spanner.NullString
		if err := orderRow.Columns(&supplierCol, &retailerCol); err != nil {
			return err
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		if retailerCol.Valid {
			retailerID = retailerCol.StringVal
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("NegotiationProposals", map[string]any{
				"ProposalId": proposalID,
				"Status":     "REJECTED",
				"Resolution": "AUTO_REJECTED",
				"ResolvedAt": now.UTC(),
				"ResolvedBy": "system",
			}),
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventNegotiationResolved, Timestamp: now.Format(time.RFC3339Nano)},
			ProposalID: proposalID,
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			DriverID:   driverID,
			Action:     "REJECT",
			Resolution: "AUTO_REJECTED",
		}); err != nil {
			return err
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})

	if err == nil && orderID != "" {
		s.broadcastNegotiation(ctx, supplierID, retailerID, driverID, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventNegotiationResolved, Timestamp: now.Format(time.RFC3339Nano)},
			ProposalID: proposalID,
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			DriverID:   driverID,
			Action:     "REJECT",
			Resolution: "AUTO_REJECTED",
		})
	}

	return err
}
