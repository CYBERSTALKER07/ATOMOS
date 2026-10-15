package order

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRejectAIOrder_UsesActualStatusAndGraph(t *testing.T) {
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:            "ord-ai",
			RetailerID:         "ret-1",
			SupplierID:         "sup-1",
			Source:             OrderSourceAIPreorder,
			ConfirmationStatus: ConfirmationStatusPending,
			Status:             StatusScheduled,
		},
	}
	svc := newTestService(repo, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))

	if _, err := svc.RejectAIOrder(context.Background(), "ret-1", RejectAIOrderRequest{OrderID: "ord-ai", Reason: "no"}); err != nil {
		t.Fatalf("reject scheduled ai order: %v", err)
	}
	if repo.captured.Status != StatusCancelled {
		t.Fatalf("status = %s", repo.captured.Status)
	}
	if len(repo.lastEvents) != 1 || !strings.Contains(string(repo.lastEvents[0].Payload), `"previous_status":"SCHEDULED"`) {
		t.Fatalf("payload = %s", repo.lastEvents[0].Payload)
	}

	repo.order = Order{
		OrderID:            "ord-ai",
		RetailerID:         "ret-1",
		SupplierID:         "sup-1",
		Source:             OrderSourceAIPreorder,
		ConfirmationStatus: ConfirmationStatusPending,
		Status:             StatusCompleted,
	}
	repo.updateCalls = 0
	if _, err := svc.RejectAIOrder(context.Background(), "ret-1", RejectAIOrderRequest{OrderID: "ord-ai"}); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("completed reject err = %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("updateCalls = %d", repo.updateCalls)
	}
}

func TestFlagPaymentAfterCancel_UsesGraph(t *testing.T) {
	repo := &testRepo{}
	svc := newTestService(repo, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))

	if err := svc.flagPaymentAfterCancel(context.Background(), Order{OrderID: "ord-1", Status: StatusCompleted}, "stripe"); err != nil {
		t.Fatalf("non-cancel: %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("updateCalls = %d", repo.updateCalls)
	}

	if err := svc.flagPaymentAfterCancel(context.Background(), Order{
		OrderID:    "ord-1",
		Status:     StatusCancelled,
		SupplierID: "sup-1",
		RetailerID: "ret-1",
	}, "stripe"); err != nil {
		t.Fatalf("cancel flag: %v", err)
	}
	if repo.captured.Status != StatusReconciliationRequired {
		t.Fatalf("status = %s", repo.captured.Status)
	}
}
