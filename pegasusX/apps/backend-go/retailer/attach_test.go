package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type captureRetailerRepo struct {
	testRetailerRepo
	last Retailer
}

func (r *captureRetailerRepo) FindByPhone(_ context.Context, phone string) (Retailer, bool, error) {
	if r.found && r.retailer.Phone == phone {
		return r.retailer, true, nil
	}
	if r.last.RetailerID != "" && r.last.Phone == phone {
		return r.last, true, nil
	}
	return Retailer{}, false, nil
}

func (r *captureRetailerRepo) CreateRetailer(_ context.Context, p Retailer, emit func(outbox.TxnBuffer) error) error {
	r.last = p
	if emit != nil {
		return emit(nopAttachBuf{})
	}
	return nil
}

type nopAttachBuf struct{}

func (nopAttachBuf) BufferOutbox(context.Context, outbox.Event) error { return nil }

func testRegisterReq(supplierID string) RegisterRequest {
	return RegisterRequest{
		Phone:      "+998901111111",
		Name:       "Shop",
		SupplierID: supplierID,
		Lat:        41.3,
		Lng:        69.2,
		H3Cell:     "8928308280fffff",
	}
}

func TestRegister_RequiresTradingPartner(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	_, err := svc.Register(context.Background(), testRegisterReq(""))
	if !errors.Is(err, ErrTradingPartnerRequired) {
		t.Fatalf("empty partner must not PreferTenant(seed): %v", err)
	}
}

func TestRegister_RejectsSeedOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	_, err := svc.Register(context.Background(), testRegisterReq("seed-1"))
	if !errors.Is(err, ErrSeedAttachForbidden) {
		t.Fatalf("err=%v", err)
	}
	if repo.last.RetailerID != "" {
		t.Fatal("must not persist a shop on seed")
	}
}

