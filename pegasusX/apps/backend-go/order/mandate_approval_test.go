package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

type mandateMemRepo struct {
	Repository
	mu             sync.Mutex
	orders         map[string]Order
	createCalls    int
	failSupplierID string
}

func newMandateMemRepo() *mandateMemRepo {
	return &mandateMemRepo{
		orders: make(map[string]Order),
	}
}

func (r *mandateMemRepo) CreateOrder(_ context.Context, o *Order, _ func(outbox.TxnBuffer) error, _ StockReservationOpts) error {
	if o == nil {
		return errors.New("nil order")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failSupplierID != "" && o.SupplierID == r.failSupplierID {
		return errors.New("simulated supplier checkout failure")
	}
	r.createCalls++
	r.orders[o.OrderID] = *o
	return nil
}

func (r *mandateMemRepo) CreateOrderWithBackorder(ctx context.Context, o *Order, _ *Order, _ func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, opts StockReservationOpts) error {
	return r.CreateOrder(ctx, o, emit, opts)
}

func (r *mandateMemRepo) GetOrder(_ context.Context, orderID string) (Order, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orders[orderID]
	return o, ok, nil
}

func (r *mandateMemRepo) GetOrderTxn(ctx context.Context, _ *spanner.ReadWriteTransaction, orderID string) (Order, bool, error) {
	return r.GetOrder(ctx, orderID)
}

func (r *mandateMemRepo) UpdateOrder(_ context.Context, o Order, _ []DeliveryProofArtifact, _ func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.OrderID] = o
	return nil
}

func (r *mandateMemRepo) countCreated() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createCalls
}

type staticWarehouseResolver struct {
	warehouseID string
}

func (w staticWarehouseResolver) ResolveNearestWarehouseID(context.Context, string, proximity.StorePoint) (string, error) {
	return w.warehouseID, nil
}

type mapMandateReader struct {
	mu      sync.RWMutex
	entries map[string]MandateLookupResult
}

func newMapMandateReader() *mapMandateReader {
	return &mapMandateReader{entries: make(map[string]MandateLookupResult)}
}

func (m *mapMandateReader) Set(retailerID, userID string, res MandateLookupResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[retailerID+"|"+userID] = res
}

func (m *mapMandateReader) ResolveSellerMandate(_ context.Context, retailerID, userID, fallbackRole string) (MandateLookupResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if res, ok := m.entries[retailerID+"|"+userID]; ok {
		return res, nil
	}
	isOwner := fallbackRole == "OWNER"
	return MandateLookupResult{
		Mandate: DefaultSellerMandateForRole(fallbackRole, isOwner),
		Role:    fallbackRole,
		IsOwner: isOwner,
		Found:   false,
	}, nil
}

func newMandateTestService(repo *mandateMemRepo, reader MandateReader, nowFn func() time.Time) *Service {
	var seq atomic.Int64
	svc := NewService(ServiceConfig{
		Repo:          repo,
		Warehouse:     staticWarehouseResolver{warehouseID: "wh-1"},
		SupplierID:    "sup-1",
		Currency:      "UZS",
		MandateReader: reader,
		Log:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:           nowFn,
		NewID: func() string {
			return fmt.Sprintf("id-%d", seq.Add(1))
		},
	})
	return svc
}

