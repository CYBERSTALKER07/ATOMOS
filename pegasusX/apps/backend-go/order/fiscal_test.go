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
	res, err := p.CreateReceipt(context.Background(), FiscalCreateRequest{OrderID: "ord-ok", AttemptID: "a2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.FiscalReceiptID == "" || res.FiscalQR == "" {
		t.Fatalf("expected receipt id and qr, got %+v", res)
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

func containsType(payload []byte, eventType string) bool {
	return strings.Contains(string(payload), eventType)
}
