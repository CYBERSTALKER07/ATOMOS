package analytics

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"
)

func TestNormalizeGraphQueryRequest_Defaults(t *testing.T) {
	input := GraphQueryRequest{QueryMode: string(GraphQueryModeLaneCapacity)}

	normalized, err := normalizeGraphQueryRequest(input)
	if err != nil {
		t.Fatalf("normalizeGraphQueryRequest() error = %v", err)
	}

	if normalized.QueryMode != GraphQueryModeLaneCapacity {
		t.Fatalf("query mode = %q, want %q", normalized.QueryMode, GraphQueryModeLaneCapacity)
	}
	if normalized.PageSize != defaultGraphQueryPageSize {
		t.Fatalf("page size = %d, want %d", normalized.PageSize, defaultGraphQueryPageSize)
	}
	if normalized.Offset != 0 {
		t.Fatalf("offset = %d, want 0", normalized.Offset)
	}
	if !normalized.To.After(normalized.From) {
		t.Fatalf("time window invalid: from=%v to=%v", normalized.From, normalized.To)
	}
}

func TestNormalizeGraphQueryRequest_InvalidMode(t *testing.T) {
	_, err := normalizeGraphQueryRequest(GraphQueryRequest{QueryMode: "UNKNOWN_MODE"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestNormalizeGraphQueryRequest_PageSizeLimit(t *testing.T) {
	_, err := normalizeGraphQueryRequest(GraphQueryRequest{
		QueryMode: string(GraphQueryModeSupplierTier),
		PageSize:  maxGraphQueryPageSize + 1,
	})
	if err == nil {
		t.Fatal("expected error for oversized page_size")
	}
}

func TestHandleGraphQuery_MethodNotAllowed(t *testing.T) {
	handler := HandleGraphQuery(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/analytics/graph/query", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGraphQuery_UnauthorizedWithoutClaims(t *testing.T) {
	handler := HandleGraphQuery(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/analytics/graph/query", bytes.NewBufferString(`{"query_mode":"LANE_CAPACITY"}`))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleGraphQuery_InvalidRequestWithClaims(t *testing.T) {
	handler := HandleGraphQuery(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/analytics/graph/query", bytes.NewBufferString(`{"query_mode":"INVALID"}`))
	req = withSupplierClaims(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func withSupplierClaims(r *http.Request) *http.Request {
	claims := &auth.PegasusClaims{
		UserID:     "supplier-user",
		SupplierID: "supplier-1",
		Role:       "SUPPLIER",
	}
	ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
	return r.WithContext(ctx)
}
