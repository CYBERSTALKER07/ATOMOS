// Package supplieroperationsroutes owns the supplier operations route
// composition for fleet management, fulfillment payment trigger, and reverse
// logistics surfaces. Handler bodies live in backend-go/supplier and
// backend-go/order.
package supplieroperationsroutes

import (
	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	kafkago "github.com/segmentio/kafka-go"

	"backend-go/auth"
	"backend-go/idempotency"
	"backend-go/order"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount supplier operations routes.
type Deps struct {
	Spanner  *spanner.Client
	Order    *order.OrderService
	Producer *kafkago.Writer
	Log      Middleware
}

// RegisterRoutes mounts the supplier operations surface:
//
//	GET/POST /v1/supplier/fleet/drivers            — supplier driver list/create
//	GET/PATCH/POST /v1/supplier/fleet/drivers/{id} — driver detail/assign/rotate-pin
//	GET/POST /v1/supplier/fleet/vehicles           — supplier vehicle list/create
//	GET/PATCH/DELETE /v1/supplier/fleet/vehicles/{id} — vehicle detail/mutate
//	POST /v1/supplier/fulfillment/pay              — staggered supplier fulfillment payment
//	GET /v1/supplier/returns                       — damaged returns queue
//	POST /v1/supplier/returns/resolve              — return disposition
//	GET /v1/supplier/quarantine-stock              — depot quarantine stock
//	POST /v1/inventory/reconcile-returns           — reverse-logistics reconciliation
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	returnsSvc := supplier.NewReturnsService(d.Spanner, d.Producer)
	reconcileSvc := supplier.NewReconcileService(d.Spanner, d.Producer)

	r.HandleFunc("/v1/supplier/fleet/drivers",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleFleetDrivers(d.Spanner)))))
	r.HandleFunc("/v1/supplier/fleet/drivers/",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleFleetDriverDetail(d.Spanner)))))
	r.HandleFunc("/v1/supplier/fleet/vehicles",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleVehicles(d.Spanner)))))
	r.HandleFunc("/v1/supplier/fleet/vehicles/",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleVehicleDetail(d.Spanner)))))
	r.HandleFunc("/v1/supplier/fulfillment/pay",
		auth.RequireRole([]string{"SUPPLIER", "DRIVER", "ADMIN"}, log(idempotency.Guard(fulfillmentPayHandler(d.Order)))))
	r.HandleFunc("/v1/supplier/returns",
		auth.RequireRole(supplierRole, log(returnsSvc.HandleReturns)))
	r.HandleFunc("/v1/supplier/returns/resolve",
		auth.RequireRole(supplierRole, log(returnsSvc.HandleResolveReturn)))
	r.HandleFunc("/v1/supplier/quarantine-stock",
		auth.RequireRole(supplierRole, log(reconcileSvc.HandleQuarantineStock)))
	r.HandleFunc("/v1/inventory/reconcile-returns",
		auth.RequireRole(supplierRole, log(reconcileSvc.HandleReconcile)))
}

func fulfillmentPayHandler(orderSvc *order.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		result, err := orderSvc.TriggerSupplierFulfillmentPayment(r.Context(), req.OrderID)
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
	}
}
