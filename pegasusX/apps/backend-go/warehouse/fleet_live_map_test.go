package warehouse

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleWarehouseFleetLiveMap_methodNotAllowed(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/fleet/live-map", nil)
	rec := httptest.NewRecorder()
	svc.HandleWarehouseFleetLiveMap(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusMethodNotAllowed)
	}
}
