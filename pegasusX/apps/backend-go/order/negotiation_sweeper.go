package order

import (
	"context"
	"fmt"
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
	if s.spannerClient == nil {
		return fmt.Errorf("spanner unavailable")
	}
	now := s.now()
	const pageSize = 100
	var afterExpiresAt *time.Time
	var afterProposalID string

	for {
		stmt := buildExpiredNegotiationsPageStatement(now, pageSize, afterExpiresAt, afterProposalID)
		iter := s.spannerClient.Single().Query(ctx, stmt)
		type proposalCursor struct {
			id        string
			expiresAt time.Time
		}
		var page []proposalCursor
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var pid string
			var exp time.Time
			if err := row.Columns(&pid, &exp); err != nil {
				iter.Stop()
				return err
			}
			page = append(page, proposalCursor{id: pid, expiresAt: exp})
		}
		iter.Stop()

		if len(page) == 0 {
			break
		}
		for _, p := range page {
			if err := s.autoRejectNegotiation(ctx, p.id, now); err != nil {
				s.log.WarnContext(ctx, "failed to auto-reject negotiation", "proposal_id", p.id, "err", err)
			}
		}
		if len(page) < pageSize {
			break
		}
		last := page[len(page)-1]
		exp := last.expiresAt
		afterExpiresAt = &exp
		afterProposalID = last.id
	}
	return nil
}

func buildExpiredNegotiationsPageStatement(now time.Time, limit int, afterExpiresAt *time.Time, afterProposalID string) spanner.Statement {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	params := map[string]interface{}{"now": now, "lim": limit}
	cursorClause := ""
	if afterExpiresAt != nil && afterProposalID != "" {
		cursorClause = " AND (ExpiresAt > @afterExpiresAt OR (ExpiresAt = @afterExpiresAt AND ProposalId > @afterProposalId))"
		params["afterExpiresAt"] = *afterExpiresAt
		params["afterProposalId"] = afterProposalID
	}
	return spanner.Statement{
		SQL: `SELECT ProposalId, ExpiresAt FROM NegotiationProposals
		      WHERE Status = 'PENDING' AND ExpiresAt <= @now` + cursorClause + `
		      ORDER BY ExpiresAt ASC, ProposalId ASC
		      LIMIT @lim`,
		Params: params,
	}
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
