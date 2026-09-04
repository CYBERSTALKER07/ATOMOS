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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type traceIDContextKey struct{}

// Event is the on-the-wire shape persisted to OutboxEvents and republished to
// Kafka. Payload is opaque JSON serialized by EmitJSON.
// SupplierID soft-partitions the outbox (Gate 5 Week 9). Always stamped;
// PlatformSupplierID when the payload has no tenant.
type Event struct {
	EventID       string
	AggregateType string
	AggregateID   string
	TopicName     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
	SupplierID    string
}

// PlatformSupplierID is stamped on OutboxEvents that are not tenant-scoped
// (system/platform emits). Required once SupplierId is NOT NULL.
const PlatformSupplierID = "_platform"

// ResolveSupplierID returns an explicit supplier, payload-derived id, or PlatformSupplierID.
func ResolveSupplierID(explicit string, payload []byte) string {
	if sid := strings.TrimSpace(explicit); sid != "" {
		return sid
	}
	if sid := SupplierIDFromPayload(payload); sid != "" {
		return sid
	}
	return PlatformSupplierID
}

// TxnBuffer is implemented by any RW-transaction handle that can buffer a
// mutation. The concrete Spanner adapter wraps *spanner.ReadWriteTransaction
// and translates Buffer into spanner.InsertOrUpdateMap on OutboxEvents.
type TxnBuffer interface {
	BufferOutbox(ctx context.Context, e Event) error
}

// Publisher is the Kafka producer seam. Implementations MUST configure
// RequiredAcks=all, high MaxAttempts, and a WriteTimeout (delivery bound).
// The Relay treats a nil error as durable success and marks PublishedAt.
type Publisher interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
}

// HeaderPublisher optionally attaches Kafka headers (e.g. event_id for dedupe).
type HeaderPublisher interface {
	PublishWithHeaders(ctx context.Context, topic string, key []byte, value []byte, headers map[string][]byte) error
}

// Store is the read side: the Relay calls Fetch to pull unpublished events and
// MarkPublished to clear them after a successful Publish.
type Store interface {
	Fetch(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, eventIDs []string, at time.Time) error
	CountUnpublished(ctx context.Context) (int64, error)
	// RecordPublishFailures increments the persistent per-event attempt counter.
	// Events reaching maxAttempts are moved to the dead-letter sink in the same
	// transaction and their IDs returned; they leave the retry set permanently
	// (never silently dropped).
	RecordPublishFailures(ctx context.Context, eventIDs []string, lastErr string, maxAttempts int64) (deadLettered []string, err error)
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
		SupplierID:    ResolveSupplierID("", raw),
	})
}

// SupplierIDFromPayload extracts supplier_id / SupplierId from JSON object payloads.
func SupplierIDFromPayload(raw []byte) string {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	for _, key := range []string{"supplier_id", "SupplierId", "supplierId"} {
		if v, ok := object[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// EventRowMap builds an OutboxEvents InsertOrUpdateMap including required SupplierId.
func EventRowMap(e Event) map[string]interface{} {
	createdAt := e.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	row := map[string]interface{}{
		"EventId":       ClampEventID(e.EventID),
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     createdAt,
		"PublishedAt":   nil,
		"ClaimedBy":     nil,
		"ClaimedUntil":  nil,
		"SupplierId":    ResolveSupplierID(e.SupplierID, e.Payload),
	}
	if e.PublishedAt != nil {
		row["PublishedAt"] = e.PublishedAt.UTC()
	}
	return row
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

// EventIDMax is OutboxEvents.EventId STRING(36).
const EventIDMax = 36

// ClampEventID returns a STRING(36)-safe id. Empty → UUIDv4. Oversize seeds
// hash to 32 hex so retries stay deterministic.
func ClampEventID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return uuid.NewString()
	}
	if len(id) <= EventIDMax {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:16])
}

// newEventID is overridable in tests. Production uses random UUIDv4 (STRING(36)).
var newEventID = func() string {
	return uuid.NewString()
}
