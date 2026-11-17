package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func retailerTestReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Idempotency-Key", "idem-"+target+"-"+body)
	return req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:       auth.RoleRetailer,
		Subject:    "ret-1",
		SupplierID: "sup-1",
	}))
}

func TestPackMultiples_FailClosedOnMetadataReadError(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	dbErr := errors.New("spanner timeout reading products")
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:               "ord-pre-1",
			SupplierID:            "sup-1",
			RetailerID:            "ret-1",
			WarehouseID:           "wh-1",
			Status:                StatusScheduled,
			Source:                OrderSourceManualPreorder,
			ConfirmationStatus:    ConfirmationStatusPending,
			RequestedDeliveryDate: func() *time.Time { v := now.Add(5 * 24 * time.Hour); return &v }(),
			Timezone:              "UTC",
			LineItems:             []LineItem{{SKU: "SKU-1", Quantity: 12, UnitPrice: 1000}},
		},
	}
	svc := newTestServiceWithResolver(repo, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	svc.SetPackRuleReader(func(context.Context, []string) (map[string]PackRule, error) {
		return nil, dbErr
	})

	// 1. Direct Create must fail closed and never create an order.
	_, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "SKU-1", Quantity: 1, UnitPrice: 1000}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if !errors.Is(err, ErrPackMetadataUnavailable) {
		t.Fatalf("expected ErrPackMetadataUnavailable, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected 0 create calls on pack metadata DB error, got %d", repo.createCalls)
	}

	// 2. HTTP HandleCreate must return 503 Service Unavailable.
	rrCreate := httptest.NewRecorder()
	svc.HandleCreate(rrCreate, retailerTestReq(http.MethodPost, "/v1/order/create",
		`{"line_items":[{"sku":"SKU-1","quantity":1,"unit_price_minor":1000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`))
	if rrCreate.Code != http.StatusServiceUnavailable {
		t.Fatalf("HandleCreate status = %d, want 503 (%s)", rrCreate.Code, rrCreate.Body.String())
	}
	var createBody map[string]any
	if err := json.Unmarshal(rrCreate.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode HandleCreate body: %v", err)
	}
	if createBody["code"] != "PACK_METADATA_UNAVAILABLE" {
		t.Fatalf("HandleCreate code = %v, want PACK_METADATA_UNAVAILABLE", createBody["code"])
	}

	// 3. HTTP HandleCheckoutPreview must return 503 Service Unavailable.
	rrPreview := httptest.NewRecorder()
	svc.HandleCheckoutPreview(rrPreview, retailerTestReq(http.MethodPost, "/v1/order/checkout-preview",
		`{"retailer_id":"ret-1","latitude":41.3,"longitude":69.2,"items":[{"sku_id":"SKU-1","quantity":1,"unit_price":1000}]}`))
	if rrPreview.Code != http.StatusServiceUnavailable {
		t.Fatalf("HandleCheckoutPreview status = %d, want 503 (%s)", rrPreview.Code, rrPreview.Body.String())
	}

	// 4. EditPreorder must fail closed as well.
	_, err = svc.EditPreorder(context.Background(), "ret-1", EditPreorderRequest{
		OrderID:               "ord-pre-1",
		LineItems:             []LineItem{{SKU: "SKU-1", Quantity: 1, UnitPrice: 1000}},
		RequestedDeliveryDate: now.Add(6 * 24 * time.Hour).Format(time.RFC3339),
	})
	if !errors.Is(err, ErrPackMetadataUnavailable) {
		t.Fatalf("EditPreorder expected ErrPackMetadataUnavailable, got %v", err)
	}
}

