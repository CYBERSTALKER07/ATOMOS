package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestFxApp_ValidateGraph(t *testing.T) {
	if err := fx.ValidateApp(AppOptions(slog.Default())...); err != nil {
		t.Fatalf("fx.ValidateApp failed: %v", err)
	}
}

type fxTestConsumer struct {
	fx.In

	Router           chi.Router
	CacheBackend     cache.Backend
	RetailerHub      *ws.Hub   `name:"retailer_hub"`
	PlatformAdminHub *ws.Hub   `name:"platform_admin_hub"`
	AllHubs          []*ws.Hub `group:"ws_hubs"`
}

func TestFxApp_ModularLifecycleWithFxTest(t *testing.T) {
	log := slog.Default()
	memBackend := cache.NewInMemoryBackend()
	hubGraph := bootstrap.ProvideHubGraph(memBackend, log)

	cfg := &bootstrap.Config{
		HTTPPort:       "0",
		WorkerHTTPPort: "0",
		RunMode:        bootstrap.RunModeAPI,
		JWTSecret:      "fx-test-jwt-secret-key-minimum-32-bytes",
		JWTIssuer:      "pegasusx-test",
	}

	stubApp := &bootstrap.App{
		Config:           cfg,
		Cache:            cache.New(memBackend, log),
		RetailerHub:      hubGraph.RetailerHub,
		SupplierHub:      hubGraph.SupplierHub,
		DriverHub:        hubGraph.DriverHub,
		PayloadHub:       hubGraph.PayloadHub,
		WarehouseHub:     hubGraph.WarehouseHub,
		FactoryHub:       hubGraph.FactoryHub,
		TelemetryHub:     hubGraph.TelemetryHub,
		PlatformAdminHub: hubGraph.PlatformAdminHub,
	}

	var populated fxTestConsumer

	app := fxtest.New(
		t,
		fx.Supply(log, cfg, stubApp),
		fx.Provide(
			bootstrap.ProvideAppSubgraph,
			provideCacheBackend,
		),
		HTTPModule,
		fx.Populate(&populated),
	)

	app.RequireStart()
	defer app.RequireStop()

	if populated.Router == nil {
		t.Fatal("expected chi.Router to be populated by fx")
	}
	if populated.CacheBackend == nil {
		t.Fatal("expected cache.Backend to be populated by fx")
	}
	if populated.RetailerHub == nil || populated.PlatformAdminHub == nil {
		t.Fatal("expected named *ws.Hub instances to be populated by fx")
	}
	if len(populated.AllHubs) != 8 {
		t.Fatalf("expected 8 hubs in ws_hubs value group, got %d", len(populated.AllHubs))
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/health/capabilities", nil)
	rec := httptest.NewRecorder()
	populated.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from /v1/health/capabilities, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode capabilities JSON: %v", err)
	}
	if payload["run_mode"] != bootstrap.RunModeAPI {
		t.Fatalf("expected run_mode=%q, got %v", bootstrap.RunModeAPI, payload["run_mode"])
	}
	if payload["api"] != true {
		t.Fatalf("expected api=true, got %v", payload["api"])
	}
}
