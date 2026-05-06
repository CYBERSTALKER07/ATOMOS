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

func TestRegisterRoutes_MigratedLegacyEndpointsMounted(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "deliver", method: http.MethodPost, path: "/v1/order/deliver"},
		{name: "validate qr", method: http.MethodPost, path: "/v1/order/validate-qr"},
		{name: "confirm offload", method: http.MethodPost, path: "/v1/order/confirm-offload"},
		{name: "complete", method: http.MethodPost, path: "/v1/order/complete"},
		{name: "collect cash", method: http.MethodPost, path: "/v1/order/collect-cash"},
		{name: "routes list", method: http.MethodGet, path: "/v1/routes"},
		{name: "prediction create", method: http.MethodPost, path: "/v1/prediction/create"},
		{name: "refund", method: http.MethodPost, path: "/v1/order/refund"},
		{name: "amend", method: http.MethodPatch, path: "/v1/order/amend"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, response.Code, http.StatusUnauthorized)
			}
		})
	}
}
