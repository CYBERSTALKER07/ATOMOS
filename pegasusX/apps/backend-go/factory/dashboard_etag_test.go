package factory

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func TestHandleDashboard_ETag304(t *testing.T) {
	_ = os.Setenv("FACTORY_PORTAL_SEED", "true")
	now := time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)
	mem := cache.NewInMemoryBackend()
	svc := NewService(ServiceConfig{
		Cache:      cache.New(mem, nil),
		SupplierID: "supplier-test",
		Currency:   "UZS",
		Now:        func() time.Time { return now },
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/factory/dashboard", nil)
	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", rr.Code, rr.Body.String())
	}
	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/factory/dashboard", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	svc.HandleDashboard(rr2, req2)
	if rr2.Code != http.StatusNotModified {
		t.Fatalf("second status=%d want 304 body=%s", rr2.Code, rr2.Body.String())
	}
}