func TestPackMultiples_HandleCreateAndPreviewReturnStructuredSuggestions(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	svc := newTestServiceWithResolver(repo, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	svc.SetPackRuleReader(func(context.Context, []string) (map[string]PackRule, error) {
		return map[string]PackRule{
			"SKU-CASE-12": {SaleUnit: "CASE", UnitsPerPack: 12},
			"SKU-BLOCK-6": {SaleUnit: "BLOCK", UnitsPerPack: 6},
		}, nil
	})

	rr := httptest.NewRecorder()
	svc.HandleCreate(rr, retailerTestReq(http.MethodPost, "/v1/order/create",
		`{"line_items":[{"sku":"SKU-CASE-12","quantity":13,"unit_price_minor":2000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("HandleCreate status = %d, want 422 (%s)", rr.Code, rr.Body.String())
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected 0 create calls when pack validation fails, got %d", repo.createCalls)
	}

	var resp struct {
		Code        string               `json:"code"`
		LineErrors  map[string]string    `json:"line_errors"`
		Suggestions []PackLineSuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode 422 response: %v", err)
	}
	if resp.Code != "PACK_MULTIPLE_REQUIRED" {
		t.Fatalf("code = %q, want PACK_MULTIPLE_REQUIRED", resp.Code)
	}
	if resp.LineErrors["SKU-CASE-12"] == "" {
		t.Fatalf("expected line_errors entry for SKU-CASE-12, got %+v", resp.LineErrors)
	}
	if len(resp.Suggestions) != 1 {
		t.Fatalf("suggestions len = %d, want 1", len(resp.Suggestions))
	}
	sugg := resp.Suggestions[0]
	if sugg.SKU != "SKU-CASE-12" || sugg.RequestedQty != 13 || sugg.PackSize != 12 ||
		sugg.SuggestedFloor != 12 || sugg.SuggestedCeil != 24 || sugg.SuggestedNearest != 12 || sugg.SaleUnit != "CASE" {
		t.Fatalf("unexpected pack suggestion: %+v", sugg)
	}

	// Verify CheckoutPreview also returns structured suggestions.
	rrPrev := httptest.NewRecorder()
	svc.HandleCheckoutPreview(rrPrev, retailerTestReq(http.MethodPost, "/v1/order/checkout-preview",
		`{"retailer_id":"ret-1","latitude":41.3,"longitude":69.2,"items":[{"sku_id":"SKU-BLOCK-6","quantity":7,"unit_price":1000}]}`))
	if rrPrev.Code != http.StatusOK {
		t.Fatalf("HandleCheckoutPreview status = %d, want 200 (%s)", rrPrev.Code, rrPrev.Body.String())
	}
	var prev CheckoutPreviewResponse
	if err := json.Unmarshal(rrPrev.Body.Bytes(), &prev); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	if prev.OK || prev.Code != "PACK_MULTIPLE_REQUIRED" || len(prev.Suggestions) != 1 {
		t.Fatalf("unexpected preview pack response: %+v", prev)
	}
	if prev.Suggestions[0].PackSize != 6 || prev.Suggestions[0].SuggestedCeil != 12 || prev.Suggestions[0].SaleUnit != "BLOCK" {
		t.Fatalf("unexpected preview suggestion: %+v", prev.Suggestions[0])
	}
}

func TestServiceCreate_SubstitutionSameBrandExactPriceGapAndDisclosure(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	svc := newTestServiceWithResolver(repo, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []LineItem) (ShortageContext, error) {
		return ShortageContext{
			AvailableBySKU: map[string]int64{
				"SKU-ORIG":        1,  // short for qty 5
				"SKU-SUB":         20, // same brand & supplier
				"SKU-OTHER-BRAND": 50, // cheaper but different brand -> must be rejected
			},
			CandidatesBySKU: map[string][]SubstituteCandidate{
				"SKU-ORIG": {
					{
						SKU:          "SKU-OTHER-BRAND",
						SupplierID:   "sup-1",
						BrandID:      "BRAND-OTHER",
						Name:         "Wrong Brand 1L",
						AvailableQty: 50,
						PriceMinor:   8000,
					},
					{
						SKU:          "SKU-SUB",
						SupplierID:   "sup-1",
						BrandID:      "BRAND-COLA",
						Name:         "Cola 1.5L",
						AvailableQty: 20,
						PriceMinor:   11500,
					},
				},
			},
			EnforceAllowlist: true,
		}, nil
	})

	body := `{
		"line_items": [{
			"sku": "SKU-ORIG",
			"name": "Cola 1L",
			"brand_id": "BRAND-COLA",
			"quantity": 5,
			"unit_price_minor": 10000,
			"allow_substitutes": true,
			"allowed_substitute_skus": ["SKU-SUB", "SKU-OTHER-BRAND"]
		}],
		"h3_cell": "872830828ffffff",
		"lat": 41.3,
		"lng": 69.2,
		"auto_confirm_opt_in": true
	}`
	rr := httptest.NewRecorder()
	svc.HandleCreate(rr, retailerTestReq(http.MethodPost, "/v1/order/create", body))
	if rr.Code != http.StatusCreated {
		t.Fatalf("HandleCreate status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}

	var resp CreateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode create resp: %v", err)
	}
	if !resp.AutoConfirmOptIn {
		t.Fatalf("expected AutoConfirmOptIn=true on response")
	}
	if resp.TotalMinor != 57500 {
		t.Fatalf("TotalMinor = %d, want 57500 (5*11500)", resp.TotalMinor)
	}
	if len(resp.Substitutions) != 1 {
		t.Fatalf("expected 1 substitution disclosure, got %+v", resp.Substitutions)
	}
	sub := resp.Substitutions[0]
	if sub.SubstituteSKU != "SKU-SUB" || sub.PriceGapMinor != 7500 || sub.Reason == "" {
		t.Fatalf("unexpected substitution disclosure: %+v", sub)
	}
	if len(resp.LineItems) != 1 || resp.LineItems[0].SKU != "SKU-SUB" || resp.LineItems[0].OriginalSKU != "SKU-ORIG" || resp.LineItems[0].PriceGapMinor != 7500 {
		t.Fatalf("unexpected substituted line item: %+v", resp.LineItems)
	}
}

func TestServiceCreate_MustHaveUnfilledHoldsOrderAndBlocksAssignment(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	svc := newTestServiceWithResolver(repo, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []LineItem) (ShortageContext, error) {
		return ShortageContext{
			AvailableBySKU: map[string]int64{
				"SKU-ANCHOR": 0, // unfilled MustHave
				"SKU-SIDE":   10,
			},
		}, nil
	})

	resp, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{
			{SKU: "SKU-ANCHOR", BrandID: "B1", Quantity: 2, UnitPrice: 5000, MustHave: true, AllowSubstitutes: true},
			{SKU: "SKU-SIDE", BrandID: "B2", Quantity: 1, UnitPrice: 3000},
		},
		H3Cell: "872830828ffffff",
		Lat:    41.3,
		Lng:    69.2,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Status != StatusDelayed {
		t.Fatalf("Status = %s, want DELAYED (HOLD_ORDER)", resp.Status)
	}
	if resp.ConfirmationStatus != ConfirmationStatusPendingWarehouse {
		t.Fatalf("ConfirmationStatus = %s, want PENDING_WAREHOUSE", resp.ConfirmationStatus)
	}
	if resp.HoldReason == "" {
		t.Fatalf("expected non-empty HoldReason on held order")
	}

	// Held order must never be assigned onto a truck/driver manifest.
	repo.found = true
	repo.order = repo.created
	_, err = svc.AssignOrder(context.Background(), auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-1",
	}, resp.OrderID, AssignOrderRequest{
		DriverID: "drv-1",
		RouteID:  "route-1",
	})
	if err == nil {
		t.Fatalf("expected AssignOrder to reject held DELAYED/PENDING_WAREHOUSE order")
	}
}

func TestServiceCreate_AllowSubstitutesFalseMustHaveFalseCutsLineAndRecomputesTotal(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	svc := newTestServiceWithResolver(repo, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []LineItem) (ShortageContext, error) {
		return ShortageContext{
			AvailableBySKU: map[string]int64{
				"SKU-CUT":  1,
				"SKU-KEEP": 10,
			},
			CandidatesBySKU: map[string][]SubstituteCandidate{
				"SKU-CUT": {{SKU: "SKU-ALT", SupplierID: "sup-1", BrandID: "B1", AvailableQty: 20, PriceMinor: 2000}},
			},
		}, nil
	})

	resp, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{
			{SKU: "SKU-CUT", BrandID: "B1", Quantity: 3, UnitPrice: 4000, MustHave: false, AllowSubstitutes: false},
			{SKU: "SKU-KEEP", BrandID: "B2", Quantity: 2, UnitPrice: 5000, MustHave: false, AllowSubstitutes: false},
		},
		H3Cell: "872830828ffffff",
		Lat:    41.3,
		Lng:    69.2,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Status != StatusPending {
		t.Fatalf("Status = %s, want PENDING", resp.Status)
	}
	if len(resp.CutLines) != 1 || resp.CutLines[0].Reason != "OUT_OF_STOCK_SUBSTITUTE_DISALLOWED" {
		t.Fatalf("CutLines = %+v, want OUT_OF_STOCK_SUBSTITUTE_DISALLOWED", resp.CutLines)
	}
	if len(resp.LineItems) != 1 || resp.LineItems[0].SKU != "SKU-KEEP" {
		t.Fatalf("expected only SKU-KEEP to survive, got %+v", resp.LineItems)
	}
	if resp.TotalMinor != 10000 {
		t.Fatalf("TotalMinor = %d, want 10000 (2*5000)", resp.TotalMinor)
	}
}

func TestDeferStockReservationAtCreate_Invariants(t *testing.T) {
	if !deferStockReservationAtCreate(true, StatusPending, OrderSourceManual, "", 1) {
		t.Fatalf("expected atomic allocationRequired PENDING create with empty warehouse to defer split reservation")
	}
	if !deferStockReservationAtCreate(true, StatusDelayed, OrderSourceManual, "wh-1", 1) {
		t.Fatalf("expected held DELAYED order to defer stock reservation")
	}
	if deferStockReservationAtCreate(false, StatusPending, OrderSourceManual, "wh-1", 1) {
		t.Fatalf("legacy immediate PENDING create with warehouse should reserve when allocationRequired=false")
	}
}

func TestAutoOrderOptIn_CreateAndAutoConfirmDueOrders(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)

	// Opt-out auto_order must stay SCHEDULED/PENDING (never auto-confirm onto a truck).
	repoOptOut := &testRepo{}
	svcOptOut := newTestServiceWithResolver(repoOptOut, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	respOut, err := svcOptOut.Create(context.Background(), "ret-1", CreateRequest{
		Source:           OrderSourceAutoOrder,
		AutoConfirmOptIn: false,
		LineItems:        []LineItem{{SKU: "SKU-1", Quantity: 2, UnitPrice: 1000}},
		H3Cell:           "872830828ffffff",
		Lat:              41.3,
		Lng:              69.2,
	})
	if err != nil {
		t.Fatalf("auto_order opt-out create failed: %v", err)
	}
	if respOut.Status != StatusScheduled || respOut.ConfirmationStatus != ConfirmationStatusPending || respOut.AutoConfirmOptIn {
		t.Fatalf("unexpected opt-out auto_order response: %+v", respOut)
	}

	// AutoConfirmDueOrders must expire unconfirmed !AutoConfirmOptIn proposals cleanly.
	repoOptOut.found = true
	repoOptOut.order = repoOptOut.created
	if err := svcOptOut.AutoConfirmDueOrders(context.Background(), 10); err != nil {
		t.Fatalf("AutoConfirmDueOrders opt-out failed: %v", err)
	}
	if repoOptOut.captured.Status != StatusCancelled || repoOptOut.captured.ConfirmationStatus != ConfirmationStatusRejected {
		t.Fatalf("expected opt-out proposal to expire CANCELLED/REJECTED, got status=%s conf=%s",
			repoOptOut.captured.Status, repoOptOut.captured.ConfirmationStatus)
	}
	if repoOptOut.captured.AutoConfirmAt != nil {
		t.Fatalf("expected AutoConfirmAt cleared on expiry")
	}

	// Opt-in auto_order auto-confirms cleanly.
	repoOptIn := &testRepo{}
	svcOptIn := newTestServiceWithResolver(repoOptIn, &testWarehouseResolver{warehouseID: "wh-1"}, now)
	respIn, err := svcOptIn.Create(context.Background(), "ret-1", CreateRequest{
		Source:           OrderSourceAutoOrder,
		AutoConfirmOptIn: true,
		LineItems:        []LineItem{{SKU: "SKU-1", Quantity: 2, UnitPrice: 1000}},
		H3Cell:           "872830828ffffff",
		Lat:              41.3,
		Lng:              69.2,
	})
	if err != nil {
		t.Fatalf("auto_order opt-in create failed: %v", err)
	}
	if respIn.Status != StatusPending || respIn.ConfirmationStatus != ConfirmationStatusAutoConfirmed || !respIn.AutoConfirmOptIn {
		t.Fatalf("unexpected opt-in auto_order response: %+v", respIn)
	}

	// Due AI_PREORDER with AutoConfirmOptIn=true promotes to AUTO_CONFIRMED.
	due := now.Add(-time.Minute)
	repoDue := &testRepo{
		found: true,
		order: Order{
			OrderID:            "ord-ai-optin",
			SupplierID:         "sup-1",
			RetailerID:         "ret-1",
			Status:             StatusScheduled,
			Source:             OrderSourceAIPreorder,
			ConfirmationStatus: ConfirmationStatusPending,
			AutoConfirmOptIn:   true,
			AutoConfirmAt:      &due,
		},
	}
	svcDue := newTestService(repoDue, now)
	if err := svcDue.AutoConfirmDueOrders(context.Background(), 10); err != nil {
		t.Fatalf("AutoConfirmDueOrders opt-in failed: %v", err)
	}
	if repoDue.captured.ConfirmationStatus != ConfirmationStatusAutoConfirmed || repoDue.captured.Status != StatusScheduled {
		t.Fatalf("expected AUTO_CONFIRMED scheduled order, got status=%s conf=%s",
			repoDue.captured.Status, repoDue.captured.ConfirmationStatus)
	}
}

func TestPromotePreorderToPending_OptInShortageAndHold(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)

	t.Run("unconfirmed_opt_out_expires_cleanly", func(t *testing.T) {
		repo := &testRepo{}
		svc := newTestService(repo, now)
		err := svc.promotePreorderToPending(context.Background(), Order{
			OrderID:            "ord-unconf-out",
			SupplierID:         "sup-1",
			RetailerID:         "ret-1",
			Status:             StatusScheduled,
			Source:             OrderSourceManualPreorder,
			ConfirmationStatus: ConfirmationStatusPending,
			AutoConfirmOptIn:   false,
			LineItems:          []LineItem{{SKU: "SKU-1", Quantity: 2, UnitPrice: 1000}},
		}, now)
		if err != nil {
			t.Fatalf("promotePreorderToPending error: %v", err)
		}
		if repo.captured.Status != StatusCancelled || repo.captured.ConfirmationStatus != ConfirmationStatusRejected {
			t.Fatalf("expected CANCELLED/REJECTED, got %s/%s", repo.captured.Status, repo.captured.ConfirmationStatus)
		}
	})

	t.Run("confirmed_point2_substitutes_cuts_and_recomputes_total", func(t *testing.T) {
		repo := &testRepo{}
		svc := newTestService(repo, now)
		svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []LineItem) (ShortageContext, error) {
			return ShortageContext{
				AvailableBySKU: map[string]int64{
					"SKU-1":     0,
					"SKU-1-SUB": 10,
					"SKU-2":     0,
				},
				CandidatesBySKU: map[string][]SubstituteCandidate{
					"SKU-1": {{SKU: "SKU-1-SUB", SupplierID: "sup-1", BrandID: "B1", AvailableQty: 10, PriceMinor: 4000}},
				},
			}, nil
		})
		err := svc.promotePreorderToPending(context.Background(), Order{
			OrderID:            "ord-promo-sub",
			SupplierID:         "sup-1",
			RetailerID:         "ret-1",
			Status:             StatusAutoAccepted,
			Source:             OrderSourceManualPreorder,
			ConfirmationStatus: ConfirmationStatusConfirmed,
			DeliveryFeeMinor:   500,
			LineItems: []LineItem{
				{SKU: "SKU-1", BrandID: "B1", Quantity: 2, UnitPrice: 5000, AllowSubstitutes: true},
				{SKU: "SKU-2", BrandID: "B2", Quantity: 1, UnitPrice: 3000, AllowSubstitutes: false, MustHave: false},
			},
			TotalMinor: 13500,
		}, now)
		if err != nil {
			t.Fatalf("promotePreorderToPending error: %v", err)
		}
		if repo.captured.Status != StatusPending {
			t.Fatalf("Status = %s, want PENDING", repo.captured.Status)
		}
		// 2*4000 + 500 delivery fee = 8500
		if repo.captured.TotalMinor != 8500 {
			t.Fatalf("TotalMinor = %d, want 8500", repo.captured.TotalMinor)
		}
		if len(repo.captured.Substitutions) != 1 || repo.captured.Substitutions[0].PriceGapMinor != -2000 {
			t.Fatalf("unexpected Substitutions: %+v", repo.captured.Substitutions)
		}
		if len(repo.captured.CutLines) != 1 || repo.captured.CutLines[0].Reason != "OUT_OF_STOCK_SUBSTITUTE_DISALLOWED" {
			t.Fatalf("unexpected CutLines: %+v", repo.captured.CutLines)
		}
	})

	t.Run("confirmed_must_have_shortage_holds_order", func(t *testing.T) {
		repo := &testRepo{}
		svc := newTestService(repo, now)
		svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []LineItem) (ShortageContext, error) {
			return ShortageContext{AvailableBySKU: map[string]int64{"SKU-MUST": 0}}, nil
		})
		err := svc.promotePreorderToPending(context.Background(), Order{
			OrderID:            "ord-promo-hold",
			SupplierID:         "sup-1",
			RetailerID:         "ret-1",
			Status:             StatusScheduled,
			Source:             OrderSourceManualPreorder,
			ConfirmationStatus: ConfirmationStatusConfirmed,
			LineItems:          []LineItem{{SKU: "SKU-MUST", BrandID: "B1", Quantity: 2, UnitPrice: 5000, MustHave: true}},
		}, now)
		if err != nil {
			t.Fatalf("promotePreorderToPending error: %v", err)
		}
		if repo.captured.Status != StatusDelayed || repo.captured.ConfirmationStatus != ConfirmationStatusPendingWarehouse {
			t.Fatalf("expected DELAYED/PENDING_WAREHOUSE hold, got %s/%s",
				repo.captured.Status, repo.captured.ConfirmationStatus)
		}
	})

	t.Run("all_lines_cut_cancels_cleanly", func(t *testing.T) {
		repo := &testRepo{}
		svc := newTestService(repo, now)
		svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []LineItem) (ShortageContext, error) {
			return ShortageContext{AvailableBySKU: map[string]int64{"SKU-ONLY": 0}}, nil
		})
		err := svc.promotePreorderToPending(context.Background(), Order{
			OrderID:            "ord-promo-cutall",
			SupplierID:         "sup-1",
			RetailerID:         "ret-1",
			Status:             StatusScheduled,
			Source:             OrderSourceManualPreorder,
			ConfirmationStatus: ConfirmationStatusConfirmed,
			LineItems:          []LineItem{{SKU: "SKU-ONLY", Quantity: 2, UnitPrice: 5000}},
			TotalMinor:         10000,
		}, now)
		if err != nil {
			t.Fatalf("promotePreorderToPending error: %v", err)
		}
		if repo.captured.Status != StatusCancelled || repo.captured.TotalMinor != 0 {
			t.Fatalf("expected CANCELLED total 0 when all lines cut, got %s total=%d", repo.captured.Status, repo.captured.TotalMinor)
		}
	})
}

func TestEditPreorderAndConfirmAIOrder_PackValidationAndOptIn(t *testing.T) {
	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	deliv := now.Add(5 * 24 * time.Hour)
	optTrue := true

	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:               "ord-edit-1",
			SupplierID:            "sup-1",
			RetailerID:            "ret-1",
			WarehouseID:           "wh-1",
			Status:                StatusScheduled,
			Source:                OrderSourceManualPreorder,
			ConfirmationStatus:    ConfirmationStatusPending,
			RequestedDeliveryDate: &deliv,
			Timezone:              "UTC",
			LineItems:             []LineItem{{SKU: "SKU-CASE", Quantity: 6, UnitPrice: 1000}},
		},
	}
	svc := newTestService(repo, now)
	svc.SetPackRuleReader(func(context.Context, []string) (map[string]PackRule, error) {
		return map[string]PackRule{"SKU-CASE": {SaleUnit: "CASE", UnitsPerPack: 6}}, nil
	})

	_, err := svc.EditPreorder(context.Background(), "ret-1", EditPreorderRequest{
		OrderID:               "ord-edit-1",
		LineItems:             []LineItem{{SKU: "SKU-CASE", Quantity: 5, UnitPrice: 1000}},
		RequestedDeliveryDate: deliv.Format(time.RFC3339),
		AutoConfirmOptIn:      &optTrue,
	})
	var pv *PackValidationError
	if !errors.As(err, &pv) || pv.Code != "PACK_MULTIPLE_REQUIRED" {
		t.Fatalf("expected PackValidationError PACK_MULTIPLE_REQUIRED, got %v", err)
	}

	resp, err := svc.EditPreorder(context.Background(), "ret-1", EditPreorderRequest{
		OrderID:               "ord-edit-1",
		LineItems:             []LineItem{{SKU: "SKU-CASE", Quantity: 12, UnitPrice: 1000}},
		RequestedDeliveryDate: deliv.Format(time.RFC3339),
		AutoConfirmOptIn:      &optTrue,
	})
	if err != nil {
		t.Fatalf("EditPreorder valid pack failed: %v", err)
	}
	if !resp.AutoConfirmOptIn || !repo.captured.AutoConfirmOptIn {
		t.Fatalf("expected AutoConfirmOptIn persisted on EditPreorder")
	}

	// ConfirmAIOrder pack validation + opt-in persistence.
	repoAI := &testRepo{
		found: true,
		order: Order{
			OrderID:            "ord-ai-edit",
			SupplierID:         "sup-1",
			RetailerID:         "ret-1",
			WarehouseID:        "wh-1",
			Status:             StatusScheduled,
			Source:             OrderSourceAIPreorder,
			ConfirmationStatus: ConfirmationStatusPending,
			LineItems:          []LineItem{{SKU: "SKU-CASE", Quantity: 6, UnitPrice: 1000}},
		},
	}
	svcAI := newTestService(repoAI, now)
	svcAI.SetPackRuleReader(func(context.Context, []string) (map[string]PackRule, error) {
		return map[string]PackRule{"SKU-CASE": {SaleUnit: "CASE", UnitsPerPack: 6}}, nil
	})
	_, err = svcAI.ConfirmAIOrder(context.Background(), "ret-1", ConfirmAIOrderRequest{
		OrderID:   "ord-ai-edit",
		LineItems: []LineItem{{SKU: "SKU-CASE", Quantity: 7, UnitPrice: 1000}},
	})
	if !errors.As(err, &pv) {
		t.Fatalf("ConfirmAIOrder expected PackValidationError, got %v", err)
	}

	aiResp, err := svcAI.ConfirmAIOrder(context.Background(), "ret-1", ConfirmAIOrderRequest{
		OrderID:          "ord-ai-edit",
		LineItems:        []LineItem{{SKU: "SKU-CASE", Quantity: 12, UnitPrice: 1000}},
		AutoConfirmOptIn: &optTrue,
	})
	if err != nil {
		t.Fatalf("ConfirmAIOrder valid pack failed: %v", err)
	}
	if !aiResp.AutoConfirmOptIn || !repoAI.captured.AutoConfirmOptIn {
		t.Fatalf("expected AutoConfirmOptIn persisted on ConfirmAIOrder")
	}
}
