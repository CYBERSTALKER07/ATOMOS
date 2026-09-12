package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTenantFromContextRoundTrip(t *testing.T) {
	ctx := WithTenant(httptest.NewRequest(http.MethodGet, "/", nil).Context(), TenantContext{
		SupplierID: "sup-a",
		Source:     "jwt",
	})
	got, ok := TenantFromContext(ctx)
	if !ok {
		t.Fatal("expected tenant")
	}
	if got.SupplierID != "sup-a" || got.Source != "jwt" {
		t.Fatalf("got=%+v", got)
	}
	if _, ok := TenantFromContext(WithTenant(ctx, TenantContext{SupplierID: "  "})); ok {
		t.Fatal("empty supplier must not attach")
	}
}

func TestRequireTenantFailClosed(t *testing.T) {
	h := RequireTenant(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Authenticated without tenant → 401
	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{Subject: "u1", Role: RoleAdmin}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}

	// Authenticated with tenant → 200
	req2 := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	ctx := WithClaims(req2.Context(), Claims{Subject: "u1", Role: RoleAdmin, SupplierID: "sup-1"})
	ctx = WithTenant(ctx, TenantContext{SupplierID: "sup-1", Source: "jwt"})
	req2 = req2.WithContext(ctx)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec2.Code)
	}

	// Unauthenticated → pass
	req3 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("public status=%d want 200", rec3.Code)
	}
}

func TestAttachTenantFromClaims(t *testing.T) {
	h := AttachTenantFromClaims(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tctx, ok := TenantFromContext(r.Context())
		if !ok || tctx.SupplierID != "sup-9" {
			t.Fatalf("tenant=%v ok=%v", tctx, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{Subject: "u", SupplierID: "sup-9"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestPreferTenantSupplierID(t *testing.T) {
	ctx := WithTenant(context.Background(), TenantContext{SupplierID: "sup-ctx", Source: "jwt"})
	if got := PreferTenantSupplierID(ctx, "seed"); got != "sup-ctx" {
		t.Fatalf("got=%q", got)
	}
	if got := PreferTenantSupplierID(context.Background(), "seed"); got != "seed" {
		t.Fatalf("fallback got=%q", got)
	}
	ctxClaims := WithClaims(context.Background(), Claims{Subject: "u", SupplierID: "sup-claim"})
	if got := PreferTenantSupplierID(ctxClaims, "seed"); got != "sup-claim" {
		t.Fatalf("claims got=%q", got)
	}
}

func TestPreferTenantSupplierIDNoSeedWhenEnforced(t *testing.T) {
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	t.Setenv("ALLOW_SEED_FALLBACK", "")
	ctx := WithClaims(context.Background(), Claims{Subject: "u", Role: RoleAdmin})
	if got := PreferTenantSupplierID(ctx, "seed"); got != "" {
		t.Fatalf("authenticated without supplier must not seed, got=%q", got)
	}
	// Unauthenticated + enforced + seed disallowed by default under enforced.
	if got := PreferTenantSupplierID(context.Background(), "seed"); got != "" {
		t.Fatalf("enforced unauthenticated must not seed without ALLOW_SEED_FALLBACK, got=%q", got)
	}
	t.Setenv("ALLOW_SEED_FALLBACK", "true")
	if got := PreferTenantSupplierID(context.Background(), "seed"); got != "seed" {
		t.Fatalf("break-glass seed got=%q", got)
	}
	t.Setenv("TENANT_CONTEXT_ENFORCED", "false")
	t.Setenv("ALLOW_SEED_FALLBACK", "true")
	if got := PreferTenantSupplierID(ctx, "seed"); got != "seed" {
		t.Fatalf("unenforced fallback got=%q", got)
	}
}

func TestTenantContextEnforcedSandboxAlias(t *testing.T) {
	t.Setenv("TENANT_CONTEXT_ENFORCED", "")
	t.Setenv("PEGASUSX_ENV", "sandbox")
	if !TenantContextEnforced() {
		t.Fatal("sandbox must enforce tenant context by default")
	}
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !TenantContextEnforced() {
		t.Fatal("ssmr alias must enforce tenant context by default")
	}
	t.Setenv("PEGASUSX_ENV", "production")
	if !TenantContextEnforced() {
		t.Fatal("production must enforce tenant context by default")
	}
	t.Setenv("PEGASUSX_ENV", "prod")
	if !TenantContextEnforced() {
		t.Fatal("prod alias must enforce tenant context by default")
	}
	t.Setenv("PEGASUSX_ENV", "staging")
	if TenantContextEnforced() {
		t.Fatal("staging must not enforce by default")
	}
	t.Setenv("PEGASUSX_ENV", "")
	if TenantContextEnforced() {
		t.Fatal("local must not enforce by default")
	}
}

func TestSeedFallbackAllowed(t *testing.T) {
	t.Setenv("TENANT_CONTEXT_ENFORCED", "true")
	t.Setenv("ALLOW_SEED_FALLBACK", "")
	if SeedFallbackAllowed() {
		t.Fatal("expected seed disallowed under enforced")
	}
	t.Setenv("ALLOW_SEED_FALLBACK", "true")
	if !SeedFallbackAllowed() {
		t.Fatal("expected break-glass allow")
	}
	t.Setenv("TENANT_CONTEXT_ENFORCED", "false")
	t.Setenv("ALLOW_SEED_FALLBACK", "")
	if !SeedFallbackAllowed() {
		t.Fatal("local/dev default allow")
	}
}

func TestRequireTenantDisabledPasses(t *testing.T) {
	h := RequireTenant(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{Subject: "u1", Role: RoleAdmin}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
}
