package creditroutes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func issueW3Token(t *testing.T, role auth.Role, retailerOrgID, supplierID string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:       "usr-" + string(role),
		Role:          role,
		RetailerOrgID: retailerOrgID,
		SupplierID:    supplierID,
	}, auth.IssueOptions{Secret: "credit-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestCreditRoutes_W3Endpoints_StatementFIFOAndPromises(t *testing.T) {
	t.Setenv("AR_INVOICES_ENABLED", "true")
	t.Setenv("AR_MAX_PROMISE_DAYS", "7")

	arRepo := ar.NewMemoryRepository()
	arSvc := ar.NewService(arRepo)
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	arSvc.SetNow(func() time.Time { return now })

	ctx := context.Background()
	_ = arRepo.OpenInvoice(ctx, ar.Invoice{
		InvoiceID: "ari-1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-1",
		Status: ar.StatusOpen, PrincipalMinor: 100_000, BalanceMinor: 100_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -10), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = arRepo.OpenInvoice(ctx, ar.Invoice{
		InvoiceID: "ari-2", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-2",
		Status: ar.StatusPartial, PrincipalMinor: 100_000, BalanceMinor: 80_000, Currency: "UZS",
		DueAt: now.AddDate(0, 0, -2), Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	arSvc.SetDisputeLookup(func(_ context.Context, _ []string) (map[string]int64, error) {
		return map[string]int64{"ord-2": 20_000}, nil
	})

	var breached int
	pw := ar.NewPromiseBreachWorker(arSvc, nil)
	pw.SetOnBreach(func(_ context.Context, _, _, _ string) error {
		breached++
		return nil
	})
	dw := ar.NewDunningWorker(arSvc, nil)

	stepUpCalled := false
	stepUp := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stepUpCalled = true
			next.ServeHTTP(w, r)
		})
	}

	r := creditRouter(Deps{
		ARService:           arSvc,
		DunningWorker:       dw,
		PromiseBreachWorker: pw,
		StepUp:              stepUp,
	})

	retTok := issueW3Token(t, auth.RoleRetailer, "ret-1", "")
	supAdminTok := issueW3Token(t, auth.RoleAdmin, "", "sup-1")
	whTok := issueW3Token(t, auth.RoleWarehouse, "", "sup-1")
	platTok := issueW3Token(t, auth.RolePlatformAdmin, "", "")

	// 1. GET /v1/retailer/ar/invoices?status=OPEN,PARTIAL
	reqInv := httptest.NewRequest(http.MethodGet, "/v1/retailer/ar/invoices?status=OPEN,PARTIAL", nil)
	reqInv.Header.Set("Authorization", "Bearer "+retTok)
	rrInv := httptest.NewRecorder()
	r.ServeHTTP(rrInv, reqInv)
	if rrInv.Code != http.StatusOK {
		t.Fatalf("invoices status=%d body=%s", rrInv.Code, rrInv.Body.String())
	}
	var invResp struct {
		Invoices []ar.Invoice `json:"invoices"`
	}
	if err := json.Unmarshal(rrInv.Body.Bytes(), &invResp); err != nil || len(invResp.Invoices) != 2 {
		t.Fatalf("unexpected invoices resp: %s err=%v", rrInv.Body.String(), err)
	}

	// 2. GET /v1/retailer/ar/statement?supplier_id=sup-1
	reqStmt := httptest.NewRequest(http.MethodGet, "/v1/retailer/ar/statement?supplier_id=sup-1&limit=50", nil)
	reqStmt.Header.Set("Authorization", "Bearer "+retTok)
	rrStmt := httptest.NewRecorder()
	r.ServeHTTP(rrStmt, reqStmt)
	if rrStmt.Code != http.StatusOK {
		t.Fatalf("statement status=%d body=%s", rrStmt.Code, rrStmt.Body.String())
	}
	var stmt ar.StatementResponse
	if err := json.Unmarshal(rrStmt.Body.Bytes(), &stmt); err != nil {
		t.Fatal(err)
	}
	if stmt.TotalOpenMinor != 180_000 || stmt.TotalDisputedMinor != 20_000 || stmt.TotalUndisputedMinor != 160_000 ||
		stmt.CurrentQuarterKey != "2026-Q4" || stmt.MaxExtensionDays != 7 || stmt.SelfServePromiseUsed {
		t.Fatalf("unexpected statement: %+v", stmt)
	}

	// 3. POST /v1/retailer/ar/promises -> 201 Created, 2nd -> 409 quarterly_promise_limit_reached
	reqProm1 := httptest.NewRequest(http.MethodPost, "/v1/retailer/ar/promises", strings.NewReader(`{"supplier_id":"sup-1","invoice_id":"ari-1","extension_days":5}`))
	reqProm1.Header.Set("Authorization", "Bearer "+retTok)
	reqProm1.Header.Set("Idempotency-Key", "prom-1")
	rrProm1 := httptest.NewRecorder()
	r.ServeHTTP(rrProm1, reqProm1)
	if rrProm1.Code != http.StatusCreated {
		t.Fatalf("create promise 1 status=%d body=%s", rrProm1.Code, rrProm1.Body.String())
	}

	reqProm2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/ar/promises", strings.NewReader(`{"supplier_id":"sup-1","invoice_id":"ari-2","extension_days":3}`))
	reqProm2.Header.Set("Authorization", "Bearer "+retTok)
	reqProm2.Header.Set("Idempotency-Key", "prom-2")
	rrProm2 := httptest.NewRecorder()
	r.ServeHTTP(rrProm2, reqProm2)
	if rrProm2.Code != http.StatusConflict {
		t.Fatalf("create promise 2 status=%d want 409 body=%s", rrProm2.Code, rrProm2.Body.String())
	}
	var err409 map[string]any
	if err := json.Unmarshal(rrProm2.Body.Bytes(), &err409); err != nil ||
		err409["code"] != "quarterly_promise_limit_reached" || err409["error"] != "quarterly_promise_limit_reached" || err409["quarter_key"] != "2026-Q4" {
		t.Fatalf("unexpected 409 payload: %+v", err409)
	}

	// 4. GET /v1/retailer/ar/promises
	reqListProm := httptest.NewRequest(http.MethodGet, "/v1/retailer/ar/promises?supplier_id=sup-1", nil)
	reqListProm.Header.Set("Authorization", "Bearer "+retTok)
	rrListProm := httptest.NewRecorder()
	r.ServeHTTP(rrListProm, reqListProm)
	if rrListProm.Code != http.StatusOK || !strings.Contains(rrListProm.Body.String(), `"self_serve_promise_used":true`) {
		t.Fatalf("retailer list promises: %d %s", rrListProm.Code, rrListProm.Body.String())
	}

	// 5. RBAC on /v1/supplier/ar/promises
	reqWhPost := httptest.NewRequest(http.MethodPost, "/v1/supplier/ar/promises", strings.NewReader(`{"retailer_id":"ret-1","extension_days":5}`))
	reqWhPost.Header.Set("Authorization", "Bearer "+whTok)
	reqWhPost.Header.Set("Idempotency-Key", "wh-post-1")
	rrWhPost := httptest.NewRecorder()
	r.ServeHTTP(rrWhPost, reqWhPost)
	if rrWhPost.Code != http.StatusForbidden {
		t.Fatalf("warehouse POST override status=%d want 403", rrWhPost.Code)
	}

	reqSupPost := httptest.NewRequest(http.MethodPost, "/v1/supplier/ar/promises", strings.NewReader(`{"retailer_id":"ret-1","invoice_id":"ari-1","extension_days":4}`))
	reqSupPost.Header.Set("Authorization", "Bearer "+supAdminTok)
	reqSupPost.Header.Set("Idempotency-Key", "sup-post-1")
	rrSupPost := httptest.NewRecorder()
	r.ServeHTTP(rrSupPost, reqSupPost)
	if rrSupPost.Code != http.StatusCreated {
		t.Fatalf("supplier admin grant promise status=%d want 201 body=%s", rrSupPost.Code, rrSupPost.Body.String())
	}

	// 6. POST /v1/retailer/ar/pay-fifo
	reqFIFO := httptest.NewRequest(http.MethodPost, "/v1/retailer/ar/pay-fifo", strings.NewReader(`{"supplier_id":"sup-1","amount_minor":200000,"currency":"UZS","payment_method":"WALLET"}`))
	reqFIFO.Header.Set("Authorization", "Bearer "+retTok)
	reqFIFO.Header.Set("Idempotency-Key", "http-fifo-1")
	rrFIFO := httptest.NewRecorder()
	r.ServeHTTP(rrFIFO, reqFIFO)
	if rrFIFO.Code != http.StatusOK {
		t.Fatalf("pay-fifo status=%d body=%s", rrFIFO.Code, rrFIFO.Body.String())
	}
	var fifoResp ar.PayFIFOResponse
	if err := json.Unmarshal(rrFIFO.Body.Bytes(), &fifoResp); err != nil {
		t.Fatal(err)
	}
	if fifoResp.PaidMinor != 160_000 || fifoResp.AdvanceAddedMinor != 40_000 || fifoResp.AdvanceBalanceMinor != 40_000 {
		t.Fatalf("unexpected pay-fifo response: %+v", fifoResp)
	}

	// 7. POST /v1/admin/ar/promises/run-once (StepUp + RBAC)
	reqRunDenied := httptest.NewRequest(http.MethodPost, "/v1/admin/ar/promises/run-once", nil)
	reqRunDenied.Header.Set("Authorization", "Bearer "+retTok)
	reqRunDenied.Header.Set("Idempotency-Key", "run-denied")
	rrRunDenied := httptest.NewRecorder()
	r.ServeHTTP(rrRunDenied, reqRunDenied)
	if rrRunDenied.Code != http.StatusForbidden {
		t.Fatalf("retailer run-once status=%d want 403", rrRunDenied.Code)
	}

	reqRunOK := httptest.NewRequest(http.MethodPost, "/v1/admin/ar/promises/run-once", nil)
	reqRunOK.Header.Set("Authorization", "Bearer "+platTok)
	reqRunOK.Header.Set("Idempotency-Key", "run-ok")
	rrRunOK := httptest.NewRecorder()
	r.ServeHTTP(rrRunOK, reqRunOK)
	if rrRunOK.Code != http.StatusOK || !stepUpCalled || breached != 0 {
		t.Fatalf("run-once status=%d stepUp=%v breached=%d body=%s", rrRunOK.Code, stepUpCalled, breached, rrRunOK.Body.String())
	}
	_ = chi.NewRouter()
}
