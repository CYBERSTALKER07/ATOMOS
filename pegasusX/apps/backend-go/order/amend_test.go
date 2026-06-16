package order

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestResolveAmendQuantitiesInfersRejectedQty(t *testing.T) {
	accepted, rejected, err := resolveAmendQuantities(5, 3, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accepted != 3 || rejected != 2 {
		t.Fatalf("accepted/rejected = %d/%d, want 3/2", accepted, rejected)
	}
}

func TestValidateAmendReasonRequiresCustomReasonForOther(t *testing.T) {
	if err := validateAmendReason("OTHER", ""); err == nil {
		t.Fatal("expected custom_reason error")
	}
	if err := validateAmendReason("OTHER", "box crushed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceAmendOrderAdjustsTotalAndPersistsReturns(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusArrived)}
	seq := 0
	svc := NewService(ServiceConfig{
		Repo: repo,
		Now:  func() time.Time { return now },
		NewID: func() string {
			seq++
			return fmt.Sprintf("ret-%d", seq)
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	resp, err := svc.AmendOrder(context.Background(), driverClaims(), AmendOrderRequest{
		OrderID: "ord-1",
		Items: []AmendItemRequest{{
			ProductID:   "sku-1",
			AcceptedQty: 2,
			RejectedQty: 1,
			Reason:      "DAMAGED",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected amend error: %v", err)
	}
	if resp.AdjustedTotal != 1000 {
		t.Fatalf("adjusted total = %d, want 1000", resp.AdjustedTotal)
	}
	if repo.captured.TotalMinor != 1000 {
		t.Fatalf("captured total = %d, want 1000", repo.captured.TotalMinor)
	}
	if repo.captured.OriginalTotalMinor != 1500 {
		t.Fatalf("original total = %d, want 1500", repo.captured.OriginalTotalMinor)
	}
	if len(repo.captured.PendingSupplierReturns) != 1 {
		t.Fatalf("pending returns = %d, want 1", len(repo.captured.PendingSupplierReturns))
	}
	ret := repo.captured.PendingSupplierReturns[0]
	if ret.SKU != "sku-1" || ret.RejectedQty != 1 || ret.Reason != "DAMAGED" {
		t.Fatalf("unexpected return row: %+v", ret)
	}
}

func TestServiceAmendOrderRequiresReasonWhenRejecting(t *testing.T) {
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusInTransit)}
	svc := newTestService(repo, time.Now().UTC())

	_, err := svc.AmendOrder(context.Background(), driverClaims(), AmendOrderRequest{
		OrderID: "ord-1",
		Items: []AmendItemRequest{{
			ProductID:   "sku-1",
			AcceptedQty: 2,
			RejectedQty: 1,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "reason required") {
		t.Fatalf("expected reason required error, got %v", err)
	}
}

func TestServiceAmendOrderRejectsInvalidState(t *testing.T) {
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusAwaitingPayment)}
	svc := newTestService(repo, time.Now().UTC())

	_, err := svc.AmendOrder(context.Background(), driverClaims(), AmendOrderRequest{
		OrderID: "ord-1",
		Items: []AmendItemRequest{{
			ProductID:   "sku-1",
			AcceptedQty: 2,
			RejectedQty: 1,
			Reason:      "MISSING",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be amended") {
		t.Fatalf("expected state error, got %v", err)
	}
}

func TestPaymentRequiredDataIncludesOriginalAmountAfterAmend(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	svc := newTestService(&testRepo{}, now)

	orderRecord := deliveryTestOrder(StatusAwaitingPayment)
	orderRecord.TotalMinor = 1000
	orderRecord.OriginalTotalMinor = 1500
	orderRecord.UpdatedAt = now

	data := svc.paymentRequiredData(context.Background(), orderRecord)
	if data["amount_minor"] != int64(1000) {
		t.Fatalf("amount_minor = %v, want 1000", data["amount_minor"])
	}
	if data["original_amount"] != int64(1500) {
		t.Fatalf("original_amount = %v, want 1500", data["original_amount"])
	}
}

func TestConfirmOffloadEmitsOriginalAmountForAmendedOrder(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	order := deliveryTestOrder(StatusArrived)
	order.TotalMinor = 1000
	order.OriginalTotalMinor = 1500
	repo := &testRepo{found: true, order: order}
	svc := newTestService(repo, now)

	_, err := svc.ConfirmOffload(context.Background(), driverClaims(), ConfirmOffloadRequest{OrderID: "ord-1"})
	if err != nil {
		t.Fatalf("unexpected confirm offload error: %v", err)
	}

	var paymentPayload map[string]any
	for _, evt := range repo.lastEvents {
		var payload map[string]any
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			continue
		}
		if payload["type"] == events.EventPaymentRequired {
			paymentPayload = payload
			break
		}
	}
	if paymentPayload == nil {
		t.Fatal("PAYMENT_REQUIRED event not found")
	}
	if paymentPayload["original_amount"] != float64(1500) && paymentPayload["original_amount"] != int64(1500) {
		t.Fatalf("original_amount = %v, want 1500", paymentPayload["original_amount"])
	}
}
