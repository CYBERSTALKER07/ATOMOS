package deliveryroutes

import (
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
