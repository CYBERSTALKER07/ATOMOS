package proximity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ─── Territory Recommendation Handlers ─────────────────────────────────────────
//
// Two endpoints:
//   GET  /v1/supplier/warehouses/territory-preview   — Read-only Voronoi + scores
//   POST /v1/supplier/warehouses/apply-territory      — Atomic H3 cell reassignment
//
// The "Apply" handler enforces DispatchLock to prevent split-brain routing while
// H3 cell borders are being redrawn.

// HandlePreviewTerritories returns a Voronoi-style territory map with load-aware
// suitability scores for all supplier warehouses. Read-only — no mutations.
// GET /v1/supplier/warehouses/territory-preview
func HandlePreviewTerritories(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		supplierID := claims.ResolveSupplierID()

		// Fetch all active warehouses with geo + H3 data
		warehouses, err := fetchWarehouseGeoList(ctx, spannerClient, supplierID)
		if err != nil {
			log.Printf("[TERRITORY] fetch warehouses: %v", err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		if len(warehouses) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"proposal": TerritoryProposal{Assignments: map[string][]CellAssignment{}},
			})
			return
		}

		// Enrich with live load from Redis
		for i := range warehouses {
			maxCap := getWarehouseMaxCapacity(ctx, spannerClient, warehouses[i].WarehouseId)
			warehouses[i].LoadPercent = cache.GetWarehouseLoad(ctx, warehouses[i].WarehouseId, maxCap)
		}

		proposal := GenerateNaturalTerritories(warehouses)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"proposal":   proposal,
			"warehouses": warehouses,
		})
	}
}

