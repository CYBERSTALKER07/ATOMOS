package orderroutes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func orderRouter(svc *order.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(auth.SessionAuth("order-test-secret"))
	RegisterRoutes(r, Deps{Service: svc})
	return r
}

func issueOrderToken(t *testing.T, role auth.Role) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:    "order-user",
		Role:       role,
		SupplierID: "sup-1",
	}, auth.IssueOptions{Secret: "order-test-secret", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestOrderRoutes_NilService(t *testing.T) {
	r := orderRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/platform/receipts/rcpt-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered routes, got %d", rr.Code)
	}
}

func TestOrderRoutes_Unauthenticated(t *testing.T) {
	svc := &order.Service{}
	r := orderRouter(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v1/order/create"},
		{http.MethodGet, "/v1/order/currencies"},
		{http.MethodPost, "/v1/delivery/arrive"},
		{http.MethodPost, "/v1/supplier/shop-closed/resolve"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without auth returned %d, want 401", ep.method, ep.path, rr.Code)
		}
	}
}

func TestOrderRoutes_RoleGuard(t *testing.T) {
	svc := &order.Service{}
	r := orderRouter(svc)

	driverToken := issueOrderToken(t, auth.RoleDriver)

	// Driver cannot create orders (Retailer required)
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+driverToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("driver creating order returned %d, want 403 Forbidden", rr.Code)
	}

	retailerToken := issueOrderToken(t, auth.RoleRetailer)

	// Retailer cannot arrive at delivery (Driver required)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/delivery/arrive", strings.NewReader(`{}`))
	req2.Header.Set("Authorization", "Bearer "+retailerToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusForbidden {
		t.Fatalf("retailer arriving delivery returned %d, want 403 Forbidden", rr2.Code)
	}
}

func TestOrderRoutes_HandlerExecutionAndValidation(t *testing.T) {
	svc := &order.Service{}
	r := orderRouter(svc)
	retailerToken := issueOrderToken(t, auth.RoleRetailer)

	// GET /v1/order/currencies executes handler and returns 200 OK with supported currencies
	req := httptest.NewRequest(http.MethodGet, "/v1/order/currencies", nil)
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /v1/order/currencies returned %d, want 200 OK", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "UZS") {
		t.Fatalf("GET /v1/order/currencies body missing UZS: %s", rr.Body.String())
	}

	// POST /v1/order/create with malformed JSON returns 400 Bad Request from handler
	req2 := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(`{bad-json`))
	req2.Header.Set("Authorization", "Bearer "+retailerToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/order/create with malformed JSON returned %d, want 400", rr2.Code)
	}
}

type routeTestRepo struct {
	order.Repository
	createCalls int
	created     *order.Order
}

func (r *routeTestRepo) CreateOrder(_ context.Context, o *order.Order, _ func(outbox.TxnBuffer) error, _ order.StockReservationOpts) error {
	r.createCalls++
	r.created = o
	return nil
}

func (r *routeTestRepo) CreateOrderWithBackorder(_ context.Context, o *order.Order, _ *order.Order, _ func(context.Context, *spanner.ReadWriteTransaction) error, _ func(outbox.TxnBuffer) error, _ order.StockReservationOpts) error {
	r.createCalls++
	r.created = o
	return nil
}

type routeWarehouseResolver struct{}

func (routeWarehouseResolver) ResolveNearestWarehouseID(context.Context, string, proximity.StorePoint) (string, error) {
	return "wh-1", nil
}

