// Package supplierroutes owns extracted supplier-only route composition for the
// self-service setup surface plus warehouse/org/staff operations that were
// previously mounted inline in main.go.
//
// Handler bodies live in backend-go/supplier and backend-go/vault. This
// package is a thin composer that mounts them behind the SUPPLIER / ADMIN role
// guard and the caller-supplied logging middleware.
package supplierroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	kafkago "github.com/segmentio/kafka-go"
	"golang.org/x/sync/singleflight"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/payment"
	"backend-go/supplier"
	"backend-go/vault"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount extracted supplier routes.
type Deps struct {
	Spanner      *spanner.Client
	Cache        *cache.Cache
	CacheFlight  *singleflight.Group
	DirectClient *payment.GlobalPayDirectClient
	Producer     *kafkago.Writer
	Log          Middleware
	Idempotency  Middleware
}

// RegisterRoutes mounts the extracted supplier setup and warehouse-ops surfaces:
//
//	POST /v1/supplier/configure                  — onboarding completion
//	POST /v1/supplier/billing/setup              — bank + gateway setup
//	GET/PUT /v1/supplier/profile                 — supplier profile
//	PATCH /v1/supplier/shift                     — shift settings
//	GET/POST/DELETE /v1/supplier/payment-config  — gateway vault configs
//	GET/POST/DELETE /v1/supplier/gateway-onboarding — gateway connect sessions
//	POST /v1/supplier/payment/recipient/register — Global Pay recipient onboarding
//	GET /v1/supplier/org/members                 — org members
//	POST /v1/supplier/org/members/invite         — org member invite
//	PUT/DELETE /v1/supplier/org/members/{id}     — org member mutation
//	GET/POST /v1/supplier/staff/payloader        — payloader staff list/create
//	POST /v1/supplier/staff/payloader/{id}/rotate-pin — payloader PIN rotation
//	GET /v1/supplier/warehouse-staff             — warehouse staff list
//	PATCH /v1/supplier/warehouse-staff/{id}      — warehouse staff toggle
//	GET/POST /v1/supplier/warehouses             — warehouse list/create
//	GET/PUT/DELETE /v1/supplier/warehouses/{id}  — warehouse detail/update/deactivate
//	POST /v1/supplier/warehouses/{id}/coverage   — warehouse polygon coverage save
//	GET /v1/supplier/warehouse-inflight-vu       — VU guardrail read
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	idem := d.Idempotency
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/configure",
		auth.RequireRole(supplierRole, log(supplier.HandleSupplierConfigure(d.Spanner))))
	r.HandleFunc("/v1/supplier/billing/setup",
		auth.RequireRole(supplierRole, log(idem(supplier.HandleBillingSetup(d.Spanner)))))
	r.HandleFunc("/v1/supplier/profile",
		auth.RequireRole(supplierRole, log(idem(supplierProfileHandler(d)))))
	r.HandleFunc("/v1/supplier/shift",
		auth.RequireRole(supplierRole, log(idem(supplier.HandleSupplierShift(d.Spanner)))))
	r.HandleFunc("/v1/supplier/payment-config",
		auth.RequireRole(supplierRole, log(idem(vault.HandlePaymentConfigs(d.Spanner)))))
	r.HandleFunc("/v1/supplier/gateway-onboarding",
		auth.RequireRole(supplierRole, log(vault.HandleGatewayOnboarding(d.Spanner))))
	r.HandleFunc("/v1/supplier/payment/recipient/register",
		auth.RequireRole(supplierRole, log(vault.HandleRegisterRecipient(d.Spanner, d.DirectClient))))
	r.HandleFunc("/v1/supplier/org/members",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleOrgMembers(d.Spanner)))))
	r.HandleFunc("/v1/supplier/org/members/invite",
		auth.RequireRole(supplierRole, log(idem(supplier.HandleOrgInvite(d.Spanner)))))
	r.HandleFunc("/v1/supplier/org/members/",
		auth.RequireRole(supplierRole, log(idem(supplier.HandleOrgMemberAction(d.Spanner)))))
	r.HandleFunc("/v1/supplier/staff/payloader",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleStaffPayloaders(d.Spanner)))))
	r.HandleFunc("/v1/supplier/staff/payloader/",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandlePayloaderDetail(d.Spanner)))))
	r.HandleFunc("/v1/supplier/warehouse-staff",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleWarehouseStaff(d.Spanner)))))
	r.HandleFunc("/v1/supplier/warehouse-staff/",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleWarehouseStaffToggle(d.Spanner)))))
	r.HandleFunc("/v1/supplier/warehouses",
		auth.RequireRole(supplierRole, log(idem(auth.RequireWarehouseScope(supplier.HandleWarehouses(d.Spanner, d.Producer))))))
	r.HandleFunc("/v1/supplier/warehouses/",
		auth.RequireRole(supplierRole, log(idem(auth.RequireWarehouseScope(supplier.HandleWarehouseByID(d.Spanner, d.Producer))))))
	r.HandleFunc("/v1/supplier/warehouse-inflight-vu",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(supplier.HandleWarehouseInflightVU(d.Spanner)))))
}

func supplierProfileHandler(d Deps) http.HandlerFunc {
	getProfile := supplier.HandleGetSupplierProfile(d.Spanner, d.Cache, d.CacheFlight)
	updateProfile := supplier.HandleUpdateSupplierProfile(d.Spanner, d.Cache)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getProfile(w, r)
		case http.MethodPut:
			updateProfile(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
