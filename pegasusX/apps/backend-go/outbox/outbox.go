// Package outbox implements the Transactional Outbox pattern: every durable
// state change writes a domain row AND an OutboxEvents row in the same Spanner
// ReadWriteTransaction. The Relay tails unpublished rows and forwards them to
// Kafka with at-least-once delivery, then marks PublishedAt.
//
// This package is storage-agnostic at compile time: TxnBuffer + Publisher
// abstract the Spanner / Kafka clients so unit tests and bootstrap can wire
// stub implementations without import cycles.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type traceIDContextKey struct{}

// Event is the on-the-wire shape persisted to OutboxEvents and republished to
// Kafka. Payload is opaque JSON serialized by EmitJSON.
type Event struct {
	EventID       string
	AggregateType string
	AggregateID   string
	TopicName     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

// TxnBuffer is implemented by any RW-transaction handle that can buffer a
// mutation. The concrete Spanner adapter wraps *spanner.ReadWriteTransaction
// and translates Buffer into spanner.InsertOrUpdateMap on OutboxEvents.
type TxnBuffer interface {
	BufferOutbox(ctx context.Context, e Event) error
}

// Publisher is the Kafka producer seam. Implementations MUST configure
// RequiredAcks=all and MaxAttempts>=5 with backoff+jitter. The Relay treats a
// nil error as durable success and clears PublishedAt accordingly.
type Publisher interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
}

// Store is the read side: the Relay calls Fetch to pull unpublished events and
// MarkPublished to clear them after a successful Publish.
type Store interface {
	Fetch(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, eventIDs []string, at time.Time) error
}

// WithTraceID attaches request trace context used by outbox emitters.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmed := strings.TrimSpace(traceID)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDContextKey{}, trimmed)
}

// TraceIDFromContext returns the trace identifier previously attached via
// WithTraceID.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(traceIDContextKey{}).(string)
	return v
}

// EmitJSON marshals payload as JSON and buffers an Event onto the active
// transaction. Callers MUST pass the same trace_id that flows through their
// request context so cross-system correlation remains intact.
//
// AggregateType: the domain noun being mutated ("Retailer", "Order").
// AggregateID:   the primary key of the row being mutated.
// Topic:         the canonical Kafka topic name.
func EmitJSON(ctx context.Context, txn TxnBuffer, aggregateType, aggregateID, topic string, payload any) error {
	if txn == nil {
		return fmt.Errorf("outbox: nil txn")
	}
	if aggregateType == "" || aggregateID == "" || topic == "" {
		return fmt.Errorf("outbox: aggregateType/aggregateID/topic required")
	}
	traceID := TraceIDFromContext(ctx)
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox: marshal payload: %w", err)
	}
	if traceID != "" {
		raw, err = injectTraceID(raw, traceID)
		if err != nil {
			return fmt.Errorf("outbox: inject trace id: %w", err)
		}
	}
	return txn.BufferOutbox(ctx, Event{
		EventID:       newEventID(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		TopicName:     topic,
		Payload:       raw,
		CreatedAt:     time.Now().UTC(),
	})
}

func injectTraceID(raw []byte, traceID string) ([]byte, error) {
	if strings.TrimSpace(traceID) == "" {
		return raw, nil
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		// Non-object payloads (or malformed JSON) are passed through unchanged
		// because upstream marshaling has already succeeded.
		return raw, nil
	}
	if _, exists := object["trace_id"]; exists {
		return raw, nil
	}
	object["trace_id"] = traceID
	patched, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return patched, nil
}

// AuditBuffer is implemented by any RW-transaction handle that can buffer an
// AuditLog mutation. The concrete Spanner adapter wraps *spanner.ReadWriteTransaction.
type AuditBuffer interface {
	BufferAudit(ctx context.Context, entry AuditEntry) error
}

// TxnAuditBuffer combines TxnBuffer and AuditBuffer for emit callbacks that
// need to write both outbox events and audit rows atomically.
type TxnAuditBuffer interface {
	TxnBuffer
	AuditBuffer
}

// AuditEntry represents one row in the AuditLog table.
type AuditEntry struct {
	AuditID       string
	SupplierID    string
	ActorID       string
	ActorRole     string
	Action        string
	AggregateType string
	AggregateID   string
	DetailsJSON   []byte
	TraceID       string
	CreatedAt     time.Time
}

// WriteAudit buffers an AuditLog row inside the current transaction. If buf is
// nil (Spanner not configured), the call is a no-op so callers don't need nil
// checks.
func WriteAudit(ctx context.Context, buf AuditBuffer, supplierID, actorID, actorRole, action, aggregateType, aggregateID string, details any) error {
	if buf == nil {
		return nil
	}
	var detailsJSON []byte
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return fmt.Errorf("audit: marshal details: %w", err)
		}
	}
	return buf.BufferAudit(ctx, AuditEntry{
		AuditID:       newEventID(),
		SupplierID:    supplierID,
		ActorID:       actorID,
		ActorRole:     actorRole,
		Action:        action,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		DetailsJSON:   detailsJSON,
		TraceID:       TraceIDFromContext(ctx),
		CreatedAt:     time.Now().UTC(),
	})
}

// AuditRowMap returns a map suitable for spanner.InsertMap("AuditLog", ...).
func (e AuditEntry) AuditRowMap() map[string]any {
	createdAt := e.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	m := map[string]any{
		"AuditId":       e.AuditID,
		"SupplierId":    e.SupplierID,
		"ActorId":       e.ActorID,
		"ActorRole":     e.ActorRole,
		"Action":        e.Action,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"DetailsJson":   e.DetailsJSON,
		"TraceId":       e.TraceID,
		"CreatedAt":     createdAt,
	}
	return m
}

// newEventID is overridable in tests. Production uses crypto/rand UUIDv7.
var newEventID = func() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
