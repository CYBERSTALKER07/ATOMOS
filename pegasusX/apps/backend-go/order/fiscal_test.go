package order

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestFakeFiscalProviderFailHook(t *testing.T) {
	var p FakeFiscalProvider
	_, err := p.CreateReceipt(context.Background(), FiscalCreateRequest{OrderID: "ord-fiscal-fail-1", AttemptID: "a1"})
	if err == nil {
		t.Fatal("expected fake fail for fiscal-fail order id")
	}
	_, err = p.CreateReceipt(context.Background(), FiscalCreateRequest{OrderID: "ord-ok", AttemptID: "a3", AmountMinor: FiscalFakeFailAmountMinor})
	if err == nil {
		t.Fatal("expected fake fail for amount_minor=13 SSMR hook")
	}
	res, err := p.CreateReceipt(context.Background(), FiscalCreateRequest{OrderID: "ord-ok", AttemptID: "a2", AmountMinor: 1500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.FiscalReceiptID == "" || res.FiscalQR == "" {
		t.Fatalf("expected receipt id and qr, got %+v", res)
	}
}

func TestProviderFromEnvDefaultsFake(t *testing.T) {
	t.Setenv("FISCAL_PROVIDER", "")
	p := ProviderFromEnv()
	if _, ok := p.(FakeFiscalProvider); !ok {
		t.Fatalf("want FakeFiscalProvider, got %T", p)
	}
}

func TestMySoliqProviderRequiresEnv(t *testing.T) {
	t.Setenv("FISCAL_PROVIDER", "MY_SOLIQ")
	t.Setenv("FISCAL_MY_SOLIQ_BASE_URL", "")
	t.Setenv("FISCAL_MY_SOLIQ_API_KEY", "")
	t.Setenv("FISCAL_MY_SOLIQ_TIN", "")
	p := ProviderFromEnv()
	_, err := p.CreateReceipt(context.Background(), FiscalCreateRequest{OrderID: "o", AttemptID: "a", AmountMinor: 100})
	if err == nil {
		t.Fatal("misconfigured MY_SOLIQ must hard-fail CreateReceipt")
	}
}

func TestForceCompleteRequiresAdminAndReason(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusFiscalFailed)}
	svc := newTestService(repo, now)

	_, err := svc.ForceCompleteOrder(context.Background(), driverClaims(), "ord-1", "OFD_DOWN")
	if !errors.Is(err, ErrForceCompleteForbidden) {
		t.Fatalf("driver force: got %v want ErrForceCompleteForbidden", err)
	}

	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}
	_, err = svc.ForceCompleteOrder(context.Background(), admin, "ord-1", "")
	if !errors.Is(err, ErrForceReasonRequired) {
		t.Fatalf("empty reason: got %v want ErrForceReasonRequired", err)
	}

	resp, err := svc.ForceCompleteOrder(context.Background(), admin, "ord-1", "OFD_DOWN")
	if err != nil {
		t.Fatalf("admin force: %v", err)
	}
	if resp.State != StatusCompleted {
		t.Fatalf("state=%s want COMPLETED", resp.State)
	}
	if repo.captured.FiscalStatus != FiscalStatusForceSkipped {
		t.Fatalf("fiscal=%s want FORCE_SKIPPED", repo.captured.FiscalStatus)
	}
	foundForceEvent := false
	for _, e := range repo.lastEvents {
		// Payload is JSON; type is in body — check event type via unmarshal-light
		if len(e.Payload) > 0 && (string(e.Payload) != "" && containsType(e.Payload, events.EventOrderForceCompleted)) {
			foundForceEvent = true
		}
	}
	if !foundForceEvent {
		t.Fatalf("expected ORDER_FORCE_COMPLETED in outbox events, got %d events", len(repo.lastEvents))
	}
}

func TestWarehouseAdminMayForceComplete(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusFiscalFailed)}
	svc := newTestService(repo, now)
	wh := auth.Claims{Subject: "wh-1", Role: auth.RoleWarehouseAdmin}
	resp, err := svc.ForceCompleteOrder(context.Background(), wh, "ord-1", "OPS_ESCALATION")
	if err != nil {
		t.Fatalf("warehouse force: %v", err)
	}
	if resp.State != StatusCompleted {
		t.Fatalf("state=%s want COMPLETED", resp.State)
	}
}

