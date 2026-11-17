package payout

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPayout_ClaimDispatchCASAndRevertOnConflict(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-po-cas-%d", suffix)
	legTime := time.Now().UTC().Add(-2 * time.Minute)
	seedSupplierProfile(t, ctx, client, supplierID)
	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-cas-%d", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CARD", "AmountMinor": int64(50000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-cas-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})
	svc := NewService(NewRepository(client))
	batches, err := svc.GenerateBatch(ctx, supplierID, legTime.Add(-time.Hour), legTime.Add(time.Hour), "admin", "")
	if err != nil || len(batches) != 1 {
		t.Fatalf("generate batches=%d err=%v", len(batches), err)
	}
	b := batches[0]

	// Invalid monotonic jump DRAFT -> PAID directly must fail with ErrBatchConflict.
	if err := svc.repo.UpdateStatusRef(ctx, b.BatchID, StatusPaid, "", "ref-invalid"); !errors.Is(err, ErrBatchConflict) {
		t.Fatalf("DRAFT -> PAID err=%v, want ErrBatchConflict", err)
	}

	// First live claim acquires StatusSubmitting CAS lock.
	prev, _, err := svc.repo.ClaimDispatchInTxn(ctx, supplierID, b.BatchID)
	if err != nil || prev != StatusDraft {
		t.Fatalf("first ClaimDispatchInTxn prev=%q err=%v", prev, err)
	}

	// Concurrent second claim while StatusSubmitting must fail with ErrBatchConflict.
	if _, _, err := svc.repo.ClaimDispatchInTxn(ctx, supplierID, b.BatchID); !errors.Is(err, ErrBatchConflict) {
		t.Fatalf("second concurrent ClaimDispatchInTxn err=%v, want ErrBatchConflict", err)
	}

	// Revert restores StatusDraft so batch can be cleanly retried or exported.
	if err := svc.repo.RevertDispatchInTxn(ctx, b.BatchID, prev); err != nil {
		t.Fatalf("RevertDispatchInTxn: %v", err)
	}
	got, found, err := svc.repo.Get(ctx, b.BatchID)
	if err != nil || !found || got.Status != StatusDraft {
		t.Fatalf("after revert: found=%v status=%s err=%v", found, got.Status, err)
	}
}
