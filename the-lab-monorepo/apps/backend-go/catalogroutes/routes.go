// Package catalogroutes owns the public /v1/catalog/* surface that serves
// the retailer mobile apps. Handlers live in backend-go/supplier — this
// package only composes them behind a chi.Router with the caching and
// logging middleware stack.
package catalogroutes

import (
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register /v1/catalog routes.
type Deps struct {
	Spanner *spanner.Client
	Log     Middleware
}

// RegisterRoutes mounts the retailer-facing catalog discovery surface:
//
//	GET /v1/catalog/platform-categories — categories for supplier onboarding (public)
//	GET /v1/catalog/categories          — retailer-visible categories (5m cache)
//	GET /v1/catalog/categories/{id}     — suppliers serving this category
//	GET /v1/catalog/products            — product listing (60s cache)
//	GET /v1/catalog/suppliers/search    — supplier name search
func RegisterRoutes(r chi.Router, d Deps) {
	s := d.Spanner
	log := d.Log
	retailer := []string{"RETAILER"}

	r.HandleFunc("/v1/catalog/platform-categories",
		log(supplier.HandleListPlatformCategories(s)))

	r.HandleFunc("/v1/catalog/categories",
		auth.RequireRole(retailer,
			log(cache.CacheHandler(cache.PrefixCacheCategories, 5*time.Minute,
				supplier.HandleListCategories(s)))))

	r.HandleFunc("/v1/catalog/categories/",
		auth.RequireRole(retailer,
			log(cache.CacheHandler(cache.PrefixCategorySuppliers, cache.TTLCategorySuppliers,
				supplier.HandleListCategorySuppliers(s)))))

	r.HandleFunc("/v1/catalog/products",
		auth.RequireRole(retailer,
			log(cache.CacheHandler(cache.PrefixCacheProducts, 60*time.Second,
				supplier.HandleListCatalogProducts(s)))))

	r.HandleFunc("/v1/catalog/suppliers/search",
		auth.RequireRole(retailer,
			log(cache.CacheHandler(cache.PrefixCatalogSearch, cache.TTLCatalogSearch,
				supplier.HandleCatalogSearch(s)))))
}
