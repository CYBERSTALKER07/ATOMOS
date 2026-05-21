package payment

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// SpannerRepository persists payment aggregates and emitted outbox events
// atomically inside one Spanner ReadWriteTransaction.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed payment repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

// CreateSession persists one payment session and any outbox events atomically.
func (r *SpannerRepository) CreateSession(ctx context.Context, s SessionRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentSessions", map[string]any{
		"SessionId":   s.SessionID,
		"OrderId":     s.OrderID,
		"SupplierId":  s.SupplierID,
		"RetailerId":  s.RetailerID,
		"Gateway":     s.Gateway,
		"Currency":    s.Currency,
		"AmountMinor": s.AmountMinor,
		"Mode":        s.Mode,
		"Status":      s.Status,
		"CreatedAt":   s.CreatedAt.UTC(),
		"UpdatedAt":   s.UpdatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base)
}

// CreateSessionWithAttempt persists one payment session and first attempt in a
// single transaction with optional outbox writes.
func (r *SpannerRepository) CreateSessionWithAttempt(ctx context.Context, s SessionRecord, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	sessionMutation := spanner.InsertOrUpdateMap("PaymentSessions", map[string]any{
		"SessionId":   s.SessionID,
		"OrderId":     s.OrderID,
		"SupplierId":  s.SupplierID,
		"RetailerId":  s.RetailerID,
		"Gateway":     s.Gateway,
		"Currency":    s.Currency,
		"AmountMinor": s.AmountMinor,
		"Mode":        s.Mode,
		"Status":      s.Status,
		"CreatedAt":   s.CreatedAt.UTC(),
		"UpdatedAt":   s.UpdatedAt.UTC(),
	})

	attemptMutation := spanner.InsertOrUpdateMap("PaymentAttempts", map[string]any{
		"AttemptId":         a.AttemptID,
		"SessionId":         a.SessionID,
		"Gateway":           a.Gateway,
		"ExecutionAction":   nullIfEmpty(a.ExecutionAction),
		"ExecutionMode":     nullIfEmpty(a.ExecutionMode),
		"ProviderReference": nullIfEmpty(a.ProviderReference),
		"Status":            a.Status,
		"CreatedAt":         a.CreatedAt.UTC(),
		"UpdatedAt":         a.UpdatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, sessionMutation, attemptMutation)
}

// SaveAttempt persists one payment attempt and any outbox events atomically.
func (r *SpannerRepository) SaveAttempt(ctx context.Context, a PaymentAttemptRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentAttempts", map[string]any{
		"AttemptId":         a.AttemptID,
		"SessionId":         a.SessionID,
		"Gateway":           a.Gateway,
		"ExecutionAction":   nullIfEmpty(a.ExecutionAction),
		"ExecutionMode":     nullIfEmpty(a.ExecutionMode),
		"ProviderReference": nullIfEmpty(a.ProviderReference),
		"Status":            a.Status,
		"CreatedAt":         a.CreatedAt.UTC(),
		"UpdatedAt":         a.UpdatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base)
}

// SaveChargeback persists one chargeback request and optional outbox events.
func (r *SpannerRepository) SaveChargeback(ctx context.Context, c ChargebackRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentChargebacks", map[string]any{
		"ChargebackId": c.ChargebackID,
		"OrderId":      c.OrderID,
		"RetailerId":   c.RetailerID,
		"Gateway":      c.Gateway,
		"AmountMinor":  c.AmountMinor,
		"Currency":     c.Currency,
		"CreatedAt":    c.CreatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base)
}

// SaveReversal persists one chargeback reversal and optional outbox events.
func (r *SpannerRepository) SaveReversal(ctx context.Context, rev ReversalRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentReversals", map[string]any{
		"ReversalId": rev.ReversalID,
		"SessionId":  rev.SessionID,
		"CreatedAt":  rev.CreatedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base)
}

// SaveWebhook persists one validated webhook and optional outbox events.
func (r *SpannerRepository) SaveWebhook(ctx context.Context, w WebhookRecord, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner payment repository: nil client")
	}

	base := spanner.InsertOrUpdateMap("PaymentWebhooks", map[string]any{
		"WebhookId":      w.WebhookID,
		"Gateway":        w.Gateway,
		"TransactionId":  w.TransactionID,
		"SessionId":      nullIfEmpty(w.SessionID),
		"OrderId":        nullIfEmpty(w.OrderID),
		"Status":         w.Status,
		"AmountMinor":    w.AmountMinor,
		"Currency":       w.Currency,
		"SignatureValid": w.SignatureValid,
		"ReceivedAt":     w.ReceivedAt.UTC(),
	})

	return r.writeWithOutbox(ctx, emit, base)
}

func (r *SpannerRepository) writeWithOutbox(ctx context.Context, emit func(outbox.TxnBuffer) error, bases ...*spanner.Mutation) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := make([]*spanner.Mutation, 0, len(bases)+len(buf.events))
		mutations = append(mutations, bases...)
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
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
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("payment write transaction: %w", err)
	}
	return nil
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
