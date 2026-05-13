package deliveryroutes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"backend-go/auth"
)

func TestRegisterRoutes_MissingItemsAllowsPayloaderRole(t *testing.T) {
	token, err := auth.GenerateTestToken("payload-role-test", "PAYLOADER")
	if err != nil {
		t.Fatalf("GenerateTestToken: %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/missing-items", strings.NewReader(`{"order_id":"o1","missing_items":[],"source":"PAYLOAD_TERMINAL"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body["status"]; got != "REPORTED" {
		t.Fatalf("status body = %v, want REPORTED", got)
	}
}

func TestRegisterRoutes_MissingItemsRejectsRetailerRole(t *testing.T) {
	token, err := auth.GenerateTestToken("retailer-role-test", "RETAILER")
	if err != nil {
		t.Fatalf("GenerateTestToken: %v", err)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/missing-items", strings.NewReader(`{"order_id":"o1","missing_items":[{"sku_id":"s1","missing_qty":1}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRegisterRoutes_HandshakeEndpointsMounted(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, Deps{
		Log: func(next http.HandlerFunc) http.HandlerFunc { return next },
	})

	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "verify handshake",
			path: "/v1/delivery/verify-handshake",
			body: `{"order_id":"o-1","handshake_token":"token","driver_latitude":41.31,"driver_longitude":69.24}`,
		},
		{
			name: "update order during delivery",
			path: "/v1/delivery/update-order-during-delivery",
			body: `{"order_id":"o-1","items":[{"product_id":"sku-1","accepted_qty":1,"rejected_qty":0}]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}
