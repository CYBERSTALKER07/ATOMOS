// Package suppliercoreroutes owns the remaining supplier core portal surface:
// dashboard, earnings, inventory, and supplier order vetting. Handler logic
// remains in backend-go/order, backend-go/analytics, and backend-go/supplier.
package suppliercoreroutes

import (
	"encoding/json"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/analytics"
	"backend-go/auth"
	"backend-go/order"
	"backend-go/proximity"
	"backend-go/supplier"
	"backend-go/ws"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles collaborators needed to mount the supplier core routes.
type Deps struct {
	Spanner      *spanner.Client
	ReadRouter   proximity.ReadRouter
	Order        *order.OrderService
	Vetting      *supplier.OrderVettingService
	SupplierHub  *ws.SupplierHub
	WarehouseHub *ws.WarehouseHub
	Log          Middleware
	Idempotency  Middleware
}

// RegisterRoutes mounts the supplier core surface:
//
//	GET /v1/supplier/dashboard       — supplier dashboard metrics
//	GET /v1/supplier/earnings        — supplier earnings analytics
//	GET/PATCH /v1/supplier/inventory — inventory list/adjustment
//	GET/POST /v1/supplier/inventory/import — import session list/create
//	GET /v1/supplier/inventory/import/upload-ticket — spreadsheet upload ticket
//	GET /v1/supplier/inventory/import/{id} — import session detail
//	GET /v1/supplier/inventory/import/{id}/rows — staged row pagination
//	PATCH /v1/supplier/inventory/import/{id}/mapping — mapping updates
//	POST /v1/supplier/inventory/import/{id}/{approve,apply} — import actions
//	GET /v1/supplier/inventory/import/{id}/status — import status
//	POST /v1/supplier/inventory/imports/ — phase-2 sandbox session init + upload ticket
//	GET /v1/supplier/inventory/imports/{id} — phase-2 sandbox session status
//	POST /v1/supplier/inventory/imports/{id}/uploaded — phase-3 upload bridge signal + discovery event emit
//	GET /v1/supplier/inventory/imports/{id}/rows — phase-5 sandbox staged-row preview pagination
//	GET /v1/supplier/inventory/imports/{id}/mapping — phase-5 sandbox mapping read for AI review
//	POST /v1/supplier/inventory/imports/{id}/mapping — phase-2 sandbox mapping submission
//	POST /v1/supplier/inventory/imports/{id}/approve — phase-2 sandbox approve transition
//	POST /v1/supplier/inventory/imports/{id}/apply — phase-6 sandbox atomic apply into production inventory
//	GET /v1/supplier/inventory/audit — inventory audit log
//	GET /v1/supplier/orders          — supplier order queue
//	POST /v1/supplier/orders/vet     — approve/reject supplier order
func RegisterRoutes(r chi.Router, d Deps) {
	supplierRole := []string{"SUPPLIER", "ADMIN"}
	log := d.Log
	idem := d.Idempotency
	withRegionScope := auth.RequireRegionScopeWithClient(d.Spanner)
	importHandler := withMethodIdempotency(supplier.HandleInventoryImports(d.Spanner), idem, http.MethodPost, http.MethodPatch)

	r.HandleFunc("/v1/supplier/dashboard",
		auth.RequireRole(supplierRole, log(withRegionScope(dashboardHandler(d.Order)))))
	r.HandleFunc("/v1/supplier/earnings",
		auth.RequireRole(supplierRole, log(withRegionScope(analytics.HandleSupplierEarnings(d.Spanner, d.ReadRouter)))))
	r.HandleFunc("/v1/supplier/inventory",
		auth.RequireRole(supplierRole, log(withRegionScope(withMethodIdempotency(supplier.HandleInventory(d.Spanner), idem, http.MethodPatch)))))
	r.HandleFunc("/v1/supplier/inventory/import",
		auth.RequireRole(supplierRole, log(withRegionScope(importHandler))))
	r.HandleFunc("/v1/supplier/inventory/import/*",
		auth.RequireRole(supplierRole, log(withRegionScope(importHandler))))
	registerImportRoutes(r, d, supplierRole, log, withRegionScope, idem)
	r.HandleFunc("/v1/supplier/inventory/audit",
		auth.RequireRole(supplierRole, log(withRegionScope(supplier.HandleInventoryAuditLog(d.Spanner)))))
	r.HandleFunc("/v1/supplier/orders",
		auth.RequireRole(supplierRole, log(withRegionScope(d.Vetting.HandleSupplierOrders))))
	r.HandleFunc("/v1/supplier/orders/vet",
		auth.RequireRole(supplierRole, log(withRegionScope(idem(d.Vetting.HandleVetOrder)))))
}

func withMethodIdempotency(next http.HandlerFunc, middleware Middleware, methods ...string) http.HandlerFunc {
	if middleware == nil || len(methods) == 0 {
		return next
	}

	allowed := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		allowed[method] = struct{}{}
	}

	guarded := middleware(next)
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := allowed[r.Method]; ok {
			guarded(w, r)
			return
		}
		next(w, r)
	}
}

func dashboardHandler(orderSvc *order.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metrics, err := orderSvc.GetSupplierMetrics(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}
