package supplier

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSupplierFleetLiveMap_methodNotAllowed(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/fleet/live-map", nil)
	rec := httptest.NewRecorder()
	svc.HandleSupplierFleetLiveMap(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rec.Code)
	}
}
