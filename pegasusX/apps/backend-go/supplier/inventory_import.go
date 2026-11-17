package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	inventoryImportMaxBytes = supplierImportMaxUploadSize
)

var errInventoryImportInvalidCSV = errors.New("inventory_import_invalid_csv")

type inventoryImportRow struct {
	ProductID        string
	WarehouseID      string
	QuantityOnHand   int64
	ReorderThreshold int64
}

type inventoryImportResult struct {
	SessionID string   `json:"session_id,omitempty"`
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
	if !enforceImportMutationAuth(w, r) {
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

	fileName := strings.TrimSpace(r.Header.Get("X-File-Name"))
	if fileName == "" {
		fileName = strings.TrimSpace(r.URL.Query().Get("file_name"))
	}

	rows, err := parseInventoryImportArtifact(fileName, body)
	if err != nil {
		if errors.Is(err, parser.ErrPasswordProtectedWorkbook) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error":   "password_protected_workbook",
				"message": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "inventory_import_empty"})
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
	staging := make([]inventoryImportStagingEntry, 0, len(rows))
	productValidationCache := make(map[string]error)
	for i, row := range rows {
		raw := inventoryImportStagingRaw(row)
		entry := inventoryImportStagingEntry{
			RowIndex: int64(i + 1),
			RawData:  raw,
		}
		if _, ok := warehouseIDs[row.WarehouseID]; !ok {
			entry.ValidationErrors = []string{"warehouse_not_in_topology"}
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: warehouse_not_in_topology", i+2))
			staging = append(staging, entry)
			continue
		}
		valErr, cached := productValidationCache[row.ProductID]
		if !cached {
			valErr = s.validateImportedProduct(r.Context(), sid, row.ProductID)
			productValidationCache[row.ProductID] = valErr
		}
		if valErr != nil {
			entry.ValidationErrors = []string{valErr.Error()}
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", i+2, valErr.Error()))
			staging = append(staging, entry)
			continue
		}
		if err := s.upsertImportedInventoryRow(r.Context(), sid, row); err != nil {
			entry.ValidationErrors = []string{err.Error()}
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %s", i+2, err.Error()))
			staging = append(staging, entry)
			continue
		}
		entry.CleanedData = raw
		result.Applied++
		staging = append(staging, entry)
	}
	if sessionID, err := s.persistInventoryImportStagingChunked(r.Context(), sid, staging, result.Applied, result.Skipped); err != nil {
		s.log.WarnContext(r.Context(), "inventory import staging persist failed", "err", err, "supplier_id", sid)
	} else if sessionID != "" {
		result.SessionID = sessionID
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

func (s *Service) validateImportedProduct(ctx context.Context, supplierID, productID string) error {
	if s.portalSpanner == nil {
		return nil
	}
	row, err := s.portalSpanner.Single().ReadRow(ctx, "Products", spanner.Key{productID}, []string{"SupplierId", "IsActive"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return s.validateImportedProductInventoryFallback(ctx, supplierID, productID)
		}
		return fmt.Errorf("load product: %w", err)
	}
	var productSupplier string
	var isActive bool
	if err := row.Columns(&productSupplier, &isActive); err != nil {
		return fmt.Errorf("decode product: %w", err)
	}
	if strings.TrimSpace(productSupplier) != strings.TrimSpace(supplierID) {
		return errors.New("product_supplier_mismatch")
	}
	if !isActive {
		return errors.New("product_inactive")
	}
	return nil
}

func (s *Service) validateImportedProductInventoryFallback(ctx context.Context, supplierID, productID string) error {
	stmt := spanner.Statement{
		SQL: `SELECT 1
		      FROM SupplierInventoryV2
		      WHERE SupplierId = @supplierId AND ProductId = @productId
		      LIMIT 1`,
		Params: map[string]any{
			"supplierId": supplierID,
			"productId":  productID,
		},
	}
	iter := s.portalSpanner.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return errors.New("product_not_found")
	}
	if err != nil {
		return fmt.Errorf("load inventory product: %w", err)
	}
	return nil
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
	return parseInventoryImportArtifact("", body)
}

func parseInventoryImportArtifact(fileName string, body []byte) ([]inventoryImportRow, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: empty import body", errInventoryImportInvalidCSV)
	}
	sniff, err := parser.SniffArtifact(fileName, body)
	if err != nil {
		return nil, err
	}

	rows := make([]inventoryImportRow, 0, 256)
	var headerChecked bool
	collectChunk := func(_ string, streamHeaders []string, chunk []parser.ParsedRow) error {
		index := mapInventoryImportHeader(streamHeaders)
		for _, col := range []string{"product_id", "warehouse_id", "quantity_on_hand"} {
			if _, ok := index[col]; !ok {
				return fmt.Errorf("%w: missing column %s", errInventoryImportInvalidCSV, col)
			}
		}
		headerChecked = true
		for _, pr := range chunk {
			row, rowErr := parseInventoryImportParsedRow(pr.Fields, streamHeaders, index)
			if rowErr != nil {
				return rowErr
			}
			if row.ProductID == "" && row.WarehouseID == "" {
				continue
			}
			rows = append(rows, row)
		}
		return nil
	}

	baseStream := strings.TrimSuffix(filepath.Base(strings.TrimSpace(fileName)), filepath.Ext(strings.TrimSpace(fileName)))
	var streams []parser.ParsedStream
	switch sniff.Format {
	case parser.FormatXLSX:
		streams, err = parser.StreamXLSX(body, collectChunk)
	case parser.FormatXLS:
		streams, err = parser.StreamXLS(body, collectChunk)
	case parser.FormatCSV, parser.FormatTSV:
		var st parser.ParsedStream
		st, err = parser.StreamDelimited(baseStream, body, "", collectChunk)
		if err == nil {
			streams = []parser.ParsedStream{st}
		}
	case parser.FormatCommerceML:
		streams, err = parser.StreamCommerceML(body, collectChunk)
	case parser.FormatJSON:
		streams, err = parser.StreamJSON(baseStream, body, collectChunk)
	default:
		return nil, fmt.Errorf("%w: unsupported format %s", parser.ErrFileFormatMismatch, sniff.Format)
	}
	if err != nil {
		if errors.Is(err, parser.ErrPasswordProtectedWorkbook) ||
			errors.Is(err, parser.ErrFileFormatMismatch) ||
			errors.Is(err, parser.ErrMalformedArtifact) ||
			errors.Is(err, errInventoryImportInvalidCSV) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", errInventoryImportInvalidCSV, err)
	}
	if !headerChecked && len(streams) > 0 {
		index := mapInventoryImportHeader(streams[0].Headers)
		for _, col := range []string{"product_id", "warehouse_id", "quantity_on_hand"} {
			if _, ok := index[col]; !ok {
				return nil, fmt.Errorf("%w: missing column %s", errInventoryImportInvalidCSV, col)
			}
		}
	}
	return rows, nil
}

