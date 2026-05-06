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
	"backend-go/migrations"
	"backend-go/cache"
	"backend-go/catalogroutes"
	"backend-go/deliveryroutes"
	"backend-go/driverroutes"
	"backend-go/factory"
	"backend-go/factoryroutes"
	"backend-go/fleet"
	"backend-go/fleetroutes"
	"backend-go/idempotency"
	"backend-go/notifications"
	"backend-go/order"
	"backend-go/orderroutes"
	"backend-go/payloaderroutes"
	"backend-go/payment"
	"backend-go/paymentroutes"
	"backend-go/proximityroutes"
	"backend-go/replenishment"
	"backend-go/retailerroutes"
	"backend-go/routing"
	"backend-go/settings"
	"backend-go/simroutes"
	"backend-go/supplier"
	"backend-go/suppliercatalogroutes"
	"backend-go/suppliercoreroutes"
	"backend-go/supplierinsightsroutes"
	"backend-go/supplierlogisticsroutes"
	"backend-go/supplieroperationsroutes"
	"backend-go/supplierplanningroutes"
	"backend-go/supplierroutes"
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
	supplierHub := app.SupplierHub
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

	// ── Vector G: B2B Dynamic Pricing Engine ──────────────────────────────────
	// (supplierPricingSvc is constructed in bootstrap and aliased above.)
	retailerPricingSvc := supplier.NewRetailerPricingService(spannerClient, app.SpannerRouter, svc.Producer, app.Cache)
	suppliercatalogroutes.RegisterRoutes(r, suppliercatalogroutes.Deps{
		Spanner:         spannerClient,
		Pricing:         supplierPricingSvc,
		RetailerPricing: retailerPricingSvc,
		Log:             loggingMiddleware,
		Idempotency:     idempotency.Guard,
	})

	// ── LEO: LOGISTICS EXECUTION ORCHESTRATOR — Manifest Loading Gate ─────

	manifestSvc := &supplier.ManifestService{
		Spanner:       spannerClient,
		Cache:         app.Cache,
		MapsAPIKey:    cfg.GoogleMapsAPIKey,
		DepotLocation: cfg.DepotLocation,
	}

	// /v1/driver/* + /v1/fleet/manifest + /v1/ws/driver — driver role-row routes.
	// Ownership lives in backend-go/driverroutes.
	driverroutes.RegisterRoutes(r, driverroutes.Deps{
		Spanner:     spannerClient,
		Order:       svc,
		ManifestSvc: manifestSvc,
		DriverHub:   driverHub,
		Cache:       app.Cache,
		CacheFlight: app.CacheFlight,
		Log:         loggingMiddleware,
	})

	supplierlogisticsroutes.RegisterRoutes(r, supplierlogisticsroutes.Deps{
		Spanner:     spannerClient,
		ReadRouter:  app.SpannerRouter,
		ManifestSvc: manifestSvc,
		Optimizer:   app.OptimizerClient,
		Counters:    app.DispatchOptimizer,
		Log:         loggingMiddleware,
		Idempotency: idempotency.Guard,
	})
	vettingSvc := supplier.NewOrderVettingService(spannerClient, svc.Producer, retailerHub)
	suppliercoreroutes.RegisterRoutes(r, suppliercoreroutes.Deps{
		Spanner:     spannerClient,
		ReadRouter:  app.SpannerRouter,
		Order:       svc,
		Vetting:     vettingSvc,
		Log:         loggingMiddleware,
		Idempotency: idempotency.Guard,
	})

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
		Idempotency:    idempotency.Guard,
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
	treasury.RegisterRoutes(r, treasury.Deps{
		Spanner:     spannerClient,
		Log:         loggingMiddleware,
		Idempotency: idempotency.Guard,
	})

	// /v1/admin/{reconciliation,audit-log,country-configs,country-configs/} moved to adminroutes.

	supplierinsightsroutes.RegisterRoutes(r, supplierinsightsroutes.Deps{
		Spanner:     spannerClient,
		ReadRouter:  app.SpannerRouter,
		CountryCfg:  countryCfgSvc,
		Log:         loggingMiddleware,
		Idempotency: idempotency.Guard,
	})
	retailerroutes.RegisterRoutes(r, retailerroutes.Deps{
		Spanner:        spannerClient,
		ReadRouter:     app.SpannerRouter,
		Cache:          app.Cache,
		CacheFlight:    app.CacheFlight,
		Order:          svc,
		SessionSvc:     sessionSvc,
		CardTokenSvc:   cardTokenSvc,
		CardsClient:    cardsClient,
		Empathy:        app.Empathy,
		RetailerHub:    retailerHub,
		DriverHub:      driverHub,
		ShopClosedDeps: &shopClosedDeps,
		Log:            loggingMiddleware,
	})

	supplieroperationsroutes.RegisterRoutes(r, supplieroperationsroutes.Deps{
		Spanner:     spannerClient,
		Order:       svc,
		Producer:    svc.Producer,
		Log:         loggingMiddleware,
		Idempotency: idempotency.Guard,
	})

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

	// /v1/orders/request-cancel moved to retailerroutes.

	// /v1/admin/orders/approve-cancel moved to adminroutes.

	// /v1/delivery/sms-complete moved to deliveryroutes.

	// /v1/delivery/shop-closed moved to deliveryroutes.

	// /v1/retailer/shop-closed-response moved to retailerroutes.

	// /v1/admin/shop-closed/resolve moved to adminroutes.

	// /v1/delivery/bypass-offload moved to deliveryroutes.

	// Edge 24: Device fingerprinting (wired into login — see auth/device.go)

	// ── v3.1 Human-Centric Edge Routes ──────────────────────────────────────

	// /v1/fleet/route/request-early-complete moved to fleetroutes.

	// /v1/admin/route/approve-early-complete moved to adminroutes.

	// /v1/delivery/negotiate moved to deliveryroutes.

	// /v1/admin/negotiate/resolve moved to adminroutes.

	// /v1/retailer/family-members* moved to retailerroutes.

	// /v1/delivery/credit-delivery moved to deliveryroutes.

	// /v1/admin/orders/resolve-credit moved to adminroutes.

	// /v1/delivery/missing-items moved to deliveryroutes.

	// Retailer AI-review and preorder lifecycle actions moved to retailerroutes.

	// /v1/delivery/split-payment moved to deliveryroutes.

	// /v1/retailer/cart/sync moved to retailerroutes.

	// /v1/user/notifications{,/read} moved to userroutes.

	// /v1/supplier/settlement-report → treasury package (registered above).

	// /v1/auth/refresh, /v1/auth/{driver,admin,supplier,retailer,payloader,factory,warehouse}/...
	// were moved to backend-go/authroutes/routes.go (registered above near the /v1/auth/login block).

	// /v1/driver/profile moved to driverroutes.

	// /v1/supplier/{configure,billing/setup,profile,shift,payment-config,
	// gateway-onboarding,payment/recipient/register} moved to supplierroutes.
	supplierroutes.RegisterRoutes(r, supplierroutes.Deps{
		Spanner:      spannerClient,
		Cache:        app.Cache,
		CacheFlight:  app.CacheFlight,
		DirectClient: directClient,
		Producer:     svc.Producer,
		SupplierHub:  supplierHub,
		Log:          loggingMiddleware,
		Idempotency:  idempotency.Guard,
	})

	// /v1/catalog/platform-categories moved to catalogroutes.

	// /v1/auth/retailer/{login,register} → authroutes package.

	// /v1/admin/{nuke,config,empathy/adoption} moved to adminroutes.

	// /v1/supplier/{org/members,staff/payloader,warehouse-staff,warehouses,
	// warehouse-inflight-vu} moved to supplierroutes.

	// /v1/supplier/{serving-warehouse,geo-report,zone-preview,
	// warehouses/validate-coverage,warehouse-loads} moved to proximityroutes.
	proximityroutes.RegisterRoutes(r, proximityroutes.Deps{
		Spanner:    spannerClient,
		ReadRouter: app.SpannerRouter,
		Log:        loggingMiddleware,
	})

	// /v1/auth/payloader/login → authroutes package.

	// /v1/payloader/* core plus /v1/payload/seal and /v1/ws/payloader.
	// Ownership lives in backend-go/payloaderroutes.
	payloaderroutes.RegisterRoutes(r, payloaderroutes.Deps{
		Spanner:      spannerClient,
		ReadRouter:   app.SpannerRouter,
		Order:        svc,
		RetailerHub:  retailerHub,
		PayloaderHub: payloaderHub,
		Log:          loggingMiddleware,
	})

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

	// /v1/order/{cash-checkout,card-checkout} moved to retailerroutes.

	// /v1/retailer/card/* moved to retailerroutes.

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

	// /v1/retailer/pending-payments and /v1/retailer/active-fulfillment moved to retailerroutes.

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

	// /v1/payload/seal moved to payloaderroutes.

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

	// /v1/order/{create,cancel} moved to retailerroutes.

	// ── Platform Config (Phase 4.1) — must init before handlers that use it ──
	platformCfg := settings.NewPlatformConfig(spannerClient)

	// ── Refund Endpoint (Phase 3.1) ──
	refundSvc := payment.NewRefundService(spannerClient, platformCfg.PlatformFeeBasisPoints())
	chargebackSvc := payment.NewChargebackService(spannerClient)
	orderroutes.RegisterRoutes(r, orderroutes.Deps{
		Spanner:    spannerClient,
		ReadRouter: app.SpannerRouter,
		Order:      svc,
		Refund:     refundSvc,
		Log:        loggingMiddleware,
	})
	http.HandleFunc("/v1/order/refund", auth.RequireRole([]string{"ADMIN", "SUPPLIER"}, loggingMiddleware(idempotency.Guard(func(w http.ResponseWriter, r *http.Request) {
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
	}))))

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

	// /v1/orders and /v1/order/refunds moved to orderroutes.

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

	// /v1/retailers/{id}/orders and /v1/retailer/tracking moved to retailerroutes.

	// ── DDL MIGRATIONS + H3 BACKFILL ─────────────────────────────────────────
	// In-process migrations are gated by MIGRATE_ON_BOOT (default = "true" for
	// dev/CI parity). Production should set MIGRATE_ON_BOOT=false and run
	// `go run ./cmd/migrate` (or the cmd/migrate Cloud Run Job) once per
	// deploy. This eliminates the N-pods-racing-DDL-on-cold-start risk that
	// motivated the extraction (see migrations/migrations.go header).
	if os.Getenv("MIGRATE_ON_BOOT") != "false" {
		log.Println("[boot] MIGRATE_ON_BOOT enabled — running in-process Spanner migrations")
		migrations.Run(ctx, opts, dbName, spannerClient)
	} else {
		log.Println("[boot] MIGRATE_ON_BOOT=false — skipping migrations (run cmd/migrate out-of-band)")
	}

	// Seed default admin account if none exist (intentionally outside the
	// migration gate — this is operator bootstrap, not schema DDL).
	auth.SeedDefaultAdmin(ctx, spannerClient)


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
		auth.RequireRole([]string{"ADMIN", "SUPPLIER"},
			loggingMiddleware(idempotency.Guard(svc.HandleClearReturns)),
		),
	)

	// /v1/retailer/settings/auto-order* moved to retailerroutes.

	// /v1/orders/line-items/history moved to orderroutes.

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
	internalKafka.StartApproachConsumer(ctx, retailerHub, fcmClient, spannerClient, cfg.KafkaBrokerAddress)

	// Phase 3b: Driver WebSocket Hub route now mounts via driverroutes.

	// Phase 3c: Payloader WebSocket Hub route now mounts via payloaderroutes.

	// Boot the Notification Dispatcher Consumer (inbox + WS + Telegram for all event types)
	internalKafka.StartNotificationDispatcher(ctx, internalKafka.NotificationDeps{
		RetailerHub:   retailerHub,
		DriverHub:     driverHub,
		PayloaderHub:  payloaderHub,
		SupplierHub:   supplierHub,
		FCM:           fcmClient,
		Telegram:      tgClient,
		SpannerClient: spannerClient,
	}, cfg.KafkaBrokerAddress)

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

	// Boot the Transactional Outbox relay (V.O.I.D. Phase VII).
	app.Outbox.SetOnFailure(func(eventID, aggregateID, topic string, err error) {
		app.WarehouseHub.BroadcastOutboxFailure(eventID, aggregateID, topic, err.Error())
	})
	app.Outbox.Start(ctx)

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

	StartPaymentSessionExpirer(sessionSvc, retailerHub)

	// Boot the Failsafe Transmitter
	internalKafka.InitDLQ(cfg.KafkaBrokerAddress)

	// 4. Initialize the Field General AI Routing Cron
	// Since we are matching the instruction closely:
	go routing.StartCron(ctx, spannerClient, cfg.GoogleMapsAPIKey, cfg.DepotLocation)

	// Boot the Replenishment Engine (4h stock deficit analysis)
	replenishEngine := &replenishment.ReplenishmentEngine{Spanner: spannerClient, Producer: svc.Producer}
	replenishEngine.StartReplenishmentCron()

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

	// 7. Edge 4: Orphaned AIPredictionItems Cleanup (daily)
	StartOrphanedPredictionCleaner(spannerClient)

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

	// ── Stealth Simulation Harness (/v1/internal/sim/) ────────────────────
	// Gated by SIMULATION_ENABLED=true at boot. ADMIN-only.
	simroutes.RegisterRoutes(r, simroutes.Deps{
		Engine: app.Simulation,
		Log:    loggingMiddleware,
	})

	// /v1/fleet/route/{id}/complete moved to fleetroutes.

	// /v1/catalog/* — 5 routes (including /v1/catalog/platform-categories and
	// /v1/catalog/suppliers/search registered below this line in the original
	// file). Ownership lives in backend-go/catalogroutes/routes.go.
	catalogroutes.RegisterRoutes(r, catalogroutes.Deps{Spanner: spannerClient, Log: loggingMiddleware})

	// /v1/retailer/{suppliers,profile} moved to retailerroutes.

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

	// ── Crons: Pull Matrix Aggregator (4h) + SLA Monitor (30min) + CurrentLoad Reset (24h) ──
	StartPullMatrixAggregator(pullMatrixSvc, predictivePushSvc)
	StartFactorySLAMonitor(slaMonitorSvc)
	StartCurrentLoadReset(spannerClient)
	StartCoverageAuditor(spannerClient)

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

	supplierplanningroutes.RegisterRoutes(r, supplierplanningroutes.Deps{
		Spanner:          spannerClient,
		Cache:            app.Cache,
		NetworkOptimizer: networkOptSvc,
		SupplyLanes:      supplyLaneSvc,
		KillSwitch:       killSwitchSvc,
		PullMatrix:       pullMatrixSvc,
		PredictivePush:   predictivePushSvc,
		IsDispatchLocked: dispatchLockSvc.IsDispatchLocked,
		Log:              loggingMiddleware,
		Idempotency:      idempotency.Guard,
	})

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
