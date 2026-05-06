// Package factoryroutes owns the /v1/factory/* surface — seventeen endpoints
// covering the factory portal: analytics, profile, transfer orders,
// truck manifests, scoped fleet view, staff CRUD, the dispatch engine
// (bin-pack + route-optimize), supply-request flow, and the payload override
// surface (manifest rebalance, cancel-transfer, cancel).
//
// Handler bodies live in backend-go/factory, backend-go/warehouse, and
// backend-go/analytics. This package is a thin composer that mounts them
// behind the FACTORY-role guard and the RequireFactoryScope middleware.
//
// Path-prefix routes (/v1/factory/transfers/*, /v1/factory/manifests/*,
// /v1/factory/staff/*, /v1/factory/supply-requests/*) now register on chi
// wildcard mounts so "{id}/{verb}" dispatch remains intact without relying on
// http.DefaultServeMux.
//
// V.O.I.D. Wave B adoption notes:
//   - Factory-scope enforcement: every route (except /v1/factory/transfers/create,
//     which SUPPLIER/ADMIN also reach) passes through auth.RequireFactoryScope
//     so FactoryId is pinned before the handler runs.
//   - Outbox adoption for TransferOrders, ManifestTransitions, and Override
//     events remains inside the factory/warehouse packages (progressive
//     migration, tracked separately). The route layer is the composition
//     seam; the transactional outbox belongs at the mutation site.
package factoryroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	kafka "github.com/segmentio/kafka-go"
	"golang.org/x/sync/singleflight"

	"cloud.google.com/go/spanner"

	"backend-go/analytics"
	"backend-go/auth"
	"backend-go/cache"
	"backend-go/factory"
	"backend-go/idempotency"
	"backend-go/proximity"
	"backend-go/warehouse"
	"backend-go/ws"
)