func TestSellerMandateGate_Create_RulesAndFallbacks(t *testing.T) {
	t.Parallel()
	fixedNow := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	repo := newMandateMemRepo()
	reader := newMapMandateReader()
	reader.Set("ret-1", "seller-cashier", MandateLookupResult{
		Role:  "CASHIER",
		Found: true,
		Mandate: SellerMandate{
			CanUseCredit:        false,
			CanPayCash:          true,
			DailyCashCapMinor:   50000,
			MaxSingleOrderMinor: 25000,
		},
	})
	reader.Set("ret-1", "seller-cardonly", MandateLookupResult{
		Role:  "SELLER",
		Found: true,
		Mandate: SellerMandate{
			CanUseCredit: false,
			CanPayCash:   false,
		},
	})
	svc := newMandateTestService(repo, reader, func() time.Time { return fixedNow })

	sellerClaims := auth.Claims{
		Subject:        "seller-cashier",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "seller-cashier",
		RetailerRole:   "CASHIER",
	}

	// 1. Credit request without credit mandate returns 409 Conflict with CASH + CARD fallback.
	creditBody := `{"payment_method":"CREDIT","line_items":[{"sku":"SKU-1","quantity":2,"unit_price_minor":5000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewBufferString(creditBody))
	req = req.WithContext(auth.WithClaims(req.Context(), sellerClaims))
	req.Header.Set("Idempotency-Key", "client-order-1")
	rr := httptest.NewRecorder()
	svc.HandleCreate(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("HandleCreate credit status = %d, want 409 (%s)", rr.Code, rr.Body.String())
	}
	var conflictBody struct {
		Code                   string   `json:"code"`
		Error                  string   `json:"error"`
		Status                 string   `json:"status"`
		TrippedRule            string   `json:"tripped_rule"`
		AmountMinor            int64    `json:"amount_minor"`
		FallbackPaymentMethods []string `json:"fallback_payment_methods"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &conflictBody); err != nil {
		t.Fatalf("decode 409 json: %v", err)
	}
	if conflictBody.Code != "owner_approval_required" || conflictBody.Error != "owner_approval_required" {
		t.Fatalf("unexpected error/code in 409: %+v", conflictBody)
	}
	if conflictBody.TrippedRule != "CREDIT_MANDATE_DISABLED" || conflictBody.Status != "PENDING_OWNER" {
		t.Fatalf("unexpected rule/status in 409: %+v", conflictBody)
	}
	if conflictBody.AmountMinor != 10000 {
		t.Fatalf("amount_minor = %d, want 10000", conflictBody.AmountMinor)
	}
	fallbacks := conflictBody.FallbackPaymentMethods
	if len(fallbacks) != 2 || fallbacks[0] != "CASH" || fallbacks[1] != "CARD" {
		t.Fatalf("fallback_payment_methods = %v, want [CASH CARD]", conflictBody.FallbackPaymentMethods)
	}
	if repo.countCreated() != 0 {
		t.Fatalf("expected 0 orders persisted on 409, got %d", repo.countCreated())
	}

	// 2. Resubmitting with 1-tap CASH fallback succeeds with 201 Created and increments daily cash usage.
	cashBody := `{"payment_method":"CASH","line_items":[{"sku":"SKU-1","quantity":2,"unit_price_minor":5000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`
	reqCash := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewBufferString(cashBody))
	reqCash = reqCash.WithContext(auth.WithClaims(reqCash.Context(), sellerClaims))
	reqCash.Header.Set("Idempotency-Key", "client-order-1")
	rrCash := httptest.NewRecorder()
	svc.HandleCreate(rrCash, reqCash)

	if rrCash.Code != http.StatusCreated {
		t.Fatalf("HandleCreate cash fallback status = %d, want 201 (%s)", rrCash.Code, rrCash.Body.String())
	}
	usage, uErr := svc.GetSellerDailyUsage(context.Background(), "ret-1", "seller-cashier", civil.DateOf(fixedNow))
	if uErr != nil {
		t.Fatalf("GetSellerDailyUsage err: %v", uErr)
	}
	if usage.SpentCashMinor != 10000 || usage.SpentCreditMinor != 0 || usage.OrderCount != 1 {
		t.Fatalf("unexpected usage after cash fallback: %+v", usage)
	}

	// 3. When CanPayCash=false, fallback_payment_methods excludes CASH and offers CARD only.
	cardOnlyClaims := sellerClaims
	cardOnlyClaims.Subject = "seller-cardonly"
	cardOnlyClaims.RetailerUserID = "seller-cardonly"
	cardOnlyClaims.RetailerRole = "SELLER"
	req3 := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewBufferString(creditBody))
	req3 = req3.WithContext(auth.WithClaims(req3.Context(), cardOnlyClaims))
	req3.Header.Set("Idempotency-Key", "client-order-cardonly")
	rr3 := httptest.NewRecorder()
	svc.HandleCreate(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("cardonly status = %d, want 409", rr3.Code)
	}
	var c3 map[string]any
	if err := json.Unmarshal(rr3.Body.Bytes(), &c3); err != nil {
		t.Fatal(err)
	}
	fb3, _ := c3["fallback_payment_methods"].([]any)
	if len(fb3) != 1 || fb3[0] != "CARD" {
		t.Fatalf("cardonly fallback_payment_methods = %v, want [CARD]", c3["fallback_payment_methods"])
	}

	// 4. OWNER bypasses seller caps completely even on huge credit order.
	ownerClaims := auth.Claims{
		Subject:        "owner-1",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "owner-1",
		RetailerRole:   "OWNER",
	}
	hugeBody := `{"payment_method":"CREDIT","line_items":[{"sku":"SKU-1","quantity":10,"unit_price_minor":50000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`
	reqOwner := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewBufferString(hugeBody))
	reqOwner = reqOwner.WithContext(auth.WithClaims(reqOwner.Context(), ownerClaims))
	reqOwner.Header.Set("Idempotency-Key", "owner-order-1")
	rrOwner := httptest.NewRecorder()
	svc.HandleCreate(rrOwner, reqOwner)
	if rrOwner.Code != http.StatusCreated {
		t.Fatalf("owner order status = %d, want 201 (%s)", rrOwner.Code, rrOwner.Body.String())
	}
}

