package retailer

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Wave C4.1: scan OPEN assist tickets past SLA and notify staff.
// Flag ASSIST_SLA_ENABLED (default off). Interval ASSIST_SLA_MINUTES (default 15).

func (s *Service) assistSLAEnabled() bool {
	if s != nil && s.assistSLAOverride != nil {
		return *s.assistSLAOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ASSIST_SLA_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func assistSLAMinutesFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("ASSIST_SLA_MINUTES"))
	if raw == "" {
		return defaultAssistSLAMinutes
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return n
	}
	return defaultAssistSLAMinutes
}

// RunAssistSLAWorker scans overdue OPEN tickets and emits breach notifications.
func (s *Service) RunAssistSLAWorker(ctx context.Context, interval time.Duration) {
	if s == nil {
		return
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if s.assistSLAEnabled() {
		if n, err := s.processAssistSLABreaches(ctx); err != nil && s.log != nil {
			s.log.Warn("assist sla worker initial pass failed", "err", err)
		} else if s.log != nil && n > 0 {
			s.log.Info("assist sla breaches processed", "count", n)
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.assistSLAEnabled() {
				continue
			}
			n, err := s.processAssistSLABreaches(ctx)
			if err != nil && s.log != nil {
				s.log.Warn("assist sla worker tick failed", "err", err)
				continue
			}
			if s.log != nil && n > 0 {
				s.log.Info("assist sla breaches processed", "count", n)
			}
		}
	}
}

func (s *Service) processAssistSLABreaches(ctx context.Context) (int, error) {
	now := s.now().UTC()
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		var n int
		for id, t := range s.assistTickets {
			if t.Status != AssistOpen || t.SlaBreachedAt != "" {
				continue
			}
			if t.SlaDueAt == "" {
				continue
			}
			due, err := time.Parse(time.RFC3339Nano, t.SlaDueAt)
			if err != nil || !now.After(due) {
				continue
			}
			t.SlaBreachedAt = now.Format(time.RFC3339Nano)
			s.assistTickets[id] = t
			s.notifyAssistBreach(ctx, t.RetailerID, t)
			n++
		}
		return n, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT TicketId, RetailerId, LocationId, SectionId, Note, Status, CreatedByUserId,
			ClaimedByUserId, CompletedByUserId, CreatedAt, ClaimedAt, CompletedAt, SlaDueAt
			FROM RetailerAssistanceTickets
			WHERE Status = @status AND SlaDueAt < @now AND SlaBreachedAt IS NULL
			LIMIT 100`,
		Params: map[string]any{"status": AssistOpen, "now": now},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	var candidates []AssistTicketDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		t, ok, err := decodeAssistRow(row)
		if err != nil || !ok {
			continue
		}
		candidates = append(candidates, t)
	}

	var processed int
	for _, t := range candidates {
		breached, err := s.markAssistSLABreached(ctx, t.TicketID, now)
		if err != nil {
			if s.log != nil {
				s.log.Warn("assist sla breach failed", "ticket_id", t.TicketID, "err", err)
			}
			continue
		}
		if !breached {
			continue
		}
		t.SlaBreachedAt = now.Format(time.RFC3339Nano)
		s.notifyAssistBreach(ctx, t.RetailerID, t)
		processed++
	}
	return processed, nil
}

func (s *Service) markAssistSLABreached(ctx context.Context, ticketID string, now time.Time) (bool, error) {
	if s.spannerClient == nil {
		return false, nil
	}
	var breached bool
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "RetailerAssistanceTickets", spanner.Key{ticketID},
			[]string{"Status", "SlaBreachedAt", "RetailerId", "SectionId"})
		if err != nil {
			return err
		}
		var status, retailerID, sectionID string
		var breachedAt spanner.NullTime
		if err := row.Columns(&status, &breachedAt, &retailerID, &sectionID); err != nil {
			return err
		}
		if status != AssistOpen || breachedAt.Valid {
			return nil
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("RetailerAssistanceTickets", map[string]any{
				"TicketId":      ticketID,
				"SlaBreachedAt": now,
			}),
		}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":       events.EventRetailerAssistTicketSLABreached,
			"timestamp":  now.Format(time.RFC3339Nano),
			"ticket_id":  ticketID,
			"retailer_id": retailerID,
			"section_id": sectionID,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
			return err
		}
		breached = true
		return buf.Flush(txn)
	})
	return breached, err
}

func (s *Service) notifyAssistBreach(ctx context.Context, orgID string, ticket AssistTicketDTO) {
	title := "Assist ticket SLA breached"
	s.notifyAssistStaff(ctx, orgID, ticket, title)
	_ = s.emitPosEvent(ctx, orgID, events.EventRetailerAssistTicketSLABreached, map[string]any{
		"ticket_id":  ticket.TicketID,
		"section_id": ticket.SectionID,
		"location_id": ticket.LocationID,
	})
}

// ProcessAssistSLABreachesOnce runs one SLA scan (tests and e2e).
func (s *Service) ProcessAssistSLABreachesOnce(ctx context.Context) (int, error) {
	return s.processAssistSLABreaches(ctx)
}
