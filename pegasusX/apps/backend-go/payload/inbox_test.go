package payload

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleUserNotifications_NilStoreUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: NewInMemoryRepository()})
	req := httptest.NewRequest(http.MethodGet, "/v1/user/notifications", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pay-1", Role: auth.RolePayload, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleUserNotifications(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_unavailable" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["notifications"]; ok {
		t.Fatal("nil store must not return notifications[]")
	}
}

func TestHandleMarkNotificationsRead_NilStoreUnavailable(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: NewInMemoryRepository()})
	req := httptest.NewRequest(http.MethodPost, "/v1/user/notifications/read",
		strings.NewReader(`{"mark_all":true}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pay-1", Role: auth.RolePayload, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleMarkNotificationsRead(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "inbox_unavailable" || payload["status"] == "ok" {
		t.Fatalf("payload=%v", payload)
	}
}
