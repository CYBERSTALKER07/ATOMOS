// Package bootstrap is the composition root for the backend-go service.
//
// It owns the construction of every long-lived client, service, and in-memory
// hub used by the HTTP routes — Spanner, Redis, Kafka, GCS, Firebase, the
// WebSocket hubs, the OrderService, the payment reconcilers, etc. — and
// hands them to main() as a single App struct.
//
// main() is responsible only for:
//  1. Loading config
//  2. Calling auth.Init(cfg.JWTSecret, cfg.InternalAPIKey)
//  3. Calling NewApp(ctx, cfg)
//  4. Mounting route registrations onto the router
//  5. Starting cron/Kafka consumers
//  6. Calling srv.ListenAndServe() + graceful shutdown
//
// Lifecycle starts (cron sweepers, Kafka consumers) are kept in main for this
// phase. Migration into App.StartBackground() is a follow-up.
package bootstrap

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/option"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/countrycfg"
	"backend-go/notifications"
	"backend-go/order"
	"backend-go/outbox"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/replenishment"
	"backend-go/secrets"
	"backend-go/settings"
	"backend-go/supplier"
	"backend-go/telemetry"
	"backend-go/vault"
	"backend-go/ws"
	"config"

	"backend-go/bootstrap/spannerrouter"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/internal/rpc/optimizergrpc"
	"backend-go/simulation"
)

