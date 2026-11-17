package orderroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type routeMandateRepo struct {
	order.Repository
	mu      sync.Mutex
	orders  map[string]order.Order
	creates int
}

func (r *routeMandateRepo) CreateOrder(_ context.Context, o *order.Order, _ func(outbox.TxnBuffer) error, _ order.StockReservationOpts) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.orders == nil {
		r.orders = make(map[string]order.Order)
	}
	r.creates++
	r.orders[o.OrderID] = *o
	return nil
}

func (r *routeMandateRepo) CreateOrderWithBackorder(ctx context.Context, o *order.Order, _ *order.Order, _ func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, opts order.StockReservationOpts) error {
	return r.CreateOrder(ctx, o, emit, opts)
}

func issueRetailerStaffToken(t *testing.T, orgID, userID, staffRole string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:        userID,
		Role:           auth.RoleRetailer,
		RetailerOrgID:  orgID,
		RetailerUserID: userID,
		RetailerRole:   staffRole,
	}, auth.IssueOptions{Secret: "order-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

type routeMandateReader struct {
	mandate order.MandateLookupResult
}

func (m routeMandateReader) ResolveSellerMandate(_ context.Context, _, _, _ string) (order.MandateLookupResult, error) {
	return m.mandate, nil
}

func TestOrderRoutes_MandatesAndOwnerApprovalEndpoints(t *testing.T) {
	t.Parallel()
	var seq atomic.Int64
	repo := &routeMandateRepo{}
	svc := order.NewService(order.ServiceConfig{
		Repo:       repo,
		Warehouse:  routeWarehouseResolver{},
		SupplierID: "sup-1",
		Currency:   "UZS",
		MandateReader: routeMandateReader{
			mandate: order.MandateLookupResult{
				Role:  "CASHIER",
				Found: true,
				Mandate: order.SellerMandate{
					CanUseCredit:        false,
					CanPayCash:          true,
					DailyCashCapMinor:   50000,
					MaxSingleOrderMinor: 50000,
				},
			},
		},
		Log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		NewID: func() string { return fmt.Sprintf("ord-route-%d", seq.Add(1)) },
	})
	router := orderRouter(svc)

	cashierTok := issueRetailerStaffToken(t, "ret-1", "cashier-1", "CASHIER")
	ownerTok := issueRetailerStaffToken(t, "ret-1", "owner-1", "OWNER")

	// Unauthenticated /v1/retailer/approvals returns 401.
	rrUnauth := httptest.NewRecorder()
	router.ServeHTTP(rrUnauth, httptest.NewRequest(http.MethodGet, "/v1/retailer/approvals", nil))
	if rrUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth approvals status = %d, want 401", rrUnauth.Code)
	}

	// Cashier submits credit order -> 409 Conflict with CASH + CARD fallback.
	req409 := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewBufferString(`{
		"payment_method": "CREDIT",
		"h3_cell": "872830828ffffff",
		"lat": 41.3,
		"lng": 69.2,
		"line_items": [{"sku":"SKU-1","quantity":1,"unit_price_minor":8000}]
	}`))
	req409.Header.Set("Authorization", "Bearer "+cashierTok)
	req409.Header.Set("Idempotency-Key", "route-client-order-1")
	rr409 := httptest.NewRecorder()
	router.ServeHTTP(rr409, req409)
	if rr409.Code != http.StatusConflict {
		t.Fatalf("POST /v1/order/create status = %d, want 409 (%s)", rr409.Code, rr409.Body.String())
	}
	var conflict map[string]any
	if err := json.Unmarshal(rr409.Body.Bytes(), &conflict); err != nil {
		t.Fatal(err)
	}
	requestID, _ := conflict["request_id"].(string)
	if requestID == "" || conflict["code"] != "owner_approval_required" {
		t.Fatalf("unexpected 409 body: %+v", conflict)
	}

	// GET /v1/retailer/approvals lists the pending request for OWNER.
	reqList := httptest.NewRequest(http.MethodGet, "/v1/retailer/approvals?status=PENDING_OWNER", nil)
	reqList.Header.Set("Authorization", "Bearer "+ownerTok)
	rrList := httptest.NewRecorder()
	router.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("GET /v1/retailer/approvals status = %d, want 200 (%s)", rrList.Code, rrList.Body.String())
	}
	var listResp order.ApprovalListResponse
	if err := json.Unmarshal(rrList.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].RequestID != requestID {
		t.Fatalf("unexpected approval queue list: %+v", listResp)
	}

	// Cashier cannot approve -> 403.
	reqCashierApp := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+requestID+"/approve", bytes.NewBufferString(`{}`))
	reqCashierApp.Header.Set("Authorization", "Bearer "+cashierTok)
	rrCashierApp := httptest.NewRecorder()
	router.ServeHTTP(rrCashierApp, reqCashierApp)
	if rrCashierApp.Code != http.StatusForbidden {
		t.Fatalf("cashier approve status = %d, want 403", rrCashierApp.Code)
	}

	// Owner approves -> 200 OK & replays whole cart.
	reqApp := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+requestID+"/approve", bytes.NewBufferString(`{"reason":"ok"}`))
	reqApp.Header.Set("Authorization", "Bearer "+ownerTok)
	rrApp := httptest.NewRecorder()
	router.ServeHTTP(rrApp, reqApp)
	if rrApp.Code != http.StatusOK {
		t.Fatalf("POST approve status = %d, want 200 (%s)", rrApp.Code, rrApp.Body.String())
	}
	if repo.creates != 1 {
		t.Fatalf("expected 1 replayed order created, got %d", repo.creates)
	}

	// Cashier creates a second credit request on /v1/order/create and owner rejects it cleanly.
	reqCreate := httptest.NewRequest(http.MethodPost, "/v1/order/create", bytes.NewBufferString(`{"payment_method":"CREDIT","line_items":[{"sku":"SKU-1","quantity":1,"unit_price_minor":4000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`))
	reqCreate.Header.Set("Authorization", "Bearer "+cashierTok)
	reqCreate.Header.Set("Idempotency-Key", "route-client-order-2")
	rrCreate := httptest.NewRecorder()
	router.ServeHTTP(rrCreate, reqCreate)
	if rrCreate.Code != http.StatusConflict {
		t.Fatalf("POST /v1/order/create status = %d, want 409", rrCreate.Code)
	}
	var c2 map[string]any
	if err := json.Unmarshal(rrCreate.Body.Bytes(), &c2); err != nil {
		t.Fatal(err)
	}
	rejID, _ := c2["request_id"].(string)
	reqRej := httptest.NewRequest(http.MethodPost, "/v1/retailer/approvals/"+rejID+"/reject", bytes.NewBufferString(`{"reason":"use cash instead"}`))
	reqRej.Header.Set("Authorization", "Bearer "+ownerTok)
	rrRej := httptest.NewRecorder()
	router.ServeHTTP(rrRej, reqRej)
	if rrRej.Code != http.StatusOK {
		t.Fatalf("POST reject status = %d, want 200 (%s)", rrRej.Code, rrRej.Body.String())
	}
}
