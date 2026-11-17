package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type statefulRetailerRepo struct {
	testRetailerRepo
	mu sync.Mutex
}

func (r *statefulRetailerRepo) GetRetailer(_ context.Context, _ string) (Retailer, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.found {
		return Retailer{}, false, nil
	}
	return r.retailer, true, nil
}

func (r *statefulRetailerRepo) UpdateRetailer(_ context.Context, ret Retailer, _ func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retailer = ret
	r.found = true
	return nil
}

type fakeMigrationAR struct {
	mu       sync.Mutex
	invoices map[string]MigrationOpeningInvoice
}

func (m *fakeMigrationAR) ListOpeningBalances(_ context.Context, retailerID string) ([]MigrationOpeningInvoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []MigrationOpeningInvoice
	for _, inv := range m.invoices {
		if inv.RetailerID == retailerID {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (m *fakeMigrationAR) AcknowledgeOpeningInvoice(_ context.Context, retailerID, invoiceID string) (MigrationOpeningInvoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.invoices[invoiceID]
	if !ok {
		return MigrationOpeningInvoice{}, errors.New("invoice_not_found")
	}
	if inv.RetailerID != retailerID {
		return MigrationOpeningInvoice{}, errors.New("invoice_forbidden")
	}
	inv.AcknowledgedAt = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	inv.DisputedAt = ""
	inv.DisputeReason = ""
	inv.DunningFrozen = false
	m.invoices[invoiceID] = inv
	return inv, nil
}

func (m *fakeMigrationAR) DisputeOpeningInvoice(_ context.Context, retailerID, invoiceID, reason string) (MigrationOpeningInvoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.invoices[invoiceID]
	if !ok {
		return MigrationOpeningInvoice{}, errors.New("invoice_not_found")
	}
	if inv.RetailerID != retailerID {
		return MigrationOpeningInvoice{}, errors.New("invoice_forbidden")
	}
	inv.DisputedAt = time.Date(2026, 11, 14, 12, 5, 0, 0, time.UTC).Format(time.RFC3339)
	inv.DisputeReason = reason
	inv.DunningFrozen = true
	m.invoices[invoiceID] = inv
	return inv, nil
}

func newRetailerINN(now time.Time) (*innlookup.Service, *innlookup.StubAdapter, *innlookup.MemoryStore) {
	stub := innlookup.NewStubAdapter()
	mem := innlookup.NewMemoryStore()
	svc := innlookup.NewService(innlookup.Config{
		Adapter:      stub,
		Cache:        mem,
		Tenants:      mem,
		JoinRequests: mem,
		Now:          func() time.Time { return now },
	})
	return svc, stub, mem
}

func retailerReq(method, target, body, retailerID, idemKey string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Role:          auth.RoleRetailer,
		Subject:       "ruser-1",
		RetailerOrgID: retailerID,
		RetailerRole:  "OWNER",
		SupplierID:    "sup-1",
	})
	return req.WithContext(ctx)
}

func TestHandleTaxIdentity_LegalInformalFailClosedAndJoinRequest(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	innSvc, stub, mem := newRetailerINN(now)
	stub.Register("305556667", "Makro Chilonzor MChJ", "Tashkent")

	repo := &statefulRetailerRepo{
		testRetailerRepo: testRetailerRepo{
			found: true,
			retailer: Retailer{
				RetailerID:  "ret-1",
				SupplierID:  "sup-1",
				Phone:       "+998901234567",
				Name:        "Magazin Baraka",
				CountryCode: "UZ",
				HomeCell:    auth.DefaultHomeCell,
			},
		},
	}
	var notifiedType, notifiedTenant string
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Idem:       idempotency.NewInMemoryStore(),
		INNLookup:  innSvc,
		Now:        func() time.Time { return now },
		MigrationNotifier: func(_ context.Context, tenantID, _, nType, _, _, _ string) error {
			notifiedTenant = tenantID
			notifiedType = nType
			return nil
		},
	})

	// 1) Missing Idempotency-Key rejected strictly.
	rrNoIdem := httptest.NewRecorder()
	svc.HandleTaxIdentity(rrNoIdem, retailerReq(http.MethodPatch, "/v1/retailer/tax-identity", `{"informal_shop":true}`, "ret-1", ""))
	if rrNoIdem.Code != http.StatusBadRequest {
		t.Fatalf("missing idem status=%d want=400 body=%s", rrNoIdem.Code, rrNoIdem.Body.String())
	}

	// 2) Informal phone-only mode sets INFORMAL_PHONE with empty NormalizedTaxID.
	rrInf := httptest.NewRecorder()
	svc.HandleTaxIdentity(rrInf, retailerReq(http.MethodPatch, "/v1/retailer/tax-identity", `{"informal_shop":true}`, "ret-1", "tax-inf-1"))
	if rrInf.Code != http.StatusOK {
		t.Fatalf("informal tax-identity status=%d want=200 body=%s", rrInf.Code, rrInf.Body.String())
	}
	if repo.retailer.LegalEntityStatus != LegalEntityInformalPhone || repo.retailer.NormalizedTaxID != "" {
		t.Fatalf("expected INFORMAL_PHONE with empty norm INN, got status=%q norm=%q", repo.retailer.LegalEntityStatus, repo.retailer.NormalizedTaxID)
	}

	// 3) Fail-closed 503 when registry unavailable.
	stub.SetForceUnavailable(true)
	rrUnavail := httptest.NewRecorder()
	svc.HandleTaxIdentity(rrUnavail, retailerReq(http.MethodPatch, "/v1/retailer/tax-identity", `{"tax_id":"305556667"}`, "ret-1", "tax-unavail-1"))
	if rrUnavail.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable registry status=%d want=503 body=%s", rrUnavail.Code, rrUnavail.Body.String())
	}
	stub.SetForceUnavailable(false)

	// 4) Valid INN upgrades store to LEGAL_VERIFIED.
	rrVer := httptest.NewRecorder()
	svc.HandleTaxIdentity(rrVer, retailerReq(http.MethodPatch, "/v1/retailer/tax-identity", `{"tax_id":"INN 305-556 667"}`, "ret-1", "tax-ver-1"))
	if rrVer.Code != http.StatusOK {
		t.Fatalf("legal verified status=%d want=200 body=%s", rrVer.Code, rrVer.Body.String())
	}
	if repo.retailer.LegalEntityStatus != LegalEntityVerified || repo.retailer.NormalizedTaxID != "305556667" || repo.retailer.InnVerifiedAt == nil {
		t.Fatalf("expected LEGAL_VERIFIED with norm INN & timestamp, got %+v", repo.retailer)
	}

	// 5) Duplicate INN in another retailer triggers 409 join_request_pending + notification.
	mem.RegisterTenant(innlookup.TenantMatch{
		HomeCell:      auth.DefaultHomeCell,
		TenantType:    innlookup.TenantTypeRetailer,
		TenantID:      "ret-owner",
		LegalName:     "Makro Chilonzor MChJ",
		NormalizedINN: "305556667",
	})
	rrDup := httptest.NewRecorder()
	svc.HandleTaxIdentity(rrDup, retailerReq(http.MethodPatch, "/v1/retailer/tax-identity", `{"tax_id":"305556667"}`, "ret-1", "tax-dup-1"))
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("duplicate retailer INN status=%d want=409 body=%s", rrDup.Code, rrDup.Body.String())
	}
	if notifiedTenant != "ret-owner" || notifiedType != "JOIN_REQUEST_CREATED" {
		t.Fatalf("expected JOIN_REQUEST_CREATED notification for ret-owner, got %q %q", notifiedTenant, notifiedType)
	}
}

