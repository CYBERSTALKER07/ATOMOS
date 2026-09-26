// Package catalogroutes mounts the catalog URL surface onto the chi router.
// Handlers live in the catalog package; this file is thin by design.
package catalogroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service         *catalog.Service
	AllowAuthBypass bool
}

// RegisterRoutes mounts catalog read/write endpoints. Reads are public catalog
// browse (retailer apps pass supplier_id); writes are SUPPLIER-only and the
// supplier scope is resolved from JWT claims inside the handlers, never the body.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	r.Get("/v1/catalog/categories", d.Service.HandleListCategories)
	r.Get("/v1/catalog/categories/{categoryID}/suppliers", d.Service.HandleListCategorySuppliers)
	r.Get("/v1/catalog/suppliers/search", d.Service.HandleSearchSuppliers)
	r.Get("/v1/products", d.Service.HandleListProductsAlias)
	r.Get("/v1/catalog/products", d.Service.HandleListProducts)
	r.Get("/v1/catalog/products/upload-ticket", d.Service.HandleGetUploadTicket)
	r.Get("/v1/catalog/products/{productID}", d.Service.HandleGetProduct)

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}, func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/catalog/categories", d.Service.HandleCreateCategory)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/catalog/products", d.Service.HandleCreateProduct)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Put("/v1/catalog/products/{productID}", d.Service.HandleUpdateProduct)
	})
}
