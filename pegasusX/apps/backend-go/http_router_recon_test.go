package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
)

func TestHTTPRouter_MountsReconRoutesWithFailClosedRBAC(t *testing.T) {
	cfg := &bootstrap.Config{
		JWTSecret: "router-recon-test-secret-32bytes!",
	}
	app := &bootstrap.App{
		Config: cfg,
		Recon:  recon.NewServiceWithRepository(recon.NewMemoryRepository()),
	}

	router := NewHTTPRouter(HTTPRouterParams{
		Config: cfg,
		App:    app,
	})

	// Unauthenticated request -> 401 (routes are mounted).
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/finance/recon/runs"},
		{http.MethodPost, "/v1/finance/recon/run"},
		{http.MethodGet, "/v1/finance/recon/breaks"},
		{http.MethodPatch, "/v1/finance/recon/breaks/brk-1"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s %s = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}

	// Non-admin role (driver) -> 403.
	drvTok, err := auth.Issue(auth.Claims{
		Subject:    "drv-1",
		Role:       auth.RoleDriver,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: cfg.JWTSecret})
	if err != nil {
		t.Fatalf("issue driver token: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/v1/finance/recon/breaks", nil)
	req2.Header.Set("Authorization", "Bearer "+drvTok)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("driver GET /v1/finance/recon/breaks = %d, want 403", rec2.Code)
	}
}
