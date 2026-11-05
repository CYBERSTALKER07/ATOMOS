package outbox

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestEventRowMap_UnpublishedAtSentinel(t *testing.T) {
	t.Parallel()

	// Pending event: UnpublishedAt must be spanner.CommitTimestamp
	ePending := Event{
		EventID:       "evt-pending-1",
		AggregateType: "Order",
		AggregateID:   "order-1",
		TopicName:     "orders",
		Payload:       []byte(`{"order_id":"order-1"}`),
		SupplierID:    "sup-1",
	}
	rowPending := EventRowMap(ePending)
	if rowPending["UnpublishedAt"] != spanner.CommitTimestamp {
		t.Fatalf("expected UnpublishedAt to be spanner.CommitTimestamp for pending event, got %#v", rowPending["UnpublishedAt"])
	}
	if rowPending["PublishedAt"] != nil {
		t.Fatalf("expected PublishedAt to be nil for pending event, got %#v", rowPending["PublishedAt"])
	}

	// Already published event: UnpublishedAt must be nil
	now := time.Now().UTC()
	ePublished := Event{
		EventID:       "evt-pub-1",
		AggregateType: "Order",
		AggregateID:   "order-1",
		TopicName:     "orders",
		Payload:       []byte(`{"order_id":"order-1"}`),
		SupplierID:    "sup-1",
		PublishedAt:   &now,
	}
	rowPublished := EventRowMap(ePublished)
	if rowPublished["UnpublishedAt"] != nil {
		t.Fatalf("expected UnpublishedAt to be nil for published event, got %#v", rowPublished["UnpublishedAt"])
	}
	if rowPublished["PublishedAt"] == nil {
		t.Fatalf("expected PublishedAt to be set for published event, got nil")
	}
}

func TestBackfillUnpublishedAt_NilClient(t *testing.T) {
	t.Parallel()

	_, err := BackfillUnpublishedAt(context.Background(), nil, 100)
	if err == nil {
		t.Fatal("expected error with nil client, got nil")
	}
}
