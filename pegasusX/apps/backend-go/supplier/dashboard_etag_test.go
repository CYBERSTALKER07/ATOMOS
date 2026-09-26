package supplier

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func TestHandleDashboard_ETag304(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		SupplierID: "sup-1",
		Cache:      cache.New(cache.NewInMemoryBackend(), nil),
		Now:        func() time.Time { return now },
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/dashboard", nil)
	rr := httptest.NewRecorder()
	svc.HandleDashboard(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", rr.Code, rr.Body.String())
	}
	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/supplier/dashboard", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	svc.HandleDashboard(rr2, req2)
	if rr2.Code != http.StatusNotModified {
		t.Fatalf("second status=%d want 304 body=%s", rr2.Code, rr2.Body.String())
	}
}
