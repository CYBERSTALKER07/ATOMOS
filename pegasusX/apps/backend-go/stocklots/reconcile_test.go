package stocklots

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepairInventoryV2_RejectsNilClientAndEmptyScope(t *testing.T) {
	if _, err := RepairInventoryV2(t.Context(), nil, "sup-1", "wh-1"); err == nil || !strings.Contains(err.Error(), "spanner unavailable") {
		t.Fatalf("expected spanner unavailable error, got %v", err)
	}
	if _, err := ReconcileInventoryV2(t.Context(), nil, "sup-1", "wh-1"); err == nil || !strings.Contains(err.Error(), "spanner unavailable") {
		t.Fatalf("expected ReconcileInventoryV2 spanner unavailable error, got %v", err)
	}
}

func TestHandleRepairInventoryV2_UnavailableWhenSpannerNil(t *testing.T) {
	h := &Handler{SupplierID: "sup-1"}
	req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/inventory-reconcile/repair?warehouse_id=wh-1", nil)
	rec := httptest.NewRecorder()
	h.HandleRepairInventoryV2(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when spanner nil, got %d", rec.Code)
	}
}
