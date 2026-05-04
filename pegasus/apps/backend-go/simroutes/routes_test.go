package simroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_NilEngineDoesNotMountSimulationRoutes(t *testing.T) {
	router := chi.NewRouter()

	RegisterRoutes(router, Deps{
		Engine: nil,
		Log:    func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	request := httptest.NewRequest(http.MethodGet, PathSimStatus, nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
