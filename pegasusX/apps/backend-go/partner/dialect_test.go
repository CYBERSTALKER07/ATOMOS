package partner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/adapters/onec"
)

func TestAllowPartnerDialect_Matrix(t *testing.T) {
	cases := []struct {
		market, dialect string
		want            error
	}{
		{"UZ", DialectOneC, nil},
		{"UZ", DialectEDIFACTLite, nil},
		{"UZ", "edifact", nil},
		{"KZ", DialectOneC, nil},
		{"EU", DialectOneC, ErrDialectNotForPack},
		{"UZ", DialectPEPPOL, ErrDialectNotForPack},
		{"EU", DialectPEPPOL, ErrDialectNotLive},
		{"US", DialectSAP, ErrDialectNotLive},
		{"US", DialectX12, ErrDialectNotLive},
		{"EU", DialectAS2, nil},
		{"UZ", DialectAS2, nil},
		{"UZ", "nope", ErrDialectUnknown},
	}
	for _, tc := range cases {
		err := AllowPartnerDialect(tc.market, tc.dialect)
		if tc.want == nil {
			if err != nil {
				t.Errorf("%s+%s: got %v want nil", tc.market, tc.dialect, err)
			}
			continue
		}
		if !errors.Is(err, tc.want) {
			t.Errorf("%s+%s: got %v want %v", tc.market, tc.dialect, err, tc.want)
		}
	}
}

func TestPackCurrencyOrEmpty_NoInvent(t *testing.T) {
	if got := packCurrencyOrEmpty("UZ"); got != "UZS" {
		t.Fatalf("UZ currency=%q", got)
	}
	if got := packCurrencyOrEmpty("EU"); got != "EUR" {
		t.Fatalf("EU catalog currency=%q (planned pack still declares EUR)", got)
	}
	if got := packCurrencyOrEmpty("XX"); got != "" {
		t.Fatalf("unknown must not invent UZS, got %q", got)
	}
}

func TestApplyPackCurrency_EmptyFromPack(t *testing.T) {
	out := applyPackCurrency([]onec.ImportProduct{{ExternalID: "SKU", Name: "Tea"}}, "UZ")
	if len(out) != 1 || out[0].Currency != "UZS" {
		t.Fatalf("%+v", out)
	}
	keep := applyPackCurrency([]onec.ImportProduct{{ExternalID: "SKU", Currency: "KZT"}}, "UZ")
	if keep[0].Currency != "KZT" {
		t.Fatalf("must not overwrite explicit currency: %+v", keep)
	}
}

func TestHandlePutEdiProfile_PEPPOLNotLive(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)}
	body, _ := json.Marshal(EdiProfile{PackName: DialectPEPPOL})
	req := httptest.NewRequest(http.MethodPut, "/partner/v1/edi/profile", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(WithPrincipal(req.Context(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-eu-edi",
	}), auth.Claims{MarketCode: "EU", SupplierID: "sup-eu-edi"}))
	rr := httptest.NewRecorder()
	h.HandlePutEdiProfile(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("dialect_not_live")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHandlePutEdiProfile_UZPEPPOLWrongPack(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)}
	body, _ := json.Marshal(EdiProfile{PackName: DialectPEPPOL})
	req := httptest.NewRequest(http.MethodPut, "/partner/v1/edi/profile", bytes.NewReader(body))
	req = req.WithContext(WithPrincipal(req.Context(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-uz-edi",
	}))
	rr := httptest.NewRecorder()
	h.HandlePutEdiProfile(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("dialect_not_for_pack")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHandlePutEdiProfile_USSAPSoldOnly(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)}
	body, _ := json.Marshal(EdiProfile{PackName: DialectSAP})
	req := httptest.NewRequest(http.MethodPut, "/partner/v1/edi/profile", bytes.NewReader(body))
	req = req.WithContext(auth.WithClaims(WithPrincipal(req.Context(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-us-sap",
	}), auth.Claims{MarketCode: "US", SupplierID: "sup-us-sap"}))
	rr := httptest.NewRecorder()
	h.HandlePutEdiProfile(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleOneCImport_EURejected(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)}
	req := httptest.NewRequest(http.MethodPost, "/partner/v1/adapters/onec/import", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(auth.WithClaims(WithPrincipal(req.Context(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-eu-1c",
	}), auth.Claims{MarketCode: "EU", SupplierID: "sup-eu-1c"}))
	rr := httptest.NewRecorder()
	h.HandleOneCImport(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("dialect_not_for_pack")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHandleListPartnerDialects_Public(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/platform/partner-dialects?pack=UZ", nil)
	rr := httptest.NewRecorder()
	HandleListPartnerDialects(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var body struct {
		Items              []PartnerDialect `json:"items"`
		RegisterNotBlocked bool             `json:"register_not_blocked"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.RegisterNotBlocked {
		t.Fatal("register must stay unblocked")
	}
	foundPEPPOL := false
	for _, it := range body.Items {
		if it.Code == DialectPEPPOL {
			foundPEPPOL = true
		}
		if it.Code == DialectOneC && !it.ExecuteLive {
			t.Fatal("1C on UZ must be live")
		}
	}
	if foundPEPPOL {
		t.Fatal("PEPPOL is not a UZ dialect")
	}
}

func TestHandleGetEdiProfile_Honesty(t *testing.T) {
	h := &Handlers{Svc: NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)}
	req := httptest.NewRequest(http.MethodGet, "/partner/v1/edi/profile", nil)
	req = req.WithContext(WithPrincipal(context.Background(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-get-edi",
	}))
	rr := httptest.NewRecorder()
	h.HandleGetEdiProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var body ediProfileResponse
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.RegisterNotBlocked || !body.DialectAllowed || body.MarketCode != "UZ" {
		t.Fatalf("%+v", body)
	}
}
