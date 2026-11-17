package pulse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type failSupplierAct struct{}

func (failSupplierAct) ListRecentSupplierOrders(context.Context, string, int) ([]SupplierActivityOrder, error) {
	return nil, errors.New("spanner_unavailable")
}

func TestListForRecipient_TransitionsError(t *testing.T) {
	svc := NewService(Config{})
	svc.transitionLister = func(context.Context, string, string, int) ([]Event, error) {
		return nil, errors.New("spanner_unavailable")
	}
	_, err := svc.ListForRecipient(context.Background(), "ret-1", "RETAILER", "ret-1", 10)
	if err == nil || !strings.Contains(err.Error(), "pulse transitions") {
		t.Fatalf("err=%v", err)
	}
}

func TestListForRecipient_SupplierActivityError(t *testing.T) {
	svc := NewService(Config{SupplierAct: failSupplierAct{}})
	_, err := svc.ListForRecipient(context.Background(), "sup-1", "ADMIN", "sup-1", 10)
	if err == nil || !strings.Contains(err.Error(), "pulse supplier activity") {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleSupplierPulse_ActivityError(t *testing.T) {
	h := &Handlers{Service: NewService(Config{SupplierAct: failSupplierAct{}})}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/pulse", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	h.HandleSupplierPulse(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "pulse_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["events"]; ok {
		t.Fatal("failed pulse must not return events[]")
	}
}