func TestRegister_AllowsExplicitPartner(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	resp, err := svc.Register(context.Background(), testRegisterReq("sup-minted"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "sup-minted" || repo.last.SupplierID != "sup-minted" {
		t.Fatalf("partner=%s persisted=%s", resp.SupplierID, repo.last.SupplierID)
	}
}

func TestRegister_UnknownPartner404(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	svc.SetTradingPartnerLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	_, err := svc.Register(context.Background(), testRegisterReq("ghost"))
	if !errors.Is(err, ErrUnknownTradingPartner) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_InviteToken(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{
		Repo: repo, SeedSupplierID: "seed-1", JWTSecret: "invite-secret",
		Now: func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
	})
	tok, _, err := MintTradingPartnerInvite("invite-secret", "sup-minted", "UZ", time.Hour, svc.now())
	if err != nil {
		t.Fatal(err)
	}
	req := testRegisterReq("")
	req.InviteToken = tok
	resp, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "sup-minted" {
		t.Fatalf("supplier_id=%s", resp.SupplierID)
	}
}

func TestRegister_SSMRAllowsExplicitSeed(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	resp, err := svc.Register(context.Background(), testRegisterReq("seed-1"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "seed-1" {
		t.Fatalf("ssmr seed attach should be explicit: %s", resp.SupplierID)
	}
}

func TestDemoPasswordDisabledOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	if secretMatchesDemo("1234") {
		t.Fatal("1234 must not be a master key outside ssmr")
	}
	svc := NewService(ServiceConfig{Repo: &testRetailerRepo{found: true, retailer: Retailer{RetailerID: "ret-1", Phone: "+1"}}, SeedSupplierID: "seed-1"})
	_, ok, err := svc.resolveRetailerLogin(context.Background(), "+1", "1234")
	if err != nil || ok {
		t.Fatalf("demo login must fail outside ssmr ok=%v err=%v", ok, err)
	}
}

func TestDemoPasswordAllowedInSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !secretMatchesDemo("1234") {
		t.Fatal("ssmr still allows demo secret")
	}
}

func TestRegister_RejectsPlannedPartner(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Cleanup(func() { auth.SetMarketProfileLookup(nil) })
	auth.SetMarketProfileLookup(func(supplierID string) (auth.MarketProfile, bool) {
		if supplierID == "sup-kz" {
			return auth.MarketProfile{MarketCode: "KZ", HomeCell: "cell-kz"}, true
		}
		return auth.MarketProfile{}, false
	})
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	_, err := svc.Register(context.Background(), testRegisterReq("sup-kz"))
	if !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_RejectsCountryMismatch(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	req := testRegisterReq("sup-minted")
	req.CountryCode = "PK"
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	_, err := svc.Register(context.Background(), req)
	if !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_RejectsInviteMarketMismatch(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{
		Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1", JWTSecret: "invite-secret",
		Now: func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
	})
	tok, _, err := MintTradingPartnerInvite("invite-secret", "sup-minted", "CA", time.Hour, svc.now())
	if err != nil {
		t.Fatal(err)
	}
	req := testRegisterReq("")
	req.InviteToken = tok
	_, err = svc.Register(context.Background(), req)
	if !errors.Is(err, auth.ErrCrossMarketDeferred) {
		t.Fatalf("err=%v", err)
	}
}

func TestRegister_LegacyInviteWithoutMarket(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{
		Repo: repo, SeedSupplierID: "seed-1", JWTSecret: "invite-secret",
		Now: func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
	})
	tok, _, err := MintTradingPartnerInvite("invite-secret", "sup-minted", "", time.Hour, svc.now())
	if err != nil {
		t.Fatal(err)
	}
	req := testRegisterReq("")
	req.InviteToken = tok
	resp, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.SupplierID != "sup-minted" {
		t.Fatalf("supplier_id=%s", resp.SupplierID)
	}
}

func TestHandleRegister_PlannedPartner404(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Cleanup(func() { auth.SetMarketProfileLookup(nil) })
	auth.SetMarketProfileLookup(func(supplierID string) (auth.MarketProfile, bool) {
		if supplierID == "sup-kz" {
			return auth.MarketProfile{MarketCode: "KZ", HomeCell: "cell-kz"}, true
		}
		return auth.MarketProfile{}, false
	})
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+99890","name":"S","supplier_id":"sup-kz","lat":41,"lng":69,"h3_cell":"8928308280fffff"}`))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "market_pack_not_shipped") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHandleRegister_CountryMismatch422(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+99890","name":"S","supplier_id":"sup-minted","country_code":"PK","lat":41,"lng":69,"h3_cell":"8928308280fffff"}`))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "cross_market_deferred") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHandleRegister_MissingPartner400(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &captureRetailerRepo{}, SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+99890","name":"S","lat":41,"lng":69,"h3_cell":"8928308280fffff"}`))
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSyncSupplierRetailerLink_NormalizesTINAndPhone(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo:           &captureRetailerRepo{},
		SeedSupplierID: "seed-1",
		Now:            func() time.Time { return time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC) },
	})
	saved, peers, err := svc.SyncSupplierRetailerLink(context.Background(), RetailerLinkInput{
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		TaxIDTIN:   " INN 301-234 567 ",
		Phone:      "90 123 45 67",
		LegalName:  " Navoiy Trade ",
	})
	if err != nil {
		t.Fatalf("SyncSupplierRetailerLink error: %v", err)
	}
	if len(peers) != 0 {
		t.Fatalf("expected 0 duplicate peers in clean attach, got %v", peers)
	}
	if saved.TaxIDTIN != "301234567" {
		t.Fatalf("TaxIDTIN=%q, want 301234567", saved.TaxIDTIN)
	}
	if saved.NormalizedPhone != "+998901234567" {
		t.Fatalf("NormalizedPhone=%q, want +998901234567", saved.NormalizedPhone)
	}
	if saved.LegalName != "Navoiy Trade" || saved.DedupStatus != "CLEAN" {
		t.Fatalf("unexpected saved link: %+v", saved)
	}
}

