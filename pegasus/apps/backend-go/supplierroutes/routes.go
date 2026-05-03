// Package supplierroutes owns the supplier self-service setup surface — the
// exact-match /v1/supplier/* endpoints for onboarding completion, billing
// setup, profile CRUD, shift settings, and payment gateway onboarding.
//
// Handler bodies live in backend-go/supplier and backend-go/vault. This
// package is a thin composer that mounts them behind the SUPPLIER / ADMIN role
// guard and the caller-supplied logging middleware.
package supplierroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/payment"
	"backend-go/supplier"
	"backend-go/vault"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount the supplier self-service routes.
type Deps struct {
	Spanner      *spanner.Client
	Cache        *cache.Cache
	CacheFlight  *singleflight.Group
	DirectClient *payment.GlobalPayDirectClient
	Log          Middleware
}

// RegisterRoutes mounts the supplier self-service setup surface:
//
//	POST /v1/supplier/configure                  — onboarding completion
//	POST /v1/supplier/billing/setup              — bank + gateway setup
//	GET/PUT /v1/supplier/profile                 — supplier profile
//	PATCH /v1/supplier/shift                     — shift settings
//	GET/POST/DELETE /v1/supplier/payment-config  — gateway vault configs
//	GET/POST/DELETE /v1/supplier/gateway-onboarding — gateway connect sessions
//	POST /v1/supplier/payment/recipient/register — Global Pay recipient onboarding
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/configure",
		auth.RequireRole(supplierRole, log(supplier.HandleSupplierConfigure(d.Spanner))))
	r.HandleFunc("/v1/supplier/billing/setup",
		auth.RequireRole(supplierRole, log(supplier.HandleBillingSetup(d.Spanner))))
	r.HandleFunc("/v1/supplier/profile",
		auth.RequireRole(supplierRole, log(supplierProfileHandler(d))))
	r.HandleFunc("/v1/supplier/shift",
		auth.RequireRole(supplierRole, log(supplier.HandleSupplierShift(d.Spanner))))
	r.HandleFunc("/v1/supplier/payment-config",
		auth.RequireRole(supplierRole, log(vault.HandlePaymentConfigs(d.Spanner))))
	r.HandleFunc("/v1/supplier/gateway-onboarding",
		auth.RequireRole(supplierRole, log(vault.HandleGatewayOnboarding(d.Spanner))))
	r.HandleFunc("/v1/supplier/payment/recipient/register",
		auth.RequireRole(supplierRole, log(vault.HandleRegisterRecipient(d.Spanner, d.DirectClient))))
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
