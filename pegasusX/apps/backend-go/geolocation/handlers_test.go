package geolocation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

func setupTestServer() (http.Handler, *countingBackend) {
	backend := &countingBackend{data: make(map[string][]byte)}
	c := cache.New(backend, nil)
	svc := NewService("", c)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.With(auth.RequireAnyAuthenticated()).Group(func(gr chi.Router) {
		RegisterRoutes(gr, h)
	})
	return r, backend
}

func TestGeocodeRoutes_UnauthenticatedRejected(t *testing.T) {
	server, _ := setupTestServer()

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/v1/platform/geocode/autocomplete?input=chorsu", ""},
		{"GET", "/v1/platform/geocode/place?place_id=p123", ""},
		{"GET", "/v1/platform/geocode/reverse?lat=41.31&lng=69.28", ""},
		{"POST", "/v1/platform/geocode/forward", `{"address":"Chorsu Bazaar"}`},
	}

	for _, ep := range endpoints {
		var req *http.Request
		if ep.body != "" {
			req = httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		} else {
			req = httptest.NewRequest(ep.method, ep.path, nil)
		}
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("[%s %s] expected status 401 Unauthorized, got %d (body: %s)", ep.method, ep.path, w.Code, w.Body.String())
		}
	}
}

func TestGeocodeRoutes_AuthenticatedAccepted(t *testing.T) {
	server, backend := setupTestServer()

	// Seed cache so uncached external network call is not made
	loc := ResolvedLocation{Address: "Chorsu Bazaar, Tashkent", Lat: 41.32, Lng: 69.23, Formatted: "Chorsu Bazaar, Tashkent"}
	raw, _ := json.Marshal(loc)
	backend.data[forwardCacheKey("uz", "chorsu bazaar")] = raw

	claims := auth.Claims{
		Subject:    "user_123",
		Role:       auth.RoleRetailer,
		MarketCode: "UZ",
	}

	req := httptest.NewRequest("POST", "/v1/platform/geocode/forward", bytes.NewBufferString(`{"address":"Chorsu Bazaar","country":"UZ"}`))
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var resp ResolvedLocation
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Address != loc.Address {
		t.Fatalf("got address %q, want %q", resp.Address, loc.Address)
	}
}
