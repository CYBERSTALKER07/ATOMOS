package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
)

type pricingTestRepo struct {
	profile      Profile
	profileFound bool
	rule         SupplierPricingRule
	ruleFound    bool
}

func (r *pricingTestRepo) GetProfile(_ context.Context, _ string) (Profile, bool, error) {
	if !r.profileFound {
		return Profile{}, false, nil
	}
	return r.profile, true, nil
}

func (r *pricingTestRepo) UpdateProfile(_ context.Context, _ Profile, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *pricingTestRepo) GetAuthByPhone(_ context.Context, _ string) (SupplierAuthRecord, bool, error) {
	return SupplierAuthRecord{}, false, nil
}

func (r *pricingTestRepo) GetTopology(_ context.Context, _ string) (SupplierTopology, error) {
	return SupplierTopology{}, nil
}

func (r *pricingTestRepo) ReplaceTopology(_ context.Context, _ string, _ SupplierTopology, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *pricingTestRepo) ListOrgMembers(_ context.Context, _ string) ([]SupplierOrgMember, error) {
	return nil, nil
}

func (r *pricingTestRepo) CreateOrgMember(_ context.Context, _ CreateOrgMemberParams, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *pricingTestRepo) ListFleetDrivers(_ context.Context, _ string) ([]SupplierFleetDriver, error) {
	return nil, nil
}

func (r *pricingTestRepo) CreateFleetDriver(_ context.Context, _ CreateFleetDriverParams, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *pricingTestRepo) ListFleetVehicles(_ context.Context, _ string) ([]SupplierFleetVehicle, error) {
	return nil, nil
}

func (r *pricingTestRepo) CreateFleetVehicle(_ context.Context, _ CreateFleetVehicleParams, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *pricingTestRepo) GetPricingRule(_ context.Context, _ string) (SupplierPricingRule, bool, error) {
	if !r.ruleFound {
		return SupplierPricingRule{}, false, nil
	}
	return r.rule, true, nil
}

func (r *pricingTestRepo) UpsertPricingRule(_ context.Context, rule SupplierPricingRule, _ func(outbox.TxnBuffer) error) error {
	if r.ruleFound {
		rule.RuleVersion = r.rule.RuleVersion + 1
	} else if rule.RuleVersion <= 0 {
		rule.RuleVersion = 1
	}
	r.rule = rule
	r.ruleFound = true
	return nil
}

func (r *pricingTestRepo) CountSuppliers(context.Context) (int64, error) {
	return 1, nil
}

type supplierOrdersTestRepo struct {
	pricingTestRepo
	orders []SupplierOrder
}

func (r *supplierOrdersTestRepo) ListOrders(_ context.Context, _ string, _, _ int) ([]SupplierOrder, error) {
	orders := make([]SupplierOrder, len(r.orders))
	copy(orders, r.orders)
	return orders, nil
}

func (r *supplierOrdersTestRepo) CountOrders(_ context.Context, _ string) (int, error) {
	return len(r.orders), nil
}

type supplierTestLocationReader struct {
	locations map[string]telemetry.DriverLocation
}

func (r *supplierTestLocationReader) GetDriverLocation(_ context.Context, driverID string) (telemetry.DriverLocation, bool, error) {
	location, found := r.locations[driverID]
	return location, found, nil
}

func TestHandlePricingRulesGetDefaultsWhenMissing(t *testing.T) {
	repo := &pricingTestRepo{profileFound: true, profile: Profile{SupplierID: "sup-1", Currency: "USD"}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Country: "UZ", Currency: "UZS"})

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/pricing/rules", nil)
	rr := httptest.NewRecorder()

	svc.HandlePricingRules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload supplierPricingRuleResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Currency != "USD" {
		t.Fatalf("currency=%q want=%q", payload.Currency, "USD")
	}
	if payload.RuleVersion != 0 {
		t.Fatalf("rule_version=%d want=0", payload.RuleVersion)
	}
}

