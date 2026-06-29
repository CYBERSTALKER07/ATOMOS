package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	supplierImportRoutePrefix   = "/v1/supplier/inventory/imports"
	supplierImportSessionPath   = "/v1/supplier/inventory/imports/{id}"
	supplierImportUploadedPath  = "/v1/supplier/inventory/imports/{id}/uploaded"
	supplierImportIngestPath    = "/v1/supplier/inventory/imports/{id}/ingest"
	supplierImportRowsPath      = "/v1/supplier/inventory/imports/{id}/rows"
	supplierImportMappingPath   = "/v1/supplier/inventory/imports/{id}/mapping"
	supplierImportApprovePath   = "/v1/supplier/inventory/imports/{id}/approve"
	supplierImportApplyPath     = "/v1/supplier/inventory/imports/{id}/apply"
	supplierImportMaxUploadSize = int64(50 * 1024 * 1024)
	supplierImportDefaultLimit  = 100
	supplierImportMaxLimit      = 1000
	supplierImportMutationBatch = 5000
	supplierImportReasonBulk    = "BULK_IMPORT"
)

var supplierImportAllowedUploadExtensions = map[string]string{
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xls":  "application/vnd.ms-excel",
	"csv":  "text/csv",
	"tsv":  "text/tab-separated-values",
}

var supplierImportAllowedStatuses = map[string]struct{}{
	"INITIALIZED":      {},
	"UPLOADED":         {},
	"DISCOVERING":      {},
	"DISCOVERED":       {},
	"MAPPING_REQUIRED": {},
	"APPROVED":         {},
	"APPLYING":         {},
	"APPLIED":          {},
	"FAILED":           {},
}

var supplierImportTransitions = map[string]map[string]struct{}{
	"INITIALIZED": {
		"UPLOADED": {},
		"FAILED":   {},
	},
	"UPLOADED": {
		"DISCOVERING":      {},
		"DISCOVERED":       {},
		"MAPPING_REQUIRED": {},
		"FAILED":           {},
	},
	"DISCOVERING": {
		"DISCOVERED":       {},
		"MAPPING_REQUIRED": {},
		"FAILED":           {},
	},
	"DISCOVERED": {
		"MAPPING_REQUIRED": {},
		"APPROVED":         {},
		"FAILED":           {},
	},
	"MAPPING_REQUIRED": {
		"APPROVED": {},
		"FAILED":   {},
	},
	"APPROVED": {
		"APPLYING": {},
		"APPLIED":  {},
		"FAILED":   {},
	},
	"APPLYING": {
		"APPLIED": {},
		"FAILED":  {},
	},
	"APPLIED": {},
	"FAILED":  {},
}

var (
	errImportSessionNotFound  = errors.New("supplier import session not found")
	errImportInvalidStatus    = errors.New("invalid supplier import status")
	errImportStateConflict    = errors.New("supplier import status transition conflict")
	errImportAccessDenied     = errors.New("supplier import access denied")
	errImportAlreadyApplied   = errors.New("supplier import session already applied")
	errImportNoApplicableRows = errors.New("supplier import session has no applicable rows")
)

// ImportSessionRecord mirrors SupplierImportSessions with spanner tags.
type ImportSessionRecord struct {
	SupplierID       string           `spanner:"supplier_id" json:"supplier_id"`
	SessionID        string           `spanner:"session_id" json:"session_id"`
	Status           string           `spanner:"status" json:"status"`
	FileName         string           `spanner:"file_name" json:"file_name"`
	TotalRows        int64            `spanner:"total_rows" json:"total_rows"`
	ErrorSummaryJSON spanner.NullJSON `spanner:"error_summary" json:"-"`
	ErrorSummary     json.RawMessage  `spanner:"-" json:"error_summary,omitempty"`
	CreatedAt        time.Time        `spanner:"created_at" json:"created_at"`
	UpdatedAt        spanner.NullTime `spanner:"updated_at" json:"-"`
	UpdatedAtRFC3339 string           `spanner:"-" json:"updated_at,omitempty"`
}

