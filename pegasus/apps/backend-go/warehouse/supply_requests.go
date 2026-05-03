package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"
	warehousews "backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Supply Request State Machine ──────────────────────────────────────────────
//
//  DRAFT → SUBMITTED → ACKNOWLEDGED → IN_PRODUCTION → READY → FULFILLED
//    ↓        ↓            ↓
//  CANCELLED  CANCELLED  CANCELLED
//

var validSupplyRequestTransitions = map[string][]string{
	"DRAFT":         {"SUBMITTED", "CANCELLED"},
	"SUBMITTED":     {"ACKNOWLEDGED", "CANCELLED"},
	"ACKNOWLEDGED":  {"IN_PRODUCTION", "CANCELLED"},
	"IN_PRODUCTION": {"READY"},
	"READY":         {"FULFILLED"},
}

func isValidSupplyTransition(from, to string) bool {
	for _, v := range validSupplyRequestTransitions[from] {
		if v == to {
			return true
		}
	}
	return false
}

// SupplyRequestService handles supply request CRUD and state transitions.
type SupplyRequestService struct {
	Spanner      *spanner.Client
	Producer     *kafka.Writer
	WarehouseHub *warehousews.WarehouseHub
}

// ── Request/Response Shapes ───────────────────────────────────────────────────

type SupplyRequestItem struct {
	ProductID         string  `json:"product_id"`
	RequestedQuantity int64   `json:"requested_quantity"`
	RecommendedQty    int64   `json:"recommended_qty,omitempty"`
	UnitVolumeVU      float64 `json:"unit_volume_vu,omitempty"`
}

type CreateSupplyRequestBody struct {
	FactoryID             string              `json:"factory_id"`
	Priority              string              `json:"priority,omitempty"`
	RequestedDeliveryDate string              `json:"requested_delivery_date,omitempty"`
	Notes                 string              `json:"notes,omitempty"`
	Items                 []SupplyRequestItem `json:"items"`
	UseDemandForecast     bool                `json:"use_demand_forecast"` // auto-fill from demand engine
}

type SupplyRequestResponse struct {
	RequestID             string                      `json:"request_id"`
	WarehouseID           string                      `json:"warehouse_id"`
	FactoryID             string                      `json:"factory_id"`
	SupplierID            string                      `json:"supplier_id"`
	State                 string                      `json:"state"`
	Priority              string                      `json:"priority"`
	RequestedDeliveryDate *time.Time                  `json:"requested_delivery_date,omitempty"`
	TotalVolumeVU         float64                     `json:"total_volume_vu"`
	Notes                 string                      `json:"notes,omitempty"`
	DemandBreakdown       json.RawMessage             `json:"demand_breakdown,omitempty"`
	TransferOrderID       string                      `json:"transfer_order_id,omitempty"`
	Items                 []SupplyRequestItemResponse `json:"items,omitempty"`
	CreatedBy             string                      `json:"created_by"`
	CreatedAt             time.Time                   `json:"created_at"`
	UpdatedAt             *time.Time                  `json:"updated_at,omitempty"`
}

type SupplyRequestItemResponse struct {
	ItemID            string  `json:"item_id"`
	ProductID         string  `json:"product_id"`
	RequestedQuantity int64   `json:"requested_quantity"`
	RecommendedQty    int64   `json:"recommended_qty"`
	UnitVolumeVU      float64 `json:"unit_volume_vu"`
}

