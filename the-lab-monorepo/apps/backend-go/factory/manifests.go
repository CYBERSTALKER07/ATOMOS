package factory

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Factory Truck Manifests ───────────────────────────────────────────────────
// Loading Bay Kanban: groups InternalTransferOrders onto factory trucks.

type ManifestResponse struct {
	ManifestId    string  `json:"manifest_id"`
	FactoryId     string  `json:"factory_id"`
	DriverId      string  `json:"driver_id,omitempty"`
	VehicleId     string  `json:"vehicle_id,omitempty"`
	State         string  `json:"state"`
	TotalVolumeVU float64 `json:"total_volume_vu"`
	MaxVolumeVU   float64 `json:"max_volume_vu"`
	StopCount     int64   `json:"stop_count"`
	RegionCode    string  `json:"region_code,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// HandleFactoryManifests — GET /v1/factory/manifests (factory-scoped, Loading Bay Kanban)
func HandleFactoryManifests(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		factoryID, ok := auth.MustFactoryID(w, r.Context())
		if !ok {
			return
		}

		stateFilter := r.URL.Query().Get("state")
		sql := `SELECT ManifestId, FactoryId, COALESCE(DriverId, ''), COALESCE(VehicleId, ''),
		               State, TotalVolumeVU, MaxVolumeVU, StopCount,
		               COALESCE(RegionCode, ''), CreatedAt
		        FROM FactoryTruckManifests WHERE FactoryId = @fid`
		params := map[string]interface{}{"fid": factoryID}

		if stateFilter != "" {
			sql += " AND State = @state"
			params["state"] = stateFilter
		}
		sql += " ORDER BY CreatedAt DESC"

		stmt := spanner.Statement{SQL: sql, Params: params}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		manifests := []ManifestResponse{}
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[FACTORY MANIFESTS] list error: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var m ManifestResponse
			var createdAt time.Time
			if err := row.Columns(&m.ManifestId, &m.FactoryId, &m.DriverId, &m.VehicleId,
				&m.State, &m.TotalVolumeVU, &m.MaxVolumeVU, &m.StopCount,
				&m.RegionCode, &createdAt); err != nil {
				log.Printf("[FACTORY MANIFESTS] parse error: %v", err)
				continue
			}
			m.CreatedAt = createdAt.Format(time.RFC3339)
			manifests = append(manifests, m)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": manifests})
	}
}

// HandleFactoryManifestDetail — GET /v1/factory/manifests/{id}
// Returns manifest header + associated transfer orders.
func HandleFactoryManifestDetail(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		manifestID := strings.TrimPrefix(r.URL.Path, "/v1/factory/manifests/")
		if manifestID == "" || strings.Contains(manifestID, "/") {
			http.Error(w, "manifest_id required in path", http.StatusBadRequest)
			return
		}

		factoryID, ok := auth.MustFactoryID(w, r.Context())
		if !ok {
			return
		}

		// Fetch manifest header
		stmt := spanner.Statement{
			SQL: `SELECT ManifestId, FactoryId, COALESCE(DriverId, ''), COALESCE(VehicleId, ''),
			             State, TotalVolumeVU, MaxVolumeVU, StopCount,
			             COALESCE(RegionCode, ''), CreatedAt
			      FROM FactoryTruckManifests WHERE ManifestId = @mid AND FactoryId = @fid`,
			Params: map[string]interface{}{"mid": manifestID, "fid": factoryID},
		}
		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			http.Error(w, `{"error":"manifest not found"}`, http.StatusNotFound)
			return
		}

		var m ManifestResponse
		var createdAt time.Time
		if err := row.Columns(&m.ManifestId, &m.FactoryId, &m.DriverId, &m.VehicleId,
			&m.State, &m.TotalVolumeVU, &m.MaxVolumeVU, &m.StopCount,
			&m.RegionCode, &createdAt); err != nil {
			log.Printf("[FACTORY MANIFESTS] detail parse error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)

		// Fetch linked transfer orders
		transferStmt := spanner.Statement{
			SQL: `SELECT TransferId, WarehouseId, State, TotalVolumeVU, Source
			      FROM InternalTransferOrders WHERE ManifestId = @mid`,
			Params: map[string]interface{}{"mid": manifestID},
		}
		transferIter := spannerClient.Single().Query(r.Context(), transferStmt)
		defer transferIter.Stop()

		type LinkedTransfer struct {
			TransferId    string  `json:"transfer_id"`
			WarehouseId   string  `json:"warehouse_id"`
			State         string  `json:"state"`
			TotalVolumeVU float64 `json:"total_volume_vu"`
			Source        string  `json:"source"`
		}
		transfers := []LinkedTransfer{}
		for {
			tRow, tErr := transferIter.Next()
			if tErr == iterator.Done {
				break
			}
			if tErr != nil {
				break
			}
			var lt LinkedTransfer
			if err := tRow.Columns(&lt.TransferId, &lt.WarehouseId, &lt.State, &lt.TotalVolumeVU, &lt.Source); err != nil {
				continue
			}
			transfers = append(transfers, lt)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"manifest":  m,
			"transfers": transfers,
		})
	}
}

// HandleFactoryManifestTransition handles state changes on manifests.
// POST /v1/factory/manifests/{id}/load     → READY_FOR_LOADING → LOADING
// POST /v1/factory/manifests/{id}/dispatch → LOADING → DISPATCHED
func HandleFactoryManifestTransition(spannerClient *spanner.Client) http.HandlerFunc {
	validManifestTransitions := map[string]string{
		"load":     "LOADING",
		"dispatch": "DISPATCHED",
	}
	validManifestFromStates := map[string]string{
		"LOADING":    "READY_FOR_LOADING",
		"DISPATCHED": "LOADING",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse: /v1/factory/manifests/{id}/{action}
		path := strings.TrimPrefix(r.URL.Path, "/v1/factory/manifests/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 {
			http.Error(w, "manifest_id and action required", http.StatusBadRequest)
			return
		}
		manifestID := parts[0]
		action := parts[1]

		targetState, ok := validManifestTransitions[action]
		if !ok {
			http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
			return
		}

		factoryID, fOk := auth.MustFactoryID(w, r.Context())
		if !fOk {
			return
		}

		row, err := spannerClient.Single().ReadRow(r.Context(), "FactoryTruckManifests",
			spanner.Key{manifestID}, []string{"State", "FactoryId"})
		if err != nil {
			http.Error(w, `{"error":"manifest not found"}`, http.StatusNotFound)
			return
		}

		var currentState, rowFid string
		if err := row.Columns(&currentState, &rowFid); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if rowFid != factoryID {
			http.Error(w, `{"error":"manifest does not belong to this factory"}`, http.StatusForbidden)
			return
		}

		expectedFrom := validManifestFromStates[targetState]
		if currentState != expectedFrom {
			http.Error(w, `{"error":"invalid manifest state transition"}`, http.StatusConflict)
			return
		}

		m := spanner.Update("FactoryTruckManifests",
			[]string{"ManifestId", "State"},
			[]interface{}{manifestID, targetState},
		)
		if _, err := spannerClient.Apply(r.Context(), []*spanner.Mutation{m}); err != nil {
			log.Printf("[FACTORY MANIFESTS] transition error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":      targetState,
			"manifest_id": manifestID,
		})
	}
}
