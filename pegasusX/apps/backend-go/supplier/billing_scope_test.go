package supplier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleConfigureBilling_RespectsTenantScopeAndIssuesScopedToken(t *testing.T) {
	t.Setenv("PEGASUSX_MARKET_PACK", "UZ")
	now := time.Now().UTC()

	repo := &dualSupplierRepo{
		profiles: map[string]Profile{
			"sup-seed": {
				SupplierID:   "sup-seed",
				LegalName:    "Seed Supplier",
				Country:      "UZ",
				Currency:     "UZS",
				IsRegistered: true,
				IsConfigured: false,
			},
			"sup-tenant-b": {
				SupplierID:   "sup-tenant-b",
				LegalName:    "Samarkand Beverage Logistics LLC",
				Country:      "UZ",
				Currency:     "UZS",
				IsRegistered: true,
				IsConfigured: false,
			},
		},
	}

	const jwtSecret = "test-supplier-billing-scope-secret-2026"
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-seed",
		Country:    "UZ",
		Currency:   "UZS",
		JWTSecret:  jwtSecret,
		JWTIssuer:  "pegasusx-test",
		JWTTTL:     time.Hour,
		Now:        func() time.Time { return now },
	})

	req := supplierAdminReq(http.MethodPost, "/v1/supplier/billing/setup", `{
		"bankName":"Kapitalbank",
		"accountHolder":"Samarkand Beverage Logistics LLC",
		"accountNumber":"20208000900012345678",
		"swiftBic":"KACHUZ22",
		"selectedGateways":["CASH"],
		"paymentAcceptor":"SUPPLIER"
	}`, "sup-tenant-b")

	rr := httptest.NewRecorder()
	svc.HandleConfigureBilling(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("HandleConfigureBilling status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var resp BillingSetupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SupplierID != "sup-tenant-b" {
		t.Fatalf("resp.SupplierID=%q want sup-tenant-b", resp.SupplierID)
	}
	if !resp.IsConfigured || resp.Token == "" {
		t.Fatalf("expected IsConfigured=true and non-empty scoped Token, got %+v", resp)
	}

	claims, err := auth.Parse(resp.Token, jwtSecret)
	if err != nil {
		t.Fatalf("auth.Parse(resp.Token): %v", err)
	}
	if claims.SupplierID != "sup-tenant-b" || !claims.IsConfigured || !claims.IsRegistered {
		t.Fatalf("unexpected token claims: %+v", claims)
	}

	if !repo.profiles["sup-tenant-b"].IsConfigured || repo.profiles["sup-tenant-b"].BankName != "Kapitalbank" {
		t.Fatalf("expected sup-tenant-b to be configured, got %+v", repo.profiles["sup-tenant-b"])
	}
	if repo.profiles["sup-seed"].IsConfigured || repo.profiles["sup-seed"].BankName != "" {
		t.Fatalf("seed supplier must remain untouched, got %+v", repo.profiles["sup-seed"])
	}

	cookies := rr.Result().Cookies()
	var sessionCookie string
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			sessionCookie = c.Value
			break
		}
	}
	cookieClaims, err := auth.Parse(sessionCookie, jwtSecret)
	if err != nil {
		t.Fatalf("auth.Parse(cookie): %v", err)
	}
	if cookieClaims.SupplierID != "sup-tenant-b" || !cookieClaims.IsConfigured {
		t.Fatalf("unexpected cookie claims: %+v", cookieClaims)
	}
}
