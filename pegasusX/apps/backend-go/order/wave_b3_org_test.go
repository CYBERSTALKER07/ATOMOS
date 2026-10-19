package order

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// Wave B3 — retailer multi-user org scope + parent-order event constants.

func TestResolveRetailerOrgID_StaffTokenUsesOrg(t *testing.T) {
	c := auth.Claims{
		Subject:       "user-staff-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-org-1",
		RetailerUserID: "user-staff-1",
	}
	if got := auth.ResolveRetailerOrgID(c); got != "ret-org-1" {
		t.Fatalf("ResolveRetailerOrgID = %q, want ret-org-1", got)
	}
	if got := auth.ResolveRetailerUserID(c); got != "user-staff-1" {
		t.Fatalf("ResolveRetailerUserID = %q, want user-staff-1", got)
	}
}

func TestResolveRetailerOrgID_LegacyOwnerFallsBackToSubject(t *testing.T) {
	c := auth.Claims{Subject: "ret-legacy", Role: auth.RoleRetailer}
	if got := auth.ResolveRetailerOrgID(c); got != "ret-legacy" {
		t.Fatalf("legacy ResolveRetailerOrgID = %q, want ret-legacy", got)
	}
}

func TestUpdateStatus_RetailerCancelUsesOrgNotSubject(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-b3-cancel",
			RetailerID: "ret-org-1",
			SupplierID: "sup-1",
			Status:     StatusPending,
			Version:    1,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	svc := newTestService(repo, now)

	// Staff JWT: Subject is person id, Org is order owner.
	staff := auth.Claims{
		Subject:        "user-staff-1",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-org-1",
		RetailerUserID: "user-staff-1",
	}
	resp, err := svc.UpdateStatus(context.Background(), staff, "ord-b3-cancel", UpdateStatusRequest{
		Status: string(StatusCancelled),
		Reason: "changed_mind",
	})
	if err != nil {
		t.Fatalf("staff cancel with matching org: %v", err)
	}
	if resp.Status != StatusCancelled {
		t.Fatalf("status = %s, want CANCELLED", resp.Status)
	}

	// Wrong org (staff of another retailer) must be forbidden even if Subject matches nothing.
	repo.order.Status = StatusPending
	repo.found = true
	wrongOrg := auth.Claims{
		Subject:       "user-staff-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-other",
	}
	_, err = svc.UpdateStatus(context.Background(), wrongOrg, "ord-b3-cancel", UpdateStatusRequest{
		Status: string(StatusCancelled),
	})
	if err == nil {
		t.Fatal("expected ErrOrderForbidden for wrong org")
	}
	if err != ErrOrderForbidden {
		t.Fatalf("err = %v, want ErrOrderForbidden", err)
	}

	// Legacy: Subject alone as org when RetailerOrgID empty.
	repo.order.RetailerID = "ret-legacy-owner"
	repo.order.Status = StatusPending
	legacy := auth.Claims{Subject: "ret-legacy-owner", Role: auth.RoleRetailer}
	if _, err := svc.UpdateStatus(context.Background(), legacy, "ord-b3-cancel", UpdateStatusRequest{
		Status: string(StatusCancelled),
	}); err != nil {
		t.Fatalf("legacy owner cancel: %v", err)
	}
}

func TestReporterAuthorized_RetailerUsesOrg(t *testing.T) {
	o := Order{OrderID: "o1", RetailerID: "ret-org-1", DriverID: "drv-1"}
	staff := auth.Claims{
		Subject:       "user-staff-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-org-1",
	}
	if !reporterAuthorizedForOrder(staff, o) {
		t.Fatal("staff with matching org should be authorized")
	}
	wrong := auth.Claims{
		Subject:       "ret-org-1", // spoof-looking subject
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-other",
	}
	if reporterAuthorizedForOrder(wrong, o) {
		t.Fatal("staff of other org must not authorize via Subject")
	}
}

func TestParentOrderEventConstants(t *testing.T) {
	if events.EventParentOrderCreated != "PARENT_ORDER_CREATED" {
		t.Fatalf("EventParentOrderCreated = %q", events.EventParentOrderCreated)
	}
	if events.EventParentOrderUpdated != "PARENT_ORDER_UPDATED" {
		t.Fatalf("EventParentOrderUpdated = %q", events.EventParentOrderUpdated)
	}
	if events.AggregateParentOrder != "ParentOrder" {
		t.Fatalf("AggregateParentOrder = %q", events.AggregateParentOrder)
	}
}

func TestSelectCashAtDelivery_OrgScopedOwnership(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-b3-cash",
			RetailerID: "ret-org-1",
			SupplierID: "sup-1",
			Status:     StatusArrived,
			TotalMinor: 5000,
			Currency:   "UZS",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	svc := newTestService(repo, now)

	// Org id matches order.
	status, amount, currency, err := svc.SelectCashAtDelivery(context.Background(), "ord-b3-cash", "ret-org-1", "user-staff-1")
	if err != nil {
		t.Fatalf("SelectCashAtDelivery org match: %v", err)
	}
	if status != string(StatusPendingCashCollection) {
		t.Fatalf("status = %q", status)
	}
	if amount != 5000 || currency != "UZS" {
		t.Fatalf("amount/currency = %d %s", amount, currency)
	}

	// Staff user id must not pass as retailer id.
	repo.order.Status = StatusArrived
	_, _, _, err = svc.SelectCashAtDelivery(context.Background(), "ord-b3-cash", "user-staff-1", "user-staff-1")
	if err != ErrOrderForbidden {
		t.Fatalf("staff id as retailer: err = %v, want ErrOrderForbidden", err)
	}
}