func TestHandlePricingRulesPatchPersistsRule(t *testing.T) {
	repo := &pricingTestRepo{profileFound: true, profile: Profile{SupplierID: "sup-1", Currency: "UZS"}}
	now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Country:    "UZ",
		Currency:   "UZS",
		Now: func() time.Time {
			return now
		},
	})

	body := `{"base_markup_bps":1200,"retailer_discount_bps":200,"min_margin_bps":100,"currency":"uzs"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/pricing/rules", strings.NewReader(body))
	rr := httptest.NewRecorder()

	svc.HandlePricingRules(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload supplierPricingRuleResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.BaseMarkupBps != 1200 || payload.RetailerDiscountBps != 200 || payload.MinMarginBps != 100 {
		t.Fatalf("unexpected pricing payload: %+v", payload)
	}
	if payload.Currency != "UZS" {
		t.Fatalf("currency=%q want=%q", payload.Currency, "UZS")
	}
	if payload.RuleVersion != 1 {
		t.Fatalf("rule_version=%d want=%d", payload.RuleVersion, 1)
	}
}

func TestHandlePricingRulesPatchRejectsDiscountAboveMarkup(t *testing.T) {
	repo := &pricingTestRepo{profileFound: true, profile: Profile{SupplierID: "sup-1", Currency: "UZS"}}
	svc := NewService(ServiceConfig{Repo: repo, SupplierID: "sup-1", Country: "UZ", Currency: "UZS"})

	body := `{"base_markup_bps":100,"retailer_discount_bps":200}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/pricing/rules", strings.NewReader(body))
	rr := httptest.NewRecorder()

	svc.HandlePricingRules(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["error"] != "retailer_discount_exceeds_base_markup" {
		t.Fatalf("error=%q want=%q", payload["error"], "retailer_discount_exceeds_base_markup")
	}
}

func TestHandleOrdersUsesDurableAssignmentsAndLiveLocation(t *testing.T) {
	now := time.Date(2026, 5, 23, 11, 30, 0, 0, time.UTC)
	repo := &supplierOrdersTestRepo{
		orders: []SupplierOrder{{
			OrderID:        "ord-1",
			SupplierID:     "sup-1",
			RetailerID:     "ret-1",
			WarehouseID:    "wh-1",
			DriverID:       "drv-1",
			VehicleID:      "veh-1",
			RouteID:        "route-1",
			ManifestID:     "manifest-1",
			Status:         "IN_TRANSIT",
			TrackingStatus: "assigned",
			TotalMinor:     4200,
			Currency:       "UZS",
			CreatedAt:      now.Add(-15 * time.Minute).Format(time.RFC3339Nano),
			UpdatedAt:      now.Add(-2 * time.Minute).Format(time.RFC3339Nano),
		}},
	}
	locations := &supplierTestLocationReader{locations: map[string]telemetry.DriverLocation{
		"drv-1": {
			DriverID:          "drv-1",
			SupplierID:        "sup-1",
			Lat:               41.31,
			Lng:               69.28,
			Latitude:          41.31,
			Longitude:         69.28,
			ReportedAt:        now.Add(-20 * time.Second),
			ReceivedAt:        now.Add(-10 * time.Second),
			StaleAfterSeconds: 30,
		},
	}}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		Locations:  locations,
		SupplierID: "sup-1",
		Country:    "UZ",
		Currency:   "UZS",
		Now: func() time.Time {
			return now
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/orders", nil)
	rr := httptest.NewRecorder()

	svc.HandleOrders(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload struct {
		Orders []SupplierOrder `json:"orders"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Orders) != 1 {
		t.Fatalf("orders=%d want=1", len(payload.Orders))
	}
	order := payload.Orders[0]
	if !order.LiveLocationAvailable || order.DriverLocation == nil {
		t.Fatalf("expected live location on durable order: %+v", order)
	}
	if order.DriverLocation.DriverID != "drv-1" || order.RouteID != "route-1" || order.ManifestID != "manifest-1" {
		t.Fatalf("unexpected durable order payload: %+v", order)
	}
}
