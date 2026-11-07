package ar

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleRunDunningOnce_PlatformAdminAllowed(t *testing.T) {
	// Worker with nil deps: when flags off, returns skipped without RunOnce.
	w := &DunningWorker{}
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/ar/dunning/run-once", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "pa-1", Role: auth.RolePlatformAdmin,
	}))
	rr := httptest.NewRecorder()
	w.HandleRunDunningOnce(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("PLATFORM_ADMIN must not be 403; body=%s", rr.Body.String())
	}
	// Flags typically off in unit test → 200 skipped, or 500 if RunOnce nil — never 403 for role.
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRunDunningOnce_RetailerForbidden(t *testing.T) {
	w := &DunningWorker{}
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/ar/dunning/run-once", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	rr := httptest.NewRecorder()
	w.HandleRunDunningOnce(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rr.Code)
	}
}
