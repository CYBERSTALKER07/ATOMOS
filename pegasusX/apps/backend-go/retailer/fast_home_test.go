package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func TestHandleFastHomeMemoryAndCutoffSync(t *testing.T) {
	t.Parallel()

	// 12:00 UTC = 17:00 Asia/Tashkent (before 18:00 cutoff -> same-day cutoff, tomorrow delivery)
	beforeCutoff := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	repo := &testRetailerRepo{
		retailer: Retailer{RetailerID: "ret-fh-1", CountryCode: "UZ"},
		found:    true,
		tracking: []TrackingOrder{
			{
				OrderID:    "ord-arriving-1",
				SupplierID: "sup-1",
				Status:     "CONFIRMED",
				TotalMinor: 45000000,
				Currency:   "UZS",
				Items:      []TrackingLineItem{{ProductID: "sku-cola", Quantity: 10}},
			},
		},
	}
	ordersFake := &testOrderLifecycle{
		predictions: []order.RetailerAIPrediction{
			{
				OrderID:            "ord-ai-99",
				SupplierID:         "sup-1",
				Source:             order.OrderSourceAIPreorder,
				ConfirmationStatus: order.ConfirmationStatusPending,
				AutoConfirmOptIn:   true,
				TotalMinor:         12000000,
				Currency:           "UZS",
				Items: []order.LineItem{
					{SKU: "sku-water", Name: "Water 1.5L", Quantity: 24, UnitPrice: 500000},
				},
			},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:   repo,
		Orders: ordersFake,
		Now:    func() time.Time { return beforeCutoff },
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/fast-home", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:           auth.RoleRetailer,
		Subject:        "ret-fh-1",
		RetailerOrgID:  "ret-fh-1",
		RetailerUserID: "usr-owner-1",
		RetailerRole:   "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleFastHome(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	var got FastHomeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.CutoffAt != "2026-11-13T13:00:00Z" {
		t.Fatalf("cutoff_at = %q, want 2026-11-13T13:00:00Z (18:00 Tashkent)", got.CutoffAt)
	}
	if got.NextDeliveryDate != "2026-11-14" {
		t.Fatalf("next_delivery_date = %q, want 2026-11-14", got.NextDeliveryDate)
	}
	if got.TomorrowDelivery == nil || got.TomorrowDelivery.OrderID != "ord-arriving-1" {
		t.Fatalf("tomorrow_delivery = %+v, want ord-arriving-1", got.TomorrowDelivery)
	}
	if got.PendingProposal == nil || got.PendingProposal.OrderID != "ord-ai-99" || !got.PendingProposal.AutoConfirmOptIn || !got.PendingProposal.CanConfirm || len(got.PendingProposal.Items) != 1 {
		t.Fatalf("pending_proposal = %+v", got.PendingProposal)
	}

	// After 18:00 Tashkent (14:00 UTC = 19:00 Tashkent) -> cutoff rolls to next day 18:00 Tashkent and next_delivery_date rolls +2 days.
	afterCutoff := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	cutoffUTC, nextDate := computeSupplierCutoff(afterCutoff, "Asia/Tashkent", "18:00", 1)
	if cutoffUTC.Format(time.RFC3339) != "2026-11-14T13:00:00Z" || nextDate != "2026-11-15" {
		t.Fatalf("after cutoff got (%s, %s), want (2026-11-14T13:00:00Z, 2026-11-15)", cutoffUTC.Format(time.RFC3339), nextDate)
	}
}

func TestHandleFastHomeNonOwnerHidesPendingApprovals(t *testing.T) {
	t.Parallel()

	svc := NewService(ServiceConfig{Repo: &testRetailerRepo{}})
	svc.SetFastHomeQuery(func(_ context.Context, retailerID, _ string) (*FastHomeResponse, error) {
		return &FastHomeResponse{
			ServerNow:             "2026-11-13T08:00:00Z",
			CutoffAt:              "2026-11-13T13:00:00Z",
			NextDeliveryDate:      "2026-11-14",
			BalanceOwedMinor:      25000000,
			OverdueMinor:          5000000,
			DaysOverdue:           3,
			CreditHeadroomMinor:   75000000,
			PendingApprovalsCount: 4,
			Locale:                "ru",
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/fast-home", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:           auth.RoleRetailer,
		Subject:        "ret-fh-2",
		RetailerOrgID:  "ret-fh-2",
		RetailerUserID: "usr-cashier",
		RetailerRole:   "CASHIER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleFastHome(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var got FastHomeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.PendingApprovalsCount != 0 {
		t.Fatalf("cashier pending_approvals_count = %d, want 0", got.PendingApprovalsCount)
	}
	if got.BalanceOwedMinor != 25000000 || got.DaysOverdue != 3 || got.Locale != "ru" {
		t.Fatalf("unexpected credit/locale fields: %+v", got)
	}
}
