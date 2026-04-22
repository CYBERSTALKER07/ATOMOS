package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Internal Transfer Order State Machine ─────────────────────────────────────
//
//  DRAFT → APPROVED → LOADING → DISPATCHED → IN_TRANSIT → ARRIVED → RECEIVED
//    ↓
//  CANCELLED
//

var validTransferTransitions = map[string][]string{
	"DRAFT":      {"APPROVED", "CANCELLED"},
	"APPROVED":   {"LOADING", "CANCELLED"},
	"LOADING":    {"DISPATCHED"},
	"DISPATCHED": {"IN_TRANSIT"},
	"IN_TRANSIT": {"ARRIVED"},
	"ARRIVED":    {"RECEIVED"},
}

func isValidTransition(from, to string) bool {
	for _, v := range validTransferTransitions[from] {
		if v == to {
			return true
		}
	}
	return false
}

// TransferResponse is the JSON shape for an InternalTransferOrder.
type TransferResponse struct {
	TransferId    string               `json:"transfer_id"`
	FactoryId     string               `json:"factory_id"`
	WarehouseId   string               `json:"warehouse_id"`
	SupplierId    string               `json:"supplier_id"`
	State         string               `json:"state"`
	TotalVolumeVU float64              `json:"total_volume_vu"`
	ManifestId    string               `json:"manifest_id,omitempty"`
	Source        string               `json:"source"`
	CreatedAt     string               `json:"created_at"`
	Items         []TransferItemDetail `json:"items,omitempty"`
}

type TransferItemDetail struct {
	ItemId    string  `json:"item_id"`
	ProductId string  `json:"product_id"`
	Quantity  int64   `json:"quantity"`
	VolumeVU  float64 `json:"volume_vu"`
}

// TransferService holds dependencies for transfer endpoints.
type TransferService struct {
	Spanner  *spanner.Client
	Producer *kafka.Writer
}

