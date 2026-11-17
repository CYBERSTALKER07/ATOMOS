package proposals

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireNonAuditorNonAgent_FailsClosedWithoutClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/proposals/p1/approve", nil)
	rr := httptest.NewRecorder()
	if requireNonAuditorNonAgent(rr, req, "") {
		t.Fatal("expected fail-closed false when claims missing")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}
