package tenantreg

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

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

type innTestRegistry struct {
	mu       sync.Mutex
	profiles map[string]supplier.Profile
	byPhone  map[string]supplier.SupplierAuthRecord
	updates  int
}

func newINNTestRegistry() *innTestRegistry {
	return &innTestRegistry{
		profiles: make(map[string]supplier.Profile),
		byPhone:  make(map[string]supplier.SupplierAuthRecord),
	}
}

func (r *innTestRegistry) GetAuthByPhone(_ context.Context, phone string) (supplier.SupplierAuthRecord, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byPhone[phone]
	return rec, ok, nil
}

func (r *innTestRegistry) GetProfile(_ context.Context, supplierID string) (supplier.Profile, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[supplierID]
	return p, ok, nil
}

func (r *innTestRegistry) UpdateProfile(_ context.Context, p supplier.Profile, _ func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates++
	r.profiles[p.SupplierID] = p
	r.byPhone[p.Phone] = supplier.SupplierAuthRecord{
		SupplierID:   p.SupplierID,
		Phone:        p.Phone,
		PasswordHash: p.AuthPasswordHash,
	}
	return nil
}

func newTestINNService(now time.Time) (*innlookup.Service, *innlookup.StubAdapter, *innlookup.MemoryStore) {
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

func TestRegisterWithValidINN_SetsNormalizedTaxIDAndInnVerifiedAt(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	reg := newINNTestRegistry()
	innSvc, stub, _ := newTestINNService(now)
	stub.Register("301234567", "Baraka Savdo MChJ", "Tashkent, Yunusabad")

	svc := NewService(Config{
		Repo:        reg,
		JWTSecret:   "test-secret-0123456789abcdef0123456789",
		Idem:        idempotency.NewInMemoryStore(),
		INNLookup:   innSvc,
		INNRequired: true,
		Now:         func() time.Time { return now },
		NewID:       func() string { return "sup-verified-1" },
	})

	resp, err := svc.Register(context.Background(), Request{
		LegalName:  "Baraka Savdo",
		Phone:      "+998901112233",
		Password:   "Str0ngPass!",
		MarketCode: "UZ",
		TaxID:      "INN 301-234 567",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if resp.NormalizedTaxID != "301234567" {
		t.Fatalf("NormalizedTaxID=%q want=301234567", resp.NormalizedTaxID)
	}
	if resp.InnVerifiedAt == "" {
		t.Fatalf("expected non-empty InnVerifiedAt")
	}
	if resp.HomeCell != auth.DefaultHomeCell {
		t.Fatalf("HomeCell=%q want=%q", resp.HomeCell, auth.DefaultHomeCell)
	}
	prof, ok := reg.profiles[resp.SupplierID]
	if !ok {
		t.Fatalf("missing persisted profile %s", resp.SupplierID)
	}
	if prof.NormalizedTaxID != "301234567" || prof.InnVerifiedAt.IsZero() {
		t.Fatalf("profile INN fields not persisted: norm=%q verifiedAt=%v", prof.NormalizedTaxID, prof.InnVerifiedAt)
	}
}

func TestRegisterWithINN_FailClosedOnUnavailableAndNotFound(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	reg := newINNTestRegistry()
	innSvc, stub, _ := newTestINNService(now)
	stub.SetForceUnavailable(true)

	svc := NewService(Config{
		Repo:        reg,
		JWTSecret:   "test-secret-0123456789abcdef0123456789",
		Idem:        idempotency.NewInMemoryStore(),
		INNLookup:   innSvc,
		INNRequired: true,
		Now:         func() time.Time { return now },
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", strings.NewReader(`{
		"legal_name":"Fail Closed LLC",
		"phone":"+998901112244",
		"password":"Str0ngPass!",
		"market_code":"UZ",
		"tax_id":"301234567"
	}`))
	req.Header.Set("Idempotency-Key", "idem-unavail-1")
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable registry status=%d want=503 body=%s", rr.Code, rr.Body.String())
	}
	if reg.updates != 0 {
		t.Fatalf("expected zero tenants minted when registry unavailable, got %d", reg.updates)
	}

	stub.SetForceUnavailable(false)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", strings.NewReader(`{
		"legal_name":"Unknown INN LLC",
		"phone":"+998901112255",
		"password":"Str0ngPass!",
		"market_code":"UZ",
		"tax_id":"309998887"
	}`))
	req2.Header.Set("Idempotency-Key", "idem-notfound-1")
	rr2 := httptest.NewRecorder()
	svc.HandleRegister(rr2, req2)

	if rr2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("not found INN status=%d want=422 body=%s", rr2.Code, rr2.Body.String())
	}
	if reg.updates != 0 {
		t.Fatalf("expected zero tenants minted for unknown INN, got %d", reg.updates)
	}
}

func TestRegisterWithDuplicateINN_CreatesJoinRequestReturns409_NoSecondTenant(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	reg := newINNTestRegistry()
	innSvc, stub, mem := newTestINNService(now)
	stub.Register("301234567", "Baraka Savdo MChJ", "Tashkent")
	mem.RegisterTenant(innlookup.TenantMatch{
		HomeCell:      auth.DefaultHomeCell,
		TenantType:    innlookup.TenantTypeSupplier,
		TenantID:      "sup-existing",
		LegalName:     "Baraka Savdo MChJ",
		NormalizedINN: "301234567",
	})

	var notifiedSupplierID string
	var notifiedJR innlookup.JoinRequest
	svc := NewService(Config{
		Repo:        reg,
		JWTSecret:   "test-secret-0123456789abcdef0123456789",
		Idem:        idempotency.NewInMemoryStore(),
		INNLookup:   innSvc,
		INNRequired: true,
		Now:         func() time.Time { return now },
		OnJoinRequest: func(_ context.Context, supplierID string, jr innlookup.JoinRequest) {
			notifiedSupplierID = supplierID
			notifiedJR = jr
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/register", strings.NewReader(`{
		"legal_name":"Baraka Savdo MChJ",
		"contact_name":"Dilshod",
		"phone":"+998907778899",
		"password":"Str0ngPass!",
		"market_code":"UZ",
		"tax_id":"INN 301-234-567",
		"role_sought":"WAREHOUSE_ADMIN"
	}`))
	req.Header.Set("Idempotency-Key", "idem-dup-inn-1")
	rr := httptest.NewRecorder()
	svc.HandleRegister(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate INN status=%d want=409 body=%s", rr.Code, rr.Body.String())
	}
	if reg.updates != 0 {
		t.Fatalf("expected 0 new tenants minted on duplicate INN, got %d", reg.updates)
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode 409 payload: %v", err)
	}
	if payload["error"] != innlookup.ErrDuplicateTenant.Error() {
		t.Fatalf("error=%v want=%s", payload["error"], innlookup.ErrDuplicateTenant.Error())
	}
	if payload["target_tenant_id"] != "sup-existing" {
		t.Fatalf("target_tenant_id=%v want=sup-existing", payload["target_tenant_id"])
	}
	if notifiedSupplierID != "sup-existing" || notifiedJR.RoleSought != string(auth.RoleWarehouseAdmin) {
		t.Fatalf("unexpected join notification: sid=%q jr=%+v", notifiedSupplierID, notifiedJR)
	}

	items, err := innSvc.ListJoinRequests(context.Background(), innlookup.TenantTypeSupplier, "sup-existing", innlookup.JoinStatusPending, 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 PENDING JoinRequest, len=%d err=%v", len(items), err)
	}
	if items[0].NormalizedINN != "301234567" || items[0].RequesterPhone != "+998907778899" {
		t.Fatalf("unexpected JoinRequest row: %+v", items[0])
	}
	if items[0].RequesterPasswordHash != "" || notifiedJR.RequesterPasswordHash != "" {
		t.Fatalf("requester_password_hash must never be returned via ListJoinRequests or notifications")
	}
}

func TestRegisterWithSameINN_DifferentCell_NoCollision(t *testing.T) {
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	reg := newINNTestRegistry()
	innSvc, stub, mem := newTestINNService(now)
	stub.Register("301234567", "Baraka KZ", "Almaty")
	mem.RegisterTenant(innlookup.TenantMatch{
		HomeCell:      "kz-01",
		TenantType:    innlookup.TenantTypeSupplier,
		TenantID:      "sup-kz-1",
		LegalName:     "Baraka KZ",
		NormalizedINN: "301234567",
	})

	svc := NewService(Config{
		Repo:        reg,
		JWTSecret:   "test-secret-0123456789abcdef0123456789",
		Idem:        idempotency.NewInMemoryStore(),
		INNLookup:   innSvc,
		INNRequired: true,
		Now:         func() time.Time { return now },
		NewID:       func() string { return "sup-uz-1" },
	})

	resp, err := svc.Register(context.Background(), Request{
		LegalName:  "Baraka UZ",
		Phone:      "+998903334455",
		Password:   "Str0ngPass!",
		MarketCode: "UZ",
		TaxID:      "301234567",
	})
	if err != nil {
		var jp *JoinRequestPendingError
		if errors.As(err, &jp) {
			t.Fatalf("same INN in kz-01 must not collide with UZ cell uz-01: %+v", jp)
		}
		t.Fatalf("Register error: %v", err)
	}
	if resp.SupplierID != "sup-uz-1" || resp.NormalizedTaxID != "301234567" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}
