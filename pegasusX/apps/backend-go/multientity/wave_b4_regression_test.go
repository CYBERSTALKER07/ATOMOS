package multientity

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRejectAuditorOrAgent_FailsClosedWithoutClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/multientity/transfers", nil)
	rr := httptest.NewRecorder()
	if rejectAuditorOrAgent(rr, req, true) {
		t.Fatal("expected fail-closed false when claims missing")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}