func TestHandleAcceptTradingPartnerInvite_NormalizesAndRejectsCrossMarket(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	repo := &testRetailerRepo{
		found: true,
		retailer: Retailer{
			RetailerID: "ret-55",
			Phone:      "090 555 66 77",
			Name:       "Silk Road Shop",
		},
	}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SeedSupplierID: "seed-1",
		JWTSecret:      "invite-secret",
		Now:            func() time.Time { return now },
	})
	tok, _, err := MintTradingPartnerInvite("invite-secret", "sup-minted", "UZ", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/attach",
		strings.NewReader(`{"invite_token":"`+tok+`","tin":"301-999-001"}`))
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-55",
		Role:    auth.RoleRetailer,
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	svc.HandleAcceptTradingPartnerInvite(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Status     string `json:"status"`
		SupplierID string `json:"supplier_id"`
		RetailerID string `json:"retailer_id"`
		Link       struct {
			TaxIDTIN        string `json:"tax_id_tin"`
			NormalizedPhone string `json:"normalized_phone"`
			LegalName       string `json:"legal_name"`
			DedupStatus     string `json:"dedup_status"`
		} `json:"link"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SupplierID != "sup-minted" || resp.RetailerID != "ret-55" {
		t.Fatalf("unexpected ids: %+v", resp)
	}
	if resp.Link.TaxIDTIN != "301999001" || resp.Link.NormalizedPhone != "+998905556677" || resp.Link.LegalName != "Silk Road Shop" {
		t.Fatalf("unexpected link payload: %+v", resp.Link)
	}
}

func TestHandleMobileRegister_DesktopSelfOnboardWithoutPartnerOrCoords(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998908880001","name":"Mahalla Informal Store","lat":0,"lng":0,"delivery_address":"Pending setup"}`))
	rr := httptest.NewRecorder()
	svc.HandleMobileRegister(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if repo.last.SupplierID != "" {
		t.Fatalf("self-onboarding must never bind seed supplier_id, got %q", repo.last.SupplierID)
	}
	if retailerProfileConfigured(repo.last) {
		t.Fatalf("deferred lat=0/lng=0 store must start with is_configured=false")
	}

	// Deferred/unshipped country must still fail closed.
	reqBadMarket := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998908880002","name":"Foreign Store","country_code":"PK","lat":0,"lng":0}`))
	rrBad := httptest.NewRecorder()
	svc.HandleMobileRegister(rrBad, reqBadMarket)
	if rrBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("cross-market status=%d want=422 body=%s", rrBad.Code, rrBad.Body.String())
	}
}

func TestHandleMobileRegister_PhoneNumberSelfOnboardAndDuplicateReject(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1", JWTSecret: "test-secret"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone_number":"+998908880003","password":"secret123","store_name":"Mobile Store","latitude":41.3,"longitude":69.24}`))
	rr := httptest.NewRecorder()
	svc.HandleMobileRegister(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("mobile register status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}
	if repo.last.SupplierID != "" {
		t.Fatalf("mobile self-onboarding must not bind seed supplier, got %q", repo.last.SupplierID)
	}

	// Existing phone via mobile unauthenticated phone_number path must fail closed (409).
	rrDup := httptest.NewRecorder()
	reqDup := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone_number":"+998908880003","password":"secret123","store_name":"Dup"}`))
	svc.HandleMobileRegister(rrDup, reqDup)
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("duplicate phone_number status=%d want=409 body=%s", rrDup.Code, rrDup.Body.String())
	}
}