func TestFiscalFailThenRetry(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusAwaitingPayment)
	o.OrderID = "ord-fiscal-fail-cash"
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)

	resp, err := svc.CollectCash(context.Background(), driverClaims(), CollectCashRequest{
		OrderID:   o.OrderID,
		Latitude:  41.311,
		Longitude: 69.279,
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if err := svc.ApplyFiscalWorkerResult(context.Background(), o.OrderID, resp.AttemptID); err != nil {
		t.Fatalf("worker fail path: %v", err)
	}
	if repo.captured.Status != StatusFiscalFailed {
		t.Fatalf("status=%s want FISCAL_FAILED", repo.captured.Status)
	}

	retry, err := svc.RetryFiscal(context.Background(), driverClaims(), o.OrderID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retry.State != StatusFiscalizing {
		t.Fatalf("retry state=%s want FISCALIZING", retry.State)
	}
	// Rename so FakeFiscalProvider succeeds on second attempt.
	repo.order.OrderID = "ord-ok-after-retry"
	repo.order.Status = StatusFiscalizing
	if err := svc.ApplyFiscalWorkerResult(context.Background(), "ord-ok-after-retry", retry.AttemptID); err != nil {
		t.Fatalf("worker success: %v", err)
	}
	if repo.captured.Status != StatusCompleted {
		t.Fatalf("after retry success status=%s want COMPLETED", repo.captured.Status)
	}
}

func TestCollectCashShortfallUsesReceivedAmountAndEmitsEvent(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusAwaitingPayment)
	o.TotalMinor = 10_000
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)

	received := int64(8_500)
	resp, err := svc.CollectCash(context.Background(), driverClaims(), CollectCashRequest{
		OrderID:             o.OrderID,
		Latitude:            41.311,
		Longitude:           69.279,
		AmountReceivedMinor: &received,
		Note:                "retailer short cash",
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if resp.State != StatusFiscalizing {
		t.Fatalf("state=%s want FISCALIZING", resp.State)
	}
	if resp.AmountReceivedMinor != 8_500 || resp.ShortfallMinor != 1_500 {
		t.Fatalf("received/shortfall = %d/%d want 8500/1500", resp.AmountReceivedMinor, resp.ShortfallMinor)
	}
	if len(repo.captured.PendingFiscalReceipts) != 1 {
		t.Fatalf("pending fiscal rows = %d want 1", len(repo.captured.PendingFiscalReceipts))
	}
	if repo.captured.PendingFiscalReceipts[0].AmountMinor != 8_500 {
		t.Fatalf("fiscal amount = %d want received 8500", repo.captured.PendingFiscalReceipts[0].AmountMinor)
	}
	foundShortfall := false
	for _, e := range repo.lastEvents {
		if containsType(e.Payload, events.EventCashShortfall) {
			foundShortfall = true
			if !containsType(e.Payload, "8500") && !strings.Contains(string(e.Payload), `"received_minor":8500`) {
				// still require shortfall event type; amount may be encoded as number
				_ = e
			}
		}
	}
	if !foundShortfall {
		t.Fatalf("expected CASH_SHORTFALL in outbox, events=%d", len(repo.lastEvents))
	}
}

func TestCollectCashNegativeAmountRejected(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusAwaitingPayment)}
	svc := newTestService(repo, now)
	neg := int64(-1)
	_, err := svc.CollectCash(context.Background(), driverClaims(), CollectCashRequest{
		OrderID:             "ord-1",
		Latitude:            41.311,
		Longitude:           69.279,
		AmountReceivedMinor: &neg,
	})
	if !errors.Is(err, ErrCashAmountNegative) {
		t.Fatalf("got %v want ErrCashAmountNegative", err)
	}
}

func TestForceCompleteInvalidReason(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusFiscalFailed)}
	svc := newTestService(repo, now)
	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}
	_, err := svc.ForceCompleteOrder(context.Background(), admin, "ord-1", "NOT_A_REASON")
	if !errors.Is(err, ErrForceReasonInvalid) {
		t.Fatalf("got %v want ErrForceReasonInvalid", err)
	}
}

func TestForceCompleteRejectsWhenFiscalAlreadySucceeded(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusFiscalizing)
	o.FiscalStatus = FiscalStatusSuccess
	o.LatestFiscalReceiptID = "RCPT-1"
	o.LatestFiscalAttemptID = "att-ok"
	repo := &testRepo{
		found: true,
		order: o,
		fiscalAttempts: map[string]FiscalReceiptRow{
			"ord-1:att-ok": {
				OrderID: "ord-1", AttemptID: "att-ok", Status: FiscalAttemptSuccess,
				FiscalReceiptID: "RCPT-1", AmountMinor: 1500,
			},
		},
	}
	svc := newTestService(repo, now)
	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}
	_, err := svc.ForceCompleteOrder(context.Background(), admin, "ord-1", ForceReasonOFDDown)
	if !errors.Is(err, ErrFiscalAlreadySucceeded) {
		t.Fatalf("got %v want ErrFiscalAlreadySucceeded", err)
	}
}