// HandleCreateSupplyRequest creates a new supply request from warehouse to factory.
// POST /v1/warehouse/supply-requests
func (s *SupplyRequestService) HandleCreateSupplyRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	warehouseID := claims.WarehouseID
	if warehouseID == "" {
		warehouseID = r.URL.Query().Get("warehouse_id")
	}
	if warehouseID == "" {
		http.Error(w, `{"error":"warehouse_id required"}`, http.StatusBadRequest)
		return
	}

	var req CreateSupplyRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if req.FactoryID == "" {
		http.Error(w, `{"error":"factory_id is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Items) == 0 && !req.UseDemandForecast {
		http.Error(w, `{"error":"items are required unless use_demand_forecast is true"}`, http.StatusBadRequest)
		return
	}

	// Resolve supplier from warehouse
	supplierID, _, _, _, err := resolveWarehouseSupplier(r.Context(), s.Spanner, warehouseID)
	if err != nil {
		http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
		return
	}

	// Validate that the target factory belongs to the same supplier
	if err := validateEntityOwnership(r.Context(), s.Spanner, "Factories", "FactoryId", req.FactoryID, supplierID); err != nil {
		http.Error(w, `{"error":"factory_id does not belong to your organization"}`, http.StatusForbidden)
		return
	}

	// Validate that the target factory belongs to the same supplier
	if err := validateEntityOwnership(r.Context(), s.Spanner, "Factories", "FactoryId", req.FactoryID, supplierID); err != nil {
		http.Error(w, `{"error":"factory_id does not belong to your organization"}`, http.StatusForbidden)
		return
	}

	// If using demand forecast, auto-fill items from the demand engine
	var demandJSON json.RawMessage
	if req.UseDemandForecast {
		products, err := computeDemandForecast(r.Context(), s.Spanner, supplierID, warehouseID, 7)
		if err != nil {
			log.Printf("[SUPPLY REQUEST] demand forecast error: %v", err)
			http.Error(w, "Failed to compute demand forecast", http.StatusInternalServerError)
			return
		}
		req.Items = make([]SupplyRequestItem, 0, len(products))
		for _, p := range products {
			if p.RecommendedQty > 0 {
				req.Items = append(req.Items, SupplyRequestItem{
					ProductID:         p.ProductID,
					RequestedQuantity: p.RecommendedQty,
					RecommendedQty:    p.RecommendedQty,
					UnitVolumeVU:      p.UnitVolumeVU,
				})
			}
		}
		breakdownData, _ := json.Marshal(products)
		demandJSON = breakdownData
	}

	if len(req.Items) == 0 {
		http.Error(w, `{"error":"no items with positive recommended quantity"}`, http.StatusBadRequest)
		return
	}

	priority := "NORMAL"
	if req.Priority == "URGENT" || req.Priority == "CRITICAL" {
		priority = req.Priority
	}

	var deliveryDate *time.Time
	if req.RequestedDeliveryDate != "" {
		t, err := time.Parse(time.RFC3339, req.RequestedDeliveryDate)
		if err != nil {
			http.Error(w, `{"error":"invalid requested_delivery_date format (RFC3339)"}`, http.StatusBadRequest)
			return
		}
		deliveryDate = &t
	}

	requestID := uuid.New().String()
	var totalVolumeVU float64

	mutations := make([]*spanner.Mutation, 0, len(req.Items)+1)

	// Create main supply request row
	cols := []string{"RequestId", "WarehouseId", "FactoryId", "SupplierId", "State", "Priority",
		"TotalVolumeVU", "Notes", "CreatedBy", "CreatedAt", "UpdatedAt"}
	vals := []interface{}{requestID, warehouseID, req.FactoryID, supplierID, "SUBMITTED", priority,
		float64(0), req.Notes, claims.UserID, spanner.CommitTimestamp, spanner.CommitTimestamp}

	if deliveryDate != nil {
		cols = append(cols, "RequestedDeliveryDate")
		vals = append(vals, *deliveryDate)
	}
	if len(demandJSON) > 0 {
		cols = append(cols, "DemandBreakdown")
		vals = append(vals, string(demandJSON))
	}

	mutations = append(mutations, spanner.Insert("SupplyRequests", cols, vals))

	// Create item rows
	for _, item := range req.Items {
		itemID := uuid.New().String()
		volume := item.UnitVolumeVU * float64(item.RequestedQuantity)
		totalVolumeVU += volume

		mutations = append(mutations, spanner.Insert("SupplyRequestItems",
			[]string{"RequestId", "ItemId", "ProductId", "RequestedQuantity", "RecommendedQuantity", "UnitVolumeVU", "CreatedAt"},
			[]interface{}{requestID, itemID, item.ProductID, item.RequestedQuantity, item.RecommendedQty, item.UnitVolumeVU, spanner.CommitTimestamp},
		))
	}

	// Update total volume
	mutations = append(mutations, spanner.Update("SupplyRequests",
		[]string{"RequestId", "TotalVolumeVU"},
		[]interface{}{requestID, totalVolumeVU},
	))

	if _, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		event := internalKafka.SupplyRequestEvent{
			RequestID:   requestID,
			WarehouseID: warehouseID,
			FactoryID:   req.FactoryID,
			SupplierID:  supplierID,
			State:       "SUBMITTED",
			Priority:    priority,
			Timestamp:   time.Now().UTC(),
		}
		return outbox.EmitJSON(txn, "SupplyRequest", requestID, internalKafka.EventSupplyRequestSubmitted, internalKafka.TopicMain, event, telemetry.TraceIDFromContext(ctx))
	}); err != nil {
		log.Printf("[SUPPLY REQUEST] spanner insert error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if s.WarehouseHub != nil {
		s.WarehouseHub.BroadcastSupplyRequestUpdate(warehouseID, requestID, "SUBMITTED")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request_id":      requestID,
		"state":           "SUBMITTED",
		"priority":        priority,
		"total_volume_vu": totalVolumeVU,
		"items_count":     len(req.Items),
	})
}

// HandleListSupplyRequests returns supply requests for a warehouse or factory.
// GET /v1/warehouse/supply-requests?state=SUBMITTED&factory_id=X
// GET /v1/factory/supply-requests?state=SUBMITTED
func (s *SupplyRequestService) HandleListSupplyRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var sql string
	params := map[string]interface{}{}

	switch claims.Role {
	case "WAREHOUSE":
		warehouseID := claims.WarehouseID
		if warehouseID == "" {
			warehouseID = r.URL.Query().Get("warehouse_id")
		}
		sql = `SELECT RequestId, WarehouseId, FactoryId, SupplierId, State, Priority,
		              RequestedDeliveryDate, TotalVolumeVU, Notes, TransferOrderId,
		              CreatedBy, CreatedAt, UpdatedAt
		       FROM SupplyRequests WHERE WarehouseId = @scope`
		params["scope"] = warehouseID
	case "FACTORY":
		factoryScope := auth.GetFactoryScope(r.Context())
		if factoryScope == nil {
			http.Error(w, "Factory scope required", http.StatusForbidden)
			return
		}
		sql = `SELECT RequestId, WarehouseId, FactoryId, SupplierId, State, Priority,
		              RequestedDeliveryDate, TotalVolumeVU, Notes, TransferOrderId,
		              CreatedBy, CreatedAt, UpdatedAt
		       FROM SupplyRequests WHERE FactoryId = @scope`
		params["scope"] = factoryScope.FactoryID
	default:
		// SUPPLIER / ADMIN — show all for the supplier
		sql = `SELECT RequestId, WarehouseId, FactoryId, SupplierId, State, Priority,
		              RequestedDeliveryDate, TotalVolumeVU, Notes, TransferOrderId,
		              CreatedBy, CreatedAt, UpdatedAt
		       FROM SupplyRequests WHERE SupplierId = @scope`
		params["scope"] = claims.UserID
	}

	if state := r.URL.Query().Get("state"); state != "" {
		sql += " AND State = @state"
		params["state"] = state
	}
	if factoryID := r.URL.Query().Get("factory_id"); factoryID != "" && claims.Role != "FACTORY" {
		sql += " AND FactoryId = @factoryId"
		params["factoryId"] = factoryID
	}

	sql += " ORDER BY CreatedAt DESC LIMIT 100"

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Spanner.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var results []SupplyRequestResponse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[SUPPLY REQUEST] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var resp SupplyRequestResponse
		var reqDelivery spanner.NullTime
		var notes, transferID spanner.NullString
		var updatedAt spanner.NullTime
		if err := row.Columns(&resp.RequestID, &resp.WarehouseID, &resp.FactoryID,
			&resp.SupplierID, &resp.State, &resp.Priority,
			&reqDelivery, &resp.TotalVolumeVU, &notes, &transferID,
			&resp.CreatedBy, &resp.CreatedAt, &updatedAt); err != nil {
			log.Printf("[SUPPLY REQUEST] row parse error: %v", err)
			continue
		}
		if reqDelivery.Valid {
			resp.RequestedDeliveryDate = &reqDelivery.Time
		}
		if notes.Valid {
			resp.Notes = notes.StringVal
		}
		if transferID.Valid {
			resp.TransferOrderID = transferID.StringVal
		}
		if updatedAt.Valid {
			resp.UpdatedAt = &updatedAt.Time
		}
		results = append(results, resp)
	}

	if results == nil {
		results = []SupplyRequestResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleSupplyRequestDetail returns a single supply request with items.
// GET /v1/warehouse/supply-requests/{id}
// GET /v1/factory/supply-requests/{id}
func (s *SupplyRequestService) HandleSupplyRequestDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/supply-requests/")
	requestID = strings.TrimPrefix(requestID, "/v1/factory/supply-requests/")
	if requestID == "" {
		http.Error(w, `{"error":"request_id required in path"}`, http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Read supply request
	row, err := s.Spanner.Single().ReadRow(r.Context(), "SupplyRequests",
		spanner.Key{requestID},
		[]string{"RequestId", "WarehouseId", "FactoryId", "SupplierId", "State", "Priority",
			"RequestedDeliveryDate", "TotalVolumeVU", "Notes", "DemandBreakdown",
			"TransferOrderId", "CreatedBy", "CreatedAt", "UpdatedAt"})
	if err != nil {
		http.Error(w, `{"error":"supply request not found"}`, http.StatusNotFound)
		return
	}

	var resp SupplyRequestResponse
	var reqDelivery spanner.NullTime
	var notes, demandJSON, transferID spanner.NullString
	var updatedAt spanner.NullTime
	if err := row.Columns(&resp.RequestID, &resp.WarehouseID, &resp.FactoryID,
		&resp.SupplierID, &resp.State, &resp.Priority,
		&reqDelivery, &resp.TotalVolumeVU, &notes, &demandJSON,
		&transferID, &resp.CreatedBy, &resp.CreatedAt, &updatedAt); err != nil {
		log.Printf("[SUPPLY REQUEST] detail parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if reqDelivery.Valid {
		resp.RequestedDeliveryDate = &reqDelivery.Time
	}
	if notes.Valid {
		resp.Notes = notes.StringVal
	}
	if demandJSON.Valid {
		resp.DemandBreakdown = json.RawMessage(demandJSON.StringVal)
	}
	if transferID.Valid {
		resp.TransferOrderID = transferID.StringVal
	}
	if updatedAt.Valid {
		resp.UpdatedAt = &updatedAt.Time
	}

	// Read items
	itemStmt := spanner.Statement{
		SQL: `SELECT ItemId, ProductId, RequestedQuantity, RecommendedQuantity, UnitVolumeVU
		      FROM SupplyRequestItems WHERE RequestId = @rid`,
		Params: map[string]interface{}{"rid": requestID},
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
		var item SupplyRequestItemResponse
		if err := itemRow.Columns(&item.ItemID, &item.ProductID, &item.RequestedQuantity,
			&item.RecommendedQty, &item.UnitVolumeVU); err != nil {
			continue
		}
		resp.Items = append(resp.Items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleSupplyRequestTransition transitions a supply request state.
// PATCH /v1/warehouse/supply-requests/{id} → { "action": "CANCEL" }
// PATCH /v1/factory/supply-requests/{id} → { "action": "ACKNOWLEDGE" | "START_PRODUCTION" | "MARK_READY" | "FULFILL" }
func (s *SupplyRequestService) HandleSupplyRequestTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := strings.TrimPrefix(r.URL.Path, "/v1/warehouse/supply-requests/")
	requestID = strings.TrimPrefix(requestID, "/v1/factory/supply-requests/")
	if requestID == "" {
		http.Error(w, `{"error":"request_id required in path"}`, http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Action          string `json:"action"`
		TransferOrderID string `json:"transfer_order_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	actionMap := map[string]string{
		"CANCEL":           "CANCELLED",
		"ACKNOWLEDGE":      "ACKNOWLEDGED",
		"START_PRODUCTION": "IN_PRODUCTION",
		"MARK_READY":       "READY",
		"FULFILL":          "FULFILLED",
	}
	newState, ok := actionMap[req.Action]
	if !ok {
		http.Error(w, fmt.Sprintf(`{"error":"invalid action '%s'. valid: CANCEL, ACKNOWLEDGE, START_PRODUCTION, MARK_READY, FULFILL"}`, req.Action), http.StatusBadRequest)
		return
	}

	var warehouseID string
	_, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplyRequests",
			spanner.Key{requestID}, []string{"State", "WarehouseId", "FactoryId", "SupplierId"})
		if err != nil {
			return fmt.Errorf("supply request not found")
		}

		var currentState, factoryID, supplierID string
		if err := row.Columns(&currentState, &warehouseID, &factoryID, &supplierID); err != nil {
			return err
		}

		if !isValidSupplyTransition(currentState, newState) {
			return fmt.Errorf("invalid transition: %s → %s", currentState, newState)
		}

		cols := []string{"RequestId", "State", "UpdatedAt"}
		vals := []interface{}{requestID, newState, spanner.CommitTimestamp}

		if req.TransferOrderID != "" && newState == "READY" {
			cols = append(cols, "TransferOrderId")
			vals = append(vals, req.TransferOrderID)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("SupplyRequests", cols, vals),
		}); err != nil {
			return err
		}

		eventType := supplyRequestEventTypeForState(newState)
		if eventType == "" {
			return nil
		}

		event := internalKafka.SupplyRequestEvent{
			RequestID:   requestID,
			WarehouseID: warehouseID,
			FactoryID:   factoryID,
			SupplierID:  supplierID,
			State:       newState,
			Priority:    "",
			Timestamp:   time.Now().UTC(),
		}
		return outbox.EmitJSON(txn, "SupplyRequest", requestID, eventType, internalKafka.TopicMain, event, telemetry.TraceIDFromContext(ctx))

	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusNotFound)
		} else if strings.Contains(errMsg, "invalid transition") {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusConflict)
		} else {
			log.Printf("[SUPPLY REQUEST] transition error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	if s.WarehouseHub != nil && warehouseID != "" {
		s.WarehouseHub.BroadcastSupplyRequestUpdate(warehouseID, requestID, newState)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"request_id": requestID,
		"state":      newState,
	})
}

func supplyRequestEventTypeForState(state string) string {
	switch state {
	case "SUBMITTED":
		return internalKafka.EventSupplyRequestSubmitted
	case "ACKNOWLEDGED":
		return internalKafka.EventSupplyRequestAcknowledged
	case "READY":
		return internalKafka.EventSupplyRequestReady
	case "FULFILLED":
		return internalKafka.EventSupplyRequestFulfilled
	case "CANCELLED":
		return internalKafka.EventSupplyRequestCancelled
	default:
		return ""
	}
}