// Middleware is the handler-wrap contract supplied by the caller (typically
// main.loggingMiddleware).
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount /v1/factory/*.
//
// TransferSvc and SupplyRequestSvc are shared with the warehouse portal, so
// the caller constructs them once and passes pointers. BatcherService and
// OverrideService are factory-only and instantiated inside RegisterRoutes.
type Deps struct {
	Spanner          *spanner.Client
	ReadRouter       proximity.ReadRouter
	Producer         *kafka.Writer
	TransferSvc      *factory.TransferService
	SupplyRequestSvc *warehouse.SupplyRequestService
	FactoryHub       *ws.FactoryHub
	Cache            *cache.Cache
	CacheFlight      *singleflight.Group
	Log              Middleware
}

// RegisterRoutes mounts the seventeen factory endpoints:
//
//	GET   /v1/factory/analytics/overview     — daily activity + lead time
//	GET   /v1/factory/profile                — factory record
//	GET   /v1/factory/transfers              — list transfer orders
//	GET   /v1/factory/transfers/{id}         — transfer detail
//	POST  /v1/factory/transfers/{id}/{verb}  — transfer state transitions
//	POST  /v1/factory/transfers/{id}/transition — compatibility transition endpoint
//	POST  /v1/factory/transfers/create       — create transfer (FACTORY|SUPPLIER|ADMIN)
//	GET   /v1/factory/manifests              — Loading Bay Kanban
//	GET   /v1/factory/manifests/{id}         — manifest detail + transfers
//	POST  /v1/factory/manifests/{id}/{verb}  — manifest state transitions
//	GET   /v1/factory/fleet/drivers          — scoped driver view
//	GET   /v1/factory/fleet/vehicles         — scoped vehicle view
//	GET   /v1/factory/staff                  — staff list
//	GET   /v1/factory/staff/{id}             — staff detail
//	POST  /v1/factory/dispatch               — bin-pack + route-optimize + LIFO
//	GET   /v1/factory/supply-requests        — list (factory-scoped view)
//	GET   /v1/factory/supply-requests/{id}   — detail / transition
//	POST  /v1/factory/manifests/rebalance    — reassign transfers during LOADING
//	POST  /v1/factory/manifests/cancel-transfer — drop a transfer off a manifest
//	POST  /v1/factory/manifests/cancel       — cancel an entire manifest
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	factoryRole := []string{"FACTORY"}
	factoryCreate := []string{"FACTORY", "SUPPLIER", "ADMIN"}
	withScope := auth.RequireFactoryScope

	batcherSvc := &factory.BatcherService{Spanner: d.Spanner, Producer: d.Producer, FactoryHub: d.FactoryHub}
	overrideSvc := &factory.OverrideService{Spanner: d.Spanner, Producer: d.Producer, FactoryHub: d.FactoryHub}

	// 1. Analytics overview.
	r.HandleFunc("/v1/factory/analytics/overview",
		auth.RequireRole(factoryRole, log(withScope(analytics.HandleFactoryAnalytics(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/factory/dashboard",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryDashboardCompat(d.Spanner)))))

	// 2. Factory profile.
	r.HandleFunc("/v1/factory/profile",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryProfile(d.Spanner, d.Cache, d.CacheFlight)))))

	// 3. List transfer orders (factory view).
	r.HandleFunc("/v1/factory/transfers",
		auth.RequireRole(factoryRole, log(withScope(d.TransferSvc.HandleListTransfers))))

	// 5. Create transfer — allowed to FACTORY, SUPPLIER, and ADMIN.
	//    Registered before the /v1/factory/transfers/* wildcard dispatcher so the
	//    exact-match wins on chi.
	r.HandleFunc("/v1/factory/transfers/create",
		auth.RequireRole(factoryCreate, log(withScope(d.TransferSvc.HandleCreateTransfer))))

	// 4. Transfer detail + state transitions — wildcard path dispatcher.
	r.HandleFunc("/v1/factory/transfers/*",
		auth.RequireRole(factoryRole, log(withScope(transferByID(d.TransferSvc)))))

	// 6. Factory truck manifests (Loading Bay Kanban).
	r.HandleFunc("/v1/factory/manifests",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryManifests(d.Spanner)))))

	// 15–17. Override endpoints — exact paths registered before the manifests/
	//        prefix dispatcher so longest-prefix match resolves to these first.
	r.HandleFunc("/v1/factory/manifests/rebalance",
		auth.RequireRole(factoryRole, log(withScope(overrideSvc.HandleManifestRebalance))))
	r.HandleFunc("/v1/factory/manifests/cancel-transfer",
		auth.RequireRole(factoryRole, log(withScope(overrideSvc.HandleCancelManifestTransfer))))
	r.HandleFunc("/v1/factory/manifests/cancel",
		auth.RequireRole(factoryRole, log(withScope(overrideSvc.HandleCancelManifest))))

	// 7. Manifest detail + state transitions — wildcard path dispatcher.
	r.HandleFunc("/v1/factory/manifests/*",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryManifestTransition(d.Spanner, d.FactoryHub)))))

	// 8–9. Factory-scoped fleet view.
	r.HandleFunc("/v1/factory/fleet",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryFleetCompat(d.Spanner)))))
	r.HandleFunc("/v1/factory/fleet/drivers",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryFleetDrivers(d.Spanner)))))
	r.HandleFunc("/v1/factory/fleet/vehicles",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryFleetVehicles(d.Spanner)))))

	// 10–11. Staff management.
	r.HandleFunc("/v1/factory/staff",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryStaff(d.Spanner)))))
	r.HandleFunc("/v1/factory/staff/*",
		auth.RequireRole(factoryRole, log(withScope(factory.HandleFactoryStaffDetail(d.Spanner)))))

	// 12. Dispatch engine — bin-pack + route-optimize + LIFO load order.
	// Idempotency-guarded: dispatch creates manifests + side-effects; client retry MUST replay
	// the original response rather than create duplicate shipments.
	r.HandleFunc("/v1/factory/dispatch",
		auth.RequireRole(factoryRole, log(idempotency.Guard(withScope(batcherSvc.HandleFactoryDispatch)))))

	// 13. Supply-request list (factory-scoped view).
	r.HandleFunc("/v1/factory/supply-requests",
		auth.RequireRole(factoryRole, log(withScope(d.SupplyRequestSvc.HandleListSupplyRequests))))

	// 14. Supply-request detail + state transition (wildcard path).
	r.HandleFunc("/v1/factory/supply-requests/*",
		auth.RequireRole(factoryRole, log(withScope(supplyRequestByID(d.SupplyRequestSvc)))))

	if d.FactoryHub != nil {
		r.HandleFunc("/v1/ws/factory",
			auth.RequireRole(factoryRole, log(withScope(d.FactoryHub.HandleConnection))))
	}
}

// supplyRequestByID routes GET→detail and PATCH→transition on
// /v1/factory/supply-requests/{id}. Mirrors the warehouse-side dispatcher.
// PATCH is idempotency-guarded so client retries replay the original transition
// instead of double-firing the state machine.
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

// transferByID routes GET→detail and POST→transition on
// /v1/factory/transfers/{id}. Supports both /{id}/{verb} and /{id}/transition.
func transferByID(svc *factory.TransferService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.HandleTransferDetail(w, r)
		case http.MethodPost:
			svc.HandleTransferTransition(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
