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
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles collaborators needed to mount the supplier core routes.
type Deps struct {
	Spanner     *spanner.Client
	ReadRouter  proximity.ReadRouter
	Order       *order.OrderService
	Vetting     *supplier.OrderVettingService
	Log         Middleware
	Idempotency Middleware
}

// RegisterRoutes mounts the supplier core surface:
//
//	GET /v1/supplier/dashboard       — supplier dashboard metrics
//	GET /v1/supplier/earnings        — supplier earnings analytics
//	GET/PATCH /v1/supplier/inventory — inventory list/adjustment
//	GET /v1/supplier/inventory/audit — inventory audit log
//	GET /v1/supplier/orders          — supplier order queue
//	POST /v1/supplier/orders/vet     — approve/reject supplier order
func RegisterRoutes(r chi.Router, d Deps) {
	supplierRole := []string{"SUPPLIER", "ADMIN"}
	log := d.Log
	idem := d.Idempotency

	r.HandleFunc("/v1/supplier/dashboard",
		auth.RequireRole(supplierRole, log(dashboardHandler(d.Order))))
	r.HandleFunc("/v1/supplier/earnings",
		auth.RequireRole(supplierRole, log(analytics.HandleSupplierEarnings(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/supplier/inventory",
		auth.RequireRole(supplierRole, log(supplier.HandleInventory(d.Spanner))))
	r.HandleFunc("/v1/supplier/inventory/audit",
		auth.RequireRole(supplierRole, log(supplier.HandleInventoryAuditLog(d.Spanner))))
	r.HandleFunc("/v1/supplier/orders",
		auth.RequireRole(supplierRole, log(d.Vetting.HandleSupplierOrders)))
	r.HandleFunc("/v1/supplier/orders/vet",
		auth.RequireRole(supplierRole, log(idem(d.Vetting.HandleVetOrder))))
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
