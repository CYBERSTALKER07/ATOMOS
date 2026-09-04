package order

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Regression: Spanner re-invokes the RW closure after an Aborted commit.
// UpdateOrder must derive the new Version from the row read inside the
// transaction, not from the caller struct — otherwise the replayed closure
// compares an already-incremented local copy against the still-old row and
// fails with a phantom "expected 2, got 1" conflict (seen in SSMR preorder
// reject returning HTTP 500).
func TestUpdateOrder_ClosureReplayIsIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newSpannerIntegrationClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	orderID := fmt.Sprintf("ord_replay_%d", suffix)
	now := time.Now().UTC().Add(-2 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            orderID,
			"RetailerId":         "ret-replay",
			"SupplierId":         "sup-replay",
			"WarehouseId":        "wh-replay",
			"Status":             string(StatusPending),
			"OrderSource":        string(OrderSourceManual),
			"ConfirmationStatus": string(ConfirmationStatusConfirmed),
			"LineItemsJson":      []byte("[]"),
			"TotalMinor":         int64(1000),
			"Currency":           "UZS",
			"H3Cell":             "",
			"Lat":                float64(41.31),
			"Lng":                float64(69.25),
			"Version":            int64(1),
			"CreatedAt":          now,
			"UpdatedAt":          now,
		}),
	})
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	repo := NewSpannerRepository(client)
	current, ok, err := repo.GetOrder(ctx, orderID)
	if err != nil || !ok {
		t.Fatalf("get order: ok=%v err=%v", ok, err)
	}
	current.Status = StatusCompleted

	// Force the transaction fn to be replayed: first invocation returns
	// Aborted, which the Spanner client treats as a transaction abort and
	// re-runs the closure. Under the old `o.Version++` the replay failed the
	// CAS against the mutated local copy.
	var emitCalls atomic.Int32
	injectAbort := func(buf outbox.TxnBuffer) error {
		if emitCalls.Add(1) == 1 {
			return status.Error(codes.Aborted, "injected replay")
		}
		return nil
	}

	if err := repo.UpdateOrder(ctx, current, nil, injectAbort); err != nil {
		t.Fatalf("UpdateOrder with replayed closure must succeed, got: %v", err)
	}
	if emitCalls.Load() < 2 {
		t.Fatalf("expected closure replay (>=2 emit calls), got %d", emitCalls.Load())
	}

	row, err := client.Single().ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"Version", "Status"})
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var version int64
	var statusVal string
	if err := row.Columns(&version, &statusVal); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if version != 2 {
		t.Errorf("want Version=2 after single update, got %d", version)
	}
	if statusVal != string(StatusCompleted) {
		t.Errorf("want Status=%s, got %s", StatusCompleted, statusVal)
	}

	// A genuinely stale caller struct must still fail the CAS cleanly.
	stale := current
	stale.Version = 1
	stale.Status = StatusCancelled
	err = repo.UpdateOrder(ctx, stale, nil, nil)
	if err == nil || !containsAll(err.Error(), "optimistic concurrency conflict", "expected 1", "got 2") {
		t.Fatalf("stale update must report caller version in conflict, got: %v", err)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
