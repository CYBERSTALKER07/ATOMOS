package partner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// trackingKeyRepo wraps MemoryKeyRepository and records Revoke args.
type trackingKeyRepo struct {
	*MemoryKeyRepository
	lastKeyID, lastTenantType, lastTenantID string
	calls                                   int
}

func (t *trackingKeyRepo) Revoke(ctx context.Context, keyID, tenantType, tenantID string) error {
	t.calls++
	t.lastKeyID = keyID
	t.lastTenantType = tenantType
	t.lastTenantID = tenantID
	// Always succeed for handler-scope tests (no key seed required).
	return nil
}

func TestHandleRevokeKey_PlatformAdminRequiresTenant(t *testing.T) {
	repo := &trackingKeyRepo{MemoryKeyRepository: NewMemoryKeyRepository()}
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys/k1/revoke", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pa-1", Role: auth.RolePlatformAdmin,
	}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyID", "k1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.HandleRevokeKey(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 tenant_id_required body=%s", rr.Code, rr.Body.String())
	}
	if repo.calls != 0 {
		t.Fatal("must not call Revoke without tenant_id")
	}
}

func TestHandleRevokeKey_PlatformAdminQueryTenant(t *testing.T) {
	repo := &trackingKeyRepo{MemoryKeyRepository: NewMemoryKeyRepository()}
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys/k1/revoke?tenant_type=SUPPLIER&tenant_id=sup-1", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pa-1", Role: auth.RolePlatformAdmin,
	}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyID", "k1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.HandleRevokeKey(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if repo.lastTenantID != "sup-1" || repo.lastTenantType != "SUPPLIER" {
		t.Fatalf("tenant=%s/%s", repo.lastTenantType, repo.lastTenantID)
	}
}

func TestHandleRevokeKey_PlatformAdminBodyTenant(t *testing.T) {
	repo := &trackingKeyRepo{MemoryKeyRepository: NewMemoryKeyRepository()}
	h := &Handlers{Svc: NewService(repo, nil, nil, nil, nil)}
	body, _ := json.Marshal(map[string]string{"tenant_type": "SUPPLIER", "tenant_id": "sup-body"})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/partner-keys/k2/revoke", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pa-1", Role: auth.RolePlatformAdmin,
	}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("keyID", "k2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.HandleRevokeKey(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if repo.lastTenantID != "sup-body" {
		t.Fatalf("tenant_id=%q", repo.lastTenantID)
	}
}
