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
	if repo.bufferedEvents != 2 {
		t.Fatalf("buffered events = %d, want 2", repo.bufferedEvents)
	}
	event := repo.lastEvents[len(repo.lastEvents)-1]
	var payload events.CreditDeliveryEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal event payload: %v", err)
	}
	if payload.Type != events.EventCreditDeliveryMarked {
		t.Fatalf("event type = %s, want %s", payload.Type, events.EventCreditDeliveryMarked)
	}
	if payload.OrderID != "ord-1" {
		t.Fatalf("event order_id = %s, want ord-1", payload.OrderID)
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
