package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/auth"
	"backend-go/models"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// EmpathyService holds the Spanner client for Empathy Engine operations.
type EmpathyService struct {
	Cache *cache.Cache
	Client *spanner.Client
}

// hasHistory returns true if the retailer has at least one COMPLETED order.
func (s *EmpathyService) hasHistory(ctx context.Context, retailerID string) bool {
	stmt := spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Orders WHERE RetailerId = @rid AND State = 'COMPLETED' LIMIT 1`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false
	}
	var cnt int64
	if row.Columns(&cnt) == nil {
		return cnt > 0
	}
	return false
}

// hasHistoryForSupplier returns true if the retailer has ≥1 COMPLETED order from the supplier.
func (s *EmpathyService) hasHistoryForSupplier(ctx context.Context, retailerID, supplierID string) bool {
	stmt := spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM Orders WHERE RetailerId = @rid AND SupplierId = @sid AND State = 'COMPLETED' LIMIT 1`,
		Params: map[string]interface{}{"rid": retailerID, "sid": supplierID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false
	}
	var cnt int64
	if row.Columns(&cnt) == nil {
		return cnt > 0
	}
	return false
}

// hasHistoryForCategory returns true if the retailer has ≥1 COMPLETED order containing a product of this category.
func (s *EmpathyService) hasHistoryForCategory(ctx context.Context, retailerID, categoryID string) bool {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders o
		      JOIN OrderLineItems li ON li.OrderId = o.OrderId
		      JOIN Products p ON p.ProductId = li.ProductId
		      WHERE o.RetailerId = @rid AND p.CategoryId = @cid AND o.State = 'COMPLETED'
		      LIMIT 1`,
		Params: map[string]interface{}{"rid": retailerID, "cid": categoryID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false
	}
	var cnt int64
	if row.Columns(&cnt) == nil {
		return cnt > 0
	}
	return false
}

// hasHistoryForProduct returns true if the retailer has ≥1 COMPLETED order containing this product.
func (s *EmpathyService) hasHistoryForProduct(ctx context.Context, retailerID, productID string) bool {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders o
		      JOIN OrderLineItems li ON li.OrderId = o.OrderId
		      WHERE o.RetailerId = @rid AND li.ProductId = @pid AND o.State = 'COMPLETED'
		      LIMIT 1`,
		Params: map[string]interface{}{"rid": retailerID, "pid": productID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false
	}
	var cnt int64
	if row.Columns(&cnt) == nil {
		return cnt > 0
	}
	return false
}

// hasHistoryForVariant returns true if the retailer has ≥1 COMPLETED order containing this SKU.
func (s *EmpathyService) hasHistoryForVariant(ctx context.Context, retailerID, skuID string) bool {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders o
		      JOIN OrderLineItems li ON li.OrderId = o.OrderId
		      WHERE o.RetailerId = @rid AND li.VariantId = @vid AND o.State = 'COMPLETED'
		      LIMIT 1`,
		Params: map[string]interface{}{"rid": retailerID, "vid": skuID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false
	}
	var cnt int64
	if row.Columns(&cnt) == nil {
		return cnt > 0
	}
	return false
}

// applyUseHistory builds analytics-start-date columns/values for use_history toggling.
func applyUseHistory(useHistory *bool, cols *[]string, vals *[]interface{}) {
	if useHistory == nil {
		return
	}
	*cols = append(*cols, "AnalyticsStartDate")
	if *useHistory {
		*vals = append(*vals, nil) // clear cut-off → use all history
	} else {
		*vals = append(*vals, time.Now().UTC()) // start fresh from now
	}
}

// HandlePatchGlobal handles PATCH /v1/retailer/settings/auto-order/global
// Body: {"enabled": true, "use_history": true|false}
// When enabling with use_history=false → AnalyticsStartDate = NOW (start fresh).
// When enabling with use_history=true → AnalyticsStartDate = NULL (use all history).
func (s *EmpathyService) HandlePatchGlobal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	retailerID := claims.UserID

	var req models.UpdateGlobalSettingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"}
	vals := []interface{}{retailerID, req.Enabled, spanner.CommitTimestamp}

	// Handle use_history when enabling
	if req.Enabled && req.UseHistory != nil {
		cols = append(cols, "AnalyticsStartDate")
		if *req.UseHistory {
			// Use all previous analytics — clear the cut-off
			vals = append(vals, nil)
		} else {
			// Start fresh — set cut-off to now
			vals = append(vals, time.Now().UTC())
		}
	}

	m := spanner.InsertOrUpdate("RetailerGlobalSettings", cols, vals)

	if _, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[EMPATHY ENGINE] Global settings update failed for %s: %v", retailerID, err)
		http.Error(w, `{"error":"database write failed"}`, http.StatusInternalServerError)
		return
	}

	// When enabling, promote DORMANT predictions to WAITING
	if req.Enabled {
		go s.promoteDormantPredictions(retailerID)
	}

	if s.Cache != nil {
		s.Cache.InvalidatePrefix(r.Context(), cache.PrefixSettings+retailerID+":")
	}
	log.Printf("[EMPATHY ENGINE] %s -> GlobalAutoOrder = %v", retailerID, req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"OK","retailer_id":"%s","global_auto_order_enabled":%v}`, retailerID, req.Enabled)
}

