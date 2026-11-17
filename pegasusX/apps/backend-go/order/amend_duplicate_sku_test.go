package order

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestAmendOrder_DuplicateSKU_AllocatesAcceptedAcrossLinesAndGeneratesSingleReturn(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusArrived)
	o.OrderID = "ord-dup-sku-amend"
	o.DriverID = "driver-1"
	// Multi-lot split: same SKU across two lines
	o.LineItems = []LineItem{
		{
			SKU:       "sku-apple",
			Name:      "Apples Lot 1",
			Quantity:  10,
			UnitPrice: 1000,
		},
		{
			SKU:       "sku-apple",
			Name:      "Apples Lot 2",
			Quantity:  15,
			UnitPrice: 1000,
		},
	}
	o.TotalMinor = 25000
	o.OriginalTotalMinor = 25000

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	driverClaims := auth.Claims{
		Subject:    "driver-1",
		Role:       auth.RoleDriver,
		MarketCode: "UZ",
	}

	// Retailer accepts 20, rejects 5 of sku-apple (which spans two lines totalling 25)
	resp, err := svc.AmendOrder(context.Background(), driverClaims, AmendOrderRequest{
		OrderID: o.OrderID,
		Items: []AmendItemRequest{
			{
				ProductID:   "sku-apple",
				AcceptedQty: 20,
				RejectedQty: 5,
				Reason:      "DAMAGED",
			},
		},
		DriverNotes: "Crate dropped during offloading",
	})
	if err != nil {
		t.Fatalf("AmendOrder failed: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected AmendOrder success true")
	}
	if resp.AdjustedTotal != 20000 {
		t.Fatalf("expected AdjustedTotal 20000, got %d", resp.AdjustedTotal)
	}

	// Verify line items allocated: line-lot-1 got 10, line-lot-2 got remaining 10
	if len(repo.captured.LineItems) != 2 {
		t.Fatalf("expected 2 line items preserved, got %d", len(repo.captured.LineItems))
	}
	if repo.captured.LineItems[0].Quantity != 10 {
		t.Fatalf("line 1 qty expected 10, got %d", repo.captured.LineItems[0].Quantity)
	}
	if repo.captured.LineItems[1].Quantity != 10 {
		t.Fatalf("line 2 qty expected 10, got %d", repo.captured.LineItems[1].Quantity)
	}

	// Verify single consolidated return for the SKU
	if len(repo.captured.PendingSupplierReturns) != 1 {
		t.Fatalf("expected 1 return entry, got %d", len(repo.captured.PendingSupplierReturns))
	}
	sr := repo.captured.PendingSupplierReturns[0]
	if sr.SKU != "sku-apple" || sr.RejectedQty != 5 || sr.Reason != "DAMAGED" {
		t.Fatalf("unexpected return entry: %+v", sr)
	}
}

func TestSubmitDamageProof_DuplicateSKU_AggregatesAndAmends(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusArrived)
	o.OrderID = "ord-dup-sku-damage"
	o.DriverID = "driver-1"
	o.LineItems = []LineItem{
		{
			SKU:       "sku-milk",
			Name:      "Milk Lot 1",
			Quantity:  8,
			UnitPrice: 2000,
		},
		{
			SKU:       "sku-milk",
			Name:      "Milk Lot 2",
			Quantity:  12,
			UnitPrice: 2000,
		},
	}
	o.TotalMinor = 40000
	o.OriginalTotalMinor = 40000

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	driverClaims := auth.Claims{
		Subject:    "driver-1",
		Role:       auth.RoleDriver,
		MarketCode: "UZ",
	}

	// 1. Damage report exceeding total (e.g. 25 damaged when order only has 20 across lines) must return 400
	{
		bodyExceed := map[string]any{
			"order_id": o.OrderID,
			"damaged_items": []map[string]any{
				{"sku": "sku-milk", "quantity": 25},
			},
			"reason": "Spoiled during transit",
		}
		raw, _ := json.Marshal(bodyExceed)
		req := httptest.NewRequest(http.MethodPost, "/v1/delivery/report-damage", bytes.NewReader(raw))
		req = req.WithContext(auth.WithClaims(req.Context(), driverClaims))
		rec := httptest.NewRecorder()

		svc.HandleReportDamage(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for exceeding quantity, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	// 2. Valid damage report (6 damaged out of 20 total) succeeds
	{
		bodyValid := map[string]any{
			"order_id": o.OrderID,
			"damaged_items": []map[string]any{
				{"sku": "sku-milk", "quantity": 6},
			},
			"reason": "Leaking cartons",
		}
		raw, _ := json.Marshal(bodyValid)
		req := httptest.NewRequest(http.MethodPost, "/v1/delivery/report-damage", bytes.NewReader(raw))
		req.Header.Set("Idempotency-Key", "idem-dmg-valid-1")
		req = req.WithContext(auth.WithClaims(req.Context(), driverClaims))
		rec := httptest.NewRecorder()

		svc.HandleReportDamage(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for valid damage report, got %d: %s", rec.Code, rec.Body.String())
		}

		// 14 accepted * 2000 = 28000
		if repo.captured.TotalMinor != 28000 {
			t.Fatalf("expected TotalMinor 28000, got %d", repo.captured.TotalMinor)
		}
		if len(repo.captured.ConditionReports) != 1 {
			t.Fatalf("expected 1 condition report, got %d", len(repo.captured.ConditionReports))
		}
	}
}
