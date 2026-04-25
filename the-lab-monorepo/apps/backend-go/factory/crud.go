package factory

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
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/iterator"
)

// validateWarehousesBelongToSupplier checks that every warehouse ID in the list
// belongs to the given supplier. Returns the first offending ID on failure.
func validateWarehousesBelongToSupplier(ctx context.Context, client *spanner.Client, whIDs []string, supplierID string) error {
	for _, id := range whIDs {
		if id == "" {
			continue
		}
		row, err := client.Single().ReadRow(ctx, "Warehouses", spanner.Key{id}, []string{"SupplierId"})
		if err != nil {
			return fmt.Errorf("warehouse %s not found", id)
		}
		var owner string
		if err := row.Column(0, &owner); err != nil || owner != supplierID {
			return fmt.Errorf("warehouse %s does not belong to your organization", id)
		}
	}
	return nil
}

// ── Factory CRUD ──────────────────────────────────────────────────────────────

type FactoryResponse struct {
	FactoryId            string  `json:"factory_id"`
	SupplierId           string  `json:"supplier_id"`
	Name                 string  `json:"name"`
	Address              string  `json:"address"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	RegionCode           string  `json:"region_code"`
	LeadTimeDays         int64   `json:"lead_time_days"`
	ProductionCapacityVU float64 `json:"production_capacity_vu"`
	IsActive             bool    `json:"is_active"`
	CreatedAt            string  `json:"created_at"`
}

// HandleFactoryProfile returns the authenticated factory staff's factory.
// GET /v1/factory/profile — FACTORY role only.
func HandleFactoryProfile(spannerClient *spanner.Client, rc *cache.Cache, flight *singleflight.Group) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		factoryID, ok := auth.MustFactoryID(w, r.Context())
		if !ok {
			return
		}

		cacheKey := cache.FactoryProfile(factoryID)

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
			return fetchFactoryProfile(r.Context(), spannerClient, factoryID)
		})
		if err != nil {
			log.Printf("[FACTORY] profile fetch error: %v", err)
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
}

// fetchFactoryProfile reads a factory's profile from Spanner.
func fetchFactoryProfile(ctx context.Context, spannerClient *spanner.Client, factoryID string) ([]byte, error) {
	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, SupplierId, Name, COALESCE(Address, ''),
		             IFNULL(Lat, 0), IFNULL(Lng, 0), COALESCE(RegionCode, ''),
		             LeadTimeDays, ProductionCapacityVU, IsActive, CreatedAt
		      FROM Factories WHERE FactoryId = @fid`,
		Params: map[string]interface{}{"fid": factoryID},
	}
	iter := spannerx.StaleQuery(ctx, spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("factory %s not found: %w", factoryID, err)
	}

	var f FactoryResponse
	var createdAt time.Time
	if err := row.Columns(&f.FactoryId, &f.SupplierId, &f.Name, &f.Address,
		&f.Lat, &f.Lng, &f.RegionCode, &f.LeadTimeDays, &f.ProductionCapacityVU,
		&f.IsActive, &createdAt); err != nil {
		return nil, fmt.Errorf("parse factory %s: %w", factoryID, err)
	}
	f.CreatedAt = createdAt.Format(time.RFC3339)

	return json.Marshal(f)
}

