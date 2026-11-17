package retailer

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func TestAutoOrderWorkerUsesRealStoreStockAndSalesHistory(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	ordersFake := &testOrderLifecycle{
		predictions: []order.RetailerAIPrediction{
			{
				OrderID:    "ai-ord-1",
				SupplierID: "sup-1",
				Source:     order.OrderSourceAIPreorder,
				Items: []order.LineItem{
					{SKU: "sku-milk-1l", Name: "Milk 1L", Quantity: 3},
				},
			},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:   &testRetailerRepo{},
		Orders: ordersFake,
		Now:    func() time.Time { return now },
	})

	// Seed real StoreStock across two store locations (30 + 12 = 42 on hand).
	svc.mu.Lock()
	svc.stockBalances[stockBalanceKey{LocationID: "loc-main", StockBin: "SELLABLE", Sku: "sku-milk-1l"}] = memStockBalance{
		RetailerID: "ret-1",
		OnHand:     30,
	}
	svc.stockBalances[stockBalanceKey{LocationID: "loc-chiller", StockBin: "SELLABLE", Sku: "sku-milk-1l"}] = memStockBalance{
		RetailerID: "ret-1",
		OnHand:     12,
	}
	// Other retailer / SKU stock must be ignored.
	svc.stockBalances[stockBalanceKey{LocationID: "loc-other", StockBin: "SELLABLE", Sku: "sku-milk-1l"}] = memStockBalance{
		RetailerID: "ret-other",
		OnHand:     999,
	}
	// Seed 4 trailing days of POS sell-through history.
	svc.sellThroughDaily[sellThroughKey{RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-10"}] = SellThroughDayDTO{
		RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-10", NetSold: 8,
	}
	svc.sellThroughDaily[sellThroughKey{RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-11"}] = SellThroughDayDTO{
		RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-11", NetSold: 12,
	}
	svc.sellThroughDaily[sellThroughKey{RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-12"}] = SellThroughDayDTO{
		RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-12", NetSold: 10,
	}
	svc.sellThroughDaily[sellThroughKey{RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-13"}] = SellThroughDayDTO{
		RetailerID: "ret-1", LocationID: "loc-main", SkuID: "sku-milk-1l", Day: "2026-11-13", NetSold: 14,
	}
	svc.mu.Unlock()

	var captured DemandPredictionRequest
	svc.predictDemandFn = func(_ context.Context, req DemandPredictionRequest) (*DemandPredictionResponse, error) {
		captured = req
		return &DemandPredictionResponse{SKU: req.SKU, OptimalReorderQty: 18}, nil
	}

	cands, src := svc.loadAutoOrderCandidatesWithSource(context.Background(), "ret-1")
	if src != "ai_predictions" {
		t.Fatalf("source = %q, want ai_predictions", src)
	}
	if captured.CurrentStock != 42 {
		t.Fatalf("CurrentStock = %d, want 42 (not stub 5)", captured.CurrentStock)
	}
	wantHistory := []int{8, 12, 10, 14}
	if !reflect.DeepEqual(captured.HistoricalDailySales, wantHistory) {
		t.Fatalf("HistoricalDailySales = %v, want %v", captured.HistoricalDailySales, wantHistory)
	}
	if len(cands) != 1 || cands[0].Qty != 18 || cands[0].SupplierID != "sup-1" {
		t.Fatalf("unexpected candidates: %+v", cands)
	}
}

func TestAutoOrderWorkerFallbackUsesOrderHistoryAndStock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)
	repo := &testRetailerRepo{
		receipts: []TrackingOrder{
			{
				OrderID:   "ord-h1",
				CreatedAt: "2026-11-10T10:00:00Z",
				Items:     []TrackingLineItem{{ProductID: "sku-bread", Quantity: 6}},
			},
			{
				OrderID:   "ord-h2",
				CreatedAt: "2026-11-11T10:00:00Z",
				Items:     []TrackingLineItem{{ProductID: "sku-bread", Quantity: 12}},
			},
			{
				OrderID:   "ord-h3",
				CreatedAt: "2026-11-12T10:00:00Z",
				Items:     []TrackingLineItem{{ProductID: "sku-bread", Quantity: 9}},
			},
		},
	}
	ordersFake := &testOrderLifecycle{
		predictions: []order.RetailerAIPrediction{
			{
				OrderID:    "ai-ord-2",
				SupplierID: "sup-2",
				Source:     order.OrderSourceAIPreorder,
				Items: []order.LineItem{
					{SKU: "sku-bread", Name: "Bread", Quantity: 1},
				},
			},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:   repo,
		Orders: ordersFake,
		Now:    func() time.Time { return now },
	})

	var captured []int
	svc.predictDemandFn = func(_ context.Context, req DemandPredictionRequest) (*DemandPredictionResponse, error) {
		captured = append([]int(nil), req.HistoricalDailySales...)
		return nil, errors.New("sidecar_offline")
	}

	// Low stock (2 on hand): avg daily = ceil((6+12+9)/3) = 9; 3-day cover = 27 -> suggest 25.
	svc.mu.Lock()
	svc.stockBalances[stockBalanceKey{LocationID: "loc-1", StockBin: "SELLABLE", Sku: "sku-bread"}] = memStockBalance{
		RetailerID: "ret-2",
		OnHand:     2,
	}
	svc.mu.Unlock()

	cands, _ := svc.loadAutoOrderCandidatesWithSource(context.Background(), "ret-2")
	wantHistory := []int{6, 12, 9}
	if !reflect.DeepEqual(captured, wantHistory) {
		t.Fatalf("captured order history = %v, want %v", captured, wantHistory)
	}
	if len(cands) != 1 || cands[0].Qty != 25 {
		t.Fatalf("low-stock fallback candidates = %+v, want qty 25", cands)
	}

	// High stock (100 on hand > 27 target): nothing should be ordered.
	svc.mu.Lock()
	svc.stockBalances[stockBalanceKey{LocationID: "loc-1", StockBin: "SELLABLE", Sku: "sku-bread"}] = memStockBalance{
		RetailerID: "ret-2",
		OnHand:     100,
	}
	svc.mu.Unlock()

	candsHigh, _ := svc.loadAutoOrderCandidatesWithSource(context.Background(), "ret-2")
	if len(candsHigh) != 0 {
		t.Fatalf("high-stock fallback candidates = %+v, want empty", candsHigh)
	}
}
