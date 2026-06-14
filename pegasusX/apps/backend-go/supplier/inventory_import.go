package supplier

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	inventoryImportMaxBytes = 2 * 1024 * 1024
	inventoryImportMaxRows  = 1000
)

var errInventoryImportInvalidCSV = errors.New("inventory_import_invalid_csv")

type inventoryImportRow struct {
	ProductID        string
	WarehouseID      string
	QuantityOnHand   int64
	ReorderThreshold int64
}

type inventoryImportResult struct {
	Applied   int      `json:"applied"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
	UpdatedAt string   `json:"updated_at"`
}

// HandleInventoryImport serves POST /v1/supplier/inventory/import.
func (s *Service) HandleInventoryImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	body, ok := readMutationBody(w, r, inventoryImportMaxBytes)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	rows, err := parseInventoryImportCSV(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "inventory_import_empty"})
		return
	}
	if len(rows) > inventoryImportMaxRows {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "inventory_import_too_many_rows"})
		return
	}
	if s.inventorySvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory_unavailable"})
		return
	}

	sid := s.scopedSupplierID(r)
	topology, err := s.repo.GetTopology(r.Context(), sid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
		return
	}
	warehouseIDs := warehouseIDSet(topology)

	result := inventoryImportResult{UpdatedAt: s.now().UTC().Format(time.RFC3339Nano)}
	for i, row := range rows {
		if _, ok := warehouseIDs[row.WarehouseID]; !ok {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: warehouse_not_in_topology", i+2))
			continue
		}
		if err := s.upsertImportedInventoryRow(r.Context(), sid, row); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", i+2, err.Error()))
			continue
		}
		result.Applied++
	}
	if s.cache != nil {
		s.cache.Invalidate(r.Context(), "supplier:inventory:"+sid)
	}
	respBytes, _ := json.Marshal(result)
	s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

func (s *Service) upsertImportedInventoryRow(ctx context.Context, supplierID string, row inventoryImportRow) error {
	if s.inventorySvc == nil {
		return errors.New("inventory_unavailable")
	}
	inventoryID := uuid.NewString()
	existingID, found, err := s.inventorySvc.FindByWarehouseProduct(ctx, row.WarehouseID, row.ProductID)
	if err != nil {
		return err
	}
	if found {
		inventoryID = existingID
	}
	return s.inventorySvc.UpsertLevel(ctx, InventoryLevelUpsert{
		InventoryID:      inventoryID,
		ProductID:        row.ProductID,
		WarehouseID:      row.WarehouseID,
		SupplierID:       supplierID,
		QuantityOnHand:   row.QuantityOnHand,
		QuantityReserved: 0,
		ReorderThreshold: row.ReorderThreshold,
		Version:          1,
	})
}

func warehouseIDSet(topology SupplierTopology) map[string]struct{} {
	set := make(map[string]struct{}, len(topology.Warehouses))
	for _, warehouse := range topology.Warehouses {
		id := strings.TrimSpace(warehouse.WarehouseID)
		if id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

func parseInventoryImportCSV(body []byte) ([]inventoryImportRow, error) {
	reader := csv.NewReader(bytes.NewReader(body))
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInventoryImportInvalidCSV, err)
	}
	index := mapInventoryImportHeader(header)
	required := []string{"product_id", "warehouse_id", "quantity_on_hand"}
	for _, col := range required {
		if _, ok := index[col]; !ok {
			return nil, fmt.Errorf("%w: missing column %s", errInventoryImportInvalidCSV, col)
		}
	}
	rows := make([]inventoryImportRow, 0)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errInventoryImportInvalidCSV, err)
		}
		row, err := parseInventoryImportRecord(record, index)
		if err != nil {
			return nil, err
		}
		if row.ProductID == "" && row.WarehouseID == "" {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func mapInventoryImportHeader(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for i, col := range header {
		key := strings.ToLower(strings.TrimSpace(col))
		switch key {
		case "sku_id", "sku", "product_id", "productid":
			index["product_id"] = i
		case "warehouse_id", "warehouseid", "warehouse":
			index["warehouse_id"] = i
		case "quantity", "quantity_on_hand", "qty", "on_hand":
			index["quantity_on_hand"] = i
		case "reorder_threshold", "threshold", "reorder":
			index["reorder_threshold"] = i
		default:
			index[key] = i
		}
	}
	return index
}

func parseInventoryImportRecord(record []string, index map[string]int) (inventoryImportRow, error) {
	read := func(key string) string {
		pos, ok := index[key]
		if !ok || pos >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[pos])
	}
	productID := read("product_id")
	warehouseID := read("warehouse_id")
	if productID == "" || warehouseID == "" {
		return inventoryImportRow{}, fmt.Errorf("%w: product_id and warehouse_id required", errInventoryImportInvalidCSV)
	}
	qty, err := strconv.ParseInt(read("quantity_on_hand"), 10, 64)
	if err != nil || qty < 0 {
		return inventoryImportRow{}, fmt.Errorf("%w: invalid quantity_on_hand", errInventoryImportInvalidCSV)
	}
	threshold := int64(0)
	if raw := read("reorder_threshold"); raw != "" {
		threshold, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || threshold < 0 {
			return inventoryImportRow{}, fmt.Errorf("%w: invalid reorder_threshold", errInventoryImportInvalidCSV)
		}
	}
	return inventoryImportRow{
		ProductID:        productID,
		WarehouseID:      warehouseID,
		QuantityOnHand:   qty,
		ReorderThreshold: threshold,
	}, nil
}
