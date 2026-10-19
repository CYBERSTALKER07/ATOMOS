package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleReportsSummaryInventoryErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.stockBalancesQuery = func(context.Context, string, string) ([]StockBalanceDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsSummary, "/v1/retailer/reports/summary")
}

func TestHandleReportsSummaryShiftsErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.shiftsQuery = func(context.Context, string, string, int) ([]ShiftDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsSummary, "/v1/retailer/reports/summary")
}

func TestHandleReportsInventoryErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.stockBalancesQuery = func(context.Context, string, string) ([]StockBalanceDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsInventory, "/v1/retailer/reports/inventory")
}

func TestHandleReportsSummaryPacksLoadErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.enabledPacksQuery = func(context.Context, string) (EnabledSet, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsSummary, "/v1/retailer/reports/summary")
	svc.enabledPacksQuery = nil
	enabled, err := svc.LoadEnabledPacks(context.Background(), "ret-rep")
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Has(PackREPORTSPRO) {
		t.Fatal("must not auto-enable REPORTS_PRO when pack load failed")
	}
}

func TestHandleReportsSummaryPackEnableErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.setPackEnabledFn = func(context.Context, string, string, string, bool, map[string]any) error {
		return errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsSummary, "/v1/retailer/reports/summary")
}

func TestHandleReportsShiftsErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.shiftsQuery = func(context.Context, string, string, int) ([]ShiftDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsShifts, "/v1/retailer/reports/shifts")
}

func TestHandleReportsSummarySalesErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.posSalesQuery = func(context.Context, string, string, time.Time, time.Time) ([]PosSaleDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsSummary, "/v1/retailer/reports/summary")
}

func TestHandleReportsSalesErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.posSalesQuery = func(context.Context, string, string, time.Time, time.Time) ([]PosSaleDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertReportsFailed(t, svc.HandleReportsSales, "/v1/retailer/reports/sales")
}

func TestHandleReportsSummarySalesFromLedger(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{Now: func() time.Time { return now }})
	svc.posSalesQuery = func(_ context.Context, orgID, locID string, from, to time.Time) ([]PosSaleDTO, error) {
		if orgID != "ret-rep" {
			t.Fatalf("orgID=%s", orgID)
		}
		if locID != "" {
			t.Fatalf("locID=%s", locID)
		}
		if !from.Before(to) {
			t.Fatalf("window from=%s to=%s", from, to)
		}
		return []PosSaleDTO{{
			SaleID: "s1", RetailerID: orgID, Status: PosSaleCompleted, TotalMinor: 1500,
			CreatedAt: now.Add(-time.Hour).Format(time.RFC3339Nano),
			Lines:     []PosSaleLine{{Sku: "A", Qty: 1, UnitPriceMinor: 1500, LineTotalMinor: 1500}},
		}}, nil
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/reports/summary", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-rep",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	svc.HandleReportsSummary(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["sales_minor"].(float64) != 1500 {
		t.Fatalf("sales_minor=%v", payload["sales_minor"])
	}
	if payload["sale_count"].(float64) != 1 {
		t.Fatalf("sale_count=%v", payload["sale_count"])
	}
}

func assertReportsFailed(t *testing.T, handler http.HandlerFunc, path string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-rep",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "reports_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["on_hand_sku_count"]; ok {
		t.Fatal("failed reports must not return zeroed summary tiles")
	}
	if _, ok := payload["items"]; ok {
		t.Fatal("failed reports must not return items[]")
	}
}