// ImportStagedRowRecord mirrors SupplierImportStagedRows with spanner tags.
type ImportStagedRowRecord struct {
	SupplierID       string           `spanner:"supplier_id" json:"supplier_id"`
	SessionID        string           `spanner:"session_id" json:"session_id"`
	RowIndex         int64            `spanner:"row_index" json:"row_index"`
	RawDataJSON      spanner.NullJSON `spanner:"raw_data" json:"-"`
	CleanedDataJSON  spanner.NullJSON `spanner:"cleaned_data" json:"-"`
	RawData          json.RawMessage  `spanner:"-" json:"raw_data,omitempty"`
	CleanedData      json.RawMessage  `spanner:"-" json:"cleaned_data,omitempty"`
	ValidationErrors []string         `spanner:"validation_errors" json:"validation_errors,omitempty"`
	IsNewProduct     bool             `spanner:"is_new_product" json:"is_new_product"`
	CreatedAt        time.Time        `spanner:"created_at" json:"created_at,omitempty"`
	UpdatedAt        spanner.NullTime `spanner:"updated_at" json:"-"`
	UpdatedAtRFC3339 string           `spanner:"-" json:"updated_at,omitempty"`
}

// ImportMappingRecord mirrors SupplierImportMapping with spanner tags.
type ImportMappingRecord struct {
	SupplierID   string           `spanner:"supplier_id" json:"supplier_id"`
	SessionID    string           `spanner:"session_id" json:"session_id"`
	MappingJSON  spanner.NullJSON `spanner:"mapping_json" json:"-"`
	Mapping      json.RawMessage  `spanner:"-" json:"mapping_json,omitempty"`
	CreatedAt    time.Time        `spanner:"created_at" json:"created_at,omitempty"`
	UpdatedAt    spanner.NullTime `spanner:"updated_at" json:"-"`
	UpdatedAtRFC string           `spanner:"-" json:"updated_at,omitempty"`
}

// ImportApplySummary captures the terminal summary of sandbox apply.
type ImportApplySummary struct {
	SessionID          string   `json:"session_id"`
	Status             string   `json:"status"`
	AppliedRows        int64    `json:"applied_rows"`
	CreatedProducts    int64    `json:"created_products"`
	AffectedWarehouses int64    `json:"affected_warehouses"`
	WarehouseIDs       []string `json:"warehouse_ids,omitempty"`
	AppliedProductIDs  []string `json:"product_ids,omitempty"`
	Timestamp          string   `json:"timestamp,omitempty"`
	Idempotent         bool     `json:"idempotent"`
}

// ImportRepository encapsulates supplier-scoped import sandbox persistence.
type ImportRepository struct {
	client *spanner.Client
}

// NewImportRepository builds a repository over SupplierImportSessions tables.
func NewImportRepository(client *spanner.Client) *ImportRepository {
	return &ImportRepository{client: client}
}

// ImportRoutesDeps wires import session routes.
type ImportRoutesDeps struct {
	Spanner      *spanner.Client
	Service      *Service
	SupplierHub  *ws.Hub
	WarehouseHub *ws.Hub
}

// RegisterImportRoutes mounts the supplier inventory import session wizard.
func RegisterImportRoutes(r chi.Router, d ImportRoutesDeps) {
	if d.Spanner == nil {
		return
	}
	repo := NewImportRepository(d.Spanner)

	r.Route(supplierImportRoutePrefix, func(imports chi.Router) {
		imports.Post("/", handleCreateImportSession(repo, d.Service))
		imports.Get("/{id}", handleGetImportSession(repo))
		imports.Post("/{id}/uploaded", handlePostImportUploaded(repo))
		imports.Post("/{id}/ingest", handlePostImportIngest(repo, d.Service))
		imports.Get("/{id}/rows", handleGetImportRows(repo))
		imports.Get("/{id}/mapping", handleGetImportMapping(repo))
		imports.Post("/{id}/mapping", handlePostImportMapping(repo))
		imports.Post("/{id}/approve", handlePostImportApprove(repo, d.Service))
		imports.Post("/{id}/apply", handlePostImportApply(repo, d.Service, d.SupplierHub, d.WarehouseHub))
	})
}

