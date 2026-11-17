package kafka

import (
	"context"
	"sync"
	"testing"
	"time"

	segmentkafka "github.com/segmentio/kafka-go"
)

func TestInMemoryEventDedupDropsDuplicate(t *testing.T) {
	t.Parallel()
	store := NewInMemoryEventDedup(time.Minute)
	ctx := context.Background()
	ok1, err := store.ShouldProcess(ctx, "evt-1")
	if err != nil || !ok1 {
		t.Fatalf("first process: ok=%v err=%v", ok1, err)
	}
	ok2, err := store.ShouldProcess(ctx, "evt-1")
	if err != nil {
		t.Fatalf("second process err: %v", err)
	}
	if ok2 {
		t.Fatal("expected duplicate to be dropped")
	}
}

func TestDedupKeyForMessageStable(t *testing.T) {
	t.Parallel()
	k1 := DedupKeyForMessage("pegasusx-orders", 2, 99)
	k2 := DedupKeyForMessage("pegasusx-orders", 2, 99)
	if k1 != k2 {
		t.Fatalf("keys differ: %q vs %q", k1, k2)
	}
}

func TestDedupKeyForConsumerGroupIndependent(t *testing.T) {
	t.Parallel()
	store := NewInMemoryEventDedup(time.Minute)
	ctx := context.Background()
	msgKey := DedupKeyForMessage("pegasusx-main", 0, 42)
	orderKey := DedupKeyForConsumerGroup("void-order-mutator", "pegasusx-main", 0, 42)
	dispatchKey := DedupKeyForConsumerGroup("void-notification-dispatcher", "pegasusx-main", 0, 42)

	okOrder, err := store.ShouldProcess(ctx, orderKey)
	if err != nil || !okOrder {
		t.Fatalf("order consumer first: ok=%v err=%v", okOrder, err)
	}
	okDispatch, err := store.ShouldProcess(ctx, dispatchKey)
	if err != nil || !okDispatch {
		t.Fatalf("notification dispatcher should not be suppressed by order consumer: ok=%v err=%v", okDispatch, err)
	}
	if orderKey == dispatchKey || orderKey == msgKey {
		t.Fatal("expected distinct dedup keys per consumer group")
	}
}

func TestConcurrentInMemoryEventDedup(t *testing.T) {
	t.Parallel()
	store := NewInMemoryEventDedup(time.Minute)
	ctx := context.Background()
	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := store.ShouldProcess(ctx, "concurrent-key")
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 1 {
		t.Fatalf("allowed=%d want 1", allowed)
	}
}

func TestEnvelopeFromMessage_HeaderFallback(t *testing.T) {
	t.Parallel()

	// 1. Envelope with type in headers only
	msg1 := segmentkafka.Message{
		Value: []byte(`{"order_id": "ord-123", "amount": 500}`),
		Headers: []segmentkafka.Header{
			{Key: "type", Value: []byte("ORDER_CREATED")},
			{Key: "trace_id", Value: []byte("trace-abc")},
			{Key: "timestamp", Value: []byte("2026-10-09T00:00:00Z")},
		},
	}
	env1, err := EnvelopeFromMessage(msg1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if env1.Type != "ORDER_CREATED" {
		t.Fatalf("expected Type 'ORDER_CREATED', got %q", env1.Type)
	}
	if env1.TraceID != "trace-abc" {
		t.Fatalf("expected TraceID 'trace-abc', got %q", env1.TraceID)
	}
	if env1.Timestamp != "2026-10-09T00:00:00Z" {
		t.Fatalf("expected Timestamp '2026-10-09T00:00:00Z', got %q", env1.Timestamp)
	}

	// 2. Envelope with type in body takes precedence
	msg2 := segmentkafka.Message{
		Value: []byte(`{"type": "ORDER_COMPLETED", "trace_id": "trace-body", "v": 2}`),
		Headers: []segmentkafka.Header{
			{Key: "type", Value: []byte("ORDER_CREATED")},
			{Key: "trace_id", Value: []byte("trace-header")},
		},
	}
	env2, err := EnvelopeFromMessage(msg2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if env2.Type != "ORDER_COMPLETED" {
		t.Fatalf("expected body Type 'ORDER_COMPLETED', got %q", env2.Type)
	}
	if env2.TraceID != "trace-body" {
		t.Fatalf("expected body TraceID 'trace-body', got %q", env2.TraceID)
	}
	if env2.Version != 2 {
		t.Fatalf("expected Version 2, got %d", env2.Version)
	}
}

func TestWithEventDedup_BindsTraceAndHandlesPayloadEventID(t *testing.T) {
	t.Parallel()

	store := NewInMemoryEventDedup(time.Minute)
	var capturedTrace string
	var callCount int

	handler := func(ctx context.Context, msg segmentkafka.Message) error {
		callCount++
		capturedTrace = TraceIDFromMessage(msg)
		return nil
	}

	wrapped := WithEventDedup(store, "test-group", handler)

	// First call with payload event_id and header trace_id
	msg := segmentkafka.Message{
		Topic:     "orders",
		Partition: 1,
		Offset:    100,
		Value:     []byte(`{"event_id": "evt-uuid-999", "amount": 100}`),
		Headers: []segmentkafka.Header{
			{Key: "trace_id", Value: []byte("trace-test-123")},
		},
	}

	if err := wrapped(context.Background(), msg); err != nil {
		t.Fatalf("wrapped err: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
	if capturedTrace != "trace-test-123" {
		t.Fatalf("expected trace-test-123, got %q", capturedTrace)
	}

	// Second call with same event_id even at different offset should be deduplicated!
	dupMsg := segmentkafka.Message{
		Topic:     "orders",
		Partition: 1,
		Offset:    101, // different offset, but same logical event_id
		Value:     []byte(`{"event_id": "evt-uuid-999", "amount": 100}`),
	}
	if err := wrapped(context.Background(), dupMsg); err != nil {
		t.Fatalf("duplicate wrapped err: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("duplicate should be suppressed! calls=%d", callCount)
	}
}

