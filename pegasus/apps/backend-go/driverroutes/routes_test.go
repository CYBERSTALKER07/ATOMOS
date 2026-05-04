package driverroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_LegacyFleetManifestAliasMounted(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/fleet/manifest?date=2026-05-04", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRegisterRoutes_NilDriverHubDoesNotMountDriverWebSocket(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log:       func(next http.HandlerFunc) http.HandlerFunc { return next },
		DriverHub: nil,
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/ws/driver", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
