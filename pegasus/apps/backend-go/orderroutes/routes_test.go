package orderroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutes_LineItemHistoryRouteMounted(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/orders/line-items/history?retailer_id=r-1", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
