package gl

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireGLMutationPerm_FailsClosedWithoutClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/gl/entries", nil)
	rr := httptest.NewRecorder()
	if requireGLMutationPerm(rr, req, "gl.post") {
		t.Fatal("expected fail-closed false when claims missing")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}
