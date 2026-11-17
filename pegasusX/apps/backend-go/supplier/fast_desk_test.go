package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

type fakeFiscalResolver struct {
	retried []string
	err     error
}

func (f *fakeFiscalResolver) RetryFiscal(_ context.Context, _ auth.Claims, orderID string) (order.CollectCashResponse, error) {
	f.retried = append(f.retried, orderID)
	if f.err != nil {
		return order.CollectCashResponse{}, f.err
	}
	return order.CollectCashResponse{OrderID: orderID, State: order.StatusCompleted}, nil
}

func TestSupplierException6HandlerClassificationSortFilterAndMeta(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Repo:       &pricingTestRepo{},
		SupplierID: "sup-desk-1",
		Now:        func() time.Time { return now },
	})

	if _, _, err := svc.RetailerLinks().Upsert(context.Background(), SupplierRetailerLink{
		SupplierID:      "sup-desk-1",
		RetailerID:      "ret-uuid-1",
		TaxIDTIN:        "301234567",
		NormalizedPhone: "+998901234567",
		LegalName:       "Baraka Savdo LLC",
	}, now); err != nil {
		t.Fatalf("upsert link: %v", err)
	}

	svc.SetExtraExceptionRowsQuery(func(_ context.Context, _ string) ([]SupplierExceptionRow, error) {
		return []SupplierExceptionRow{
			{
				OrderID:          "ord-note-1",
				Kind:             "OPERATOR_NOTE",
				Status:           "CONFIRMED",
				RetailerID:       "ret-uuid-1",
				MoneyAtRiskMinor: 10000000,
				UpdatedAt:        "2026-11-13T09:59:00Z",
			},
			{
				OrderID:          "ord-closed-1",
				Kind:             "SHOP_CLOSED",
				Status:           "PENDING",
				RetailerID:       "ret-uuid-1",
				MoneyAtRiskMinor: 30000000,
				UpdatedAt:        "2026-11-13T09:50:00Z",
			},
			{
				OrderID:          "ord-cash-1",
				Kind:             "CASH_DISCREPANCY",
				Status:           "PENDING",
				DriverID:         "drv-1",
				MoneyAtRiskMinor: 20000000,
				UpdatedAt:        "2026-11-13T09:55:00Z",
			},
			{
				OrderID:          "ord-fiscal-high",
				Kind:             "FISCAL_FAILED",
				Status:           "FISCAL_FAILED",
				RetailerID:       "ret-uuid-1",
				MoneyAtRiskMinor: 85000000,
				UpdatedAt:        "2026-11-13T09:40:00Z",
			},
			{
				OrderID:          "ord-fiscal-dlq",
				Kind:             "FISCAL_DEAD_LETTER",
				Status:           "DELIVERED",
				RetailerID:       "ret-uuid-1",
				MoneyAtRiskMinor: 95000000,
				UpdatedAt:        "2026-11-13T09:30:00Z",
			},
			{
				OrderID:          "claim-1",
				Kind:             "CLAIM_REVIEW",
				Status:           "OPEN",
				RetailerID:       "ret-uuid-1",
				MoneyAtRiskMinor: 15000000,
				UpdatedAt:        "2026-11-13T09:35:00Z",
			},
			{
				OrderID:          "prom-1",
				Kind:             "PROMISE_BREACH",
				Status:           "BREACHED",
				RetailerID:       "ret-uuid-1",
				MoneyAtRiskMinor: 25000000,
				UpdatedAt:        "2026-11-13T09:36:00Z",
			},
		}, nil
	})

	// 1) All exceptions: sorted deterministically CRITICAL > HIGH > MEDIUM > LOW, money DESC.
	rrAll := httptest.NewRecorder()
	svc.HandleExceptions(rrAll, supplierAdminReq(http.MethodGet, "/v1/supplier/exceptions", "", "sup-desk-1"))
	if rrAll.Code != http.StatusOK {
		t.Fatalf("GET /exceptions code=%d body=%s", rrAll.Code, rrAll.Body.String())
	}
	var allResp struct {
		Exceptions []SupplierExceptionRow `json:"exceptions"`
	}
	if err := json.NewDecoder(rrAll.Body).Decode(&allResp); err != nil {
		t.Fatalf("decode exceptions: %v", err)
	}
	if len(allResp.Exceptions) != 7 {
		t.Fatalf("len=%d, want 7", len(allResp.Exceptions))
	}
	// First must be FISCAL_DEAD_LETTER (CRITICAL, 95M), second FISCAL_FAILED (CRITICAL, 85M),
	// third PROMISE_BREACH (CRITICAL since BREACHED, 25M), ... last OPERATOR_NOTE (LOW).
	if allResp.Exceptions[0].OrderID != "ord-fiscal-dlq" || allResp.Exceptions[0].HandlerGroup != HandlerFiscalAndExportBlockers || allResp.Exceptions[0].SeverityTier != SeverityTierCritical {
		t.Fatalf("unexpected #0 exception: %+v", allResp.Exceptions[0])
	}
	if allResp.Exceptions[1].OrderID != "ord-fiscal-high" {
		t.Fatalf("unexpected #1 exception: %+v", allResp.Exceptions[1])
	}
	if allResp.Exceptions[len(allResp.Exceptions)-1].OrderID != "ord-note-1" || allResp.Exceptions[len(allResp.Exceptions)-1].SeverityTier != SeverityTierLow {
		t.Fatalf("unexpected last exception: %+v", allResp.Exceptions[len(allResp.Exceptions)-1])
	}
	if allResp.Exceptions[0].DisplayName != "Baraka Savdo LLC" || allResp.Exceptions[0].TIN != "301234567" || allResp.Exceptions[0].Phone != "+998901234567" {
		t.Fatalf("missing retailer metadata enrichment on row #0: %+v", allResp.Exceptions[0])
	}

	// Verify all 6 handlers are explicitly assigned cleanly.
	handlersSeen := map[string]bool{}
	for _, ex := range allResp.Exceptions {
		handlersSeen[ex.HandlerGroup] = true
	}
	for _, h := range []string{
		HandlerFiscalAndExportBlockers,
		HandlerCashDiscrepancy,
		HandlerClaimsReview,
		HandlerRouteAndClosedDoor,
		HandlerCreditAndPromise,
		HandlerFulfillmentAndCatalog,
	} {
		if !handlersSeen[h] {
			t.Fatalf("handler group %q missing from classified exceptions", h)
		}
	}

	// 2) Filter ?handler=ROUTE_AND_CLOSED_DOOR returns only SHOP_CLOSED.
	rrFiltered := httptest.NewRecorder()
	svc.HandleExceptions(rrFiltered, supplierAdminReq(http.MethodGet, "/v1/supplier/exceptions?handler=ROUTE_AND_CLOSED_DOOR", "", "sup-desk-1"))
	if rrFiltered.Code != http.StatusOK {
		t.Fatalf("filtered exceptions code=%d", rrFiltered.Code)
	}
	var filteredResp struct {
		Exceptions []SupplierExceptionRow `json:"exceptions"`
	}
	if err := json.NewDecoder(rrFiltered.Body).Decode(&filteredResp); err != nil {
		t.Fatalf("decode filtered: %v", err)
	}
	if len(filteredResp.Exceptions) != 1 || filteredResp.Exceptions[0].Kind != "SHOP_CLOSED" {
		t.Fatalf("filtered exceptions = %+v, want 1 SHOP_CLOSED", filteredResp.Exceptions)
	}
}

