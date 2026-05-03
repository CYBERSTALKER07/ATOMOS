package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"backend-go/admin"
	"backend-go/adminroutes"
	"backend-go/airoutes"
	"backend-go/analytics"
	"backend-go/auth"
	"backend-go/authroutes"
	"backend-go/bootstrap"
	"backend-go/cache"
	"backend-go/catalogroutes"
	"backend-go/countrycfg"
	"backend-go/deliveryroutes"
	"backend-go/driverroutes"
	"backend-go/factory"
	"backend-go/factoryroutes"
	"backend-go/fleet"
	"backend-go/fleetroutes"
	"backend-go/idempotency"
	"backend-go/notifications"
	"backend-go/order"
	"backend-go/payloaderroutes"
	"backend-go/payment"
	"backend-go/paymentroutes"
	"backend-go/proximity"
	"backend-go/replenishment"
	"backend-go/routing"
	"backend-go/settings"
	"backend-go/simulation"
	"backend-go/supplier"
	"backend-go/sync"
	"backend-go/telemetry"
	"backend-go/treasury"
	"backend-go/userroutes"
	"backend-go/vault"
	"backend-go/warehouse"
	"backend-go/warehouseroutes"
	"backend-go/webhookroutes"
	"backend-go/ws"
	"config"

	internalKafka "backend-go/kafka"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"
)

type DeliverySubmitRequest struct {
	OrderID   string  `json:"order_id"`
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Observability Middleware
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		analytics.IncrementRequest()
		defer analytics.DecrementRequest()
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("[HTTP] %s %s | Duration: %v\n", r.Method, r.URL.Path, duration)
	}
}

// parseCORSAllowlist builds the CORS origin set from CORS_ALLOWED_ORIGINS env
// var (comma-separated). Falls back to localhost dev defaults when unset.
func parseCORSAllowlist() map[string]bool {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		// Dev-mode defaults — overridden in production via env var
		return map[string]bool{
			"http://localhost:3000":  true,
			"http://localhost:3001":  true,
			"http://localhost:3002":  true,
			"http://localhost:8081":  true,
			"http://localhost:19006": true,
		}
	}
	allowed := make(map[string]bool)
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = true
		}
	}
	return allowed
}

var corsAllowlist = parseCORSAllowlist()

