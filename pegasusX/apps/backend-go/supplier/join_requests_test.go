package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
)

func newTestSupplierINN(now time.Time) (*innlookup.Service, *innlookup.StubAdapter, *innlookup.MemoryStore) {
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

func TestHandleJoinRequests_ListApproveRejectAndRBAC(t *testing.T) {
	now := time.Date(2026, 11, 14, 11, 0, 0, 0, time.UTC)
	innSvc, _, mem := newTestSupplierINN(now)

	jrApprove, err := innSvc.CreateJoinRequest(context.Background(), innlookup.JoinRequest{
		HomeCell:         auth.DefaultHomeCell,
		TargetTenantType: innlookup.TenantTypeSupplier,
		TargetTenantID:   "sup-1",
		NormalizedINN:    "301234567",
		RequesterPhone:   "+998901112233",
		RequesterName:    "Aziz",
		RoleSought:       string(auth.RoleWarehouseAdmin),
		CreatedAt:        now,
	}, nil)
	if err != nil {
		t.Fatalf("CreateJoinRequest #1: %v", err)
	}
	jrReject, err := innSvc.CreateJoinRequest(context.Background(), innlookup.JoinRequest{
		HomeCell:         auth.DefaultHomeCell,
		TargetTenantType: innlookup.TenantTypeSupplier,
		TargetTenantID:   "sup-1",
		NormalizedINN:    "301234567",
		RequesterPhone:   "+998904445566",
		RequesterName:    "Bobur",
		RoleSought:       string(auth.RoleDriver),
		CreatedAt:        now,
	}, nil)
	if err != nil {
		t.Fatalf("CreateJoinRequest #2: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo:       &pricingTestRepo{},
		SupplierID: "sup-1",
		INNLookup:  innSvc,
		Now:        func() time.Time { return now },
	})

	r := chi.NewRouter()
	r.Get("/v1/supplier/join-requests", svc.HandleListJoinRequests)
	r.Post("/v1/supplier/join-requests/{requestId}/approve", svc.HandleApproveJoinRequest)
	r.Post("/v1/supplier/join-requests/{requestId}/reject", svc.HandleRejectJoinRequest)

	// 1) List returns both PENDING requests.
	rrList := httptest.NewRecorder()
	r.ServeHTTP(rrList, supplierAdminReq(http.MethodGet, "/v1/supplier/join-requests?status=PENDING", "", "sup-1"))
	if rrList.Code != http.StatusOK {
		t.Fatalf("list status=%d want=200 body=%s", rrList.Code, rrList.Body.String())
	}
	var listResp struct {
		SupplierID string                  `json:"supplier_id"`
		Requests   []innlookup.JoinRequest `json:"requests"`
	}
	if err := json.Unmarshal(rrList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Requests) != 2 {
		t.Fatalf("requests len=%d want=2", len(listResp.Requests))
	}

	// 2) Non-admin role cannot approve/reject.
	nonAdminReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/join-requests/"+jrApprove.RequestID+"/approve", strings.NewReader(`{}`))
	nonAdminReq = nonAdminReq.WithContext(auth.WithClaims(nonAdminReq.Context(), auth.Claims{
		Subject:    "wh-1",
		Role:       auth.RoleWarehouseAdmin,
		SupplierID: "sup-1",
	}))
	rrNonAdmin := httptest.NewRecorder()
	r.ServeHTTP(rrNonAdmin, nonAdminReq)
	if rrNonAdmin.Code != http.StatusForbidden {
		t.Fatalf("non-admin approve status=%d want=403", rrNonAdmin.Code)
	}

	// 3) Admin approves first request -> APPROVED and provisions membership.
	rrApp := httptest.NewRecorder()
	r.ServeHTTP(rrApp, supplierAdminReq(http.MethodPost, "/v1/supplier/join-requests/"+jrApprove.RequestID+"/approve", `{"reason":"Verified manager"}`, "sup-1"))
	if rrApp.Code != http.StatusOK {
		t.Fatalf("approve status=%d want=200 body=%s", rrApp.Code, rrApp.Body.String())
	}
	role, ok := mem.MembershipRole("sup-1", "+998901112233")
	if !ok || role != string(auth.RoleWarehouseAdmin) {
		t.Fatalf("expected membership provisioned for WAREHOUSE_ADMIN, ok=%v role=%q", ok, role)
	}

	// 4) Double decide returns 409 Conflict.
	rrDup := httptest.NewRecorder()
	r.ServeHTTP(rrDup, supplierAdminReq(http.MethodPost, "/v1/supplier/join-requests/"+jrApprove.RequestID+"/reject", `{"reason":"too late"}`, "sup-1"))
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("double-decide status=%d want=409 body=%s", rrDup.Code, rrDup.Body.String())
	}

	// 5) Admin rejects second request -> REJECTED.
	rrRej := httptest.NewRecorder()
	r.ServeHTTP(rrRej, supplierAdminReq(http.MethodPost, "/v1/supplier/join-requests/"+jrReject.RequestID+"/reject", `{"reason":"Unknown contractor"}`, "sup-1"))
	if rrRej.Code != http.StatusOK {
		t.Fatalf("reject status=%d want=200 body=%s", rrRej.Code, rrRej.Body.String())
	}
	if _, ok := mem.MembershipRole("sup-1", "+998904445566"); ok {
		t.Fatalf("rejected requester must not receive membership")
	}
}

func TestHandleSupplierBusinessSetup_DuplicateINN_Returns409(t *testing.T) {
	now := time.Date(2026, 11, 14, 11, 0, 0, 0, time.UTC)
	innSvc, stub, mem := newTestSupplierINN(now)
	stub.Register("301234567", "Owner Distributor MChJ", "Tashkent")
	mem.RegisterTenant(innlookup.TenantMatch{
		HomeCell:      auth.DefaultHomeCell,
		TenantType:    innlookup.TenantTypeSupplier,
		TenantID:      "sup-owner",
		LegalName:     "Owner Distributor MChJ",
		NormalizedINN: "301234567",
	})

	repo := &pricingTestRepo{
		profileFound: true,
		profile: Profile{
			SupplierID:  "sup-new",
			LegalName:   "New Applicant",
			Phone:       "+998907771122",
			Country:     "UZ",
			MarketCode:  "UZ",
			HomeCell:    auth.DefaultHomeCell,
		},
	}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-new",
		INNLookup:  innSvc,
		Now:        func() time.Time { return now },
	})

	rr := httptest.NewRecorder()
	svc.HandleSupplierBusinessSetup(rr, supplierAdminReq(http.MethodPost, "/v1/supplier/setup/business", `{
		"taxId":"INN 301-234 567",
		"headquartersAddress":"Chilonzor 12",
		"city":"Tashkent"
	}`, "sup-new"))

	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate business setup INN status=%d want=409 body=%s", rr.Code, rr.Body.String())
	}
	items, err := innSvc.ListJoinRequests(context.Background(), innlookup.TenantTypeSupplier, "sup-owner", innlookup.JoinStatusPending, 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected PENDING join request for sup-owner, len=%d err=%v", len(items), err)
	}
}
