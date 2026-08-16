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
		t.Fatal("catalog flag stays false until GS-M2")
	}
	if p.FiscalAdapter != "MY_SOLIQ" {
		t.Fatalf("fiscal=%s", p.FiscalAdapter)
	}
	if p.PayoutRail != "bank-file" {
		t.Fatalf("payout_rail=%s", p.PayoutRail)
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

func TestIssue_StampsEnvWhenEmpty(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "")
	t.Setenv("HOME_CELL", "")
	tok, err := Issue(Claims{
		Subject: "u1", Role: RoleAdmin, SupplierID: "s1",
	}, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != DefaultMarketCode || got.HomeCell != DefaultHomeCell {
		t.Fatalf("expected env defaults, got market=%q cell=%q", got.MarketCode, got.HomeCell)
	}
}

func TestIssue_PreservesExplicitClaims(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Setenv("HOME_CELL", "cell-uz")
	tok, err := Issue(Claims{
		Subject: "u1", Role: RoleRetailer, SupplierID: "s1",
		MarketCode: "kz", HomeCell: "CELL-KZ",
	}, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != "KZ" || got.HomeCell != "cell-kz" {
		t.Fatalf("explicit claims must win: %+v", got)
	}
}

func TestIssue_ProfileBeatsEnv(t *testing.T) {
	t.Cleanup(func() { SetMarketProfileLookup(nil) })
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Setenv("HOME_CELL", "cell-uz")
	SetMarketProfileLookup(func(supplierID string) (MarketProfile, bool) {
		if supplierID != "s-profile" {
			return MarketProfile{}, false
		}
		return MarketProfile{MarketCode: "EU", HomeCell: "cell-eu"}, true
	})
	tok, err := Issue(Claims{
		Subject: "u1", Role: RoleAdmin, SupplierID: "s-profile",
	}, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != "EU" || got.HomeCell != "cell-eu" {
		t.Fatalf("profile must beat env: %+v", got)
	}
}

func TestIssue_RefreshCopiesStampedClaims(t *testing.T) {
	first, err := Issue(Claims{
		Subject: "u1", Role: RoleDriver, SupplierID: "s1",
		MarketCode: "US", HomeCell: "cell-us",
	}, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(first, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	parsed.JTI = ""
	second, err := Issue(parsed, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(second, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != "US" || got.HomeCell != "cell-us" {
		t.Fatalf("refresh must copy pack claims: %+v", got)
	}
}

func TestIssue_ProfileBeatsEnvStampedClaim(t *testing.T) {
	t.Cleanup(func() { SetMarketProfileLookup(nil) })
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Setenv("HOME_CELL", "cell-uz")
	SetMarketProfileLookup(func(supplierID string) (MarketProfile, bool) {
		if supplierID != "s-profile" {
			return MarketProfile{}, false
		}
		return MarketProfile{MarketCode: "EU", HomeCell: "cell-eu"}, true
	})
	tok, err := Issue(Claims{
		Subject: "u1", Role: RoleAdmin, SupplierID: "s-profile",
		MarketCode: "UZ", HomeCell: "cell-uz",
	}, IssueOptions{Secret: "test-secret", Issuer: "t", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got.MarketCode != "EU" || got.HomeCell != "cell-eu" {
		t.Fatalf("profile must beat env-stamped claim: %+v", got)
	}
}

func TestResolveMarketAssignment_Sources(t *testing.T) {
	t.Cleanup(func() { SetMarketProfileLookup(nil) })
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Setenv("HOME_CELL", "cell-uz")

	env := ResolveMarketAssignment(Claims{Subject: "u", Role: RoleAdmin, SupplierID: "s1"})
	if env.Source != MarketSourceEnv || env.MarketCode != "UZ" {
		t.Fatalf("empty=%+v", env)
	}

	stamped := ResolveMarketAssignment(Claims{
		Subject: "u", Role: RoleAdmin, SupplierID: "s1",
		MarketCode: "UZ", HomeCell: "cell-uz",
	})
	if stamped.Source != MarketSourceEnv {
		t.Fatalf("env-stamped JWT must not look chosen: %+v", stamped)
	}

	explicit := ResolveMarketAssignment(Claims{
		Subject: "u", Role: RoleAdmin, SupplierID: "s1",
		MarketCode: "US", HomeCell: "cell-us",
	})
	if explicit.Source != MarketSourceClaim || explicit.MarketCode != "US" {
		t.Fatalf("explicit=%+v", explicit)
	}

	SetMarketProfileLookup(func(supplierID string) (MarketProfile, bool) {
		return MarketProfile{MarketCode: "KZ", HomeCell: "cell-kz"}, true
	})
	fromProfile := ResolveMarketAssignment(Claims{
		Subject: "u", Role: RoleAdmin, SupplierID: "s1",
		MarketCode: "UZ",
	})
	if fromProfile.Source != MarketSourceProfile || fromProfile.MarketCode != "KZ" {
		t.Fatalf("profile=%+v", fromProfile)
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
	if body["source"] != MarketSourceEnv {
		t.Fatalf("empty profile must be source=env, got %v", body["source"])
	}
	if body["checkout_reads_this"] != false {
		t.Fatalf("checkout_reads_this=%v", body["checkout_reads_this"])
	}
	pack, _ := body["pack"].(map[string]any)
	if pack["fiscal_adapter"] != "MY_SOLIQ" {
		t.Fatalf("pack=%v", pack)
	}
	if pack["payout_rail"] != "bank-file" {
		t.Fatalf("payout_rail=%v", pack["payout_rail"])
	}
}

func TestHandleGetMarketPack_Unknown(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleGetMarketPack("XX")(rr, httptest.NewRequest(http.MethodGet, "/v1/platform/market-packs/XX", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestHandleSession_ProfileSource(t *testing.T) {
	t.Setenv("HOME_CELL", "cell-uz")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Cleanup(func() { SetMarketProfileLookup(nil) })
	SetMarketProfileLookup(func(supplierID string) (MarketProfile, bool) {
		return MarketProfile{MarketCode: "EU", HomeCell: "cell-eu"}, true
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req = req.WithContext(WithClaims(req.Context(), Claims{
		Subject: "admin-1", Role: RoleAdmin, SupplierID: "sup-1",
		MarketCode: "UZ",
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
	if body["source"] != MarketSourceProfile || body["market_code"] != "EU" {
		t.Fatalf("body=%v", body)
	}
	if body["home_cell"] != "cell-eu" || body["api_url"] != "https://api-eu.pegasusx.app" {
		t.Fatalf("eu session api_url=%v home_cell=%v", body["api_url"], body["home_cell"])
	}
}

func TestHandleListMarketPacks(t *testing.T) {
	rr := httptest.NewRecorder()
	HandleListMarketPacks(rr, httptest.NewRequest(http.MethodGet, "/v1/platform/market-packs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d", rr.Code)
	}
}