func TestOrderRoutes_CreatePackValidationSubstitutionAndOptIn(t *testing.T) {
	repo := &routeTestRepo{}
	svc := order.NewService(order.ServiceConfig{
		Repo:       repo,
		Warehouse:  routeWarehouseResolver{},
		SupplierID: "sup-1",
		Currency:   "UZS",
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	svc.SetPackRuleReader(func(context.Context, []string) (map[string]order.PackRule, error) {
		return map[string]order.PackRule{
			"SKU-10": {SaleUnit: "CASE", UnitsPerPack: 10},
		}, nil
	})
	svc.SetShortageStockReader(func(context.Context, *spanner.ReadWriteTransaction, string, string, []order.LineItem) (order.ShortageContext, error) {
		return order.ShortageContext{
			AvailableBySKU: map[string]int64{
				"SKU-10":     2,
				"SKU-10-SUB": 50,
			},
			CandidatesBySKU: map[string][]order.SubstituteCandidate{
				"SKU-10": {{
					SKU:          "SKU-10-SUB",
					SupplierID:   "sup-1",
					BrandID:      "BRAND-A",
					Name:         "Sub 10-Pack",
					AvailableQty: 50,
					PriceMinor:   2100,
				}},
			},
			EnforceAllowlist: true,
		}, nil
	})

	r := orderRouter(svc)
	retailerToken := issueOrderToken(t, auth.RoleRetailer)

	// 1. Non-pack-multiple returns 422 PACK_MULTIPLE_REQUIRED with structured suggestions.
	badPayload := `{"line_items":[{"sku":"SKU-10","quantity":15,"unit_price_minor":2000}],"h3_cell":"872830828ffffff","lat":41.3,"lng":69.2}`
	req := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(badPayload))
	req.Header.Set("Authorization", "Bearer "+retailerToken)
	req.Header.Set("Idempotency-Key", "routes-pack-422-1")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST /v1/order/create non-pack returned %d, want 422 (%s)", rr.Code, rr.Body.String())
	}
	var errResp struct {
		Code        string                     `json:"code"`
		Suggestions []order.PackLineSuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode 422 JSON: %v", err)
	}
	if errResp.Code != "PACK_MULTIPLE_REQUIRED" || len(errResp.Suggestions) != 1 ||
		errResp.Suggestions[0].SKU != "SKU-10" || errResp.Suggestions[0].PackSize != 10 ||
		errResp.Suggestions[0].SuggestedFloor != 10 || errResp.Suggestions[0].SuggestedCeil != 20 ||
		errResp.Suggestions[0].SaleUnit != "CASE" {
		t.Fatalf("unexpected PACK_MULTIPLE_REQUIRED response: %+v", errResp)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected 0 createCalls on invalid pack, got %d", repo.createCalls)
	}

	// 2. Valid pack multiple substitutes within same brand & exposes disclosure + auto_confirm_opt_in.
	goodPayload := `{
		"line_items": [{
			"sku": "SKU-10",
			"brand_id": "BRAND-A",
			"quantity": 10,
			"unit_price_minor": 2000,
			"must_have": true,
			"allow_substitutes": true,
			"allowed_substitute_skus": ["SKU-10-SUB"]
		}],
		"h3_cell": "872830828ffffff",
		"lat": 41.3,
		"lng": 69.2,
		"auto_confirm_opt_in": true
	}`
	req2 := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(goodPayload))
	req2.Header.Set("Authorization", "Bearer "+retailerToken)
	req2.Header.Set("Idempotency-Key", "routes-pack-201-2")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("POST /v1/order/create valid returned %d, want 201 (%s)", rr2.Code, rr2.Body.String())
	}
	var created order.CreateResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode 201 JSON: %v", err)
	}
	if !created.AutoConfirmOptIn || created.TotalMinor != 21000 ||
		len(created.Substitutions) != 1 || created.Substitutions[0].PriceGapMinor != 1000 ||
		created.Substitutions[0].SubstituteSKU != "SKU-10-SUB" || !created.LineItems[0].MustHave {
		t.Fatalf("unexpected 201 create response: %+v", created)
	}

	// 3. Fail-closed metadata error returns 503 Service Unavailable.
	svc.SetPackRuleReader(func(context.Context, []string) (map[string]order.PackRule, error) {
		return nil, errors.New("spanner connection reset")
	})
	req3 := httptest.NewRequest(http.MethodPost, "/v1/order/create", strings.NewReader(goodPayload))
	req3.Header.Set("Authorization", "Bearer "+retailerToken)
	req3.Header.Set("Idempotency-Key", "routes-pack-503-3")
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /v1/order/create metadata error returned %d, want 503 (%s)", rr3.Code, rr3.Body.String())
	}
}
