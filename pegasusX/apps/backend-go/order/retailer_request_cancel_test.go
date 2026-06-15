package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleRetailerRequestCancel_TransitionsInTransitOrder(t *testing.T) {
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			RetailerID: "ret-1",
			SupplierID: "sup-1",
			Status:     StatusInTransit,
			Version:    1,
		},
	}
	svc := &Service{
		repo: repo,
		now:  func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) },
	}

	body := `{"order_id":"ord-1","retailer_id":"ret-1","reason":"changed mind"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orders/request-cancel", strings.NewReader(body))
	req.Header.Set("Idempotency-Key", "retailer-request-cancel:ord-1")
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1",
		Role:    auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()

	svc.HandleRetailerRequestCancel(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if repo.captured.Status != StatusCancelRequested {
		t.Fatalf("status=%s want CANCEL_REQUESTED", repo.captured.Status)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "cancel_requested" {
		t.Fatalf("response=%v", resp)
	}
}

func TestHandleRetailerRequestCancel_RejectsPendingOrder(t *testing.T) {
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			RetailerID: "ret-1",
			Status:     StatusPending,
		},
	}
	svc := &Service{repo: repo, now: time.Now}

	req := httptest.NewRequest(http.MethodPost, "/v1/orders/request-cancel", strings.NewReader(`{"order_id":"ord-1","retailer_id":"ret-1"}`))
	req = req.WithContext(auth.WithClaims(context.Background(), auth.Claims{Subject: "ret-1", Role: auth.RoleRetailer}))
	rr := httptest.NewRecorder()

	svc.HandleRetailerRequestCancel(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
