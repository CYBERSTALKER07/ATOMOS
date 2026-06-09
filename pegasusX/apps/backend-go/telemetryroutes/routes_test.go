package telemetryroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/retailer"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type telemetryTestConnection struct {
	id     string
	claims auth.Claims
	sent   chan []byte
}

func newTelemetryTestConnection(id string) *telemetryTestConnection {
	return &telemetryTestConnection{id: id, sent: make(chan []byte, 4)}
}

func (c *telemetryTestConnection) ID() string { return c.id }

func (c *telemetryTestConnection) Identity() auth.Claims { return c.claims }

func (c *telemetryTestConnection) Send(_ context.Context, payload []byte) error {
	if len(payload) == 0 {
		return errors.New("empty payload")
	}
	c.sent <- append([]byte(nil), payload...)
	return nil
}

type telemetryTestLocationWriter struct {
	saved   telemetry.DriverLocation
	saveErr error
}

func (w *telemetryTestLocationWriter) SaveDriverLocation(_ context.Context, location telemetry.DriverLocation) error {
	if w.saveErr != nil {
		return w.saveErr
	}
	w.saved = location
	return nil
}

type telemetryTrackingRepo struct {
	orders []retailer.TrackingOrder
}

func (r *telemetryTrackingRepo) CreateRetailer(_ context.Context, _ retailer.Retailer, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *telemetryTrackingRepo) FindByPhone(_ context.Context, _ string) (retailer.Retailer, bool, error) {
	return retailer.Retailer{}, false, nil
}

func (r *telemetryTrackingRepo) GetRetailer(_ context.Context, _ string) (retailer.Retailer, bool, error) {
	return retailer.Retailer{}, false, nil
}

func (r *telemetryTrackingRepo) UpdateRetailer(_ context.Context, _ retailer.Retailer, _ func(outbox.TxnBuffer) error) error {
	return nil
}

func (r *telemetryTrackingRepo) ListRetailersBySupplier(_ context.Context, _ string) ([]retailer.Retailer, error) {
	return nil, nil
}

func (r *telemetryTrackingRepo) GetSupplierPricingRule(_ context.Context, _ string) (retailer.SupplierPricingRule, bool, error) {
	return retailer.SupplierPricingRule{}, false, nil
}

func (r *telemetryTrackingRepo) ListTrackingOrders(_ context.Context, _ string, _ int) ([]retailer.TrackingOrder, error) {
	return r.orders, nil
}

func (r *telemetryTrackingRepo) ListRecentReceipts(_ context.Context, _ string, _ int) ([]retailer.TrackingOrder, error) {
	return []retailer.TrackingOrder{}, nil
}

