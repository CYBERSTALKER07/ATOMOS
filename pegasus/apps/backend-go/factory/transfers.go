package factory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
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
	ID                     string               `json:"id"`
	TransferId             string               `json:"transfer_id"`
	FactoryId              string               `json:"factory_id"`
	SourceFactoryId        string               `json:"source_factory_id"`
	WarehouseId            string               `json:"warehouse_id"`
	DestinationWarehouseId string               `json:"destination_warehouse_id"`
	WarehouseName          string               `json:"warehouse_name"`
	SupplierId             string               `json:"supplier_id"`
	State                  string               `json:"state"`
	Priority               string               `json:"priority"`
	TotalItems             int64                `json:"total_items"`
	TotalVolumeL           float64              `json:"total_volume_l"`
	TotalVolumeM3          float64              `json:"total_volume_m3"`
	TotalVolumeVU          float64              `json:"total_volume_vu"`
	ManifestId             string               `json:"manifest_id,omitempty"`
	Source                 string               `json:"source"`
	Notes                  string               `json:"notes"`
	CreatedAt              string               `json:"created_at"`
	UpdatedAt              string               `json:"updated_at"`
	Items                  []TransferItemDetail `json:"items,omitempty"`
}

type TransferItemDetail struct {
	ID                string  `json:"id"`
	ItemId            string  `json:"item_id"`
	ProductId         string  `json:"product_id"`
	SkuId             string  `json:"sku_id"`
	ProductName       string  `json:"product_name"`
	Quantity          int64   `json:"quantity"`
	QuantityAvailable int64   `json:"quantity_available"`
	UnitVolumeL       float64 `json:"unit_volume_l"`
	VolumeM3          float64 `json:"volume_m3"`
	VolumeVU          float64 `json:"volume_vu"`
}

type TransferListResponse struct {
	Transfers []TransferResponse `json:"transfers"`
	Total     int                `json:"total"`
	Data      []TransferResponse `json:"data,omitempty"`
}

type approveTransferResponse struct {
	TransferResponse
	ConvoyCount int      `json:"convoy_count"`
	ManifestIDs []string `json:"manifest_ids"`
	VolumeVU    float64  `json:"volume_vu"`
}

type transferHeader struct {
	TransferId    string
	FactoryId     string
	WarehouseId   string
	SupplierId    string
	State         string
	TotalVolumeVU float64
	ManifestId    string
	Source        string
	CreatedAt     time.Time
	UpdatedAt     spanner.NullTime
}

type transferTransitionRequest struct {
	Action      string `json:"action"`
	TargetState string `json:"target_state"`
}

