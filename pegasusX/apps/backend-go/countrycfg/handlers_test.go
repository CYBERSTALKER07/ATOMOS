package countrycfg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestUZDefaultDoesNotClaimCheckout(t *testing.T) {
	cfg := UZDefault()
	if cfg.CheckoutReadsThis {
		t.Fatal("checkout must not read countrycfg")
	}
	if cfg.CountryCode != "UZ" || cfg.CurrencyCode != "UZS" {
		t.Fatalf("%+v", cfg)
	}
}

func TestHandleCountryConfig_UZ(t *testing.T) {
	h := &Handlers{}
	r := chi.NewRouter()
	r.Get("/v1/country-configs/{code}", h.HandleCountryConfig)
	req := httptest.NewRequest(http.MethodGet, "/v1/country-configs/UZ", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-1"}))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var cfg Config
	if err := json.Unmarshal(rr.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.CheckoutReadsThis || cfg.Timezone != "Asia/Tashkent" {
		t.Fatalf("%+v", cfg)
	}
}

func TestHandleCountryConfig_US(t *testing.T) {
	h := &Handlers{}
	r := chi.NewRouter()
	r.Get("/v1/country-configs/{code}", h.HandleCountryConfig)
	req := httptest.NewRequest(http.MethodGet, "/v1/country-configs/US", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var cfg Config
	if err := json.Unmarshal(rr.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.CheckoutReadsThis || !cfg.OpsReadsThis || cfg.CurrencyCode != "USD" {
		t.Fatalf("%+v", cfg)
	}
}

func TestHandleCountryConfig_Unknown404(t *testing.T) {
	h := &Handlers{}
	r := chi.NewRouter()
	r.Get("/v1/country-configs/{code}", h.HandleCountryConfig)
	req := httptest.NewRequest(http.MethodGet, "/v1/country-configs/XX", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}
