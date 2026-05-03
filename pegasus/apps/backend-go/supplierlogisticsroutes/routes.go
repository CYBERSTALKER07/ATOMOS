// Package supplierlogisticsroutes owns the supplier logistics route
// composition for picking manifests, truck manifests, manifest exceptions, and
// dispatch bridge flows. Handler bodies live in backend-go/supplier.
package supplierlogisticsroutes

import (
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/dispatch/optimizerclient"
	"backend-go/dispatch/plan"
	"backend-go/idempotency"
	"backend-go/proximity"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount supplier logistics routes.
type Deps struct {
	Spanner     *spanner.Client
	ReadRouter  proximity.ReadRouter
	ManifestSvc *supplier.ManifestService
	Optimizer   *optimizerclient.Client
	Counters    *plan.SourceCounters
	Log         Middleware
}

// RegisterRoutes mounts the supplier logistics surface:
//
//	GET /v1/supplier/picking-manifests                 — aggregated daily pick list
//	GET /v1/supplier/picking-manifests/orders          — manifest order rows
//	GET /v1/supplier/manifests                         — supplier truck manifest list
//	GET /v1/supplier/manifests/{id}                    — supplier truck manifest detail
//	POST /v1/supplier/manifests/{id}/start-loading     — DRAFT → LOADING
//	POST /v1/supplier/manifests/{id}/seal              — LOADING → SEALED
//	POST /v1/supplier/manifests/{id}/inject-order      — mid-load order injection
//	POST /v1/payload/manifest-exception                — payloader manifest exception
//	GET /v1/supplier/manifest-exceptions               — exception queue
//	POST /v1/supplier/manifests/auto-dispatch          — auto-dispatch execution
//	POST /v1/supplier/manifests/dispatch-recommend     — dry-run recommendation
//	POST /v1/supplier/manifests/manual-dispatch        — manual manifest creation
//	GET /v1/supplier/manifests/waiting-room            — post-snapshot waiting room
//	GET /v1/supplier/fleet-volumetrics                 — fleet capacity vs backlog
//	POST /v1/supplier/dispatch-queue                   — ready-for-dispatch queue execution
//	GET /v1/supplier/dispatch-preview                  — H3 dispatch preview
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	supplierRole := []string{"SUPPLIER", "ADMIN"}
	supplierOrPayload := []string{"SUPPLIER", "PAYLOADER", "ADMIN"}
	supplierOrPayloadException := []string{"SUPPLIER", "ADMIN", "PAYLOAD"}

	r.HandleFunc("/v1/supplier/picking-manifests",
		auth.RequireRole(supplierOrPayload, log(supplier.HandleManifests(d.Spanner))))
	r.HandleFunc("/v1/supplier/picking-manifests/orders",
		auth.RequireRole(supplierOrPayload, log(supplier.HandleManifestOrders(d.Spanner))))
	r.HandleFunc("/v1/supplier/manifests",
		auth.RequireRole(supplierRole, log(manifestRootHandler(d.ManifestSvc))))
	r.HandleFunc("/v1/supplier/manifests/",
		auth.RequireRole(supplierRole, log(manifestPathHandler(d.ManifestSvc))))
	r.HandleFunc("/v1/payload/manifest-exception",
		auth.RequireRole(supplierOrPayloadException, log(idempotency.Guard(d.ManifestSvc.HandleManifestException()))))
	r.HandleFunc("/v1/supplier/manifest-exceptions",
		auth.RequireRole(supplierRole, log(d.ManifestSvc.HandleListExceptions())))
	r.HandleFunc("/v1/supplier/manifests/auto-dispatch",
		auth.RequireRole(supplierRole, log(supplier.HandleAutoDispatch(d.Spanner, d.ReadRouter, d.ManifestSvc, d.Optimizer, d.Counters))))
	r.HandleFunc("/v1/supplier/manifests/dispatch-recommend",
		auth.RequireRole(supplierRole, log(supplier.HandleDispatchRecommend(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/manifests/manual-dispatch",
		auth.RequireRole(supplierRole, log(supplier.HandleManualDispatch(d.Spanner, d.ReadRouter, d.ManifestSvc))))
	r.HandleFunc("/v1/supplier/manifests/waiting-room",
		auth.RequireRole(supplierRole, log(supplier.HandleWaitingRoom(d.Spanner))))
	r.HandleFunc("/v1/supplier/fleet-volumetrics",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleFleetVolumetrics(d.Spanner)))))
	r.HandleFunc("/v1/supplier/dispatch-queue",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleDispatchQueue(d.Spanner, d.ReadRouter, d.ManifestSvc, d.Optimizer, d.Counters)))))
	r.HandleFunc("/v1/supplier/dispatch-preview",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleH3RoutePreview(d.Spanner, d.ReadRouter)))))
}

func manifestRootHandler(manifestSvc *supplier.ManifestService) http.HandlerFunc {
	listManifests := manifestSvc.HandleListManifests()

	return func(w http.ResponseWriter, r *http.Request) {
		listManifests(w, r)
	}
}

func manifestPathHandler(manifestSvc *supplier.ManifestService) http.HandlerFunc {
	startLoading := idempotency.Guard(manifestSvc.HandleStartLoading())
	sealManifest := idempotency.Guard(manifestSvc.HandleSealManifest())
	injectOrder := idempotency.Guard(manifestSvc.HandleInjectOrder())
	manifestDetail := manifestSvc.HandleManifestDetail()
	listManifests := manifestSvc.HandleListManifests()

	return func(w http.ResponseWriter, r *http.Request) {
		switch manifestPathKind(r.URL.Path) {
		case manifestPathStartLoading:
			startLoading(w, r)
		case manifestPathSeal:
			sealManifest(w, r)
		case manifestPathInjectOrder:
			injectOrder(w, r)
		case manifestPathDetail:
			manifestDetail(w, r)
		default:
			listManifests(w, r)
		}
	}
}

type manifestPathType string

const (
	manifestPathList         manifestPathType = "list"
	manifestPathDetail       manifestPathType = "detail"
	manifestPathStartLoading manifestPathType = "start-loading"
	manifestPathSeal         manifestPathType = "seal"
	manifestPathInjectOrder  manifestPathType = "inject-order"
)

func manifestPathKind(path string) manifestPathType {
	remainder := strings.TrimPrefix(path, "/v1/supplier/manifests/")
	if remainder == "" {
		return manifestPathList
	}
	remainder = strings.TrimSuffix(remainder, "/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 1 && parts[0] != "" {
		return manifestPathDetail
	}
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return manifestPathList
	}
	switch parts[1] {
	case "start-loading":
		return manifestPathStartLoading
	case "seal":
		return manifestPathSeal
	case "inject-order":
		return manifestPathInjectOrder
	default:
		return manifestPathList
	}
}
