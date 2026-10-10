package supplier

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestScopedSupplierIDPrefersTenant(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "seed"})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithTenant(req.Context(), auth.TenantContext{SupplierID: "sup-live", Source: "jwt"}))
	if got := svc.ScopedSupplierID(req); got != "sup-live" {
		t.Fatalf("got=%q", got)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := svc.ScopedSupplierID(req2); got != "seed" {
		t.Fatalf("fallback got=%q", got)
	}
}
