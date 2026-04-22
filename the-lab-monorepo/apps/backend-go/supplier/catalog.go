package supplier

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"
	"backend-go/storage"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

// HandleGetUploadTicket grants Next.js the right to upload an image
func HandleGetUploadTicket(w http.ResponseWriter, r *http.Request) {
	var supplierId string
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if ok && claims != nil {
		supplierId = claims.ResolveSupplierID()
	} else {
		val := r.Context().Value("user_id")
		if val != nil {
			supplierId = val.(string)
		} else {
			http.Error(w, "Unauthorized: Context missing", http.StatusUnauthorized)
			return
		}
	}

	extension := r.URL.Query().Get("ext") // e.g., "png" or "jpg"

	if extension == "" {
		http.Error(w, "Missing file extension", http.StatusBadRequest)
		return
	}
	extension = strings.ToLower(extension)
	allowedExts := map[string]bool{"jpg": true, "jpeg": true, "png": true, "webp": true}
	if !allowedExts[extension] {
		http.Error(w, "Unsupported file extension. Allowed: jpg, jpeg, png, webp", http.StatusBadRequest)
		return
	}

	signedUrl, publicUrl, err := storage.GenerateUploadTicket(supplierId, extension)
	if err != nil {
		http.Error(w, "Ticket generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"upload_url": signedUrl,
		"image_url":  publicUrl, // The frontend will hold this and send it to the POST /products route
	})
}