func (r *ImportRepository) CreateImportSession(ctx context.Context, supplierID, sessionID, fileName, initialStatus string) (ImportSessionRecord, error) {
	if r.client == nil {
		return ImportSessionRecord{}, errors.New("spanner unavailable")
	}
	status := normalizeImportStatus(initialStatus)
	if _, ok := supplierImportAllowedStatuses[status]; !ok {
		return ImportSessionRecord{}, errImportInvalidStatus
	}
	if _, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.Insert(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "file_name", "total_rows", "error_summary", "created_at", "updated_at"},
			[]any{supplierID, sessionID, status, fileName, int64(0), nil, spanner.CommitTimestamp, spanner.CommitTimestamp},
		)})
	}); err != nil {
		return ImportSessionRecord{}, err
	}
	return r.GetSession(ctx, supplierID, sessionID)
}

func (r *ImportRepository) UpdateSessionStatus(ctx context.Context, supplierID, sessionID, nextStatus string) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	nextStatus = normalizeImportStatus(nextStatus)
	if _, ok := supplierImportAllowedStatuses[nextStatus]; !ok {
		return errImportInvalidStatus
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}
		var currentStatus string
		if err := row.Columns(&currentStatus); err != nil {
			return err
		}
		currentStatus = normalizeImportStatus(currentStatus)
		if currentStatus == nextStatus {
			return nil
		}
		nextSet, ok := supplierImportTransitions[currentStatus]
		if !ok {
			return errImportStateConflict
		}
		if _, allowed := nextSet[nextStatus]; !allowed {
			return errImportStateConflict
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "updated_at"},
			[]any{supplierID, sessionID, nextStatus, spanner.CommitTimestamp},
		)})
	})
	return err
}

func (r *ImportRepository) MarkSessionUploadedAndEmit(ctx context.Context, supplierID, sessionID, gcsPath string) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}

		var currentStatus string
		if err := row.Columns(&currentStatus); err != nil {
			return err
		}
		currentStatus = normalizeImportStatus(currentStatus)
		if currentStatus == "UPLOADED" {
			return nil
		}
		if currentStatus != "INITIALIZED" {
			return errImportStateConflict
		}

		if err := txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "updated_at"},
			[]any{supplierID, sessionID, "UPLOADED", spanner.CommitTimestamp},
		)}); err != nil {
			return err
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, "SupplierImportSession", sessionID, events.TopicInventoryImportEvents, events.InventoryImportEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventInventoryImportUploaded, Version: 1},
			SessionID:  sessionID,
			SupplierID: supplierID,
			GCSPath:    gcsPath,
		}); err != nil {
			return err
		}

		mutations := importOutboxMutations(buf)
		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	return err
}

func importOutboxMutations(buf *spannerTxnBuffer) []*spanner.Mutation {
	if buf == nil || len(buf.events) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(buf.events))
	for _, e := range buf.events {
		createdAt := e.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		row := map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}
		if e.PublishedAt != nil {
			row["PublishedAt"] = e.PublishedAt.UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
	}
	return mutations
}

func (r *ImportRepository) SaveStagedRows(ctx context.Context, supplierID, sessionID string, rows []ImportStagedRowRecord) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	if len(rows) == 0 {
		return nil
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sessionRow, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"total_rows"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}
		var existingTotal int64
		if err := sessionRow.Columns(&existingTotal); err != nil {
			return err
		}

		maxRow := existingTotal
		mutations := make([]*spanner.Mutation, 0, len(rows)+1)
		for _, row := range rows {
			if row.RowIndex+1 > maxRow {
				maxRow = row.RowIndex + 1
			}
			rawJSON, err := importToNullJSON(row.RawData)
			if err != nil {
				return fmt.Errorf("parse raw_data row %d: %w", row.RowIndex, err)
			}
			cleanedJSON, err := importToNullJSON(row.CleanedData)
			if err != nil {
				return fmt.Errorf("parse cleaned_data row %d: %w", row.RowIndex, err)
			}

			mutations = append(mutations, spanner.InsertOrUpdate(
				"SupplierImportStagedRows",
				[]string{"supplier_id", "session_id", "row_index", "raw_data", "cleaned_data", "validation_errors", "is_new_product", "created_at", "updated_at"},
				[]any{supplierID, sessionID, row.RowIndex, rawJSON, cleanedJSON, row.ValidationErrors, row.IsNewProduct, spanner.CommitTimestamp, spanner.CommitTimestamp},
			))
		}

		mutations = append(mutations, spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "total_rows", "updated_at"},
			[]any{supplierID, sessionID, maxRow, spanner.CommitTimestamp},
		))
		return txn.BufferWrite(mutations)
	})
	return err
}

