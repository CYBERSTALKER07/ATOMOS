package retailer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

func TestHandlePaymentCatalog_UZOmitsStripeAdyen(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/payment-catalog", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:       "ret-1",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-1",
		MarketCode:    "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePaymentCatalog(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		CurrencyCode string               `json:"currency_code"`
		MarketCode   string               `json:"market_code"`
		Catalog      []payment.PSPListing `json:"catalog"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.CurrencyCode != "UZS" {
		t.Fatalf("currency=%s", body.CurrencyCode)
	}
	if body.MarketCode != "UZ" {
		t.Fatalf("market=%s", body.MarketCode)
	}
	seen := map[string]bool{}
	for _, row := range body.Catalog {
		seen[row.Code] = true
	}
	if seen["STRIPE"] || seen["ADYEN"] || seen["AIRWALLEX"] {
		t.Fatalf("UZ catalog leaked foreign rails: %#v", body.Catalog)
	}
	if !seen["CASH"] || !seen["GLOBAL_PAY"] {
		t.Fatalf("UZ catalog missing live rails: %#v", body.Catalog)
	}
}

func TestHandlePaymentCatalog_PlannedFailsClosed(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/payment-catalog", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", MarketCode: "CA",
	}))
	rr := httptest.NewRecorder()
	svc.HandlePaymentCatalog(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandlePaymentCatalog_Unauthorized(t *testing.T) {
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/payment-catalog", nil)
	rr := httptest.NewRecorder()
	svc.HandlePaymentCatalog(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