func TestLocationRequiresDriverClaims(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	RegisterRoutes(router, Deps{TelemetryHub: ws.NewHub("telemetry", nil, nil), SupplierID: "sup-1"})

	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(`{"lat":41.3,"lng":69.2}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLocationRejectsInvalidCoordinates(t *testing.T) {
	t.Parallel()

	router := telemetryRouter(auth.Claims{Role: auth.RoleDriver, Subject: "drv-1", SupplierID: "sup-1"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(`{"lat":141.3,"lng":69.2}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLocationBroadcastsClaimsScopedEnvelope(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub("telemetry", nil, nil)
	driverConn := newTelemetryTestConnection("driver")
	supplierConn := newTelemetryTestConnection("supplier")
	hub.Subscribe("telemetry:driver:drv-1", driverConn)
	hub.Subscribe("telemetry:supplier:sup-1", supplierConn)

	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1", SupplierID: "sup-1"}
	router := telemetryRouter(claims, hub)
	body := `{"driver_id":"spoofed-driver","lat":41.3,"lng":69.2,"timestamp":"2026-05-22T10:11:12Z","velocity":14.5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(body))
	req = req.WithContext(outbox.WithTraceID(req.Context(), "trace-location-1"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	assertLocationFrame(t, driverConn.sent)
	assertLocationFrame(t, supplierConn.sent)
}

func TestLocationStoresClaimsScopedLastLocation(t *testing.T) {
	t.Parallel()

	store := &telemetryTestLocationWriter{}
	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1", SupplierID: "sup-1"}
	router := telemetryRouterWithStore(claims, ws.NewHub("telemetry", nil, nil), store)
	body := `{"driver_id":"spoofed-driver","lat":41.3,"lng":69.2,"timestamp":"2026-05-22T10:11:12Z","velocity":14.5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if store.saved.DriverID != "drv-1" || store.saved.SupplierID != "sup-1" {
		t.Fatalf("saved identity = %+v, want claims-derived drv-1/sup-1", store.saved)
	}
	if store.saved.Lat != 41.3 || store.saved.Lng != 69.2 {
		t.Fatalf("saved coordinate = %.1f/%.1f", store.saved.Lat, store.saved.Lng)
	}
	if store.saved.StaleAfterSeconds != locationStaleAfterSec {
		t.Fatalf("stale_after_seconds = %d, want %d", store.saved.StaleAfterSeconds, locationStaleAfterSec)
	}
}

func TestLocationSaveFailureStillBroadcastsAndAccepts(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub("telemetry", nil, nil)
	driverConn := newTelemetryTestConnection("driver")
	hub.Subscribe("telemetry:driver:drv-1", driverConn)

	store := &telemetryTestLocationWriter{saveErr: errors.New("cache unavailable")}
	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1", SupplierID: "sup-1"}
	router := telemetryRouterWithStore(claims, hub, store)
	body := `{"lat":41.3,"lng":69.2,"timestamp":"2026-05-22T10:11:12Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(body))
	req = req.WithContext(outbox.WithTraceID(req.Context(), "trace-location-1"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	assertLocationFrame(t, driverConn.sent)
}

func TestLocationIngressToRetailerTrackingFreshnessUnderReconnectChurn(t *testing.T) {
	t.Parallel()

	cacheClient := cache.New(cache.NewInMemoryBackend(), nil)
	locationStore := telemetry.NewCacheLastLocationStore(cacheClient, telemetry.DefaultLastLocationTTL)
	hub := ws.NewHub("telemetry", nil, nil)
	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1", SupplierID: "sup-1"}
	router := telemetryRouterWithStore(claims, hub, locationStore)
	body := `{"lat":41.3,"lng":69.2,"timestamp":"2026-05-22T10:11:12Z"}`

	for i := 0; i < 6; i++ {
		driverConn := newTelemetryTestConnection("driver-" + strconv.Itoa(i))
		supplierConn := newTelemetryTestConnection("supplier-" + strconv.Itoa(i))
		unsubscribeDriver := hub.Subscribe("telemetry:driver:drv-1", driverConn)
		unsubscribeSupplier := hub.Subscribe("telemetry:supplier:sup-1", supplierConn)

		req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(body))
		req = req.WithContext(outbox.WithTraceID(req.Context(), "trace-location-1"))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
		}
		assertLocationFrame(t, driverConn.sent)
		assertLocationFrame(t, supplierConn.sent)

		unsubscribeDriver()
		unsubscribeSupplier()
	}

	saved, found, err := locationStore.GetDriverLocation(context.Background(), "drv-1")
	if err != nil {
		t.Fatalf("read cached location: %v", err)
	}
	if !found {
		t.Fatal("expected cached location after telemetry ingress")
	}

	repo := &telemetryTrackingRepo{orders: []retailer.TrackingOrder{{
		OrderID:        "ord-1",
		SupplierID:     "sup-1",
		RetailerID:     "ret-1",
		DriverID:       "drv-1",
		RouteID:        "route-1",
		Status:         "IN_TRANSIT",
		TrackingStatus: "assigned",
		TotalMinor:     1500,
		Currency:       "UZS",
		Items:          []retailer.TrackingLineItem{},
	}}}

	freshService := retailer.NewService(retailer.ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Locations:  locationStore,
		Now: func() time.Time {
			return saved.ReceivedAt.Add(time.Second)
		},
	})
	freshReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	freshReq = freshReq.WithContext(auth.WithClaims(freshReq.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	freshRec := httptest.NewRecorder()
	freshService.HandleTracking(freshRec, freshReq)
	if freshRec.Code != http.StatusOK {
		t.Fatalf("fresh status = %d, want %d body=%s", freshRec.Code, http.StatusOK, freshRec.Body.String())
	}
	var freshPayload map[string]any
	if err := json.Unmarshal(freshRec.Body.Bytes(), &freshPayload); err != nil {
		t.Fatalf("decode fresh payload: %v", err)
	}
	freshOrders := freshPayload["orders"].([]any)
	freshFirst := freshOrders[0].(map[string]any)
	if freshFirst["live_location_available"] != true {
		t.Fatalf("fresh live_location_available=%v want true", freshFirst["live_location_available"])
	}
	if _, ok := freshFirst["driver_latitude"]; !ok || freshFirst["driver_latitude"] == nil {
		t.Fatalf("fresh driver_latitude missing: %v", freshFirst)
	}

	staleService := retailer.NewService(retailer.ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Locations:  locationStore,
		Now: func() time.Time {
			return saved.ReceivedAt.Add(time.Duration(saved.StaleAfterSeconds+1) * time.Second)
		},
	})
	staleReq := httptest.NewRequest(http.MethodGet, "/v1/retailer/tracking", nil)
	staleReq = staleReq.WithContext(auth.WithClaims(staleReq.Context(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}))
	staleRec := httptest.NewRecorder()
	staleService.HandleTracking(staleRec, staleReq)
	if staleRec.Code != http.StatusOK {
		t.Fatalf("stale status = %d, want %d body=%s", staleRec.Code, http.StatusOK, staleRec.Body.String())
	}
	var stalePayload map[string]any
	if err := json.Unmarshal(staleRec.Body.Bytes(), &stalePayload); err != nil {
		t.Fatalf("decode stale payload: %v", err)
	}
	staleOrders := stalePayload["orders"].([]any)
	staleFirst := staleOrders[0].(map[string]any)
	if staleFirst["live_location_available"] != false {
		t.Fatalf("stale live_location_available=%v want false", staleFirst["live_location_available"])
	}
	if staleFirst["driver_latitude"] != nil {
		t.Fatalf("stale driver_latitude leaked: %v", staleFirst["driver_latitude"])
	}
}

func TestLocationAcceptsSessionAuthBearerJWT(t *testing.T) {
	t.Parallel()

	const secret = "dev-only-change-me"
	hub := ws.NewHub("telemetry", nil, nil)
	router := chi.NewRouter()
	router.Use(auth.SessionAuth(secret))
	RegisterRoutes(router, Deps{TelemetryHub: hub, SupplierID: "sup-1"})

	token, err := auth.Issue(auth.Claims{
		Subject:      "ssmr-driver-1",
		Role:         auth.RoleDriver,
		SupplierID:   "sup-1",
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "ssmr-warehouse-1",
	}, auth.IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry/location", bytes.NewBufferString(`{"lat":41.3,"lng":69.2}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d body %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
}

func telemetryRouter(claims auth.Claims, hub *ws.Hub) *chi.Mux {
	return telemetryRouterWithStore(claims, hub, nil)
}

func telemetryRouterWithStore(claims auth.Claims, hub *ws.Hub, store telemetry.LastLocationWriter) *chi.Mux {
	if hub == nil {
		hub = ws.NewHub("telemetry", nil, nil)
	}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	})
	RegisterRoutes(router, Deps{TelemetryHub: hub, LastLocations: store, SupplierID: "sup-1"})
	return router
}

func assertLocationFrame(t *testing.T, ch <-chan []byte) {
	t.Helper()
	select {
	case raw := <-ch:
		var envelope locationEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("unmarshal frame: %v", err)
		}
		if envelope.Type != events.EventDriverLocationUpdated {
			t.Fatalf("type = %q, want %q", envelope.Type, events.EventDriverLocationUpdated)
		}
		if envelope.TraceID != "trace-location-1" {
			t.Fatalf("trace_id = %q, want trace-location-1", envelope.TraceID)
		}
		if envelope.Data.DriverID != "drv-1" || envelope.Data.SupplierID != "sup-1" {
			t.Fatalf("identity = %+v, want claims-derived drv-1/sup-1", envelope.Data)
		}
		if envelope.Data.Lat != 41.3 || envelope.Data.Lng != 69.2 {
			t.Fatalf("coordinate = %.1f/%.1f", envelope.Data.Lat, envelope.Data.Lng)
		}
		if envelope.Data.ReportedAt != "2026-05-22T10:11:12Z" {
			t.Fatalf("reported_at = %q", envelope.Data.ReportedAt)
		}
		if envelope.Data.StaleAfterSeconds != locationStaleAfterSec {
			t.Fatalf("stale_after_seconds = %d", envelope.Data.StaleAfterSeconds)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for telemetry frame")
	}
}