// HandleSupplierFactories handles GET (list) and POST (create) for /v1/supplier/factories.
// SUPPLIER role — manages factories under their organization.
func HandleSupplierFactories(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listSupplierFactories(w, r, spannerClient)
		case http.MethodPost:
			createFactory(w, r, spannerClient)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleSupplierFactoryDetail handles GET, PUT, DELETE for /v1/supplier/factories/{id}
// Also routes /v1/supplier/factories/{id}/warehouses to the assignment handler.
func HandleSupplierFactoryDetail(spannerClient *spanner.Client) http.HandlerFunc {
	assignHandler := HandleFactoryWarehouseAssignment(spannerClient)

	return func(w http.ResponseWriter, r *http.Request) {
		remainder := strings.TrimPrefix(r.URL.Path, "/v1/supplier/factories/")
		if remainder == "" {
			http.Error(w, "factory_id required in path", http.StatusBadRequest)
			return
		}

		// Route /v1/supplier/factories/{id}/warehouses to assignment handler
		if strings.Contains(remainder, "/") {
			parts := strings.SplitN(remainder, "/", 2)
			if parts[1] == "warehouses" {
				assignHandler.ServeHTTP(w, r)
				return
			}
			http.Error(w, "factory_id required in path", http.StatusBadRequest)
			return
		}

		factoryID := remainder

		switch r.Method {
		case http.MethodGet:
			getFactory(w, r, spannerClient, factoryID)
		case http.MethodPut:
			updateFactory(w, r, spannerClient, factoryID)
		case http.MethodDelete:
			deactivateFactory(w, r, spannerClient, factoryID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── Private handlers ──────────────────────────────────────────────────────────

func listSupplierFactories(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, SupplierId, Name, COALESCE(Address, ''),
		             IFNULL(Lat, 0), IFNULL(Lng, 0), COALESCE(RegionCode, ''),
		             LeadTimeDays, ProductionCapacityVU, IsActive, CreatedAt
		      FROM Factories WHERE SupplierId = @sid ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"sid": claims.ResolveSupplierID()},
	}
	iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
	defer iter.Stop()

	factories := []FactoryResponse{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[FACTORY] list query error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var f FactoryResponse
		var createdAt time.Time
		if err := row.Columns(&f.FactoryId, &f.SupplierId, &f.Name, &f.Address,
			&f.Lat, &f.Lng, &f.RegionCode, &f.LeadTimeDays, &f.ProductionCapacityVU,
			&f.IsActive, &createdAt); err != nil {
			log.Printf("[FACTORY] list parse error: %v", err)
			continue
		}
		f.CreatedAt = createdAt.Format(time.RFC3339)
		factories = append(factories, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": factories})
}

// SOVEREIGN ACTION: createFactory requires GLOBAL_ADMIN supplier role.
func createFactory(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}

	var req struct {
		Name                 string   `json:"name"`
		Address              string   `json:"address"`
		Lat                  float64  `json:"lat"`
		Lng                  float64  `json:"lng"`
		RegionCode           string   `json:"region_code"`
		LeadTimeDays         int64    `json:"lead_time_days"`
		ProductionCapacityVU float64  `json:"production_capacity_vu"`
		WarehouseIDs         []string `json:"warehouse_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if req.LeadTimeDays <= 0 {
		req.LeadTimeDays = 2
	}

	factoryId := uuid.New().String()
	h3Index := ""
	if req.Lat != 0 || req.Lng != 0 {
		h3Index = proximity.LookupCell(req.Lat, req.Lng)
	}

	// Build mutations: factory INSERT + optional warehouse assignments
	mutations := []*spanner.Mutation{
		spanner.Insert("Factories",
			[]string{"FactoryId", "SupplierId", "Name", "Address", "Lat", "Lng",
				"RegionCode", "LeadTimeDays", "ProductionCapacityVU", "IsActive", "CreatedAt", "H3Index"},
			[]interface{}{factoryId, claims.ResolveSupplierID(), req.Name, req.Address, req.Lat, req.Lng,
				req.RegionCode, req.LeadTimeDays, req.ProductionCapacityVU, true, spanner.CommitTimestamp, h3Index},
		),
	}

	// Assign PrimaryFactoryId on selected warehouses (if provided)
	assignedCount := 0
	if len(req.WarehouseIDs) > 0 {
		if err := validateWarehousesBelongToSupplier(r.Context(), spannerClient, req.WarehouseIDs, claims.ResolveSupplierID()); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusForbidden)
			return
		}
	}
	for _, whID := range req.WarehouseIDs {
		if whID == "" {
			continue
		}
		mutations = append(mutations, spanner.Update("Warehouses",
			[]string{"WarehouseId", "PrimaryFactoryId", "UpdatedAt"},
			[]interface{}{whID, factoryId, spanner.CommitTimestamp},
		))
		assignedCount++
	}

	if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite(mutations)
	}); err != nil {
		log.Printf("[FACTORY] create error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"factory_id":        factoryId,
		"supplier_id":       claims.ResolveSupplierID(),
		"name":              req.Name,
		"warehouses_linked": assignedCount,
	})
}

func getFactory(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, factoryID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stmt := spanner.Statement{
		SQL: `SELECT FactoryId, SupplierId, Name, COALESCE(Address, ''),
		             IFNULL(Lat, 0), IFNULL(Lng, 0), COALESCE(RegionCode, ''),
		             LeadTimeDays, ProductionCapacityVU, IsActive, CreatedAt
		      FROM Factories WHERE FactoryId = @fid AND SupplierId = @sid`,
		Params: map[string]interface{}{"fid": factoryID, "sid": claims.ResolveSupplierID()},
	}
	iter := spannerx.StaleQuery(r.Context(), spannerClient, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}

	var f FactoryResponse
	var createdAt time.Time
	if err := row.Columns(&f.FactoryId, &f.SupplierId, &f.Name, &f.Address,
		&f.Lat, &f.Lng, &f.RegionCode, &f.LeadTimeDays, &f.ProductionCapacityVU,
		&f.IsActive, &createdAt); err != nil {
		log.Printf("[FACTORY] detail parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	f.CreatedAt = createdAt.Format(time.RFC3339)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f)
}

func updateFactory(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, factoryID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name                 *string  `json:"name"`
		Address              *string  `json:"address"`
		Lat                  *float64 `json:"lat"`
		Lng                  *float64 `json:"lng"`
		RegionCode           *string  `json:"region_code"`
		LeadTimeDays         *int64   `json:"lead_time_days"`
		ProductionCapacityVU *float64 `json:"production_capacity_vu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	// Verify ownership
	fRow, err := spannerClient.Single().ReadRow(r.Context(), "Factories",
		spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}
	var ownerSid string
	if err := fRow.Columns(&ownerSid); err != nil || ownerSid != claims.ResolveSupplierID() {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}

	cols := []string{"FactoryId", "UpdatedAt"}
	vals := []interface{}{factoryID, spanner.CommitTimestamp}

	if req.Name != nil {
		cols = append(cols, "Name")
		vals = append(vals, *req.Name)
	}
	if req.Address != nil {
		cols = append(cols, "Address")
		vals = append(vals, *req.Address)
	}
	if req.Lat != nil {
		cols = append(cols, "Lat")
		vals = append(vals, *req.Lat)
	}
	if req.Lng != nil {
		cols = append(cols, "Lng")
		vals = append(vals, *req.Lng)
	}
	// Recompute H3Index when coordinates change
	if req.Lat != nil || req.Lng != nil {
		newLat := 0.0
		newLng := 0.0
		if req.Lat != nil {
			newLat = *req.Lat
		}
		if req.Lng != nil {
			newLng = *req.Lng
		}
		if newLat != 0 || newLng != 0 {
			cols = append(cols, "H3Index")
			vals = append(vals, proximity.LookupCell(newLat, newLng))
		}
	}
	if req.RegionCode != nil {
		cols = append(cols, "RegionCode")
		vals = append(vals, *req.RegionCode)
	}
	if req.LeadTimeDays != nil {
		cols = append(cols, "LeadTimeDays")
		vals = append(vals, *req.LeadTimeDays)
	}
	if req.ProductionCapacityVU != nil {
		cols = append(cols, "ProductionCapacityVU")
		vals = append(vals, *req.ProductionCapacityVU)
	}

	m := spanner.Update("Factories", cols, vals)
	if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[FACTORY] update error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "UPDATED",
		"factory_id": factoryID,
	})
}

func deactivateFactory(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, factoryID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// SOVEREIGN ACTION: Factory deletion requires GLOBAL_ADMIN
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}

	// Verify ownership
	fRow, err := spannerClient.Single().ReadRow(r.Context(), "Factories",
		spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}
	var ownerSid string
	if err := fRow.Columns(&ownerSid); err != nil || ownerSid != claims.ResolveSupplierID() {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}

	m := spanner.Update("Factories",
		[]string{"FactoryId", "IsActive", "UpdatedAt"},
		[]interface{}{factoryID, false, spanner.CommitTimestamp},
	)
	if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
		log.Printf("[FACTORY] deactivate error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "DEACTIVATED",
		"factory_id": factoryID,
	})
}

// HandleFactoryWarehouseAssignment handles PUT /v1/supplier/factories/{id}/warehouses
// Sets PrimaryFactoryId on the listed warehouses, linking them to this factory.
func HandleFactoryWarehouseAssignment(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract factory ID from path: /v1/supplier/factories/{id}/warehouses
		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/factories/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] != "warehouses" {
			http.Error(w, `{"error":"invalid path — expected /v1/supplier/factories/{id}/warehouses"}`, http.StatusBadRequest)
			return
		}
		factoryID := parts[0]

		// Verify factory ownership
		fRow, err := spannerClient.Single().ReadRow(r.Context(), "Factories",
			spanner.Key{factoryID}, []string{"SupplierId", "IsActive"})
		if err != nil {
			http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
			return
		}
		var ownerSid string
		var isActive bool
		if err := fRow.Columns(&ownerSid, &isActive); err != nil || ownerSid != claims.ResolveSupplierID() {
			http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
			return
		}
		if !isActive {
			http.Error(w, `{"error":"factory is deactivated"}`, http.StatusConflict)
			return
		}

		var req struct {
			WarehouseIDs []string `json:"warehouse_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if len(req.WarehouseIDs) == 0 {
			http.Error(w, `{"error":"warehouse_ids array required"}`, http.StatusBadRequest)
			return
		}

		// Validate all warehouse IDs belong to this supplier
		if err := validateWarehousesBelongToSupplier(r.Context(), spannerClient, req.WarehouseIDs, claims.ResolveSupplierID()); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusForbidden)
			return
		}

		// Build update mutations — set PrimaryFactoryId on each warehouse
		var mutations []*spanner.Mutation
		for _, whID := range req.WarehouseIDs {
			if whID == "" {
				continue
			}
			mutations = append(mutations, spanner.Update("Warehouses",
				[]string{"WarehouseId", "PrimaryFactoryId", "UpdatedAt"},
				[]interface{}{whID, factoryID, spanner.CommitTimestamp},
			))
		}

		if len(mutations) == 0 {
			http.Error(w, `{"error":"no valid warehouse_ids provided"}`, http.StatusBadRequest)
			return
		}

		if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite(mutations)
		}); err != nil {
			log.Printf("[FACTORY] assign warehouses error: %v", err)
			http.Error(w, `{"error":"failed to assign warehouses — verify all IDs belong to your organization"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ASSIGNED",
			"factory_id": factoryID,
			"updated":    len(mutations),
		})
	}
}
