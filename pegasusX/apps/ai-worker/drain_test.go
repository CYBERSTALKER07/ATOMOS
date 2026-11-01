package main

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestDetachedDrainContext_RemainsActiveWhenParentActive(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	drainCtx, drainCancel := detachedDrainContext(parent, 200*time.Millisecond)
	defer drainCancel()

	select {
	case <-drainCtx.Done():
		t.Fatal("expected drainCtx to stay alive while parent is alive")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestDetachedDrainContext_GrantsDrainWindowOnParentCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())

	drainCtx, drainCancel := detachedDrainContext(parent, 150*time.Millisecond)
	defer drainCancel()

	// Cancel parent (simulating SIGTERM)
	cancel()

	// Immediately after cancel, drainCtx should STILL be active
	if err := drainCtx.Err(); err != nil {
		t.Fatalf("drainCtx cancelled prematurely immediately after parent cancel: %v", err)
	}

	// 50ms into 150ms window, should still be active
	time.Sleep(50 * time.Millisecond)
	if err := drainCtx.Err(); err != nil {
		t.Fatalf("drainCtx cancelled before drain window elapsed: %v", err)
	}

	// After drain window (another 120ms, total 170ms > 150ms), should now be cancelled
	time.Sleep(120 * time.Millisecond)
	if err := drainCtx.Err(); err == nil {
		t.Fatal("expected drainCtx to be cancelled after drain timeout expired")
	}
}

func TestRouteToDLQ_NilWriterIsNoOp(t *testing.T) {
	msg := kafka.Message{
		Topic: "test-topic",
		Value: []byte("poison-pill"),
	}
	if err := routeToDLQ(context.Background(), nil, msg, "test-err"); err != nil {
		t.Fatalf("expected nil error on nil writer, got %v", err)
	}
}
