package retailer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

// Wave C4.1 Assist SLA worker.
// Flag ASSIST_SLA_ENABLED (default off). Default SLA 15 min (ASSIST_SLA_MINUTES / pack config).
// Channel: in-app notification (feeds push + WS via existing notif fabric). SMS only if ASSIST_SLA_SMS=1 and wired later.

func (s *Service) assistSLAEnabled() bool {
	if s != nil && s.assistSLAOverride != nil {
		return *s.assistSLAOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("ASSIST_SLA_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// SweepAssistSLA finds OPEN tickets past SlaDueAt without prior breach notice,
// emits notif + outbox event, marks SlaBreachNotifiedAt (idempotent).
// Returns number of tickets notified.
func (s *Service) SweepAssistSLA(ctx context.Context) (int, error) {
	if s == nil || !s.assistSLAEnabled() {
		return 0, nil
	}
	now := s.now().UTC()
	overdue, err := s.listAssistOpenPastSLA(ctx, now, 100)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, t := range overdue {
		if strings.TrimSpace(t.SlaBreachNotifiedAt) != "" {
			continue
		}
		if err := s.notifyAssistSLABreach(ctx, t); err != nil {
			if s.log != nil {
				s.log.Warn("assist sla notify failed", "ticket_id", t.TicketID, "err", err)
			}
			continue
		}
		t.SlaBreachNotifiedAt = now.Format(time.RFC3339Nano)
		if err := s.saveAssistTicket(ctx, t); err != nil {
			if s.log != nil {
				s.log.Warn("assist sla mark notified failed", "ticket_id", t.TicketID, "err", err)
			}
			continue
		}
		_ = s.emitPosEvent(ctx, t.RetailerID, events.EventRetailerAssistSLABreached, map[string]any{
			"ticket_id":   t.TicketID,
			"section_id":  t.SectionID,
			"location_id": t.LocationID,
			"sla_due_at":  t.SlaDueAt,
		})
		n++
	}
	return n, nil
}

func (s *Service) notifyAssistSLABreach(ctx context.Context, ticket AssistTicketDTO) error {
	title := "Assist SLA breached"
	body := fmt.Sprintf("Ticket %s (section %s) open past SLA: %s",
		shortID(ticket.TicketID), ticket.SectionID, ticket.Note)
	deep := "/assist?ticket_id=" + ticket.TicketID
	// Reuse staff fanout (in-app → push/WS via notification dispatcher).
	s.notifyAssistStaff(ctx, ticket.RetailerID, ticket, title)
	// Also explicit event type for SLA if writer supports it
	if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
		_ = w.CreateNotification(ctx, ticket.RetailerID, "RETAILER",
			events.EventRetailerAssistSLABreached, title, body, deep)
		if ticket.CreatedByUserID != "" {
			_ = w.CreateNotification(ctx, ticket.CreatedByUserID, "RETAILER",
				events.EventRetailerAssistSLABreached, title, body, deep)
		}
	}
	// Optional SMS hook (not implemented — env gates future wiring).
	if strings.TrimSpace(os.Getenv("ASSIST_SLA_SMS")) == "1" {
		if s.log != nil {
			s.log.Info("assist sla sms skipped (channel not configured)", "ticket_id", ticket.TicketID)
		}
	}
	return nil
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// listAssistOpenPastSLA returns OPEN tickets with SlaDueAt < now and no breach notice yet.
func (s *Service) listAssistOpenPastSLA(ctx context.Context, now time.Time, limit int) ([]AssistTicketDTO, error) {
	if limit <= 0 {
		limit = 100
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []AssistTicketDTO
		for _, t := range s.assistTickets {
			if t.Status != AssistOpen {
				continue
			}
			if strings.TrimSpace(t.SlaBreachNotifiedAt) != "" {
				continue
			}
			if t.SlaDueAt == "" {
				continue
			}
			due, err := time.Parse(time.RFC3339Nano, t.SlaDueAt)
			if err != nil {
				due, err = time.Parse(time.RFC3339, t.SlaDueAt)
			}
			if err != nil || !now.After(due) {
				continue
			}
			out = append(out, t)
			if len(out) >= limit {
				break
			}
		}
		return out, nil
	}

	// Prefer index-friendly query; fall back if SlaBreachNotifiedAt missing.
	sql := `SELECT TicketId, RetailerId, LocationId, SectionId, Note, Status,
		CreatedByUserId, ClaimedByUserId, CompletedByUserId, CreatedAt, ClaimedAt, CompletedAt, SlaDueAt, SlaBreachNotifiedAt
		FROM RetailerAssistanceTickets
		WHERE Status = @st AND SlaDueAt IS NOT NULL AND SlaDueAt < @now
		  AND SlaBreachNotifiedAt IS NULL
		ORDER BY SlaDueAt ASC
		LIMIT @lim`
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"st": AssistOpen, "now": now, "lim": int64(limit)},
	})
	defer iter.Stop()
	var out []AssistTicketDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "SlaBreachNotifiedAt") || strings.Contains(err.Error(), "not found") {
				return s.listAssistOpenPastSLALegacy(ctx, now, limit)
			}
			return nil, err
		}
		t, _, err := decodeAssistRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Service) listAssistOpenPastSLALegacy(ctx context.Context, now time.Time, limit int) ([]AssistTicketDTO, error) {
	sql := `SELECT TicketId, RetailerId, LocationId, SectionId, Note, Status,
		CreatedByUserId, ClaimedByUserId, CompletedByUserId, CreatedAt, ClaimedAt, CompletedAt, SlaDueAt
		FROM RetailerAssistanceTickets
		WHERE Status = @st AND SlaDueAt IS NOT NULL AND SlaDueAt < @now
		ORDER BY SlaDueAt ASC
		LIMIT @lim`
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: map[string]any{"st": AssistOpen, "now": now, "lim": int64(limit)},
	})
	defer iter.Stop()
	var out []AssistTicketDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, _, err := decodeAssistRow(row)
		if err != nil {
			return nil, err
		}
		// Without column, use memory side-map if present
		if t.SlaBreachNotifiedAt != "" {
			continue
		}
		s.mu.RLock()
		if s.assistSLANotified != nil && s.assistSLANotified[t.TicketID] {
			s.mu.RUnlock()
			continue
		}
		s.mu.RUnlock()
		out = append(out, t)
	}
	return out, nil
}

// RunAssistSLAWorker periodically sweeps overdue OPEN assist tickets.
// Interval default 1 minute. No-op when ASSIST_SLA_ENABLED is off.
func (s *Service) RunAssistSLAWorker(ctx context.Context, interval time.Duration) {
	if s == nil {
		return
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if s.assistSLAEnabled() {
		if n, err := s.SweepAssistSLA(ctx); err != nil {
			if s.log != nil {
				s.log.Warn("assist sla sweep initial failed", "err", err)
			}
		} else if s.log != nil && n > 0 {
			s.log.Info("assist sla breaches notified", "count", n)
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
			n, err := s.SweepAssistSLA(ctx)
			if err != nil {
				if s.log != nil {
					s.log.Warn("assist sla sweep failed", "err", err)
				}
				continue
			}
			if s.log != nil && n > 0 {
				s.log.Info("assist sla breaches notified", "count", n)
			}
		}
	}
}