// SupplierProduct payload mapped to your new Spanner DDL
type SupplierProduct struct {
	SkuId           string   `json:"sku_id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	ImageUrl        string   `json:"image_url"`
	CategoryId      string   `json:"category_id"`
	SellByBlock     bool     `json:"sell_by_block"`
	UnitsPerBlock   int64    `json:"units_per_block"`
	BasePrice       int64    `json:"base_price"`
	VolumetricUnit  float64  `json:"volumetric_unit"`     // VU: 1.0 = standard case of 1L water bottles
	MinimumOrderQty int64    `json:"minimum_order_qty"`   // MOQ: minimum units the AI may order at once
	StepSize        int64    `json:"step_size"`           // Order quantity must be a multiple of this (e.g. 24 for a case)
	LengthCM        *float64 `json:"length_cm,omitempty"` // Physical product dimensions (optional)
	WidthCM         *float64 `json:"width_cm,omitempty"`
	HeightCM        *float64 `json:"height_cm,omitempty"`
}

// HandleCreateProduct writes the catalog item to Spanner
func HandleCreateProduct(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var supplierId string
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if ok && claims != nil {
			supplierId = claims.ResolveSupplierID()
		} else {
			val := r.Context().Value("user_id")
			if val != nil {
				supplierId = val.(string)
			} else {
				http.Error(w, "Unauthorized: Context missing", http.StatusUnauthorized)
				return
			}
		}

		var p SupplierProduct
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Malformed product payload", http.StatusBadRequest)
			return
		}
		if err := ensureCanonicalCategoriesSeeded(r.Context(), client); err != nil {
			log.Printf("[SUPPLIER CATALOG] category seed error: %v", err)
			http.Error(w, "Category catalog unavailable", http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(p.Name) == "" {
			http.Error(w, "Product name is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(p.CategoryId) == "" {
			http.Error(w, "category_id is required", http.StatusBadRequest)
			return
		}
		if _, ok := canonicalCategoryIndex[p.CategoryId]; !ok {
			http.Error(w, "Unknown category_id", http.StatusBadRequest)
			return
		}
		if p.BasePrice <= 0 {
			http.Error(w, "base_price must be greater than zero", http.StatusBadRequest)
			return
		}
		if p.ImageUrl != "" && !strings.HasPrefix(p.ImageUrl, "https://") {
			http.Error(w, "image_url must use HTTPS", http.StatusBadRequest)
			return
		}
		if p.UnitsPerBlock <= 0 {
			p.UnitsPerBlock = 1
		}
		// Packaging constraints: StepSize must be >= 1; MOQ must be >= StepSize.
		// These drive the AI Worker's math.Ceil() rounding to prevent fractional-case orders.
		if p.StepSize <= 0 {
			p.StepSize = 1
		}
		if p.MinimumOrderQty <= 0 {
			p.MinimumOrderQty = p.StepSize
		}
		if p.MinimumOrderQty < p.StepSize {
			p.MinimumOrderQty = p.StepSize
		}

		isConfigured, operatingCategories, err := loadSupplierCategoryAccess(r.Context(), client, supplierId)
		if err != nil {
			log.Printf("[SUPPLIER CATALOG] load supplier access error: %v", err)
			http.Error(w, "Supplier profile unavailable", http.StatusInternalServerError)
			return
		}
		if !isConfigured {
			http.Error(w, "Supplier must complete onboarding before adding products", http.StatusForbidden)
			return
		}
		if !containsCategoryID(operatingCategories, p.CategoryId) {
			http.Error(w, "category_id is not enabled for this supplier", http.StatusBadRequest)
			return
		}

		if p.SkuId == "" {
			p.SkuId = uuid.New().String() // Auto-generate if not provided
		}

		// Default VU to 1.0 if not provided (safe baseline = 1 standard case)
		// If physical dimensions are supplied, compute VU precisely: (L×W×H) / 5000 cm³
		if p.LengthCM != nil && p.WidthCM != nil && p.HeightCM != nil {
			computed := (*p.LengthCM * *p.WidthCM * *p.HeightCM) / 5000.0
			if computed > 0 {
				p.VolumetricUnit = computed
			}
		}
		if p.VolumetricUnit <= 0 {
			p.VolumetricUnit = 1.0
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		productCols := []string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl", "CategoryId", "SellByBlock", "UnitsPerBlock", "BasePrice", "VolumetricUnit", "MinimumOrderQty", "StepSize", "IsActive", "CreatedAt"}
		productVals := []interface{}{p.SkuId, supplierId, p.Name, p.Description, p.ImageUrl, p.CategoryId, p.SellByBlock, p.UnitsPerBlock, p.BasePrice, p.VolumetricUnit, p.MinimumOrderQty, p.StepSize, true, spanner.CommitTimestamp}
		if p.LengthCM != nil {
			productCols = append(productCols, "LengthCM", "WidthCM", "HeightCM")
			productVals = append(productVals, *p.LengthCM, *p.WidthCM, *p.HeightCM)
		}

		productMut := spanner.Insert("SupplierProducts", productCols, productVals)

		// Also seed an initial SupplierInventory row with zero stock so
		// checkout doesn't 409 with "no inventory record". Supplier adjusts
		// stock via /v1/supplier/inventory PATCH after product creation.
		inventoryMut := spanner.Insert("SupplierInventory",
			[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
			[]interface{}{p.SkuId, supplierId, int64(0), spanner.CommitTimestamp},
		)

		_, err = client.Apply(ctx, []*spanner.Mutation{productMut, inventoryMut})
		if err != nil {
			http.Error(w, "Ledger write fault", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":        "PRODUCT_LOCKED",
			"sku_id":        p.SkuId,
			"category_id":   p.CategoryId,
			"category_name": categoryDisplayNameByID(p.CategoryId),
		})
	}
}

// HandleSupplierLogin authenticates a supplier with phone + password
// POST /v1/auth/supplier/login
//
// Authentication order:
// 1. Try SupplierUsers table (sub-accounts with explicit GLOBAL_ADMIN / NODE_ADMIN)
// 2. Fall back to Suppliers table (root registrant — implicit GLOBAL_ADMIN)
func HandleSupplierLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phone    string `json:"phone"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Phone == "" || req.Password == "" {
			http.Error(w, `{"error":"phone and password required"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// ── 1. Try SupplierUsers table first (sub-accounts) ────────────
		suStmt := spanner.Statement{
			SQL: `SELECT su.UserId, su.SupplierId, su.Name, su.PasswordHash, su.SupplierRole,
			             COALESCE(su.AssignedWarehouseId, ''),
			             COALESCE(su.AssignedFactoryId, ''),
			             COALESCE(s.CountryCode, 'UZ'),
			             COALESCE(s.IsConfigured, false)
			      FROM SupplierUsers su
			      LEFT JOIN Suppliers s ON su.SupplierId = s.SupplierId
			      WHERE su.Phone = @phone AND su.IsActive = true`,
			Params: map[string]interface{}{"phone": req.Phone},
		}
		suIter := spannerClient.Single().Query(ctx, suStmt)
		suRow, suErr := suIter.Next()
		if suErr == nil {
			var userID, supplierID, name, pwHash, supplierRole, warehouseID, factoryID, countryCode string
			var isCfg bool
			if err := suRow.Columns(&userID, &supplierID, &name, &pwHash, &supplierRole, &warehouseID, &factoryID, &countryCode, &isCfg); err == nil && pwHash != "" {
				suIter.Stop()

				if err := bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(req.Password)); err != nil {
					http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
					return
				}

				// Derive factory role for claims
				factoryRole := ""
				if supplierRole == "FACTORY_ADMIN" || supplierRole == "FACTORY_PAYLOADER" {
					factoryRole = supplierRole
				}

				token, err := auth.MintIdentityToken(&auth.LabClaims{
					UserID:       userID,
					SupplierID:   supplierID,
					Role:         "SUPPLIER",
					SupplierRole: supplierRole,
					WarehouseID:  warehouseID,
					FactoryID:    factoryID,
					FactoryRole:  factoryRole,
					CountryCode:  countryCode,
					IsConfigured: isCfg,
				})
				if err != nil {
					log.Printf("[SUPPLIER AUTH] token generation error: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// Mint Firebase custom token with full scope
				var firebaseToken string
				if auth.FirebaseAuthClient != nil {
					var fbUid string
					_ = spannerClient.Single().Query(ctx, spanner.Statement{
						SQL:    "SELECT COALESCE(FirebaseUid, '') FROM SupplierUsers WHERE UserId = @id",
						Params: map[string]interface{}{"id": userID},
					}).Do(func(row *spanner.Row) error { return row.Columns(&fbUid) })
					if fbUid != "" {
						firebaseToken, _ = auth.MintCustomToken(ctx, fbUid, map[string]interface{}{
							"role":          "SUPPLIER",
							"supplier_id":   supplierID,
							"supplier_role": supplierRole,
							"warehouse_id":  warehouseID,
							"factory_id":    factoryID,
							"factory_role":  factoryRole,
						})
					}
				}

				resp := map[string]interface{}{
					"token":         token,
					"user_id":       userID,
					"supplier_id":   supplierID,
					"role":          "SUPPLIER",
					"supplier_role": supplierRole,
					"warehouse_id":  warehouseID,
					"factory_id":    factoryID,
					"name":          name,
					"is_configured": isCfg,
					"country_code":  countryCode,
				}
				if firebaseToken != "" {
					resp["firebase_token"] = firebaseToken
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}
		}
		suIter.Stop()

		// ── 2. Fall back to Suppliers table (root registrant = implicit GLOBAL_ADMIN) ──
		stmt := spanner.Statement{
			SQL: `SELECT SupplierId, Name, COALESCE(PasswordHash, ''), COALESCE(Category, ''),
			             COALESCE(CountryCode, 'UZ'), COALESCE(IsConfigured, false)
			      FROM Suppliers WHERE Phone = @phone`,
			Params: map[string]interface{}{
				"phone": req.Phone,
			},
		}

		iter := spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("[SUPPLIER AUTH] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var supplierID, name, passwordHash, category, countryCode string
		var isConfigured bool
		if err := row.Columns(&supplierID, &name, &passwordHash, &category, &countryCode, &isConfigured); err != nil {
			log.Printf("[SUPPLIER AUTH] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if passwordHash == "" {
			http.Error(w, `{"error":"no credentials configured"}`, http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		// Root supplier → implicit GLOBAL_ADMIN with empty WarehouseID (all warehouses)
		token, err := auth.MintIdentityToken(&auth.LabClaims{
			UserID:       supplierID,
			SupplierID:   supplierID,
			Role:         "SUPPLIER",
			SupplierRole: "GLOBAL_ADMIN",
			CountryCode:  countryCode,
			IsConfigured: isConfigured,
		})
		if err != nil {
			log.Printf("[SUPPLIER AUTH] token generation error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Mint Firebase custom token (graceful degradation)
		var firebaseToken string
		if auth.FirebaseAuthClient != nil {
			var fbUid string
			_ = spannerClient.Single().Query(ctx, spanner.Statement{
				SQL:    "SELECT COALESCE(FirebaseUid, '') FROM Suppliers WHERE SupplierId = @id",
				Params: map[string]interface{}{"id": supplierID},
			}).Do(func(row *spanner.Row) error { return row.Columns(&fbUid) })
			if fbUid != "" {
				firebaseToken, _ = auth.MintCustomToken(ctx, fbUid, map[string]interface{}{
					"role":          "SUPPLIER",
					"supplier_id":   supplierID,
					"supplier_role": "GLOBAL_ADMIN",
				})
			}
		}

		resp := map[string]interface{}{
			"token":         token,
			"user_id":       supplierID,
			"supplier_id":   supplierID,
			"role":          "SUPPLIER",
			"supplier_role": "GLOBAL_ADMIN",
			"name":          name,
			"category":      category,
			"is_configured": isConfigured,
			"country_code":  countryCode,
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// HandleListSupplierProducts returns products belonging to the authenticated supplier.
// GET /v1/supplier/products
func HandleListSupplierProducts(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		stmt := spanner.Statement{
			SQL: `SELECT sp.SkuId, sp.Name, COALESCE(sp.Description, ''), COALESCE(sp.ImageUrl, ''),
			             sp.SellByBlock, sp.UnitsPerBlock, sp.BasePrice, sp.IsActive, COALESCE(sp.CategoryId, ''),
			             COALESCE(c.Name, ''),
			             COALESCE(sp.VolumetricUnit, COALESCE(sp.PalletFootprint, 1.0)),
			             COALESCE(sp.MinimumOrderQty, 1), COALESCE(sp.StepSize, 1), sp.CreatedAt,
			             sp.LengthCM, sp.WidthCM, sp.HeightCM
			      FROM SupplierProducts sp
			      LEFT JOIN Categories c ON c.CategoryId = sp.CategoryId
			      WHERE SupplierId = @supplierId
			      ORDER BY Name ASC`,
			Params: map[string]interface{}{
				"supplierId": supplierID,
			},
		}

		iter := spannerx.StaleQuery(r.Context(), client, stmt)
		defer iter.Stop()

		type ProductItem struct {
			SkuID           string   `json:"sku_id"`
			Name            string   `json:"name"`
			Description     string   `json:"description"`
			ImageURL        string   `json:"image_url"`
			SellByBlock     bool     `json:"sell_by_block"`
			UnitsPerBlock   int64    `json:"units_per_block"`
			BasePrice       int64    `json:"base_price"`
			IsActive        bool     `json:"is_active"`
			CategoryID      string   `json:"category_id"`
			CategoryName    string   `json:"category_name"`
			VolumetricUnit  float64  `json:"volumetric_unit"`
			MinimumOrderQty int64    `json:"minimum_order_qty"`
			StepSize        int64    `json:"step_size"`
			CreatedAt       string   `json:"created_at"`
			LengthCM        *float64 `json:"length_cm,omitempty"`
			WidthCM         *float64 `json:"width_cm,omitempty"`
			HeightCM        *float64 `json:"height_cm,omitempty"`
		}

		var products []ProductItem
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[SUPPLIER CATALOG] query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var p ProductItem
			var createdAt time.Time
			if err := row.Columns(&p.SkuID, &p.Name, &p.Description, &p.ImageURL,
				&p.SellByBlock, &p.UnitsPerBlock, &p.BasePrice, &p.IsActive, &p.CategoryID, &p.CategoryName, &p.VolumetricUnit, &p.MinimumOrderQty, &p.StepSize, &createdAt,
				&p.LengthCM, &p.WidthCM, &p.HeightCM); err != nil {
				log.Printf("[SUPPLIER CATALOG] parse error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			p.CreatedAt = createdAt.Format(time.RFC3339)
			products = append(products, p)
		}

		if products == nil {
			products = []ProductItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": products,
		})
	}
}

// HandleRetailerLogin authenticates a retailer with phone + password.
// POST /v1/auth/retailer/login  →  { phone_number, password }
func HandleRetailerLogin(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			PhoneNumber string `json:"phone_number"`
			Password    string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.PhoneNumber == "" || req.Password == "" {
			http.Error(w, `{"error":"phone_number and password required"}`, http.StatusBadRequest)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT RetailerId, Name, COALESCE(PasswordHash, ''), COALESCE(Status, ''), COALESCE(ShopName, '')
			      FROM Retailers WHERE Phone = @phone`,
			Params: map[string]interface{}{
				"phone": req.PhoneNumber,
			},
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("[RETAILER AUTH] query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var retailerID, name, passwordHash, status, shopName string
		if err := row.Columns(&retailerID, &name, &passwordHash, &status, &shopName); err != nil {
			log.Printf("[RETAILER AUTH] parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if passwordHash == "" {
			http.Error(w, `{"error":"no credentials configured"}`, http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}

		if status != "VERIFIED" {
			http.Error(w, `{"error":"account not verified - check KYC status"}`, http.StatusForbidden)
			return
		}

		token, err := auth.GenerateTestToken(retailerID, "RETAILER")
		if err != nil {
			log.Printf("[RETAILER AUTH] token generation error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Mint Firebase custom token (graceful degradation)
		var firebaseToken string
		if auth.FirebaseAuthClient != nil {
			// Retailer uses phone-based Firebase identity — look up by phone via Spanner FirebaseUid
			var fbUid string
			_ = spannerClient.Single().Query(r.Context(), spanner.Statement{
				SQL:    "SELECT COALESCE(FirebaseUid, '') FROM Retailers WHERE RetailerId = @id",
				Params: map[string]interface{}{"id": retailerID},
			}).Do(func(row *spanner.Row) error { return row.Columns(&fbUid) })
			if fbUid != "" {
				firebaseToken, _ = auth.MintCustomToken(r.Context(), fbUid, map[string]interface{}{"role": "RETAILER", "retailer_id": retailerID})
			}
		}

		resp := map[string]interface{}{
			"token": token,
			"user": map[string]interface{}{
				"id":         retailerID,
				"name":       name,
				"company":    shopName,
				"email":      "",
				"avatar_url": nil,
			},
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
