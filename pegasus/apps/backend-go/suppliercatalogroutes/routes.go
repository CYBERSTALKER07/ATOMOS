// Package suppliercatalogroutes owns the supplier catalog and pricing route
// composition that serves the supplier portal's products, pricing, and
// per-retailer override surfaces. Handler bodies live in backend-go/supplier.
package suppliercatalogroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount the supplier catalog surface.
type Deps struct {
	Spanner         *spanner.Client
	Pricing         *supplier.PricingService
	RetailerPricing *supplier.RetailerPricingService
	Log             Middleware
}

// RegisterRoutes mounts the supplier catalog and pricing surface:
//
//	GET /v1/supplier/products/upload-ticket              — direct-upload signed URL
//	GET/POST /v1/supplier/products                       — product list/create
//	GET/PUT/DELETE /v1/supplier/products/{sku_id}        — product detail/update/deactivate
//	GET/POST /v1/supplier/pricing/rules                  — pricing rule list/upsert
//	DELETE /v1/supplier/pricing/rules/{tier_id}          — pricing rule deactivate
//	GET/POST /v1/supplier/pricing/retailer-overrides     — retailer override list/create
//	DELETE /v1/supplier/pricing/retailer-overrides/{id}  — retailer override deactivate
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	supplierRole := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/supplier/products/upload-ticket",
		auth.RequireRole(supplierRole, log(supplierUploadTicketHandler())))
	r.HandleFunc("/v1/supplier/products",
		auth.RequireRole(supplierRole, log(supplierProductsHandler(d.Spanner))))
	r.HandleFunc("/v1/supplier/products/",
		auth.RequireRole(supplierRole, log(supplierProductDetailHandler(d.Spanner))))
	r.HandleFunc("/v1/supplier/pricing/rules",
		auth.RequireRole(supplierRole, log(d.Pricing.HandleUpsertPricingRule)))
	r.HandleFunc("/v1/supplier/pricing/rules/",
		auth.RequireRole(supplierRole, log(d.Pricing.HandlePricingRuleAction)))
	r.HandleFunc("/v1/supplier/pricing/retailer-overrides",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(d.RetailerPricing.HandleRetailerPricingOverrides))))
	r.HandleFunc("/v1/supplier/pricing/retailer-overrides/",
		auth.RequireRole(supplierRole, log(auth.RequireWarehouseScope(d.RetailerPricing.HandleRetailerPricingOverrideAction))))
}

func supplierUploadTicketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplier.HandleGetUploadTicket(w, r)
	}
}

func supplierProductsHandler(spannerClient *spanner.Client) http.HandlerFunc {
	listProducts := supplier.HandleListSupplierProducts(spannerClient)
	createProduct := supplier.HandleCreateProduct(spannerClient)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listProducts(w, r)
		case http.MethodPost:
			createProduct(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func supplierProductDetailHandler(spannerClient *spanner.Client) http.HandlerFunc {
	getProduct := supplier.HandleGetProduct(spannerClient)
	updateProduct := supplier.HandleUpdateProduct(spannerClient)
	deactivateProduct := supplier.HandleDeactivateProduct(spannerClient)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getProduct(w, r)
		case http.MethodPut:
			updateProduct(w, r)
		case http.MethodDelete:
			deactivateProduct(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
