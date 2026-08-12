package payout

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

// seedSupplierProfile inserts the bank details a dispatch requires.
func seedSupplierProfile(t *testing.T, ctx context.Context, client *spanner.Client, supplierID string) {
	t.Helper()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("SupplierProfiles", map[string]any{
			"SupplierId":    supplierID,
			"ContactName":   "Test Supplier LLC",
			"BankName":      "Test Bank",
			"AccountHolder": "Test Supplier LLC",
			"AccountNumber": "20208000123456789001",
			"RegisteredAt":  spanner.CommitTimestamp,
			"UpdatedAt":     spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed supplier profile: %v", err)
	}
}

// fakeLiveRail records a dispatch and returns a reference without moving money.
type fakeLiveRail struct {
	submitted bool
	ref       string
}

func (f *fakeLiveRail) Name() string { return "fake-live" }
func (f *fakeLiveRail) IsLive() bool { return true }
func (f *fakeLiveRail) Submit(_ context.Context, b Batch, _ SupplierBankDetails, live bool) (string, error) {
	if !live {
		return "", nil
	}
	f.submitted = true
	return f.ref, nil
}

func TestPayoutRail_DispatchThenSettlementConfirm(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-po-rail-%d", suffix)
	legTime := time.Now().UTC().Add(-2 * time.Minute)

	// Supplier bank details required for dispatch.
	seedSupplierProfile(t, ctx, client, supplierID)
	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-rail-%d", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CARD", "AmountMinor": int64(100000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-rail-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})

	svc := NewService(NewRepository(client))
	rail := &fakeLiveRail{ref: "rail-ref-123"}
	svc.SetRail(rail)

	b, err := svc.GenerateBatch(ctx, supplierID, legTime.Add(-time.Hour), legTime.Add(time.Hour), "admin", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Live dispatch moves DRAFT -> SUBMITTED and records the rail reference.
	b, err = svc.SubmitForDispatch(ctx, b.BatchID, true)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if !rail.submitted {
		t.Fatal("rail.Submit not called on live dispatch")
	}
	if b.Status != StatusSubmitted {
		t.Fatalf("status = %s, want SUBMITTED", b.Status)
	}

	// Settlement webhook flips SUBMITTED -> PAID; replay is a no-op.
	if err := svc.ConfirmSettlement(ctx, b.BatchID, rail.ref); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	got, found, err := svc.repo.Get(ctx, b.BatchID)
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if got.Status != StatusPaid {
		t.Fatalf("status = %s, want PAID", got.Status)
	}
	if got.RailReference != rail.ref {
		t.Fatalf("rail ref = %q, want %q", got.RailReference, rail.ref)
	}
	if err := svc.ConfirmSettlement(ctx, b.BatchID, rail.ref); err != nil {
		t.Fatalf("idempotent confirm: %v", err)
	}
}

// TestPayoutRail_LiveDispatchOnFileRailFailsClosed proves a batch can never be
// stranded in SUBMITTED: live=true on the default (non-live) bank-file rail is
// rejected, leaving the batch DRAFT so it can still be exported and marked paid.
func TestPayoutRail_LiveDispatchOnFileRailFailsClosed(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-po-fc-%d", suffix)
	legTime := time.Now().UTC().Add(-2 * time.Minute)
	seedSupplierProfile(t, ctx, client, supplierID)
	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-fc-%d", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CARD", "AmountMinor": int64(100000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-fc-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})
	svc := NewService(NewRepository(client)) // default BankFileRail (not live)
	b, err := svc.GenerateBatch(ctx, supplierID, legTime.Add(-time.Hour), legTime.Add(time.Hour), "admin", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := svc.SubmitForDispatch(ctx, b.BatchID, true); err == nil {
		t.Fatal("live dispatch on non-live rail must fail closed")
	}
	got, _, _ := svc.repo.Get(ctx, b.BatchID)
	if got.Status == StatusSubmitted {
		t.Fatalf("batch stranded in SUBMITTED with empty rail ref; status = %s", got.Status)
	}
	// Still dispatchable as a file export.
	if _, err := svc.SubmitForDispatch(ctx, b.BatchID, false); err != nil {
		t.Fatalf("file export after fail-closed live attempt: %v", err)
	}
}

func TestPayoutRail_ConfirmBeforeSubmitFails(t *testing.T) {
	ctx := context.Background()
	client := newEmulatorClient(t, ctx)
	defer client.Close()

	suffix := time.Now().UnixNano()
	supplierID := fmt.Sprintf("sup-po-early-%d", suffix)
	legTime := time.Now().UTC().Add(-2 * time.Minute)
	seedLegs(t, ctx, client, supplierID, fmt.Sprintf("ord-po-early-%d", suffix), legTime, []map[string]any{
		{"LegId": "l1", "Method": "CARD", "AmountMinor": int64(100000), "Status": "CAPTURED", "IdempotencyKey": fmt.Sprintf("cap-early-%d", suffix), "CreatedAt": legTime, "CapturedAt": legTime},
	})
	svc := NewService(NewRepository(client))
	b, err := svc.GenerateBatch(ctx, supplierID, legTime.Add(-time.Hour), legTime.Add(time.Hour), "admin", "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := svc.ConfirmSettlement(ctx, b.BatchID, "ref"); err == nil {
		t.Fatal("settlement confirm on DRAFT batch must fail")
	}
}
