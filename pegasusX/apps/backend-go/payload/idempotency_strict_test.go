package payload

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuardIdempotencyStrict_RequiresKey(t *testing.T) {
	s := &Service{}
	req := httptest.NewRequest(http.MethodPost, "/v1/payload/seal", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	if !s.guardIdempotencyStrict(rec, req, []byte(`{}`)) {
		t.Fatal("missing key should stop the handler")
	}
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "idempotency_key_required") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
