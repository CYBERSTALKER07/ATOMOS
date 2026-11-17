package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

func TestHTTPRouter_MountsGLRoutesWithFailClosedRBAC(t *testing.T) {
	cfg := &bootstrap.Config{
		JWTSecret: "router-gl-test-secret-key-32bytes!",
	}
	app := &bootstrap.App{
		Config: cfg,
		GL:     gl.NewService(nil),
	}

	router := NewHTTPRouter(HTTPRouterParams{
		Config: cfg,
		App:    app,
	})

	// Unauthenticated request to GL endpoint must hit CookieAuth -> 401 (not 404).
	req := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /v1/finance/gl/trial-balance = %d, want 401 (route must be mounted)", rec.Code)
	}

	// Non-admin role (driver) must be rejected with 403 by glroutes RBAC.
	drvTok, err := auth.Issue(auth.Claims{
		Subject:    "drv-1",
		Role:       auth.RoleDriver,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: cfg.JWTSecret})
	if err != nil {
		t.Fatalf("issue driver token: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/v1/finance/gl/trial-balance", nil)
	req2.Header.Set("Authorization", "Bearer "+drvTok)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("driver /v1/finance/gl/trial-balance = %d, want 403", rec2.Code)
	}

	// Mutation route mounted and auth-protected too.
	req3 := httptest.NewRequest(http.MethodPost, "/v1/finance/journal/manual", nil)
	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated POST /v1/finance/journal/manual = %d, want 401", rec3.Code)
	}
	_ = context.Background()
}
