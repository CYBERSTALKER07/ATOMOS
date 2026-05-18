package entityresolution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-go/auth"
)

func TestHandleResolve_UnauthorizedWhenClaimsMissing(t *testing.T) {
	handler := HandleResolve(NewService(fakeRepository{}))
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/entity-resolution/resolve", strings.NewReader(`{"entity_type":"ANY","query":"ord"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	assertErrorCode(t, rr.Body.Bytes(), "unauthorized")
}

func TestHandleResolve_InvalidJSONBody(t *testing.T) {
	handler := HandleResolve(NewService(fakeRepository{}))
	req := newRequestWithSupplierClaims(http.MethodPost, "/v1/supplier/entity-resolution/resolve", `{"entity_type":"ANY"`)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	assertErrorCode(t, rr.Body.Bytes(), "invalid_json_body")
}

func TestHandleResolve_SuccessEnvelope(t *testing.T) {
	svc := NewService(fakeRepository{
		listScoped: func(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error) {
			return []EntityRecord{
				{
					EntityType: EntityTypeOrder,
					EntityID:   "ord-1",
					Label:      "ord-1",
					SearchText: "ord-1 delivered",
				},
			}, nil
		},
	})
	handler := HandleResolve(svc)
	req := newRequestWithSupplierClaims(http.MethodPost, "/v1/supplier/entity-resolution/resolve", `{"entity_type":"ANY","query":"ord"}`)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("status field = %v, want ok", payload["status"])
	}

	result, ok := payload["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("result field missing or malformed")
	}
	resolved, ok := result["resolved"].(map[string]interface{})
	if !ok {
		t.Fatalf("resolved field missing or malformed")
	}
	if resolved["entity_id"] != "ord-1" {
		t.Fatalf("resolved.entity_id = %v, want ord-1", resolved["entity_id"])
	}
}

func TestHandleExplain_NotFoundMapsTo404(t *testing.T) {
	svc := NewService(fakeRepository{
		findExact: func(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error) {
			return nil, nil
		},
	})
	handler := HandleExplain(svc)
	req := newRequestWithSupplierClaims(http.MethodPost, "/v1/supplier/entity-resolution/explain", `{"entity_type":"DRIVER","entity_id":"drv-404"}`)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	assertErrorCode(t, rr.Body.Bytes(), "entity_not_found")
}

func newRequestWithSupplierClaims(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	claims := &auth.PegasusClaims{UserID: "user-1", SupplierID: "sup-1", Role: "ADMIN"}
	ctx := context.WithValue(req.Context(), auth.ClaimsContextKey, claims)
	return req.WithContext(ctx)
}

func assertErrorCode(t *testing.T, body []byte, expected string) {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid json error body: %v", err)
	}
	if payload["error"] != expected {
		t.Fatalf("error = %q, want %q", payload["error"], expected)
	}
}
