package payloaderroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_RegistersPayloaderEndpoints(t *testing.T) {
	r := chi.NewRouter()
	identity := func(next http.HandlerFunc) http.HandlerFunc { return next }

	RegisterRoutes(r, Deps{
		Spanner:    &spanner.Client{},
		ReadRouter: nil,
		Log:        identity,
	})

	expected := map[string]bool{
		http.MethodGet + " /v1/payloader/trucks":              false,
		http.MethodGet + " /v1/payloader/orders":              false,
		http.MethodPost + " /v1/payloader/recommend-reassign": false,
		http.MethodPost + " /v1/payload/seal":                 false,
	}

	if err := chi.Walk(r, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	for key, seen := range expected {
		if !seen {
			t.Fatalf("missing expected route: %s", key)
		}
	}
}

func TestRegisterRoutes_NilPayloaderHubDoesNotMountPayloaderWebSocket(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log:          func(next http.HandlerFunc) http.HandlerFunc { return next },
		PayloaderHub: nil,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/ws/payloader", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRegisterRoutes_PayloadSealMounted(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/payload/seal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
