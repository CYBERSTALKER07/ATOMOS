// Package warehouseroutes owns the /v1/warehouse/* surface — twenty-eight
// endpoints spanning the inbound-transfer acceptance flow (emergency,
// standard receive, force-receive), replenishment insights + actions,
// demand forecast, the supply-request CRUD + state machine, the
// WAREHOUSE_ADMIN Ops Portal (dashboard, fleet, staff, orders,
// dispatch-preview, inventory, products, manifests, analytics, CRM,
// returns, treasury, payment-config), and the dispatch-lock system.
//
// Handler bodies live in backend-go/warehouse, backend-go/factory, and
// backend-go/replenishment. This package is a thin composer that mounts
// them behind the WAREHOUSE / SUPPLIER / ADMIN role guards and the
// caller-supplied observability middleware.
//
// V.O.I.D. Wave B adoption notes:
//   - Scope enforcement: non-ops warehouse routes pass through
//     auth.RequireWarehouseScope so WarehouseId is pinned before the
//     handler runs. Ops routes layer auth.RequireWarehouseOpsScope on
//     top for the WAREHOUSE_ADMIN operator surface.
//   - Path-prefix dispatchers (/v1/warehouse/transfers/*,
//     /v1/warehouse/replenishment/insights/*,
//     /v1/warehouse/supply-requests/*, /v1/warehouse/ops/{drivers,
//     vehicles,staff,orders}/*) register on chi wildcard mounts so
//     {id}/{verb} sub-path dispatch remains intact on chi-native
//     routing.
//   - Outbox adoption for transfer-receive, supply-request transitions,
//     and dispatch-lock mutations remains inside the factory/warehouse
//     packages — progressive migration, tracked separately.
package warehouseroutes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	kafka "github.com/segmentio/kafka-go"

	"cloud.google.com/go/spanner"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/factory"
	"backend-go/idempotency"
	"backend-go/order"
	"backend-go/proximity"
	"backend-go/replenishment"
	"backend-go/warehouse"
	"backend-go/ws"
)

