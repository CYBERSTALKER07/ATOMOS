package supplier

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestResolveSupplierDispatchWarehouseID_QueryWinsOverBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/dispatch/execute?warehouse_id=wh-query", bytes.NewBufferString(`{"warehouse_id":"wh-body"}`))
	got := resolveSupplierDispatchWarehouseID(req)
	if got != "wh-query" {
		t.Fatalf("warehouse_id = %q want wh-query", got)
	}
}

func TestResolveSupplierDispatchWarehouseID_BodyFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/dispatch/execute", bytes.NewBufferString(`{"warehouse_id":"wh-body","mode":"AUTO"}`))
	got := resolveSupplierDispatchWarehouseID(req)
	if got != "wh-body" {
		t.Fatalf("warehouse_id = %q want wh-body", got)
	}
}

func TestResolveSupplierDispatchWarehouseID_JWTScopeWins(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/dispatch/execute?warehouse_id=wh-query", bytes.NewBufferString(`{"warehouse_id":"wh-body"}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.WarehouseScopeKey, &auth.WarehouseScope{
		SupplierID:  "sup-1",
		WarehouseID: "wh-jwt",
	}))
	got := resolveSupplierDispatchWarehouseID(req)
	if got != "wh-jwt" {
		t.Fatalf("warehouse_id = %q want wh-jwt", got)
	}
}

func TestResolveSupplierDispatchWarehouseID_EmptyWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/dispatch/preview", nil)
	got := resolveSupplierDispatchWarehouseID(req)
	if got != "" {
		t.Fatalf("warehouse_id = %q want empty", got)
	}
}
