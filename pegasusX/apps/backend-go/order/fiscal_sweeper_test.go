package order

import (
	"context"
	"testing"
	"time"
)

func TestCalculateFiscalBackoff_Bounds(t *testing.T) {
	for attempt := int64(1); attempt <= 15; attempt++ {
		backoff := CalculateFiscalBackoff(attempt)
		if backoff < 0 {
			t.Fatalf("attempt %d: negative backoff %v", attempt, backoff)
		}
		if backoff > 300*time.Second {
			t.Fatalf("attempt %d: backoff %v exceeded 300s max cap", attempt, backoff)
		}
	}
}

func TestFiscalSweeper_AutoRetryWhenBackoffElapsed(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusFiscalFailed)
	o.OrderID = "ord-auto-retry-test"
	o.FiscalStatus = FiscalStatusFailed
	o.UpdatedAt = now.Add(-10 * time.Minute) // 10 minutes ago, well past backoff

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)
	svc.SetFiscalProvider(FakeFiscalProvider{})

	// Override backoff function to be deterministic
	origBackoff := FiscalSweeperBackoffFn
	FiscalSweeperBackoffFn = func(attempt int64) time.Duration {
		return 5 * time.Second
	}
	defer func() { FiscalSweeperBackoffFn = origBackoff }()

	// Run sweeper pass
	err := svc.RunFiscalSweeperOnce(context.Background())
	if err != nil {
		t.Fatalf("RunFiscalSweeperOnce error: %v", err)
	}

	// Verify order was transitioned to FISCALIZING (retry initiated)
	if repo.captured.Status != StatusFiscalizing {
		t.Fatalf("expected order status FISCALIZING after sweeper retry, got %s", repo.captured.Status)
	}
	if repo.captured.FiscalStatus != FiscalStatusPending {
		t.Fatalf("expected fiscal status PENDING after sweeper retry, got %s", repo.captured.FiscalStatus)
	}

	// Simulate worker processing the retry successfully
	repo.order.Status = StatusFiscalizing
	repo.order.OrderID = "ord-ok-after-retry"
	err = svc.ApplyFiscalWorkerResult(context.Background(), "ord-ok-after-retry", repo.captured.LatestFiscalAttemptID)
	if err != nil {
		t.Fatalf("worker result: %v", err)
	}
	if repo.captured.Status != StatusCompleted {
		t.Fatalf("expected COMPLETED after successful retry worker, got %s", repo.captured.Status)
	}
}

func TestFiscalSweeper_SkipWhenBackoffNotElapsed(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusFiscalFailed)
	o.OrderID = "ord-backoff-waiting"
	o.FiscalStatus = FiscalStatusFailed
	o.UpdatedAt = now.Add(-2 * time.Second) // Updated 2s ago

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	// Backoff requires 60s
	origBackoff := FiscalSweeperBackoffFn
	FiscalSweeperBackoffFn = func(attempt int64) time.Duration {
		return 60 * time.Second
	}
	defer func() { FiscalSweeperBackoffFn = origBackoff }()

	err := svc.RunFiscalSweeperOnce(context.Background())
	if err != nil {
		t.Fatalf("RunFiscalSweeperOnce error: %v", err)
	}

	// Verify order was NOT updated / retried yet
	if repo.captured.Status == StatusFiscalizing {
		t.Fatalf("order should not have been retried before backoff elapsed")
	}
}

func TestFiscalSweeper_ExhaustedAttemptsQuarantinedToDLQ(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusFiscalFailed)
	o.OrderID = "ord-dlq-quarantine"
	o.FiscalStatus = FiscalStatusFailed
	o.UpdatedAt = now.Add(-10 * time.Minute)

	repo := &testRepo{
		found: true,
		order: o,
		fiscalAttempts: map[string]FiscalReceiptRow{
			"att-1": {OrderID: o.OrderID, Status: FiscalAttemptFailed},
			"att-2": {OrderID: o.OrderID, Status: FiscalAttemptFailed},
			"att-3": {OrderID: o.OrderID, Status: FiscalAttemptFailed},
		},
	}
	svc := newTestService(repo, now)

	err := svc.RunFiscalSweeperOnce(context.Background())
	if err != nil {
		t.Fatalf("RunFiscalSweeperOnce error: %v", err)
	}

	// Verify order was quarantined
	if repo.captured.FiscalStatus != FiscalStatusDeadLetter {
		t.Fatalf("expected FiscalStatus DEAD_LETTER, got %s", repo.captured.FiscalStatus)
	}

	// Second pass should skip already-dead-lettered order
	repo.order = repo.captured
	repo.captured = Order{}
	err = svc.RunFiscalSweeperOnce(context.Background())
	if err != nil {
		t.Fatalf("second pass error: %v", err)
	}
	if repo.captured.OrderID != "" {
		t.Fatalf("expected second pass to skip quarantined order, but captured %s", repo.captured.OrderID)
	}
}
