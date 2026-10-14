package order

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestHandleCreate_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{"line_items":[]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1",
		Role:    auth.RoleRetailer,
	}))
	rec := httptest.NewRecorder()
	svc.HandleCreate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	var errResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "idempotency_key_required" {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleCollectCash_RequiresIdempotencyKey(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo: &testRepo{},
		Idem: idempotency.NewInMemoryStore(),
		Log:  slog.Default(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/order/collect-cash", strings.NewReader(`{"order_id":"ord-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "drv-1",
		Role:    auth.RoleDriver,
	}))
	rec := httptest.NewRecorder()
	svc.HandleCollectCash(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}
