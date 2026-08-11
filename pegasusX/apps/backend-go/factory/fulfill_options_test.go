package factory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBuildSupplyFulfillOptionsMemory(t *testing.T) {
	svc := &Service{
		seedSupplierID: "sup-1",
		factoryNodeID:  "fac-1",
	}
	svc.supplyRequests = []SupplyRequest{
		{
			RequestID:   "sr-1",
			WarehouseID: "wh-1",
			Status:      "READY",
		},
	}

	opts, err := svc.buildSupplyFulfillOptionsMemory("sr-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if opts.TransferMode != "TRUCK" {
		t.Fatalf("expected TRUCK got %q", opts.TransferMode)
	}
	if opts.WarehouseName != "wh-1" {
		t.Fatalf("unexpected warehouse name %q", opts.WarehouseName)
	}
}

func TestHandleSupplyRequestFulfillOptions_NotFound(t *testing.T) {
	svc := &Service{seedSupplierID: "sup-1", factoryNodeID: "fac-1"}
	req := httptest.NewRequest(http.MethodGet, "/v1/factory/supply-requests/missing/fulfill-options", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	svc.HandleSupplyRequestFulfillOptions(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rr.Code)
	}
}
