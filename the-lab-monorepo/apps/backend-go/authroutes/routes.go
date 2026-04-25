package authroutes

import (
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/factory"
	"backend-go/supplier"
	"backend-go/warehouse"
)

// Register mounts the full /v1/auth/* surface onto r. Every public
// login/register endpoint is wrapped in the caller-supplied Log middleware;
// the rate-limited subset (admin/driver/supplier/retailer/warehouse login &
// register) additionally goes through Deps.RateLimit.
//
// Routes mounted:
//
//	POST /v1/auth/login               — legacy web retailer login
//	POST /v1/auth/refresh             — token refresh (24h grace)
//	POST /v1/auth/factory/refresh     — factory native refresh alias
//	POST /v1/auth/warehouse/refresh   — warehouse native refresh alias
//	POST /v1/auth/driver/login        — driver PIN auth (rate-limited)
//	POST /v1/auth/admin/login         — admin email+password (rate-limited)
//	POST /v1/auth/admin/register      — admin self-registration (rate-limited)
//	POST /v1/auth/supplier/login      — supplier phone+password (rate-limited)
//	POST /v1/auth/supplier/register   — supplier registration (rate-limited)
//	POST /v1/auth/retailer/login      — retailer phone+password (rate-limited)
//	POST /v1/auth/retailer/register   — retailer registration (rate-limited)
//	POST /v1/auth/payloader/login     — payloader PIN auth
//	POST /v1/auth/factory/login       — factory login
//	POST /v1/auth/factory/register    — factory register (SUPPLIER/ADMIN-gated)
//	POST /v1/auth/warehouse/login     — warehouse login (rate-limited)
//	POST /v1/auth/warehouse/register  — warehouse register (SUPPLIER/ADMIN-gated)
func Register(r chi.Router, deps Deps) {
	s := deps.Spanner
	log := deps.Log
	rl := deps.RateLimit

	// Legacy web retailer login + generic refresh (no rate-limiter — matches
	// the original main.go placement).
	refresh := log(auth.HandleTokenRefresh())
	r.HandleFunc("/v1/auth/login", log(handleLegacyRetailerLogin(deps.RetailerStatus)))
	r.HandleFunc("/v1/auth/refresh", refresh)
	r.HandleFunc("/v1/auth/factory/refresh", refresh)
	r.HandleFunc("/v1/auth/warehouse/refresh", refresh)

	// Rate-limited login/register pairs.
	r.HandleFunc("/v1/auth/driver/login", rl(log(supplier.HandleDriverLogin(s))))
	r.HandleFunc("/v1/auth/admin/login", rl(log(auth.HandleAdminLogin(s))))
	r.HandleFunc("/v1/auth/admin/register", rl(log(auth.HandleAdminRegister(s))))
	r.HandleFunc("/v1/auth/supplier/login", rl(log(supplier.HandleSupplierLogin(s))))
	r.HandleFunc("/v1/auth/supplier/register", rl(log(supplier.HandleSupplierRegister(s))))
	r.HandleFunc("/v1/auth/retailer/login", rl(log(supplier.HandleRetailerLogin(s))))
	r.HandleFunc("/v1/auth/retailer/register", rl(log(supplier.HandleRetailerRegister(s))))
	r.HandleFunc("/v1/auth/warehouse/login", rl(log(warehouse.HandleWarehouseLogin(s))))

	// No-rate-limit payloader + factory login (matches original placement).
	r.HandleFunc("/v1/auth/payloader/login", log(supplier.HandlePayloaderLogin(s)))
	r.HandleFunc("/v1/auth/factory/login", log(factory.HandleFactoryLogin(s)))

	// Role-gated registration endpoints (SUPPLIER or ADMIN required).
	r.HandleFunc("/v1/auth/factory/register",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, log(factory.HandleFactoryRegister(s))))
	r.HandleFunc("/v1/auth/warehouse/register",
		auth.RequireRole([]string{"SUPPLIER", "ADMIN"}, log(warehouse.HandleWarehouseRegister(s))))
}
