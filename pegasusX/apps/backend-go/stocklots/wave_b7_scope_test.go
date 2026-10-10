package stocklots

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// B7 WH-P0-4: assertResourceWarehouse membership helper.
func TestAssertResourceWarehouse_Mismatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/lots/x?warehouse_id=wh-a", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:         auth.RoleWarehouse,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-a",
	}))
	// Ops scope empty → falls back to query warehouse_id
	rr := httptest.NewRecorder()
	if assertResourceWarehouse(rr, req, "wh-b") {
		t.Fatal("expected false for mismatch")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rr.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "warehouse_scope_forbidden" {
		t.Fatalf("error=%q", body["error"])
	}
}

func TestAssertResourceWarehouse_Match(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?warehouse_id=wh-a", nil)
	rr := httptest.NewRecorder()
	if !assertResourceWarehouse(rr, req, "wh-a") {
		t.Fatalf("expected true body=%s", rr.Body.String())
	}
}

func TestAssertResourceWarehouse_MissingScope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	if assertResourceWarehouse(rr, req, "wh-a") {
		t.Fatal("expected false without warehouse scope")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rr.Code)
	}
}
