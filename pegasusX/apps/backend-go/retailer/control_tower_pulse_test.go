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

func TestControlTowerPulseEmptyAndLive(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "ct-" + string(rune('A'+n%26))
		},
	})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-ct",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleControlTowerPulse(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var pulse ControlTowerPulse
	_ = json.Unmarshal(rr.Body.Bytes(), &pulse)
	if !pulse.Empty {
		t.Fatalf("expected empty pulse, got %+v", pulse)
	}
	if pulse.RetailerID != "ret-ct" {
		t.Fatalf("retailer_id=%s", pulse.RetailerID)
	}
	if len(pulse.Capabilities) == 0 {
		t.Fatal("capabilities should include CORE")
	}
	if pulse.Source != "empty" {
		t.Fatalf("source=%s want empty", pulse.Source)
	}
	if pulse.Loyalty.Enrolled {
		t.Fatal("empty pulse must not invent Bronze enrollment")
	}
	if len(pulse.OrdersByStatus) != len(retailerOrderStatusFunnel) {
		t.Fatalf("orders_by_status keys=%d want %d", len(pulse.OrdersByStatus), len(retailerOrderStatusFunnel))
	}
	for _, key := range retailerOrderStatusFunnel {
		if pulse.OrdersByStatus[key] != 0 {
			t.Fatalf("empty %s=%d", key, pulse.OrdersByStatus[key])
		}
	}

	// Seed open POS session + sale → non-empty sales and sessions
	primary, _ := svc.EnsurePrimaryLocation(t.Context(), "ret-ct")
	regID := svc.newID()
	sessID := svc.newID()
	_ = svc.saveRegister(t.Context(), RegisterDTO{
		RegisterID: regID, RetailerID: "ret-ct", LocationID: primary.LocationID,
		Label: "R", Status: RegisterStatusActive,
	})
	_ = svc.savePosSession(t.Context(), PosSessionDTO{
		SessionID: sessID, RegisterID: regID, LocationID: primary.LocationID,
		RetailerID: "ret-ct", OpenedByUserID: "o", Status: PosSessionOpen,
		OpeningFloatMinor: 0, Currency: "UZS",
		OpenedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	_ = svc.savePosSale(t.Context(), PosSaleDTO{
		SaleID: svc.newID(), SessionID: sessID, RegisterID: regID,
		LocationID: primary.LocationID, RetailerID: "ret-ct", CashierUserID: "o",
		Status: "COMPLETED", TotalMinor: 1200, Currency: "UZS", ReceiptNumber: "RCT1",
		Lines:     []PosSaleLine{{Sku: "X", Qty: 1, UnitPriceMinor: 1200, LineTotalMinor: 1200}},
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	_ = svc.applyDelta(t.Context(), "ret-ct", primary.LocationID, BinFloor, "X", 3, MoveReceive, "TEST", "o1", "o", "")

	req2 := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	req2 = req2.WithContext(auth.WithClaims(req2.Context(), owner))
	rr2 := httptest.NewRecorder()
	svc.HandleControlTowerPulse(rr2, req2)
	_ = json.Unmarshal(rr2.Body.Bytes(), &pulse)
	if pulse.Empty {
		t.Fatalf("expected non-empty after seed, got %+v", pulse)
	}
	if pulse.PosOpenSessions < 1 {
		t.Fatalf("pos_open_sessions=%d", pulse.PosOpenSessions)
	}
	if pulse.SalesMinor7d < 1200 {
		t.Fatalf("sales_minor_7d=%d", pulse.SalesMinor7d)
	}
}

func TestControlTowerPulseFiscalFailedSupplierFacet(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now,
		DashboardOrders: func(_ context.Context, retailerID string) ([]RetailerOrderStatusRow, error) {
			if retailerID != "ret-ct-facet" {
				t.Fatalf("retailerID=%s", retailerID)
			}
			return []RetailerOrderStatusRow{
				{SupplierID: "sup-a", Status: "FISCAL_FAILED", Count: 1},
				{SupplierID: "sup-b", Status: "COMPLETED", Count: 1},
			}, nil
		},
	})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-ct-facet",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleControlTowerPulse(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var pulse ControlTowerPulse
	if err := json.Unmarshal(rr.Body.Bytes(), &pulse); err != nil {
		t.Fatal(err)
	}
	if pulse.Empty {
		t.Fatal("FISCAL_FAILED child must not look like an empty pulse")
	}
	if pulse.Source != "spanner" {
		t.Fatalf("source=%s", pulse.Source)
	}
	if pulse.Loyalty.Enrolled {
		t.Fatal("pulse loyalty must stay unenrolled (no invented Bronze)")
	}
	if pulse.OrdersByStatus["FISCAL_FAILED"] != 1 {
		t.Fatalf("stack FISCAL_FAILED=%d", pulse.OrdersByStatus["FISCAL_FAILED"])
	}
	if pulse.OrdersByStatus["COMPLETED"] != 1 {
		t.Fatalf("stack COMPLETED=%d", pulse.OrdersByStatus["COMPLETED"])
	}
	if pulse.OpenOrders != 1 {
		t.Fatalf("open_orders=%d want 1 (child FISCAL_FAILED, not blended parent)", pulse.OpenOrders)
	}
	if len(pulse.OrdersByStatus) != 17 {
		t.Fatalf("funnel keys=%d", len(pulse.OrdersByStatus))
	}
	var sawA, sawB bool
	for _, facet := range pulse.OrdersBySupplier {
		switch facet.SupplierID {
		case "sup-a":
			sawA = true
			if facet.OrdersByStatus["FISCAL_FAILED"] != 1 {
				t.Fatalf("supplier A FISCAL_FAILED=%d", facet.OrdersByStatus["FISCAL_FAILED"])
			}
			if facet.OrdersByStatus["COMPLETED"] != 0 {
				t.Fatalf("supplier A must not blend B COMPLETED")
			}
		case "sup-b":
			sawB = true
			if facet.OrdersByStatus["COMPLETED"] != 1 {
				t.Fatalf("supplier B COMPLETED=%d", facet.OrdersByStatus["COMPLETED"])
			}
			if facet.OrdersByStatus["FISCAL_FAILED"] != 0 {
				t.Fatalf("supplier B must not hide behind a blended total")
			}
		}
	}
	if !sawA || !sawB {
		t.Fatalf("facets=%+v", pulse.OrdersBySupplier)
	}
}

func TestControlTowerPulseTrackingErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:  time.Now,
		Repo: &testRetailerRepo{trackingErr: errors.New("spanner_unavailable")},
	})
	assertControlTowerPulseFailed(t, svc, "ret-ct-track")
}

func TestControlTowerPulseDashboardOrdersErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now: time.Now,
		DashboardOrders: func(_ context.Context, retailerID string) ([]RetailerOrderStatusRow, error) {
			if retailerID != "ret-ct-dash" {
				t.Fatalf("retailerID=%s", retailerID)
			}
			return nil, errors.New("spanner_unavailable")
		},
	})
	assertControlTowerPulseFailed(t, svc, "ret-ct-dash")
}

func TestControlTowerPulseSalesErrorFailed(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now})
	svc.posSalesQuery = func(context.Context, string, string, time.Time, time.Time) ([]PosSaleDTO, error) {
		return nil, errors.New("spanner_unavailable")
	}
	assertControlTowerPulseFailed(t, svc, "ret-ct-sales")
}

func assertControlTowerPulseFailed(t *testing.T, svc *Service, orgID string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/control-tower/pulse", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: orgID,
		RetailerRole: "OWNER", RetailerUserID: "o",
	}))
	rr := httptest.NewRecorder()
	svc.HandleControlTowerPulse(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "control_tower_pulse_failed" {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["empty"]; ok {
		t.Fatal("failed pulse must not return empty")
	}
	if _, ok := payload["open_orders"]; ok {
		t.Fatal("failed pulse must not return zeroed tiles")
	}
}