// Middleware is the handler-wrap contract supplied by the caller (typically
// main.loggingMiddleware).
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount /v1/warehouse/*.
//
// TransferSvc and SupplyReqSvc are shared with the factory portal, so
// the caller constructs them once and passes pointers. EmergencySvc,
// ForceReceiveSvc, and DispatchLockSvc are warehouse-only but kept in
// Deps (rather than instantiated here) so tests can inject fakes.
type Deps struct {
	Spanner         *spanner.Client
	Producer        *kafka.Writer
	TransferSvc     *factory.TransferService
	EmergencySvc    *factory.EmergencyTransferService
	ForceReceiveSvc *factory.ForceReceiveService
	SupplyReqSvc    *warehouse.SupplyRequestService
	DispatchLockSvc *warehouse.DispatchLockService
	OrderSvc        *order.OrderService
	ReadRouter      proximity.ReadRouter
	Optimizer       *optimizerclient.Client
	DispatchCounts  *plan.SourceCounters
	WarehouseHub    *ws.WarehouseHub
	Log             Middleware
	Cache           *cache.Cache
}

// RegisterRoutes mounts the twenty-eight warehouse endpoints:
//
//	POST  /v1/warehouse/transfers/emergency        — emergency transfer request
//	POST  /v1/warehouse/transfers/{id}/receive     — receive inbound transfer
//	POST  /v1/warehouse/transfers/force-receive    — DLQ reconciliation
//	GET   /v1/warehouse/replenishment/insights     — deficit insights
//	POST  /v1/warehouse/replenishment/insights/{id}/{verb}
//	GET   /v1/warehouse/demand/forecast            — 7-day demand curve
//	GET/POST /v1/warehouse/supply-requests         — list + create
//	GET/PATCH /v1/warehouse/supply-requests/{id}   — detail + transition
//	GET   /v1/warehouse/ops/dashboard              — ops KPIs
//	GET   /v1/warehouse/ops/drivers{,/{id}}        — driver roster + detail
//	GET   /v1/warehouse/ops/vehicles{,/{id}}       — vehicle roster + detail
//	GET   /v1/warehouse/ops/staff{,/{id}}          — staff roster + detail
//	GET   /v1/warehouse/ops/orders{,/{id}}         — order list + detail
//	POST  /v1/warehouse/ops/dispatch/preview       — bin-pack preview
//	GET   /v1/warehouse/ops/inventory              — stock snapshot
//	GET   /v1/warehouse/ops/products               — catalog (read-only)
//	GET   /v1/warehouse/ops/manifests              — manifest list
//	GET   /v1/warehouse/ops/analytics              — ops analytics
//	GET   /v1/warehouse/ops/crm                    — retailer relationship
//	GET   /v1/warehouse/ops/returns                — returns workbench
//	GET   /v1/warehouse/ops/treasury               — cash / settlement view
//	GET   /v1/warehouse/ops/payment-config         — payment config (read-only)
//	POST/DELETE /v1/warehouse/dispatch-lock        — acquire / release
//	GET   /v1/warehouse/dispatch-locks             — list active locks
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	warehouseScoped := []string{"SUPPLIER", "ADMIN"}
	warehouseTriad := []string{"WAREHOUSE", "SUPPLIER", "ADMIN"}
	warehouseOnly := []string{"WAREHOUSE"}
	withScope := auth.RequireWarehouseScopeWithClient(d.Spanner)

	// whOps layers the WAREHOUSE role guard + RequireWarehouseOpsScope
	// middleware for the WAREHOUSE_ADMIN operator surface.
	whOps := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireRole(warehouseOnly, log(auth.RequireWarehouseOpsScope(d.Spanner, next)))
	}

	// 1. Emergency transfer request (exact path — wins over the
	//    /v1/warehouse/transfers/ prefix dispatcher below).
	r.HandleFunc("/v1/warehouse/transfers/emergency",
		auth.RequireRole(warehouseScoped, log(withScope(d.EmergencySvc.HandleEmergencyTransfer))))

	// 3. Force-receive DLQ reconciliation (exact path, registered before
	//    the prefix dispatcher for clarity).
	r.HandleFunc("/v1/warehouse/transfers/force-receive",
		auth.RequireRole(warehouseScoped, log(withScope(d.ForceReceiveSvc.HandleForceReceive))))

	// 2. Standard inbound-transfer receive — wildcard dispatcher on
	//    /v1/warehouse/transfers/{id}/receive.
	r.HandleFunc("/v1/warehouse/transfers/*",
		auth.RequireRole(warehouseScoped, log(withScope(d.TransferSvc.HandleWarehouseReceiveTransfer))))

	// 4. Replenishment insights (exact) + action dispatcher (wildcard).
	r.HandleFunc("/v1/warehouse/replenishment/insights",
		auth.RequireRole(warehouseScoped, log(withScope(replenishment.HandleInsights(d.Spanner)))))
	r.HandleFunc("/v1/warehouse/replenishment/insights/*",
		auth.RequireRole(warehouseScoped, log(withScope(replenishment.HandleInsightAction(d.Spanner, d.Producer)))))

	// 6. Demand forecast.
	r.HandleFunc("/v1/warehouse/demand/forecast",
		auth.RequireRole(warehouseTriad, log(warehouse.HandleDemandForecast(d.Spanner, d.ReadRouter))))

	// 7. Supply-request list + create (exact).
	r.HandleFunc("/v1/warehouse/supply-requests",
		auth.RequireRole(warehouseTriad, log(supplyRequestList(d.SupplyReqSvc))))
	// 8. Supply-request detail + transition (wildcard dispatcher).
	r.HandleFunc("/v1/warehouse/supply-requests/*",
		auth.RequireRole(warehouseTriad, log(supplyRequestByID(d.SupplyReqSvc))))

	// 9-26. Warehouse Ops Portal (WAREHOUSE_ADMIN scope).
	r.HandleFunc("/v1/warehouse/ops/dashboard", whOps(warehouse.HandleDashboard(d.Spanner)))

	r.HandleFunc("/v1/warehouse/ops/drivers", whOps(warehouse.HandleOpsDrivers(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/drivers/*", whOps(warehouse.HandleOpsDriverDetail(d.Spanner)))

	r.HandleFunc("/v1/warehouse/ops/vehicles", whOps(warehouse.HandleOpsVehicles(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/vehicles/*", whOps(warehouse.HandleOpsVehicleDetail(d.Spanner)))

	r.HandleFunc("/v1/warehouse/ops/staff", whOps(warehouse.HandleOpsStaff(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/staff/*", whOps(warehouse.HandleOpsStaffDetail(d.Spanner)))

	r.HandleFunc("/v1/warehouse/ops/orders", whOps(warehouse.HandleOpsOrders(d.Spanner, d.ReadRouter)))
	// Phase V LEO: ops marks an order DELAYED (capacity overflow / hold).
	// Registered as an exact chi route BEFORE the /v1/warehouse/ops/orders/*
	// prefix dispatcher so longest-match selects this for /{id}/delay.
	if d.OrderSvc != nil {
		r.Post("/v1/warehouse/ops/orders/{id}/delay", whOps(d.OrderSvc.HandleMarkDelayed()))
		r.Post("/v1/warehouse/ops/orders/{id}/reject", whOps(d.OrderSvc.HandleOrderRejection()))
		r.Post("/v1/warehouse/ops/orders/{id}/overflow", whOps(d.OrderSvc.HandlePayloadOverflow()))
	}
	r.HandleFunc("/v1/warehouse/ops/orders/*", whOps(warehouse.HandleOpsOrderDetail(d.Spanner, d.ReadRouter)))

	r.HandleFunc("/v1/warehouse/ops/dispatch/preview", whOps(warehouse.HandleOpsDispatchPreview(d.Spanner, d.Optimizer, d.DispatchCounts)))
	r.HandleFunc("/v1/warehouse/ops/inventory", whOps(warehouse.HandleOpsInventory(d.Spanner, d.Cache)))
	r.HandleFunc("/v1/warehouse/ops/products", whOps(warehouse.HandleOpsProducts(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/manifests", whOps(warehouse.HandleOpsManifests(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/analytics", whOps(warehouse.HandleOpsAnalytics(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/crm", whOps(warehouse.HandleOpsCRM(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/returns", whOps(warehouse.HandleOpsReturns(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/treasury", whOps(warehouse.HandleOpsTreasury(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/financials", whOps(warehouse.HandleWarehouseFinancials(d.Spanner)))
	r.HandleFunc("/v1/warehouse/ops/payment-config", whOps(warehouse.HandleOpsPaymentConfig(d.Spanner)))

	// 27-28. Dispatch Lock System.
	// Idempotency-guarded: lock acquire/release are mutations whose retry must not produce
	// duplicate locks or release a lock twice (race window with AI-worker).
	r.HandleFunc("/v1/warehouse/dispatch-lock",
		auth.RequireRole(warehouseTriad, log(idempotency.Guard(dispatchLockHandler(d.DispatchLockSvc)))))
	r.HandleFunc("/v1/warehouse/dispatch-locks",
		auth.RequireRole(warehouseTriad, log(d.DispatchLockSvc.HandleListDispatchLocks)))

	if d.WarehouseHub != nil {
		r.HandleFunc("/ws/warehouse",
			auth.RequireRoleWithGrace([]string{"WAREHOUSE", "SUPPLIER", "ADMIN"}, 2*time.Hour, d.WarehouseHub.HandleConnection))
	}
}

// supplyRequestList routes GET→list and POST→create on
// /v1/warehouse/supply-requests.
func supplyRequestList(svc *warehouse.SupplyRequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.HandleListSupplyRequests(w, r)
		case http.MethodPost:
			svc.HandleCreateSupplyRequest(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// supplyRequestByID routes GET→detail and PATCH→transition on
// /v1/warehouse/supply-requests/{id}. The PATCH branch is idempotency-guarded
// because state transitions (PENDING→APPROVED→DISPATCHED→RECEIVED) must not
// double-fire on client retry; GET passes through unchanged.
func supplyRequestByID(svc *warehouse.SupplyRequestService) http.HandlerFunc {
	guardedTransition := idempotency.Guard(svc.HandleSupplyRequestTransition)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.HandleSupplyRequestDetail(w, r)
		case http.MethodPatch:
			guardedTransition(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// dispatchLockHandler routes POST→acquire and DELETE→release on
// /v1/warehouse/dispatch-lock.
func dispatchLockHandler(svc *warehouse.DispatchLockService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			svc.HandleAcquireDispatchLock(w, r)
		case http.MethodDelete:
			svc.HandleReleaseDispatchLock(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