// HandleListTransfers — GET /v1/factory/transfers (factory-scoped)
func (s *TransferService) HandleListTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	stateFilter := r.URL.Query().Get("state")
	sql := `SELECT TransferId, FactoryId, WarehouseId, SupplierId, State,
	               TotalVolumeVU, COALESCE(ManifestId, ''), Source, CreatedAt
	        FROM InternalTransferOrders WHERE FactoryId = @fid`
	params := map[string]interface{}{"fid": factoryID}

	if stateFilter != "" {
		sql += " AND State = @state"
		params["state"] = stateFilter
	}
	sql += " ORDER BY CreatedAt DESC"

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	transfers := []TransferResponse{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[TRANSFERS] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var t TransferResponse
		var createdAt time.Time
		if err := row.Columns(&t.TransferId, &t.FactoryId, &t.WarehouseId, &t.SupplierId,
			&t.State, &t.TotalVolumeVU, &t.ManifestId, &t.Source, &createdAt); err != nil {
			log.Printf("[TRANSFERS] parse error: %v", err)
			continue
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		transfers = append(transfers, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": transfers})
}

// HandleTransferDetail — GET /v1/factory/transfers/{id}
func (s *TransferService) HandleTransferDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	transferID := strings.TrimPrefix(r.URL.Path, "/v1/factory/transfers/")
	if transferID == "" || strings.Contains(transferID, "/") {
		http.Error(w, "transfer_id required in path", http.StatusBadRequest)
		return
	}

	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	// Fetch transfer header
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, FactoryId, WarehouseId, SupplierId, State,
		             TotalVolumeVU, COALESCE(ManifestId, ''), Source, CreatedAt
		      FROM InternalTransferOrders WHERE TransferId = @tid AND FactoryId = @fid`,
		Params: map[string]interface{}{"tid": transferID, "fid": factoryID},
	}
	iter := s.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		http.Error(w, `{"error":"transfer not found"}`, http.StatusNotFound)
		return
	}

	var t TransferResponse
	var createdAt time.Time
	if err := row.Columns(&t.TransferId, &t.FactoryId, &t.WarehouseId, &t.SupplierId,
		&t.State, &t.TotalVolumeVU, &t.ManifestId, &t.Source, &createdAt); err != nil {
		log.Printf("[TRANSFERS] detail parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	t.CreatedAt = createdAt.Format(time.RFC3339)

	// Fetch items (interleaved)
	itemStmt := spanner.Statement{
		SQL:    `SELECT ItemId, ProductId, Quantity, VolumeVU FROM InternalTransferItems WHERE TransferId = @tid`,
		Params: map[string]interface{}{"tid": transferID},
	}
	itemIter := s.Spanner.Single().Query(r.Context(), itemStmt)
	defer itemIter.Stop()

	for {
		itemRow, err := itemIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var item TransferItemDetail
		if err := itemRow.Columns(&item.ItemId, &item.ProductId, &item.Quantity, &item.VolumeVU); err != nil {
			continue
		}
		t.Items = append(t.Items, item)
	}
	if t.Items == nil {
		t.Items = []TransferItemDetail{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// HandleTransferTransition handles state machine transitions.
// POST /v1/factory/transfers/{id}/accept  → DRAFT → APPROVED
// POST /v1/factory/transfers/{id}/start-loading → APPROVED → LOADING
// POST /v1/factory/transfers/{id}/dispatch → LOADING → DISPATCHED
// POST /v1/factory/transfers/{id}/in-transit → DISPATCHED → IN_TRANSIT
// POST /v1/factory/transfers/{id}/arrive → IN_TRANSIT → ARRIVED
// POST /v1/factory/transfers/{id}/cancel → DRAFT|APPROVED → CANCELLED
func (s *TransferService) HandleTransferTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse: /v1/factory/transfers/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/v1/factory/transfers/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "transfer_id and action required in path", http.StatusBadRequest)
		return
	}
	transferID := parts[0]
	action := parts[1]

	targetState := ""
	switch action {
	case "accept", "approve":
		// Delegate to dedicated approval handler with convoy manifest generation
		s.HandleApproveTransfer(w, r)
		return
	case "start-loading":
		targetState = "LOADING"
	case "dispatch":
		targetState = "DISPATCHED"
	case "in-transit":
		targetState = "IN_TRANSIT"
	case "arrive":
		targetState = "ARRIVED"
	case "cancel":
		targetState = "CANCELLED"
	default:
		http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
		return
	}

	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	err := func() error {
		_, txErr := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, readErr := txn.ReadRow(ctx, "InternalTransferOrders",
				spanner.Key{transferID}, []string{"State", "FactoryId", "SupplierId", "WarehouseId"})
			if readErr != nil {
				return readErr
			}

			var currentState, rowFactoryID, supplierID, warehouseID string
			if colErr := row.Columns(&currentState, &rowFactoryID, &supplierID, &warehouseID); colErr != nil {
				return colErr
			}

			if rowFactoryID != factoryID {
				return fmt.Errorf("ownership mismatch")
			}
			if !isValidTransition(currentState, targetState) {
				return fmt.Errorf("invalid transition: %s → %s", currentState, targetState)
			}

			m := spanner.Update("InternalTransferOrders",
				[]string{"TransferId", "State", "UpdatedAt"},
				[]interface{}{transferID, targetState, spanner.CommitTimestamp},
			)
			if writeErr := txn.BufferWrite([]*spanner.Mutation{m}); writeErr != nil {
				return writeErr
			}

			evt := map[string]interface{}{
				"event":        internalKafka.EventTransferStateChanged,
				"transfer_id":  transferID,
				"factory_id":   factoryID,
				"warehouse_id": warehouseID,
				"supplier_id":  supplierID,
				"from_state":   currentState,
				"to_state":     targetState,
				"timestamp":    time.Now().UTC().Format(time.RFC3339),
			}
			return outbox.EmitJSON(txn, "InternalTransferOrder", transferID,
				internalKafka.EventTransferStateChanged, internalKafka.TopicMain, evt,
				telemetry.TraceIDFromContext(ctx))
		})
		return txErr
	}()
	if err != nil {
		switch {
		case errors.Is(err, spanner.ErrRowNotFound):
			http.Error(w, `{"error":"transfer not found"}`, http.StatusNotFound)
		case strings.Contains(err.Error(), "ownership mismatch"):
			http.Error(w, `{"error":"transfer does not belong to this factory"}`, http.StatusForbidden)
		case strings.Contains(err.Error(), "invalid transition"):
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
		default:
			log.Printf("[TRANSFERS] transition error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":      targetState,
		"transfer_id": transferID,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// HandleApproveTransfer — POST /v1/factory/transfers/{id}/approve
//
// Dedicated approval handler: DRAFT → APPROVED with post-approval convoy
// manifest generation. Convoy manifests are deferred to this point (not DRAFT
// creation) so the Volumetric Engine operates on confirmed transfers only.
//
// Flow:
//  1. Validate DRAFT → APPROVED transition
//  2. Apply state change
//  3. Generate FactoryTruckManifests (convoy splitting at 400 VU / Class-C)
//  4. Emit Kafka event
// ═══════════════════════════════════════════════════════════════════════════════

func (s *TransferService) HandleApproveTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse: /v1/factory/transfers/{id}/approve
	path := strings.TrimPrefix(r.URL.Path, "/v1/factory/transfers/")
	transferID := strings.TrimSuffix(path, "/approve")
	if transferID == "" || strings.Contains(transferID, "/") {
		http.Error(w, "transfer_id required in path", http.StatusBadRequest)
		return
	}

	factoryID, ok := auth.MustFactoryID(w, r.Context())
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// 1. Read transfer and validate DRAFT → APPROVED
	row, err := s.Spanner.Single().ReadRow(ctx, "InternalTransferOrders",
		spanner.Key{transferID},
		[]string{"State", "FactoryId", "SupplierId", "WarehouseId", "TotalVolumeVU"})
	if err != nil {
		http.Error(w, `{"error":"transfer not found"}`, http.StatusNotFound)
		return
	}

	var currentState, rowFactoryID, supplierID, warehouseID string
	var totalVolumeVU float64
	if err := row.Columns(&currentState, &rowFactoryID, &supplierID, &warehouseID, &totalVolumeVU); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if rowFactoryID != factoryID {
		http.Error(w, `{"error":"transfer does not belong to this factory"}`, http.StatusForbidden)
		return
	}

	if currentState != "DRAFT" {
		http.Error(w, fmt.Sprintf(`{"error":"invalid transition: %s → APPROVED (must be DRAFT)"}`, currentState), http.StatusConflict)
		return
	}

	// 2. Apply DRAFT → APPROVED
	m := spanner.Update("InternalTransferOrders",
		[]string{"TransferId", "State", "UpdatedAt"},
		[]interface{}{transferID, "APPROVED", spanner.CommitTimestamp},
	)
	if _, err := s.Spanner.Apply(ctx, []*spanner.Mutation{m}); err != nil {
		log.Printf("[TRANSFERS] approve error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. Post-approval: generate convoy manifests via Volumetric Engine
	convoyCount := int(math.Ceil(totalVolumeVU / FactoryClassCCapacityVU))
	if convoyCount < 1 {
		convoyCount = 1
	}

	var manifestIDs []string
	if convoyCount > 1 || totalVolumeVU > 0 {
		var mutations []*spanner.Mutation
		for i := 0; i < convoyCount; i++ {
			manifestID := uuid.New().String()
			truckVU := FactoryClassCCapacityVU
			if i == convoyCount-1 {
				// Last truck gets the remainder
				truckVU = totalVolumeVU - float64(i)*FactoryClassCCapacityVU
				if truckVU <= 0 {
					continue
				}
			}
			mutations = append(mutations, spanner.InsertMap("FactoryTruckManifests", map[string]interface{}{
				"ManifestId":    manifestID,
				"FactoryId":     factoryID,
				"State":         "PENDING",
				"TotalVolumeVU": truckVU,
				"MaxVolumeVU":   FactoryClassCCapacityVU,
				"StopCount":     int64(1),
				"CreatedAt":     spanner.CommitTimestamp,
			}))
			manifestIDs = append(manifestIDs, manifestID)
		}
		if len(mutations) > 0 {
			if _, err := s.Spanner.Apply(ctx, mutations); err != nil {
				log.Printf("[TRANSFERS] convoy manifest creation failed for %s: %v", transferID, err)
				// Non-fatal — transfer is already APPROVED, manifests can be retried
			} else {
				log.Printf("[TRANSFERS] Created %d convoy manifests for transfer %s (%.1f VU)",
					len(manifestIDs), transferID[:8], totalVolumeVU)
			}
		}
	}

	// 4. Emit transfer-approval event through outbox (durable relay).
	evt := map[string]interface{}{
		"event":        internalKafka.EventTransferApproved,
		"transfer_id":  transferID,
		"factory_id":   factoryID,
		"warehouse_id": warehouseID,
		"supplier_id":  supplierID,
		"volume_vu":    totalVolumeVU,
		"convoy_count": convoyCount,
		"manifest_ids": manifestIDs,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}
	if _, emitErr := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return outbox.EmitJSON(txn, "InternalTransferOrder", transferID,
			internalKafka.EventTransferApproved, internalKafka.TopicMain, evt,
			telemetry.TraceIDFromContext(ctx))
	}); emitErr != nil {
		log.Printf("[TRANSFERS] approve outbox emit failed for %s: %v", transferID, emitErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer_id":  transferID,
		"state":        "APPROVED",
		"convoy_count": convoyCount,
		"manifest_ids": manifestIDs,
		"volume_vu":    totalVolumeVU,
	})
}

// HandleWarehouseReceiveTransfer — POST /v1/warehouse/transfers/{id}/receive
// Warehouse-scoped endpoint: ARRIVED → RECEIVED. Increments warehouse stock.
func (s *TransferService) HandleWarehouseReceiveTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	transferID := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/transfers/")
	transferID = strings.TrimSuffix(transferID, "/receive")
	if transferID == "" {
		http.Error(w, "transfer_id required in path", http.StatusBadRequest)
		return
	}

	whID := auth.EffectiveWarehouseID(r.Context())
	if whID == "" {
		http.Error(w, `{"error":"warehouse scope required"}`, http.StatusBadRequest)
		return
	}

	// Read transfer + items atomically
	var currentState, factoryID, supplierID, rowWhID string
	items := []TransferItemDetail{}

	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read transfer header
		hdrRow, err := txn.ReadRow(ctx, "InternalTransferOrders",
			spanner.Key{transferID}, []string{"State", "FactoryId", "SupplierId", "WarehouseId"})
		if err != nil {
			return fmt.Errorf("transfer not found: %w", err)
		}
		if err := hdrRow.Columns(&currentState, &factoryID, &supplierID, &rowWhID); err != nil {
			return err
		}
		if rowWhID != whID {
			return fmt.Errorf("transfer does not belong to this warehouse")
		}
		if currentState != "ARRIVED" {
			return fmt.Errorf("invalid transition: %s → RECEIVED (must be ARRIVED)", currentState)
		}

		// Read items
		itemStmt := spanner.Statement{
			SQL:    `SELECT ItemId, ProductId, Quantity, VolumeVU FROM InternalTransferItems WHERE TransferId = @tid`,
			Params: map[string]interface{}{"tid": transferID},
		}
		itemIter := txn.Query(ctx, itemStmt)
		defer itemIter.Stop()

		var mutations []*spanner.Mutation

		for {
			itemRow, err := itemIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var item TransferItemDetail
			if err := itemRow.Columns(&item.ItemId, &item.ProductId, &item.Quantity, &item.VolumeVU); err != nil {
				return err
			}
			items = append(items, item)

			// Increment stock: read current → update with added quantity
			stockRow, stockErr := txn.ReadRow(ctx, "SupplierInventory",
				spanner.Key{item.ProductId}, []string{"QuantityAvailable"})
			var currentQty int64
			if stockErr == nil {
				_ = stockRow.Columns(&currentQty)
			}
			mutations = append(mutations, spanner.InsertOrUpdate("SupplierInventory",
				[]string{"ProductId", "SupplierId", "WarehouseId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{item.ProductId, supplierID, whID, currentQty + item.Quantity, spanner.CommitTimestamp},
			))
		}

		// Update transfer state → RECEIVED
		mutations = append(mutations, spanner.Update("InternalTransferOrders",
			[]string{"TransferId", "State", "UpdatedAt"},
			[]interface{}{transferID, "RECEIVED", spanner.CommitTimestamp},
		))
		if writeErr := txn.BufferWrite(mutations); writeErr != nil {
			return writeErr
		}

		evt := map[string]interface{}{
			"event":        internalKafka.EventTransferReceived,
			"transfer_id":  transferID,
			"factory_id":   factoryID,
			"warehouse_id": whID,
			"supplier_id":  supplierID,
			"items_count":  len(items),
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		return outbox.EmitJSON(txn, "InternalTransferOrder", transferID,
			internalKafka.EventTransferReceived, internalKafka.TopicMain, evt,
			telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		log.Printf("[TRANSFERS] receive error: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "RECEIVED",
		"transfer_id":   transferID,
		"items_updated": len(items),
	})
}

// HandleCreateTransfer — POST /v1/factory/transfers — Create a new internal transfer order.
// Used by replenishment engine or manual factory admin action.
func (s *TransferService) HandleCreateTransfer(w http.ResponseWriter, r *http.Request) {
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
		FactoryId   string `json:"factory_id"`
		WarehouseId string `json:"warehouse_id"`
		Source      string `json:"source"` // MANUAL_EMERGENCY | SYSTEM_THRESHOLD | SYSTEM_PREDICTED
		Items       []struct {
			ProductId string  `json:"product_id"`
			Quantity  int64   `json:"quantity"`
			VolumeVU  float64 `json:"volume_vu"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.FactoryId == "" || req.WarehouseId == "" || len(req.Items) == 0 {
		http.Error(w, `{"error":"factory_id, warehouse_id, and items are required"}`, http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		req.Source = "MANUAL_EMERGENCY"
	}

	// Resolve supplier from factory
	fRow, err := s.Spanner.Single().ReadRow(r.Context(), "Factories",
		spanner.Key{req.FactoryId}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}
	var supplierID string
	if err := fRow.Columns(&supplierID); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	transferID := uuid.New().String()
	var totalVolumeVU float64
	mutations := []*spanner.Mutation{
		spanner.Insert("InternalTransferOrders",
			[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State", "TotalVolumeVU", "Source", "CreatedAt"},
			[]interface{}{transferID, req.FactoryId, req.WarehouseId, supplierID, "DRAFT", 0.0, req.Source, spanner.CommitTimestamp},
		),
	}

	for _, item := range req.Items {
		itemID := uuid.New().String()
		totalVolumeVU += item.VolumeVU
		mutations = append(mutations, spanner.Insert("InternalTransferItems",
			[]string{"TransferId", "ItemId", "ProductId", "Quantity", "VolumeVU"},
			[]interface{}{transferID, itemID, item.ProductId, item.Quantity, item.VolumeVU},
		))
	}

	// Update total volume on the header
	mutations[0] = spanner.Insert("InternalTransferOrders",
		[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State", "TotalVolumeVU", "Source", "CreatedAt"},
		[]interface{}{transferID, req.FactoryId, req.WarehouseId, supplierID, "DRAFT", totalVolumeVU, req.Source, spanner.CommitTimestamp},
	)

	if _, err := s.Spanner.Apply(r.Context(), mutations); err != nil {
		log.Printf("[TRANSFERS] create error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer_id":     transferID,
		"factory_id":      req.FactoryId,
		"warehouse_id":    req.WarehouseId,
		"state":           "DRAFT",
		"total_volume_vu": totalVolumeVU,
		"items_count":     len(req.Items),
	})
}