func TestHandleMobileRegister_ExistingPhoneWithSupplierID_ReturnsConflict(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1", JWTSecret: "test-secret"})

	// 1. Initial legitimate registration
	reqInit := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone_number":"+998901234567","supplier_id":"sup-minted","store_name":"Victim Store","latitude":41.3,"longitude":69.24}`))
	rrInit := httptest.NewRecorder()
	svc.HandleMobileRegister(rrInit, reqInit)
	if rrInit.Code != http.StatusOK {
		t.Fatalf("initial register status=%d want=200 body=%s", rrInit.Code, rrInit.Body.String())
	}

	var initResp map[string]any
	if err := json.Unmarshal(rrInit.Body.Bytes(), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp["token"] == nil || initResp["token"] == "" {
		t.Fatal("expected token in initial registration response")
	}

	// 2. Exploit attempt: unauthenticated attacker attempts to re-register victim phone with supplier_id
	reqAttack := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone_number":"+998901234567","supplier_id":"sup-minted","store_name":"Attacker Store","latitude":41.3,"longitude":69.24}`))
	rrAttack := httptest.NewRecorder()
	svc.HandleMobileRegister(rrAttack, reqAttack)

	if rrAttack.Code != http.StatusConflict {
		t.Fatalf("exploit attempt status=%d want=409 body=%s", rrAttack.Code, rrAttack.Body.String())
	}

	var attackResp map[string]any
	if err := json.Unmarshal(rrAttack.Body.Bytes(), &attackResp); err != nil {
		t.Fatalf("unmarshal attack response: %v", err)
	}
	if attackResp["error"] != "phone_already_registered" {
		t.Fatalf("expected error='phone_already_registered', got %v", attackResp["error"])
	}
	if attackResp["token"] != nil {
		t.Fatalf("exploit succeeded: token minted for existing phone: %v", attackResp["token"])
	}
	if attackResp["refresh_token"] != nil {
		t.Fatalf("exploit succeeded: refresh_token minted for existing phone: %v", attackResp["refresh_token"])
	}
}

func TestHandleMobileRegister_ExistingPhoneWithInviteToken_ReturnsConflict(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{
		Repo:           repo,
		SeedSupplierID: "seed-1",
		JWTSecret:      "test-secret",
		Now:            func() time.Time { return now },
	})

	tok, _, err := MintTradingPartnerInvite("test-secret", "sup-minted", "UZ", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Initial legitimate registration
	reqInit := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone_number":"+998909876543","invite_token":"`+tok+`","store_name":"Victim Shop","latitude":41.3,"longitude":69.24}`))
	rrInit := httptest.NewRecorder()
	svc.HandleMobileRegister(rrInit, reqInit)
	if rrInit.Code != http.StatusOK {
		t.Fatalf("initial register status=%d want=200 body=%s", rrInit.Code, rrInit.Body.String())
	}

	// 2. Exploit attempt with same phone and invite token
	reqAttack := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone_number":"+998909876543","invite_token":"`+tok+`","store_name":"Attacker Shop","latitude":41.3,"longitude":69.24}`))
	rrAttack := httptest.NewRecorder()
	svc.HandleMobileRegister(rrAttack, reqAttack)

	if rrAttack.Code != http.StatusConflict {
		t.Fatalf("exploit attempt status=%d want=409 body=%s", rrAttack.Code, rrAttack.Body.String())
	}
	var attackResp map[string]any
	if err := json.Unmarshal(rrAttack.Body.Bytes(), &attackResp); err != nil {
		t.Fatalf("unmarshal attack response: %v", err)
	}
	if attackResp["error"] != "phone_already_registered" {
		t.Fatalf("expected error='phone_already_registered', got %v", attackResp["error"])
	}
	if attackResp["token"] != nil || attackResp["refresh_token"] != nil {
		t.Fatal("exploit succeeded: tokens minted for duplicate phone")
	}
}

func TestWriteCanonicalRegisterResponse_ExistingPhone_ReturnsConflict(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})

	// 1. Initial canonical registration (supplier-bound)
	reqInit := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998907770001","supplier_id":"sup-minted","name":"First Store","lat":41.3,"lng":69.24,"h3_cell":"8928308280fffff"}`))
	rrInit := httptest.NewRecorder()
	svc.HandleMobileRegister(rrInit, reqInit)
	if rrInit.Code != http.StatusCreated {
		t.Fatalf("initial canonical register status=%d want=201 body=%s", rrInit.Code, rrInit.Body.String())
	}

	// 2. Duplicate canonical registration with supplier_id -> 409 Conflict
	reqDupSup := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998907770001","supplier_id":"sup-minted","name":"Dup Store","lat":41.3,"lng":69.24,"h3_cell":"8928308280fffff"}`))
	rrDupSup := httptest.NewRecorder()
	svc.HandleMobileRegister(rrDupSup, reqDupSup)
	if rrDupSup.Code != http.StatusConflict {
		t.Fatalf("duplicate canonical with supplier status=%d want=409 body=%s", rrDupSup.Code, rrDupSup.Body.String())
	}

	// 3. Duplicate canonical registration without supplier_id (self-onboard) -> 409 Conflict
	reqDupSelf := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998907770001","name":"Dup Store 2","lat":41.3,"lng":69.24}`))
	rrDupSelf := httptest.NewRecorder()
	svc.HandleMobileRegister(rrDupSelf, reqDupSelf)
	if rrDupSelf.Code != http.StatusConflict {
		t.Fatalf("duplicate canonical self-onboard status=%d want=409 body=%s", rrDupSelf.Code, rrDupSelf.Body.String())
	}
}

func TestHandleRegister_ExistingPhone_ReturnsConflict(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	repo := &captureRetailerRepo{}
	svc := NewService(ServiceConfig{Repo: repo, SeedSupplierID: "seed-1"})

	// 1. First registration via HandleRegister
	reqInit := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998906660001","supplier_id":"sup-minted","name":"Shop One","lat":41.3,"lng":69.24,"h3_cell":"8928308280fffff"}`))
	rrInit := httptest.NewRecorder()
	svc.HandleRegister(rrInit, reqInit)
	if rrInit.Code != http.StatusCreated {
		t.Fatalf("initial HandleRegister status=%d want=201 body=%s", rrInit.Code, rrInit.Body.String())
	}

	// 2. Duplicate registration via HandleRegister -> 409 Conflict
	reqDup := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/register",
		strings.NewReader(`{"phone":"+998906660001","supplier_id":"sup-minted","name":"Shop Dup","lat":41.3,"lng":69.24,"h3_cell":"8928308280fffff"}`))
	rrDup := httptest.NewRecorder()
	svc.HandleRegister(rrDup, reqDup)
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("duplicate HandleRegister status=%d want=409 body=%s", rrDup.Code, rrDup.Body.String())
	}
}
