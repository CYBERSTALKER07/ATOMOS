package outbox

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

type mockTxnBuffer struct {
	bufferedEvents []Event
}

func (m *mockTxnBuffer) BufferOutbox(_ context.Context, e Event) error {
	m.bufferedEvents = append(m.bufferedEvents, e)
	return nil
}

type mockAuditBuffer struct {
	bufferedAudits []AuditEntry
}

func (m *mockAuditBuffer) BufferAudit(_ context.Context, entry AuditEntry) error {
	m.bufferedAudits = append(m.bufferedAudits, entry)
	return nil
}

type mockTxnAuditBuffer struct {
	mockTxnBuffer
	mockAuditBuffer
}

func TestNewEventID_GeneratesValidUUID(t *testing.T) {
	t.Parallel()

	id := newEventID()
	if len(id) != 36 {
		t.Fatalf("newEventID() length = %d, want 36 (UUID format)", len(id))
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("newEventID() returned invalid UUID %q: %v", id, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("newEventID() UUID version = %d, want 4", parsed.Version())
	}
}

func TestNewEventID_Uniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := newEventID()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate event ID generated: %s at iteration %d", id, i)
		}
		seen[id] = struct{}{}
	}
}

func TestEmitJSON_AssignsUUIDAndBuffersEvent(t *testing.T) {
	t.Parallel()

	mock := &mockTxnBuffer{}
	ctx := WithTraceID(context.Background(), "trace-outbox-test-123")

	payload := map[string]any{"order_id": "ord_100", "status": "COMPLETED"}
	err := EmitJSON(ctx, mock, "Order", "ord_100", "orders.events", payload)
	if err != nil {
		t.Fatalf("EmitJSON failed: %v", err)
	}

	if len(mock.bufferedEvents) != 1 {
		t.Fatalf("bufferedEvents count = %d, want 1", len(mock.bufferedEvents))
	}

	event := mock.bufferedEvents[0]
	if _, err := uuid.Parse(event.EventID); err != nil {
		t.Fatalf("EventID %q is not a valid UUID: %v", event.EventID, err)
	}
	if event.AggregateType != "Order" || event.AggregateID != "ord_100" || event.TopicName != "orders.events" {
		t.Fatalf("unexpected event metadata: %+v", event)
	}
	if event.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt should be non-zero")
	}

	var unmarshaled map[string]any
	if err := json.Unmarshal(event.Payload, &unmarshaled); err != nil {
		t.Fatalf("unmarshal event payload failed: %v", err)
	}
	if unmarshaled["trace_id"] != "trace-outbox-test-123" {
		t.Fatalf("trace_id in payload = %v, want trace-outbox-test-123", unmarshaled["trace_id"])
	}
}

func TestWriteAudit_AssignsUUIDAndBuffersAudit(t *testing.T) {
	t.Parallel()

	mock := &mockAuditBuffer{}
	ctx := WithTraceID(context.Background(), "trace-audit-test-456")

	err := WriteAudit(ctx, mock, "sup_1", "act_1", "SUPPLIER_ADMIN", "UPDATE_PRICE", "CatalogItem", "item_1", map[string]any{"price": 5000})
	if err != nil {
		t.Fatalf("WriteAudit failed: %v", err)
	}

	if len(mock.bufferedAudits) != 1 {
		t.Fatalf("bufferedAudits count = %d, want 1", len(mock.bufferedAudits))
	}

	audit := mock.bufferedAudits[0]
	if _, err := uuid.Parse(audit.AuditID); err != nil {
		t.Fatalf("AuditID %q is not a valid UUID: %v", audit.AuditID, err)
	}
	if audit.SupplierID != "sup_1" || audit.ActorID != "act_1" || audit.Action != "UPDATE_PRICE" {
		t.Fatalf("unexpected audit entry: %+v", audit)
	}
	if audit.TraceID != "trace-audit-test-456" {
		t.Fatalf("TraceID = %q, want trace-audit-test-456", audit.TraceID)
	}

	rowMap := audit.AuditRowMap()
	if rowMap["AuditId"] != audit.AuditID {
		t.Fatalf("AuditRowMap AuditId = %v, want %v", rowMap["AuditId"], audit.AuditID)
	}
}

func TestWriteAudit_NilBufferIsNoOp(t *testing.T) {
	t.Parallel()

	err := WriteAudit(context.Background(), nil, "sup_1", "act_1", "SUPPLIER_ADMIN", "ACTION", "Type", "id", nil)
	if err != nil {
		t.Fatalf("WriteAudit with nil buffer returned error: %v", err)
	}
}
