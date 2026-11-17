package ar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func arWithURLParam(req *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestWriteOffAndPayInvoice_TenantAuthorization(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	seedInv := func(id, sup, ret string, amt int64) {
		repo.mu.Lock()
		repo.byID[id] = Invoice{
			InvoiceID:      id,
			SupplierID:     sup,
			RetailerID:     ret,
			OrderID:        "ord-" + id,
			Status:         StatusOpen,
			PrincipalMinor: amt,
			BalanceMinor:   amt,
			Currency:       "UZS",
			DueAt:          now.Add(7 * 24 * time.Hour),
		}
		repo.mu.Unlock()
	}
	seedInv("inv-sup-a", "sup-a", "ret-1", 50000)
	seedInv("inv-sup-b", "sup-b", "ret-2", 60000)

	// 1. WriteOffInvoice directly with context claims rejects unscoped or wrong supplier.
	if _, err := svc.WriteOffInvoice(auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-no-sup", Role: auth.RoleAdmin, SupplierID: "",
	}), "inv-sup-a", "admin-no-sup", WriteOffRequest{Reason: "test"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("empty supplier writeoff got %v want ErrForbidden", err)
	}
	if _, err := svc.WriteOffInvoice(auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-b", Role: auth.RoleAdmin, SupplierID: "sup-b",
	}), "inv-sup-a", "admin-b", WriteOffRequest{Reason: "cross"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross supplier writeoff got %v want ErrForbidden", err)
	}

	// HandleInvoiceWriteOff HTTP also rejects empty SupplierID & wrong SupplierID.
	reqWO := httptest.NewRequest(http.MethodPost, "/v1/supplier/ar/invoices/inv-sup-a/write-off", strings.NewReader(`{"reason":"bad debt"}`))
	reqWO = arWithURLParam(reqWO, "id", "inv-sup-a")
	reqWO = reqWO.WithContext(auth.WithClaims(reqWO.Context(), auth.Claims{
		Subject: "admin-no-sup", Role: auth.RoleAdmin, SupplierID: "",
	}))
	rrWO := httptest.NewRecorder()
	svc.HandleInvoiceWriteOff(rrWO, reqWO)
	if rrWO.Code != http.StatusForbidden {
		t.Fatalf("HandleInvoiceWriteOff empty supplier status=%d want 403", rrWO.Code)
	}

	// Owning supplier write-off succeeds.
	reqWOOwner := arWithURLParam(httptest.NewRequest(http.MethodPost, "/v1/supplier/ar/invoices/inv-sup-a/write-off", strings.NewReader(`{"reason":"bad debt"}`)), "id", "inv-sup-a")
	reqWOOwner = reqWOOwner.WithContext(auth.WithClaims(reqWOOwner.Context(), auth.Claims{
		Subject: "admin-a", Role: auth.RoleAdmin, SupplierID: "sup-a",
	}))
	rrWOOwner := httptest.NewRecorder()
	svc.HandleInvoiceWriteOff(rrWOOwner, reqWOOwner)
	if rrWOOwner.Code != http.StatusOK {
		t.Fatalf("HandleInvoiceWriteOff owner status=%d body=%s", rrWOOwner.Code, rrWOOwner.Body.String())
	}

	// 2. HandleRetailerPayInvoice: Driver or cross-supplier Admin or wrong RetailerOrgID must fail 403!
	callPay := func(invID string, c auth.Claims, body string) *httptest.ResponseRecorder {
		req := arWithURLParam(httptest.NewRequest(http.MethodPost, "/v1/retailer/ar/invoices/"+invID+"/pay", strings.NewReader(body)), "id", invID)
		req = req.WithContext(auth.WithClaims(req.Context(), c))
		rr := httptest.NewRecorder()
		svc.HandleRetailerPayInvoice(rr, req)
		return rr
	}
	if rr := callPay("inv-sup-b", auth.Claims{Subject: "drv-1", Role: auth.RoleDriver}, `{"amount_minor":1000}`); rr.Code != http.StatusForbidden {
		t.Fatalf("driver pay status=%d want 403", rr.Code)
	}
	if rr := callPay("inv-sup-b", auth.Claims{Subject: "admin-a", Role: auth.RoleAdmin, SupplierID: "sup-a"}, `{"amount_minor":1000}`); rr.Code != http.StatusForbidden {
		t.Fatalf("cross-supplier admin pay status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
	if rr := callPay("inv-sup-b", auth.Claims{Subject: "ret-2", Role: auth.RoleRetailer, RetailerOrgID: "ret-1"}, `{"amount_minor":1000}`); rr.Code != http.StatusForbidden {
		t.Fatalf("spoofed Subject wrong RetailerOrgID pay status=%d want 403", rr.Code)
	}
	// Staff user under ret-2 org succeeds via ResolveRetailerOrgID!
	if rr := callPay("inv-sup-b", auth.Claims{Subject: "staff-9", Role: auth.RoleRetailer, RetailerOrgID: "ret-2"}, `{"amount_minor":10000}`); rr.Code != http.StatusOK {
		t.Fatalf("valid retailer staff pay status=%d body=%s", rr.Code, rr.Body.String())
	}
	// Service-level empty retailerID with no claims fails closed (no bypass).
	if _, err := svc.RetailerPayInvoice(context.Background(), "", "inv-sup-b", RetailerPayInvoiceRequest{AmountMinor: 1000}, "k1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("empty retailerID without claims got %v want ErrForbidden", err)
	}
}

func TestARCursorPaginationAndMultiPageSweeps(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	t.Setenv("AR_DUNNING_ENABLED", "true")
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	svc := NewService(repo)
	svc.now = func() time.Time { return now }

	// Seed 5 overdue invoices for (sup-1, ret-1) across staggered due dates.
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("inv-%d", i)
		repo.mu.Lock()
		repo.byID[id] = Invoice{
			InvoiceID:       id,
			SupplierID:      "sup-1",
			RetailerID:      "ret-1",
			OrderID:         "ord-" + id,
			Status:          StatusOpen,
			PrincipalMinor:  10000,
			BalanceMinor:    10000,
			Currency:        "UZS",
			DueAt:           now.Add(-time.Duration(10+i*10) * 24 * time.Hour),
			GracePeriodDays: 3,
		}
		repo.mu.Unlock()
	}

	// Paginate HTTP GET /v1/supplier/ar/invoices?limit=2 through all 3 pages.
	seen := make([]string, 0, 5)
	cursor := ""
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/v1/supplier/ar/invoices?limit=2&cursor="+cursor, nil)
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1",
		}))
		rr := httptest.NewRecorder()
		svc.HandleListSupplierInvoices(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var body struct {
			Invoices   []Invoice `json:"invoices"`
			NextCursor string    `json:"next_cursor"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		for _, inv := range body.Invoices {
			seen = append(seen, inv.InvoiceID)
		}
		if body.NextCursor == "" {
			break
		}
		cursor = body.NextCursor
	}
	want := []string{"inv-5", "inv-4", "inv-3", "inv-2", "inv-1"}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("paginated invoices got %v want %v", seen, want)
	}

	// Bad cursor returns 400.
	badReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/ar/invoices?cursor=invalid-cursor", nil)
	badReq = badReq.WithContext(auth.WithClaims(badReq.Context(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}))
	badRR := httptest.NewRecorder()
	svc.HandleListRetailerInvoices(badRR, badReq)
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("bad cursor status=%d want 400", badRR.Code)
	}

	// AdvanceDunning with small page size=2 sweeps all 5 invoices via cursor.
	worker := NewDunningWorker(svc, nil)
	adv, _, _, err := worker.AdvanceDunning(context.Background(), 2)
	if err != nil || adv != 5 {
		t.Fatalf("AdvanceDunning(page=2) advanced=%d err=%v (want 5)", adv, err)
	}
}
