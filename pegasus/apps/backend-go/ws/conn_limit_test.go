package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnforceWSConnectionLimits_UserCap(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/ws/retailer", nil)
	if EnforceWSConnectionLimits(w, r, "retailer", "ret-1", defaultWSMaxPerUser) {
		t.Fatal("expected user cap rejection")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
}

func TestEnforceWSConnectionLimits_AllowsUnderCap(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/ws/retailer", nil)
	if !EnforceWSConnectionLimits(w, r, "retailer", "ret-1", 0) {
		t.Fatal("expected connection allowed without redis")
	}
}
