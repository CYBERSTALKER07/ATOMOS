package outbox

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// Event is a single entry in the transactional outbox.
type Event struct {
	// EventID is a stable UUIDv4. If left empty, Emit generates one.
	EventID string
	// AggregateType names the owning domain entity (Driver, Vehicle, Order, ...).
	AggregateType string
	// AggregateID is the primary key of the aggregate root. Used as the Kafka
	// message key so partitions preserve per-entity ordering.
	AggregateID string
	// EventType is the discriminator (e.g. "ORDER_DISPATCHED"). The relay
	// injects this as the Kafka `event_type` header so consumers can route
	// without parsing the payload. The Kafka message key stays as AggregateID
	// to preserve per-entity partition ordering.
	EventType string
	// TopicName is the destination Kafka topic.
	TopicName string
	// Payload is the already-JSON-encoded event body.
	Payload []byte
	// TraceID is the request-scoped correlation token. Propagated as a Kafka
	// header so downstream consumers can continue the trace. Empty is safe —
	// the relay omits the header rather than writing a blank value.
	TraceID string
}

// Emit appends an outbox row to the caller's spanner.ReadWriteTransaction.
// The caller is responsible for committing the transaction; if the commit
// aborts, the outbox row disappears with the rest of the work — which is
// exactly the atomicity guarantee the pattern exists to provide.
//
// Emit never publishes to Kafka directly. The Relay goroutine does that.
func Emit(txn *spanner.ReadWriteTransaction, ev Event) error {
	if txn == nil {
		return fmt.Errorf("outbox.Emit: nil transaction")
	}
	if ev.AggregateType == "" || ev.AggregateID == "" {
		return fmt.Errorf("outbox.Emit: AggregateType and AggregateID are required")
	}
	if ev.EventType == "" {
		return fmt.Errorf("outbox.Emit: EventType is required (discriminator for consumer routing)")
	}
	if ev.TopicName == "" {
		return fmt.Errorf("outbox.Emit: TopicName is required")
	}
	if len(ev.Payload) == 0 {
		return fmt.Errorf("outbox.Emit: Payload is required")
	}
	if ev.EventID == "" {
		ev.EventID = uuid.NewString()
	}

	cols := []string{"EventId", "AggregateType", "AggregateId", "EventType", "TopicName", "Payload", "CreatedAt"}
	vals := []interface{}{ev.EventID, ev.AggregateType, ev.AggregateID, ev.EventType, ev.TopicName, ev.Payload, spanner.CommitTimestamp}
	if ev.TraceID != "" {
		cols = append(cols, "TraceID")
		vals = append(vals, ev.TraceID)
	}

	return txn.BufferWrite([]*spanner.Mutation{
		spanner.Insert("OutboxEvents", cols, vals),
	})
}

// EmitJSON is a convenience wrapper that marshals an arbitrary payload to
// JSON before delegating to Emit. eventType is the discriminator string
// (e.g. kafka.EventOrderDispatched) the dispatcher will switch on.
// traceID is required at the API boundary to prevent accidental omission.
// Callers should pass telemetry.TraceIDFromContext(ctx) where available.
func EmitJSON(txn *spanner.ReadWriteTransaction, aggregateType, aggregateID, eventType, topic string, payload interface{}, traceID string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox.EmitJSON: marshal payload: %w", err)
	}
	return Emit(txn, Event{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		TopicName:     topic,
		Payload:       body,
		TraceID:       traceID,
	})
}
