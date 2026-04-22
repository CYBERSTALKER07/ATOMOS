package proximity

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"backend-go/auth"
	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// WarehouseLoadEntry is a single warehouse's live load status — Redis-only, no Spanner.
type WarehouseLoadEntry struct {
	WarehouseId string  `json:"warehouse_id"`
	Name        string  `json:"name"`
	QueueDepth  int64   `json:"queue_depth"`
	MaxCapacity int64   `json:"max_capacity"`
	LoadPercent float64 `json:"load_percent"`
	LoadStatus  string  `json:"load_status"` // GREEN | YELLOW | RED
}

// HandleWarehouseLoads — GET /v1/supplier/warehouse-loads
// Lightweight Redis-primary endpoint that returns current queue depth and load
// factor for all supplier warehouses. Designed for 15-second polling from the
// admin dashboard. Spanner is used only to fetch the warehouse list + max capacity;
// the actual load data comes from Redis.
func HandleWarehouseLoads(spannerClient *spanner.Client) http.HandlerFunc {
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

		ctx := r.Context()
		supplierID := claims.ResolveSupplierID()

		// Fetch warehouse metadata (ID, name, max capacity) from Spanner
		warehouses, err := fetchWarehouseLoadMeta(ctx, spannerClient, supplierID)
		if err != nil {
			log.Printf("[WAREHOUSE-LOADS] fetch meta error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Batch-fetch queue depths from Redis
		whIDs := make([]string, len(warehouses))
		for i, wh := range warehouses {
			whIDs[i] = wh.WarehouseId
		}
		depths := cache.GetAllWarehouseLoads(ctx, whIDs)

		// Build response
		entries := make([]WarehouseLoadEntry, len(warehouses))
		for i, wh := range warehouses {
			depth := depths[wh.WarehouseId]
			maxCap := wh.MaxCapacity
			if maxCap <= 0 {
				maxCap = 100
			}
			loadPct := float64(depth) / float64(maxCap)
			if loadPct > 1.0 {
				loadPct = 1.0
			}
			entries[i] = WarehouseLoadEntry{
				WarehouseId: wh.WarehouseId,
				Name:        wh.Name,
				QueueDepth:  depth,
				MaxCapacity: maxCap,
				LoadPercent: loadPct,
				LoadStatus:  cache.LoadStatus(loadPct),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"warehouses": entries,
		})
	}
}

// warehouseLoadMeta holds just the fields needed for the loads endpoint.
type warehouseLoadMeta struct {
	WarehouseId string
	Name        string
	MaxCapacity int64
}

func fetchWarehouseLoadMeta(ctx context.Context, client *spanner.Client, supplierID string) ([]warehouseLoadMeta, error) {
	stmt := spanner.Statement{
		SQL: `SELECT WarehouseId, Name, COALESCE(MaxCapacityThreshold, 100) AS MaxCapacity
		      FROM Warehouses
		      WHERE SupplierId = @sid AND IsActive = true`,
		Params: map[string]interface{}{"sid": supplierID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []warehouseLoadMeta
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var wh warehouseLoadMeta
		if err := row.Columns(&wh.WarehouseId, &wh.Name, &wh.MaxCapacity); err != nil {
			continue
		}
		results = append(results, wh)
	}

	return results, nil
}
