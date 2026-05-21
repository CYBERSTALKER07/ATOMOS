// Package supplierroutes mounts the supplier-portal URL surface onto the chi
// router. Handlers live in the supplier package; this file is thin by design.
package supplierroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service   *supplier.Service
	JWTSecret string
}

// RegisterRoutes mounts:
//
//	POST /v1/auth/supplier/register    (public)
//	POST /v1/auth/supplier/login       (public)
//	POST /v1/supplier/configure        (requires session cookie, ADMIN role)
//	POST /v1/supplier/billing/setup    (requires session cookie, ADMIN role)
//	GET/PUT /v1/supplier/profile       (requires session cookie, ADMIN role)
//	GET/PUT /v1/supplier/topology      (requires session cookie, ADMIN role)
//	GET /v1/supplier/dashboard         (requires session cookie, ADMIN role)
//	GET /v1/supplier/earnings          (requires session cookie, ADMIN role)
//	GET/PATCH /v1/supplier/inventory   (requires session cookie, ADMIN role)
//	GET /v1/supplier/inventory/audit   (requires session cookie, ADMIN role)
//	GET /v1/supplier/orders            (requires session cookie, ADMIN role)
//	POST /v1/supplier/orders/vet       (requires session cookie, ADMIN role)
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	r.Post("/v1/auth/supplier/register", d.Service.HandleRegister)
	r.Post("/v1/auth/supplier/login", d.Service.HandleLogin)

	r.Group(func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		gr.Use(auth.RequireRole(auth.RoleAdmin))
		gr.Post("/v1/supplier/configure", d.Service.HandleConfigure)
		gr.Post("/v1/supplier/billing/setup", d.Service.HandleConfigureBilling)
		gr.Get("/v1/supplier/profile", d.Service.HandleProfile)
		gr.Put("/v1/supplier/profile", d.Service.HandleProfile)
		gr.Get("/v1/supplier/topology", d.Service.HandleTopology)
		gr.Put("/v1/supplier/topology", d.Service.HandleTopology)
		gr.Get("/v1/supplier/dashboard", d.Service.HandleDashboard)
		gr.Get("/v1/supplier/earnings", d.Service.HandleEarnings)
		gr.Get("/v1/supplier/inventory", d.Service.HandleInventory)
		gr.Patch("/v1/supplier/inventory", d.Service.HandleInventory)
		gr.Get("/v1/supplier/inventory/audit", d.Service.HandleInventoryAudit)
		gr.Get("/v1/supplier/orders", d.Service.HandleOrders)
		gr.Post("/v1/supplier/orders/vet", d.Service.HandleVetOrder)
	})
}
