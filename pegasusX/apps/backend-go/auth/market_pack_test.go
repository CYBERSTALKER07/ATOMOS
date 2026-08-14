package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveMarketPack_UZShipped(t *testing.T) {
	p, ok := ResolveShippedMarketPack("uz")
	if !ok || p.Code != "UZ" || p.CurrencyCode != "UZS" {
		t.Fatalf("uz pack: %+v ok=%v", p, ok)
	}
	if p.CheckoutReadsThis {
		t.Fatal("GS-A: checkout must not read pack yet")
	}
	if p.FiscalAdapter != "MY_SOLIQ" {
		t.Fatalf("fiscal=%s", p.FiscalAdapter)
	}
}

func TestResolveMarketPack_PlannedNotShipped(t *testing.T) {
	p, ok := ResolveMarketPack("EU")
	if !ok || p.Status != MarketPackPlanned {
		t.Fatalf("EU should be planned: %+v", p)
	}
	if _, shipped := ResolveShippedMarketPack("EU"); shipped {
		t.Fatal("planned pack must fail ResolveShippedMarketPack")
	}
	if _, ok := ResolveMarketPack("XX"); ok {
		t.Fatal("unknown must be false")
	}
}

func TestIssueParse_MarketCodeHomeCell(t *testing.T) {
	tok, err := Issue(Claims{
		Subject: "u1", Role: RoleAdmin, SupplierID: "s1",
		MarketCode: "UZ", HomeCell: "cell-uz",
	}, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != "UZ" || got.HomeCell != "cell-uz" {
		t.Fatalf("claims=%+v", got)
	}
}

func TestHandleSession_Unauthorized(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleSession(rr, httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandleSession_UZPack(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject: "admin-1", Role: RoleAdmin, SupplierID: "sup-1",
	}))
	rr := httptest.NewRecorder()
	HandleSession(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["market_code"] != "UZ" {
		t.Fatalf("market_code=%v", body["market_code"])
	}
	if body["checkout_reads_this"] != false {
		t.Fatalf("checkout_reads_this=%v", body["checkout_reads_this"])
	}
	pack, _ := body["pack"].(map[string]any)
	if pack["fiscal_adapter"] != "MY_SOLIQ" {
		t.Fatalf("pack=%v", pack)
	}
}

func TestHandleGetMarketPack_Unknown(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleGetMarketPack("XX")(rr, httptest.NewRequest(http.MethodGet, "/v1/platform/market-packs/XX", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandleListMarketPacks(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleListMarketPacks(rr, httptest.NewRequest(http.MethodGet, "/v1/platform/market-packs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}