// HandleApplyTerritory atomically reassigns H3 cells to a target warehouse.
// Removes the cells from any other warehouse's H3Indexes (preventing double-fulfillment)
// and writes a TERRITORY_REALLOCATION audit record.
//
// POST /v1/supplier/warehouses/apply-territory
//
// Request body:
//
//	{
//	  "warehouse_id": "...",
//	  "h3_cells": ["872830828ffffff", ...]
//	}
//
// Preconditions:
//   - DispatchLock must NOT be active for the supplier (prevents mid-dispatch border change)
//   - The handler itself acquires a transient SPATIAL_UPDATE lock for the duration of the txn
//
// isDispatchLocked is injected to avoid circular imports (proximity ↔ warehouse).
func HandleApplyTerritory(spannerClient *spanner.Client, isDispatchLocked func(ctx context.Context, supplierID, warehouseID, factoryID string) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			WarehouseID string   `json:"warehouse_id"`
			H3Cells     []string `json:"h3_cells"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WarehouseID == "" || len(req.H3Cells) == 0 {
			http.Error(w, `{"error":"warehouse_id and h3_cells[] required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		supplierID := claims.ResolveSupplierID()

		// Validate warehouse belongs to this supplier
		whRow, err := spannerClient.Single().ReadRow(ctx, "Warehouses", spanner.Key{req.WarehouseID}, []string{"SupplierId"})
		if err != nil {
			http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
			return
		}
		var whOwner string
		if cErr := whRow.Column(0, &whOwner); cErr != nil || whOwner != supplierID {
			http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
			return
		}

		// Guard: reject if any dispatch operation is active (supplier-wide check)
		if isDispatchLocked(ctx, supplierID, "", "") {
			http.Error(w, `{"error":"dispatch lock active — wait for current dispatch to complete"}`, http.StatusConflict)
			return
		}

		// Acquire transient SPATIAL_UPDATE lock scoped to the TARGET warehouse
		spatialLockID := uuid.New().String()
		_, lockErr := spannerClient.Apply(ctx, []*spanner.Mutation{
			spanner.InsertMap("DispatchLocks", map[string]interface{}{
				"LockId":      spatialLockID,
				"SupplierId":  supplierID,
				"WarehouseId": req.WarehouseID,
				"LockType":    "SPATIAL_UPDATE",
				"LockedAt":    spanner.CommitTimestamp,
				"LockedBy":    claims.UserID,
			}),
		})
		if lockErr != nil {
			log.Printf("[TERRITORY] Failed to acquire spatial lock: %v", lockErr)
			http.Error(w, `{"error":"could not acquire spatial lock"}`, http.StatusConflict)
			return
		}
		// Guarantee lock release even on panic
		defer func() {
			_, _ = spannerClient.Apply(context.Background(), []*spanner.Mutation{
				spanner.UpdateMap("DispatchLocks", map[string]interface{}{
					"LockId":     spatialLockID,
					"UnlockedAt": TashkentNow(),
				}),
			})
		}()

		// Deduplicate and validate cell IDs
		cellSet := make(map[string]struct{}, len(req.H3Cells))
		for _, c := range req.H3Cells {
			if _, _, ok := parseCellCoords(c); ok {
				cellSet[c] = struct{}{}
			}
		}
		if len(cellSet) == 0 {
			http.Error(w, `{"error":"no valid H3 cells"}`, http.StatusBadRequest)
			return
		}

		// Atomic territory update in a single ReadWriteTransaction
		var removedFrom []string // warehouse IDs that lost cells
		_, txnErr := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			removedFrom = nil // reset on retry

			// Step 1: Read all active warehouses for this supplier with their H3Indexes (native ARRAY)
			stmt := spanner.Statement{
				SQL: `SELECT WarehouseId, H3Indexes
				      FROM Warehouses
				      WHERE SupplierId = @sid AND IsActive = true`,
				Params: map[string]interface{}{"sid": supplierID},
			}
			iter := txn.Query(ctx, stmt)
			type whRow struct {
				id    string
				cells map[string]struct{}
			}
			var allWarehouses []whRow
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return fmt.Errorf("read warehouses: %w", err)
				}
				var wid string
				var h3arr []string
				if err := row.Columns(&wid, &h3arr); err != nil {
					return err
				}
				cells := make(map[string]struct{}, len(h3arr))
				for _, c := range h3arr {
					cells[c] = struct{}{}
				}
				allWarehouses = append(allWarehouses, whRow{id: wid, cells: cells})
			}
			iter.Stop()

			// Step 2: Search-and-Destroy — remove claimed cells from every OTHER warehouse
			var mutations []*spanner.Mutation
			for _, wh := range allWarehouses {
				if wh.id == req.WarehouseID {
					continue // target warehouse — handled in Step 3
				}
				modified := false
				for c := range cellSet {
					if _, has := wh.cells[c]; has {
						delete(wh.cells, c)
						modified = true
					}
				}
				if modified {
					removedFrom = append(removedFrom, wh.id)
					// spanner.UpdateMap prevents partial state corruption on ARRAY columns
					mutations = append(mutations, spanner.UpdateMap("Warehouses", map[string]interface{}{
						"WarehouseId": wh.id,
						"H3Indexes":   cellMapToSlice(wh.cells),
						"UpdatedAt":   TashkentNow(),
					}))
				}
			}

			// Step 3: Merge — add claimed cells to target warehouse (preserving its existing cells)
			var targetExisting map[string]struct{}
			for _, wh := range allWarehouses {
				if wh.id == req.WarehouseID {
					targetExisting = wh.cells
					break
				}
			}
			if targetExisting == nil {
				targetExisting = make(map[string]struct{})
			}
			for c := range cellSet {
				targetExisting[c] = struct{}{}
			}
			mutations = append(mutations, spanner.UpdateMap("Warehouses", map[string]interface{}{
				"WarehouseId": req.WarehouseID,
				"H3Indexes":   cellMapToSlice(targetExisting),
				"UpdatedAt":   TashkentNow(),
			}))

			// Step 4: Audit record with Tashkent timestamp
			mutations = append(mutations, spanner.InsertMap("DispatchAudit", map[string]interface{}{
				"AuditId":     uuid.New().String(),
				"SupplierId":  supplierID,
				"WarehouseId": req.WarehouseID,
				"AuditType":   "TERRITORY_REALLOCATION",
				"CreatedAt":   spanner.CommitTimestamp,
			}))

			txn.BufferWrite(mutations)
			return nil
		})

		if txnErr != nil {
			log.Printf("[TERRITORY] Apply transaction failed: %v", txnErr)
			http.Error(w, fmt.Sprintf(`{"error":"transaction failed: %s"}`, txnErr.Error()), http.StatusInternalServerError)
			return
		}

		// Post-commit: Trigger coverage audit asynchronously
		go func() {
			bgCtx, bgCancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer bgCancel()
			if err := VerifyCoverageConsistency(bgCtx, spannerClient, supplierID); err != nil {
				log.Printf("[TERRITORY] Post-apply coverage audit failed: %v", err)
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             "APPLIED",
			"warehouse_id":       req.WarehouseID,
			"cells_assigned":     len(cellSet),
			"cells_removed_from": removedFrom,
		})
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// fetchWarehouseGeoList returns all active warehouses for a supplier with geo data.
func fetchWarehouseGeoList(ctx context.Context, client *spanner.Client, supplierID string) ([]WarehouseGeo, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Lat, Lng, CoverageRadiusKm, H3Indexes
		      FROM Warehouses
		      WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]interface{}{"sid": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var warehouses []WarehouseGeo
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var wh WarehouseGeo
		var h3arr []string
		if err := row.Columns(&wh.WarehouseId, &wh.Lat, &wh.Lng, &wh.CoverageRadiusKm, &h3arr); err != nil {
			return nil, err
		}
		wh.H3Indexes = h3arr
		warehouses = append(warehouses, wh)
	}
	return warehouses, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func cellMapToSlice(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}
