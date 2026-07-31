package cashrecon

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

const EventCashReconciliationEscalation = "cash_reconciliation.escalation"

// EscalationWorker scans stale cash reconciliations and notifies supplier finance.
type EscalationWorker struct {
	Spanner     *spanner.Client
	Notifier    *notifications.Service
	SupplierID  string
	Now         func() time.Time
	StaleAfter  time.Duration
}

// RunNightlyWorker escalates PENDING/DISPUTED rows older than StaleAfter (default 24h).
func (w *EscalationWorker) RunNightlyWorker(ctx context.Context, interval time.Duration) {
	if w == nil || w.Spanner == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	w.runOnce(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *EscalationWorker) runOnce(ctx context.Context) {
	now := w.Now()
	stale := w.StaleAfter
	if stale <= 0 {
		stale = 24 * time.Hour
	}
	cutoff := now.Add(-stale)

	iter := w.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT ReconciliationId, DriverId, DifferenceMinor, CreatedAt
			FROM CashReconciliations
			WHERE Status IN ('PENDING','DISPUTED') AND CreatedAt < @cutoff
			ORDER BY CreatedAt ASC
			LIMIT 200`,
		Params: map[string]any{"cutoff": cutoff},
	})
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.WarnContext(ctx, "cash recon escalation scan failed", "err", err)
			return
		}
		var reconID, driverID string
		var diff int64
		var createdAt time.Time
		if err := row.Columns(&reconID, &driverID, &diff, &createdAt); err != nil {
			continue
		}
		if err := w.escalateOne(ctx, reconID, driverID, diff, createdAt, now); err != nil {
			slog.WarnContext(ctx, "cash recon escalation failed", "reconciliation_id", reconID, "err", err)
		}
	}
}

func (w *EscalationWorker) escalateOne(ctx context.Context, reconID, driverID string, diff int64, createdAt, now time.Time) error {
	supplierID := stringsTrim(w.SupplierID)
	if supplierID == "" {
		return nil
	}

	send, err := w.shouldNotify(ctx, supplierID, now)
	if err != nil {
		return err
	}
	if !send {
		return nil
	}

	payload := map[string]any{
		"type":              EventCashReconciliationEscalation,
		"reconciliation_id": reconID,
		"driver_id":         driverID,
		"difference_minor":  diff,
		"supplier_id":       supplierID,
		"created_at":        createdAt.UTC().Format(time.RFC3339Nano),
		"escalated_at":      now.UTC().Format(time.RFC3339Nano),
	}

	_, err = w.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, reconID, events.TopicMain, payload); err != nil {
			return err
		}
		return txn.BufferWrite(outboxMutations(buf.events))
	})
	if err != nil {
		return err
	}

	if w.Notifier != nil {
		formatted := notifications.FormatFromEvent(EventCashReconciliationEscalation, mustJSON(payload))
		_ = w.Notifier.CreateNotification(ctx, supplierID, "ADMIN", EventCashReconciliationEscalation,
			formatted.Title, formatted.Body, formatted.DeepLink)
	}
	return nil
}

func (w *EscalationWorker) shouldNotify(ctx context.Context, principalID string, now time.Time) (bool, error) {
	if w.Notifier == nil {
		return true, nil
	}
	return w.Notifier.ShouldSendNotification(ctx, principalID, EventCashReconciliationEscalation, "PUSH", now)
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
