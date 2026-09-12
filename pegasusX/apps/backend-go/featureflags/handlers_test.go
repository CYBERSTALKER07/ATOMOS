package featureflags

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/platformadmin"
)

func TestFlagHandlersWritePlatformAdminAudit(t *testing.T) {
	t.Setenv("AUTO_ORDER_PLACE_ENABLED", "false")
	flagRepo := NewMemoryRepository()
	platRepo := platformadmin.NewMemoryRepository()
	platSvc := platformadmin.NewService(platRepo)
	h := &Handlers{Svc: NewService(flagRepo), Audit: platSvc}

	r := chi.NewRouter()
	RegisterRoutes(r, h)

	putBody, _ := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER",
		"tenant_id":   "s1",
		"enabled":     true,
		"reason":      "pilot",
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/platform-admin/flags/AUTO_ORDER_PLACE_ENABLED", bytes.NewReader(putBody))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "setter-a",
		Role:    auth.RolePlatformAdmin,
	}))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("set status=%d body=%s", rr.Code, rr.Body.String())
	}

	approveBody, _ := json.Marshal(map[string]any{"tenant_type": "SUPPLIER", "tenant_id": "s1"})
	req2 := httptest.NewRequest(http.MethodPost, "/v1/platform-admin/flags/AUTO_ORDER_PLACE_ENABLED/approve", bytes.NewReader(approveBody))
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), auth.Claims{
		Subject: "approver-b",
		Role:    auth.RolePlatformAdmin,
	}))
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", rr2.Code, rr2.Body.String())
	}

	rows, err := platSvc.ListAudit(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected set+approve audit rows, got %d", len(rows))
	}
	actions := map[string]string{}
	for _, row := range rows {
		actions[row.Action] = row.ActorSubject
	}
	if actions[auditActionOverrideSet] != "setter-a" {
		t.Fatalf("set audit actor=%q", actions[auditActionOverrideSet])
	}
	if actions[AuditActionAutoOrderPlace] != "approver-b" {
		t.Fatalf("approve audit actor=%q action map=%v", actions[AuditActionAutoOrderPlace], actions)
	}
}

func TestFlagSetRejectsEmptyActorAsUnknownOnlyWithoutClaims(t *testing.T) {
	// Middleware would normally block; exercise ActorFromContext fallback path
	// by calling the handler with no claims (role gate bypassed).
	flagRepo := NewMemoryRepository()
	platRepo := platformadmin.NewMemoryRepository()
	h := &Handlers{Svc: NewService(flagRepo), Audit: platformadmin.NewService(platRepo)}

	body, _ := json.Marshal(map[string]any{
		"tenant_type": "SUPPLIER", "tenant_id": "s1", "enabled": false, "reason": "x",
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/platform-admin/flags/PROMO_RULES_ENABLED", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	// Direct handler call (no RequireRole) — actor must be "unknown", still audited.
	req = muxFlag(req, "PROMO_RULES_ENABLED")
	h.HandleSetOverride(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	rows, _ := platformadmin.NewService(platRepo).ListAudit(context.Background(), 5)
	if len(rows) != 1 || rows[0].ActorSubject != "unknown" {
		t.Fatalf("expected unknown actor audit, got %+v", rows)
	}
}

func TestHandleListPending_EmptyAndLive(t *testing.T) {
	flagRepo := NewMemoryRepository()
	h := &Handlers{Svc: NewService(flagRepo)}
	r := chi.NewRouter()
	RegisterRoutes(r, h)

	req := httptest.NewRequest(http.MethodGet, "/v1/platform-admin/flags/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pa", Role: auth.RolePlatformAdmin,
	}))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("empty status=%d body=%s", rr.Code, rr.Body.String())
	}
	var empty map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if empty["count"] != float64(0) || empty["available"] != true {
		t.Fatalf("empty pending must be available zero, got %+v", empty)
	}

	if err := h.Svc.SetOverride(context.Background(), Override{
		FlagKey: "AUTO_ORDER_PLACE_ENABLED", TenantType: "SUPPLIER", TenantID: "s1",
		Enabled: true, Reason: "pilot", UpdatedBy: "setter",
	}); err != nil {
		t.Fatal(err)
	}
	if err := h.Svc.SetOverride(context.Background(), Override{
		FlagKey: "PROMO_RULES_ENABLED", TenantType: "SUPPLIER", TenantID: "s1",
		Enabled: true, UpdatedBy: "setter",
	}); err != nil {
		t.Fatal(err)
	}

	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req)
	var live struct {
		Items []Override `json:"items"`
		Count int        `json:"count"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	if live.Count != 1 || len(live.Items) != 1 || live.Items[0].FlagKey != "AUTO_ORDER_PLACE_ENABLED" {
		t.Fatalf("want only pending money flag, got %+v", live)
	}
}

func muxFlag(req *http.Request, flag string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("flagKey", flag)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
