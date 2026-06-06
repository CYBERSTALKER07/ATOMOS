// Package catalogroutes mounts the catalog URL surface onto the chi router.
// Handlers live in the catalog package; this file is thin by design.
package catalogroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service *catalog.Service
}

// RegisterRoutes mounts catalog read/write endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	r.Get("/v1/catalog/categories", d.Service.HandleListCategories)
	r.Post("/v1/catalog/categories", d.Service.HandleCreateCategory)
	r.Get("/v1/catalog/categories/{categoryID}/suppliers", d.Service.HandleListCategorySuppliers)
	r.Get("/v1/catalog/suppliers/search", d.Service.HandleSearchSuppliers)
	r.Get("/v1/products", d.Service.HandleListProductsAlias)
	r.Get("/v1/catalog/products", d.Service.HandleListProducts)
	r.Post("/v1/catalog/products", d.Service.HandleCreateProduct)
	r.Get("/v1/catalog/products/{productID}", d.Service.HandleGetProduct)
	r.Put("/v1/catalog/products/{productID}", d.Service.HandleUpdateProduct)
}
