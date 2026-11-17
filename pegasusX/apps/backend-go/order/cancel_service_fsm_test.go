package order

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestCancelOrderWithReason_TerminalAndPostDeliveryRejections(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	invalidStatuses := []Status{
		StatusCompleted,
		StatusFiscalizing,
		StatusDeliveredOnCredit,
	}

	for _, st := range invalidStatuses {
		t.Run(string(st), func(t *testing.T) {
			o := deliveryTestOrder(st)
			o.OrderID = "ord-invalid-cancel-" + string(st)
			repo := &testRepo{
				found: true,
				order: o,
			}
			svc := newTestService(repo, now)

			// 1. cancelOrderWithReason direct call
			_, err := svc.cancelOrderWithReason(context.Background(), &o, "admin-1", "ADMIN", "customer request", "")
			if err == nil {
				t.Fatalf("expected error when cancelling order in state %s, got nil", st)
			}

			// 2. PartnerCancelOrder call
			_, err = svc.PartnerCancelOrder(context.Background(), o.OrderID, "partner request")
			if err == nil {
				t.Fatalf("expected PartnerCancelOrder to fail for state %s, got nil", st)
			}
		})
	}
}

func TestPartnerCancelOrder_CapturesPreviousStatus(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusPending)
	o.OrderID = "ord-partner-cancel-prev"
	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	resp, err := svc.PartnerCancelOrder(context.Background(), o.OrderID, "stockout")
	if err != nil {
		t.Fatalf("PartnerCancelOrder failed: %v", err)
	}

	if resp.PreviousStatus != StatusPending {
		t.Fatalf("expected PreviousStatus to be %s, got %s", StatusPending, resp.PreviousStatus)
	}
	if resp.Status != StatusCancelled {
		t.Fatalf("expected Status to be %s, got %s", StatusCancelled, resp.Status)
	}
}

func TestRequestCancelOrder_AndResumeLifecycle(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusPending)
	o.OrderID = "ord-req-cancel-1"
	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	retailerClaims := auth.Claims{
		Subject:    o.RetailerID,
		Role:       auth.RoleRetailer,
		MarketCode: "UZ",
	}

	// 1. PENDING -> CANCEL_REQUESTED
	resp, err := svc.RequestCancelOrder(context.Background(), retailerClaims, o.OrderID, RequestCancelOrderRequest{
		Reason: "ordered by mistake",
	})
	if err != nil {
		t.Fatalf("RequestCancelOrder failed: %v", err)
	}
	if resp.Status != StatusCancelRequested {
		t.Fatalf("expected status %s, got %s", StatusCancelRequested, resp.Status)
	}

	// Update repo state to CANCEL_REQUESTED
	repo.order.Status = StatusCancelRequested

	// 2. Anti-brick safeguard: resume back to PENDING
	adminClaims := auth.Claims{
		Subject:    "ops-admin",
		Role:       auth.RoleAdmin,
		SupplierID: o.SupplierID,
		MarketCode: "UZ",
	}
	resumeResp, err := svc.UpdateStatus(context.Background(), adminClaims, o.OrderID, UpdateStatusRequest{
		Status: string(StatusPending),
		Reason: "cancellation denied by warehouse",
	})
	if err != nil {
		t.Fatalf("resume to PENDING failed: %v", err)
	}
	if resumeResp.Status != StatusPending {
		t.Fatalf("expected resumed status %s, got %s", StatusPending, resumeResp.Status)
	}

	// 3. Move back to CANCEL_REQUESTED then confirm CANCELLED
	repo.order.Status = StatusPending
	_, err = svc.RequestCancelOrder(context.Background(), retailerClaims, o.OrderID, RequestCancelOrderRequest{Reason: "retry cancel"})
	if err != nil {
		t.Fatalf("second cancel request failed: %v", err)
	}
	repo.order.Status = StatusCancelRequested

	cancelResp, err := svc.UpdateStatus(context.Background(), adminClaims, o.OrderID, UpdateStatusRequest{
		Status: string(StatusCancelled),
		Reason: "cancellation approved",
	})
	if err != nil {
		t.Fatalf("transition from CANCEL_REQUESTED to CANCELLED failed: %v", err)
	}
	if cancelResp.Status != StatusCancelled {
		t.Fatalf("expected final status %s, got %s", StatusCancelled, cancelResp.Status)
	}
}

func TestHandleRequestCancel_HTTP(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusPending)
	o.OrderID = "ord-http-cancel-1"
	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	retailerClaims := auth.Claims{
		Subject:    o.RetailerID,
		Role:       auth.RoleRetailer,
		MarketCode: "UZ",
	}

	body, _ := json.Marshal(RequestCancelOrderRequest{Reason: "no longer needed"})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/"+o.OrderID+"/cancel-request", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), retailerClaims))

	// Mount router to bind chi URLParam
	r := chi.NewRouter()
	r.Post("/v1/order/{orderID}/cancel-request", svc.HandleRequestCancel)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp UpdateStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != StatusCancelRequested {
		t.Fatalf("expected status %s, got %s", StatusCancelRequested, resp.Status)
	}
}
