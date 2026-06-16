package supplier

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleReturnsMethodNotAllowed(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/returns", nil)
	rec := httptest.NewRecorder()
	svc.HandleReturns(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleResolveReturnRequiresReturnID(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/returns/resolve", nil)
	rec := httptest.NewRecorder()
	svc.HandleResolveReturn(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