func TestHandleRetailerSetup_DualTaxIdentityAndInformal(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	innSvc, stub, _ := newRetailerINN(now)
	stub.Register("308889991", "Yangi Do'kon XK", "Samarkand")

	repo := &statefulRetailerRepo{
		testRetailerRepo: testRetailerRepo{
			found: true,
			retailer: Retailer{
				RetailerID:  "ret-setup-1",
				SupplierID:  "sup-1",
				Phone:       "+998901239988",
				CountryCode: "UZ",
				HomeCell:    auth.DefaultHomeCell,
			},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		JWTSecret:  "retailer-setup-secret-0123456789abc",
		INNLookup:  innSvc,
		Now:        func() time.Time { return now },
	})

	rr := httptest.NewRecorder()
	svc.HandleRetailerSetup(rr, retailerReq(http.MethodPost, "/v1/retailer/setup", `{
		"retailer_id":"ret-setup-1",
		"tax_id":"308889991",
		"shippingAddress":"Registon 5",
		"city":"Samarkand"
	}`, "ret-setup-1", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("setup with tax_id status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}
	if repo.retailer.LegalEntityStatus != LegalEntityVerified || repo.retailer.NormalizedTaxID != "308889991" {
		t.Fatalf("expected LEGAL_VERIFIED after setup, got status=%q norm=%q", repo.retailer.LegalEntityStatus, repo.retailer.NormalizedTaxID)
	}
	var authResp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &authResp); err != nil {
		t.Fatalf("decode setup resp: %v", err)
	}
	if authResp["legal_entity_status"] != LegalEntityVerified {
		t.Fatalf("response legal_entity_status=%v want=%s", authResp["legal_entity_status"], LegalEntityVerified)
	}
}

func TestHandleNasiyaOpeningBalances_AcknowledgeAndDispute(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	arMgr := &fakeMigrationAR{
		invoices: map[string]MigrationOpeningInvoice{
			"inv-mig-1": {
				InvoiceID:      "inv-mig-1",
				SupplierID:     "sup-1",
				SupplierName:   "Baraka Supplier",
				RetailerID:     "ret-1",
				PrincipalMinor: 5000000,
				BalanceMinor:   5000000,
				Currency:       "UZS",
				Status:         "OPEN",
				Source:         "MIGRATION",
				DunningFrozen:  true,
			},
		},
	}

	var notifications []string
	svc := NewService(ServiceConfig{
		Repo:        &statefulRetailerRepo{},
		SupplierID:  "sup-1",
		Idem:        idempotency.NewInMemoryStore(),
		MigrationAR: arMgr,
		Now:         func() time.Time { return now },
		MigrationNotifier: func(_ context.Context, tenantID, _, nType, _, _, _ string) error {
			notifications = append(notifications, tenantID+":"+nType)
			return nil
		},
	})

	r := chi.NewRouter()
	r.Get("/v1/retailer/nasiya/opening-balances", svc.HandleListNasiyaOpeningBalances)
	r.Post("/v1/retailer/ar-invoices/{invoiceID}/acknowledge", svc.HandleAcknowledgeARInvoice)
	r.Post("/v1/retailer/ar-invoices/{invoiceID}/dispute", svc.HandleDisputeARInvoice)

	// 1) List opening balances.
	rrList := httptest.NewRecorder()
	r.ServeHTTP(rrList, retailerReq(http.MethodGet, "/v1/retailer/nasiya/opening-balances", "", "ret-1", ""))
	if rrList.Code != http.StatusOK {
		t.Fatalf("list opening balances status=%d want=200 body=%s", rrList.Code, rrList.Body.String())
	}
	var listResp struct {
		RetailerID string                    `json:"retailer_id"`
		Invoices   []MigrationOpeningInvoice `json:"invoices"`
	}
	if err := json.Unmarshal(rrList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Invoices) != 1 || listResp.Invoices[0].Source != "MIGRATION" || !listResp.Invoices[0].DunningFrozen {
		t.Fatalf("unexpected opening balances list: %+v", listResp.Invoices)
	}

	// 2) Dispute requires non-empty reason and Idempotency-Key.
	rrBadDispute := httptest.NewRecorder()
	r.ServeHTTP(rrBadDispute, retailerReq(http.MethodPost, "/v1/retailer/ar-invoices/inv-mig-1/dispute", `{"reason":""}`, "ret-1", "disp-empty-1"))
	if rrBadDispute.Code != http.StatusBadRequest {
		t.Fatalf("empty dispute reason status=%d want=400", rrBadDispute.Code)
	}

	// 3) Valid dispute keeps dunning frozen and notifies supplier.
	rrDisp := httptest.NewRecorder()
	r.ServeHTTP(rrDisp, retailerReq(http.MethodPost, "/v1/retailer/ar-invoices/inv-mig-1/dispute", `{"reason":"Paid 2,000,000 UZS cash last Friday"}`, "ret-1", "disp-1"))
	if rrDisp.Code != http.StatusOK {
		t.Fatalf("dispute status=%d want=200 body=%s", rrDisp.Code, rrDisp.Body.String())
	}
	if !arMgr.invoices["inv-mig-1"].DunningFrozen {
		t.Fatalf("disputed opening invoice must remain dunning-frozen")
	}

	// 4) Strict idempotency on acknowledge: missing key -> 400, valid key -> 200 & unfreezes dunning.
	rrAckNoIdem := httptest.NewRecorder()
	r.ServeHTTP(rrAckNoIdem, retailerReq(http.MethodPost, "/v1/retailer/ar-invoices/inv-mig-1/acknowledge", `{}`, "ret-1", ""))
	if rrAckNoIdem.Code != http.StatusBadRequest {
		t.Fatalf("ack without idem status=%d want=400", rrAckNoIdem.Code)
	}

	rrAck := httptest.NewRecorder()
	r.ServeHTTP(rrAck, retailerReq(http.MethodPost, "/v1/retailer/ar-invoices/inv-mig-1/acknowledge", `{}`, "ret-1", "ack-1"))
	if rrAck.Code != http.StatusOK {
		t.Fatalf("ack status=%d want=200 body=%s", rrAck.Code, rrAck.Body.String())
	}
	if arMgr.invoices["inv-mig-1"].DunningFrozen || arMgr.invoices["inv-mig-1"].AcknowledgedAt == "" {
		t.Fatalf("acknowledged invoice must release dunning freeze: %+v", arMgr.invoices["inv-mig-1"])
	}

	if len(notifications) != 2 || notifications[0] != "sup-1:AR_MIGRATION_DISPUTED" || notifications[1] != "sup-1:AR_MIGRATION_ACKNOWLEDGED" {
		t.Fatalf("unexpected supplier notifications: %v", notifications)
	}
}

func TestHandleRetailerSetup_DesktopAddressSetsDeliveryAndConfigured(t *testing.T) {
	repo := &statefulRetailerRepo{
		testRetailerRepo: testRetailerRepo{
			found: true,
			retailer: Retailer{
				RetailerID: "ret-desktop-1",
				Phone:      "+998901112233",
			},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		JWTSecret:  "test-secret-key-for-desktop-setup",
		Idem:       idempotency.NewInMemoryStore(),
	})

	rr := httptest.NewRecorder()
	svc.HandleRetailerSetup(rr, retailerReq(http.MethodPost, "/v1/retailer/setup", `{
		"informal_shop":true,
		"legal_entity_status":"INFORMAL_PHONE",
		"billingAddress":"14 Navoi St",
		"shippingAddress":"14 Navoi St",
		"city":"Tashkent",
		"postalCode":"100000"
	}`, "ret-desktop-1", "setup-desktop-1"))

	if rr.Code != http.StatusOK {
		t.Fatalf("desktop setup status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		IsConfigured bool `json:"is_configured"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.IsConfigured {
		t.Fatalf("expected is_configured=true after desktop address setup, body=%s", rr.Body.String())
	}
	if repo.retailer.DeliveryAddress != "14 Navoi St, Tashkent, 100000" {
		t.Fatalf("delivery_address=%q want %q", repo.retailer.DeliveryAddress, "14 Navoi St, Tashkent, 100000")
	}
	if repo.retailer.Lat == 0 && repo.retailer.Lng == 0 {
		t.Fatalf("expected non-zero market coordinates after address setup")
	}
	if repo.retailer.H3Cell == "" {
		t.Fatalf("expected H3Cell to be stamped after address setup")
	}
}
