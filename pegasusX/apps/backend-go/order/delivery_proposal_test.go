package order

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestWarehouseProposeDeliveryDate_SetsPendingWarehouse(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	requested := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	proposed := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		order: Order{
			OrderID:               "ord_prop",
			SupplierID:            "sup_1",
			WarehouseID:           "wh_1",
			RetailerID:            "ret_1",
			Status:                StatusScheduled,
			Source:                OrderSourceManualPreorder,
			RequestedDeliveryDate: &requested,
			Currency:              "UZS",
		},
		found: true,
	}
	svc := newTestService(repo, now)
	ops := &auth.WarehouseOps{WarehouseID: "wh_1", SupplierID: "sup_1"}
	resp, err := svc.WarehouseProposeDeliveryDate(context.Background(), ops, "ord_prop", ProposeDeliveryDateRequest{
		ProposedDeliveryDate: proposed.Format(time.RFC3339),
		Reason:               "Capacity shift",
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if repo.captured.ConfirmationStatus != ConfirmationStatusPendingWarehouse {
		t.Fatalf("confirmation_status=%s want PENDING_WAREHOUSE", repo.captured.ConfirmationStatus)
	}
	if repo.captured.ProposedDeliveryDate == nil || !repo.captured.ProposedDeliveryDate.Equal(proposed.UTC()) {
		t.Fatalf("proposed date not stored")
	}
	if !repo.captured.RequestedDeliveryDate.Equal(requested.UTC()) {
		t.Fatalf("requested date should remain until accept")
	}
	if resp.PreorderBadge != "REVIEW_DELIVERY" {
		t.Fatalf("badge=%q want REVIEW_DELIVERY", resp.PreorderBadge)
	}
	if len(repo.lastEvents) == 0 || !strings.Contains(string(repo.lastEvents[len(repo.lastEvents)-1].Payload), events.EventPreOrderDateProposed) {
		t.Fatalf("expected %s event", events.EventPreOrderDateProposed)
	}
}

func TestAcceptDeliveryProposal_AppliesDate(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	requested := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	proposed := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	proposalAt := now.Add(-time.Hour)
	repo := &testRepo{
		order: Order{
			OrderID:                "ord_acc",
			SupplierID:             "sup_1",
			WarehouseID:            "wh_1",
			RetailerID:             "ret_1",
			Status:                 StatusScheduled,
			Source:                 OrderSourceManualPreorder,
			ConfirmationStatus:     ConfirmationStatusPendingWarehouse,
			RequestedDeliveryDate:  &requested,
			ProposedDeliveryDate:   &proposed,
			DeliveryProposalAt:     &proposalAt,
			DeliveryProposalReason: "Capacity shift",
			Currency:               "UZS",
		},
		found: true,
	}
	svc := newTestService(repo, now)
	resp, err := svc.AcceptDeliveryProposal(context.Background(), "ret_1", AcceptDeliveryProposalRequest{OrderID: "ord_acc"})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if repo.captured.ConfirmationStatus != ConfirmationStatusConfirmed {
		t.Fatalf("confirmation_status=%s want CONFIRMED", repo.captured.ConfirmationStatus)
	}
	if repo.captured.ProposedDeliveryDate != nil {
		t.Fatal("proposal columns should be cleared")
	}
	if !repo.captured.RequestedDeliveryDate.Equal(proposed.UTC()) {
		t.Fatalf("requested date not applied")
	}
	if resp.PreorderBadge == "REVIEW_DELIVERY" {
		t.Fatal("badge should not remain REVIEW_DELIVERY after accept")
	}
}

func TestRejectDeliveryProposal_CancelsOrder(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	requested := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	proposed := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		order: Order{
			OrderID:                "ord_rej",
			SupplierID:             "sup_1",
			WarehouseID:            "wh_1",
			RetailerID:             "ret_1",
			Status:                 StatusScheduled,
			Source:                 OrderSourceManualPreorder,
			ConfirmationStatus:     ConfirmationStatusPendingWarehouse,
			RequestedDeliveryDate:  &requested,
			ProposedDeliveryDate:   &proposed,
			Currency:               "UZS",
		},
		found: true,
	}
	svc := newTestService(repo, now)
	_, err := svc.RejectDeliveryProposal(context.Background(), "ret_1", RejectDeliveryProposalRequest{
		OrderID: "ord_rej",
		Reason:  "Cannot accept new date",
	})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if repo.captured.Status != StatusCancelled {
		t.Fatalf("status=%s want CANCELLED", repo.captured.Status)
	}
	if repo.captured.ProposedDeliveryDate != nil {
		t.Fatal("proposal columns should be cleared on cancel")
	}
}

func TestWarehouseProposeDeliveryDate_BlockedInsideT2Lock(t *testing.T) {
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	requested := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	proposed := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		order: Order{
			OrderID:               "ord_lock",
			SupplierID:            "sup_1",
			WarehouseID:           "wh_1",
			RetailerID:            "ret_1",
			Status:                StatusScheduled,
			Source:                OrderSourceManualPreorder,
			RequestedDeliveryDate: &requested,
			Currency:              "UZS",
		},
		found: true,
	}
	svc := newTestService(repo, now)
	ops := &auth.WarehouseOps{WarehouseID: "wh_1", SupplierID: "sup_1"}
	_, err := svc.WarehouseProposeDeliveryDate(context.Background(), ops, "ord_lock", ProposeDeliveryDateRequest{
		ProposedDeliveryDate: proposed.Format(time.RFC3339),
		Reason:               "Too late",
	})
	if err == nil {
		t.Fatal("expected T-2 lock error")
	}
}
