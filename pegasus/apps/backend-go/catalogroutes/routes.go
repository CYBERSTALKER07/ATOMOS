// Package catalogroutes owns the public /v1/catalog/* surface that serves
// the retailer mobile apps. Handlers live in backend-go/supplier — this
// package only composes them behind a chi.Router with the caching and
// logging middleware stack.
package catalogroutes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"

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

	r.HandleFunc("/v1/catalog/categories/*",
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

	// Legacy compatibility endpoint used by older clients.
	r.HandleFunc("/v1/products",
		auth.RequireRole([]string{"RETAILER", "ADMIN"}, log(handleLegacyProducts(s))))
}

func handleLegacyProducts(spannerClient *spanner.Client) http.HandlerFunc {
	type variant struct {
		ID            string  `json:"id"`
		Size          string  `json:"size"`
		Pack          string  `json:"pack"`
		PackCount     int64   `json:"pack_count"`
		WeightPerUnit string  `json:"weight_per_unit"`
		Price         float64 `json:"price"`
	}

	type product struct {
		ID               string    `json:"id"`
		Name             string    `json:"name"`
		Description      string    `json:"description"`
		Nutrition        string    `json:"nutrition"`
		ImageURL         string    `json:"image_url"`
		Variants         []variant `json:"variants"`
		SupplierID       string    `json:"supplier_id"`
		SupplierName     string    `json:"supplier_name"`
		SupplierCategory string    `json:"supplier_category"`
		CategoryID       string    `json:"category_id"`
		CategoryName     string    `json:"category_name"`
		SellByBlock      bool      `json:"sell_by_block"`
		UnitsPerBlock    int64     `json:"units_per_block"`
		Price            int64     `json:"price"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if spannerClient == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT sp.SkuId, sp.SupplierId, sp.Name, sp.Description, sp.ImageUrl,
			             sp.SellByBlock, sp.UnitsPerBlock, sp.BasePrice, sp.CategoryId,
			             COALESCE(c.Name, '') AS CategoryName,
			             COALESCE(s.Name, '') AS SupplierName,
			             COALESCE(s.Category, '') AS SupplierCategory
			      FROM SupplierProducts sp
			      LEFT JOIN Suppliers s ON sp.SupplierId = s.SupplierId
			      LEFT JOIN Categories c ON c.CategoryId = sp.CategoryId
			      WHERE sp.IsActive = TRUE
			      ORDER BY sp.Name ASC`,
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var productList []product
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to query products: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var skuID, supplierID, name string
			var desc, imageURL, catID, categoryName, supplierName, supplierCategory spanner.NullString
			var sellByBlock bool
			var unitsPerBlock, basePrice int64

			if err := row.Columns(&skuID, &supplierID, &name, &desc, &imageURL,
				&sellByBlock, &unitsPerBlock, &basePrice, &catID, &categoryName, &supplierName, &supplierCategory); err != nil {
				log.Printf("Failed to parse product row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			p := product{
				ID:            skuID,
				Name:          name,
				SellByBlock:   sellByBlock,
				UnitsPerBlock: unitsPerBlock,
				Price:         basePrice,
				SupplierID:    supplierID,
			}
			if desc.Valid {
				p.Description = desc.StringVal
			}
			if imageURL.Valid {
				p.ImageURL = imageURL.StringVal
			}
			if catID.Valid {
				p.CategoryID = catID.StringVal
			}
			if categoryName.Valid {
				p.CategoryName = categoryName.StringVal
			}
			if supplierName.Valid {
				p.SupplierName = supplierName.StringVal
			}
			if supplierCategory.Valid {
				p.SupplierCategory = supplierCategory.StringVal
			}

			packLabel := "Per unit"
			if sellByBlock && unitsPerBlock > 1 {
				packLabel = fmt.Sprintf("Block of %d", unitsPerBlock)
			}
			p.Variants = []variant{{
				ID:            skuID,
				Size:          "Standard",
				Pack:          packLabel,
				PackCount:     1,
				WeightPerUnit: "1 unit",
				Price:         float64(basePrice),
			}}

			productList = append(productList, p)
		}

		if productList == nil {
			productList = []product{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(productList); err != nil {
			log.Printf("Failed to write products response payload: %v", err)
		}
	}
}
