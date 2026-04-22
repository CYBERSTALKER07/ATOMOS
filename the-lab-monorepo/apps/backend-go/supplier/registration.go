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

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/iterator"
)

// HandleSupplierRegister creates a new supplier account with full onboarding in one step.
// POST /v1/auth/supplier/register → { phone, password, company_name, email, categories, tax_id }
func HandleSupplierRegister(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Phone                   string   `json:"phone"`
			Password                string   `json:"password"`
			CompanyName             string   `json:"company_name"`
			Email                   string   `json:"email"`
			Categories              []string `json:"categories"`
			TaxId                   string   `json:"tax_id"`
			ContactPerson           string   `json:"contact_person"`
			CompanyRegNumber        string   `json:"company_reg_number"`
			BillingAddress          string   `json:"billing_address"`
			WarehouseAddress        string   `json:"warehouse_address"`
			WarehouseLat            float64  `json:"warehouse_lat"`
			WarehouseLng            float64  `json:"warehouse_lng"`
			CountryCode             string   `json:"country_code"`
			FleetColdChainCompliant bool     `json:"fleet_cold_chain_compliant"`
			PalletizationStandard   string   `json:"palletization_standard"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.Phone == "" || req.Password == "" || req.CompanyName == "" {
			http.Error(w, `{"error":"phone, password, and company_name are required"}`, http.StatusBadRequest)
			return
		}
		if len(req.Password) < 8 {
			http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
			return
		}
		if len(req.Categories) == 0 {
			// Categories are now optional at registration — supplier can configure later
			req.Categories = []string{}
		}
		if err := ensureCanonicalCategoriesSeeded(r.Context(), spannerClient); err != nil {
			log.Printf("[SUPPLIER REGISTER] category seed error: %v", err)
			http.Error(w, `{"error":"category catalog unavailable"}`, http.StatusInternalServerError)
			return
		}

		var validCategories, invalidCategories []string
		if len(req.Categories) > 0 {
			validCategories, invalidCategories = normalizeValidCategoryIDs(req.Categories)
			if len(invalidCategories) > 0 {
				http.Error(w, fmt.Sprintf(`{"error":"invalid categories: %s"}`, strings.Join(invalidCategories, ", ")), http.StatusBadRequest)
				return
			}
		}

		// Check if phone already exists
		stmt := spanner.Statement{
			SQL:    `SELECT SupplierId FROM Suppliers WHERE Phone = @phone LIMIT 1`,
			Params: map[string]interface{}{"phone": req.Phone},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		row, err := iter.Next()
		iter.Stop()
		if row != nil && err == nil {
			http.Error(w, `{"error":"phone number already registered"}`, http.StatusConflict)
			return
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[SUPPLIER REGISTER] bcrypt error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		supplierId := uuid.New().String()
		isConfigured := len(validCategories) > 0 && req.TaxId != ""
		primaryCategory := primaryCategoryName(validCategories)

		// Normalise country code — default to UZ if not provided
		countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
		if countryCode == "" || len(countryCode) != 2 {
			countryCode = "UZ"
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cols := []string{"SupplierId", "Name", "Phone", "PasswordHash", "Category", "IsConfigured", "OperatingCategories", "CountryCode", "CreatedAt"}
		vals := []interface{}{supplierId, req.CompanyName, req.Phone, string(hash), primaryCategory, isConfigured, validCategories, countryCode, spanner.CommitTimestamp}

		if req.Email != "" {
			cols = append(cols, "Email")
			vals = append(vals, req.Email)
		}
		if req.TaxId != "" {
			cols = append(cols, "TaxId")
			vals = append(vals, req.TaxId)
		}
		if req.ContactPerson != "" {
			cols = append(cols, "ContactPerson")
			vals = append(vals, req.ContactPerson)
		}
		if req.CompanyRegNumber != "" {
			cols = append(cols, "CompanyRegNumber")
			vals = append(vals, req.CompanyRegNumber)
		}
		if req.BillingAddress != "" {
			cols = append(cols, "BillingAddress")
			vals = append(vals, req.BillingAddress)
		}
		if req.WarehouseAddress != "" {
			cols = append(cols, "WarehouseLocation")
			vals = append(vals, req.WarehouseAddress)
		}
		if req.WarehouseLat != 0 || req.WarehouseLng != 0 {
			cols = append(cols, "WarehouseLat", "WarehouseLng")
			vals = append(vals, req.WarehouseLat, req.WarehouseLng)
		}
		if req.FleetColdChainCompliant {
			cols = append(cols, "FleetColdChainCompliant")
			vals = append(vals, req.FleetColdChainCompliant)
		}
		if req.PalletizationStandard != "" {
			cols = append(cols, "PalletizationStandard")
			vals = append(vals, req.PalletizationStandard)
		}

		m := spanner.Insert("Suppliers", cols, vals)
		if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{m}); err != nil {
			log.Printf("[SUPPLIER REGISTER] spanner insert error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// ── T=0 Mirror: root supplier → SupplierUsers as GLOBAL_ADMIN ────
		// Unifies the identity table from day one. No lazy-mirror needed.
		mirrorID := uuid.New().String()
		mirrorCols := []string{"UserId", "SupplierId", "Name", "Phone", "PasswordHash",
			"SupplierRole", "IsActive", "CreatedAt"}
		mirrorVals := []interface{}{mirrorID, supplierId, req.CompanyName, req.Phone, string(hash),
			"GLOBAL_ADMIN", true, spanner.CommitTimestamp}
		if req.Email != "" {
			mirrorCols = append(mirrorCols, "Email")
			mirrorVals = append(mirrorVals, req.Email)
		}
		mirrorMut := spanner.Insert("SupplierUsers", mirrorCols, mirrorVals)
		if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{mirrorMut}); err != nil {
			// Non-fatal: root still exists in Suppliers table, login works via fallback.
			log.Printf("[SUPPLIER REGISTER] T=0 mirror to SupplierUsers failed (non-fatal): %v", err)
		}

		token, err := auth.MintIdentityToken(&auth.LabClaims{
			UserID:       mirrorID,
			SupplierID:   supplierId,
			Role:         "SUPPLIER",
			SupplierRole: "GLOBAL_ADMIN",
			CountryCode:  countryCode,
			IsConfigured: isConfigured,
		})
		if err != nil {
			log.Printf("[SUPPLIER REGISTER] token error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Create Firebase Auth user + mint custom token (graceful degradation)
		var firebaseToken string
		fbUid, fbErr := auth.CreateFirebaseUser(ctx, req.Email, req.Password, req.CompanyName, req.Phone, "SUPPLIER", map[string]interface{}{
			"supplier_id":   supplierId,
			"supplier_role": "GLOBAL_ADMIN",
		})
		if fbErr == nil && fbUid != "" {
			// Store Firebase UID on both tables
			_, _ = spannerClient.Apply(ctx, []*spanner.Mutation{
				spanner.Update("Suppliers", []string{"SupplierId", "FirebaseUid"}, []interface{}{supplierId, fbUid}),
				spanner.Update("SupplierUsers", []string{"UserId", "FirebaseUid"}, []interface{}{mirrorID, fbUid}),
			})
			firebaseToken, _ = auth.MintCustomToken(ctx, fbUid, map[string]interface{}{
				"role":          "SUPPLIER",
				"supplier_id":   supplierId,
				"supplier_role": "GLOBAL_ADMIN",
			})
		}

		resp := map[string]interface{}{
			"token":         token,
			"user_id":       mirrorID,
			"supplier_id":   supplierId,
			"role":          "SUPPLIER",
			"supplier_role": "GLOBAL_ADMIN",
			"name":          req.CompanyName,
			"is_configured": isConfigured,
			"country_code":  countryCode,
		}
		if firebaseToken != "" {
			resp["firebase_token"] = firebaseToken
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

// HandleSupplierConfigure completes onboarding: TaxId, categories.
// POST /v1/supplier/configure → { tax_id, operating_categories }
func HandleSupplierConfigure(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			TaxId               string   `json:"tax_id"`
			OperatingCategories []string `json:"operating_categories"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.TaxId == "" || len(req.OperatingCategories) == 0 {
			http.Error(w, `{"error":"tax_id and operating_categories are required"}`, http.StatusBadRequest)
			return
		}
		if err := ensureCanonicalCategoriesSeeded(r.Context(), spannerClient); err != nil {
			log.Printf("[SUPPLIER CONFIGURE] category seed error: %v", err)
			http.Error(w, `{"error":"category catalog unavailable"}`, http.StatusInternalServerError)
			return
		}
		validCategories, invalidCategories := normalizeValidCategoryIDs(req.OperatingCategories)
		if len(invalidCategories) > 0 {
			http.Error(w, fmt.Sprintf(`{"error":"invalid operating categories: %s"}`, strings.Join(invalidCategories, ", ")), http.StatusBadRequest)
			return
		}
		if len(validCategories) == 0 {
			http.Error(w, `{"error":"at least one valid operating category is required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		supplierID := claims.ResolveSupplierID()

		m := spanner.Update("Suppliers",
			[]string{"SupplierId", "TaxId", "Category", "IsConfigured", "OperatingCategories"},
			[]interface{}{supplierID, req.TaxId, primaryCategoryName(validCategories), true, validCategories},
		)
		if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{m}); err != nil {
			log.Printf("[SUPPLIER CONFIGURE] spanner update error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "CONFIGURED",
			"supplier_id":   supplierID,
			"is_configured": true,
		})
	}
}

// HandleBillingSetup sets the supplier's bank details and payment gateway.
// POST /v1/supplier/billing/setup → { bank_name, account_number, card_number, payment_gateway }
// This is the post-registration billing step — decoupled from signup.
func HandleBillingSetup(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			BankName       string `json:"bank_name"`
			AccountNumber  string `json:"account_number"`
			CardNumber     string `json:"card_number"`
			PaymentGateway string `json:"payment_gateway"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if req.PaymentGateway == "" {
			http.Error(w, `{"error":"payment_gateway is required"}`, http.StatusBadRequest)
			return
		}

		validGateways := map[string]bool{"PAYME": true, "CLICK": true, "CARD": true, "BANK": true}
		if !validGateways[req.PaymentGateway] {
			http.Error(w, `{"error":"invalid payment_gateway — must be PAYME, CLICK, CARD, or BANK"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		supplierID := claims.ResolveSupplierID()

		cols := []string{"SupplierId", "PaymentGateway"}
		vals := []interface{}{supplierID, req.PaymentGateway}

		if req.BankName != "" {
			cols = append(cols, "BankName")
			vals = append(vals, req.BankName)
		}
		if req.AccountNumber != "" {
			cols = append(cols, "AccountNumber")
			vals = append(vals, req.AccountNumber)
		}
		if req.CardNumber != "" {
			cols = append(cols, "CardNumber")
			vals = append(vals, req.CardNumber)
		}

		m := spanner.Update("Suppliers", cols, vals)
		if _, err := spannerClient.Apply(ctx, []*spanner.Mutation{m}); err != nil {
			log.Printf("[SUPPLIER BILLING] spanner update error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "BILLING_CONFIGURED",
			"supplier_id":     supplierID,
			"payment_gateway": req.PaymentGateway,
		})
	}
}

// HandleListPlatformCategories returns all platform categories for onboarding.
// GET /v1/catalog/platform-categories
func HandleListPlatformCategories(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := ensureCanonicalCategoriesSeeded(r.Context(), spannerClient); err != nil {
			log.Printf("[PLATFORM CATEGORIES] seed error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT CategoryId, DisplayName, COALESCE(IconUrl, '') AS IconUrl, DisplayOrder
			      FROM PlatformCategories ORDER BY DisplayOrder ASC`,
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		type Category struct {
			CategoryId   string `json:"category_id"`
			DisplayName  string `json:"display_name"`
			IconUrl      string `json:"icon_url"`
			DisplayOrder int64  `json:"display_order"`
		}

		var categories []Category
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[PLATFORM CATEGORIES] query error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var c Category
			if err := row.Columns(&c.CategoryId, &c.DisplayName, &c.IconUrl, &c.DisplayOrder); err != nil {
				log.Printf("[PLATFORM CATEGORIES] parse error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			categories = append(categories, c)
		}
		if categories == nil {
			categories = []Category{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": categories,
		})
	}
}

// HandleGetSupplierProfile returns the authenticated supplier's profile including IsConfigured.
// GET /v1/supplier/profile
func HandleGetSupplierProfile(spannerClient *spanner.Client, rc *cache.Cache, flight *singleflight.Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		supplierID := claims.ResolveSupplierID()
		cacheKey := cache.SupplierProfile(supplierID)

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

		// Singleflight: coalesce concurrent cache-miss reads for the same supplier
		type profileResult struct {
			Body []byte
			Err  error
		}
		val, err, _ := flight.Do(cacheKey, func() (interface{}, error) {
			return fetchSupplierProfile(r.Context(), spannerClient, supplierID)
		})
		if err != nil {
			log.Printf("[SUPPLIER PROFILE] fetch error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		body := val.([]byte)

		// Backfill cache asynchronously
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
}

// fetchSupplierProfile reads a supplier's full profile from Spanner and returns
// the serialised JSON body. Extracted so singleflight.Do can call it.
func fetchSupplierProfile(ctx context.Context, spannerClient *spanner.Client, supplierID string) ([]byte, error) {
	stmt := spanner.Statement{
		SQL: `SELECT Name, COALESCE(Phone, ''), COALESCE(Category, ''),
		             COALESCE(TaxId, ''), IFNULL(IsConfigured, false),
		             COALESCE(OperatingCategories, []),
		             COALESCE(WarehouseLocation, ''),
		             IFNULL(WarehouseLat, 0.0),
		             IFNULL(WarehouseLng, 0.0),
		             COALESCE(Email, ''),
		             COALESCE(ContactPerson, ''),
		             COALESCE(CompanyRegNumber, ''),
		             COALESCE(BillingAddress, ''),
		             COALESCE(BankName, ''),
		             COALESCE(AccountNumber, ''),
		             COALESCE(CardNumber, ''),
		             COALESCE(PaymentGateway, ''),
		             IFNULL(ManualOffShift, false),
		             COALESCE(TO_JSON_STRING(OperatingSchedule), '{}')
		      FROM Suppliers WHERE SupplierId = @id`,
		Params: map[string]interface{}{"id": supplierID},
	}
	iter := spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("supplier %s not found", supplierID)
	}
	if err != nil {
		return nil, fmt.Errorf("query supplier %s: %w", supplierID, err)
	}

	var name, phone, category, taxId, warehouseLocation string
	var email, contactPerson, companyRegNumber, billingAddress string
	var bankName, accountNumber, cardNumber, paymentGateway string
	var isConfigured, manualOffShift bool
	var operatingCategories []string
	var warehouseLat, warehouseLng float64
	var operatingScheduleJSON string
	if err := row.Columns(&name, &phone, &category, &taxId, &isConfigured, &operatingCategories, &warehouseLocation, &warehouseLat, &warehouseLng, &email, &contactPerson, &companyRegNumber, &billingAddress, &bankName, &accountNumber, &cardNumber, &paymentGateway, &manualOffShift, &operatingScheduleJSON); err != nil {
		return nil, fmt.Errorf("parse supplier %s: %w", supplierID, err)
	}

	maskedCard := cardNumber
	if len(cardNumber) > 4 {
		maskedCard = "****" + cardNumber[len(cardNumber)-4:]
	}

	isActive := resolveIsActive(operatingScheduleJSON, manualOffShift, time.Now())

	return json.Marshal(map[string]interface{}{
		"supplier_id":          supplierID,
		"name":                 name,
		"phone":                phone,
		"email":                email,
		"category":             category,
		"tax_id":               taxId,
		"contact_person":       contactPerson,
		"company_reg_number":   companyRegNumber,
		"billing_address":      billingAddress,
		"is_configured":        isConfigured,
		"operating_categories": operatingCategories,
		"warehouse_location":   warehouseLocation,
		"warehouse_lat":        warehouseLat,
		"warehouse_lng":        warehouseLng,
		"bank_name":            bankName,
		"account_number":       accountNumber,
		"card_number":          maskedCard,
		"payment_gateway":      paymentGateway,
		"manual_off_shift":     manualOffShift,
		"operating_schedule":   json.RawMessage(operatingScheduleJSON),
		"is_active":            isActive,
	})
}

// HandleUpdateSupplierProfile applies partial updates to the supplier's own profile.
// PUT /v1/supplier/profile
func HandleUpdateSupplierProfile(spannerClient *spanner.Client, rc *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var update struct {
			Name              *string  `json:"name"`
			Phone             *string  `json:"phone"`
			Email             *string  `json:"email"`
			ContactPerson     *string  `json:"contact_person"`
			TaxID             *string  `json:"tax_id"`
			CompanyRegNumber  *string  `json:"company_reg_number"`
			BillingAddress    *string  `json:"billing_address"`
			WarehouseLocation *string  `json:"warehouse_location"`
			WarehouseLat      *float64 `json:"warehouse_lat"`
			WarehouseLng      *float64 `json:"warehouse_lng"`
			BankName          *string  `json:"bank_name"`
			AccountNumber     *string  `json:"account_number"`
			CardNumber        *string  `json:"card_number"`
			PaymentGateway    *string  `json:"payment_gateway"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		supplierID := claims.ResolveSupplierID()

		cols := []string{"SupplierId"}
		vals := []interface{}{supplierID}

		if update.Name != nil {
			cols = append(cols, "Name")
			vals = append(vals, *update.Name)
		}
		if update.Phone != nil {
			cols = append(cols, "Phone")
			vals = append(vals, *update.Phone)
		}
		if update.Email != nil {
			cols = append(cols, "Email")
			vals = append(vals, *update.Email)
		}
		if update.ContactPerson != nil {
			cols = append(cols, "ContactPerson")
			vals = append(vals, *update.ContactPerson)
		}
		if update.TaxID != nil {
			cols = append(cols, "TaxId")
			vals = append(vals, *update.TaxID)
		}
		if update.CompanyRegNumber != nil {
			cols = append(cols, "CompanyRegNumber")
			vals = append(vals, *update.CompanyRegNumber)
		}
		if update.BillingAddress != nil {
			cols = append(cols, "BillingAddress")
			vals = append(vals, *update.BillingAddress)
		}
		if update.WarehouseLocation != nil {
			cols = append(cols, "WarehouseLocation")
			vals = append(vals, *update.WarehouseLocation)
		}
		if update.WarehouseLat != nil {
			cols = append(cols, "WarehouseLat")
			vals = append(vals, *update.WarehouseLat)
		}
		if update.WarehouseLng != nil {
			cols = append(cols, "WarehouseLng")
			vals = append(vals, *update.WarehouseLng)
		}
		if update.BankName != nil {
			cols = append(cols, "BankName")
			vals = append(vals, *update.BankName)
		}
		if update.AccountNumber != nil {
			cols = append(cols, "AccountNumber")
			vals = append(vals, *update.AccountNumber)
		}
		if update.CardNumber != nil {
			cols = append(cols, "CardNumber")
			vals = append(vals, *update.CardNumber)
		}
		if update.PaymentGateway != nil {
			cols = append(cols, "PaymentGateway")
			vals = append(vals, *update.PaymentGateway)
		}

		if len(cols) <= 1 {
			http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
			return
		}

		_, err := spannerClient.Apply(r.Context(), []*spanner.Mutation{
			spanner.Update("Suppliers", cols, vals),
		})
		if err != nil {
			log.Printf("[SUPPLIER PROFILE] update error for %s: %v", claims.UserID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Invalidate cached profile after successful write
		rc.Invalidate(r.Context(), cache.SupplierProfile(supplierID))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "PROFILE_UPDATED"})
	}
}