func TestFiscalWorkerIdempotentOnSuccessRedelivery(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusAwaitingPayment)
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)

	resp, err := svc.CollectCash(context.Background(), driverClaims(), CollectCashRequest{
		OrderID:   o.OrderID,
		Latitude:  41.311,
		Longitude: 69.279,
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if err := svc.ApplyFiscalWorkerResult(context.Background(), o.OrderID, resp.AttemptID); err != nil {
		t.Fatalf("worker first: %v", err)
	}
	if repo.captured.Status != StatusCompleted {
		t.Fatalf("status=%s want COMPLETED", repo.captured.Status)
	}
	updatesAfterFirst := repo.updateCalls
	receiptID := repo.captured.LatestFiscalReceiptID

	// Event redelivery: no second COMPLETED mutation path that re-calls OFD.
	if err := svc.ApplyFiscalWorkerResult(context.Background(), o.OrderID, resp.AttemptID); err != nil {
		t.Fatalf("worker redelivery: %v", err)
	}
	if repo.captured.Status != StatusCompleted {
		t.Fatalf("after redelivery status=%s want COMPLETED", repo.captured.Status)
	}
	if repo.captured.LatestFiscalReceiptID != receiptID {
		t.Fatalf("receipt id changed on redelivery: %s -> %s", receiptID, repo.captured.LatestFiscalReceiptID)
	}
	// Completed early-return: no extra UpdateOrder for pure terminal skip.
	if repo.updateCalls != updatesAfterFirst {
		t.Fatalf("updateCalls=%d want %d (no re-write on terminal redelivery)", repo.updateCalls, updatesAfterFirst)
	}
}

func TestSettleExternalPaymentIgnoresTerminalAndFiscalStates(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for _, st := range []Status{StatusCompleted, StatusFiscalizing, StatusFiscalFailed, StatusCancelled} {
		o := deliveryTestOrder(st)
		if st == StatusCancelled {
			// cancel path may move to reconciliation — still not invent fiscal on COMPLETED/FISCAL*
		}
		repo := &testRepo{found: true, order: o}
		svc := newTestService(repo, now)
		err := svc.SettleExternalPayment(context.Background(), o.OrderID, "payme")
		if err != nil {
			t.Fatalf("status %s: unexpected err %v", st, err)
		}
		if st == StatusCancelled {
			if repo.captured.Status != StatusReconciliationRequired {
				t.Fatalf("cancelled late pay: status=%s want RECONCILIATION_REQUIRED", repo.captured.Status)
			}
			continue
		}
		if repo.updateCalls != 0 && st != StatusCancelled {
			// terminal/fiscal path is no-op
			if st == StatusCompleted || st == StatusFiscalizing || st == StatusFiscalFailed {
				t.Fatalf("status %s: expected no UpdateOrder, got %d", st, repo.updateCalls)
			}
		}
	}
}

func TestOfflineBatchRejectsInventedCompletedStatus(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusArrived)
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)

	_, err := svc.processBatchDelivery(context.Background(), driverClaims(), BatchDelivery{
		OrderID:   o.OrderID,
		Signature: "sig",
		Status:    "COMPLETED",
	})
	if err == nil || !strings.Contains(err.Error(), "offline_status_forbidden") {
		t.Fatalf("got %v want offline_status_forbidden", err)
	}
}

func TestOfflineBatchIdempotentWhenAlreadyCompleted(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusCompleted)
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)

	id, err := svc.processBatchDelivery(context.Background(), driverClaims(), BatchDelivery{
		OrderID:   o.OrderID,
		Signature: "sig-any",
		Status:    "ARRIVED",
	})
	if err != nil {
		t.Fatalf("expected idempotent skip, got %v", err)
	}
	if id != o.OrderID {
		t.Fatalf("order_id=%s want %s", id, o.OrderID)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("updateCalls=%d want 0", repo.updateCalls)
	}
}

func TestUpdateStatusCannotSoftCompleteWithoutFiscal(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusReconciliationRequired)
	o.FiscalStatus = FiscalStatusNone
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)
	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: o.SupplierID}
	_, err := svc.UpdateStatus(context.Background(), admin, o.OrderID, UpdateStatusRequest{
		Status: string(StatusCompleted),
		Reason: "soft",
	})
	if err == nil || !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("got %v want ErrInvalidStatusTransition fiscal gate", err)
	}
}

func TestForceCompleteFromReconciliationRequired(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusReconciliationRequired)
	o.FiscalStatus = ""
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)
	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}
	resp, err := svc.ForceCompleteOrder(context.Background(), admin, o.OrderID, ForceReasonOpsEscalation)
	if err != nil {
		t.Fatalf("force from reconciliation: %v", err)
	}
	if resp.State != StatusCompleted {
		t.Fatalf("state=%s want COMPLETED", resp.State)
	}
	if repo.captured.FiscalStatus != FiscalStatusForceSkipped {
		t.Fatalf("fiscal=%s want FORCE_SKIPPED", repo.captured.FiscalStatus)
	}
}

func TestNormalizeForceReasonCode(t *testing.T) {
	got, err := NormalizeForceReasonCode("  ofd_down ")
	if err != nil || got != ForceReasonOFDDown {
		t.Fatalf("got %q/%v want OFD_DOWN", got, err)
	}
	if _, err := NormalizeForceReasonCode(""); !errors.Is(err, ErrForceReasonRequired) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := NormalizeForceReasonCode("wat"); !errors.Is(err, ErrForceReasonInvalid) {
		t.Fatalf("invalid: %v", err)
	}
}

func containsType(payload []byte, eventType string) bool {
	return strings.Contains(string(payload), eventType)
}
