package order

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestUnifiedCheckout_CreatesSingleSupplierOrder(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "false")
	t.Setenv("PEGASUSX_ENV", "test")

	repo := &testRepo{}
	warehouse := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		Warehouse:    warehouse,
		SupplierID:   "sup-1",
		SupplierName: "Test Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	// Scaffold path (no Spanner): client unit prices still accepted for unit tests.
	resp, err := svc.UnifiedCheckout(context.Background(), "ret-1", UnifiedCheckoutRequest{
		Latitude:  41.31,
		Longitude: 69.24,
		Items: []UnifiedCheckoutLineItem{{
			SkuID:     "sku-1",
			Quantity:  2,
			UnitPrice: 15000,
		}},
	})
	if err != nil {
		t.Fatalf("UnifiedCheckout() err = %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.Status)
	}
	if resp.InvoiceID == "" {
		t.Fatal("invoice_id empty")
	}
	if resp.Total != 30000 {
		t.Fatalf("total = %d, want 30000", resp.Total)
	}
	if len(resp.SupplierOrders) != 1 {
		t.Fatalf("supplier_orders len = %d, want 1", len(resp.SupplierOrders))
	}
	so := resp.SupplierOrders[0]
	if so.OrderID == "" || so.SupplierID != "sup-1" || so.Total != 30000 || so.ItemCount != 1 {
		t.Fatalf("unexpected supplier order: %+v", so)
	}
	if resp.Currency != "UZS" || resp.MarketCode != "UZ" {
		t.Fatalf("pack stamp currency=%s market=%s", resp.Currency, resp.MarketCode)
	}
	if repo.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", repo.createCalls)
	}
}

func TestUnifiedCheckout_MultiSupplierSplit(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "true")

	repo := &testRepo{}
	warehouse := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		Warehouse:    warehouse,
		SupplierID:   "sup-seed",
		SupplierName: "Test Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	resp, err := svc.UnifiedCheckout(context.Background(), "ret-1", UnifiedCheckoutRequest{
		Latitude:  41.31,
		Longitude: 69.24,
		Items: []UnifiedCheckoutLineItem{
			{SkuID: "sku-a", Quantity: 1, UnitPrice: 1000, SupplierID: "sup-a"},
			{SkuID: "sku-b", Quantity: 2, UnitPrice: 2000, SupplierID: "sup-b"},
		},
	})
	if err != nil {
		t.Fatalf("UnifiedCheckout() err = %v", err)
	}
	if resp.ParentOrderID == "" {
		t.Fatal("parent_order_id empty")
	}
	if len(resp.SupplierOrders) != 2 {
		t.Fatalf("supplier_orders len = %d, want 2", len(resp.SupplierOrders))
	}
	if resp.Total != 5000 {
		t.Fatalf("total = %d, want 5000", resp.Total)
	}
	if repo.createCalls != 2 {
		t.Fatalf("createCalls = %d, want 2", repo.createCalls)
	}
	if repo.created.ParentOrderID != resp.ParentOrderID {
		t.Fatalf("child ParentOrderID = %q, want %q", repo.created.ParentOrderID, resp.ParentOrderID)
	}
	seen := map[string]bool{}
	for _, so := range resp.SupplierOrders {
		seen[so.SupplierID] = true
	}
	if !seen["sup-a"] || !seen["sup-b"] {
		t.Fatalf("supplier_orders missing tenants: %+v", resp.SupplierOrders)
	}
}

func TestRollupParentStatus(t *testing.T) {
	t.Parallel()
	if got := rollupParentStatus(nil); got != parentStatusPending {
		t.Fatalf("empty = %q", got)
	}
	if got := rollupParentStatus([]ParentOrderChild{{Status: string(StatusPending)}, {Status: string(StatusPending)}}); got != parentStatusPending {
		t.Fatalf("pending = %q", got)
	}
	if got := rollupParentStatus([]ParentOrderChild{{Status: string(StatusCompleted)}, {Status: string(StatusCompleted)}}); got != parentStatusComplete {
		t.Fatalf("complete = %q", got)
	}
	if got := rollupParentStatus([]ParentOrderChild{{Status: string(StatusCancelled)}, {Status: string(StatusCancelled)}}); got != parentStatusCancelled {
		t.Fatalf("cancelled = %q", got)
	}
	if got := rollupParentStatus([]ParentOrderChild{{Status: string(StatusCompleted)}, {Status: string(StatusPending)}}); got != parentStatusPartial {
		t.Fatalf("partial = %q", got)
	}
}

