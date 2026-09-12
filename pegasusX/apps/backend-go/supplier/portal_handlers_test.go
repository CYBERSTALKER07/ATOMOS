package supplier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleInventoryAuditGone(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/inventory/audit", nil)
	rr := httptest.NewRecorder()
	svc.HandleInventoryAudit(rr, req)

	if rr.Code != http.StatusGone {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusGone, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if body["error"] != "audit_unwired" {
		t.Fatalf("error=%v want audit_unwired body=%s", body["error"], rr.Body.String())
	}
	if _, hasEntries := body["entries"]; hasEntries {
		t.Fatal("theatre entries[] must not appear")
	}
}

func TestHandleInventoryAuditMethodNotAllowed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/audit", nil)
	rr := httptest.NewRecorder()
	svc.HandleInventoryAudit(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}
}