func TestSupplierExceptionResolveAndBatchResolve(t *testing.T) {
	t.Parallel()

	svc := NewService(ServiceConfig{
		Repo:       &pricingTestRepo{},
		SupplierID: "sup-desk-2",
	})
	fiscalFake := &fakeFiscalResolver{}

	deps := ExceptionResolveDeps{
		Service:      svc,
		FiscalOrders: fiscalFake,
	}
	r := chi.NewRouter()
	r.Post("/v1/supplier/exceptions/batch-resolve", HandleBatchResolveExceptions(deps))
	r.Post("/v1/supplier/exceptions/{kind}/{id}/resolve", HandleResolveException(deps))

	// Single resolve: FISCAL_FAILED retry invokes FiscalOrders.RetryFiscal.
	rrSingle := httptest.NewRecorder()
	r.ServeHTTP(rrSingle, supplierAdminReq(http.MethodPost, "/v1/supplier/exceptions/FISCAL_FAILED/ord-f1/resolve", `{"action":"retry"}`, "sup-desk-2"))
	if rrSingle.Code != http.StatusOK {
		t.Fatalf("single fiscal retry code=%d body=%s", rrSingle.Code, rrSingle.Body.String())
	}
	if len(fiscalFake.retried) != 1 || fiscalFake.retried[0] != "ord-f1" {
		t.Fatalf("retried orders = %v, want [ord-f1]", fiscalFake.retried)
	}

	// Batch resolve: partial success across commercial_override, SHOP_CLOSED, ORDER_CANCELLED, and an unknown kind.
	batchBody := `{"items":[
		{"kind":"FISCAL_DEAD_LETTER","id":"ord-f2","action":"commercial_override","note":"manual e-receipt"},
		{"kind":"SHOP_CLOSED","id":"ord-sc1","action":"BYPASS"},
		{"kind":"ORDER_CANCELLED","id":"ord-c1"},
		{"kind":"UNKNOWN_KIND","id":"bad-1"}
	]}`
	rrBatch := httptest.NewRecorder()
	r.ServeHTTP(rrBatch, supplierAdminReq(http.MethodPost, "/v1/supplier/exceptions/batch-resolve", batchBody, "sup-desk-2"))
	if rrBatch.Code != http.StatusOK {
		t.Fatalf("batch resolve code=%d body=%s", rrBatch.Code, rrBatch.Body.String())
	}
	var batchResp struct {
		Resolved int                   `json:"resolved"`
		Failed   []BatchResolveFailure `json:"failed"`
		Results  []BatchResolveResult  `json:"results"`
	}
	if err := json.NewDecoder(rrBatch.Body).Decode(&batchResp); err != nil {
		t.Fatalf("decode batch resp: %v", err)
	}
	if batchResp.Resolved != 3 || len(batchResp.Failed) != 1 || len(batchResp.Results) != 4 {
		t.Fatalf("unexpected batch resolve response: %+v", batchResp)
	}
	if batchResp.Failed[0].Kind != "UNKNOWN_KIND" || batchResp.Results[3].Status != "failed" {
		t.Fatalf("expected UNKNOWN_KIND failure in results: %+v", batchResp)
	}
}