func TestAuthoritativeCheckoutLines_RejectsBadItems(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{SupplierID: "sup-1", Currency: "UZS", Log: slog.Default()})
	_, err := svc.authoritativeCheckoutLines(context.Background(), "ret-1", []UnifiedCheckoutLineItem{{
		SkuID: "", Quantity: 1, UnitPrice: 100,
	}})
	if err == nil {
		t.Fatal("expected error for empty sku")
	}
}

func TestCheckoutSnapshot_ForbidsForeignRetailer(t *testing.T) {
	t.Parallel()

	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			RetailerID: "ret-owner",
			TotalMinor: 5000,
			Currency:   "UZS",
		},
	}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Currency: "UZS"})

	_, _, err := svc.CheckoutSnapshot(context.Background(), "ord-1", "ret-other")
	if err == nil || err != ErrOrderForbidden {
		t.Fatalf("CheckoutSnapshot() err = %v, want %v", err, ErrOrderForbidden)
	}
}

func TestAssertChildSuppliersSameMarket_PlannedChild(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Cleanup(func() { auth.SetMarketProfileLookup(nil) })
	auth.SetMarketProfileLookup(func(supplierID string) (auth.MarketProfile, bool) {
		switch supplierID {
		case "sup-uz":
			return auth.MarketProfile{MarketCode: "UZ", HomeCell: "cell-uz"}, true
		case "sup-kz":
			return auth.MarketProfile{MarketCode: "KZ", HomeCell: "cell-kz"}, true
		default:
			return auth.MarketProfile{}, false
		}
	})
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	err := assertChildSuppliersSameMarket(context.Background(), pack, []checkoutLineGroup{
		{SupplierID: "sup-uz"},
		{SupplierID: "sup-kz"},
	})
	if !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("mixed UZ+KZ must fail before parent insert: %v", err)
	}
}

