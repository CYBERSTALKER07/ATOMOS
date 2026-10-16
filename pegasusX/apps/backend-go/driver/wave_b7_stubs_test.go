package driver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// B7 D-P0-6 / D-P0-7: depart/return-complete never 200 without wired domain fn.
func TestHandleDriverDepart_Unwired_503(t *testing.T) {
	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	// depart remains nil
	req := httptest.NewRequest(http.MethodPost, "/v1/fleet/driver/depart", strings.NewReader(`{}`))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1", Role: auth.RoleDriver})
	req.Header.Set("Idempotency-Key", "driver-depart:drv-1")
	rr := httptest.NewRecorder()
	svc.HandleDriverDepart(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("depart unwired: status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "depart_unwired" {
		t.Fatalf("error=%q want depart_unwired", body["error"])
	}
}

func TestHandleDriverReturnComplete_Unwired_503(t *testing.T) {
	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	req := httptest.NewRequest(http.MethodPost, "/v1/fleet/driver/return-complete", strings.NewReader(`{}`))
	req = withDriverClaims(req, auth.Claims{Subject: "drv-1", Role: auth.RoleDriver})
	req.Header.Set("Idempotency-Key", "driver-return-complete:drv-1")
	rr := httptest.NewRecorder()
	svc.HandleDriverReturnComplete(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("return-complete unwired: status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "return_complete_unwired" {
		t.Fatalf("error=%q want return_complete_unwired", body["error"])
	}
	// Must not flip availability as success side-effect.
	svc.mu.Lock()
	on, ok := svc.availability["drv-1"]
	svc.mu.Unlock()
	if ok && on {
		t.Fatalf("availability must not be set true on unwired return-complete")
	}
}

func TestDriverShiftMutations_RequireIdempotencyKey(t *testing.T) {
	svc := newDriverTestService(&driverRepoSpy{}, &driverCacheBackendSpy{})
	cases := []struct {
		name string
		path string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"depart", "/v1/fleet/driver/depart", svc.HandleDriverDepart},
		{"return-complete", "/v1/fleet/driver/return-complete", svc.HandleDriverReturnComplete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{}`))
			req = withDriverClaims(req, auth.Claims{Subject: "drv-1", Role: auth.RoleDriver})
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "idempotency_key_required") {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}
