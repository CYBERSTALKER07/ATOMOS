package factory

import (
	"context"
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
	ManifestId    string                    `json:"manifest_id"`
	FactoryId     string                    `json:"factory_id"`
	DriverId      string                    `json:"driver_id,omitempty"`
	DriverName    string                    `json:"driver_name,omitempty"`
	VehicleId     string                    `json:"vehicle_id,omitempty"`
	VehicleLabel  string                    `json:"vehicle_label,omitempty"`
	TruckID       string                    `json:"truck_id,omitempty"`
	TruckPlate    string                    `json:"truck_plate,omitempty"`
	State         string                    `json:"state"`
	Status        string                    `json:"status,omitempty"`
	TotalVolumeVU float64                   `json:"total_volume_vu"`
	MaxVolumeVU   float64                   `json:"max_volume_vu"`
	MaxCapacityVU float64                   `json:"max_capacity_vu"`
	StopCount     int64                     `json:"stop_count"`
	RegionCode    string                    `json:"region_code,omitempty"`
	CreatedAt     string                    `json:"created_at"`
	Transfers     []ManifestTransferSummary `json:"transfers,omitempty"`
}

type ManifestTransferSummary struct {
	TransferID  string  `json:"transfer_id"`
	ProductName string  `json:"product_name"`
	Quantity    int64   `json:"quantity"`
	VolumeVU    float64 `json:"volume_vu"`
	State       string  `json:"state"`
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
		sql := `SELECT m.ManifestId, m.FactoryId, COALESCE(m.DriverId, ''), COALESCE(d.Name, ''),
		               COALESCE(m.VehicleId, ''), COALESCE(v.Label, ''), COALESCE(v.LicensePlate, ''),
		               m.State, m.TotalVolumeVU, m.MaxVolumeVU, m.StopCount,
		               COALESCE(m.RegionCode, ''), m.CreatedAt
		        FROM FactoryTruckManifests m
		        LEFT JOIN Drivers d ON d.DriverId = m.DriverId
		        LEFT JOIN Vehicles v ON v.VehicleId = m.VehicleId
		        WHERE m.FactoryId = @fid`
		params := map[string]interface{}{"fid": factoryID}

		if stateFilter != "" {
			sql += " AND State = @state"
			params["state"] = stateFilter
		}
		sql += " ORDER BY m.CreatedAt DESC"

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
			if err := row.Columns(&m.ManifestId, &m.FactoryId, &m.DriverId, &m.DriverName,
				&m.VehicleId, &m.VehicleLabel, &m.TruckPlate,
				&m.State, &m.TotalVolumeVU, &m.MaxVolumeVU, &m.StopCount,
				&m.RegionCode, &createdAt); err != nil {
				log.Printf("[FACTORY MANIFESTS] parse error: %v", err)
				continue
			}
			m.TruckID = m.VehicleId
			m.Status = m.State
			m.MaxCapacityVU = m.MaxVolumeVU
			m.CreatedAt = createdAt.Format(time.RFC3339)
			manifests = append(manifests, m)
		}

		if err := attachManifestTransfers(r.Context(), spannerClient, manifests); err != nil {
			log.Printf("[FACTORY MANIFESTS] transfer attach error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":      manifests,
			"manifests": manifests,
			"total":     len(manifests),
		})
	}
}

func attachManifestTransfers(ctx context.Context, spannerClient *spanner.Client, manifests []ManifestResponse) error {
	if len(manifests) == 0 {
		return nil
	}

	manifestIDs := make([]string, 0, len(manifests))
	manifestIndex := make(map[string]int, len(manifests))
	for i := range manifests {
		manifestIDs = append(manifestIDs, manifests[i].ManifestId)
		manifestIndex[manifests[i].ManifestId] = i
	}

	transferStmt := spanner.Statement{
		SQL: `SELECT ManifestId, TransferId, State, TotalVolumeVU
		      FROM InternalTransferOrders
		      WHERE ManifestId IN UNNEST(@manifest_ids)
		      ORDER BY UpdatedAt DESC`,
		Params: map[string]interface{}{"manifest_ids": manifestIDs},
	}
	transferIter := spannerClient.Single().Query(ctx, transferStmt)
	defer transferIter.Stop()

	type transferLocation struct {
		manifestIdx int
		transferIdx int
	}
	transferLocations := make(map[string]transferLocation)
	transferIDs := make([]string, 0)

	for {
		row, err := transferIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		var manifestID string
		var transfer ManifestTransferSummary
		if err := row.Columns(&manifestID, &transfer.TransferID, &transfer.State, &transfer.VolumeVU); err != nil {
			continue
		}

		idx, ok := manifestIndex[manifestID]
		if !ok {
			continue
		}

		manifests[idx].Transfers = append(manifests[idx].Transfers, transfer)
		transferIdx := len(manifests[idx].Transfers) - 1
		transferLocations[transfer.TransferID] = transferLocation{manifestIdx: idx, transferIdx: transferIdx}
		transferIDs = append(transferIDs, transfer.TransferID)
	}

	if len(transferIDs) == 0 {
		return nil
	}

	itemStmt := spanner.Statement{
		SQL: `SELECT i.TransferId,
		             COALESCE(SUM(i.Quantity), 0) AS total_quantity,
		             COALESCE(MIN(p.Name), '') AS product_name
		      FROM InternalTransferItems i
		      LEFT JOIN Products p ON p.ProductId = i.ProductId
		      WHERE i.TransferId IN UNNEST(@transfer_ids)
		      GROUP BY i.TransferId`,
		Params: map[string]interface{}{"transfer_ids": transferIDs},
	}
	itemIter := spannerClient.Single().Query(ctx, itemStmt)
	defer itemIter.Stop()

	for {
		row, err := itemIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		var transferID string
		var quantity int64
		var productName string
		if err := row.Columns(&transferID, &quantity, &productName); err != nil {
			continue
		}

		loc, ok := transferLocations[transferID]
		if !ok {
			continue
		}

		manifests[loc.manifestIdx].Transfers[loc.transferIdx].Quantity = quantity
		if productName != "" {
			manifests[loc.manifestIdx].Transfers[loc.transferIdx].ProductName = productName
		}
	}

	for transferID, loc := range transferLocations {
		if manifests[loc.manifestIdx].Transfers[loc.transferIdx].ProductName == "" {
			manifests[loc.manifestIdx].Transfers[loc.transferIdx].ProductName = "Transfer " + shortRef(transferID)
		}
	}

	return nil
}

func shortRef(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
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
		if _, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{m})
		}); err != nil {
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