func TestAssertChildSuppliersSameMarket_SamePack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	pack, ok := auth.ResolveShippedMarketPack("UZ")
	if !ok {
		t.Fatal("uz pack")
	}
	if err := assertChildSuppliersSameMarket(context.Background(), pack, []checkoutLineGroup{
		{SupplierID: "sup-a"},
		{SupplierID: "sup-b"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUnifiedCheckout_PlannedPackFailsClosed(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "false")
	svc := NewService(ServiceConfig{
		Repo:       &testRepo{},
		Warehouse:  &testWarehouseResolver{warehouseID: "wh-1"},
		SupplierID: "sup-1",
		Currency:   "UZS",
		Log:        slog.Default(),
	})
	ctx := auth.WithClaims(context.Background(), auth.Claims{MarketCode: "EU", Subject: "ret-1"})
	_, err := svc.UnifiedCheckout(ctx, "ret-1", UnifiedCheckoutRequest{
		Latitude: 41.31, Longitude: 69.24,
		Items: []UnifiedCheckoutLineItem{{SkuID: "sku-1", Quantity: 1, UnitPrice: 1000}},
	})
	if !errors.Is(err, auth.ErrMarketPackNotShipped) {
		t.Fatalf("err=%v", err)
	}
}

func TestUnifiedCheckout_CurrencyMismatch(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "false")
	svc := NewService(ServiceConfig{
		Repo:       &testRepo{},
		Warehouse:  &testWarehouseResolver{warehouseID: "wh-1"},
		SupplierID: "sup-1",
		Currency:   "UZS",
		Log:        slog.Default(),
	})
	_, err := svc.UnifiedCheckout(context.Background(), "ret-1", UnifiedCheckoutRequest{
		Latitude: 41.31, Longitude: 69.24, Currency: "EUR",
		Items: []UnifiedCheckoutLineItem{{SkuID: "sku-1", Quantity: 1, UnitPrice: 1000}},
	})
	if !errors.Is(err, auth.ErrPackCurrencyMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleUnifiedCheckout_StrictIdempotency(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "false")
	t.Setenv("PEGASUSX_ENV", "test")
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")

	repo := &testRepo{}
	warehouse := &testWarehouseResolver{warehouseID: "wh-1"}
	idemStore := idempotency.NewInMemoryStore()
	svc := NewService(ServiceConfig{
		Repo:         repo,
		Warehouse:    warehouse,
		SupplierID:   "sup-1",
		SupplierName: "Test Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
		Idem:         idemStore,
	})

	cartBody := `{"retailer_id":"ret-1","payment_gateway":"CASH","latitude":41.31,"longitude":69.24,"items":[{"sku_id":"sku-1","quantity":2,"unit_price":15000}]}`

	// 1. Missing Idempotency-Key must fail with 400 Bad Request
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(cartBody))
		ctx := auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "ret-1",
			MarketCode: "UZ",
			Role:       auth.RoleRetailer,
		})
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		svc.HandleUnifiedCheckout(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing idempotency key, got %d, body: %s", rec.Code, rec.Body.String())
		}
		var errResp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "idempotency_key_required" {
			t.Fatalf("expected error 'idempotency_key_required', got: %v", errResp)
		}
		if repo.createCalls != 0 {
			t.Fatalf("createCalls = %d, want 0", repo.createCalls)
		}
	}

	// 2. First checkout with Session 1 -> succeeds with 201 Created
	var firstOrderID string
	var firstInvoiceID string
	keySess1 := "retailer-checkout:CASH:sess-1:sku-1:2:15000"
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(cartBody))
		ctx := auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "ret-1",
			MarketCode: "UZ",
			Role:       auth.RoleRetailer,
		})
		req = req.WithContext(ctx)
		req.Header.Set("Idempotency-Key", keySess1)
		rec := httptest.NewRecorder()

		svc.HandleUnifiedCheckout(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for first checkout, got %d, body: %s", rec.Code, rec.Body.String())
		}
		var resp UnifiedCheckoutResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(resp.SupplierOrders) == 0 {
			t.Fatalf("expected supplier orders, got none")
		}
		firstOrderID = resp.SupplierOrders[0].OrderID
		firstInvoiceID = resp.InvoiceID
		if firstOrderID == "" || firstInvoiceID == "" {
			t.Fatalf("expected order and invoice ids, got order=%q invoice=%q", firstOrderID, firstInvoiceID)
		}
		if repo.createCalls != 1 {
			t.Fatalf("createCalls = %d, want 1", repo.createCalls)
		}
	}

	// 3. Replay with identical body and same Session 1 key -> returns 201 Replay without calling CreateOrder again
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(cartBody))
		ctx := auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "ret-1",
			MarketCode: "UZ",
			Role:       auth.RoleRetailer,
		})
		req = req.WithContext(ctx)
		req.Header.Set("Idempotency-Key", keySess1)
		rec := httptest.NewRecorder()

		svc.HandleUnifiedCheckout(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created on replay, got %d, body: %s", rec.Code, rec.Body.String())
		}
		var resp UnifiedCheckoutResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.SupplierOrders[0].OrderID != firstOrderID || resp.InvoiceID != firstInvoiceID {
			t.Fatalf("expected replayed IDs to match (%s, %s), got (%s, %s)",
				firstOrderID, firstInvoiceID, resp.SupplierOrders[0].OrderID, resp.InvoiceID)
		}
		// createCalls MUST STILL BE 1 (deduplicated)
		if repo.createCalls != 1 {
			t.Fatalf("createCalls on replay = %d, want 1", repo.createCalls)
		}
	}

	// 4. Distinct checkout session with identical items -> creates a fresh, distinct order!
	keySess2 := "retailer-checkout:CASH:sess-2:sku-1:2:15000"
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(cartBody))
		ctx := auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "ret-1",
			MarketCode: "UZ",
			Role:       auth.RoleRetailer,
		})
		req = req.WithContext(ctx)
		req.Header.Set("Idempotency-Key", keySess2)
		rec := httptest.NewRecorder()

		svc.HandleUnifiedCheckout(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for new session checkout, got %d, body: %s", rec.Code, rec.Body.String())
		}
		var resp UnifiedCheckoutResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		secondOrderID := resp.SupplierOrders[0].OrderID
		if secondOrderID == firstOrderID {
			t.Fatalf("expected new distinct order ID for new session, but got duplicate %s", secondOrderID)
		}
		if repo.createCalls != 2 {
			t.Fatalf("createCalls = %d, want 2", repo.createCalls)
		}
	}

	// 5. Payload mismatch with same key returns 409 Conflict
	{
		differentBody := `{"retailer_id":"ret-1","payment_gateway":"CASH","latitude":41.31,"longitude":69.24,"items":[{"sku_id":"sku-1","quantity":5,"unit_price":15000}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1/checkout/unified", strings.NewReader(differentBody))
		ctx := auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "ret-1",
			MarketCode: "UZ",
			Role:       auth.RoleRetailer,
		})
		req = req.WithContext(ctx)
		req.Header.Set("Idempotency-Key", keySess2)
		rec := httptest.NewRecorder()

		svc.HandleUnifiedCheckout(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict on payload mismatch, got %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}

