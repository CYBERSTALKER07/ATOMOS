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
	"backend-go/kafka"
	"backend-go/outbox"
	"backend-go/proximity"
	"backend-go/telemetry"
	"backend-go/workers"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Warehouse DTOs ────────────────────────────────────────────────────────

type CreateWarehouseRequest struct {
	Name             string  `json:"name"`
	Address          string  `json:"address"`
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	CoverageRadiusKm float64 `json:"coverage_radius_km"`
}

type UpdateWarehouseRequest struct {
	Name              *string  `json:"name,omitempty"`
	Address           *string  `json:"address,omitempty"`
	Lat               *float64 `json:"lat,omitempty"`
	Lng               *float64 `json:"lng,omitempty"`
	CoverageRadiusKm  *float64 `json:"coverage_radius_km,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
	IsOnShift         *bool    `json:"is_on_shift,omitempty"`
	OperatingSchedule *string  `json:"operating_schedule,omitempty"` // JSON: {"mon":{"open":"09:00","close":"18:00"}, ...}
	DisabledReason    *string  `json:"disabled_reason,omitempty"`
	MaxCapacity       *int64   `json:"max_capacity,omitempty"`
}

type WarehouseResponse struct {
	WarehouseId      string   `json:"warehouse_id"`
	SupplierId       string   `json:"supplier_id"`
	Name             string   `json:"name"`
	Address          string   `json:"address"`
	Lat              float64  `json:"lat"`
	Lng              float64  `json:"lng"`
	H3Indexes        []string `json:"h3_indexes"`
	CoverageRadiusKm float64  `json:"coverage_radius_km"`
	IsActive         bool     `json:"is_active"`
	IsDefault        bool     `json:"is_default"`
	IsOnShift        bool     `json:"is_on_shift"`
	CreatedAt        string   `json:"created_at"`
	// Aggregated stats
	DriverCount int64 `json:"driver_count"`
	StaffCount  int64 `json:"staff_count"`
}

// ── Handlers ──────────────────────────────────────────────────────────────

// HandleWarehouses routes GET (list) and POST (create) for /v1/supplier/warehouses
func HandleWarehouses(spannerClient *spanner.Client, producer ...*kafkago.Writer) http.HandlerFunc {
	var kafkaProducer *kafkago.Writer
	if len(producer) > 0 {
		kafkaProducer = producer[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listWarehouses(w, r, spannerClient)
		case http.MethodPost:
			createWarehouse(w, r, spannerClient, kafkaProducer)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// HandleWarehouseByID routes GET, PUT, DELETE for /v1/supplier/warehouses/{id}
func HandleWarehouseByID(spannerClient *spanner.Client, producer ...*kafkago.Writer) http.HandlerFunc {
	var kafkaProducer *kafkago.Writer
	if len(producer) > 0 {
		kafkaProducer = producer[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract warehouse ID from path: /v1/supplier/warehouses/{id}
		path := strings.TrimPrefix(r.URL.Path, "/v1/supplier/warehouses/")
		if path == "" || strings.Contains(path, "/") {
			http.Error(w, "warehouse_id required in path", http.StatusBadRequest)
			return
		}
		warehouseID := path

		switch r.Method {
		case http.MethodGet:
			getWarehouse(w, r, spannerClient, warehouseID)
		case http.MethodPut:
			updateWarehouse(w, r, spannerClient, warehouseID, kafkaProducer)
		case http.MethodDelete:
			deactivateWarehouse(w, r, spannerClient, warehouseID, kafkaProducer)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── List Warehouses ───────────────────────────────────────────────────────

func listWarehouses(w http.ResponseWriter, r *http.Request, client *spanner.Client) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	stmt := spanner.Statement{
		SQL: `SELECT w.WarehouseId, w.Name, w.Address, w.Lat, w.Lng, w.H3Indexes,
		             w.CoverageRadiusKm, w.IsActive, w.IsDefault, w.IsOnShift, w.CreatedAt,
		             (SELECT COUNT(*) FROM Drivers d WHERE d.WarehouseId = w.WarehouseId OR (d.HomeNodeType = 'WAREHOUSE' AND d.HomeNodeId = w.WarehouseId)) AS DriverCount,
		             (SELECT COUNT(*) FROM WarehouseStaff ws WHERE ws.WarehouseId = w.WarehouseId) AS StaffCount
		      FROM Warehouses w
		      WHERE w.SupplierId = @supplierId
		      ORDER BY w.IsDefault DESC, w.CreatedAt ASC`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
		},
	}

	// NODE_ADMIN: only see their assigned warehouse
	stmt = auth.AppendWarehouseFilterStmt(r.Context(), stmt, "w")

	iter := client.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var warehouses []WarehouseResponse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[WAREHOUSE] List query error for supplier %s: %v", supplierID, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var wh WarehouseResponse
		var lat, lng spanner.NullFloat64
		var h3Indexes []string
		var createdAt spanner.NullTime

		if err := row.Columns(&wh.WarehouseId, &wh.Name, &wh.Address, &lat, &lng,
			&h3Indexes, &wh.CoverageRadiusKm, &wh.IsActive, &wh.IsDefault, &wh.IsOnShift,
			&createdAt, &wh.DriverCount, &wh.StaffCount); err != nil {
			log.Printf("[WAREHOUSE] Row parse error: %v", err)
			continue
		}

		wh.SupplierId = supplierID
		if lat.Valid {
			wh.Lat = lat.Float64
		}
		if lng.Valid {
			wh.Lng = lng.Float64
		}
		wh.H3Indexes = h3Indexes
		if createdAt.Valid {
			wh.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
		}
		warehouses = append(warehouses, wh)
	}

	if warehouses == nil {
		warehouses = []WarehouseResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"warehouses": warehouses,
		"total":      len(warehouses),
	})
}

// ── Get Single Warehouse ──────────────────────────────────────────────────

func getWarehouse(w http.ResponseWriter, r *http.Request, client *spanner.Client, warehouseID string) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	// Verify ownership + scope
	if scope := auth.GetWarehouseScope(r.Context()); scope != nil && scope.IsNodeAdmin && scope.WarehouseID != warehouseID {
		http.Error(w, "Access denied: warehouse scope violation", http.StatusForbidden)
		return
	}

	row, err := client.Single().ReadRow(r.Context(), "Warehouses",
		spanner.Key{warehouseID},
		[]string{"WarehouseId", "SupplierId", "Name", "Address", "Lat", "Lng",
			"H3Indexes", "CoverageRadiusKm", "IsActive", "IsDefault", "IsOnShift", "CreatedAt"})
	if err != nil {
		http.Error(w, "Warehouse not found", http.StatusNotFound)
		return
	}

	var wh WarehouseResponse
	var whSupplierId string
	var lat, lng spanner.NullFloat64
	var h3Indexes []string
	var createdAt spanner.NullTime

	if err := row.Columns(&wh.WarehouseId, &whSupplierId, &wh.Name, &wh.Address, &lat, &lng,
		&h3Indexes, &wh.CoverageRadiusKm, &wh.IsActive, &wh.IsDefault, &wh.IsOnShift, &createdAt); err != nil {
		log.Printf("[WAREHOUSE] Parse error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if whSupplierId != supplierID {
		http.Error(w, "Warehouse not found", http.StatusNotFound)
		return
	}

	wh.SupplierId = supplierID
	if lat.Valid {
		wh.Lat = lat.Float64
	}
	if lng.Valid {
		wh.Lng = lng.Float64
	}
	wh.H3Indexes = h3Indexes
	if createdAt.Valid {
		wh.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wh)
}

// ── Create Warehouse ──────────────────────────────────────────────────────

func createWarehouse(w http.ResponseWriter, r *http.Request, client *spanner.Client, producer *kafkago.Writer) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	// Only GLOBAL_ADMIN can create new warehouses
	if scope := auth.GetWarehouseScope(r.Context()); scope != nil && scope.IsNodeAdmin {
		http.Error(w, "Only headquarters admin can create warehouses", http.StatusForbidden)
		return
	}

	var req CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if req.Lat == 0 && req.Lng == 0 {
		http.Error(w, `{"error":"lat and lng are required"}`, http.StatusBadRequest)
		return
	}
	if req.CoverageRadiusKm <= 0 {
		req.CoverageRadiusKm = 50.0
	}

	warehouseID := uuid.New().String()
	h3Cells := proximity.ComputeGridCoverage(req.Lat, req.Lng, req.CoverageRadiusKm)

	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Insert("Warehouses",
			[]string{"WarehouseId", "SupplierId", "Name", "Address", "Lat", "Lng",
				"H3Indexes", "CoverageRadiusKm", "IsActive", "IsDefault", "IsOnShift", "CreatedAt"},
			[]interface{}{warehouseID, supplierID, req.Name, req.Address, req.Lat, req.Lng,
				h3Cells, req.CoverageRadiusKm, true, false, true, spanner.CommitTimestamp},
		)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		event := kafka.WarehouseCreatedEvent{
			WarehouseId:    warehouseID,
			SupplierId:     supplierID,
			Name:           req.Name,
			Lat:            req.Lat,
			Lng:            req.Lng,
			H3Count:        len(h3Cells),
			CoverageRadius: req.CoverageRadiusKm,
			Timestamp:      time.Now().UTC(),
		}
		return outbox.EmitJSON(txn, "Warehouse", warehouseID,
			kafka.EventWarehouseCreated, kafka.TopicMain, event,
			telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[WAREHOUSE] Create failed for supplier %s: %v", supplierID, err)
		http.Error(w, "Failed to create warehouse", http.StatusInternalServerError)
		return
	}

	// Index in Redis cache
	workers.EventPool.Submit(func() {
		_ = cache.IndexWarehouse(r.Context(), cache.WarehouseGeoEntry{
			WarehouseId: warehouseID,
			SupplierId:  supplierID,
			Name:        req.Name,
			Lat:         req.Lat,
			Lng:         req.Lng,
			RadiusKm:    req.CoverageRadiusKm,
			H3Cells:     h3Cells,
		})
	})

	// Background coverage consistency check — detect orphaned retailers
	capturedSupplier := supplierID
	workers.EventPool.Submit(func() {
		if err := proximity.VerifyCoverageConsistency(context.Background(), client, capturedSupplier); err != nil {
			log.Printf("[WAREHOUSE] Coverage audit failed post-create for supplier %s: %v", capturedSupplier, err)
		}
	})

	// Phase 7.2: WS WRH_NEW delta — broadcast to all supplier admin portals
	go broadcastWarehouseCreated(supplierID, warehouseID, req.Name, req.Lat, req.Lng, req.CoverageRadiusKm, len(h3Cells))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(WarehouseResponse{
		WarehouseId:      warehouseID,
		SupplierId:       supplierID,
		Name:             req.Name,
		Address:          req.Address,
		Lat:              req.Lat,
		Lng:              req.Lng,
		H3Indexes:        h3Cells,
		CoverageRadiusKm: req.CoverageRadiusKm,
		IsActive:         true,
		IsDefault:        false,
		IsOnShift:        true,
	})
}

// ── Update Warehouse ──────────────────────────────────────────────────────

func updateWarehouse(w http.ResponseWriter, r *http.Request, client *spanner.Client, warehouseID string, producer *kafkago.Writer) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	var req UpdateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	var geoChanged bool
	var statusChanged bool
	var oldIsActive, newIsActive bool
	var oldIsOnShift, newIsOnShift bool
	var cacheRefreshNeeded bool
	var oldCacheEntry, newCacheEntry cache.WarehouseGeoEntry

	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		geoChanged = false
		statusChanged = false
		cacheRefreshNeeded = false

		// Verify ownership — also read current status fields for change detection
		row, err := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
			[]string{"SupplierId", "Name", "Lat", "Lng", "CoverageRadiusKm", "H3Indexes", "IsActive", "IsOnShift"})
		if err != nil {
			return fmt.Errorf("warehouse not found: %w", err)
		}
		var existingSupplierId string
		var existingName string
		var existingLat, existingLng spanner.NullFloat64
		var existingRadius float64
		var existingH3Cells []string
		var curActive bool
		var curOnShift spanner.NullBool
		if err := row.Columns(&existingSupplierId, &existingName, &existingLat, &existingLng, &existingRadius, &existingH3Cells, &curActive, &curOnShift); err != nil {
			return err
		}
		if existingSupplierId != supplierID {
			return fmt.Errorf("ownership mismatch")
		}

		oldIsActive = curActive
		oldIsOnShift = !curOnShift.Valid || curOnShift.Bool // default true when NULL
		newIsActive = oldIsActive
		newIsOnShift = oldIsOnShift

		cols := []string{"WarehouseId", "UpdatedAt"}
		vals := []interface{}{warehouseID, spanner.CommitTimestamp}

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
		if req.CoverageRadiusKm != nil {
			cols = append(cols, "CoverageRadiusKm")
			vals = append(vals, *req.CoverageRadiusKm)
		}
		if req.IsActive != nil {
			cols = append(cols, "IsActive")
			vals = append(vals, *req.IsActive)
			newIsActive = *req.IsActive
		}
		if req.IsOnShift != nil {
			cols = append(cols, "IsOnShift")
			vals = append(vals, *req.IsOnShift)
			newIsOnShift = *req.IsOnShift
		}
		if req.OperatingSchedule != nil {
			cols = append(cols, "OperatingSchedule")
			vals = append(vals, *req.OperatingSchedule)
		}
		if req.DisabledReason != nil {
			cols = append(cols, "DisabledReason")
			vals = append(vals, *req.DisabledReason)
		}
		if req.MaxCapacity != nil {
			// VU guardrail: reject if new cap is below inflight VU
			vuStmt := spanner.Statement{
				SQL: `SELECT IFNULL(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, 1.0)), 0) AS inflight_vu
				      FROM Orders o
				      JOIN OrderLineItems li ON o.OrderId = li.OrderId
				      LEFT JOIN SupplierProducts sp ON li.ProductId = sp.ProductId AND o.SupplierId = sp.SupplierId
				      WHERE o.WarehouseId = @wid
				        AND o.State NOT IN ('COMPLETED', 'CANCELLED', 'REJECTED', 'RETURNED')`,
				Params: map[string]interface{}{"wid": warehouseID},
			}
			vuIter := txn.Query(ctx, vuStmt)
			defer vuIter.Stop()
			vuRow, vuErr := vuIter.Next()
			if vuErr != nil {
				return fmt.Errorf("vu query: %w", vuErr)
			}
			var inflightVU float64
			if err := vuRow.Columns(&inflightVU); err != nil {
				return fmt.Errorf("vu scan: %w", err)
			}
			if float64(*req.MaxCapacity) < inflightVU {
				return fmt.Errorf("VU_CAPACITY_VIOLATION:%.1f", inflightVU)
			}
			cols = append(cols, "MaxCapacityThreshold")
			vals = append(vals, *req.MaxCapacity)
		}

		// Track status change
		if newIsActive != oldIsActive || newIsOnShift != oldIsOnShift {
			statusChanged = true
		}

		// Recompute H3 coverage if lat/lng/radius changed
		lat := existingLat.Float64
		lng := existingLng.Float64
		radius := existingRadius
		name := existingName
		h3Cells := existingH3Cells

		if req.Name != nil {
			name = *req.Name
			cacheRefreshNeeded = true
		}

		if req.Lat != nil {
			lat = *req.Lat
			geoChanged = true
		}
		if req.Lng != nil {
			lng = *req.Lng
			geoChanged = true
		}
		if req.CoverageRadiusKm != nil {
			radius = *req.CoverageRadiusKm
			geoChanged = true
		}

		if geoChanged {
			h3Cells = proximity.ComputeGridCoverage(lat, lng, radius)
			cols = append(cols, "H3Indexes")
			vals = append(vals, h3Cells)
			cacheRefreshNeeded = true
		}

		oldCacheEntry = cache.WarehouseGeoEntry{
			WarehouseId: warehouseID,
			SupplierId:  supplierID,
			Name:        existingName,
			Lat:         existingLat.Float64,
			Lng:         existingLng.Float64,
			RadiusKm:    existingRadius,
			H3Cells:     existingH3Cells,
		}
		newCacheEntry = cache.WarehouseGeoEntry{
			WarehouseId: warehouseID,
			SupplierId:  supplierID,
			Name:        name,
			Lat:         lat,
			Lng:         lng,
			RadiusKm:    radius,
			H3Cells:     h3Cells,
		}

		m := spanner.Update("Warehouses", cols, vals)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}

		traceID := telemetry.TraceIDFromContext(ctx)
		if geoChanged {
			event := kafka.WarehouseSpatialUpdatedEvent{
				WarehouseId:    warehouseID,
				SupplierId:     supplierID,
				CoverageRadius: radius,
				Timestamp:      time.Now().UTC(),
			}
			if err := outbox.EmitJSON(txn, "Warehouse", warehouseID,
				kafka.EventWarehouseSpatialUpdated, kafka.TopicMain, event, traceID); err != nil {
				return err
			}
		}

		if statusChanged {
			if newIsActive != oldIsActive {
				event := kafka.WarehouseStatusChangedEvent{
					WarehouseId: warehouseID,
					SupplierId:  supplierID,
					Field:       "is_active",
					OldValue:    oldIsActive,
					NewValue:    newIsActive,
					Reason:      ptrStringOrEmpty(req.DisabledReason),
					Timestamp:   time.Now().UTC(),
				}
				if err := outbox.EmitJSON(txn, "Warehouse", warehouseID,
					kafka.EventWarehouseStatusChanged, kafka.TopicMain, event, traceID); err != nil {
					return err
				}
			}
			if newIsOnShift != oldIsOnShift {
				event := kafka.WarehouseStatusChangedEvent{
					WarehouseId: warehouseID,
					SupplierId:  supplierID,
					Field:       "is_on_shift",
					OldValue:    oldIsOnShift,
					NewValue:    newIsOnShift,
					Reason:      ptrStringOrEmpty(req.DisabledReason),
					Timestamp:   time.Now().UTC(),
				}
				if err := outbox.EmitJSON(txn, "Warehouse", warehouseID,
					kafka.EventWarehouseStatusChanged, kafka.TopicMain, event, traceID); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "ownership mismatch") {
			http.Error(w, "Warehouse not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "VU_CAPACITY_VIOLATION") {
			// Extract the inflight VU value from "VU_CAPACITY_VIOLATION:123.4"
			parts := strings.SplitN(err.Error(), "VU_CAPACITY_VIOLATION:", 2)
			inflightStr := "0"
			if len(parts) == 2 {
				inflightStr = parts[1]
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "New capacity is below current inflight volumetric load",
				"code":        "VU_CAPACITY_VIOLATION",
				"inflight_vu": inflightStr,
			})
			return
		}
		log.Printf("[WAREHOUSE] Update failed for %s: %v", warehouseID, err)
		http.Error(w, "Failed to update warehouse", http.StatusInternalServerError)
		return
	}

	if cacheRefreshNeeded {
		capturedOld := oldCacheEntry
		capturedNew := newCacheEntry
		capturedGeoChanged := geoChanged
		workers.EventPool.Submit(func() {
			if capturedGeoChanged {
				if err := cache.RemoveWarehouse(context.Background(), capturedOld); err != nil {
					log.Printf("[WAREHOUSE] Cache remove failed for %s: %v", capturedOld.WarehouseId, err)
				}
			}
			if err := cache.IndexWarehouse(context.Background(), capturedNew); err != nil {
				log.Printf("[WAREHOUSE] Cache index failed for %s: %v", capturedNew.WarehouseId, err)
			}
		})
	}

	if geoChanged {
		// Background coverage consistency check — detect orphaned retailers.
		capturedSupplier := supplierID
		workers.EventPool.Submit(func() {
			if err := proximity.VerifyCoverageConsistency(context.Background(), client, capturedSupplier); err != nil {
				log.Printf("[WAREHOUSE] Coverage audit failed post-update for supplier %s: %v", capturedSupplier, err)
			}
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "warehouse_id": warehouseID})
}

// ── Deactivate Warehouse ──────────────────────────────────────────────────

func deactivateWarehouse(w http.ResponseWriter, r *http.Request, client *spanner.Client, warehouseID string, producer *kafkago.Writer) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID := claims.ResolveSupplierID()

	// Only GLOBAL_ADMIN can deactivate warehouses
	if scope := auth.GetWarehouseScope(r.Context()); scope != nil && scope.IsNodeAdmin {
		http.Error(w, "Only headquarters admin can deactivate warehouses", http.StatusForbidden)
		return
	}

	_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify ownership + not default
		row, err := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID},
			[]string{"SupplierId", "IsDefault"})
		if err != nil {
			return fmt.Errorf("warehouse not found: %w", err)
		}
		var existingSupplierId string
		var isDefault bool
		if err := row.Columns(&existingSupplierId, &isDefault); err != nil {
			return err
		}
		if existingSupplierId != supplierID {
			return fmt.Errorf("ownership mismatch")
		}
		if isDefault {
			return fmt.Errorf("cannot_deactivate_default")
		}

		// Check for active orders at this warehouse
		countStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Orders
			      WHERE WarehouseId = @warehouseId AND State NOT IN ('COMPLETED', 'CANCELLED')`,
			Params: map[string]interface{}{"warehouseId": warehouseID},
		}
		countIter := txn.Query(ctx, countStmt)
		defer countIter.Stop()
		countRow, err := countIter.Next()
		if err != nil {
			return err
		}
		var activeOrders int64
		if err := countRow.Columns(&activeOrders); err != nil {
			return err
		}
		if activeOrders > 0 {
			return fmt.Errorf("has_active_orders:%d", activeOrders)
		}

		m := spanner.Update("Warehouses",
			[]string{"WarehouseId", "IsActive", "UpdatedAt"},
			[]interface{}{warehouseID, false, spanner.CommitTimestamp})
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		event := kafka.WarehouseStatusChangedEvent{
			WarehouseId: warehouseID,
			SupplierId:  supplierID,
			Field:       "is_active",
			OldValue:    true,
			NewValue:    false,
			Reason:      "deactivated",
			Timestamp:   time.Now().UTC(),
		}
		return outbox.EmitJSON(txn, "Warehouse", warehouseID,
			kafka.EventWarehouseStatusChanged, kafka.TopicMain, event,
			telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "ownership mismatch"):
			http.Error(w, "Warehouse not found", http.StatusNotFound)
		case strings.Contains(errStr, "cannot_deactivate_default"):
			http.Error(w, `{"error":"Cannot deactivate the default warehouse"}`, http.StatusConflict)
		case strings.Contains(errStr, "has_active_orders"):
			http.Error(w, `{"error":"Warehouse has active orders — reassign or complete them first"}`, http.StatusConflict)
		default:
			log.Printf("[WAREHOUSE] Deactivate failed for %s: %v", warehouseID, err)
			http.Error(w, "Failed to deactivate warehouse", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated", "warehouse_id": warehouseID})

}