// HandlePatchSupplier handles PATCH /v1/retailer/settings/auto-order/supplier/{supplier_id}
func (s *EmpathyService) HandlePatchSupplier(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	retailerID := claims.UserID

	supplierID := strings.TrimPrefix(r.URL.Path, "/v1/retailer/settings/auto-order/supplier/")
	if supplierID == "" || strings.Contains(supplierID, "/") {
		http.Error(w, `{"error":"supplier_id required in path"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateSupplierSettingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"RetailerId", "SupplierId", "AutoOrderEnabled", "UpdatedAt"}
	vals := []interface{}{retailerID, supplierID, req.Enabled, spanner.CommitTimestamp}
	applyUseHistory(req.UseHistory, &cols, &vals)

	m := spanner.InsertOrUpdate("RetailerSupplierSettings", cols, vals)

	if _, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[EMPATHY ENGINE] Supplier settings update failed for %s/%s: %v", retailerID, supplierID, err)
		http.Error(w, `{"error":"database write failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[EMPATHY ENGINE] %s -> Supplier %s AutoOrder = %v", retailerID, supplierID, req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"OK","retailer_id":"%s","supplier_id":"%s","auto_order_enabled":%v}`, retailerID, supplierID, req.Enabled)
}

// HandlePatchCategory handles PATCH /v1/retailer/settings/auto-order/category/{category_id}
func (s *EmpathyService) HandlePatchCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	retailerID := claims.UserID

	categoryID := strings.TrimPrefix(r.URL.Path, "/v1/retailer/settings/auto-order/category/")
	if categoryID == "" || strings.Contains(categoryID, "/") {
		http.Error(w, `{"error":"category_id required in path"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateCategorySettingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"RetailerId", "CategoryId", "AutoOrderEnabled", "UpdatedAt"}
	vals := []interface{}{retailerID, categoryID, req.Enabled, spanner.CommitTimestamp}
	applyUseHistory(req.UseHistory, &cols, &vals)

	m := spanner.InsertOrUpdate("RetailerCategorySettings", cols, vals)

	if _, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[EMPATHY ENGINE] Category settings update failed for %s/%s: %v", retailerID, categoryID, err)
		http.Error(w, `{"error":"database write failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[EMPATHY ENGINE] %s -> Category %s AutoOrder = %v", retailerID, categoryID, req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"OK","retailer_id":"%s","category_id":"%s","auto_order_enabled":%v}`, retailerID, categoryID, req.Enabled)
}

// HandlePatchProduct handles PATCH /v1/retailer/settings/auto-order/product/{product_id}
func (s *EmpathyService) HandlePatchProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	retailerID := claims.UserID

	productID := strings.TrimPrefix(r.URL.Path, "/v1/retailer/settings/auto-order/product/")
	if productID == "" || strings.Contains(productID, "/") {
		http.Error(w, `{"error":"product_id required in path"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateProductSettingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"RetailerId", "ProductId", "AutoOrderEnabled", "UpdatedAt"}
	vals := []interface{}{retailerID, productID, req.Enabled, spanner.CommitTimestamp}
	applyUseHistory(req.UseHistory, &cols, &vals)

	m := spanner.InsertOrUpdate("RetailerProductSettings", cols, vals)

	if _, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[EMPATHY ENGINE] Product settings update failed for %s/%s: %v", retailerID, productID, err)
		http.Error(w, `{"error":"database write failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[EMPATHY ENGINE] %s -> Product %s AutoOrder = %v", retailerID, productID, req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"OK","retailer_id":"%s","product_id":"%s","auto_order_enabled":%v}`, retailerID, productID, req.Enabled)
}

// HandlePatchVariant handles PATCH /v1/retailer/settings/auto-order/variant/{sku_id}
func (s *EmpathyService) HandlePatchVariant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	retailerID := claims.UserID

	skuID := strings.TrimPrefix(r.URL.Path, "/v1/retailer/settings/auto-order/variant/")
	if skuID == "" || strings.Contains(skuID, "/") {
		http.Error(w, `{"error":"sku_id required in path"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateVariantSettingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	cols := []string{"RetailerId", "SkuId", "AutoOrderEnabled", "UpdatedAt"}
	vals := []interface{}{retailerID, skuID, req.Enabled, spanner.CommitTimestamp}
	applyUseHistory(req.UseHistory, &cols, &vals)

	m := spanner.InsertOrUpdate("RetailerVariantSettings", cols, vals)

	if _, err := s.Client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[EMPATHY ENGINE] Variant settings update failed for %s/%s: %v", retailerID, skuID, err)
		http.Error(w, `{"error":"database write failed"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[EMPATHY ENGINE] %s -> Variant %s AutoOrder = %v", retailerID, skuID, req.Enabled)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"OK","retailer_id":"%s","sku_id":"%s","auto_order_enabled":%v}`, retailerID, skuID, req.Enabled)
}

// HandleGetAutoOrderSettings handles GET /v1/retailer/settings/auto-order
// Returns the full hierarchy of auto-order settings for the authenticated retailer.
func (s *EmpathyService) HandleGetAutoOrderSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	retailerID := claims.UserID
	ctx := r.Context()

	resp := models.AutoOrderSettingsResponse{
		SupplierOverrides: []models.SupplierOverrideResponse{},
		CategoryOverrides: []models.CategoryOverrideResponse{},
		ProductOverrides:  []models.ProductOverrideResponse{},
		VariantOverrides:  []models.VariantOverrideResponse{},
	}

	// 0. Check if retailer has any completed order history at all
	resp.HasAnyHistory = s.hasHistory(ctx, retailerID)

	// 1. Global settings
	row, err := spannerx.StaleReadRow(ctx, s.Client, "RetailerGlobalSettings",
		spanner.Key{retailerID},
		[]string{"GlobalAutoOrderEnabled", "AnalyticsStartDate"})
	if err == nil {
		var enabled bool
		var analyticsStart spanner.NullTime
		if scanErr := row.Columns(&enabled, &analyticsStart); scanErr == nil {
			resp.GlobalEnabled = enabled
			if analyticsStart.Valid {
				ts := analyticsStart.Time.Format(time.RFC3339)
				resp.AnalyticsStartDate = &ts
			}
		}
	}

	// 2. Supplier overrides
	suppStmt := spanner.Statement{
		SQL: `SELECT SupplierId, AutoOrderEnabled, AnalyticsStartDate FROM RetailerSupplierSettings
		      WHERE RetailerId = @rid`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, suppStmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var sid string
		var enabled bool
		var analyticsStart spanner.NullTime
		if row.Columns(&sid, &enabled, &analyticsStart) == nil {
			ov := models.SupplierOverrideResponse{
				SupplierID: sid,
				Enabled:    enabled,
				HasHistory: s.hasHistoryForSupplier(ctx, retailerID, sid),
			}
			if analyticsStart.Valid {
				ts := analyticsStart.Time.Format(time.RFC3339)
				ov.AnalyticsStartDate = &ts
			}
			resp.SupplierOverrides = append(resp.SupplierOverrides, ov)
		}
	}

	// 3. Category overrides
	catStmt := spanner.Statement{
		SQL: `SELECT CategoryId, AutoOrderEnabled, AnalyticsStartDate FROM RetailerCategorySettings
		      WHERE RetailerId = @rid`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter2 := spannerx.StaleQuery(ctx, s.Client, catStmt)
	defer iter2.Stop()
	for {
		row, err := iter2.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var cid string
		var enabled bool
		var analyticsStart spanner.NullTime
		if row.Columns(&cid, &enabled, &analyticsStart) == nil {
			ov := models.CategoryOverrideResponse{
				CategoryID: cid,
				Enabled:    enabled,
				HasHistory: s.hasHistoryForCategory(ctx, retailerID, cid),
			}
			if analyticsStart.Valid {
				ts := analyticsStart.Time.Format(time.RFC3339)
				ov.AnalyticsStartDate = &ts
			}
			resp.CategoryOverrides = append(resp.CategoryOverrides, ov)
		}
	}

	// 4. Product overrides
	prodStmt := spanner.Statement{
		SQL: `SELECT ProductId, AutoOrderEnabled, AnalyticsStartDate FROM RetailerProductSettings
		      WHERE RetailerId = @rid`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter3 := spannerx.StaleQuery(ctx, s.Client, prodStmt)
	defer iter3.Stop()
	for {
		row, err := iter3.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var pid string
		var enabled bool
		var analyticsStart spanner.NullTime
		if row.Columns(&pid, &enabled, &analyticsStart) == nil {
			ov := models.ProductOverrideResponse{
				ProductID:  pid,
				Enabled:    enabled,
				HasHistory: s.hasHistoryForProduct(ctx, retailerID, pid),
			}
			if analyticsStart.Valid {
				ts := analyticsStart.Time.Format(time.RFC3339)
				ov.AnalyticsStartDate = &ts
			}
			resp.ProductOverrides = append(resp.ProductOverrides, ov)
		}
	}

	// 5. Variant overrides
	varStmt := spanner.Statement{
		SQL: `SELECT SkuId, AutoOrderEnabled, AnalyticsStartDate FROM RetailerVariantSettings
		      WHERE RetailerId = @rid`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter4 := spannerx.StaleQuery(ctx, s.Client, varStmt)
	defer iter4.Stop()
	for {
		row, err := iter4.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var skuID string
		var enabled bool
		var analyticsStart spanner.NullTime
		if row.Columns(&skuID, &enabled, &analyticsStart) == nil {
			ov := models.VariantOverrideResponse{
				SkuID:      skuID,
				Enabled:    enabled,
				HasHistory: s.hasHistoryForVariant(ctx, retailerID, skuID),
			}
			if analyticsStart.Valid {
				ts := analyticsStart.Time.Format(time.RFC3339)
				ov.AnalyticsStartDate = &ts
			}
			resp.VariantOverrides = append(resp.VariantOverrides, ov)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// promoteDormantPredictions promotes DORMANT predictions to WAITING for a retailer
// when auto-order is enabled. Runs async — best effort.
func (s *EmpathyService) promoteDormantPredictions(retailerID string) {
	ctx := context.Background()
	stmt := spanner.Statement{
		SQL: `SELECT PredictionId FROM AIPredictions
		      WHERE RetailerId = @rid AND Status = 'DORMANT'`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter := spannerx.StaleQuery(ctx, s.Client, stmt)
	defer iter.Stop()

	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[EMPATHY ENGINE] promoteDormant query error: %v", err)
			return
		}
		var id string
		if row.Columns(&id) == nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return
	}

	var mutations []*spanner.Mutation
	for _, id := range ids {
		mutations = append(mutations, spanner.Update("AIPredictions",
			[]string{"PredictionId", "Status"}, []interface{}{id, "WAITING"}))
	}

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		log.Printf("[EMPATHY ENGINE] promoteDormant commit error: %v", err)
		return
	}
	log.Printf("[EMPATHY ENGINE] Promoted %d DORMANT→WAITING for %s", len(ids), retailerID)
}