func TestSellerMandateGate_UnifiedCheckout_WholeCartPreGateAndIdempotency(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "true")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	fixedNow := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	repo := newMandateMemRepo()
	reader := newMapMandateReader()
	reader.Set("ret-1", "buyer-1", MandateLookupResult{
		Role:  "BUYER",
		Found: true,
		Mandate: SellerMandate{
			CanUseCredit:        true,
			CanPayCash:          true,
			DailyCreditCapMinor: 10000, // combined 6000 + 7000 = 13000 > 10000
			DailyCashCapMinor:   50000,
			MaxSingleOrderMinor: 20000,
		},
	})
	svc := newMandateTestService(repo, reader, func() time.Time { return fixedNow })

	buyerClaims := auth.Claims{
		Subject:        "buyer-1",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "buyer-1",
		RetailerRole:   "BUYER",
	}
	unifiedPayload := `{
		"retailer_id": "ret-1",
		"payment_method": "CREDIT",
		"latitude": 41.3,
		"longitude": 69.2,
		"items": [
			{"supplier_id":"sup-a","sku_id":"SKU-A","quantity":1,"unit_price":6000},
			{"supplier_id":"sup-b","sku_id":"SKU-B","quantity":1,"unit_price":7000}
		]
	}`

	var firstReqID string
	for attempt := range 2 {
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", bytes.NewBufferString(unifiedPayload))
		req = req.WithContext(auth.WithClaims(req.Context(), buyerClaims))
		req.Header.Set("Idempotency-Key", "whole-cart-key-1")
		rr := httptest.NewRecorder()
		svc.HandleUnifiedCheckout(rr, req)
		if rr.Code != http.StatusConflict {
			t.Fatalf("attempt %d status = %d, want 409 (%s)", attempt, rr.Code, rr.Body.String())
		}
		var out struct {
			RequestID   string `json:"request_id"`
			TrippedRule string `json:"tripped_rule"`
			AmountMinor int64  `json:"amount_minor"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.TrippedRule != "DAILY_CREDIT_CAP_EXCEEDED" || out.AmountMinor != 13000 {
			t.Fatalf("unexpected 409 payload: %+v", out)
		}
		rid := out.RequestID
		if rid == "" {
			t.Fatalf("missing request_id in 409 payload")
		}
		if attempt == 0 {
			firstReqID = rid
		} else if rid != firstReqID {
			t.Fatalf("idempotent approval upsert failed: got %q, want %q", rid, firstReqID)
		}
	}
	if repo.countCreated() != 0 {
		t.Fatalf("pre-saga gate should block before child orders are created, got createCalls=%d", repo.countCreated())
	}
}

func approvalTestRouter(svc *Service) http.Handler {
	r := chi.NewRouter()
	r.Get("/v1/retailer/approvals", svc.HandleListApprovals)
	r.Post("/v1/retailer/approvals/{requestID}/approve", svc.HandleApproveRequest)
	r.Post("/v1/retailer/approvals/{requestID}/reject", svc.HandleRejectRequest)
	return r
}

func TestApprovalsQueue_CursorPagination_ApproveReplay_Reject(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "true")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	nowCounter := atomic.Int64{}
	baseTime := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	repo := newMandateMemRepo()
	reader := newMapMandateReader()
	buyerMandate := MandateLookupResult{
		Role:  "BUYER",
		Found: true,
		Mandate: SellerMandate{
			CanUseCredit:        true,
			CanPayCash:          true,
			DailyCreditCapMinor: 10000,
			MaxSingleOrderMinor: 20000,
		},
	}
	reader.Set("ret-1", "buyer-1", buyerMandate)
	reader.Set("ret-1", "buyer-2", buyerMandate)
	svc := newMandateTestService(repo, reader, func() time.Time {
		return baseTime.Add(time.Duration(nowCounter.Add(1)) * time.Second)
	})
	router := approvalTestRouter(svc)

	buyerClaims := auth.Claims{
		Subject:        "buyer-1",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "buyer-1",
		RetailerRole:   "BUYER",
	}
	otherSellerClaims := auth.Claims{
		Subject:        "buyer-2",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "buyer-2",
		RetailerRole:   "BUYER",
	}
	ownerClaims := auth.Claims{
		Subject:        "owner-1",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "owner-1",
		RetailerRole:   "OWNER",
	}
	foreignOwner := auth.Claims{
		Subject:        "owner-foreign",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-other",
		RetailerUserID: "owner-foreign",
		RetailerRole:   "OWNER",
	}

	// Create 2 single-order approvals (buyer-1, buyer-2) + 1 unified multi-supplier approval (buyer-1).
	var unifiedReqID, singleReqIDToReject string
	for i, spec := range []struct {
		key    string
		claims auth.Claims
		body   string
		uni    bool
	}{
		{
			key:    "k-single-1",
			claims: buyerClaims,
			body:   `{"payment_method":"CREDIT","line_items":[{"sku":"SKU-1","quantity":1,"unit_price_minor":12000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`,
		},
		{
			key:    "k-single-2",
			claims: otherSellerClaims,
			body:   `{"payment_method":"CREDIT","line_items":[{"sku":"SKU-2","quantity":1,"unit_price_minor":11000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`,
		},
		{
			key:    "k-unified-3",
			claims: buyerClaims,
			uni:    true,
			body:   `{"retailer_id":"ret-1","payment_method":"CREDIT","latitude":41.3,"longitude":69.2,"items":[{"supplier_id":"sup-a","sku_id":"SKU-A","quantity":1,"unit_price":6000},{"supplier_id":"sup-b","sku_id":"SKU-B","quantity":1,"unit_price":7000}]}`,
		},
	} {
		path := "/v1/order/create"
		handler := svc.HandleCreate
		if spec.uni {
			path = "/v1/checkout/unified"
			handler = svc.HandleUnifiedCheckout
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(spec.body))
		req = req.WithContext(auth.WithClaims(req.Context(), spec.claims))
		req.Header.Set("Idempotency-Key", spec.key)
		rr := httptest.NewRecorder()
		handler(rr, req)
		if rr.Code != http.StatusConflict {
			t.Fatalf("seed %d returned %d, want 409 (%s)", i, rr.Code, rr.Body.String())
		}
		var bodyMap map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &bodyMap); err != nil {
			t.Fatal(err)
		}
		rid, _ := bodyMap["request_id"].(string)
		if spec.key == "k-single-1" {
			singleReqIDToReject = rid
		}
		if spec.key == "k-unified-3" {
			unifiedReqID = rid
		}
	}

	// Keyset cursor pagination: limit=2 returns 2 items + next_cursor, then cursor page returns 1 item.
	reqP1 := httptest.NewRequest(http.MethodGet, "/v1/retailer/approvals?status=PENDING_OWNER&limit=2", nil)
	reqP1 = reqP1.WithContext(auth.WithClaims(reqP1.Context(), ownerClaims))
	rrP1 := httptest.NewRecorder()
	router.ServeHTTP(rrP1, reqP1)
	if rrP1.Code != http.StatusOK {
		t.Fatalf("page1 status = %d, want 200 (%s)", rrP1.Code, rrP1.Body.String())
	}
	var p1 ApprovalListResponse
	if err := json.Unmarshal(rrP1.Body.Bytes(), &p1); err != nil {
		t.Fatal(err)
	}
	if len(p1.Items) != 2 || p1.NextCursor == "" {
		t.Fatalf("page1 len=%d next_cursor=%q, want len=2 and non-empty cursor", len(p1.Items), p1.NextCursor)
	}
	if p1.Items[0].RequestID != unifiedReqID {
		t.Fatalf("expected newest request first (%s), got %s", unifiedReqID, p1.Items[0].RequestID)
	}

	reqP2 := httptest.NewRequest(http.MethodGet, "/v1/retailer/approvals?status=PENDING_OWNER&limit=2&cursor="+p1.NextCursor, nil)
	reqP2 = reqP2.WithContext(auth.WithClaims(reqP2.Context(), ownerClaims))
	rrP2 := httptest.NewRecorder()
	router.ServeHTTP(rrP2, reqP2)
	if rrP2.Code != http.StatusOK {
		t.Fatalf("page2 status = %d", rrP2.Code)
	}
	var p2 ApprovalListResponse
	if err := json.Unmarshal(rrP2.Body.Bytes(), &p2); err != nil {
		t.Fatal(err)
	}
	if len(p2.Items) != 1 || p2.NextCursor != "" {
		t.Fatalf("page2 len=%d next_cursor=%q, want len=1 empty cursor", len(p2.Items), p2.NextCursor)
	}

	// Non-manager only sees own requests; invalid cursor returns 400.
	reqBuyerList := httptest.NewRequest(http.MethodGet, "/v1/retailer/approvals", nil)
	reqBuyerList = reqBuyerList.WithContext(auth.WithClaims(reqBuyerList.Context(), buyerClaims))
	rrBuyerList := httptest.NewRecorder()
	router.ServeHTTP(rrBuyerList, reqBuyerList)
	var buyerList ApprovalListResponse
	if err := json.Unmarshal(rrBuyerList.Body.Bytes(), &buyerList); err != nil {
		t.Fatal(err)
	}
	if len(buyerList.Items) != 2 {
		t.Fatalf("buyer should only see own 2 requests, got %d", len(buyerList.Items))
	}

	reqBadCur := httptest.NewRequest(http.MethodGet, "/v1/retailer/approvals?cursor=not-valid-b64", nil)
	reqBadCur = reqBadCur.WithContext(auth.WithClaims(reqBadCur.Context(), ownerClaims))
	rrBadCur := httptest.NewRecorder()
	router.ServeHTTP(rrBadCur, reqBadCur)
	if rrBadCur.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status = %d, want 400", rrBadCur.Code)
	}

	// Non-manager cannot approve -> 403; foreign tenant -> 404.
	reqForb := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+unifiedReqID+"/approve", bytes.NewBufferString(`{}`))
	reqForb = reqForb.WithContext(auth.WithClaims(reqForb.Context(), buyerClaims))
	rrForb := httptest.NewRecorder()
	router.ServeHTTP(rrForb, reqForb)
	if rrForb.Code != http.StatusForbidden {
		t.Fatalf("buyer approve status = %d, want 403", rrForb.Code)
	}

	reqForeign := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+unifiedReqID+"/approve", bytes.NewBufferString(`{}`))
	reqForeign = reqForeign.WithContext(auth.WithClaims(reqForeign.Context(), foreignOwner))
	rrForeign := httptest.NewRecorder()
	router.ServeHTTP(rrForeign, reqForeign)
	if rrForeign.Code != http.StatusNotFound {
		t.Fatalf("foreign tenant approve status = %d, want 404", rrForeign.Code)
	}

	// Owner approves multi-supplier UNIFIED_CHECKOUT -> replays entire cart idempotently.
	reqApp1 := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+unifiedReqID+"/approve", bytes.NewBufferString(`{"reason":"approved by owner"}`))
	reqApp1 = reqApp1.WithContext(auth.WithClaims(reqApp1.Context(), ownerClaims))
	rrApp1 := httptest.NewRecorder()
	router.ServeHTTP(rrApp1, reqApp1)
	if rrApp1.Code != http.StatusOK {
		t.Fatalf("owner approve unified status = %d, want 200 (%s)", rrApp1.Code, rrApp1.Body.String())
	}
	var appResp1 ApprovalDecisionResponse
	if err := json.Unmarshal(rrApp1.Body.Bytes(), &appResp1); err != nil {
		t.Fatal(err)
	}
	if appResp1.Status != ApprovalStatusApproved || appResp1.UnifiedCheckout == nil || len(appResp1.UnifiedCheckout.SupplierOrders) != 2 {
		t.Fatalf("unexpected unified approval response: %+v", appResp1)
	}
	if repo.countCreated() != 2 {
		t.Fatalf("expected 2 child orders created on unified approval, got %d", repo.countCreated())
	}

	// Re-approving same request returns same replay result idempotently with 0 additional orders.
	reqApp2 := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+unifiedReqID+"/approve", bytes.NewBufferString(`{}`))
	reqApp2 = reqApp2.WithContext(auth.WithClaims(reqApp2.Context(), ownerClaims))
	rrApp2 := httptest.NewRecorder()
	router.ServeHTTP(rrApp2, reqApp2)
	if rrApp2.Code != http.StatusOK {
		t.Fatalf("re-approve unified status = %d, want 200 (%s)", rrApp2.Code, rrApp2.Body.String())
	}
	var appResp2 ApprovalDecisionResponse
	if err := json.Unmarshal(rrApp2.Body.Bytes(), &appResp2); err != nil {
		t.Fatal(err)
	}
	if appResp2.ParentOrderID != appResp1.ParentOrderID || appResp2.OrderID != appResp1.OrderID || repo.countCreated() != 2 {
		t.Fatalf("idempotent re-approve created duplicate orders: resp2=%+v createCalls=%d", appResp2, repo.countCreated())
	}

	// Rejecting pending request sets REJECTED with reason; subsequent approve fails with 409.
	reqRej := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+singleReqIDToReject+"/reject", bytes.NewBufferString(`{"reason":"over weekly budget"}`))
	reqRej = reqRej.WithContext(auth.WithClaims(reqRej.Context(), ownerClaims))
	rrRej := httptest.NewRecorder()
	router.ServeHTTP(rrRej, reqRej)
	if rrRej.Code != http.StatusOK {
		t.Fatalf("reject status = %d, want 200 (%s)", rrRej.Code, rrRej.Body.String())
	}
	var rejResp ApprovalDecisionResponse
	if err := json.Unmarshal(rrRej.Body.Bytes(), &rejResp); err != nil {
		t.Fatal(err)
	}
	if rejResp.Status != ApprovalStatusRejected || rejResp.DecisionReason != "over weekly budget" {
		t.Fatalf("unexpected reject response: %+v", rejResp)
	}

	reqAppAfterRej := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+singleReqIDToReject+"/approve", bytes.NewBufferString(`{}`))
	reqAppAfterRej = reqAppAfterRej.WithContext(auth.WithClaims(reqAppAfterRej.Context(), ownerClaims))
	rrAppAfterRej := httptest.NewRecorder()
	router.ServeHTTP(rrAppAfterRej, reqAppAfterRej)
	if rrAppAfterRej.Code != http.StatusConflict {
		t.Fatalf("approve after reject status = %d, want 409", rrAppAfterRej.Code)
	}
}

func TestCreateLockOrder_StrictOrderAndSagaCompensationRollback(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "true")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	fixedNow := time.Date(2026, 11, 13, 11, 0, 0, 0, time.UTC)
	repo := newMandateMemRepo()
	reader := newMapMandateReader()
	reader.Set("ret-1", "buyer-1", MandateLookupResult{
		Role:  "BUYER",
		Found: true,
		Mandate: SellerMandate{
			CanUseCredit:        true,
			CanPayCash:          true,
			DailyCreditCapMinor: 50000,
			DailyCashCapMinor:   50000,
			MaxSingleOrderMinor: 50000,
		},
	})
	svc := newMandateTestService(repo, reader, func() time.Time { return fixedNow })

	buyerCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:        "buyer-1",
		Role:           auth.RoleRetailer,
		RetailerOrgID:  "ret-1",
		RetailerUserID: "buyer-1",
		RetailerRole:   "BUYER",
	})

	// Verify strict lock sequence: RetailerSellerDailyUsage -> CreditProfile -> Orders.
	var stages []string
	tracedCtx := WithLockOrderTracer(buyerCtx, func(stage string) {
		stages = append(stages, stage)
	})
	_, err := svc.Create(tracedCtx, "ret-1", CreateRequest{
		PaymentMethod: PaymentMethodCredit,
		H3Cell:        "872830828ffffff",
		Lat:           41.3,
		Lng:           69.2,
		LineItems:     []LineItem{{SKU: "SKU-LOCK", Quantity: 1, UnitPrice: 3000}},
	})
	if err != nil {
		t.Fatalf("Create credit failed: %v", err)
	}
	wantStages := []string{LockStageUsage, LockStageCreditProfile, LockStageOrders}
	if !reflect.DeepEqual(stages, wantStages) {
		t.Fatalf("lock order stages = %v, want %v", stages, wantStages)
	}

	// Now fail second supplier in multi-supplier checkout so saga compensation rolls back leg 1 usage.
	repo.failSupplierID = "sup-b"
	_, uniErr := svc.UnifiedCheckout(buyerCtx, "ret-1", UnifiedCheckoutRequest{
		RetailerID:    "ret-1",
		PaymentMethod: PaymentMethodCredit,
		Latitude:      41.3,
		Longitude:     69.2,
		Items: []UnifiedCheckoutLineItem{
			{SupplierID: "sup-a", SkuID: "SKU-A", Quantity: 1, UnitPrice: 4000},
			{SupplierID: "sup-b", SkuID: "SKU-B", Quantity: 1, UnitPrice: 7000},
		},
	})
	if uniErr == nil {
		t.Fatal("expected UnifiedCheckout to fail when sup-b fails")
	}

	// Usage should remain at 3000 (from the first Create), NOT 3000+4000=7000.
	usage, uErr := svc.GetSellerDailyUsage(context.Background(), "ret-1", "buyer-1", civil.DateOf(fixedNow))
	if uErr != nil {
		t.Fatalf("GetSellerDailyUsage err: %v", uErr)
	}
	if usage.SpentCreditMinor != 3000 || usage.OrderCount != 1 {
		t.Fatalf("usage after saga rollback = %+v, want spent_credit_minor=3000 order_count=1", usage)
	}

	// Verify CompensateSaga repeat call does not double-decrement.
	repo.mu.Lock()
	var compensatedOrders []Order
	for _, o := range repo.orders {
		if o.SupplierID == "sup-a" {
			compensatedOrders = append(compensatedOrders, o)
		}
	}
	repo.mu.Unlock()
	if err := svc.CompensateSaga(buyerCtx, "parent-repeat", compensatedOrders, "repeat"); err != nil {
		t.Fatalf("repeat CompensateSaga err: %v", err)
	}
	usageAfterRepeat, uErr2 := svc.GetSellerDailyUsage(context.Background(), "ret-1", "buyer-1", civil.DateOf(fixedNow))
	if uErr2 != nil {
		t.Fatalf("GetSellerDailyUsage err2: %v", uErr2)
	}
	if usageAfterRepeat.SpentCreditMinor != 3000 || usageAfterRepeat.OrderCount != 1 {
		t.Fatalf("repeat CompensateSaga double-decremented usage: %+v", usageAfterRepeat)
	}
}