// App is the fully-initialised set of clients, services, and hubs the HTTP
// handlers depend on. Construction order matters (see NewApp); field
// population is exhaustive — a handler can expect every pointer to be non-nil
// unless the field's doc string says otherwise.
type App struct {
	// Config is the parsed environment schema. Handlers should read config
	// through this pointer, not via os.Getenv.
	Config *config.EnvConfig

	// ── Core infrastructure ──────────────────────────────────────────────
	Spanner *spanner.Client // Google Cloud Spanner client (always non-nil)

	// SpannerRouter routes read-only Spanner queries to the nearest regional
	// replica when enable_multiregion=true. Writes must always use
	// SpannerRouter.Primary() (same as Spanner above). Handlers that know
	// an entity's H3 cell call SpannerRouter.For(h3Cell) to get the closest
	// read client. Handlers without location context use Primary().
	// In single-region mode, For() and Primary() both return Spanner.
	SpannerRouter *spannerrouter.Router
	Cache         *cache.Cache // Redis handle (nil-safe client in degraded mode)
	// CacheFlight coalesces identical in-flight cache-miss reads so that N
	// concurrent requests for the same cold key produce exactly one Spanner
	// query instead of N.
	CacheFlight *singleflight.Group
	// SpannerDBName is the fully-qualified database URI
	// (projects/.../instances/.../databases/...). Surfaced for DDL admin
	// operations that construct their own admin clients.
	SpannerDBName string
	// SpannerClientOpts are the option.ClientOption values used to dial
	// Spanner (emulator endpoint, insecure gRPC, no-auth). Reused when
	// booting DatabaseAdminClient for inline DDL migrations.
	SpannerClientOpts []option.ClientOption
	// KafkaBroker is the broker address used by consumer starters in main.
	// Kafka producers (sync, correction, DLQ) live as package-level writers
	// inside backend-go/kafka — bootstrap owns their Init() lifecycle.
	KafkaBroker string

	// ── Payment + vault services ────────────────────────────────────────
	Vault        *vault.Service
	SessionSvc   *payment.SessionService
	CardTokenSvc *payment.CardTokenService
	CardsClient  *payment.GlobalPayCardsClient  // nil if GLOBAL_PAY_GATEWAY_BASE_URL unset
	DirectClient *payment.GlobalPayDirectClient // nil if GLOBAL_PAY_GATEWAY_BASE_URL unset
	GPReconciler *payment.GlobalPayReconciler

	// ── Domain services ─────────────────────────────────────────────────
	Order             *order.OrderService
	CountryConfig     *countrycfg.Service
	PlatformConfig    *settings.PlatformConfig
	Empathy           *settings.EmpathyService
	SupplierPricing   *supplier.PricingService
	ProximityEngine   *proximity.Engine
	Replenishment     *replenishment.ReplenishmentEngine
	Broadcast         *notifications.BroadcastService
	Reconciler        *payment.ReconcilerService
	ShopClosedDeps    order.ShopClosedDeps
	EarlyCompleteDeps order.EarlyCompleteDeps
	NegotiationDeps   order.NegotiationDeps

	// ── Communication spine ─────────────────────────────────────────────
	FCM      *notifications.FCMClient
	Telegram *notifications.TelegramClient

	// ── WebSocket hubs ──────────────────────────────────────────────────
	RetailerHub  *ws.RetailerHub
	DriverHub    *ws.DriverHub
	PayloaderHub *ws.PayloaderHub
	SupplierHub  *ws.SupplierHub
	WarehouseHub *ws.WarehouseHub
	FactoryHub   *ws.FactoryHub
	FleetHub     *telemetry.Hub // shared package-level hub; field is a convenience alias

	// ── Transactional Outbox relay (V.O.I.D. Phase VII) ─────────────────
	// Outbox tails OutboxEvents and publishes to Kafka. The relay goroutine
	// is started by main() after NewApp returns, via Outbox.Start(ctx).
	Outbox *outbox.Relay

	// ── Phase 2 dispatch optimiser (apps/ai-worker) ─────────────────────
	// OptimizerClient is the typed VRP solver client. Nil when
	// OPTIMIZER_BASE_URL is unset — dispatch.plan.OptimizeAndValidate
	// treats nil as "fall straight through to Phase 1 fallback", so
	// nilness is the documented degraded mode, not an error.
	OptimizerClient *optimizerclient.Client

	// OptimizerGRPC is the gRPC replacement for OptimizerClient. When
	// OPTIMIZER_GRPC_ADDR is set (or "xds" to enable mesh routing), this
	// client is preferred over OptimizerClient. Nil = use HTTP client.
	OptimizerGRPC *optimizergrpc.GRPCClient

	// DispatchOptimizer tracks per-source call attribution. Always non-nil;
	// safe to read concurrently. Populated by dispatch shadow + primary paths.
	DispatchOptimizer *plan.SourceCounters

	// ── Stealth simulation harness (SIMULATION_ENABLED=true) ────────────
	// Nil when the env flag is unset — routes are not mounted in that case.
	// When armed, exposes /v1/internal/sim/{start,stop,status} to ADMIN.
	Simulation *simulation.Engine

	// ── OpenTelemetry TracerProvider shutdown ────────────────────────────
	// TracerShutdown flushes pending OTel spans. MUST be called from the
	// SIGTERM handler before pod exit — losing crash-path spans is the
	// "silent P0" the doctrine calls out.
	TracerShutdown telemetry.ShutdownFunc

	// ── Middleware + traffic control ────────────────────────────────────
	Backpressure *cache.BackpressureEngine
	// PriorityGuard is the priority-aware load-shedding HTTP middleware.
	// Wrap a handler: app.PriorityGuard(h).
	PriorityGuard func(http.HandlerFunc) http.HandlerFunc

	// CORSAllowlist is the origin set resolved at boot (cfg value if
	// non-empty, otherwise localhost dev defaults).
	CORSAllowlist map[string]bool
}

// Close tears down long-lived resources in reverse construction order.
// Safe to call multiple times; each underlying Close is idempotent.
func (a *App) Close() {
	if a == nil {
		return
	}
	if a.Cache != nil {
		a.Cache.Close()
	}
	if a.Backpressure != nil {
		a.Backpressure.Stop()
	}
	if a.Spanner != nil {
		a.Spanner.Close()
	}
	secrets.Close()

	// WebSocket hubs — inform connected clients of shutdown.
	if a.RetailerHub != nil {
		a.RetailerHub.Close()
	}
	if a.DriverHub != nil {
		a.DriverHub.Close()
	}
	if a.PayloaderHub != nil {
		a.PayloaderHub.Close()
	}
	if a.SupplierHub != nil {
		a.SupplierHub.Close()
	}
	if a.WarehouseHub != nil {
		a.WarehouseHub.Close()
	}
	if a.FactoryHub != nil {
		a.FactoryHub.Close()
	}
	if a.FleetHub != nil {
		a.FleetHub.Close()
	}

	// Firebase Admin SDK manages its own lifecycle; nothing to close here.
	_ = auth.FirebaseAuthClient
}