// CORS Middleware Global Wrap
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && corsAllowlist[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin != "" && (strings.HasSuffix(origin, ".ngrok-free.app") || strings.HasSuffix(origin, ".expo.dev") || strings.HasPrefix(origin, "http://192.168.") || strings.HasPrefix(origin, "http://10.0.")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// Same-origin or non-browser clients (mobile apps)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Internal-Key, X-Trace-Id")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	log.Println("Booting up Pegasus - Backend API...")

	// 1. Load config + fail-closed auth.
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Fatal config load error: %v", err)
	}
	auth.Init(cfg.JWTSecret, cfg.InternalAPIKey)

	ctx := context.Background()

	// 2. Composition root: all clients, services, hubs, middleware.
	app, err := bootstrap.NewApp(ctx, cfg)
	if err != nil {
		log.Fatalf("Fatal bootstrap error: %v", err)
	}
	defer app.Close()

	// 3. Aliases preserve the legacy variable names used throughout the 279
	// route registrations below. Route-by-route migration into domain
	// packages (Phase 5) progressively removes these.
	spannerClient := app.Spanner
	svc := app.Order
	vaultSvc := app.Vault
	sessionSvc := app.SessionSvc
	cardTokenSvc := app.CardTokenSvc
	cardsClient := app.CardsClient
	directClient := app.DirectClient
	countryCfgSvc := app.CountryConfig
	supplierPricingSvc := app.SupplierPricing
	retailerHub := app.RetailerHub
	driverHub := app.DriverHub
	payloaderHub := app.PayloaderHub
	warehouseHub := app.WarehouseHub
	priorityGuard := app.PriorityGuard
	shopClosedDeps := app.ShopClosedDeps
	earlyCompleteDeps := app.EarlyCompleteDeps
	negotiationDeps := app.NegotiationDeps

	// Silence unused warnings on aliases that are only consumed inside
	// inline closures; the Go compiler treats captures as usage, but
	// several of these may be unused depending on which routes remain in
	// main versus migrated — keeping them explicit for the session.
	_ = vaultSvc
	_ = sessionSvc
	_ = cardTokenSvc
	_ = cardsClient
	_ = directClient
	_ = countryCfgSvc
	_ = supplierPricingSvc
	_ = warehouseHub
	_ = payloaderHub

	// 4. Chi router — mechanical swap for http.DefaultServeMux. Domain
	// subrouters (see Phase 5) will mount under r via r.Mount(prefix, sub).
	// Until every legacy http.HandleFunc call site is migrated, the default
	// mux is mounted at "/" so new and legacy registrations coexist.
	r := chi.NewRouter()
	r.Use(bootstrap.TraceMiddleware) // Glass Box: trace_id on every request
	r.Mount("/", http.DefaultServeMux)

	// Spanner admin surface: the inline DDL migrations below build their
	// own DatabaseAdminClient using the same dial options and database
	// URI that bootstrap used for the data-plane client.
	opts := app.SpannerClientOpts
	dbName := app.SpannerDBName

	// 5. Seed the admin account (idempotent) before any routes serve traffic.
	admin.StartReconciliationCron(ctx, spannerClient)

	// Desert Protocol — offline batch + reconnection catch-up.
	// Ownership lives in backend-go/sync/routes.go.
	sync.RegisterRoutes(r, spannerClient, loggingMiddleware)

	// Telemetry Matrix — uses grace period auth to allow expired driver tokens for 2h (A-4)
	http.HandleFunc("/ws/telemetry", auth.RequireRoleWithGrace([]string{"DRIVER", "ADMIN", "SUPPLIER"}, 2*time.Hour, telemetry.FleetHub.HandleConnection))
	http.HandleFunc("/ws/fleet", auth.RequireRoleWithGrace([]string{"DRIVER", "ADMIN", "SUPPLIER"}, 2*time.Hour, telemetry.FleetHub.HandleConnection))

	// GET /v1/supplier/dashboard — Aggregates Orders and AIPredictions for the Productioner metrics
	http.HandleFunc("/v1/supplier/dashboard", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metrics, err := svc.GetSupplierMetrics(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})))

	// ── Vector G: B2B Dynamic Pricing Engine ──────────────────────────────────
	// (supplierPricingSvc is constructed in bootstrap and aliased above.)

	// POST /v1/supplier/products/upload-ticket - Generates Signed URL for GCS direct upload
	http.HandleFunc("/v1/supplier/products/upload-ticket",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}
			supplier.HandleGetUploadTicket(w, r)
		})))

	// GET+POST /v1/supplier/products - List or create supplier products
	http.HandleFunc("/v1/supplier/products",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supplier.HandleListSupplierProducts(spannerClient)(w, r)
			case http.MethodPost:
				supplier.HandleCreateProduct(spannerClient)(w, r)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		})))

	// GET/PUT/DELETE /v1/supplier/products/{sku_id} — Read, update, or deactivate a product
	http.HandleFunc("/v1/supplier/products/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supplier.HandleGetProduct(spannerClient)(w, r)
			case http.MethodPut:
				supplier.HandleUpdateProduct(spannerClient)(w, r)
			case http.MethodDelete:
				supplier.HandleDeactivateProduct(spannerClient)(w, r)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		})))

	// POST /v1/supplier/pricing/rules — Manufacturer locks a discount bracket.
	// Legacy admin portal surface; accepts SUPPLIER or ADMIN JWT.
	http.HandleFunc("/v1/supplier/pricing/rules",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplierPricingSvc.HandleUpsertPricingRule)))

	// DELETE /v1/supplier/pricing/rules/{tier_id} — Deactivate a pricing rule
	http.HandleFunc("/v1/supplier/pricing/rules/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplierPricingSvc.HandlePricingRuleAction)))

	// ── Per-Retailer Pricing Overrides ──────────────────────────────────────
	retailerPricingSvc := supplier.NewRetailerPricingService(spannerClient, app.SpannerRouter, svc.Producer, app.Cache)
	http.HandleFunc("/v1/supplier/pricing/retailer-overrides",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(retailerPricingSvc.HandleRetailerPricingOverrides))))
	http.HandleFunc("/v1/supplier/pricing/retailer-overrides/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(retailerPricingSvc.HandleRetailerPricingOverrideAction))))

	// ── LEO: LOGISTICS EXECUTION ORCHESTRATOR — Manifest Loading Gate ─────

	manifestSvc := &supplier.ManifestService{
		Spanner:       spannerClient,
		Cache:         app.Cache,
		MapsAPIKey:    cfg.GoogleMapsAPIKey,
		DepotLocation: cfg.DepotLocation,
	}

	// GET /v1/supplier/manifests — List manifests (filterable by ?state=DRAFT|LOADING|SEALED|...)
	// GET /v1/supplier/manifests/{id} — Manifest detail with orders
	http.HandleFunc("/v1/supplier/manifests/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			// Determine if this is a detail or action route
			id := supplier.ExtractPathParam(r.URL.Path, "manifests")
			switch {
			case id != "" && strings.HasSuffix(r.URL.Path, "/start-loading"):
				idempotency.Guard(manifestSvc.HandleStartLoading())(w, r)
			case id != "" && strings.HasSuffix(r.URL.Path, "/seal"):
				idempotency.Guard(manifestSvc.HandleSealManifest())(w, r)
			case id != "" && strings.HasSuffix(r.URL.Path, "/inject-order"):
				idempotency.Guard(manifestSvc.HandleInjectOrder())(w, r)
			case id != "":
				manifestSvc.HandleManifestDetail()(w, r)
			default:
				manifestSvc.HandleListManifests()(w, r)
			}
		})))

	// GET /v1/supplier/manifests — List manifests (root path, no trailing slash)
	http.HandleFunc("/v1/supplier/manifests",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			manifestSvc.HandleListManifests()(w, r)
		})))

	// /v1/driver/* — 7 routes (earnings, history, availability, pending-collections,
	// profile, manifest-gate, manifest). Ownership lives in backend-go/driverroutes.
	driverroutes.RegisterRoutes(r, driverroutes.Deps{
		Spanner:     spannerClient,
		Order:       svc,
		ManifestSvc: manifestSvc,
		Cache:       app.Cache,
		CacheFlight: app.CacheFlight,
		Log:         loggingMiddleware,
	})

	// POST /v1/payload/manifest-exception — Payloader removes order from manifest (OVERFLOW/DAMAGED/MANUAL)
	http.HandleFunc("/v1/payload/manifest-exception",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN", "PAYLOAD"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			idempotency.Guard(manifestSvc.HandleManifestException())(w, r)
		})))

	// GET /v1/supplier/manifest-exceptions — List exceptions (DLQ escalations with ?escalated=true)
	http.HandleFunc("/v1/supplier/manifest-exceptions",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			manifestSvc.HandleListExceptions()(w, r)
		})))

	// ── Phase VII: Delivery Zones CRUD ────────────────────────────────────
	// GET/POST /v1/supplier/delivery-zones — List or create delivery fee zones
	http.HandleFunc("/v1/supplier/delivery-zones/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(
			supplier.HandleDeliveryZoneAction(spannerClient))))
	http.HandleFunc("/v1/supplier/delivery-zones",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(
			supplier.HandleDeliveryZones(spannerClient))))

	// /v1/checkout/{b2b,unified} + /v1/payment/* moved to paymentroutes
	// (registered below after chargebackSvc is constructed).

	// /v1/ai/* — Empathy Engine preorder trigger + prediction feedback (3 routes).
	// Ownership lives in backend-go/airoutes/routes.go.
	airoutes.RegisterRoutes(r, airoutes.Deps{
		Spanner:  spannerClient,
		Preorder: svc,
		Log:      loggingMiddleware,
	})

	// /v1/auth/* — full login/register surface (14 routes).
	// Ownership lives in backend-go/authroutes/routes.go.
	authroutes.Register(r, authroutes.Deps{
		Spanner:        spannerClient,
		RetailerStatus: svc,
		Log:            loggingMiddleware,
		RateLimit:      cache.RateLimitMiddleware(cache.AuthRateLimit()),
		ActorRateLimit: cache.RateLimitMiddleware(cache.APIRateLimit()),
	})

	// /v1/user/* — device-token + notification inbox (3 routes).
	// Ownership lives in backend-go/userroutes/routes.go.
	deviceTokenSvc := &notifications.DeviceTokenService{Spanner: spannerClient}
	userroutes.RegisterRoutes(r, userroutes.Deps{
		Spanner:        spannerClient,
		DeviceTokenSvc: deviceTokenSvc,
		Log:            loggingMiddleware,
	})

	// /v1/treasury/* + /v1/supplier/settlement-report → treasury package.
	treasury.RegisterRoutes(r, treasury.Deps{Spanner: spannerClient, Log: loggingMiddleware})

	// /v1/admin/{reconciliation,audit-log,country-configs,country-configs/} moved to adminroutes.

	// GET/PUT /v1/supplier/country-overrides - Supplier-specific country config overrides
	// GET  list all overrides for this supplier (with effective merged values)
	// PUT  upsert an override for a specific country
	http.HandleFunc("/v1/supplier/country-overrides", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(countrycfg.HandleSupplierCountryOverrides(countryCfgSvc))))
	// GET single override + effective config / DELETE to revert to platform defaults
	http.HandleFunc("/v1/supplier/country-overrides/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(countrycfg.HandleSupplierCountryOverrideByCode(countryCfgSvc))))

	// GET /v1/supplier/analytics/velocity - Real-time sales data Oracle for Suppliers
	http.HandleFunc("/v1/supplier/analytics/velocity", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(analytics.HandleGetVelocity(spannerClient, app.SpannerRouter))))

	// GET /v1/supplier/analytics/demand/today — AI Future Demand (next 24h) summary
	http.HandleFunc("/v1/supplier/analytics/demand/today", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(analytics.HandleDemandToday(spannerClient, app.SpannerRouter))))

	// GET /v1/supplier/analytics/demand/history — Predicted vs Actual time-series + upcoming detail
	http.HandleFunc("/v1/supplier/analytics/demand/history", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(analytics.HandleDemandHistory(spannerClient, app.SpannerRouter))))

	// ── Intelligence Vector Analytics (Phase 6) ─────────────────────────────
	http.HandleFunc("/v1/supplier/analytics/transit-heatmap", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleTransitHeatmap(spannerClient, app.SpannerRouter)))))
	http.HandleFunc("/v1/supplier/analytics/throughput", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleThroughput(spannerClient, app.SpannerRouter)))))
	http.HandleFunc("/v1/supplier/analytics/load-distribution", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleLoadDistribution(spannerClient, app.SpannerRouter)))))
	http.HandleFunc("/v1/supplier/analytics/node-efficiency", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleNodeEfficiency(spannerClient, app.SpannerRouter)))))
	http.HandleFunc("/v1/supplier/analytics/sla-health", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleSLAHealth(spannerClient, app.SpannerRouter)))))

	// ── Advanced Revenue + CRM Analytics ──
	http.HandleFunc("/v1/supplier/analytics/revenue", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleRevenue(spannerClient, app.SpannerRouter)))))
	http.HandleFunc("/v1/supplier/analytics/top-retailers", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(analytics.HandleTopRetailers(spannerClient, app.SpannerRouter)))))

	// GET /v1/supplier/financials — Supplier-wide financials: revenue, fees, net payout, cash
	http.HandleFunc("/v1/supplier/financials", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(analytics.HandleSupplierFinancials(spannerClient, app.SpannerRouter))))

	// ── Retailer Expense Analytics (Phase 1: Insights Dashboard) ──
	http.HandleFunc("/v1/retailer/analytics/expenses", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(analytics.HandleGetRetailerExpenses(spannerClient, app.SpannerRouter))))

	// ── Retailer Detailed Analytics (Advanced Analytics) ──
	http.HandleFunc("/v1/retailer/analytics/detailed", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(analytics.HandleRetailerDetailedAnalytics(spannerClient, app.SpannerRouter))))

	// ── Supplier CRM: Retailer Intelligence ──
	http.HandleFunc("/v1/supplier/crm/retailers", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleCRMRetailers(spannerClient))))
	http.HandleFunc("/v1/supplier/crm/retailers/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleCRMRetailerDetail(spannerClient))))

	// ── Supplier Fleet Provisioning (Phase 10: Driver Badging) ──────────────
	http.HandleFunc("/v1/supplier/fleet/drivers", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleFleetDrivers(spannerClient)))))
	http.HandleFunc("/v1/supplier/fleet/drivers/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleFleetDriverDetail(spannerClient)))))

	// ── Supplier Fleet Vehicles (Phase 6: Volumetric Units) ─────────────────
	http.HandleFunc("/v1/supplier/fleet/vehicles", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleVehicles(spannerClient)))))
	http.HandleFunc("/v1/supplier/fleet/vehicles/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleVehicleDetail(spannerClient)))))

	// POST /v1/supplier/fulfillment/pay — Trigger per-supplier staggered payment
	// after driver arrival + order amendment.
	// Charges ONLY this supplier's adjusted line-item total (never the full multi-supplier cart).
	// Role: SUPPLIER or DRIVER (driver triggers at delivery, supplier can trigger from portal).
	http.HandleFunc("/v1/supplier/fulfillment/pay", auth.RequireRole([]string{"SUPPLIER", "DRIVER", "ADMIN"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
			return
		}
		result, err := svc.TriggerSupplierFulfillmentPayment(r.Context(), req.OrderID)
		if err != nil {
			errMsg := err.Error()
			switch {
			case strings.Contains(errMsg, "GEOFENCE_VIOLATION"):
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
			case strings.Contains(errMsg, "state conflict"):
				http.Error(w, errMsg, http.StatusConflict)
			case strings.Contains(errMsg, "not found"):
				http.Error(w, errMsg, http.StatusNotFound)
			case strings.Contains(errMsg, "credentials unavailable"):
				http.Error(w, errMsg, http.StatusServiceUnavailable)
			default:
				http.Error(w, errMsg, http.StatusBadGateway)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))))

	// ── Phase 5-6: System Health + Driver + Retailer + Supplier API Gap Closure ──

	// GET /v1/health — Load balancer health check (no auth required)
	http.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		// Quick Spanner ping
		row, err := spannerClient.Single().ReadRow(r.Context(), "Suppliers", spanner.Key{"health-check-probe"}, []string{"SupplierId"})
		spannerOK := err != nil || row != nil // NOT_FOUND is still healthy — means Spanner is reachable
		_ = row

		redisOK := cache.Client != nil
		if redisOK {
			if err := cache.Client.Ping(r.Context()).Err(); err != nil {
				redisOK = false
			}
		}

		status := "healthy"
		code := http.StatusOK
		if !spannerOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  status,
			"spanner": spannerOK,
			"redis":   redisOK,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// GET /metrics and /v1/metrics — Prometheus and legacy JSON process metrics.
	analytics.RegisterMetricsRoutes(http.DefaultServeMux, loggingMiddleware)

	// /v1/driver/{earnings,history,availability} moved to driverroutes.

	// /v1/fleet/drivers/{id}/status moved to fleetroutes.

	// /v1/driver/pending-collections moved to driverroutes.

	// ── v2.2 Edge Case Routes ───────────────────────────────────────────────

	// /v1/admin/orders/payment-bypass moved to adminroutes.

	// /v1/delivery/confirm-payment-bypass moved to deliveryroutes.

	// Edge 7: POST /v1/orders/request-cancel — Retailer requests cancellation
	http.HandleFunc("/v1/orders/request-cancel", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(order.HandleRequestCancel(svc))))

	// /v1/admin/orders/approve-cancel moved to adminroutes.

	// /v1/delivery/sms-complete moved to deliveryroutes.

	// /v1/delivery/shop-closed moved to deliveryroutes.

	// P0: POST /v1/retailer/shop-closed-response — Retailer responds to shop-closed alert
	http.HandleFunc("/v1/retailer/shop-closed-response", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(svc.HandleShopClosedResponse(&shopClosedDeps))))

	// /v1/admin/shop-closed/resolve moved to adminroutes.

	// /v1/delivery/bypass-offload moved to deliveryroutes.

	// Edge 24: Device fingerprinting (wired into login — see auth/device.go)

	// ── v3.1 Human-Centric Edge Routes ──────────────────────────────────────

	// /v1/fleet/route/request-early-complete moved to fleetroutes.

	// /v1/admin/route/approve-early-complete moved to adminroutes.

	// /v1/delivery/negotiate moved to deliveryroutes.

	// /v1/admin/negotiate/resolve moved to adminroutes.

	// Edge 29: GET /v1/retailer/family-members — List family sub-profiles
	http.HandleFunc("/v1/retailer/family-members", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			auth.HandleListFamilyMembers(spannerClient)(w, r)
		} else if r.Method == http.MethodPost {
			func(w http.ResponseWriter, r *http.Request) {
				invalidate := func(ctx context.Context, keys ...string) {
					if app.Cache != nil {
						app.Cache.Invalidate(ctx, keys...)
					}
				}
				auth.HandleCreateFamilyMember(spannerClient, invalidate)(w, r)
			}(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Edge 29: DELETE /v1/retailer/family-members/{id} — Remove family sub-profile
	http.HandleFunc("/v1/retailer/family-members/", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(auth.HandleDeleteFamilyMember(spannerClient, func(ctx context.Context, keys ...string) {
		if app.Cache != nil {
			app.Cache.Invalidate(ctx, keys...)
		}
	}))))

	// /v1/delivery/credit-delivery moved to deliveryroutes.

	// /v1/admin/orders/resolve-credit moved to adminroutes.

	// /v1/delivery/missing-items moved to deliveryroutes.

	// Edge 34: POST /v1/retailer/orders/confirm-ai — Retailer confirms AI-suggested order
	http.HandleFunc("/v1/retailer/orders/confirm-ai", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(order.HandleConfirmAiOrder(svc))))

	// Edge 34: POST /v1/retailer/orders/reject-ai — Retailer rejects AI-suggested order
	http.HandleFunc("/v1/retailer/orders/reject-ai", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(order.HandleRejectAiOrder(svc))))

	// Preorder lifecycle: edit and confirm
	http.HandleFunc("/v1/orders/edit-preorder", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(order.HandleEditPreorder(svc))))
	http.HandleFunc("/v1/orders/confirm-preorder", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(order.HandleConfirmPreorder(svc))))

	// /v1/delivery/split-payment moved to deliveryroutes.

	// GET/POST /v1/retailer/cart/sync — Server-side cart persistence
	http.HandleFunc("/v1/retailer/cart/sync", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(order.HandleCartSync(spannerClient))))

	// /v1/user/notifications{,/read} moved to userroutes.

	// GET /v1/supplier/earnings — Supplier revenue breakdown
	http.HandleFunc("/v1/supplier/earnings", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(analytics.HandleSupplierEarnings(spannerClient, app.SpannerRouter))))

	// /v1/supplier/settlement-report → treasury package (registered above).

	// /v1/auth/refresh, /v1/auth/{driver,admin,supplier,retailer,payloader,factory,warehouse}/...
	// were moved to backend-go/authroutes/routes.go (registered above near the /v1/auth/login block).

	// /v1/driver/profile moved to driverroutes.

	// POST /v1/supplier/configure — Supplier onboarding (TaxId, categories)
	http.HandleFunc("/v1/supplier/configure",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleSupplierConfigure(spannerClient))))

	// POST /v1/supplier/billing/setup — Post-registration billing setup (bank, payment gateway)
	http.HandleFunc("/v1/supplier/billing/setup",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleBillingSetup(spannerClient))))

	// GET/PUT /v1/supplier/profile — Supplier profile read & update
	http.HandleFunc("/v1/supplier/profile",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supplier.HandleGetSupplierProfile(spannerClient, app.Cache, app.CacheFlight)(w, r)
			case http.MethodPut:
				supplier.HandleUpdateSupplierProfile(spannerClient, app.Cache)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})))

	// PATCH /v1/supplier/shift — Toggle ManualOffShift and/or update OperatingSchedule
	http.HandleFunc("/v1/supplier/shift",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleSupplierShift(spannerClient))))

	// GET/POST/DELETE /v1/supplier/payment-config — Supplier payment gateway vault CRUD
	http.HandleFunc("/v1/supplier/payment-config",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(vault.HandlePaymentConfigs(spannerClient))))

	// POST/GET/DELETE /v1/supplier/gateway-onboarding — Supplier gateway connect sessions
	http.HandleFunc("/v1/supplier/gateway-onboarding",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(vault.HandleGatewayOnboarding(spannerClient))))

	// POST /v1/supplier/payment/recipient/register — Global Pay split-payment recipient onboarding
	http.HandleFunc("/v1/supplier/payment/recipient/register",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(vault.HandleRegisterRecipient(spannerClient, directClient))))

	// /v1/catalog/platform-categories moved to catalogroutes.

	// /v1/auth/retailer/{login,register} → authroutes package.

	// /v1/admin/{nuke,config,empathy/adoption} moved to adminroutes.

	// ── Organization Members (SupplierUsers — Sub-Accounts) ─────────────────
	http.HandleFunc("/v1/supplier/org/members", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleOrgMembers(spannerClient)))))
	http.HandleFunc("/v1/supplier/org/members/invite", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleOrgInvite(spannerClient))))
	http.HandleFunc("/v1/supplier/org/members/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleOrgMemberAction(spannerClient))))

	// ── Warehouse Staff (Payloader) Provisioning ────────────────────────────
	http.HandleFunc("/v1/supplier/staff/payloader", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleStaffPayloaders(spannerClient)))))
	http.HandleFunc("/v1/supplier/staff/payloader/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandlePayloaderDetail(spannerClient)))))

	// ── Warehouse Staff Management (Supplier-scoped) ────────────────────────
	http.HandleFunc("/v1/supplier/warehouse-staff", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleWarehouseStaff(spannerClient)))))
	http.HandleFunc("/v1/supplier/warehouse-staff/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleWarehouseStaffToggle(spannerClient)))))

	// ── Warehouse Management (1:N Supplier→Warehouse) ───────────────────────
	http.HandleFunc("/v1/supplier/warehouses", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleWarehouses(spannerClient, svc.Producer)))))
	http.HandleFunc("/v1/supplier/warehouses/", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleWarehouseByID(spannerClient, svc.Producer)))))
	// GET /v1/supplier/warehouse-inflight-vu?warehouse_id=X — VU utilization for capacity guardrails
	http.HandleFunc("/v1/supplier/warehouse-inflight-vu", auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleWarehouseInflightVU(spannerClient)))))

	// ── Geo-Spatial Sovereignty ──────────────────────────────────────────────
	// GET /v1/supplier/serving-warehouse?retailer_lat=X&retailer_lng=Y — Exclusive warehouse resolver
	http.HandleFunc("/v1/supplier/serving-warehouse",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleGetServingWarehouse(spannerClient, app.SpannerRouter))))
	// GET /v1/supplier/geo-report — Coverage health: dead zones, overlaps, warehouse stats
	http.HandleFunc("/v1/supplier/geo-report",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleGeoReport(spannerClient))))
	// GET /v1/supplier/zone-preview?lat=X&lng=Y&radius_km=R — Real-time density preview for warehouse zone
	http.HandleFunc("/v1/supplier/zone-preview",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleZonePreview(spannerClient))))
	// POST /v1/supplier/warehouses/validate-coverage — H3 cell computation + overlap detection for CoverageEditor
	http.HandleFunc("/v1/supplier/warehouses/validate-coverage",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleValidateCoverage(spannerClient))))
	// GET /v1/supplier/warehouse-loads — Live warehouse queue depth + load factor (Redis-primary, 15s poll)
	http.HandleFunc("/v1/supplier/warehouse-loads",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleWarehouseLoads(spannerClient))))

	// /v1/auth/payloader/login → authroutes package.

	// /v1/payloader/* — 3 routes (trucks, orders, recommend-reassign).
	// Ownership lives in backend-go/payloaderroutes.
	payloaderroutes.RegisterRoutes(r, payloaderroutes.Deps{Spanner: spannerClient, ReadRouter: app.SpannerRouter, Log: loggingMiddleware})

	// /v1/delivery/* — 9 routes (arrive, confirm-payment-bypass, sms-complete,
	// shop-closed, bypass-offload, negotiate, credit-delivery, missing-items,
	// split-payment). Ownership lives in backend-go/deliveryroutes.
	deliveryroutes.RegisterRoutes(r, deliveryroutes.Deps{
		Order:             svc,
		Cache:             app.Cache,
		FleetHub:          app.FleetHub,
		ShopClosedDeps:    &shopClosedDeps,
		EarlyCompleteDeps: &earlyCompleteDeps,
		NegotiationDeps:   &negotiationDeps,
		Log:               loggingMiddleware,
	})

	// /v1/fleet/* — 12 routes (drivers/{id}/status, route/request-early-complete,
	// dispatch, reassign, capacity, active, trucks/{id}/{seal|status},
	// driver/depart, driver/return-complete, route/reorder, orders,
	// route/{id}/complete). Ownership lives in backend-go/fleetroutes.
	fleetroutes.RegisterRoutes(r, fleetroutes.Deps{
		Spanner:           spannerClient,
		Order:             svc,
		RetailerHub:       retailerHub,
		EarlyCompleteDeps: &earlyCompleteDeps,
		Producer:          svc.Producer,
		MapsAPIKey:        cfg.GoogleMapsAPIKey,
		Log:               loggingMiddleware,
	})

	http.HandleFunc("/v1/order/deliver", auth.RequireRole([]string{"DRIVER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			OrderId      string  `json:"order_id"`
			ScannedToken string  `json:"scanned_token"`
			Latitude     float64 `json:"latitude"`
			Longitude    float64 `json:"longitude"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderId == "" || req.ScannedToken == "" {
			http.Error(w, "Invalid payload. order_id and scanned_token required.", http.StatusBadRequest)
			return
		}

		supplierID, err := svc.CompleteDeliveryWithToken(r.Context(), req.OrderId, req.ScannedToken, req.Latitude, req.Longitude)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden) // 403 Forbidden if the QR is wrong!
			return
		}

		// Invalidate Redis-cached delivery token — order is delivered
		go svc.InvalidateDeliveryToken(context.Background(), req.OrderId)

		// Push ORDER_STATE_CHANGED to supplier admin portal via WebSocket
		if supplierID != "" {
			go telemetry.FleetHub.BroadcastOrderStateChange(supplierID, req.OrderId, "COMPLETED", "")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETED",
			"message": "Handshake successful. Delivery completed.",
		})

		// Auto-release truck if manifest is fully delivered
		go fleet.CheckAndAutoReleaseTruck(context.Background(), spannerClient, req.OrderId, cfg.GoogleMapsAPIKey)
	}))))

	// ── NEW DELIVERY FLOW: validate-qr → confirm-offload → complete ──────────

	// POST /v1/order/validate-qr — Driver scans QR, validates token, sees order info
	http.HandleFunc("/v1/order/validate-qr", auth.RequireRole([]string{"DRIVER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID      string `json:"order_id"`
			ScannedToken string `json:"scanned_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.ScannedToken == "" {
			http.Error(w, "order_id and scanned_token required", http.StatusBadRequest)
			return
		}
		resp, err := svc.ValidateQRToken(r.Context(), req.OrderID, req.ScannedToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})))

	// POST /v1/order/confirm-offload — Driver confirms offload, triggers payment flow
	http.HandleFunc("/v1/order/confirm-offload", auth.RequireRole([]string{"DRIVER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}
		resp, err := svc.ConfirmOffload(r.Context(), req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		// Push PAYMENT_REQUIRED to the retailer's WebSocket for all payment methods
		retailerHub.PushToRetailer(resp.RetailerID, map[string]interface{}{
			"type":                    ws.EventPaymentRequired,
			"order_id":                resp.OrderID,
			"invoice_id":              resp.InvoiceID,
			"session_id":              resp.SessionID,
			"amount":                  resp.Amount,
			"original_amount":         resp.OriginalAmount,
			"payment_method":          resp.PaymentMethod,
			"available_card_gateways": resp.AvailableCardGateways,
			"message":                 fmt.Sprintf("Payment of %d required for order %s", resp.Amount, resp.OrderID),
		})

		// Push ORDER_STATE_CHANGED to supplier admin portal via WebSocket
		if resp.SupplierID != "" {
			go telemetry.FleetHub.BroadcastOrderStateChange(resp.SupplierID, resp.OrderID, "AWAITING_PAYMENT", "")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))))

	// POST /v1/order/complete — Driver finalizes delivery after payment
	http.HandleFunc("/v1/order/complete", auth.RequireRole([]string{"DRIVER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}
		supplierID, err := svc.CompleteOrder(r.Context(), req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		// Invalidate Redis-cached delivery token
		go svc.InvalidateDeliveryToken(context.Background(), req.OrderID)

		// Push ORDER_STATE_CHANGED to supplier admin portal via WebSocket
		if supplierID != "" {
			go telemetry.FleetHub.BroadcastOrderStateChange(supplierID, req.OrderID, "COMPLETED", "")
		}

		// Auto-release truck
		go fleet.CheckAndAutoReleaseTruck(context.Background(), spannerClient, req.OrderID, cfg.GoogleMapsAPIKey)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETED",
			"message": "Delivery finalized.",
		})
	}))))

	// POST /v1/order/cash-checkout — Retailer selects cash payment after offload
	http.HandleFunc("/v1/order/cash-checkout", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}

		resp, err := svc.CashCheckout(r.Context(), req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		// Push CASH_COLLECTION_REQUIRED to the driver via WebSocket
		driverHub.PushToDriver(resp.DriverID, map[string]interface{}{
			"type":     ws.EventCashCollectionRequired,
			"order_id": resp.OrderID,
			"amount":   resp.Amount,
			"message":  resp.Message,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})))

	// POST /v1/order/card-checkout — Retailer selects a hosted card gateway after offload
	http.HandleFunc("/v1/order/card-checkout", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
			Gateway string `json:"gateway"` // "GLOBAL_PAY", "CASH", or "GLOBAL_PAY"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.Gateway == "" {
			http.Error(w, "order_id and gateway required", http.StatusBadRequest)
			return
		}

		resp, err := svc.CardCheckout(r.Context(), req.OrderID, req.Gateway, requestBaseURL(r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})))

	// ── RETAILER CARD MANAGEMENT ──────────────────────────────────────────────

	// POST /v1/retailer/card/initiate — Start card tokenization (OTP flow)
	http.HandleFunc("/v1/retailer/card/initiate", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cardsClient == nil {
			http.Error(w, "card tokenization not configured", http.StatusServiceUnavailable)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		var req struct {
			Gateway string `json:"gateway"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Gateway == "" {
			req.Gateway = "GLOBAL_PAY"
		}

		phone, phoneErr := svc.LookupRetailerPhone(r.Context(), claims.UserID)
		if phoneErr != nil || phone == "" {
			http.Error(w, "retailer phone number required for card tokenization", http.StatusBadRequest)
			return
		}

		creds, credErr := payment.ResolveGlobalPayCredentials("", "", "")
		if credErr != nil {
			http.Error(w, "payment gateway credentials not configured", http.StatusServiceUnavailable)
			return
		}
		result, err := cardsClient.InitiateCardSave(r.Context(), creds, phone)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})))

	// POST /v1/retailer/card/confirm — Confirm OTP and save card token
	http.HandleFunc("/v1/retailer/card/confirm", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cardsClient == nil {
			http.Error(w, "card tokenization not configured", http.StatusServiceUnavailable)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}
		var req struct {
			CardToken string `json:"card_token"`
			OTPCode   string `json:"otp_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CardToken == "" || req.OTPCode == "" {
			http.Error(w, "card_token and otp_code required", http.StatusBadRequest)
			return
		}

		creds, credErr := payment.ResolveGlobalPayCredentials("", "", "")
		if credErr != nil {
			http.Error(w, "payment gateway credentials not configured", http.StatusServiceUnavailable)
			return
		}
		result, err := cardsClient.ConfirmCardOTP(r.Context(), creds, req.CardToken, req.OTPCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if !result.Confirmed {
			http.Error(w, "OTP confirmation failed", http.StatusUnprocessableEntity)
			return
		}

		tokenID, saveErr := cardTokenSvc.SaveCard(r.Context(), claims.UserID, "GLOBAL_PAY", req.CardToken, result.CardLast4, result.CardType)
		if saveErr != nil {
			http.Error(w, "card confirmed but failed to save: "+saveErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token_id":   tokenID,
			"card_last4": result.CardLast4,
			"card_type":  result.CardType,
			"confirmed":  true,
		})
	})))

	// GET /v1/retailer/cards — List saved cards
	http.HandleFunc("/v1/retailer/cards", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}
		cards, err := cardTokenSvc.ListCards(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cards == nil {
			cards = []payment.RetailerCardToken{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"cards": cards})
	})))

	// DELETE /v1/retailer/card — Deactivate a saved card
	http.HandleFunc("/v1/retailer/card/deactivate", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}
		var req struct {
			TokenID string `json:"token_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TokenID == "" {
			http.Error(w, "token_id required", http.StatusBadRequest)
			return
		}
		if err := cardTokenSvc.DeactivateCard(r.Context(), req.TokenID, claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deactivated"})
	})))

	// POST /v1/retailer/card/default — Set default card
	http.HandleFunc("/v1/retailer/card/default", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}
		var req struct {
			TokenID string `json:"token_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TokenID == "" {
			http.Error(w, "token_id required", http.StatusBadRequest)
			return
		}
		if err := cardTokenSvc.SetDefaultCard(r.Context(), req.TokenID, claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "default_set"})
	})))

	// POST /v1/order/collect-cash — Driver confirms geofenced cash collection
	http.HandleFunc("/v1/order/collect-cash", auth.RequireRole([]string{"DRIVER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			OrderID   string  `json:"order_id"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}
		if req.Latitude == 0 && req.Longitude == 0 {
			http.Error(w, "GPS coordinates required (latitude, longitude)", http.StatusBadRequest)
			return
		}

		// Extract driver_id from JWT claims
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "driver identity missing from token", http.StatusUnauthorized)
			return
		}

		resp, err := svc.CollectCash(r.Context(), order.CollectCashRequest{
			OrderID:   req.OrderID,
			DriverID:  claims.UserID,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		// Push ORDER_COMPLETED to the retailer via WebSocket
		retailerHub.PushToRetailer(resp.RetailerID, map[string]interface{}{
			"type":     ws.EventOrderCompleted,
			"order_id": resp.OrderID,
			"amount":   resp.Amount,
			"message":  resp.Message,
		})

		// Auto-release truck
		go fleet.CheckAndAutoReleaseTruck(context.Background(), spannerClient, req.OrderID, cfg.GoogleMapsAPIKey)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))))

	// GET /v1/retailer/pending-payments — Returns all active payment sessions for the retailer
	http.HandleFunc("/v1/retailer/pending-payments", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		sessions, err := sessionSvc.GetPendingSessionsByRetailer(r.Context(), claims.UserID)
		if err != nil {
			http.Error(w, "failed to retrieve pending payments", http.StatusInternalServerError)
			return
		}
		if sessions == nil {
			sessions = []payment.PaymentSession{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pending_payments": sessions,
			"count":            len(sessions),
		})
	})))

	// GET /v1/retailer/active-fulfillment — Incoming deliveries visible to the retailer
	// Returns orders in IN_TRANSIT / ARRIVED / AWAITING_PAYMENT with supplier name and adjusted amount.
	http.HandleFunc("/v1/retailer/active-fulfillment", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "retailer identity missing from token", http.StatusUnauthorized)
			return
		}

		items, err := svc.ActiveFulfillments(r.Context(), claims.UserID)
		if err != nil {
			log.Printf("[ACTIVE FULFILLMENT] Query failed for retailer %s: %v", claims.UserID, err)
			http.Error(w, "failed to retrieve active fulfillments", http.StatusInternalServerError)
			return
		}
		if items == nil {
			items = []order.ActiveFulfillmentItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fulfillments": items,
			"count":        len(items),
		})
	})))

	http.HandleFunc("/v1/routes", auth.RequireRole([]string{"ADMIN", "SUPPLIER", "PAYLOADER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		routes, err := svc.ListRoutes(r.Context())
		if err != nil {
			log.Printf("Failed to list routes: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routes)
	})))

	http.HandleFunc("/v1/payload/seal", auth.RequireRole([]string{"ADMIN", "SUPPLIER", "PAYLOADER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req order.PayloadSealRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		retailerID, err := svc.SealPayload(r.Context(), req)
		if err != nil {
			log.Printf("Payload Seal Hash Failure for order %s: %v", req.OrderID, err)

			// Distinguish between bad requests/conflicts vs internal errors
			if strings.Contains(err.Error(), "bad request") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if strings.Contains(err.Error(), "conflict") {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}

			http.Error(w, "Internal Server Error during Payload Seal", http.StatusInternalServerError)
			return
		}

		// Push ORDER_STATUS_CHANGED (DISPATCHED) to retailer via WebSocket
		if retailerID != "" {
			go retailerHub.PushToRetailer(retailerID, map[string]interface{}{
				"type":      ws.EventOrderStatusChanged,
				"order_id":  req.OrderID,
				"state":     "DISPATCHED",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		dispatchCode := order.GenerateSecureToken()
		json.NewEncoder(w).Encode(map[string]string{
			"status":        "PAYLOAD_SEALED_AND_DISPATCHED",
			"dispatch_code": dispatchCode,
			"order_id":      req.OrderID,
		})
	}))))

	http.HandleFunc("/v1/prediction/create", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			RetailerId  string `json:"retailer_id"`
			Amount      int64  `json:"amount"`
			TriggerDate string `json:"trigger_date"`
			Status      string `json:"status,omitempty"` // WAITING or DORMANT
			WarehouseId string `json:"warehouse_id,omitempty"`
			Items       []struct {
				SkuID    string `json:"sku_id"`
				Quantity int64  `json:"quantity"`
				Price    int64  `json:"price"`
			} `json:"items,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if len(req.Items) > 0 {
			// SKU-level prediction (v2)
			var items []order.PredictionItem
			for _, it := range req.Items {
				items = append(items, order.PredictionItem{
					SkuID: it.SkuID, Quantity: it.Quantity, Price: it.Price,
				})
			}
			err := svc.SavePredictionWithItems(r.Context(), req.RetailerId, req.Amount, req.TriggerDate, items, req.Status, req.WarehouseId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// Legacy amount-only prediction (v1)
			err := svc.SavePrediction(r.Context(), req.RetailerId, req.Amount, req.TriggerDate, req.WarehouseId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "PREDICTION_LOCKED"})
	})))

	http.HandleFunc("/v1/order/create", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Accept simplified procurement payload: { retailer_id, items: [{product_id, quantity}] }
		type ProcurementItem struct {
			ProductID string `json:"product_id"`
			Quantity  int64  `json:"quantity"`
		}
		type ProcurementRequest struct {
			RetailerID string            `json:"retailer_id"`
			Items      []ProcurementItem `json:"items"`
		}

		var req ProcurementRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.RetailerID == "" {
			http.Error(w, `{"error":"retailer_id is required"}`, http.StatusUnprocessableEntity)
			return
		}
		if len(req.Items) == 0 {
			http.Error(w, `{"error":"items must not be empty"}`, http.StatusUnprocessableEntity)
			return
		}

		ctx := r.Context()

		// Look up retailer location for the order
		var lat, lng float64
		row, err := spannerClient.Single().ReadRow(ctx, "Retailers",
			spanner.Key{req.RetailerID}, []string{"Latitude", "Longitude"})
		if err == nil {
			var nLat, nLng spanner.NullFloat64
			if colErr := row.Columns(&nLat, &nLng); colErr == nil {
				lat, lng = nLat.Float64, nLng.Float64
			}
		}

		// Look up prices from SupplierProducts for each item
		productIDs := make([]string, len(req.Items))
		for i, item := range req.Items {
			productIDs[i] = item.ProductID
		}
		stmt := spanner.Statement{
			SQL:    `SELECT SkuId, BasePrice FROM SupplierProducts WHERE SkuId IN UNNEST(@ids) AND IsActive = TRUE`,
			Params: map[string]interface{}{"ids": productIDs},
		}
		priceMap := make(map[string]int64)
		iter := spannerClient.Single().Query(ctx, stmt)
		for {
			priceRow, iterErr := iter.Next()
			if iterErr != nil {
				break
			}
			var skuId string
			var price int64
			if colErr := priceRow.Columns(&skuId, &price); colErr == nil {
				priceMap[skuId] = price
			}
		}
		iter.Stop()

		// Compute total
		var totalAmount int64
		for _, item := range req.Items {
			if p, ok := priceMap[item.ProductID]; ok {
				totalAmount += p * item.Quantity
			}
		}

		// Create the order
		orderID, err := svc.CreateOrder(ctx, order.CreateOrderRequest{
			RetailerID:     req.RetailerID,
			Amount:         totalAmount,
			PaymentGateway: "PENDING",
			Latitude:       lat,
			Longitude:      lng,
			OrderSource:    "PROCUREMENT",
			State:          "PENDING",
		})
		if err != nil {
			log.Printf("[PROCUREMENT] CreateOrder failed for retailer %s: %v", req.RetailerID, err)
			http.Error(w, `{"error":"Order creation failed"}`, http.StatusInternalServerError)
			return
		}

		// Insert line items
		var mutations []*spanner.Mutation
		for _, item := range req.Items {
			lineItemID := fmt.Sprintf("LI-%s", order.GenerateSecureToken())
			unitPrice := priceMap[item.ProductID]
			mutations = append(mutations, spanner.Insert("OrderLineItems",
				[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Status"},
				[]interface{}{lineItemID, orderID, item.ProductID, item.Quantity, unitPrice, "PENDING"},
			))
		}
		if len(mutations) > 0 {
			if _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite(mutations)
			}); err != nil {
				log.Printf("[PROCUREMENT] OrderLineItems insert failed for %s: %v", orderID, err)
			}
		}

		log.Printf("[PROCUREMENT] OrderID=%s RetailerId=%s Total=%d Items=%d",
			orderID, req.RetailerID, totalAmount, len(req.Items))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "PROCUREMENT_AUTHORIZED",
			"order_id": orderID,
			"total":    totalAmount,
		})
	})))

	// POST /v1/order/cancel — Retailer cancellation with access control firewall
	http.HandleFunc("/v1/order/cancel", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req order.CancelOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		// Auto-fill RetailerID from JWT so mobile clients don't have to send it
		if req.RetailerID == "" {
			if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims); ok && claims.Role == "RETAILER" {
				req.RetailerID = claims.UserID
			}
		}
		if req.OrderID == "" || req.RetailerID == "" {
			http.Error(w, "order_id and retailer_id are required", http.StatusBadRequest)
			return
		}

		err := svc.CancelOrder(r.Context(), req)
		if err != nil {
			// State conflict → 409
			var conflict *order.ErrStateConflict
			if errors.As(err, &conflict) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": conflict.Error()})
				return
			}
			// OCC version conflict → 409
			var versionConflict *order.ErrVersionConflict
			if errors.As(err, &versionConflict) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": versionConflict.Error()})
				return
			}
			// Freeze lock → 423 Locked
			var freezeLock *order.ErrFreezeLock
			if errors.As(err, &freezeLock) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(423) // Locked
				json.NewEncoder(w).Encode(map[string]string{"error": freezeLock.Error()})
				return
			}
			// Firewall rejection → 403
			var forbidden *order.ErrCancelForbidden
			if errors.As(err, &forbidden) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, forbidden.Reason)))
				return
			}
			// Not found
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			log.Printf("CancelOrder failed for %s: %v", req.OrderID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"status": "ORDER_CANCELLED", "order_id": "%s"}`, req.OrderID)))

		// Invalidate Redis-cached delivery token on cancellation
		go svc.InvalidateDeliveryToken(context.Background(), req.OrderID)
	}))))

	// ── Platform Config (Phase 4.1) — must init before handlers that use it ──
	platformCfg := settings.NewPlatformConfig(spannerClient)

	// ── Refund Endpoint (Phase 3.1) ──
	refundSvc := payment.NewRefundService(spannerClient, platformCfg.PlatformFeeBasisPoints())
	chargebackSvc := payment.NewChargebackService(spannerClient)
	http.HandleFunc("/v1/order/refund", auth.RequireRole([]string{"ADMIN", "SUPPLIER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		var req payment.RefundRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.OrderID == "" {
			http.Error(w, `{"error":"order_id is required"}`, http.StatusBadRequest)
			return
		}

		result, err := refundSvc.InitiateRefund(r.Context(), req, claims.UserID)
		if err != nil {
			log.Printf("[REFUND] Failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	})))

	// GET /v1/order/{id}/refunds — List refunds for an order
	http.HandleFunc("/v1/order/refunds", auth.RequireRole([]string{"ADMIN", "SUPPLIER", "RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		orderID := r.URL.Query().Get("order_id")
		if orderID == "" {
			http.Error(w, `{"error":"order_id query param required"}`, http.StatusBadRequest)
			return
		}
		refunds, err := refundSvc.GetRefundsByOrder(r.Context(), orderID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(refunds)
	})))

	// /v1/checkout/* + /v1/payment/* — 5 routes (b2b, unified, chargeback,
	// chargeback/reversal, global_pay/initiate). Ownership lives in backend-go/paymentroutes.
	paymentroutes.RegisterRoutes(r, paymentroutes.Deps{
		Spanner:       spannerClient,
		Checkout:      svc,
		Chargeback:    chargebackSvc,
		Log:           loggingMiddleware,
		PriorityGuard: priorityGuard,
		Idempotency:   idempotency.Guard,
	})

	// /v1/admin/config/platform-fee moved to adminroutes.

	// /v1/fleet/{dispatch,reassign,capacity,active} moved to fleetroutes.

	// /v1/fleet/{trucks,driver/depart,driver/return-complete,route/reorder,orders} moved to fleetroutes.

	// GET /v1/orders — List orders with optional filters.
	// POST /v1/orders is REMOVED — use POST /v1/order/create (OrderService.CreateOrder) instead.
	http.HandleFunc("/v1/orders", auth.RequireRole([]string{"ADMIN", "RETAILER", "SUPPLIER", "PAYLOADER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed — use POST /v1/order/create", http.StatusMethodNotAllowed)
			return
		}

		routeId := r.URL.Query().Get("route_id")
		stateFilter := r.URL.Query().Get("state")
		retailerId := r.URL.Query().Get("retailer_id")

		claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if claims != nil && claims.Role == "RETAILER" {
			retailerId = claims.UserID
		}

		limit := 100
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, parseErr := strconv.Atoi(raw)
			if parseErr != nil {
				http.Error(w, "Invalid limit", http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		offset := int64(0)
		if raw := r.URL.Query().Get("offset"); raw != "" {
			parsed, parseErr := strconv.ParseInt(raw, 10, 64)
			if parseErr != nil {
				http.Error(w, "Invalid offset", http.StatusBadRequest)
				return
			}
			offset = parsed
		}

		orders, err := svc.ListOrdersPaginated(r.Context(), routeId, stateFilter, retailerId, limit, offset)
		if err != nil {
			log.Printf("Failed to list orders: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			log.Printf("Failed to write orders response: %v", err)
		}
	})))

	http.HandleFunc("/v1/orders/", auth.RequireRole([]string{"ADMIN", "DRIVER", "RETAILER"}, loggingMiddleware(order.HandleLegacyOrdersPath(svc))))

	// Legacy /v1/orders/{id}/items and /v1/order-items/ removed — use OrderLineItems via OrderService.

	http.HandleFunc("/v1/products", auth.RequireRole([]string{"RETAILER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		type Variant struct {
			ID            string  `json:"id"`
			Size          string  `json:"size"`
			Pack          string  `json:"pack"`
			PackCount     int64   `json:"pack_count"`
			WeightPerUnit string  `json:"weight_per_unit"`
			Price         float64 `json:"price"`
		}

		type Product struct {
			ID               string    `json:"id"`
			Name             string    `json:"name"`
			Description      string    `json:"description"`
			Nutrition        string    `json:"nutrition"`
			ImageURL         string    `json:"image_url"`
			Variants         []Variant `json:"variants"`
			SupplierID       string    `json:"supplier_id"`
			SupplierName     string    `json:"supplier_name"`
			SupplierCategory string    `json:"supplier_category"`
			CategoryID       string    `json:"category_id"`
			CategoryName     string    `json:"category_name"`
			SellByBlock      bool      `json:"sell_by_block"`
			UnitsPerBlock    int64     `json:"units_per_block"`
			Price            int64     `json:"price"`
		}

		stmt := spanner.Statement{
			SQL: `SELECT sp.SkuId, sp.SupplierId, sp.Name, sp.Description, sp.ImageUrl,
			             sp.SellByBlock, sp.UnitsPerBlock, sp.BasePrice, sp.CategoryId,
			             COALESCE(c.Name, '') AS CategoryName,
			             COALESCE(s.Name, '') AS SupplierName,
			             COALESCE(s.Category, '') AS SupplierCategory
			      FROM SupplierProducts sp
			      LEFT JOIN Suppliers s ON sp.SupplierId = s.SupplierId
			      LEFT JOIN Categories c ON c.CategoryId = sp.CategoryId
			      WHERE sp.IsActive = TRUE
			      ORDER BY sp.Name ASC`,
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var productList []Product
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to query products: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var skuId, supplierId, name string
			var desc, imageUrl, catId, categoryName, supplierName, supplierCategory spanner.NullString
			var sellByBlock bool
			var unitsPerBlock, basePrice int64

			if err := row.Columns(&skuId, &supplierId, &name, &desc, &imageUrl,
				&sellByBlock, &unitsPerBlock, &basePrice, &catId, &categoryName, &supplierName, &supplierCategory); err != nil {
				log.Printf("Failed to parse product row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			p := Product{
				ID:            skuId,
				Name:          name,
				SellByBlock:   sellByBlock,
				UnitsPerBlock: unitsPerBlock,
				Price:         basePrice,
				SupplierID:    supplierId,
			}
			if desc.Valid {
				p.Description = desc.StringVal
			}
			if imageUrl.Valid {
				p.ImageURL = imageUrl.StringVal
			}
			if catId.Valid {
				p.CategoryID = catId.StringVal
			}
			if categoryName.Valid {
				p.CategoryName = categoryName.StringVal
			}
			if supplierName.Valid {
				p.SupplierName = supplierName.StringVal
			}
			if supplierCategory.Valid {
				p.SupplierCategory = supplierCategory.StringVal
			}

			// Create a synthetic variant so the iOS detail view's variant picker + cart flow works
			packLabel := "Per unit"
			if sellByBlock && unitsPerBlock > 1 {
				packLabel = fmt.Sprintf("Block of %d", unitsPerBlock)
			}
			v := Variant{
				ID:            skuId,
				Size:          "Standard",
				Pack:          packLabel,
				PackCount:     1,
				WeightPerUnit: "1 unit",
				Price:         float64(basePrice),
			}
			p.Variants = []Variant{v}

			productList = append(productList, p)
		}

		if productList == nil {
			productList = []Product{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(productList); err != nil {
			log.Printf("Failed to write products response payload: %v", err)
		}
	})))

	http.HandleFunc("/v1/retailers/", auth.RequireRole([]string{"ADMIN", "RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Expected path: /v1/retailers/{id}/orders
		path := strings.TrimPrefix(r.URL.Path, "/v1/retailers/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 || parts[1] != "orders" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		retailerId := parts[0]

		// Enforce: RETAILER role can only query their own orders
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if ok && claims.Role == "RETAILER" && claims.UserID != retailerId {
			http.Error(w, `{"error":"forbidden: cannot access another retailer's orders"}`, http.StatusForbidden)
			return
		}

		orders, err := svc.ListOrders(r.Context(), "", "", retailerId)
		if err != nil {
			log.Printf("Failed to list orders for retailer %s: %v", retailerId, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Map to mobile-friendly response shape
		type mobileLineItem struct {
			ID          string `json:"id"`
			ProductID   string `json:"product_id"`
			ProductName string `json:"product_name"`
			VariantID   string `json:"variant_id,omitempty"`
			VariantSize string `json:"variant_size,omitempty"`
			Quantity    int64  `json:"quantity"`
			UnitPrice   int64  `json:"unit_price"`
			TotalPrice  int64  `json:"total_price"`
		}
		type mobileOrder struct {
			OrderID           string           `json:"order_id"`
			RetailerID        string           `json:"retailer_id"`
			SupplierID        string           `json:"supplier_id,omitempty"`
			SupplierName      string           `json:"supplier_name,omitempty"`
			State             string           `json:"state"`
			Amount            int64            `json:"amount"`
			OrderSource       string           `json:"order_source,omitempty"`
			CreatedAt         string           `json:"created_at"`
			UpdatedAt         string           `json:"updated_at,omitempty"`
			EstimatedDelivery string           `json:"estimated_delivery,omitempty"`
			DeliveryToken     string           `json:"delivery_token,omitempty"`
			Items             []mobileLineItem `json:"items"`
		}

		result := make([]mobileOrder, 0, len(orders))
		for _, o := range orders {
			mo := mobileOrder{
				OrderID:      o.ID,
				RetailerID:   o.RetailerID,
				SupplierID:   o.SupplierID,
				SupplierName: o.SupplierName,
				State:        o.State,
				Amount:       o.Amount,
				CreatedAt:    o.CreatedAt.Format(time.RFC3339),
			}
			if o.OrderSource.Valid {
				mo.OrderSource = o.OrderSource.StringVal
			}
			if o.DeliverBefore.Valid {
				mo.EstimatedDelivery = o.DeliverBefore.Time.Format(time.RFC3339)
			}
			if o.DeliveryToken.Valid {
				mo.DeliveryToken = o.DeliveryToken.StringVal
			}
			mo.Items = make([]mobileLineItem, 0, len(o.Items))
			for _, li := range o.Items {
				mo.Items = append(mo.Items, mobileLineItem{
					ID:          li.LineItemID,
					ProductID:   li.SkuID,
					ProductName: li.SkuName,
					Quantity:    li.Quantity,
					UnitPrice:   li.UnitPrice,
					TotalPrice:  li.Quantity * li.UnitPrice,
				})
			}
			result = append(result, mo)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("Failed to write orders response payload: %v", err)
		}
	})))

	// ── Retailer Delivery Tracking (real-time driver positions via Redis GEO) ──
	http.HandleFunc("/v1/retailer/tracking", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		retailerID := claims.UserID

		// 1. Fetch active orders for this retailer with SupplierId + DriverId
		activeStmt := spanner.Statement{
			SQL: `SELECT OrderId, SupplierId, DriverId, State, Amount, DeliveryToken, OrderSource, CreatedAt,
			             COALESCE(WarehouseId, '') AS WarehouseId
			      FROM Orders
			      WHERE RetailerId = @retailerId
			        AND State IN ('PENDING', 'LOADED', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED')
			      ORDER BY CreatedAt DESC
			      LIMIT 50`,
			Params: map[string]interface{}{"retailerId": retailerID},
		}

		type trackingRow struct {
			OrderID       string
			SupplierID    string
			DriverID      string
			State         string
			Amount        int64
			DeliveryToken string
			OrderSource   string
			CreatedAt     time.Time
			WarehouseID   string
		}

		iter := spannerClient.Single().Query(r.Context(), activeStmt)
		defer iter.Stop()

		var rows []trackingRow
		supplierIDs := map[string]bool{}
		driverIDs := map[string]bool{}
		warehouseIDs := map[string]bool{}
		var orderIDs []string

		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[TRACKING] Failed to query active orders for retailer %s: %v", retailerID, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var orderID, stateVal string
			var supplierID, driverID, deliveryToken, orderSource spanner.NullString
			var amount spanner.NullInt64
			var createdAt spanner.NullTime

			var warehouseID string
			if err := row.Columns(&orderID, &supplierID, &driverID, &stateVal, &amount, &deliveryToken, &orderSource, &createdAt, &warehouseID); err != nil {
				log.Printf("[TRACKING] Column parse failed: %v", err)
				continue
			}

			tr := trackingRow{
				OrderID:       orderID,
				SupplierID:    supplierID.StringVal,
				DriverID:      driverID.StringVal,
				State:         stateVal,
				Amount:        amount.Int64,
				DeliveryToken: deliveryToken.StringVal,
				OrderSource:   orderSource.StringVal,
				CreatedAt:     createdAt.Time,
				WarehouseID:   warehouseID,
			}
			rows = append(rows, tr)
			orderIDs = append(orderIDs, orderID)
			if supplierID.Valid && supplierID.StringVal != "" {
				supplierIDs[supplierID.StringVal] = true
			}
			if driverID.Valid && driverID.StringVal != "" {
				driverIDs[driverID.StringVal] = true
			}
			if warehouseID != "" {
				warehouseIDs[warehouseID] = true
			}
		}

		if rows == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"orders":[]}`))
			return
		}

		// 2. Batch-fetch supplier names
		supplierNames := map[string]string{}
		if len(supplierIDs) > 0 {
			sids := make([]string, 0, len(supplierIDs))
			for id := range supplierIDs {
				sids = append(sids, id)
			}
			snStmt := spanner.Statement{
				SQL:    `SELECT SupplierId, Name FROM Suppliers WHERE SupplierId IN UNNEST(@sids)`,
				Params: map[string]interface{}{"sids": sids},
			}
			snIter := spannerClient.Single().Query(r.Context(), snStmt)
			defer snIter.Stop()
			for {
				snRow, err := snIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var sid, name string
				if err := snRow.Columns(&sid, &name); err == nil {
					supplierNames[sid] = name
				}
			}
		}

		// 3. Batch-fetch warehouse names
		warehouseNames := map[string]string{}
		if len(warehouseIDs) > 0 {
			wids := make([]string, 0, len(warehouseIDs))
			for id := range warehouseIDs {
				wids = append(wids, id)
			}
			whStmt := spanner.Statement{
				SQL:    `SELECT WarehouseId, Name FROM Warehouses WHERE WarehouseId IN UNNEST(@wids)`,
				Params: map[string]interface{}{"wids": wids},
			}
			whIter := spannerClient.Single().Query(r.Context(), whStmt)
			defer whIter.Stop()
			for {
				whRow, err := whIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var wid, name string
				if err := whRow.Columns(&wid, &name); err == nil {
					warehouseNames[wid] = name
				}
			}
		}

		// 4. Read driver positions from Redis GEO (nil-safe)
		driverPositions := map[string][2]float64{} // driverID → [lat, lng]
		if cache.Client != nil && len(driverIDs) > 0 {
			driverSlice := make([]string, 0, len(driverIDs))
			for id := range driverIDs {
				driverSlice = append(driverSlice, id)
			}
			members := make([]string, len(driverSlice))
			for i, id := range driverSlice {
				members[i] = cache.DriverGeoMember(id)
			}
			positions, err := cache.Client.GeoPos(r.Context(), cache.KeyGeoProximity, members...).Result()
			if err == nil {
				for i, id := range driverSlice {
					if i < len(positions) && positions[i] != nil {
						driverPositions[id] = [2]float64{positions[i].Latitude, positions[i].Longitude}
					}
				}
			}
		}

		// 5. Check approaching flags from Redis SET (nil-safe)
		approachingSet := map[string]bool{}
		if cache.Client != nil {
			for _, oid := range orderIDs {
				isApproaching, err := cache.Client.SIsMember(r.Context(), cache.KeyArrivingSet, oid).Result()
				if err == nil && isApproaching {
					approachingSet[oid] = true
				}
			}
		}

		// 6. Hydrate line items
		type trackingItem struct {
			ProductID   string `json:"product_id"`
			ProductName string `json:"product_name"`
			Quantity    int64  `json:"quantity"`
			UnitPrice   int64  `json:"unit_price"`
			LineTotal   int64  `json:"line_total"`
		}
		orderItems := map[string][]trackingItem{}
		if len(orderIDs) > 0 {
			liStmt := spanner.Statement{
				SQL: `SELECT li.OrderId, li.SkuId, COALESCE(sp.Name, li.SkuId) AS SkuName, li.Quantity, li.UnitPrice
				      FROM OrderLineItems li
				      LEFT JOIN SupplierProducts sp ON li.SkuId = sp.SkuId
				      WHERE li.OrderId IN UNNEST(@orderIds)`,
				Params: map[string]interface{}{"orderIds": orderIDs},
			}
			liIter := spannerClient.Single().Query(r.Context(), liStmt)
			defer liIter.Stop()
			for {
				liRow, err := liIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					break
				}
				var oid, skuID, skuName string
				var qty, unitPrice int64
				if err := liRow.Columns(&oid, &skuID, &skuName, &qty, &unitPrice); err == nil {
					orderItems[oid] = append(orderItems[oid], trackingItem{
						ProductID:   skuID,
						ProductName: skuName,
						Quantity:    qty,
						UnitPrice:   unitPrice,
						LineTotal:   qty * unitPrice,
					})
				}
			}
		}

		// 7. Build response
		type trackingOrder struct {
			OrderID         string         `json:"order_id"`
			SupplierID      string         `json:"supplier_id"`
			SupplierName    string         `json:"supplier_name"`
			WarehouseID     string         `json:"warehouse_id,omitempty"`
			WarehouseName   string         `json:"warehouse_name,omitempty"`
			DriverID        string         `json:"driver_id,omitempty"`
			State           string         `json:"state"`
			TotalAmount     int64          `json:"total_amount"`
			OrderSource     string         `json:"order_source,omitempty"`
			DriverLatitude  *float64       `json:"driver_latitude"`
			DriverLongitude *float64       `json:"driver_longitude"`
			IsApproaching   bool           `json:"is_approaching"`
			DeliveryToken   string         `json:"delivery_token,omitempty"`
			CreatedAt       string         `json:"created_at"`
			Items           []trackingItem `json:"items"`
		}

		trackingOrders := make([]trackingOrder, 0, len(rows))
		for _, tr := range rows {
			to := trackingOrder{
				OrderID:       tr.OrderID,
				SupplierID:    tr.SupplierID,
				SupplierName:  supplierNames[tr.SupplierID],
				WarehouseID:   tr.WarehouseID,
				WarehouseName: warehouseNames[tr.WarehouseID],
				DriverID:      tr.DriverID,
				State:         tr.State,
				TotalAmount:   tr.Amount,
				OrderSource:   tr.OrderSource,
				IsApproaching: approachingSet[tr.OrderID],
				DeliveryToken: tr.DeliveryToken,
				CreatedAt:     tr.CreatedAt.Format(time.RFC3339),
				Items:         orderItems[tr.OrderID],
			}
			if pos, ok := driverPositions[tr.DriverID]; ok {
				to.DriverLatitude = &pos[0]
				to.DriverLongitude = &pos[1]
			}
			if to.Items == nil {
				to.Items = []trackingItem{}
			}
			trackingOrders = append(trackingOrders, to)
		}

		resp := map[string]interface{}{
			"orders": trackingOrders,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("[TRACKING] Failed to write response: %v", err)
		}
	})))

	// ── TEMPORARY MIGRATION: Complete Schema Synchronization ────────────────────
	adminClient, adminErr := database.NewDatabaseAdminClient(ctx, opts...)
	if adminErr == nil {
		columnsToDropIn := []string{
			"ALTER TABLE Orders ADD COLUMN Amount INT64",
			"ALTER TABLE Orders ADD COLUMN PaymentGateway STRING(MAX)",
			"ALTER TABLE Orders ADD COLUMN ShopLocation STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN Status STRING(20)",
			"ALTER TABLE Orders ADD COLUMN RouteId STRING(MAX)",
			"ALTER TABLE Orders ADD COLUMN OrderSource STRING(MAX)",
			"ALTER TABLE Orders ADD COLUMN AutoConfirmAt TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN DeliverBefore TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN DeliveryToken STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN ShopName STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN PasswordHash STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN FcmToken STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN TelegramChatId STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN Phone STRING(MAX)",
			"ALTER TABLE Retailers ADD COLUMN Latitude FLOAT64",
			"ALTER TABLE Retailers ADD COLUMN Longitude FLOAT64",
			"ALTER TABLE RetailerSupplierSettings ADD COLUMN AnalyticsStartDate TIMESTAMP",
			"ALTER TABLE RetailerProductSettings ADD COLUMN AnalyticsStartDate TIMESTAMP",
			"ALTER TABLE RetailerVariantSettings ADD COLUMN AnalyticsStartDate TIMESTAMP",
		}

		for _, stmt := range columnsToDropIn {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			} else {
				// Ignore errors for already existing columns
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Cart Fan-Out (Phase 1 — MasterInvoices + Orders supplier columns) ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		fanOutDDL := []string{
			`CREATE TABLE MasterInvoices (
				InvoiceId    STRING(36)  NOT NULL,
				RetailerId   STRING(36)  NOT NULL,
				Total     INT64       NOT NULL,
				State        STRING(20)  NOT NULL,
				CreatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (InvoiceId)`,
			`CREATE INDEX Idx_MasterInvoice_Retailer ON MasterInvoices(RetailerId)`,
			"ALTER TABLE Orders ADD COLUMN InvoiceId STRING(36)",
			"ALTER TABLE Orders ADD COLUMN SupplierId STRING(36)",
			`CREATE INDEX Idx_Orders_InvoiceId ON Orders(InvoiceId)`,
			`CREATE INDEX Idx_Orders_SupplierId ON Orders(SupplierId)`,
			"ALTER TABLE Products ADD COLUMN SupplierId STRING(36)",
			`CREATE INDEX Idx_Products_BySupplierId ON Products(SupplierId)`,
		}
		for _, stmt := range fanOutDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Normalize Order state CHECK constraint to the golden path ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		arrivingDDL := []string{
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED', 'AWAITING_PAYMENT', 'COMPLETED', 'CANCELLED', 'SCHEDULED'))",
			"ALTER TABLE Orders ADD COLUMN QRValidatedAt TIMESTAMP",
			"ALTER TABLE MasterInvoices ADD COLUMN OrderId STRING(36)",
			"CREATE INDEX Idx_MasterInvoice_OrderId ON MasterInvoices(OrderId)",
		}
		for _, stmt := range arrivingDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Empathy Engine — Hierarchical Auto-Order Settings ──────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		empathyDDL := []string{
			`CREATE TABLE RetailerGlobalSettings (
				RetailerId             STRING(36) NOT NULL,
				GlobalAutoOrderEnabled BOOL       NOT NULL,
				UpdatedAt              TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId)`,
			`CREATE TABLE RetailerSupplierSettings (
				RetailerId       STRING(36) NOT NULL,
				SupplierId       STRING(36) NOT NULL,
				AutoOrderEnabled BOOL       NOT NULL,
				UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId, SupplierId),
			  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE`,
			`CREATE TABLE RetailerProductSettings (
				RetailerId       STRING(36) NOT NULL,
				ProductId        STRING(36) NOT NULL,
				AutoOrderEnabled BOOL       NOT NULL,
				UpdatedAt        TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId, ProductId),
			  INTERLEAVE IN PARENT RetailerGlobalSettings ON DELETE CASCADE`,
		}
		for _, stmt := range empathyDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:50]+"...")
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Empathy Engine — Category-level settings ────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		categoryDDL := []string{
			`CREATE TABLE RetailerCategorySettings (
				RetailerId         STRING(36) NOT NULL,
				CategoryId         STRING(50) NOT NULL,
				AutoOrderEnabled   BOOL       NOT NULL,
				AnalyticsStartDate TIMESTAMP,
				UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (RetailerId, CategoryId)`,
			`CREATE INDEX Idx_RetailerCategorySettings_ByRetailer ON RetailerCategorySettings(RetailerId)`,
		}
		for _, stmt := range categoryDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
			}
		}
		adminClient.Close()
	}

	// ── TEMPORARY MIGRATION: The Temporal Brain (AIPredictions) ────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE AIPredictions (
					PredictionId STRING(36) NOT NULL,
					RetailerId STRING(36) NOT NULL,
					PredictedAmount INT64 NOT NULL,
					TriggerDate TIMESTAMP,
					TriggerShard INT64,
					Status STRING(32) NOT NULL,
					CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true),
				) PRIMARY KEY (PredictionId)`,
				`CREATE INDEX Idx_AIPredictions_ByRetailer ON AIPredictions(RetailerId)`,
				`CREATE INDEX Idx_AIPredictions_ByTriggerShardStatusDate ON AIPredictions(TriggerShard, Status, TriggerDate DESC)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: AIPredictions table forged.")
		} else {
			fmt.Printf("DDL migration skipped (table may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Spanner Hotspot Hardening — shard-first time access paths ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		hotspotDDL := []string{
			"ALTER TABLE Orders ADD COLUMN RequestedDeliveryDate TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN ScheduleShard INT64",
			`CREATE INDEX Idx_Orders_ByScheduleShardStateDate ON Orders(ScheduleShard, State, RequestedDeliveryDate DESC)`,
			"DROP INDEX IDX_Orders_Scheduled",
			"ALTER TABLE AIPredictions ADD COLUMN TriggerShard INT64",
			"ALTER TABLE AIPredictions ADD COLUMN CreatedAt TIMESTAMP OPTIONS (allow_commit_timestamp=true)",
			`CREATE INDEX Idx_AIPredictions_ByRetailer ON AIPredictions(RetailerId)`,
			`CREATE INDEX Idx_AIPredictions_ByTriggerShardStatusDate ON AIPredictions(TriggerShard, Status, TriggerDate DESC)`,
			"DROP INDEX Idx_AIPredictions_ByStatus",
			`CREATE TABLE AIPredictionItems (
				PredictionId      STRING(36) NOT NULL,
				PredictionItemId  STRING(36) NOT NULL,
				SkuId             STRING(50) NOT NULL,
				PredictedQuantity INT64      NOT NULL,
				UnitPrice      INT64      NOT NULL,
				CreatedAt         TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (PredictionId, PredictionItemId),
			  INTERLEAVE IN PARENT AIPredictions ON DELETE CASCADE`,
			`CREATE INDEX Idx_PredictionItems_BySku ON AIPredictionItems(SkuId)`,
		}
		for _, stmt := range hotspotDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Inventory Ledger — SupplierInventory table ──────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE SupplierInventory (
					ProductId          STRING(36) NOT NULL,
					SupplierId         STRING(36) NOT NULL,
					QuantityAvailable  INT64      NOT NULL,
					UpdatedAt          TIMESTAMP  OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (ProductId)`,
				`CREATE INDEX Idx_Inventory_BySupplier ON SupplierInventory(SupplierId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: SupplierInventory table forged.")
		} else {
			fmt.Printf("DDL migration skipped (SupplierInventory may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Inventory Audit Log — InventoryAuditLog table ──────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE InventoryAuditLog (
					AuditId      STRING(36)  NOT NULL,
					ProductId    STRING(36)  NOT NULL,
					SupplierId   STRING(36)  NOT NULL,
					AdjustedBy   STRING(36)  NOT NULL,
					PreviousQty  INT64       NOT NULL,
					NewQty       INT64       NOT NULL,
					Delta        INT64       NOT NULL,
					Reason       STRING(50)  NOT NULL,
					AdjustedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (AuditId)`,
				`CREATE INDEX Idx_AuditLog_BySupplier ON InventoryAuditLog(SupplierId)`,
				`CREATE INDEX Idx_AuditLog_ByProduct  ON InventoryAuditLog(ProductId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: InventoryAuditLog table forged.")
		} else {
			fmt.Printf("DDL migration skipped (InventoryAuditLog may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: SupplierReturns — Partial-Qty Reconciliation (Phase 9) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`CREATE TABLE SupplierReturns (
					ReturnId     STRING(36)  NOT NULL,
					OrderId      STRING(36)  NOT NULL,
					SkuId        STRING(50)  NOT NULL,
					RejectedQty  INT64       NOT NULL,
					Reason       STRING(50)  NOT NULL,
					DriverNotes  STRING(MAX),
					CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (ReturnId)`,
				`CREATE INDEX Idx_Returns_ByOrder ON SupplierReturns(OrderId)`,
				`CREATE INDEX Idx_Returns_BySku   ON SupplierReturns(SkuId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: SupplierReturns table forged.")
		} else {
			fmt.Printf("DDL migration skipped (SupplierReturns may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Drivers Fleet Provisioning — Phone, PIN, Supplier columns (Phase 10) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		fleetDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN Phone STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN PinHash STRING(MAX)",
			"ALTER TABLE Drivers ADD COLUMN SupplierId STRING(36)",
			"ALTER TABLE Drivers ADD COLUMN DriverType STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN VehicleType STRING(50)",
			"ALTER TABLE Drivers ADD COLUMN LicensePlate STRING(30)",
			"ALTER TABLE Drivers ADD COLUMN IsActive BOOL",
			`CREATE INDEX Idx_Drivers_BySupplierId ON Drivers(SupplierId)`,
			`CREATE INDEX Idx_Drivers_ByPhone ON Drivers(Phone)`,
		}
		for _, stmt := range fleetDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Vehicles Table + Drivers extra columns (Phase 10b) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		vehicleDDL := []string{
			`CREATE TABLE Vehicles (
				VehicleId    STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				VehicleClass STRING(10)  NOT NULL,
				Label        STRING(100),
				LicensePlate STRING(30),
				MaxVolumeVU  FLOAT64     NOT NULL,
				IsActive     BOOL        NOT NULL DEFAULT (true),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (VehicleId)`,
			`CREATE INDEX Idx_Vehicles_BySupplier ON Vehicles(SupplierId)`,
			"ALTER TABLE Drivers ADD COLUMN VehicleId STRING(36)",
			"ALTER TABLE Drivers ADD COLUMN TruckStatus STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN DepartedAt TIMESTAMP",
			"ALTER TABLE Drivers ADD COLUMN MaxPalletCapacity INT64",
		}
		for _, stmt := range vehicleDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Payment Settlement — GlobalPayTransactionId on MasterInvoices ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database: dbName,
			Statements: []string{
				`ALTER TABLE MasterInvoices ADD COLUMN GlobalPayTransactionId STRING(64)`,
				`CREATE INDEX Idx_MasterInvoice_GlobalPayTxn ON MasterInvoices(GlobalPayTransactionId)`,
			},
		})
		if ddlErr == nil {
			op.Wait(ctx)
			fmt.Println("DATABASE MIGRATION SUCCESS: GlobalPayTransactionId column added to MasterInvoices.")
		} else {
			fmt.Printf("DDL migration skipped (GlobalPayTransactionId may already exist): %v\n", ddlErr)
		}
		adminClient.Close()
	}

	// ── MIGRATION: Optimistic Concurrency Control & Freeze Locks (Phase 12) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		occDDL := []string{
			"ALTER TABLE Orders ADD COLUMN Version INT64 NOT NULL DEFAULT (1)",
			"ALTER TABLE Orders ADD COLUMN LockedUntil TIMESTAMP",
		}
		for _, stmt := range occDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier Registration Pipeline — TaxId, IsConfigured, OperatingCategories, PlatformCategories ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		supplierRegDDL := []string{
			"ALTER TABLE Suppliers ADD COLUMN TaxId STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN IsConfigured BOOL",
			"ALTER TABLE Suppliers ADD COLUMN OperatingCategories ARRAY<STRING(MAX)>",
			`CREATE TABLE PlatformCategories (
				CategoryId    STRING(36)  NOT NULL,
				DisplayName   STRING(MAX) NOT NULL,
				IconUrl       STRING(MAX),
				DisplayOrder  INT64       NOT NULL DEFAULT (0)
			) PRIMARY KEY (CategoryId)`,
			`CREATE TABLE Categories (
				CategoryId   STRING(36)  NOT NULL,
				Name         STRING(255) NOT NULL,
				Icon         STRING(100),
				SortOrder    INT64       NOT NULL DEFAULT (0),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (CategoryId)`,
		}
		for _, stmt := range supplierRegDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier Extended Profile Columns (Email, Bank, Payment) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		supplierProfileDDL := []string{
			"ALTER TABLE Suppliers ADD COLUMN Email STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN ContactPerson STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN CompanyRegNumber STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN BillingAddress STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN BankName STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN AccountNumber STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN CardNumber STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN PaymentGateway STRING(20)",
		}
		for _, stmt := range supplierProfileDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Admins Table ────────────────────────────────────
	{
		adminClient, adminErr := database.NewDatabaseAdminClient(ctx)
		if adminErr == nil {
			adminsDDL := []string{
				`CREATE TABLE Admins (
					AdminId       STRING(36)  NOT NULL,
					Email         STRING(MAX) NOT NULL,
					PasswordHash  STRING(MAX) NOT NULL,
					DisplayName   STRING(MAX),
					CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
				) PRIMARY KEY (AdminId)`,
				`CREATE UNIQUE INDEX Idx_Admins_ByEmail ON Admins(Email)`,
			}
			for _, stmt := range adminsDDL {
				op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
					Database:   dbName,
					Statements: []string{stmt},
				})
				if ddlErr == nil {
					op.Wait(ctx)
					preview := stmt
					if len(preview) > 60 {
						preview = preview[:60] + "..."
					}
					fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
				}
			}
			adminClient.Close()
		}
	}

	// Seed default admin account if none exist
	auth.SeedDefaultAdmin(ctx, spannerClient)

	// ── MIGRATION: Truck State Machine — TruckStatus column on Drivers (Fleet Availability) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		truckStatusDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN TruckStatus STRING(20) DEFAULT ('AVAILABLE')",
			`CREATE INDEX Idx_Drivers_ByTruckStatus ON Drivers(TruckStatus)`,
			"ALTER TABLE Drivers ADD COLUMN DepartedAt TIMESTAMP",
		}
		for _, stmt := range truckStatusDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: SupplierProducts CategoryId + CategoryName, PricingTiers, Warehouse columns ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		catalogPricingDDL := []string{
			"ALTER TABLE SupplierProducts ADD COLUMN CategoryId STRING(36)",
			"ALTER TABLE SupplierProducts ADD COLUMN CategoryName STRING(MAX)",
			"ALTER TABLE SupplierProducts ADD COLUMN PalletFootprint FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN VolumetricUnit FLOAT64",
			"ALTER TABLE Suppliers ADD COLUMN WarehouseLocation STRING(MAX)",
			"ALTER TABLE Suppliers ADD COLUMN WarehouseLat FLOAT64",
			"ALTER TABLE Suppliers ADD COLUMN WarehouseLng FLOAT64",
			`CREATE TABLE PricingTiers (
				TierId              STRING(36)  NOT NULL,
				SupplierId          STRING(36)  NOT NULL,
				SkuId               STRING(50)  NOT NULL,
				MinPallets          INT64       NOT NULL,
				DiscountPct         INT64       NOT NULL,
				TargetRetailerTier  STRING(20)  NOT NULL,
				ValidUntil          TIMESTAMP,
				IsActive            BOOL        NOT NULL
			) PRIMARY KEY (TierId)`,
			`CREATE INDEX Idx_PricingTiers_BySupplierId ON PricingTiers(SupplierId)`,
		}
		for _, stmt := range catalogPricingDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				fmt.Println("DATABASE MIGRATION SUCCESS:", preview)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Warehouse Staff (Payloader) Provisioning ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		warehouseStaffDDL := []string{
			`CREATE TABLE WarehouseStaff (
				WorkerId    STRING(36)  NOT NULL,
				SupplierId  STRING(36)  NOT NULL,
				Name        STRING(MAX) NOT NULL,
				Phone       STRING(20)  NOT NULL,
				PinHash     STRING(MAX) NOT NULL,
				IsActive    BOOL        NOT NULL,
				CreatedAt   TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (WorkerId)`,
			`CREATE INDEX Idx_WarehouseStaff_BySupplierId ON WarehouseStaff(SupplierId)`,
			`CREATE INDEX Idx_WarehouseStaff_ByPhone ON WarehouseStaff(Phone)`,
		}
		for _, stmt := range warehouseStaffDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Dimensional VU Engine + Registration Expansion ────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		dimensionalDDL := []string{
			"ALTER TABLE Vehicles ADD COLUMN LengthCM FLOAT64",
			"ALTER TABLE Vehicles ADD COLUMN WidthCM FLOAT64",
			"ALTER TABLE Vehicles ADD COLUMN HeightCM FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN LengthCM FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN WidthCM FLOAT64",
			"ALTER TABLE SupplierProducts ADD COLUMN HeightCM FLOAT64",
			"ALTER TABLE Retailers ADD COLUMN ReceivingWindowOpen STRING(10)",
			"ALTER TABLE Retailers ADD COLUMN ReceivingWindowClose STRING(10)",
			"ALTER TABLE Retailers ADD COLUMN AccessType STRING(30)",
			"ALTER TABLE Retailers ADD COLUMN StorageCeilingHeightCM FLOAT64",
			"ALTER TABLE Suppliers ADD COLUMN FleetColdChainCompliant BOOL",
			"ALTER TABLE Suppliers ADD COLUMN PalletizationStandard STRING(30)",
		}
		for _, stmt := range dimensionalDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Concurrency Crash — BACKORDERED state for partial-fill checkout ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		backorderDDL := []string{
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'BACKORDERED'))",
		}
		for _, stmt := range backorderDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phantom Cargo — RejectedQty + ReturnClearedAt on OrderLineItems ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phantooCargoDDL := []string{
			"ALTER TABLE OrderLineItems ADD COLUMN RejectedQty INT64 NOT NULL DEFAULT (0)",
			"ALTER TABLE OrderLineItems ADD COLUMN ReturnClearedAt TIMESTAMP",
		}
		for _, stmt := range phantooCargoDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: UOM Collision — MinimumOrderQty + StepSize on SupplierProducts ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		uomDDL := []string{
			"ALTER TABLE SupplierProducts ADD COLUMN MinimumOrderQty INT64 NOT NULL DEFAULT (1)",
			"ALTER TABLE SupplierProducts ADD COLUMN StepSize INT64 NOT NULL DEFAULT (1)",
		}
		for _, stmt := range uomDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Supplier Shift State — OperatingSchedule + ManualOffShift ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		shiftDDL := []string{
			"ALTER TABLE Suppliers ADD COLUMN OperatingSchedule JSON",
			"ALTER TABLE Suppliers ADD COLUMN ManualOffShift BOOL NOT NULL DEFAULT (false)",
		}
		for _, stmt := range shiftDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Cash Logistics — PENDING_CASH_COLLECTION state + MasterInvoices cash-custody columns ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		cashLogisticsDDL := []string{
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN ('PENDING', 'PENDING_REVIEW', 'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION', 'COMPLETED', 'CANCELLED', 'SCHEDULED', 'BACKORDERED', 'QUARANTINE'))",
			"ALTER TABLE MasterInvoices ADD COLUMN PaymentMode STRING(20)",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectorDriverId STRING(36)",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectedAt TIMESTAMP",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectionLat FLOAT64",
			"ALTER TABLE MasterInvoices ADD COLUMN CollectionLng FLOAT64",
			"ALTER TABLE MasterInvoices ADD COLUMN GeofenceDistanceM FLOAT64",
			"ALTER TABLE MasterInvoices ADD COLUMN CustodyStatus STRING(20)",
		}
		for _, stmt := range cashLogisticsDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Multi-vendor Payment — PaymentStatus column + SupplierPaymentConfigs table ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		multiVendorPaymentDDL := []string{
			"ALTER TABLE Orders ADD COLUMN PaymentStatus STRING(30) NOT NULL DEFAULT ('PENDING')",
			"ALTER TABLE Orders ADD CONSTRAINT CHK_PaymentStatus CHECK (PaymentStatus IN ('PENDING', 'PENDING_CASH_COLLECTION', 'AWAITING_GATEWAY_WEBHOOK', 'PAID', 'FAILED'))",
			`CREATE TABLE SupplierPaymentConfigs (
				ConfigId     STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				GatewayName  STRING(20)  NOT NULL,
				MerchantId   STRING(MAX) NOT NULL,
				ServiceId    STRING(MAX),
				SecretKey    BYTES(MAX)  NOT NULL,
				IsActive     BOOL        NOT NULL DEFAULT (true),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'GLOBAL_PAY'))
			) PRIMARY KEY (ConfigId)`,
			"CREATE INDEX Idx_SupplierPaymentConfigs_BySupplierId ON SupplierPaymentConfigs(SupplierId)",
			"CREATE UNIQUE INDEX Idx_SupplierPaymentConfigs_Unique ON SupplierPaymentConfigs(SupplierId, GatewayName)",
			// Phase 2 addendum: ServiceId for Cash gateway
			"ALTER TABLE SupplierPaymentConfigs ADD COLUMN ServiceId STRING(MAX)",
			"ALTER TABLE SupplierPaymentConfigs DROP CONSTRAINT CHK_GatewayName",
			"ALTER TABLE SupplierPaymentConfigs ADD CONSTRAINT CHK_GatewayName CHECK (GatewayName IN ('CASH', 'GLOBAL_PAY', 'GLOBAL_PAY'))",
		}
		for _, stmt := range multiVendorPaymentDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Payment Sessions + Attempts tables (Phase 13) ──────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		paymentSessionDDL := []string{
			`CREATE TABLE PaymentSessions (
				SessionId         STRING(36)  NOT NULL,
				OrderId           STRING(36)  NOT NULL,
				RetailerId        STRING(36)  NOT NULL,
				SupplierId        STRING(36)  NOT NULL,
				Gateway           STRING(20)  NOT NULL,
				LockedAmount   INT64       NOT NULL,
				Currency          STRING(3)   NOT NULL DEFAULT ('UZS'),
				Status            STRING(30)  NOT NULL DEFAULT ('CREATED'),
				CurrentAttemptNo  INT64       NOT NULL DEFAULT (0),
				InvoiceId         STRING(36),
				RedirectUrl       STRING(MAX),
				ProviderReference STRING(MAX),
				ExpiresAt         TIMESTAMP,
				LastErrorCode     STRING(50),
				LastErrorMessage  STRING(MAX),
				CreatedAt         TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt         TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				SettledAt         TIMESTAMP,
				CONSTRAINT CHK_SessionStatus CHECK (Status IN ('CREATED', 'PENDING', 'SETTLED', 'FAILED', 'EXPIRED', 'CANCELLED'))
			) PRIMARY KEY (SessionId)`,
			"CREATE INDEX Idx_PaymentSessions_ByOrderId ON PaymentSessions(OrderId)",
			"CREATE INDEX Idx_PaymentSessions_BySupplierId ON PaymentSessions(SupplierId)",
			"CREATE INDEX Idx_PaymentSessions_ByStatus ON PaymentSessions(Status)",
			"ALTER TABLE PaymentSessions ADD COLUMN ProviderReference STRING(MAX)",
			`CREATE TABLE PaymentAttempts (
				AttemptId             STRING(36)  NOT NULL,
				SessionId             STRING(36)  NOT NULL,
				AttemptNo             INT64       NOT NULL,
				Gateway               STRING(20)  NOT NULL,
				ProviderTransactionId STRING(64),
				Status                STRING(30)  NOT NULL DEFAULT ('INITIATED'),
				FailureCode           STRING(50),
				FailureMessage        STRING(MAX),
				RequestDigest         STRING(MAX),
				StartedAt             TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				FinishedAt            TIMESTAMP,
				CONSTRAINT CHK_AttemptStatus CHECK (Status IN ('INITIATED', 'REDIRECTED', 'PROCESSING', 'SUCCESS', 'FAILED', 'CANCELLED', 'TIMED_OUT'))
			) PRIMARY KEY (AttemptId)`,
			"CREATE INDEX Idx_PaymentAttempts_BySessionId ON PaymentAttempts(SessionId)",
			"CREATE INDEX Idx_PaymentAttempts_ByProviderTxn ON PaymentAttempts(ProviderTransactionId)",
		}
		for _, stmt := range paymentSessionDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Gateway Onboarding Sessions (Supplier Connect) ──────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		onboardingDDL := []string{
			`CREATE TABLE GatewayOnboardingSessions (
				SessionId      STRING(36)  NOT NULL,
				SupplierId     STRING(36)  NOT NULL,
				Gateway        STRING(20)  NOT NULL,
				Status         STRING(30)  NOT NULL DEFAULT ('CREATED'),
				StateNonce     STRING(128),
				ReturnSurface  STRING(10)  NOT NULL DEFAULT ('web'),
				RedirectUrl    STRING(MAX),
				ErrorMessage   STRING(MAX),
				ExpiresAt      TIMESTAMP   NOT NULL,
				CreatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt      TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_OnboardStatus CHECK (Status IN ('CREATED', 'PENDING', 'COMPLETED', 'FAILED', 'CANCELLED', 'EXPIRED'))
			) PRIMARY KEY (SessionId)`,
			"CREATE INDEX Idx_GatewayOnboarding_BySupplierId ON GatewayOnboardingSessions(SupplierId)",
			"CREATE INDEX Idx_GatewayOnboarding_ByStatus ON GatewayOnboardingSessions(Status)",
		}
		for _, stmt := range onboardingDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Firebase Auth Identity Linking ─────────────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		firebaseUidDDL := []string{
			"ALTER TABLE Admins ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE Suppliers ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE Retailers ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE Drivers ADD COLUMN FirebaseUid STRING(128)",
			"ALTER TABLE WarehouseStaff ADD COLUMN FirebaseUid STRING(128)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Admins_ByFirebaseUid ON Admins(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Suppliers_ByFirebaseUid ON Suppliers(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Retailers_ByFirebaseUid ON Retailers(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_Drivers_ByFirebaseUid ON Drivers(FirebaseUid)",
			"CREATE UNIQUE NULL_FILTERED INDEX Idx_WarehouseStaff_ByFirebaseUid ON WarehouseStaff(FirebaseUid)",
		}
		for _, stmt := range firebaseUidDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Driver Availability Session Management ────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		driverSessionDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN OfflineReason STRING(30)",
			"ALTER TABLE Drivers ADD COLUMN OfflineReasonNote STRING(500)",
			"ALTER TABLE Drivers ADD COLUMN OfflineAt TIMESTAMP",
			"CREATE INDEX Idx_Drivers_ByActiveStatus ON Drivers(SupplierId, IsActive)",
		}
		for _, stmt := range driverSessionDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Geo-Spatial Sovereignty — H3Index on Retailers, Factories & Orders ────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		geoH3DDL := []string{
			"ALTER TABLE Retailers ADD COLUMN H3Index STRING(MAX)",
			"CREATE INDEX Idx_Retailers_ByH3Index ON Retailers(H3Index)",
			"ALTER TABLE Factories ADD COLUMN H3Index STRING(MAX)",
			"CREATE INDEX Idx_Factories_ByH3Index ON Factories(H3Index)",
			"ALTER TABLE Orders ADD COLUMN H3Index STRING(MAX)",
			"CREATE INDEX Idx_Orders_ByH3Index ON Orders(H3Index)",
		}
		for _, stmt := range geoH3DDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── BACKFILL: H3Index for rows created before the H3 migration ────────────
	// Runs once per boot, a no-op after the first successful pass (all rows
	// already have H3Index populated). Uses h3-go/v4 at resolution 7 to emit
	// 15-char lowercase hex cell IDs compatible with h3-js on the frontend.
	backfillH3Indexes(ctx, spannerClient)

	// ── MIGRATION: Address-as-Label — AddressVerified flag ──────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		addrDDL := []string{
			"ALTER TABLE Retailers ADD COLUMN AddressVerified BOOL",
			"ALTER TABLE Warehouses ADD COLUMN AddressVerified BOOL",
		}
		for _, stmt := range addrDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Warehouse Load Balancing — MaxCapacityThreshold + composite order index ──
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		loadBalanceDDL := []string{
			"ALTER TABLE Warehouses ADD COLUMN MaxCapacityThreshold INT64",
			"CREATE INDEX Idx_Orders_ByWarehouseStateCreated ON Orders(WarehouseId, State, CreatedAt DESC)",
		}
		for _, stmt := range loadBalanceDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt)
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase E — Warehouses, SupplierUsers, Factories (CREATE TABLE) ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseEDDL := []string{
			// ── Warehouses ──
			`CREATE TABLE Warehouses (
				WarehouseId      STRING(36)  NOT NULL,
				SupplierId       STRING(36)  NOT NULL,
				Name             STRING(255) NOT NULL,
				Address          STRING(MAX),
				Lat              FLOAT64,
				Lng              FLOAT64,
				H3Indexes        ARRAY<STRING(MAX)>,
				CoverageRadiusKm FLOAT64     NOT NULL DEFAULT (50.0),
				IsActive         BOOL        NOT NULL DEFAULT (true),
				IsDefault        BOOL        NOT NULL DEFAULT (false),
				IsOnShift        BOOL        NOT NULL DEFAULT (true),
				CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt        TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (WarehouseId)`,
			`CREATE INDEX Idx_Warehouses_BySupplierId ON Warehouses(SupplierId)`,

			// ── SupplierUsers (RBAC) ──
			`CREATE TABLE SupplierUsers (
				UserId               STRING(36)  NOT NULL,
				SupplierId           STRING(36)  NOT NULL,
				Email                STRING(MAX),
				Phone                STRING(20),
				Name                 STRING(MAX) NOT NULL,
				PasswordHash         STRING(MAX) NOT NULL,
				SupplierRole         STRING(30)  NOT NULL,
				AssignedWarehouseId  STRING(36),
				AssignedFactoryId    STRING(36),
				IsActive             BOOL        NOT NULL DEFAULT (true),
				FirebaseUid          STRING(128),
				CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_SupplierRole CHECK (SupplierRole IN ('GLOBAL_ADMIN', 'NODE_ADMIN', 'FACTORY_ADMIN', 'FACTORY_PAYLOADER'))
			) PRIMARY KEY (UserId)`,
			`CREATE INDEX Idx_SupplierUsers_BySupplierId ON SupplierUsers(SupplierId)`,
			`CREATE INDEX Idx_SupplierUsers_ByPhone ON SupplierUsers(Phone)`,
			`CREATE UNIQUE NULL_FILTERED INDEX Idx_SupplierUsers_ByFirebaseUid ON SupplierUsers(FirebaseUid)`,

			// ── Factories ──
			`CREATE TABLE Factories (
				FactoryId            STRING(36)  NOT NULL,
				SupplierId           STRING(36)  NOT NULL,
				Name                 STRING(255) NOT NULL,
				Address              STRING(MAX),
				Lat                  FLOAT64,
				Lng                  FLOAT64,
				RegionCode           STRING(20),
				LeadTimeDays         INT64       NOT NULL DEFAULT (2),
				ProductionCapacityVU FLOAT64     NOT NULL DEFAULT (0),
				IsActive             BOOL        NOT NULL DEFAULT (true),
				CreatedAt            TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt            TIMESTAMP   OPTIONS (allow_commit_timestamp=true)
			) PRIMARY KEY (FactoryId)`,
			`CREATE INDEX Idx_Factories_BySupplierId ON Factories(SupplierId)`,

			// ── FactoryStaff ──
			`CREATE TABLE FactoryStaff (
				StaffId      STRING(36)  NOT NULL,
				FactoryId    STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				Name         STRING(MAX) NOT NULL,
				Phone        STRING(20),
				PasswordHash STRING(MAX) NOT NULL,
				StaffRole    STRING(30)  NOT NULL,
				IsActive     BOOL        NOT NULL DEFAULT (true),
				FirebaseUid  STRING(128),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_FactoryStaffRole CHECK (StaffRole IN ('FACTORY_ADMIN', 'FACTORY_PAYLOADER'))
			) PRIMARY KEY (StaffId)`,
			`CREATE INDEX Idx_FactoryStaff_ByFactoryId ON FactoryStaff(FactoryId)`,
			`CREATE INDEX Idx_FactoryStaff_ByPhone ON FactoryStaff(Phone)`,
			`CREATE UNIQUE NULL_FILTERED INDEX Idx_FactoryStaff_ByFirebaseUid ON FactoryStaff(FirebaseUid)`,

			// ── InternalTransferOrders ──
			`CREATE TABLE InternalTransferOrders (
				TransferId   STRING(36)  NOT NULL,
				FactoryId    STRING(36)  NOT NULL,
				WarehouseId  STRING(36)  NOT NULL,
				SupplierId   STRING(36)  NOT NULL,
				State        STRING(20)  NOT NULL DEFAULT ('DRAFT'),
				TotalVolumeVU FLOAT64    NOT NULL DEFAULT (0),
				ManifestId   STRING(36),
				Source       STRING(30)  NOT NULL DEFAULT ('MANUAL_EMERGENCY'),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				UpdatedAt    TIMESTAMP   OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_TransferState CHECK (State IN ('DRAFT', 'APPROVED', 'LOADING', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVED', 'RECEIVED', 'CANCELLED')),
				CONSTRAINT CHK_TransferSource CHECK (Source IN ('SYSTEM_THRESHOLD', 'SYSTEM_PREDICTED', 'MANUAL_EMERGENCY'))
			) PRIMARY KEY (TransferId)`,
			`CREATE INDEX Idx_Transfers_ByFactoryId ON InternalTransferOrders(FactoryId)`,
			`CREATE INDEX Idx_Transfers_ByWarehouseId ON InternalTransferOrders(WarehouseId)`,
			`CREATE INDEX Idx_Transfers_BySupplierId ON InternalTransferOrders(SupplierId)`,
			`CREATE INDEX Idx_Transfers_ByState ON InternalTransferOrders(State)`,

			// ── InternalTransferItems (interleaved) ──
			`CREATE TABLE InternalTransferItems (
				TransferId STRING(36) NOT NULL,
				ItemId     STRING(36) NOT NULL,
				ProductId  STRING(36) NOT NULL,
				Quantity   INT64      NOT NULL,
				VolumeVU   FLOAT64    NOT NULL DEFAULT (0)
			) PRIMARY KEY (TransferId, ItemId),
			  INTERLEAVE IN PARENT InternalTransferOrders ON DELETE CASCADE`,

			// ── FactoryTruckManifests ──
			`CREATE TABLE FactoryTruckManifests (
				ManifestId   STRING(36)  NOT NULL,
				FactoryId    STRING(36)  NOT NULL,
				DriverId     STRING(36),
				VehicleId    STRING(36),
				State        STRING(20)  NOT NULL DEFAULT ('PENDING'),
				TotalVolumeVU FLOAT64    NOT NULL DEFAULT (0),
				MaxVolumeVU  FLOAT64     NOT NULL DEFAULT (0),
				StopCount    INT64       NOT NULL DEFAULT (0),
				RegionCode   STRING(20),
				RoutePath    STRING(MAX),
				CreatedAt    TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_ManifestState CHECK (State IN ('PENDING', 'READY_FOR_LOADING', 'LOADING', 'DISPATCHED', 'COMPLETED'))
			) PRIMARY KEY (ManifestId)`,
			`CREATE INDEX Idx_FactoryManifests_ByFactoryId ON FactoryTruckManifests(FactoryId)`,
			`CREATE INDEX Idx_FactoryManifests_ByState ON FactoryTruckManifests(State)`,

			// ── ReplenishmentInsights ──
			`CREATE TABLE ReplenishmentInsights (
				InsightId        STRING(36)  NOT NULL,
				WarehouseId      STRING(36)  NOT NULL,
				ProductId        STRING(36)  NOT NULL,
				SupplierId       STRING(36)  NOT NULL,
				CurrentStock     INT64       NOT NULL DEFAULT (0),
				DailyBurnRate    FLOAT64     NOT NULL DEFAULT (0),
				TimeToEmptyDays  FLOAT64     NOT NULL DEFAULT (0),
				SuggestedQuantity INT64      NOT NULL DEFAULT (0),
				UrgencyLevel     STRING(20)  NOT NULL DEFAULT ('STABLE'),
				ReasonCode       STRING(30)  NOT NULL DEFAULT ('LOW_STOCK'),
				Status           STRING(20)  NOT NULL DEFAULT ('PENDING'),
				TargetFactoryId  STRING(36),
				DemandBreakdown  STRING(MAX),
				CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				CONSTRAINT CHK_InsightUrgency CHECK (UrgencyLevel IN ('CRITICAL', 'WARNING', 'STABLE')),
				CONSTRAINT CHK_InsightReason CHECK (ReasonCode IN ('HIGH_VELOCITY', 'LOW_STOCK', 'PREDICTED_SPIKE')),
				CONSTRAINT CHK_InsightStatus CHECK (Status IN ('PENDING', 'APPROVED', 'DISMISSED'))
			) PRIMARY KEY (InsightId)`,
			`CREATE INDEX Idx_Insights_ByWarehouse ON ReplenishmentInsights(WarehouseId)`,
			`CREATE INDEX Idx_Insights_BySupplierId ON ReplenishmentInsights(SupplierId)`,
			`CREATE INDEX Idx_Insights_ByStatus ON ReplenishmentInsights(Status)`,

			// ── Warehouse linkage to operational tables ──
			"ALTER TABLE Drivers ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE Vehicles ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE WarehouseStaff ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE SupplierInventory ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE InventoryAuditLog ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE Orders ADD COLUMN WarehouseId STRING(36)",
			"ALTER TABLE RetailerCarts ADD COLUMN WarehouseId STRING(36)",
			"CREATE INDEX Idx_Drivers_ByWarehouseId ON Drivers(WarehouseId)",
			"CREATE INDEX Idx_Vehicles_ByWarehouseId ON Vehicles(WarehouseId)",
			"CREATE INDEX Idx_WarehouseStaff_ByWarehouseId ON WarehouseStaff(WarehouseId)",
			"CREATE INDEX Idx_Inventory_ByWarehouseId ON SupplierInventory(SupplierId, WarehouseId)",
			"CREATE INDEX Idx_Orders_ByWarehouseId ON Orders(WarehouseId)",
			"ALTER TABLE AIPredictions ADD COLUMN WarehouseId STRING(36)",
			"CREATE INDEX Idx_AIPredictions_ByWarehouse ON AIPredictions(WarehouseId)",

			// ── Warehouse-Factory linkage ──
			"ALTER TABLE Warehouses ADD COLUMN PrimaryFactoryId STRING(36)",
			"ALTER TABLE Warehouses ADD COLUMN SecondaryFactoryId STRING(36)",
		}
		for _, stmt := range phaseEDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase IV — Pre-order policy columns + Order state expansion ─────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseIVDDL := []string{
			"ALTER TABLE Orders ADD COLUMN CancelLockedAt TIMESTAMP",
			"ALTER TABLE Orders ADD COLUMN CancelLockReason STRING(30)",
			"ALTER TABLE Orders ADD COLUMN ConfirmationNotifiedAt TIMESTAMP",
			`CREATE INDEX Idx_Orders_PreOrderLockPending
				ON Orders(State, RequestedDeliveryDate)
				WHERE CancelLockedAt IS NULL AND ConfirmationNotifiedAt IS NULL
				  AND State IN ('SCHEDULED', 'PENDING_REVIEW')`,
		}
		for _, stmt := range phaseIVDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase V — Temporal Traceability & Notification Correlation ───
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseVDDL := []string{
			// 1. Orders.ReplenishmentId — forward-link from replenishment transfer to fulfilled orders
			"ALTER TABLE Orders ADD COLUMN ReplenishmentId STRING(36)",
			`CREATE INDEX Idx_Orders_ByReplenishmentId ON Orders(ReplenishmentId) WHERE ReplenishmentId IS NOT NULL`,

			// 2. Expand Order state machine — add PENDING_CONFIRMATION, LOCKED, AUTO_ACCEPTED,
			//    NO_CAPACITY, STALE_AUDIT (last two already used in application code but unconstrained)
			"ALTER TABLE Orders DROP CONSTRAINT CHK_Order_State",
			`ALTER TABLE Orders ADD CONSTRAINT CHK_Order_State CHECK (State IN (
				'PENDING', 'PENDING_REVIEW', 'PENDING_CONFIRMATION',
				'LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED',
				'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION',
				'COMPLETED', 'CANCELLED',
				'SCHEDULED', 'BACKORDERED', 'QUARANTINE',
				'LOCKED', 'AUTO_ACCEPTED',
				'NO_CAPACITY', 'STALE_AUDIT'
			))`,

			// 3. Notifications.ExpiresAt — soft-expiry for stale alerts
			"ALTER TABLE Notifications ADD COLUMN ExpiresAt TIMESTAMP",

			// 4. Notifications.CorrelationId — links notification to triggering entity (e.g. ord_confirm_{OrderId})
			"ALTER TABLE Notifications ADD COLUMN CorrelationId STRING(36)",
			`CREATE INDEX Idx_Notifications_ByCorrelationId ON Notifications(CorrelationId) WHERE CorrelationId IS NOT NULL`,
			`CREATE INDEX Idx_Notifications_ByExpiresAt ON Notifications(ExpiresAt) WHERE ExpiresAt IS NOT NULL AND ReadAt IS NULL`,
		}
		for _, stmt := range phaseVDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase V.5 — Look-Ahead Source Type ───────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		lookAheadDDL := []string{
			"ALTER TABLE InternalTransferOrders DROP CONSTRAINT CHK_TransferSource",
			`ALTER TABLE InternalTransferOrders ADD CONSTRAINT CHK_TransferSource
				CHECK (Source IN ('SYSTEM_THRESHOLD', 'SYSTEM_PREDICTED', 'MANUAL_EMERGENCY', 'WAREHOUSE_REQUEST', 'SYSTEM_LOOKAHEAD'))`,
		}
		for _, stmt := range lookAheadDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase VI — Fleet Offline State ────────────────────────────────
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseVIDDL := []string{
			// 1. Drivers.IsOffline — distinguishes "temporarily offline" (app backgrounded, phone dead)
			//    from "account deactivated" (IsActive=false). Dispatch queries check both.
			"ALTER TABLE Drivers ADD COLUMN IsOffline BOOL DEFAULT (false)",
			`CREATE INDEX Idx_Drivers_ByOffline ON Drivers(SupplierId, IsOffline) WHERE IsOffline = true`,

			// 2. Orders.NudgeNotifiedAt — dedup marker for T-5 soft reminder
			"ALTER TABLE Orders ADD COLUMN NudgeNotifiedAt TIMESTAMP",
		}
		for _, stmt := range phaseVIDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}
		adminClient.Close()
	}

	// ── MIGRATION: Phase VII — V.O.I.D. Home Node Lifecycle ─────────────────────
	// Drivers/Vehicles carry a canonical (HomeNodeType, HomeNodeId) tuple so a
	// resource can be home-based at a Warehouse OR a Factory. WarehouseId stays
	// denormalised during the migration window; new writes dual-populate.
	adminClient, err = database.NewDatabaseAdminClient(ctx, opts...)
	if err == nil {
		phaseVIIDDL := []string{
			"ALTER TABLE Drivers ADD COLUMN HomeNodeType STRING(20)",
			"ALTER TABLE Drivers ADD COLUMN HomeNodeId STRING(36)",
			"ALTER TABLE Vehicles ADD COLUMN HomeNodeType STRING(20)",
			"ALTER TABLE Vehicles ADD COLUMN HomeNodeId STRING(36)",
			"CREATE INDEX Idx_Drivers_ByHomeNode ON Drivers(HomeNodeType, HomeNodeId)",
			"CREATE INDEX Idx_Vehicles_ByHomeNode ON Vehicles(HomeNodeType, HomeNodeId)",
			// Transactional Outbox — single mechanism for durable state-change
			// Kafka events. Entity mutations dual-write to OutboxEvents inside
			// the same ReadWriteTransaction; the relay tails this table and
			// publishes. See backend-go/outbox/.
			`CREATE TABLE OutboxEvents (
				EventId       STRING(36)  NOT NULL,
				AggregateType STRING(30)  NOT NULL,
				AggregateId   STRING(36)  NOT NULL,
				TopicName     STRING(100) NOT NULL,
				Payload       BYTES(MAX)  NOT NULL,
				CreatedAt     TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
				PublishedAt   TIMESTAMP,
			) PRIMARY KEY (EventId)`,
			`CREATE INDEX Idx_OutboxEvents_Unpublished ON OutboxEvents(CreatedAt) WHERE PublishedAt IS NULL`,
		}
		for _, stmt := range phaseVIIDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}

		// ── Glass Box: OutboxEvents.TraceID for end-to-end trace propagation ──
		glassBoxDDL := []string{
			"ALTER TABLE OutboxEvents ADD COLUMN TraceID STRING(36)",
		}
		for _, stmt := range glassBoxDDL {
			op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
				Database:   dbName,
				Statements: []string{stmt},
			})
			if ddlErr == nil {
				op.Wait(ctx)
				fmt.Println("DATABASE MIGRATION SUCCESS:", stmt[:minInt(80, len(stmt))])
			}
		}

		adminClient.Close()
	}

	fleet.AvailabilityEmitter = func(driverID, supplierID string, available bool, reason, note, truckID string) {
		// 1. Kafka event
		svc.PublishEvent(context.Background(), internalKafka.EventDriverAvailabilityChanged, internalKafka.DriverAvailabilityChangedEvent{
			DriverID:   driverID,
			SupplierID: supplierID,
			Available:  available,
			Reason:     reason,
			Note:       note,
			TruckID:    truckID,
			Timestamp:  time.Now().UTC(),
		})
		// 2. WebSocket push to admin portal
		telemetry.FleetHub.BroadcastDriverAvailability(supplierID, driverID, available, reason)
	}

	// /v1/admin/retailer/{pending,approve,reject} moved to adminroutes.

	// POST /v1/order/amend — Driver partial-quantity reconciliation at delivery.
	// Recalculates order total, inserts SupplierReturns, emits ORDER_MODIFIED to Kafka.
	// Wraps HandleAmendOrder with WS push for ORDER_AMENDED event.
	http.HandleFunc("/v1/order/amend",
		auth.RequireRole([]string{"DRIVER", "ADMIN"},
			loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				var req order.AmendOrderRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if req.OrderID == "" || len(req.Items) == 0 {
					http.Error(w, "order_id and items are required", http.StatusBadRequest)
					return
				}

				resp, err := svc.AmendOrder(r.Context(), req)
				if err != nil {
					var versionConflict *order.ErrVersionConflict
					if errors.As(err, &versionConflict) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusConflict)
						json.NewEncoder(w).Encode(map[string]string{"error": versionConflict.Error()})
						return
					}
					var freezeLock *order.ErrFreezeLock
					if errors.As(err, &freezeLock) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(423)
						json.NewEncoder(w).Encode(map[string]string{"error": freezeLock.Error()})
						return
					}
					if strings.Contains(err.Error(), "cannot be amended") {
						http.Error(w, err.Error(), http.StatusConflict)
					} else if strings.Contains(err.Error(), "not found") {
						http.Error(w, err.Error(), http.StatusNotFound)
					} else {
						http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
					}
					return
				}

				// Push ORDER_AMENDED to retailer + driver via WebSocket
				if resp.RetailerID != "" {
					go retailerHub.PushToRetailer(resp.RetailerID, map[string]interface{}{
						"type":         ws.EventOrderAmended,
						"order_id":     req.OrderID,
						"amendment_id": resp.AmendmentID,
						"new_total":    resp.AdjustedTotal,
						"message":      resp.Message,
					})
				}
				if resp.DriverID != "" {
					go driverHub.PushToDriver(resp.DriverID, map[string]interface{}{
						"type":         ws.EventOrderAmended,
						"order_id":     req.OrderID,
						"amendment_id": resp.AmendmentID,
						"new_total":    resp.AdjustedTotal,
						"message":      resp.Message,
					})
				}
				if resp.SupplierID != "" {
					go telemetry.FleetHub.BroadcastOrderStateChange(resp.SupplierID, req.OrderID, "AMENDED", "")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			}),
		),
	)

	// POST /v1/vehicle/{vehicleId}/clear-returns — Supplier confirms return receipt at depot.
	// Clears ReturnClearedAt on rejected OrderLineItems, releasing locked VU from capacity.
	http.HandleFunc("/v1/vehicle/",
		auth.RequireRole([]string{"ADMIN"},
			loggingMiddleware(svc.HandleClearReturns),
		),
	)

	// ── Empathy Engine: Hierarchical Auto-Order Settings (PATCH) ──
	empathySvc := &settings.EmpathyService{Client: spannerClient, Cache: app.Cache}
	http.HandleFunc("/v1/retailer/settings/auto-order/global",
		auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(empathySvc.HandlePatchGlobal)))
	http.HandleFunc("/v1/retailer/settings/auto-order/supplier/",
		auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(empathySvc.HandlePatchSupplier)))
	http.HandleFunc("/v1/retailer/settings/auto-order/product/",
		auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(empathySvc.HandlePatchProduct)))
	http.HandleFunc("/v1/retailer/settings/auto-order/variant/",
		auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(empathySvc.HandlePatchVariant)))
	http.HandleFunc("/v1/retailer/settings/auto-order/category/",
		auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(empathySvc.HandlePatchCategory)))

	// GET /v1/retailer/settings/auto-order — Full hierarchy settings for retailer
	http.HandleFunc("/v1/retailer/settings/auto-order",
		auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(empathySvc.HandleGetAutoOrderSettings)))

	// GET /v1/orders/line-items/history — SKU purchase history for AI Worker
	http.HandleFunc("/v1/orders/line-items/history",
		auth.RequireRole([]string{"RETAILER", "ADMIN"}, loggingMiddleware(order.HandleLineItemHistory(spannerClient, app.SpannerRouter))))

	// Initialize the Communication Spine: FCM (primary) + Telegram (fallback)
	// FCM boots as graceful no-op if firebase credentials are absent (local dev).
	fcmCredPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	var fcmClient *notifications.FCMClient
	if fcmCredPath != "" {
		fcmClient, err = notifications.InitFCM(fcmCredPath)
		if err != nil {
			log.Printf("[COMMUNICATION SPINE] FCM init failed (%v) — falling back to no-op mode", err)
			fcmClient = notifications.NewNoOpFCMClient()
		}
	} else {
		fcmClient = notifications.NewNoOpFCMClient()
	}
	fcmClient.SpannerClient = spannerClient // Enable stale token auto-purge
	tgClient := notifications.NewTelegramClient()

	// Initialize Shop-Closed Protocol dependencies now that FCM is available
	shopClosedDeps = order.ShopClosedDeps{
		RetailerPush: retailerHub.PushToRetailer,
		DriverPush:   driverHub.PushToDriver,
		AdminBroadcast: func(payload interface{}) {
			data, _ := json.Marshal(payload)
			telemetry.FleetHub.BroadcastToAdmins(data)
		},
		NotifyUser: func(ctx context.Context, userID, role, title, body string, data map[string]string) {
			fcmClient.SendDataMessage(userID, data)
		},
	}

	// v3.1 Edge 27: Early Route Complete deps
	earlyCompleteDeps = order.EarlyCompleteDeps{
		SupplierPush: func(supplierID string, payload interface{}) bool {
			data, _ := json.Marshal(payload)
			telemetry.FleetHub.BroadcastToAdmins(data)
			return true
		},
		DriverPush: driverHub.PushToDriver,
		NotifyUser: func(ctx context.Context, userID, role, title, body string, data map[string]string) {
			fcmClient.SendDataMessage(userID, data)
		},
	}

	// v3.1 Edge 28: Negotiation deps
	negotiationDeps = order.NegotiationDeps{
		SupplierPush: func(supplierID string, payload interface{}) bool {
			data, _ := json.Marshal(payload)
			telemetry.FleetHub.BroadcastToAdmins(data)
			return true
		},
		DriverPush: driverHub.PushToDriver,
		NotifyUser: func(ctx context.Context, userID, role, title, body string, data map[string]string) {
			fcmClient.SendDataMessage(userID, data)
		},
	}

	// Initialize Firebase Auth (dual-mode: emulator for local dev, credentials for prod).
	// When FIREBASE_AUTH_EMULATOR_HOST is set, connects to emulator without credentials.
	// When nil, the system falls back to legacy-only HS256 JWT mode.
	if _, fbAuthErr := auth.InitFirebaseAuth(ctx); fbAuthErr != nil {
		log.Printf("[FIREBASE AUTH] Init skipped: %v — legacy JWT mode active", fbAuthErr)
	}

	// Start the temporal heartbeat Awakener
	StartAwakener(svc, fcmClient, tgClient, app.Cache)

	// Broadcast service instance — route mounted by adminroutes below.
	broadcastSvc := &notifications.BroadcastService{Spanner: spannerClient, FCM: fcmClient}

	// Start the Scheduled Order Promoter (SCHEDULED → PENDING within 24h of delivery)
	StartScheduledOrderPromoter(spannerClient)

	// Phase 3: Boot the Retailer WebSocket Hub + DRIVER_APPROACHING Kafka consumer
	http.HandleFunc("/v1/ws/retailer",
		auth.RequireRole([]string{"RETAILER"}, retailerHub.HandleConnection))
	internalKafka.StartApproachConsumer(ctx, retailerHub, fcmClient, spannerClient, cfg.KafkaBrokerAddress)
	fmt.Println("[BOOT] Retailer WebSocket Hub + Approach Consumer: ONLINE")

	// Phase 3b: Boot the Driver WebSocket Hub (receives PAYMENT_SETTLED pushes)
	http.HandleFunc("/v1/ws/driver",
		auth.RequireRole([]string{"DRIVER"}, driverHub.HandleConnection))
	fmt.Println("[BOOT] Driver WebSocket Hub: ONLINE")

	// Phase 3c: Boot the Payloader WebSocket Hub (receives PAYLOAD_READY_TO_SEAL pushes)
	http.HandleFunc("/v1/ws/payloader",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, payloaderHub.HandleConnection))
	fmt.Println("[BOOT] Payloader WebSocket Hub: ONLINE")

	// Boot the Notification Dispatcher Consumer (inbox + WS + Telegram for all event types)
	internalKafka.StartNotificationDispatcher(ctx, internalKafka.NotificationDeps{
		RetailerHub:   retailerHub,
		DriverHub:     driverHub,
		PayloaderHub:  payloaderHub,
		FCM:           fcmClient,
		Telegram:      tgClient,
		SpannerClient: spannerClient,
	}, cfg.KafkaBrokerAddress)
	fmt.Println("[BOOT] Notification Dispatcher Consumer: ONLINE")

	// Boot the Financial Worker
	internalKafka.StartTreasurer(ctx, spannerClient, cfg.KafkaBrokerAddress, platformCfg)
	internalKafka.StartGatewayWorker(ctx, internalKafka.GatewayWorkerDeps{
		Spanner:       spannerClient,
		BrokerAddress: cfg.KafkaBrokerAddress,
		Vault:         &vault.PaymentVaultAdapter{Svc: vaultSvc},
		GPDirect:      directClient,
	})
	reconcilerSvc := payment.NewReconcilerService(cfg, spannerClient)
	go reconcilerSvc.Start(ctx)
	log.Println("[BOOT] Financial Reconciler Service: ONLINE")

	// Boot the Transactional Outbox relay (V.O.I.D. Phase VII).
	app.Outbox.SetOnFailure(func(eventID, aggregateID, topic string, err error) {
		app.WarehouseHub.BroadcastOutboxFailure(eventID, aggregateID, topic, err.Error())
	})
	app.Outbox.Start(ctx)
	log.Println("[BOOT] Transactional Outbox Relay: ONLINE")

	// Boot the Global Pay Session Sweeper (expired + stale session recovery)
	gpReconciler := &payment.GlobalPayReconciler{
		Spanner:       spannerClient,
		SessionSvc:    sessionSvc,
		VaultResolver: &vault.PaymentVaultAdapter{Svc: vaultSvc},
		Producer:      svc.Producer,
		DriverHub:     driverHub,
		RetailerHub:   retailerHub,
	}
	StartGlobalPaySweeper(gpReconciler)
	log.Println("[BOOT] Global Pay Session Sweeper: ONLINE")

	StartPaymentSessionExpirer(sessionSvc, retailerHub)
	log.Println("[BOOT] Payment Session Expiry Cron: ONLINE")

	// Boot the Failsafe Transmitter
	internalKafka.InitDLQ(cfg.KafkaBrokerAddress)

	// 4. Initialize the Field General AI Routing Cron
	fmt.Println("[BOOT] Arming Field General Route Optimizer (04:00 AM UTC+5)...")
	// Since we are matching the instruction closely:
	go routing.StartCron(ctx, spannerClient, cfg.GoogleMapsAPIKey, cfg.DepotLocation)

	// Boot the Replenishment Engine (4h stock deficit analysis)
	replenishEngine := &replenishment.ReplenishmentEngine{Spanner: spannerClient, Producer: svc.Producer}
	replenishEngine.StartReplenishmentCron()
	log.Println("[BOOT] Replenishment Engine Cron: ONLINE (4h interval)")

	// /v1/admin/* — 22 endpoints. Ownership lives in backend-go/adminroutes.
	adminroutes.RegisterRoutes(r, adminroutes.Deps{
		Spanner:            spannerClient,
		ReadRouter:         app.SpannerRouter,
		Order:              svc,
		CountryConfig:      countryCfgSvc,
		PlatformCfg:        platformCfg,
		SessionSvc:         sessionSvc,
		GPReconciler:       gpReconciler,
		ReplenishEngine:    replenishEngine,
		BroadcastSvc:       broadcastSvc,
		KafkaBrokerAddress: cfg.KafkaBrokerAddress,
		ShopClosedDeps:     &shopClosedDeps,
		EarlyCompleteDeps:  &earlyCompleteDeps,
		NegotiationDeps:    &negotiationDeps,
		Log:                loggingMiddleware,
	})

	// 6. Quarantine Protocol — Stale Order Auditor (15min sweep)
	StartStaleOrderAuditor(spannerClient)
	log.Println("[BOOT] Stale Order Auditor (Quarantine Protocol): ONLINE (15min interval)")

	// 7. Edge 4: Orphaned AIPredictionItems Cleanup (daily)
	StartOrphanedPredictionCleaner(spannerClient)
	log.Println("[BOOT] Orphaned Prediction Cleaner: ONLINE (daily interval)")

	// 5. Boot Up the Server
	srv := &http.Server{
		Addr:              ":" + cfg.BackendPort,
		Handler:           cache.LimitBodyMiddleware(cache.MaxBodySize)(enableCORS(r)),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// /v1/driver/manifest moved to driverroutes.

	// /v1/admin/dlq{,/replay} moved to adminroutes.

	// ── Supplier Operational Tools (Phase 8) ─────────────────────────────

	// Inventory Management — GET (list) / PATCH (adjust stock)
	http.HandleFunc("/v1/supplier/inventory",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleInventory(spannerClient))))

	// Inventory Audit Log — GET (last 100 adjustments)
	http.HandleFunc("/v1/supplier/inventory/audit",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleInventoryAuditLog(spannerClient))))

	// Order Vetting — Supplier approval queue
	vettingSvc := supplier.NewOrderVettingService(spannerClient, svc.Producer, retailerHub)
	http.HandleFunc("/v1/supplier/orders",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(vettingSvc.HandleSupplierOrders)))
	http.HandleFunc("/v1/supplier/orders/vet",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(vettingSvc.HandleVetOrder)))

	// WAREHOUSE PICKING MANIFESTS: changed endpoint to /v1/supplier/picking-manifests
	// to avoid duplicate registration conflict with SupplierTruckManifests
	http.HandleFunc("/v1/supplier/picking-manifests",
		auth.RequireRole([]string{"SUPPLIER", "PAYLOADER", "ADMIN"}, loggingMiddleware(supplier.HandleManifests(spannerClient))))
	http.HandleFunc("/v1/supplier/picking-manifests/orders",
		auth.RequireRole([]string{"SUPPLIER", "PAYLOADER", "ADMIN"}, loggingMiddleware(supplier.HandleManifestOrders(spannerClient))))

	// POST /v1/supplier/manifests/auto-dispatch — Auto-Dispatch Engine: geo-cluster + bin-pack
	http.HandleFunc("/v1/supplier/manifests/auto-dispatch",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleAutoDispatch(spannerClient, app.SpannerRouter, manifestSvc, app.OptimizerClient, app.DispatchOptimizer))))

	// POST /v1/supplier/manifests/dispatch-recommend — Dry-run dispatch: returns recommended truck groupings without mutations
	http.HandleFunc("/v1/supplier/manifests/dispatch-recommend",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleDispatchRecommend(spannerClient, app.SpannerRouter))))

	// POST /v1/supplier/manifests/manual-dispatch — Create a single manifest for admin-chosen driver+orders
	http.HandleFunc("/v1/supplier/manifests/manual-dispatch",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleManualDispatch(spannerClient, app.SpannerRouter, manifestSvc))))

	// ── Fleet Bridge: Volumetrics + Dispatch Queue + H3 Route Preview ───────
	// GET /v1/supplier/fleet-volumetrics — Fleet capacity vs. warehouse backlog
	http.HandleFunc("/v1/supplier/fleet-volumetrics",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleFleetVolumetrics(spannerClient)))))
	// POST /v1/supplier/dispatch-queue — Move READY_FOR_DISPATCH orders into manifests
	http.HandleFunc("/v1/supplier/dispatch-queue",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleDispatchQueue(spannerClient, app.SpannerRouter, manifestSvc, app.OptimizerClient, app.DispatchOptimizer)))))

	// ── Stealth Simulation Harness (/v1/internal/sim/) ────────────────────
	// Gated by SIMULATION_ENABLED=true at boot. ADMIN-only.
	if app.Simulation != nil {
		http.HandleFunc("/v1/internal/sim/start",
			auth.RequireRole([]string{"ADMIN"}, loggingMiddleware(simulation.HandleStart(app.Simulation))))
		http.HandleFunc("/v1/internal/sim/stop",
			auth.RequireRole([]string{"ADMIN"}, loggingMiddleware(simulation.HandleStop(app.Simulation))))
		http.HandleFunc("/v1/internal/sim/status",
			auth.RequireRole([]string{"ADMIN"}, loggingMiddleware(simulation.HandleStatus(app.Simulation))))
	}

	// GET /v1/supplier/dispatch-preview — H3-clustered order groups for planning
	http.HandleFunc("/v1/supplier/dispatch-preview",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(supplier.HandleH3RoutePreview(spannerClient, app.SpannerRouter)))))

	// GET /v1/supplier/manifests/waiting-room — Orders created after dispatch snapshot
	http.HandleFunc("/v1/supplier/manifests/waiting-room",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleWaitingRoom(spannerClient))))

	// Dispute & Returns Queue — REJECTED_DAMAGED line items
	returnsSvc := supplier.NewReturnsService(spannerClient, svc.Producer)
	http.HandleFunc("/v1/supplier/returns",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(returnsSvc.HandleReturns)))
	http.HandleFunc("/v1/supplier/returns/resolve",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(returnsSvc.HandleResolveReturn)))

	// Depot Reconciliation — QUARANTINE reverse-logistics (vehicle-scoped bulk resolution)
	reconcileSvc := supplier.NewReconcileService(spannerClient, svc.Producer)
	http.HandleFunc("/v1/supplier/quarantine-stock",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(reconcileSvc.HandleQuarantineStock)))
	http.HandleFunc("/v1/inventory/reconcile-returns",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(reconcileSvc.HandleReconcile)))

	// /v1/fleet/route/{id}/complete moved to fleetroutes.

	// /v1/catalog/* — 5 routes (including /v1/catalog/platform-categories and
	// /v1/catalog/suppliers/search registered below this line in the original
	// file). Ownership lives in backend-go/catalogroutes/routes.go.
	catalogroutes.RegisterRoutes(r, catalogroutes.Deps{Spanner: spannerClient, Log: loggingMiddleware})
	http.HandleFunc("/v1/retailer/suppliers", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(supplier.HandleRetailerSuppliers(spannerClient))))
	http.HandleFunc("/v1/retailer/suppliers/", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(supplier.HandleRetailerSuppliers(spannerClient))))

	// GET/PUT /v1/retailer/profile — Retailer profile management
	http.HandleFunc("/v1/retailer/profile", auth.RequireRole([]string{"RETAILER"}, loggingMiddleware(supplier.HandleRetailerProfile(spannerClient, app.Cache, app.CacheFlight))))

	// /v1/catalog/suppliers/search + /v1/ai/predictions{,/correct} were moved to
	// backend-go/catalogroutes and backend-go/airoutes respectively.

	// ── Payment Webhooks (NO JWT — authenticated via gateway signature/Basic Auth) ────
	webhookSvc := &payment.WebhookService{
		Spanner:       spannerClient,
		Producer:      svc.Producer,
		DriverHub:     driverHub,
		RetailerHub:   retailerHub,
		VaultResolver: &vault.PaymentVaultAdapter{Svc: vaultSvc},
		SessionSvc:    sessionSvc,
	}
	// /v1/webhooks/* — 3 gateway callbacks. Ownership lives in backend-go/webhookroutes.
	webhookroutes.RegisterRoutes(r, webhookroutes.Deps{
		WebhookSvc:    webhookSvc,
		Log:           loggingMiddleware,
		PriorityGuard: priorityGuard,
	})

	// /v1/admin/payment/reconcile moved to adminroutes.

	// /v1/payment/global_pay/initiate (DEPRECATED) moved to paymentroutes.

	// ══════════════════════════════════════════════════════════════════════════════
	// FACTORY-TO-WAREHOUSE REPLENISHMENT ROUTES
	// ══════════════════════════════════════════════════════════════════════════════

	// Factory Transfer Services (need Kafka producer)
	transferSvc := &factory.TransferService{Spanner: spannerClient, Cache: app.Cache, Producer: svc.Producer}
	emergencySvc := &factory.EmergencyTransferService{Spanner: spannerClient, Producer: svc.Producer}

	// /v1/auth/factory/{login,register} → authroutes package.

	// ── Supplier → Factory CRUD ───────────────────────────────────────────────
	http.HandleFunc("/v1/supplier/factories",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(factory.HandleSupplierFactories(spannerClient, app.Cache)))))
	http.HandleFunc("/v1/supplier/factories/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(auth.RequireWarehouseScope(factory.HandleSupplierFactoryDetail(spannerClient, app.Cache)))))

	// ── Factory-Warehouse Recommendations & Assignments ───────────────────────
	http.HandleFunc("/v1/supplier/factories/recommend-warehouses",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(factory.HandleRecommendWarehouses(spannerClient))))
	http.HandleFunc("/v1/supplier/factories/optimal-assignments",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(factory.HandleOptimalAssignments(spannerClient))))

	// ── Reverse Geocoding ─────────────────────────────────────────────────────
	http.HandleFunc("/v1/supplier/geocode/reverse",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleReverseGeocode())))

	// ── Retailer Locations (Map Surface) ──────────────────────────────────────
	http.HandleFunc("/v1/supplier/retailers/locations",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplier.HandleRetailerLocations(spannerClient))))

	// /v1/factory/{analytics/overview,profile,transfers,transfers/{id},transfers/create,
	// manifests,manifests/{id},fleet/drivers,fleet/vehicles,staff,staff/{id}} moved to factoryroutes.

	// Warehouse-side transfer + force-receive services (moved to warehouseroutes below).
	forceReceiveSvc := &factory.ForceReceiveService{Spanner: spannerClient, Producer: svc.Producer}

	// /v1/warehouse/{transfers/emergency,transfers/,transfers/force-receive,
	// replenishment/insights,replenishment/insights/} moved to warehouseroutes.

	// /v1/admin/replenishment/trigger moved to adminroutes.

	// ══════════════════════════════════════════════════════════════════════════════
	// REPLENISHMENT GRAPH HARDENING — Supply Lanes, Network Optimizer, Kill Switch
	// ══════════════════════════════════════════════════════════════════════════════

	// Service Initialization
	networkOptSvc := &factory.NetworkOptimizerService{Spanner: spannerClient, Producer: svc.Producer}
	replenLockSvc := &factory.ReplenishmentLockService{Spanner: spannerClient, Producer: svc.Producer}
	supplyLaneSvc := &factory.SupplyLanesService{Spanner: spannerClient, Producer: svc.Producer}
	killSwitchSvc := &factory.KillSwitchService{Spanner: spannerClient, Producer: svc.Producer}
	pullMatrixSvc := &factory.PullMatrixService{Spanner: spannerClient, Producer: svc.Producer, LockSvc: replenLockSvc, Optimizer: networkOptSvc}
	predictivePushSvc := &factory.PredictivePushService{Spanner: spannerClient, Producer: svc.Producer, Optimizer: networkOptSvc}
	slaMonitorSvc := &factory.SLAMonitorService{Spanner: spannerClient, Producer: svc.Producer, Optimizer: networkOptSvc}

	// ── Supply Lanes CRUD ─────────────────────────────────────────────────────
	http.HandleFunc("/v1/supplier/supply-lanes",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplyLaneSvc.HandleSupplyLanes)))
	http.HandleFunc("/v1/supplier/supply-lanes/",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(supplyLaneSvc.HandleSupplyLaneAction)))

	// ── Network Optimization Mode ────────────────────────────────────────────
	http.HandleFunc("/v1/supplier/network-mode",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				networkOptSvc.HandleGetNetworkMode(w, r)
			case http.MethodPut:
				networkOptSvc.HandleSetNetworkMode(w, r)
			default:
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		})))
	http.HandleFunc("/v1/supplier/network-analytics",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(networkOptSvc.HandleNetworkAnalytics)))

	// ── Kill Switch (halt all automated replenishment) ───────────────────────
	http.HandleFunc("/v1/supplier/replenishment/kill-switch",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(killSwitchSvc.HandleKillSwitch)))
	http.HandleFunc("/v1/supplier/replenishment/audit",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(killSwitchSvc.HandleListKillSwitchAudit)))

	// ── Pull Matrix (manual trigger) ─────────────────────────────────────────
	http.HandleFunc("/v1/supplier/replenishment/pull-matrix",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(pullMatrixSvc.HandleManualPullMatrix)))

	// ── Predictive Push (manual trigger) ─────────────────────────────────────
	http.HandleFunc("/v1/supplier/replenishment/predictive-push",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(predictivePushSvc.HandleManualPredictivePush)))

	// ── Crons: Pull Matrix Aggregator (4h) + SLA Monitor (30min) + CurrentLoad Reset (24h) ──
	StartPullMatrixAggregator(pullMatrixSvc, predictivePushSvc)
	StartFactorySLAMonitor(slaMonitorSvc)
	StartCurrentLoadReset(spannerClient)
	StartCoverageAuditor(spannerClient)
	log.Println("[BOOT] Pull Matrix Aggregator Cron: ONLINE (4h interval)")
	log.Println("[BOOT] Factory SLA Monitor Cron: ONLINE (30min interval)")
	log.Println("[BOOT] Factory CurrentLoad Reset Cron: ONLINE (24h interval)")
	log.Println("[BOOT] Coverage Auditor Cron: ONLINE (6h interval)")

	// /v1/factory/dispatch moved to factoryroutes.

	// ══════════════════════════════════════════════════════════════════════════════
	// PHASE IV: WAREHOUSE SUPPLY CHAIN & PRE-ORDER POLICY ROUTES
	// ══════════════════════════════════════════════════════════════════════════════

	// /v1/auth/warehouse/{login,register} → authroutes package.

	// /v1/warehouse/{demand/forecast,supply-requests,supply-requests/} moved to
	// warehouseroutes. supplyReqSvc is kept here because factoryroutes below
	// also consumes it (shared supply-request surface).
	supplyReqSvc := &warehouse.SupplyRequestService{Spanner: spannerClient, Producer: svc.Producer, WarehouseHub: app.WarehouseHub}

	// /v1/factory/* — 17 routes (analytics, profile, transfers, manifests,
	// fleet/drivers, fleet/vehicles, staff, dispatch, supply-requests,
	// manifests/{rebalance,cancel-transfer,cancel}). Ownership lives in
	// backend-go/factoryroutes. Mounted here because it needs supplyReqSvc
	// (shared with the warehouse/supply-requests surface above) and
	// transferSvc (shared with /v1/warehouse/transfers/ below).
	factoryroutes.RegisterRoutes(r, factoryroutes.Deps{
		Spanner:          spannerClient,
		ReadRouter:       app.SpannerRouter,
		Producer:         svc.Producer,
		TransferSvc:      transferSvc,
		SupplyRequestSvc: supplyReqSvc,
		Cache:            app.Cache,
		CacheFlight:      app.CacheFlight,
		Log:              loggingMiddleware,
	})

	// dispatchLockSvc kept in main.go — proximity.HandleApplyTerritory below
	// also consumes IsDispatchLocked for territory re-assignment safety.
	dispatchLockSvc := &warehouse.DispatchLockService{Spanner: spannerClient, WarehouseHub: app.WarehouseHub}

	// /v1/warehouse/* — 28 endpoints spanning transfer acceptance, replenishment
	// insights, demand forecast, supply-request CRUD + state machine, the
	// WAREHOUSE_ADMIN Ops Portal, and the dispatch-lock system. Ownership
	// lives in backend-go/warehouseroutes.
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{
		Spanner:         spannerClient,
		Producer:        svc.Producer,
		TransferSvc:     transferSvc,
		EmergencySvc:    emergencySvc,
		ForceReceiveSvc: forceReceiveSvc,
		SupplyReqSvc:    supplyReqSvc,
		DispatchLockSvc: dispatchLockSvc,
		OrderSvc:        svc,
		ReadRouter:      app.SpannerRouter,
		Optimizer:       app.OptimizerClient,
		DispatchCounts:  app.DispatchOptimizer,
		Log:             loggingMiddleware,
		Cache:           app.Cache,
	})

	// ── Spatial Recommendation (Territory Voronoi + Apply) ────────────────────
	http.HandleFunc("/v1/supplier/warehouses/territory-preview",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandlePreviewTerritories(spannerClient))))
	http.HandleFunc("/v1/supplier/warehouses/apply-territory",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, loggingMiddleware(proximity.HandleApplyTerritory(spannerClient, dispatchLockSvc.IsDispatchLocked))))

	// ── Warehouse WebSocket Hub ──────────────────────────────────────────────
	http.HandleFunc("/ws/warehouse",
		auth.RequireRoleWithGrace([]string{"WAREHOUSE", "SUPPLIER", "ADMIN"}, 2*time.Hour, warehouseHub.HandleConnection))

	// ── Pre-Order Confirmation Policy Cron ────────────────────────────────────
	StartPreOrderConfirmationSweeper(spannerClient, fcmClient, tgClient, func(ctx context.Context, eventType string, payload interface{}) {
		svc.PublishEvent(ctx, eventType, payload)
	})

	// ── Auto-Confirm Sweeper (consumes AutoConfirmAt from Awakener) ───────────
	StartAutoConfirmSweeper(spannerClient, func(ctx context.Context, eventType string, payload interface{}) {
		svc.PublishEvent(ctx, eventType, payload)
	})

	// ── Notification Expirer (soft-deletes stale notifications) ───────────────
	StartNotificationExpirer(spannerClient)

	// DEBUG ROUTE: strictly gated to ENVIRONMENT=development. Any other
	// value (including empty, which is now rejected at config load) keeps
	// /debug/mint-token unmounted.
	if cfg.IsDevelopment() {
		log.Println("[SECURITY] /debug/mint-token is MOUNTED (ENVIRONMENT=development)")
		http.HandleFunc("/debug/mint-token", func(w http.ResponseWriter, r *http.Request) {
			role := r.URL.Query().Get("role")
			if role == "" {
				role = "RETAILER" // Default to lowest clearance
			}

			userId := r.URL.Query().Get("user_id")
			if userId == "" {
				userId = "TEST-USER-99"
			}

			tokenString, err := auth.GenerateTestToken(userId, role)
			if err != nil {
				http.Error(w, "Failed to forge token", http.StatusInternalServerError)
				return
			}

			w.Write([]byte(tokenString))
		})
	}

	go func() {
		log.Printf("Server actively listening on localhost:%s\n", cfg.BackendPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	// 6. Graceful Shutdown orchestration
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("\nSIGTERM received, executing graceful shutdown sequence...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server shutdown forced: %v", err)
	}

	// Close all WebSocket hubs (send CloseGoingAway to connected clients)
	driverHub.Close()
	retailerHub.Close()
	payloaderHub.Close()
	warehouseHub.Close()
	telemetry.FleetHub.Close()

	spannerClient.Close()
	// Kafka Writer close (if directly attached to struct)
	if svc.Producer != nil {
		svc.Producer.Close()
	}
	// Close singleton Kafka writers (sync, correction, DLQ)
	internalKafka.CloseWriters()
	// Close Redis connection pool
	cache.Close()

	// ── Flush OTel spans before exit ─────────────────────────────────────
	// This is the "silent P0" guard: if the pod dies before flushing the
	// OTel buffer, we lose the logs of the crash — which are the most
	// important logs we have. The shutdown context gives 5 s for flush.
	if app.TracerShutdown != nil {
		otelCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := app.TracerShutdown(otelCtx); err != nil {
			log.Printf("OTel tracer shutdown error: %v", err)
		}
		otelCancel()
	}

	log.Println("Backend teardown complete.")
}
