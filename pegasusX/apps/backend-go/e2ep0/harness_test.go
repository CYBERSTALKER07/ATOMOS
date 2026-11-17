package e2ep0

import (
	"net/http"
	"testing"
)

func TestHarness_SanityCheck(t *testing.T) {
	h := NewTestHarness(t)

	// Test probe endpoint without auth -> 401
	resp := h.Client.Get("/v1/test/auth-probe", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without token, got %d (body: %s)", resp.StatusCode, resp.String())
	}

	// Test probe endpoint with Admin token -> 200
	tok := h.IssueAdminToken(DefaultSupplierA, "test-admin")
	respAuth := h.Client.Get("/v1/test/auth-probe", tok)
	if respAuth.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK with admin token, got %d (body: %s)", respAuth.StatusCode, respAuth.String())
	}

	var data map[string]any
	if err := respAuth.JSON(&data); err != nil {
		t.Fatalf("failed to parse probe JSON: %v", err)
	}
	if data["authenticated"] != true || data["subject"] != "test-admin" || data["supplier_id"] != DefaultSupplierA {
		t.Fatalf("unexpected probe data: %+v", data)
	}
}
