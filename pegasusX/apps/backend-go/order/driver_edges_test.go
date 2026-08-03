package order

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestHandleCreditDeliveryDriverSuccess(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	order := deliveryTestOrder(StatusArrived)
	repo := &testRepo{found: true, order: order}
	svc := newTestService(repo, now)

	body := `{"order_id":"ord-1","photo_proof_url":"https://proof.example.com/photo.jpg"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/credit-delivery", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), driverClaims()))
	rr := httptest.NewRecorder()

	svc.HandleCreditDelivery(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if repo.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", repo.updateCalls)
	}
	if repo.captured.Status != StatusDeliveredOnCredit {
		t.Fatalf("captured status = %s, want %s", repo.captured.Status, StatusDeliveredOnCredit)
	}
	// ORDER_STATUS_CHANGED + CREDIT_DELIVERY_MARKED + CREDIT_LEAVE
	if repo.bufferedEvents != 3 {
		t.Fatalf("buffered events = %d, want 3", repo.bufferedEvents)
	}
	foundMarked := false
	for _, event := range repo.lastEvents {
		var base struct {
			Type    string `json:"type"`
			OrderID string `json:"order_id"`
		}
		if err := json.Unmarshal(event.Payload, &base); err != nil {
			t.Fatalf("unmarshal event payload: %v", err)
		}
		if base.Type == events.EventCreditDeliveryMarked {
			foundMarked = true
			if base.OrderID != "ord-1" {
				t.Fatalf("event order_id = %s, want ord-1", base.OrderID)
			}
		}
	}
	if !foundMarked {
		t.Fatalf("missing %s among emitted events", events.EventCreditDeliveryMarked)
	}
}

func TestHandleCreditDeliveryRejectsNonDriver(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc := newTestService(&testRepo{}, now)

	body := `{"order_id":"ord-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/credit-delivery", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()

	svc.HandleCreditDelivery(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestHandleCreditDeliveryRejectsMissingOrderID(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc := newTestService(&testRepo{}, now)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/credit-delivery", strings.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), driverClaims()))
	rr := httptest.NewRecorder()

	svc.HandleCreditDelivery(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCreditDeliveryRejectsWrongMethod(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc := newTestService(&testRepo{}, now)

	req := httptest.NewRequest(http.MethodGet, "/v1/delivery/credit-delivery", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), driverClaims()))
	rr := httptest.NewRecorder()

	svc.HandleCreditDelivery(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
