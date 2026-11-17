package payment

import (
	"context"
	"testing"

	"cloud.google.com/go/spanner"
)

func TestPaymeMerchantStore_SpannerWiringAndNilFallback(t *testing.T) {
	t.Parallel()

	// Nil Spanner client uses in-memory store for unit tests.
	svc := NewService(ServiceConfig{})
	if _, ok := svc.ensurePaymeTx().(*paymeMerchantMemory); !ok {
		t.Fatalf("expected *paymeMerchantMemory when SpannerClient is nil, got %T", svc.ensurePaymeTx())
	}

	// Non-nil Spanner client switches to PaymeMerchantTransactions Spanner store.
	fakeClient := &spanner.Client{}
	svcWithSpanner := NewService(ServiceConfig{SpannerClient: fakeClient})
	if _, ok := svcWithSpanner.ensurePaymeTx().(*paymeMerchantSpanner); !ok {
		t.Fatalf("expected *paymeMerchantSpanner when SpannerClient is set, got %T", svcWithSpanner.ensurePaymeTx())
	}

	// Backward-compatible setter switches between Spanner and nil-DB memory.
	svc.SetSpannerClient(fakeClient)
	if _, ok := svc.ensurePaymeTx().(*paymeMerchantSpanner); !ok {
		t.Fatalf("expected *paymeMerchantSpanner after SetSpannerClient(non-nil), got %T", svc.ensurePaymeTx())
	}
	svc.SetSpannerClient(nil)
	if _, ok := svc.ensurePaymeTx().(*paymeMerchantMemory); !ok {
		t.Fatalf("expected *paymeMerchantMemory after SetSpannerClient(nil), got %T", svc.ensurePaymeTx())
	}
}

func TestPaymeMerchantSpanner_NilClientErrorsAndMutationMapping(t *testing.T) {
	t.Parallel()

	var store *paymeMerchantSpanner
	if _, _, err := store.snapshot(context.Background(), "pm-1"); err == nil {
		t.Fatal("expected error for nil paymeMerchantSpanner snapshot")
	}
	if _, _, err := store.snapshotByOrder(context.Background(), "ord-1"); err == nil {
		t.Fatal("expected error for nil paymeMerchantSpanner snapshotByOrder")
	}
	if err := store.put(context.Background(), paymeMerchantTx{PaymeID: "pm-1"}); err == nil {
		t.Fatal("expected error for nil paymeMerchantSpanner put")
	}
	if _, err := store.statement(context.Background(), 0, 100); err == nil {
		t.Fatal("expected error for nil paymeMerchantSpanner statement")
	}

	// Verify target type defaults cleanly to ORDER when omitted and preserves AR_ALLOCATION.
	if got := normalizePaymeTargetType(""); got != paymeTargetTypeOrder {
		t.Fatalf("normalizePaymeTargetType(\"\") = %q want %q", got, paymeTargetTypeOrder)
	}
	if got := normalizePaymeTargetType("AR_ALLOCATION"); got != paymeTargetTypeARAllocation {
		t.Fatalf("normalizePaymeTargetType(AR_ALLOCATION) = %q", got)
	}
	reason := 3
	mut := paymeMerchantMutation(paymeMerchantTx{
		PaymeID:     "pm-tx-1",
		OrderID:     "ord-1",
		AmountMinor: 250000,
		State:       paymeStateCancelled,
		CreateTime:  1700000000000,
		CancelTime:  1700000005000,
		Reason:      &reason,
		MerchantTx:  "sess-1",
	})
	if mut == nil {
		t.Fatal("expected non-nil Spanner mutation")
	}
}

func TestPaymeMerchantMemory_StatementAndTargetType(t *testing.T) {
	t.Parallel()

	mem := newPaymeMerchantMemory()
	ctx := context.Background()
	if err := mem.put(ctx, paymeMerchantTx{
		PaymeID:     "pm-1",
		OrderID:     "ord-1",
		AmountMinor: 100000,
		State:       paymeStateCreated,
		CreateTime:  1000,
		MerchantTx:  "sess-1",
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, ok, err := mem.snapshotByOrder(ctx, "ord-1")
	if err != nil || !ok {
		t.Fatalf("snapshotByOrder ok=%v err=%v", ok, err)
	}
	if got.TargetType != paymeTargetTypeOrder {
		t.Fatalf("TargetType = %q want %q", got.TargetType, paymeTargetTypeOrder)
	}
	items, err := mem.statement(ctx, 500, 1500)
	if err != nil {
		t.Fatalf("statement: %v", err)
	}
	if len(items) != 1 || items[0].PaymeID != "pm-1" {
		t.Fatalf("statement items = %+v", items)
	}
	empty, err := mem.statement(ctx, 2000, 3000)
	if err != nil || len(empty) != 0 {
		t.Fatalf("out-of-window statement = %+v err=%v", empty, err)
	}
}