// ── Helpers ───────────────────────────────────────────────────────────────

func ptrStringOrEmpty(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// broadcastWarehouseCreated pushes a WRH_NEW delta to all supplier admin portals
// via the telemetry FleetHub so connected map layers update in real time.
func broadcastWarehouseCreated(supplierID, warehouseID, name string, lat, lng, radiusKm float64, h3Count int) {
	delta := ws.NewDelta(ws.DeltaWarehouseNew, warehouseID, map[string]interface{}{
		"name":               name,
		"lat":                lat,
		"lng":                lng,
		"coverage_radius_km": radiusKm,
		"h3_count":           h3Count,
		"status":             "active",
	})
	data, err := json.Marshal(delta)
	if err != nil {
		log.Printf("[WAREHOUSE] marshal WRH_NEW delta: %v", err)
		return
	}
	telemetry.FleetHub.BroadcastToSupplier(supplierID, data)
}

// ── Inflight VU Query ─────────────────────────────────────────────────────

// HandleWarehouseInflightVU returns the current volumetric utilization for a warehouse.
// GET /v1/supplier/warehouse-inflight-vu?warehouse_id=X
func HandleWarehouseInflightVU(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()
		warehouseID := r.URL.Query().Get("warehouse_id")
		if warehouseID == "" {
			http.Error(w, "warehouse_id required", http.StatusBadRequest)
			return
		}

		// Verify ownership + read MaxCapacityThreshold
		row, err := spannerClient.Single().ReadRow(r.Context(), "Warehouses", spanner.Key{warehouseID},
			[]string{"SupplierId", "MaxCapacityThreshold"})
		if err != nil {
			http.Error(w, "Warehouse not found", http.StatusNotFound)
			return
		}
		var ownerID string
		var maxCap spanner.NullInt64
		if err := row.Columns(&ownerID, &maxCap); err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		if ownerID != supplierID {
			http.Error(w, "Warehouse not found", http.StatusNotFound)
			return
		}

		maxCapacity := int64(100) // default
		if maxCap.Valid {
			maxCapacity = maxCap.Int64
		}

		// Query inflight VU
		stmt := spanner.Statement{
			SQL: `SELECT IFNULL(SUM(li.Quantity * COALESCE(sp.VolumetricUnit, 1.0)), 0) AS inflight_vu
			      FROM Orders o
			      JOIN OrderLineItems li ON o.OrderId = li.OrderId
			      LEFT JOIN SupplierProducts sp ON li.ProductId = sp.ProductId AND o.SupplierId = sp.SupplierId
			      WHERE o.WarehouseId = @wid
			        AND o.State NOT IN ('COMPLETED', 'CANCELLED', 'REJECTED', 'RETURNED')`,
			Params: map[string]interface{}{"wid": warehouseID},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()
		vuRow, err := iter.Next()
		if err != nil {
			log.Printf("[WAREHOUSE] inflight VU query failed for %s: %v", warehouseID, err)
			http.Error(w, "Failed to query utilization", http.StatusInternalServerError)
			return
		}
		var inflightVU float64
		if err := vuRow.Columns(&inflightVU); err != nil {
			http.Error(w, "scan error", http.StatusInternalServerError)
			return
		}

		utilization := float64(0)
		if maxCapacity > 0 {
			utilization = (inflightVU / float64(maxCapacity)) * 100
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"warehouse_id": warehouseID,
			"inflight_vu":  inflightVU,
			"max_capacity": maxCapacity,
			"utilization":  utilization,
		})
	}
}
