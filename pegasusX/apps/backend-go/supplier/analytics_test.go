package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestBuildVelocityResponseCountsCreatedAndCompleted(t *testing.T) {
	now := time.Date(2026, 6, 14, 15, 0, 0, 0, time.UTC)
	orders := []SupplierOrder{
		{OrderID: "o-1", Status: "PENDING", CreatedAt: "2026-06-14T10:00:00Z", UpdatedAt: "2026-06-14T10:00:00Z"},
		{OrderID: "o-2", Status: "COMPLETED", CreatedAt: "2026-06-13T10:00:00Z", UpdatedAt: "2026-06-13T18:00:00Z"},
	}
	resp := buildVelocityResponse(orders, now, 2)
	if len(resp.Points) != 2 {
		t.Fatalf("points=%d want=2", len(resp.Points))
	}
	if resp.Points[1].OrdersCreated != 1 {
		t.Fatalf("created today=%d want=1", resp.Points[1].OrdersCreated)
	}
	if resp.Points[0].OrdersCompleted != 1 {
		t.Fatalf("completed yesterday=%d want=1", resp.Points[0].OrdersCompleted)
	}
}

func TestBuildRevenueResponseSumsCompletedOrders(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	orders := []SupplierOrder{
		{Status: "COMPLETED", TotalMinor: 5000, UpdatedAt: "2026-06-14T09:00:00Z"},
		{Status: "COMPLETED", TotalMinor: 2500, UpdatedAt: "2026-06-14T11:00:00Z"},
		{Status: "PENDING", TotalMinor: 9000, UpdatedAt: "2026-06-14T11:00:00Z"},
	}
	resp := buildRevenueResponse(orders, "UZS", now, 1)
	if resp.TotalMinor != 7500 {
		t.Fatalf("total_minor=%d want=7500", resp.TotalMinor)
	}
	if resp.Currency != "UZS" {
		t.Fatalf("currency=%q", resp.Currency)
	}
}

func TestBuildRevenueResponse_EmptyCurrencyDoesNotInventUZS(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	resp := buildRevenueResponse(nil, "", now, 1)
	if resp.Currency != "" {
		t.Fatalf("currency=%q want empty", resp.Currency)
	}
}

func TestHandleAnalyticsRevenue_EmptyCurrencyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	svc := NewService(ServiceConfig{SupplierID: "sup-rev"})
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/analytics/revenue", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-rev", MarketCode: "UZ",
	}))
	rr := httptest.NewRecorder()
	svc.HandleAnalyticsRevenue(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Currency != "UZS" {
		t.Fatalf("currency=%q want pack UZS", body.Currency)
	}
}

func TestHandleAnalyticsRevenue_PlannedFailsClosed(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-ca"})
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/analytics/revenue", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-ca", MarketCode: "CA",
	}))
	rr := httptest.NewRecorder()
	svc.HandleAnalyticsRevenue(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestImportRowCurrency_EmptyUsesPack(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-imp", MarketCode: "UZ",
	})
	cur, err := importRowCurrency(ctx, "sup-imp", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cur != "UZS" {
		t.Fatalf("currency=%q want pack UZS", cur)
	}
}

func TestImportRowCurrency_USDOnUZRejected(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-imp-usd", MarketCode: "UZ",
	})
	_, err := importRowCurrency(ctx, "sup-imp-usd", map[string]any{"currency": "USD"}, nil)
	if err != auth.ErrPackCurrencyMismatch {
		t.Fatalf("err=%v want pack_currency_mismatch", err)
	}
}

func TestImportRowCurrency_PlannedFailsClosed(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-imp-ca", MarketCode: "CA",
	})
	_, err := importRowCurrency(ctx, "sup-imp-ca", nil, nil)
	if err == nil {
		t.Fatal("expected planned pack to fail closed")
	}
}

func TestBuildDemandHistoryFromDayMaps(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	predicted := map[string]int64{"2026-06-14": 12, "2026-06-13": 8}
	actual := map[string]int64{"2026-06-14": 10, "2026-06-13": 9}
	recs := []AIRecommendation{
		{
			RecommendationID: "r-1",
			AggregateID:      "ret-1",
			AggregateType:    "DEMAND",
			Action:           "RESTOCK",
			Explanation:      "sku-a",
			Score:            5,
			GeneratedAt:      "2026-06-14T08:00:00Z",
			Status:           "PENDING",
		},
	}
	resp := buildDemandHistoryFromDayMaps(predicted, actual, recs, now, 14)
	if len(resp.TimeSeries) != 14 {
		t.Fatalf("time_series=%d want=14", len(resp.TimeSeries))
	}
	last := resp.TimeSeries[len(resp.TimeSeries)-1]
	if last.ActualQty != 10 || last.PredictedQty != 12 {
		t.Fatalf("last point actual=%d predicted=%d", last.ActualQty, last.PredictedQty)
	}
	if len(resp.Upcoming) != 1 || resp.Upcoming[0].PredictedQty != 5 {
		t.Fatalf("upcoming=%+v", resp.Upcoming)
	}
}

func TestParseInventoryImportCSV(t *testing.T) {
	csvBody := []byte("product_id,warehouse_id,quantity_on_hand,reorder_threshold\nsku-1,wh-1,10,2\n")
	rows, err := parseInventoryImportCSV(csvBody)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 1 || rows[0].ProductID != "sku-1" || rows[0].QuantityOnHand != 10 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}
