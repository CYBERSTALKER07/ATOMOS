package order

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
)

func TestRecordDoorstepShortfallInTxn_FailsClosedOnSpannerTxnWhenHookMissing(t *testing.T) {
	svc := &Service{now: time.Now}

	err := svc.recordDoorstepShortfallInTxn(context.Background(), &spanner.ReadWriteTransaction{}, "ord-shortfall-1", "sup-1", "drv-1", "UZS", 50000, 40000)
	if err == nil || !strings.Contains(err.Error(), "doorstep shortfall journal hook required") {
		t.Fatalf("expected fail-closed error when txn != nil and hook nil, got %v", err)
	}

	// No shortfall should not fail even with nil hook.
	if err := svc.recordDoorstepShortfallInTxn(context.Background(), &spanner.ReadWriteTransaction{}, "ord-shortfall-1", "sup-1", "drv-1", "UZS", 50000, 50000); err != nil {
		t.Fatalf("expected nil when received covers total, got %v", err)
	}

	// Nil txn (in-memory test repo) remains allowed when hook is unset.
	if err := svc.recordDoorstepShortfallInTxn(context.Background(), nil, "ord-shortfall-1", "sup-1", "drv-1", "UZS", 50000, 40000); err != nil {
		t.Fatalf("expected nil for in-memory nil txn, got %v", err)
	}

	// Configured hook error propagates.
	sentinel := errors.New("shortfall ledger write failed")
	svc.SetDoorstepShortfall(func(_ context.Context, _ *spanner.ReadWriteTransaction, _, _, _, _ string, _, _ int64, _ time.Time) error {
		return sentinel
	})
	if err := svc.recordDoorstepShortfallInTxn(context.Background(), &spanner.ReadWriteTransaction{}, "ord-shortfall-1", "sup-1", "drv-1", "UZS", 50000, 40000); !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