func parseInventoryImportParsedRow(fields map[string]any, headers []string, index map[string]int) (inventoryImportRow, error) {
	read := func(key string) string {
		pos, ok := index[key]
		if !ok || pos < 0 || pos >= len(headers) {
			return ""
		}
		val, exists := fields[headers[pos]]
		if !exists || val == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(val))
	}
	productID := read("product_id")
	warehouseID := read("warehouse_id")
	if productID == "" || warehouseID == "" {
		return inventoryImportRow{}, fmt.Errorf("%w: product_id and warehouse_id required", errInventoryImportInvalidCSV)
	}
	qtyRaw := strings.ReplaceAll(read("quantity_on_hand"), ",", "")
	qty, err := strconv.ParseInt(qtyRaw, 10, 64)
	if err != nil || qty < 0 {
		return inventoryImportRow{}, fmt.Errorf("%w: invalid quantity_on_hand", errInventoryImportInvalidCSV)
	}
	threshold := int64(0)
	if raw := strings.ReplaceAll(read("reorder_threshold"), ",", ""); raw != "" {
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

func (s *Service) persistInventoryImportStagingChunked(
	ctx context.Context,
	supplierID string,
	entries []inventoryImportStagingEntry,
	applied, skipped int,
) (string, error) {
	const stageBatchSize = 2500
	if len(entries) <= stageBatchSize {
		return s.persistInventoryImportStaging(ctx, supplierID, entries, applied, skipped)
	}
	if s.portalSpanner == nil {
		return "", nil
	}
	sessionID := uuid.NewString()
	summaryJSON, err := json.Marshal(inventoryImportStagingSummary{
		Applied: applied,
		Skipped: skipped,
	})
	if err != nil {
		return "", fmt.Errorf("marshal import staging summary: %w", err)
	}
	if _, err := s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "file_name", "total_rows", "error_summary", "created_at", "updated_at"},
				[]any{
					supplierID, sessionID, "APPLIED", inventoryImportStagingFileName, int64(len(entries)),
					spanner.NullJSON{Value: json.RawMessage(summaryJSON), Valid: true},
					spanner.CommitTimestamp, spanner.CommitTimestamp,
				},
			),
		})
	}); err != nil {
		return "", fmt.Errorf("persist inventory import session header: %w", err)
	}

	for start := 0; start < len(entries); start += stageBatchSize {
		end := start + stageBatchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[start:end]
		if _, err := s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			mutations := make([]*spanner.Mutation, 0, len(batch))
			for _, entry := range batch {
				rawJSON, rErr := inventoryImportToNullJSON(entry.RawData)
				if rErr != nil {
					return fmt.Errorf("raw_data row %d: %w", entry.RowIndex, rErr)
				}
				cleanedJSON, cErr := inventoryImportToNullJSON(entry.CleanedData)
				if cErr != nil {
					return fmt.Errorf("cleaned_data row %d: %w", entry.RowIndex, cErr)
				}
				mutations = append(mutations, spanner.InsertOrUpdate(
					"SupplierImportStagedRows",
					[]string{
						"supplier_id", "session_id", "row_index", "raw_data", "cleaned_data",
						"validation_errors", "is_new_product", "created_at", "updated_at",
					},
					[]any{
						supplierID, sessionID, entry.RowIndex, rawJSON, cleanedJSON,
						entry.ValidationErrors, false, spanner.CommitTimestamp, spanner.CommitTimestamp,
					},
				))
			}
			return txn.BufferWrite(mutations)
		}); err != nil {
			return "", fmt.Errorf("persist inventory import staged rows batch: %w", err)
		}
	}
	return sessionID, nil
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