func (r *ImportRepository) SaveMapping(ctx context.Context, supplierID, sessionID string, mapping json.RawMessage) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	mappingJSON, err := importToNullJSON(mapping)
	if err != nil {
		return err
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"session_id"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate(
				"SupplierImportMapping",
				[]string{"supplier_id", "session_id", "mapping_json", "created_at", "updated_at"},
				[]any{supplierID, sessionID, mappingJSON, spanner.CommitTimestamp, spanner.CommitTimestamp},
			),
			spanner.Update(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "updated_at"},
				[]any{supplierID, sessionID, "MAPPING_REQUIRED", spanner.CommitTimestamp},
			),
		})
	})
	return err
}

func (r *ImportRepository) PersistDiscovery(ctx context.Context, supplierID, sessionID, status string, mappingDoc, errorSummary map[string]any) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	mappingBytes, err := json.Marshal(mappingDoc)
	if err != nil {
		return err
	}
	errorBytes, err := json.Marshal(errorSummary)
	if err != nil {
		return err
	}
	mappingJSON, err := importToNullJSON(mappingBytes)
	if err != nil {
		return err
	}
	errorJSON, err := importToNullJSON(errorBytes)
	if err != nil {
		return err
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errImportSessionNotFound
			}
			return readErr
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Update(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
				[]any{supplierID, sessionID, status, errorJSON, spanner.CommitTimestamp},
			),
			spanner.InsertOrUpdate(
				"SupplierImportMapping",
				[]string{"supplier_id", "session_id", "mapping_json", "created_at", "updated_at"},
				[]any{supplierID, sessionID, mappingJSON, spanner.CommitTimestamp, spanner.CommitTimestamp},
			),
		})
	})
	return err
}

func (r *ImportRepository) GetSession(ctx context.Context, supplierID, sessionID string) (ImportSessionRecord, error) {
	if r.client == nil {
		return ImportSessionRecord{}, errors.New("spanner unavailable")
	}

	row, err := r.client.Single().ReadRow(
		ctx,
		"SupplierImportSessions",
		spanner.Key{supplierID, sessionID},
		[]string{"supplier_id", "session_id", "status", "file_name", "total_rows", "error_summary", "created_at", "updated_at"},
	)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ImportSessionRecord{}, errImportSessionNotFound
		}
		return ImportSessionRecord{}, err
	}

	var record ImportSessionRecord
	if err := row.Columns(
		&record.SupplierID,
		&record.SessionID,
		&record.Status,
		&record.FileName,
		&record.TotalRows,
		&record.ErrorSummaryJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return ImportSessionRecord{}, err
	}

	if record.ErrorSummaryJSON.Valid {
		encoded, marshalErr := json.Marshal(record.ErrorSummaryJSON.Value)
		if marshalErr != nil {
			return ImportSessionRecord{}, marshalErr
		}
		record.ErrorSummary = encoded
	}
	record.CreatedAt = record.CreatedAt.UTC()
	if record.UpdatedAt.Valid {
		record.UpdatedAtRFC3339 = record.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}
	return record, nil
}

