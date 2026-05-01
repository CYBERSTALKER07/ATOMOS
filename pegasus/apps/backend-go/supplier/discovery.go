package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/iterator"
)

// ── Catalog Discovery Handlers (Retailer-facing) ──────────────────────────

type CategoryResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Icon          string `json:"icon"`
	ProductCount  int64  `json:"product_count"`
	SupplierCount int64  `json:"supplier_count"`
}

type SupplierResponse struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	LogoURL                string   `json:"logo_url"`
	Category               string   `json:"category"`
	PrimaryCategoryID      string   `json:"primary_category_id,omitempty"`
	OperatingCategoryIDs   []string `json:"operating_category_ids,omitempty"`
	OperatingCategoryNames []string `json:"operating_category_names,omitempty"`
	ProductCount           int64    `json:"product_count,omitempty"`
	OrderCount             int64    `json:"order_count,omitempty"`
}

// HandleListCategories returns all product categories with counts.
// GET /v1/catalog/categories
func HandleListCategories(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		if err := ensureCanonicalCategoriesSeeded(ctx, client); err != nil {
			log.Printf("[catalog] Failed to seed categories: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Get categories with product count from SupplierProducts
		stmt := spanner.Statement{
			SQL: `SELECT c.CategoryId, c.Name, c.Icon,
			             (SELECT COUNT(*) FROM SupplierProducts sp WHERE sp.CategoryId = c.CategoryId AND sp.IsActive = TRUE) AS ProductCount,
			             (SELECT COUNT(DISTINCT sp.SupplierId) FROM SupplierProducts sp WHERE sp.CategoryId = c.CategoryId AND sp.IsActive = TRUE) AS SupplierCount
			      FROM Categories c
			      ORDER BY c.SortOrder ASC`,
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var categories []CategoryResponse
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[catalog] Failed to query categories: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var c CategoryResponse
			var icon spanner.NullString
			if err := row.Columns(&c.ID, &c.Name, &icon, &c.ProductCount, &c.SupplierCount); err != nil {
				log.Printf("[catalog] Failed to parse category row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if icon.Valid {
				c.Icon = icon.StringVal
			}
			categories = append(categories, c)
		}

		if categories == nil {
			categories = []CategoryResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(categories)
	}
}

// HandleListCatalogProducts returns filtered products.
// GET /v1/catalog/products?category_id=X&supplier_id=Y
func HandleListCatalogProducts(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		if err := ensureCanonicalCategoriesSeeded(ctx, client); err != nil {
			log.Printf("[catalog] Failed to seed categories: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		categoryID := r.URL.Query().Get("category_id")
		supplierID := r.URL.Query().Get("supplier_id")

		sql := `SELECT sp.SkuId, sp.SupplierId, sp.Name, sp.Description, sp.ImageUrl,
		               sp.SellByBlock, sp.UnitsPerBlock, sp.BasePrice, sp.CategoryId,
		               COALESCE(c.Name, '') AS CategoryName,
		               COALESCE(s.Name, '') AS SupplierName,
		               COALESCE(s.Category, '') AS SupplierCategory
		        FROM SupplierProducts sp
		        LEFT JOIN Suppliers s ON sp.SupplierId = s.SupplierId
		        LEFT JOIN Categories c ON c.CategoryId = sp.CategoryId
		        WHERE sp.IsActive = TRUE`

		params := map[string]interface{}{}
		if categoryID != "" {
			sql += ` AND sp.CategoryId = @categoryId`
			params["categoryId"] = categoryID
		}
		if supplierID != "" {
			sql += ` AND sp.SupplierId = @supplierId`
			params["supplierId"] = supplierID
		}
		sql += ` ORDER BY sp.Name ASC`

		stmt := spanner.Statement{SQL: sql, Params: params}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		type ProductRow struct {
			ID               string `json:"id"`
			SupplierID       string `json:"supplier_id"`
			SupplierName     string `json:"supplier_name"`
			SupplierCategory string `json:"supplier_category"`
			Name             string `json:"name"`
			Description      string `json:"description"`
			ImageURL         string `json:"image_url"`
			SellByBlock      bool   `json:"sell_by_block"`
			UnitsPerBlock    int64  `json:"units_per_block"`
			Price            int64  `json:"price"`
			CategoryID       string `json:"category_id"`
			CategoryName     string `json:"category_name"`
		}

		var products []ProductRow
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[catalog] Failed to query products: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var p ProductRow
			var desc, imageUrl, catId, categoryName, supplierName, supplierCategory spanner.NullString
			if err := row.Columns(&p.ID, &p.SupplierID, &p.Name, &desc, &imageUrl,
				&p.SellByBlock, &p.UnitsPerBlock, &p.Price, &catId, &categoryName, &supplierName, &supplierCategory); err != nil {
				log.Printf("[catalog] Failed to parse product row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if desc.Valid {
				p.Description = desc.StringVal
			}
			if imageUrl.Valid {
				p.ImageURL = imageUrl.StringVal
			}
			if catId.Valid {
				p.CategoryID = catId.StringVal
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
			products = append(products, p)
		}

		if products == nil {
			products = []ProductRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	}
}

// HandleListCategorySuppliers returns suppliers that have products in a given category.
// GET /v1/catalog/categories/{id}/suppliers
func HandleListCategorySuppliers(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract category ID from path: /v1/catalog/categories/{id}/suppliers
		path := r.URL.Path
		// path = /v1/catalog/categories/cat-123/suppliers
		parts := splitPath(path)
		if len(parts) < 5 {
			http.Error(w, "Missing category ID", http.StatusBadRequest)
			return
		}
		categoryID := parts[3] // ["v1", "catalog", "categories", "{id}", "suppliers"]

		ctx := r.Context()
		if err := ensureCanonicalCategoriesSeeded(ctx, client); err != nil {
			log.Printf("[catalog] Failed to seed categories: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT DISTINCT s.SupplierId, s.Name, s.LogoUrl, s.Category, COALESCE(s.OperatingCategories, []),
			             (SELECT COUNT(*) FROM SupplierProducts sp2
			              WHERE sp2.SupplierId = s.SupplierId AND sp2.CategoryId = @categoryId AND sp2.IsActive = TRUE) AS ProductCount,
			             IFNULL(s.ManualOffShift, false),
			             COALESCE(TO_JSON_STRING(s.OperatingSchedule), '{}')
			      FROM Suppliers s
			      JOIN SupplierProducts sp ON s.SupplierId = sp.SupplierId
			      WHERE sp.CategoryId = @categoryId AND sp.IsActive = TRUE
			      ORDER BY s.Name ASC`,
			Params: map[string]interface{}{
				"categoryId": categoryID,
			},
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		type SupplierWithCount struct {
			ID                     string   `json:"id"`
			Name                   string   `json:"name"`
			LogoURL                string   `json:"logo_url"`
			Category               string   `json:"category"`
			PrimaryCategoryID      string   `json:"primary_category_id,omitempty"`
			OperatingCategoryIDs   []string `json:"operating_category_ids,omitempty"`
			OperatingCategoryNames []string `json:"operating_category_names,omitempty"`
			ProductCount           int64    `json:"product_count"`
			IsActive               bool     `json:"is_active"`
		}

		now := time.Now()
		var suppliers []SupplierWithCount
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[catalog] Failed to query category suppliers: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var s SupplierWithCount
			var logoUrl, category spanner.NullString
			var operatingCategoryIDs []string
			var manualOff bool
			var schedJSON string
			if err := row.Columns(&s.ID, &s.Name, &logoUrl, &category, &operatingCategoryIDs, &s.ProductCount, &manualOff, &schedJSON); err != nil {
				log.Printf("[catalog] Failed to parse supplier row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if logoUrl.Valid {
				s.LogoURL = logoUrl.StringVal
			}
			if category.Valid {
				s.Category = category.StringVal
			}
			s.OperatingCategoryIDs = operatingCategoryIDs
			s.OperatingCategoryNames = categoryDisplayNames(operatingCategoryIDs)
			if len(operatingCategoryIDs) > 0 {
				s.PrimaryCategoryID = operatingCategoryIDs[0]
			}
			s.IsActive = resolveIsActive(schedJSON, manualOff, now)
			suppliers = append(suppliers, s)
		}

		if suppliers == nil {
			suppliers = []SupplierWithCount{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(suppliers)
	}
}

// splitPath splits a URL path into parts, stripping the leading slash.
func splitPath(path string) []string {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	parts := []string{}
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// HandleCatalogSearch searches for suppliers by name.
// GET /v1/catalog/suppliers/search?q=<query>
func HandleCatalogSearch(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" || len(q) < 2 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]SupplierResponse{})
			return
		}

		ctx := r.Context()
		pattern := "%" + q + "%"

		stmt := spanner.Statement{
			SQL: `SELECT s.SupplierId, s.Name, s.LogoUrl, s.Category,
			             COALESCE(s.OperatingCategories, []),
			             (SELECT COUNT(*) FROM SupplierProducts sp WHERE sp.SupplierId = s.SupplierId AND sp.IsActive = true) AS ProductCount,
			             IFNULL(s.ManualOffShift, false),
			             COALESCE(TO_JSON_STRING(s.OperatingSchedule), '{}')
			      FROM Suppliers s
			      WHERE LOWER(s.Name) LIKE LOWER(@pattern)
			      ORDER BY s.Name
			      LIMIT 20`,
			Params: map[string]interface{}{
				"pattern": pattern,
			},
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		now := time.Now()
		var suppliers []SupplierResponse
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[catalog-search] Failed to query suppliers: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var s SupplierResponse
			var logoUrl, category spanner.NullString
			var operatingCategoryIDs []string
			var manualOff bool
			var schedJSON string
			if err := row.Columns(&s.ID, &s.Name, &logoUrl, &category, &operatingCategoryIDs, &s.ProductCount, &manualOff, &schedJSON); err != nil {
				log.Printf("[catalog-search] Failed to parse supplier row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if logoUrl.Valid {
				s.LogoURL = logoUrl.StringVal
			}
			if category.Valid {
				s.Category = category.StringVal
			}
			s.OperatingCategoryIDs = operatingCategoryIDs
			s.OperatingCategoryNames = categoryDisplayNames(operatingCategoryIDs)
			if len(operatingCategoryIDs) > 0 {
				s.PrimaryCategoryID = operatingCategoryIDs[0]
			}
			_ = resolveIsActive(schedJSON, manualOff, now)
			suppliers = append(suppliers, s)
		}

		if suppliers == nil {
			suppliers = []SupplierResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(suppliers)
	}
}

// HandleRetailerProfile handles GET and PUT for the retailer's own profile.
// GET  /v1/retailer/profile → return profile from Retailers table
// PUT  /v1/retailer/profile → update name, shop name
func HandleRetailerProfile(client *spanner.Client, rc *cache.Cache, flight *singleflight.Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		retailerID := claims.UserID

		switch r.Method {
		case http.MethodGet:
			getRetailerProfileCached(w, r, client, rc, flight, retailerID)
		case http.MethodPut:
			putRetailerProfile(w, r, client, rc, retailerID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getRetailerProfileCached(w http.ResponseWriter, r *http.Request, client *spanner.Client, rc *cache.Cache, flight *singleflight.Group, retailerID string) {
	cacheKey := cache.RetailerProfile(retailerID)

	// Read-through: serve from Redis if warm
	if rc != nil && rc.Client() != nil {
		cacheCtx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		cached, err := rc.Client().Get(cacheCtx, cacheKey).Result()
		cancel()
		if err == nil && cached != "" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write([]byte(cached))
			return
		}
	}

	// Singleflight: coalesce concurrent cache-miss reads
	val, err, _ := flight.Do(cacheKey, func() (interface{}, error) {
		return fetchRetailerProfile(r.Context(), client, retailerID)
	})
	if err != nil {
		log.Printf("[profile] Failed to query profile for %s: %v", retailerID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	body := val.([]byte)

	// Backfill cache
	if rc != nil && rc.Client() != nil {
		go func() {
			setCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			rc.Client().Set(setCtx, cacheKey, string(body), cache.TTLProfile)
			cancel()
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(body)
}

func fetchRetailerProfile(ctx context.Context, client *spanner.Client, retailerID string) ([]byte, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Name, Phone, ShopName, ShopLocation, TaxIdentificationNumber, Status
		      FROM Retailers WHERE RetailerId = @id`,
		Params: map[string]interface{}{"id": retailerID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("retailer %s not found", retailerID)
	}
	if err != nil {
		return nil, fmt.Errorf("query retailer %s: %w", retailerID, err)
	}

	var id, name string
	var phone, shopName, shopLocation, taxID, status spanner.NullString
	if err := row.Columns(&id, &name, &phone, &shopName, &shopLocation, &taxID, &status); err != nil {
		return nil, fmt.Errorf("parse retailer %s: %w", retailerID, err)
	}

	return json.Marshal(map[string]interface{}{
		"id":       id,
		"name":     name,
		"phone":    phone.StringVal,
		"company":  shopName.StringVal,
		"location": shopLocation.StringVal,
		"tax_id":   taxID.StringVal,
		"status":   status.StringVal,
	})
}

func putRetailerProfile(w http.ResponseWriter, r *http.Request, client *spanner.Client, rc *cache.Cache, retailerID string) {
	var req struct {
		Name                 string `json:"name"`
		Company              string `json:"company"`
		Location             string `json:"location"`
		ReceivingWindowOpen  string `json:"receiving_window_open"`
		ReceivingWindowClose string `json:"receiving_window_close"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"RetailerId"}
	vals := []interface{}{retailerID}

	if req.Name != "" {
		cols = append(cols, "Name")
		vals = append(vals, req.Name)
	}
	if req.Company != "" {
		cols = append(cols, "ShopName")
		vals = append(vals, req.Company)
	}
	if req.Location != "" {
		cols = append(cols, "ShopLocation")
		vals = append(vals, req.Location)
	}
	if req.ReceivingWindowOpen != "" {
		canon, err := proximity.ValidateReceivingWindow(req.ReceivingWindowOpen)
		if err != nil {
			http.Error(w, `{"error":"invalid receiving_window_open: expected HH:MM 24-hour format"}`, http.StatusBadRequest)
			return
		}
		cols = append(cols, "ReceivingWindowOpen")
		vals = append(vals, canon)
	}
	if req.ReceivingWindowClose != "" {
		canon, err := proximity.ValidateReceivingWindow(req.ReceivingWindowClose)
		if err != nil {
			http.Error(w, `{"error":"invalid receiving_window_close: expected HH:MM 24-hour format"}`, http.StatusBadRequest)
			return
		}
		cols = append(cols, "ReceivingWindowClose")
		vals = append(vals, canon)
	}

	if len(cols) == 1 {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("Retailers", cols, vals),
		})
	})
	if err != nil {
		log.Printf("[profile] Failed to update profile for %s: %v", retailerID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Invalidate cached profile after successful write
	rc.Invalidate(ctx, cache.RetailerProfile(retailerID))

	// Return updated profile (fresh from Spanner)
	body, fetchErr := fetchRetailerProfile(ctx, client, retailerID)
	if fetchErr != nil {
		log.Printf("[profile] Failed to refetch profile for %s: %v", retailerID, fetchErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
