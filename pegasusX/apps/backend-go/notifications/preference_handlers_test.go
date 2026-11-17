package notifications

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestPatchPreferencesUnavailableWithoutStore(t *testing.T) {
	h := &PreferenceHandlers{}
	req := httptest.NewRequest(http.MethodPatch, "/v1/user/notification-preferences", strings.NewReader(`{"preferences":[{"event_type":"ORDER","channel":"push"}]}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	rr := httptest.NewRecorder()
	h.HandlePatchPreferences(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rr.Code, rr.Body.String())
	}
}