// TransferService holds dependencies for transfer endpoints.
type TransferService struct {
	Spanner  *spanner.Client
	Cache    *cache.Cache
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

	stateFilters := parseTransferStateFilters(r)
	limit := parseTransferLimit(r.URL.Query().Get("limit"))

	headers, err := s.listTransferHeaders(r.Context(), factoryID, stateFilters, limit)
	if err != nil {
		log.Printf("[TRANSFERS] list error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	transfers, err := s.buildTransferListResponses(r.Context(), headers)
	if err != nil {
		log.Printf("[TRANSFERS] list enrich error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TransferListResponse{
		Transfers: transfers,
		Total:     len(transfers),
		Data:      transfers,
	})
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

	transfer, err := s.loadTransferResponse(r.Context(), factoryID, transferID)
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			http.Error(w, `{"error":"transfer not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("[TRANSFERS] detail error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transfer)
}

func parseTransferStateFilters(r *http.Request) []string {
	stateFilters := make([]string, 0, 4)
	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		stateFilters = append(stateFilters, strings.ToUpper(state))
	}
	if states := strings.TrimSpace(r.URL.Query().Get("states")); states != "" {
		for _, state := range strings.Split(states, ",") {
			normalized := strings.ToUpper(strings.TrimSpace(state))
			if normalized != "" {
				stateFilters = append(stateFilters, normalized)
			}
		}
	}
	return stateFilters
}

func parseTransferLimit(raw string) int {
	if raw == "" {
		return 100
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 100
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizedTransferAction(action string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(action), "_", "-"))
}

func actionForTargetState(targetState string) string {
	switch strings.ToUpper(strings.TrimSpace(targetState)) {
	case "APPROVED":
		return "approve"
	case "LOADING":
		return "start-loading"
	case "DISPATCHED":
		return "dispatch"
	case "IN_TRANSIT":
		return "in-transit"
	case "ARRIVED":
		return "arrive"
	case "CANCELLED":
		return "cancel"
	default:
		return ""
	}
}

func transferPriority(source string) string {
	switch source {
	case "MANUAL_EMERGENCY", "SYSTEM_THRESHOLD":
		return "HIGH"
	case "SYSTEM_PREDICTED":
		return "MEDIUM"
	default:
		return "STANDARD"
	}
}

func transferUpdatedAt(createdAt time.Time, updatedAt spanner.NullTime) string {
	if updatedAt.Valid {
		return updatedAt.Time.Format(time.RFC3339)
	}
	return createdAt.Format(time.RFC3339)
}

func buildTransferResponse(header transferHeader, warehouseName string, totalItems int64, items []TransferItemDetail) TransferResponse {
	return TransferResponse{
		ID:                     header.TransferId,
		TransferId:             header.TransferId,
		FactoryId:              header.FactoryId,
		SourceFactoryId:        header.FactoryId,
		WarehouseId:            header.WarehouseId,
		DestinationWarehouseId: header.WarehouseId,
		WarehouseName:          warehouseName,
		SupplierId:             header.SupplierId,
		State:                  header.State,
		Priority:               transferPriority(header.Source),
		TotalItems:             totalItems,
		TotalVolumeL:           header.TotalVolumeVU,
		TotalVolumeM3:          header.TotalVolumeVU,
		TotalVolumeVU:          header.TotalVolumeVU,
		ManifestId:             header.ManifestId,
		Source:                 header.Source,
		Notes:                  "",
		CreatedAt:              header.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              transferUpdatedAt(header.CreatedAt, header.UpdatedAt),
		Items:                  items,
	}
}

func (s *TransferService) listTransferHeaders(ctx context.Context, factoryID string, stateFilters []string, limit int) ([]transferHeader, error) {
	sql := `SELECT TransferId, FactoryId, WarehouseId, SupplierId, State,
	               TotalVolumeVU, COALESCE(ManifestId, ''), Source, CreatedAt, UpdatedAt
	        FROM InternalTransferOrders WHERE FactoryId = @fid`
	params := map[string]interface{}{"fid": factoryID}

	if len(stateFilters) == 1 {
		sql += " AND State = @state"
		params["state"] = stateFilters[0]
	} else if len(stateFilters) > 1 {
		sql += " AND State IN UNNEST(@states)"
		params["states"] = stateFilters
	}

	sql += fmt.Sprintf(" ORDER BY COALESCE(UpdatedAt, CreatedAt) DESC LIMIT %d", limit)
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	headers := make([]transferHeader, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var header transferHeader
		if err := row.Columns(
			&header.TransferId,
			&header.FactoryId,
			&header.WarehouseId,
			&header.SupplierId,
			&header.State,
			&header.TotalVolumeVU,
			&header.ManifestId,
			&header.Source,
			&header.CreatedAt,
			&header.UpdatedAt,
		); err != nil {
			return nil, err
		}
		headers = append(headers, header)
	}

	return headers, nil
}

func (s *TransferService) loadTransferHeader(ctx context.Context, factoryID, transferID string) (transferHeader, error) {
	stmt := spanner.Statement{
		SQL: `SELECT TransferId, FactoryId, WarehouseId, SupplierId, State,
		             TotalVolumeVU, COALESCE(ManifestId, ''), Source, CreatedAt, UpdatedAt
		      FROM InternalTransferOrders WHERE TransferId = @tid AND FactoryId = @fid`,
		Params: map[string]interface{}{"tid": transferID, "fid": factoryID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return transferHeader{}, spanner.ErrRowNotFound
		}
		return transferHeader{}, err
	}

	var header transferHeader
	if err := row.Columns(
		&header.TransferId,
		&header.FactoryId,
		&header.WarehouseId,
		&header.SupplierId,
		&header.State,
		&header.TotalVolumeVU,
		&header.ManifestId,
		&header.Source,
		&header.CreatedAt,
		&header.UpdatedAt,
	); err != nil {
		return transferHeader{}, err
	}

	return header, nil
}

func (s *TransferService) fetchWarehouseNames(ctx context.Context, warehouseIDs []string) (map[string]string, error) {
	if len(warehouseIDs) == 0 {
		return map[string]string{}, nil
	}

	stmt := spanner.Statement{
		SQL:    `SELECT WarehouseId, Name FROM Warehouses WHERE WarehouseId IN UNNEST(@warehouse_ids)`,
		Params: map[string]interface{}{"warehouse_ids": warehouseIDs},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	names := make(map[string]string, len(warehouseIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var warehouseID, warehouseName string
		if err := row.Columns(&warehouseID, &warehouseName); err != nil {
			return nil, err
		}
		names[warehouseID] = warehouseName
	}

	return names, nil
}

func (s *TransferService) fetchProductSuppliersBySku(ctx context.Context, skuIDs []string) (map[string]string, error) {
	if len(skuIDs) == 0 {
		return map[string]string{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT SkuId, SupplierId
		      FROM SupplierProducts
		      WHERE SkuId IN UNNEST(@sku_ids)`,
		Params: map[string]interface{}{"sku_ids": skuIDs},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	suppliers := make(map[string]string, len(skuIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var skuID string
		var supplierID spanner.NullString
		if err := row.Columns(&skuID, &supplierID); err != nil {
			return nil, err
		}
		if supplierID.Valid && supplierID.StringVal != "" {
			suppliers[skuID] = supplierID.StringVal
		}
	}

	return suppliers, nil
}

func (s *TransferService) fetchTransferItemTotals(ctx context.Context, transferIDs []string) (map[string]int64, error) {
	if len(transferIDs) == 0 {
		return map[string]int64{}, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT TransferId, SUM(Quantity)
		      FROM InternalTransferItems
		      WHERE TransferId IN UNNEST(@transfer_ids)
		      GROUP BY TransferId`,
		Params: map[string]interface{}{"transfer_ids": transferIDs},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	totals := make(map[string]int64, len(transferIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var transferID string
		var totalItems spanner.NullInt64
		if err := row.Columns(&transferID, &totalItems); err != nil {
			return nil, err
		}
		if totalItems.Valid {
			totals[transferID] = totalItems.Int64
		}
	}

	return totals, nil
}

func (s *TransferService) buildTransferListResponses(ctx context.Context, headers []transferHeader) ([]TransferResponse, error) {
	if len(headers) == 0 {
		return []TransferResponse{}, nil
	}

	transferIDs := make([]string, 0, len(headers))
	warehouseIDs := make([]string, 0, len(headers))
	for _, header := range headers {
		transferIDs = append(transferIDs, header.TransferId)
		warehouseIDs = append(warehouseIDs, header.WarehouseId)
	}

	warehouseNames, err := s.fetchWarehouseNames(ctx, warehouseIDs)
	if err != nil {
		return nil, err
	}
	totalItems, err := s.fetchTransferItemTotals(ctx, transferIDs)
	if err != nil {
		return nil, err
	}

	transfers := make([]TransferResponse, 0, len(headers))
	for _, header := range headers {
		transfers = append(transfers, buildTransferResponse(
			header,
			warehouseNames[header.WarehouseId],
			totalItems[header.TransferId],
			nil,
		))
	}

	return transfers, nil
}

func (s *TransferService) fetchProductNames(ctx context.Context, productIDs []string) (map[string]string, error) {
	if len(productIDs) == 0 {
		return map[string]string{}, nil
	}

	stmt := spanner.Statement{
		SQL:    `SELECT ProductId, Name FROM Products WHERE ProductId IN UNNEST(@product_ids)`,
		Params: map[string]interface{}{"product_ids": productIDs},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	names := make(map[string]string, len(productIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID, productName string
		if err := row.Columns(&productID, &productName); err != nil {
			return nil, err
		}
		names[productID] = productName
	}

	return names, nil
}

func (s *TransferService) loadTransferItems(ctx context.Context, transferID string) ([]TransferItemDetail, int64, error) {
	itemStmt := spanner.Statement{
		SQL:    `SELECT ItemId, ProductId, Quantity, VolumeVU FROM InternalTransferItems WHERE TransferId = @tid`,
		Params: map[string]interface{}{"tid": transferID},
	}
	itemIter := s.Spanner.Single().Query(ctx, itemStmt)
	defer itemIter.Stop()

	type itemRow struct {
		ItemId    string
		ProductId string
		Quantity  int64
		VolumeVU  float64
	}

	rows := make([]itemRow, 0, 8)
	productIDs := make([]string, 0, 8)
	var totalItems int64

	for {
		itemRecord, err := itemIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		var row itemRow
		if err := itemRecord.Columns(&row.ItemId, &row.ProductId, &row.Quantity, &row.VolumeVU); err != nil {
			return nil, 0, err
		}

		totalItems += row.Quantity
		productIDs = append(productIDs, row.ProductId)
		rows = append(rows, row)
	}

	productNames, err := s.fetchProductNames(ctx, productIDs)
	if err != nil {
		return nil, 0, err
	}

	items := make([]TransferItemDetail, 0, len(rows))
	for _, row := range rows {
		items = append(items, TransferItemDetail{
			ID:                row.ItemId,
			ItemId:            row.ItemId,
			ProductId:         row.ProductId,
			SkuId:             row.ProductId,
			ProductName:       productNames[row.ProductId],
			Quantity:          row.Quantity,
			QuantityAvailable: row.Quantity,
			UnitVolumeL:       row.VolumeVU,
			VolumeM3:          row.VolumeVU,
			VolumeVU:          row.VolumeVU,
		})
	}

	return items, totalItems, nil
}

func (s *TransferService) loadTransferResponse(ctx context.Context, factoryID, transferID string) (TransferResponse, error) {
	header, err := s.loadTransferHeader(ctx, factoryID, transferID)
	if err != nil {
		return TransferResponse{}, err
	}

	warehouseNames, err := s.fetchWarehouseNames(ctx, []string{header.WarehouseId})
	if err != nil {
		return TransferResponse{}, err
	}

	items, totalItems, err := s.loadTransferItems(ctx, transferID)
	if err != nil {
		return TransferResponse{}, err
	}

	return buildTransferResponse(header, warehouseNames[header.WarehouseId], totalItems, items), nil
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
	action := normalizedTransferAction(parts[1])

	if action == "transition" {
		var req transferTransitionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		switch {
		case req.TargetState != "":
			action = actionForTargetState(req.TargetState)
		case req.Action != "":
			action = normalizedTransferAction(req.Action)
		default:
			http.Error(w, `{"error":"transition action required"}`, http.StatusBadRequest)
			return
		}
	}

	targetState := ""
	switch action {
	case "accept", "approve":
		compatRequest := r.Clone(r.Context())
		compatURL := *r.URL
		compatURL.Path = fmt.Sprintf("/v1/factory/transfers/%s/approve", transferID)
		compatRequest.URL = &compatURL
		s.HandleApproveTransfer(w, compatRequest)
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

	var supplierID, warehouseID string

	err := func() error {
		_, txErr := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, readErr := txn.ReadRow(ctx, "InternalTransferOrders",
				spanner.Key{transferID}, []string{"State", "FactoryId", "SupplierId", "WarehouseId"})
			if readErr != nil {
				return readErr
			}

			var currentState, rowFactoryID string
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

	if s.Cache != nil {
		s.Cache.Invalidate(r.Context(),
			cache.PrefixFactoryProfile+factoryID,
			cache.PrefixWarehouseDetail+warehouseID,
		)
	}

	transfer, loadErr := s.loadTransferResponse(r.Context(), factoryID, transferID)
	if loadErr != nil {
		log.Printf("[TRANSFERS] transition response load error: %v", loadErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transfer)
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
	if _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	}); err != nil {
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
			if _, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite(mutations)
			}); err != nil {
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

	if s.Cache != nil {
		s.Cache.Invalidate(ctx,
			cache.PrefixFactoryProfile+factoryID,
			cache.PrefixWarehouseDetail+warehouseID,
		)
	}

	transfer, loadErr := s.loadTransferResponse(ctx, factoryID, transferID)
	if loadErr != nil {
		log.Printf("[TRANSFERS] approve response load error: %v", loadErr)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approveTransferResponse{
		TransferResponse: transfer,
		ConvoyCount:      convoyCount,
		ManifestIDs:      manifestIDs,
		VolumeVU:         totalVolumeVU,
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

	if s.Cache != nil {
		s.Cache.Invalidate(r.Context(),
			cache.PrefixFactoryProfile+factoryID,
			cache.PrefixWarehouseDetail+whID,
		)
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

	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
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
	if req.WarehouseId == "" || len(req.Items) == 0 {
		http.Error(w, `{"error":"warehouse_id and items are required"}`, http.StatusBadRequest)
		return
	}
	if req.Source == "" {
		req.Source = "MANUAL_EMERGENCY"
	}

	factoryID := req.FactoryId
	if claims.Role == "FACTORY" {
		factoryScope := auth.GetFactoryScope(r.Context())
		if factoryScope == nil || factoryScope.FactoryID == "" {
			http.Error(w, `{"error":"factory scope required"}`, http.StatusForbidden)
			return
		}
		if req.FactoryId != "" && req.FactoryId != factoryScope.FactoryID {
			http.Error(w, `{"error":"factory_id must match authenticated factory scope"}`, http.StatusForbidden)
			return
		}
		factoryID = factoryScope.FactoryID
	}
	if factoryID == "" {
		http.Error(w, `{"error":"factory_id is required"}`, http.StatusBadRequest)
		return
	}

	// Resolve supplier from factory and enforce caller ownership.
	fRow, err := s.Spanner.Single().ReadRow(r.Context(), "Factories",
		spanner.Key{factoryID}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"factory not found"}`, http.StatusNotFound)
		return
	}
	var supplierID string
	if err := fRow.Columns(&supplierID); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if claims.Role != "FACTORY" && supplierID != claims.ResolveSupplierID() {
		http.Error(w, `{"error":"factory does not belong to your organization"}`, http.StatusForbidden)
		return
	}

	whRow, err := s.Spanner.Single().ReadRow(r.Context(), "Warehouses",
		spanner.Key{req.WarehouseId}, []string{"SupplierId"})
	if err != nil {
		http.Error(w, `{"error":"warehouse not found"}`, http.StatusNotFound)
		return
	}
	var warehouseSupplierID string
	if err := whRow.Columns(&warehouseSupplierID); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if warehouseSupplierID != supplierID {
		http.Error(w, `{"error":"warehouse does not belong to your organization"}`, http.StatusForbidden)
		return
	}

	uniqueSkuIDs := make([]string, 0, len(req.Items))
	seenSku := make(map[string]struct{}, len(req.Items))
	for _, item := range req.Items {
		skuID := strings.TrimSpace(item.ProductId)
		if skuID == "" {
			http.Error(w, `{"error":"product_id is required for all items"}`, http.StatusBadRequest)
			return
		}
		if _, exists := seenSku[skuID]; exists {
			continue
		}
		seenSku[skuID] = struct{}{}
		uniqueSkuIDs = append(uniqueSkuIDs, skuID)
	}

	productSuppliers, err := s.fetchProductSuppliersBySku(r.Context(), uniqueSkuIDs)
	if err != nil {
		log.Printf("[TRANSFERS] failed to validate product ownership: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for _, item := range req.Items {
		if productSupplierID, ok := productSuppliers[item.ProductId]; ok && productSupplierID != supplierID {
			log.Printf("[TRANSFERS] cross-tenant ProductId injection blocked: product=%s belongs to %s, caller supplier=%s",
				item.ProductId, productSupplierID, supplierID)
			http.Error(w, `{"error":"product does not belong to your organization"}`, http.StatusForbidden)
			return
		}
	}

	transferID := uuid.New().String()
	var totalVolumeVU float64
	mutations := []*spanner.Mutation{
		spanner.Insert("InternalTransferOrders",
			[]string{"TransferId", "FactoryId", "WarehouseId", "SupplierId", "State", "TotalVolumeVU", "Source", "CreatedAt"},
			[]interface{}{transferID, factoryID, req.WarehouseId, supplierID, "DRAFT", 0.0, req.Source, spanner.CommitTimestamp},
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
		[]interface{}{transferID, factoryID, req.WarehouseId, supplierID, "DRAFT", totalVolumeVU, req.Source, spanner.CommitTimestamp},
	)

	if _, err := s.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		evt := struct {
			Event       string  `json:"event"`
			TransferID  string  `json:"transfer_id"`
			FactoryID   string  `json:"factory_id"`
			WarehouseID string  `json:"warehouse_id"`
			SupplierID  string  `json:"supplier_id"`
			Source      string  `json:"source"`
			VolumeVU    float64 `json:"total_volume_vu"`
			ItemsCount  int     `json:"items_count"`
		}{
			Event:       internalKafka.EventReplenishmentTransferCreated,
			TransferID:  transferID,
			FactoryID:   factoryID,
			WarehouseID: req.WarehouseId,
			SupplierID:  supplierID,
			Source:      req.Source,
			VolumeVU:    totalVolumeVU,
			ItemsCount:  len(req.Items),
		}

		return outbox.EmitJSON(txn, "InternalTransferOrder", transferID, internalKafka.EventReplenishmentTransferCreated, internalKafka.TopicMain, evt, telemetry.TraceIDFromContext(ctx))
	}); err != nil {
		log.Printf("[TRANSFERS] create error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if s.Cache != nil {
		s.Cache.Invalidate(r.Context(),
			cache.PrefixFactoryProfile+factoryID,
			cache.PrefixWarehouseDetail+req.WarehouseId,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transfer_id":     transferID,
		"factory_id":      factoryID,
		"warehouse_id":    req.WarehouseId,
		"state":           "DRAFT",
		"total_volume_vu": totalVolumeVU,
		"source":          req.Source,
		"items_count":     len(req.Items),
	})
}