func (r *ImportRepository) GetMapping(ctx context.Context, supplierID, sessionID string) (ImportMappingRecord, error) {
	if r.client == nil {
		return ImportMappingRecord{}, errors.New("spanner unavailable")
	}

	if _, err := r.GetSession(ctx, supplierID, sessionID); err != nil {
		return ImportMappingRecord{}, err
	}

	row, err := r.client.Single().ReadRow(
		ctx,
		"SupplierImportMapping",
		spanner.Key{supplierID, sessionID},
		[]string{"supplier_id", "session_id", "mapping_json", "created_at", "updated_at"},
	)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ImportMappingRecord{SupplierID: supplierID, SessionID: sessionID}, nil
		}
		return ImportMappingRecord{}, err
	}

	var record ImportMappingRecord
	if err := row.Columns(
		&record.SupplierID,
		&record.SessionID,
		&record.MappingJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return ImportMappingRecord{}, err
	}

	if record.MappingJSON.Valid {
		encoded, marshalErr := json.Marshal(record.MappingJSON.Value)
		if marshalErr != nil {
			return ImportMappingRecord{}, marshalErr
		}
		record.Mapping = encoded
	}
	record.CreatedAt = record.CreatedAt.UTC()
	if record.UpdatedAt.Valid {
		record.UpdatedAtRFC = record.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}
	return record, nil
}

func (r *ImportRepository) ListRows(ctx context.Context, supplierID, sessionID string, limit, offset int) ([]ImportStagedRowRecord, bool, error) {
	if r.client == nil {
		return nil, false, errors.New("spanner unavailable")
	}

	if _, err := r.GetSession(ctx, supplierID, sessionID); err != nil {
		return nil, false, err
	}

	if limit <= 0 {
		limit = supplierImportDefaultLimit
	}
	if limit > supplierImportMaxLimit {
		limit = supplierImportMaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	stmt := spanner.Statement{
		SQL: `SELECT supplier_id, session_id, row_index, raw_data, cleaned_data, validation_errors, is_new_product, created_at, updated_at
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId AND session_id = @sessionId
		      ORDER BY row_index
		      LIMIT @limit OFFSET @offset`,
		Params: map[string]any{
			"supplierId": supplierID,
			"sessionId":  sessionID,
			"limit":      int64(limit + 1),
			"offset":     int64(offset),
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]ImportStagedRowRecord, 0, limit+1)
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, false, err
		}

		var record ImportStagedRowRecord
		if err := row.Columns(
			&record.SupplierID,
			&record.SessionID,
			&record.RowIndex,
			&record.RawDataJSON,
			&record.CleanedDataJSON,
			&record.ValidationErrors,
			&record.IsNewProduct,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, false, err
		}

		if record.RawDataJSON.Valid {
			encoded, marshalErr := json.Marshal(record.RawDataJSON.Value)
			if marshalErr != nil {
				return nil, false, marshalErr
			}
			record.RawData = encoded
		}
		if record.CleanedDataJSON.Valid {
			encoded, marshalErr := json.Marshal(record.CleanedDataJSON.Value)
			if marshalErr != nil {
				return nil, false, marshalErr
			}
			record.CleanedData = encoded
		}
		record.CreatedAt = record.CreatedAt.UTC()
		if record.UpdatedAt.Valid {
			record.UpdatedAtRFC3339 = record.UpdatedAt.Time.UTC().Format(time.RFC3339)
		}

		rows = append(rows, record)
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}

func (r *ImportRepository) ProductExists(ctx context.Context, supplierID, productID string) (bool, error) {
	if r.client == nil {
		return false, errors.New("spanner unavailable")
	}
	row, err := r.client.Single().ReadRow(ctx, "Products", spanner.Key{productID}, []string{"SupplierId", "IsActive"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	var owner string
	var isActive bool
	if err := row.Columns(&owner, &isActive); err != nil {
		return false, err
	}
	return strings.TrimSpace(owner) == strings.TrimSpace(supplierID) && isActive, nil
}

func normalizeImportStatus(status string) string {
	return strings.ToUpper(strings.TrimSpace(status))
}

func importToNullJSON(raw json.RawMessage) (spanner.NullJSON, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return spanner.NullJSON{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return spanner.NullJSON{}, err
	}
	return spanner.NullJSON{Value: value, Valid: true}, nil
}

func importJSONRawOrNull(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func parseImportPagination(r *http.Request) (int, int) {
	limit := supplierImportDefaultLimit
	offset := 0

	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			if parsed > 0 && parsed <= supplierImportMaxLimit {
				limit = parsed
			}
		}
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}
