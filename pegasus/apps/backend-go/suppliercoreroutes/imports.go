package suppliercoreroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/storage"
	"backend-go/telemetry"
	"backend-go/ws"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	gcs "cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const (
	supplierImportRoutePrefix   = "/v1/supplier/inventory/imports"
	supplierImportMaxUploadSize = int64(50 * 1024 * 1024) // 50MB
	supplierImportDefaultLimit  = 100
	supplierImportMaxLimit      = 1000
	supplierImportMutationBatch = 5000
	supplierImportReasonBulk    = "BULK_IMPORT"
)

var supplierImportAllowedUploadExtensions = map[string]string{
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xls":  "application/vnd.ms-excel",
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
	errSupplierImportSessionNotFound  = errors.New("supplier import session not found")
	errSupplierImportInvalidStatus    = errors.New("invalid supplier import status")
	errSupplierImportStateConflict    = errors.New("supplier import status transition conflict")
	errSupplierImportAccessDenied     = errors.New("supplier import access denied")
	errSupplierImportAlreadyApplied   = errors.New("supplier import session already applied")
	errSupplierImportNoApplicableRows = errors.New("supplier import session has no applicable rows")
)

// SupplierImportSessionRecord mirrors SupplierImportSessions with spanner tags.
type SupplierImportSessionRecord struct {
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

// SupplierImportStagedRowRecord mirrors SupplierImportStagedRows with spanner tags.
type SupplierImportStagedRowRecord struct {
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

// SupplierImportMappingRecord mirrors SupplierImportMapping with spanner tags.
type SupplierImportMappingRecord struct {
	SupplierID   string           `spanner:"supplier_id" json:"supplier_id"`
	SessionID    string           `spanner:"session_id" json:"session_id"`
	MappingJSON  spanner.NullJSON `spanner:"mapping_json" json:"-"`
	Mapping      json.RawMessage  `spanner:"-" json:"mapping_json,omitempty"`
	CreatedAt    time.Time        `spanner:"created_at" json:"created_at,omitempty"`
	UpdatedAt    spanner.NullTime `spanner:"updated_at" json:"-"`
	UpdatedAtRFC string           `spanner:"-" json:"updated_at,omitempty"`
}

// SupplierImportApplySummary captures the terminal summary of sandbox->production apply.
type SupplierImportApplySummary struct {
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

type supplierImportFactKey struct {
	WarehouseID string
	SKU         string
	FactDate    civil.Date
}

type supplierImportFactAggregate struct {
	AppliedRows   int64
	QuantityDelta int64
	SessionCount  int64
	LastSessionID string
	LastAppliedAt time.Time
}

// SupplierImportRepository encapsulates supplier-scoped import sandbox persistence.
type SupplierImportRepository struct {
	client *spanner.Client

	auditMetadataColumnOnce   sync.Once
	auditMetadataColumnExists bool
	auditMetadataColumnType   string

	importFactTableOnce   sync.Once
	importFactTableExists bool
}

func NewSupplierImportRepository(client *spanner.Client) *SupplierImportRepository {
	return &SupplierImportRepository{client: client}
}

func (r *SupplierImportRepository) detectImportAnalyticsFactTable(ctx context.Context) bool {
	if r.client == nil {
		return false
	}

	r.importFactTableOnce.Do(func() {
		stmt := spanner.Statement{
			SQL: `SELECT COUNT(1)
			      FROM INFORMATION_SCHEMA.TABLES
			      WHERE TABLE_NAME = 'SupplierImportAnalyticsFacts'`,
		}

		iter := r.client.Single().Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			if err != iterator.Done {
				slog.Warn("supplier import analytics fact detection failed", "err", err)
			}
			return
		}

		var tableCount int64
		if err := row.Columns(&tableCount); err != nil {
			slog.Warn("supplier import analytics fact count parse failed", "err", err)
			return
		}

		r.importFactTableExists = tableCount > 0
	})

	return r.importFactTableExists
}

func (r *SupplierImportRepository) detectInventoryAuditMetadataColumn(ctx context.Context) (bool, string) {
	if r.client == nil {
		return false, ""
	}

	r.auditMetadataColumnOnce.Do(func() {
		stmt := spanner.Statement{
			SQL: `SELECT SPANNER_TYPE
			      FROM INFORMATION_SCHEMA.COLUMNS
			      WHERE TABLE_NAME = 'InventoryAuditLog'
			        AND COLUMN_NAME = 'Metadata'
			      LIMIT 1`,
		}

		iter := r.client.Single().Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			if err != iterator.Done {
				slog.Warn("supplier import metadata detection failed", "err", err)
			}
			return
		}

		if err := row.Columns(&r.auditMetadataColumnType); err != nil {
			slog.Warn("supplier import metadata column parse failed", "err", err)
			return
		}

		r.auditMetadataColumnExists = true
	})

	return r.auditMetadataColumnExists, strings.ToUpper(strings.TrimSpace(r.auditMetadataColumnType))
}

func (r *SupplierImportRepository) CreateImportSession(ctx context.Context, supplierID string, sessionID string, fileName string, initialStatus string) (SupplierImportSessionRecord, error) {
	if r.client == nil {
		return SupplierImportSessionRecord{}, errors.New("spanner unavailable")
	}
	status := normalizeSupplierImportStatus(initialStatus)
	if _, ok := supplierImportAllowedStatuses[status]; !ok {
		return SupplierImportSessionRecord{}, errSupplierImportInvalidStatus
	}
	if _, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.Insert(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "file_name", "total_rows", "error_summary", "created_at", "updated_at"},
			[]interface{}{supplierID, sessionID, status, fileName, int64(0), nil, spanner.CommitTimestamp, spanner.CommitTimestamp},
		)})
	}); err != nil {
		return SupplierImportSessionRecord{}, err
	}
	return r.GetSession(ctx, supplierID, sessionID)
}

func (r *SupplierImportRepository) UpdateSessionStatus(ctx context.Context, supplierID string, sessionID string, nextStatus string) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	nextStatus = normalizeSupplierImportStatus(nextStatus)
	if _, ok := supplierImportAllowedStatuses[nextStatus]; !ok {
		return errSupplierImportInvalidStatus
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errSupplierImportSessionNotFound
			}
			return readErr
		}
		var currentStatus string
		if err := row.Columns(&currentStatus); err != nil {
			return err
		}
		currentStatus = normalizeSupplierImportStatus(currentStatus)
		if currentStatus == nextStatus {
			return nil
		}
		nextSet, ok := supplierImportTransitions[currentStatus]
		if !ok {
			return errSupplierImportStateConflict
		}
		if _, allowed := nextSet[nextStatus]; !allowed {
			return errSupplierImportStateConflict
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "updated_at"},
			[]interface{}{supplierID, sessionID, nextStatus, spanner.CommitTimestamp},
		)})
	})
	return err
}

func (r *SupplierImportRepository) MarkSessionUploadedAndEmit(ctx context.Context, supplierID string, sessionID string, gcsPath string) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}

	traceID := telemetry.TraceIDFromContext(ctx)
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errSupplierImportSessionNotFound
			}
			return readErr
		}

		var currentStatus string
		if err := row.Columns(&currentStatus); err != nil {
			return err
		}
		currentStatus = normalizeSupplierImportStatus(currentStatus)
		if currentStatus == "UPLOADED" {
			return nil
		}
		if currentStatus != "INITIALIZED" {
			return errSupplierImportStateConflict
		}

		if err := txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "updated_at"},
			[]interface{}{supplierID, sessionID, "UPLOADED", spanner.CommitTimestamp},
		)}); err != nil {
			return err
		}

		event := kafkaEvents.InventoryImportUploadedEvent{
			SessionID:  sessionID,
			SupplierID: supplierID,
			GCSPath:    gcsPath,
		}
		return outbox.EmitJSON(
			txn,
			"SupplierImportSession",
			sessionID,
			kafkaEvents.EventInventoryImportUploaded,
			kafkaEvents.TopicInventoryImportEvents,
			event,
			traceID,
		)
	})

	return err
}

func (r *SupplierImportRepository) SaveStagedRows(ctx context.Context, supplierID string, sessionID string, rows []SupplierImportStagedRowRecord) error {
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
				return errSupplierImportSessionNotFound
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
			rawJSON, err := toNullJSON(row.RawData)
			if err != nil {
				return fmt.Errorf("parse raw_data row %d: %w", row.RowIndex, err)
			}
			cleanedJSON, err := toNullJSON(row.CleanedData)
			if err != nil {
				return fmt.Errorf("parse cleaned_data row %d: %w", row.RowIndex, err)
			}

			mutations = append(mutations, spanner.InsertOrUpdate(
				"SupplierImportStagedRows",
				[]string{"supplier_id", "session_id", "row_index", "raw_data", "cleaned_data", "validation_errors", "is_new_product", "created_at", "updated_at"},
				[]interface{}{supplierID, sessionID, row.RowIndex, rawJSON, cleanedJSON, row.ValidationErrors, row.IsNewProduct, spanner.CommitTimestamp, spanner.CommitTimestamp},
			))
		}

		mutations = append(mutations, spanner.InsertOrUpdate(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "total_rows", "updated_at"},
			[]interface{}{supplierID, sessionID, maxRow, spanner.CommitTimestamp},
		))
		return txn.BufferWrite(mutations)
	})
	return err
}

func (r *SupplierImportRepository) SaveMapping(ctx context.Context, supplierID string, sessionID string, mapping json.RawMessage) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	mappingJSON, err := toNullJSON(mapping)
	if err != nil {
		return err
	}

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"session_id"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errSupplierImportSessionNotFound
			}
			return readErr
		}

		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdate(
				"SupplierImportMapping",
				[]string{"supplier_id", "session_id", "mapping_json", "created_at", "updated_at"},
				[]interface{}{supplierID, sessionID, mappingJSON, spanner.CommitTimestamp, spanner.CommitTimestamp},
			),
			spanner.Update(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "updated_at"},
				[]interface{}{supplierID, sessionID, "MAPPING_REQUIRED", spanner.CommitTimestamp},
			),
		})
	})
	return err
}

func (r *SupplierImportRepository) GetSession(ctx context.Context, supplierID string, sessionID string) (SupplierImportSessionRecord, error) {
	if r.client == nil {
		return SupplierImportSessionRecord{}, errors.New("spanner unavailable")
	}

	row, err := r.client.Single().ReadRow(
		ctx,
		"SupplierImportSessions",
		spanner.Key{supplierID, sessionID},
		[]string{"supplier_id", "session_id", "status", "file_name", "total_rows", "error_summary", "created_at", "updated_at"},
	)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return SupplierImportSessionRecord{}, errSupplierImportSessionNotFound
		}
		return SupplierImportSessionRecord{}, err
	}

	var record SupplierImportSessionRecord
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
		return SupplierImportSessionRecord{}, err
	}

	if record.ErrorSummaryJSON.Valid {
		encoded, marshalErr := json.Marshal(record.ErrorSummaryJSON.Value)
		if marshalErr != nil {
			return SupplierImportSessionRecord{}, marshalErr
		}
		record.ErrorSummary = encoded
	}
	record.CreatedAt = record.CreatedAt.UTC()
	if record.UpdatedAt.Valid {
		record.UpdatedAtRFC3339 = record.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}
	return record, nil
}

func (r *SupplierImportRepository) GetMapping(ctx context.Context, supplierID string, sessionID string) (SupplierImportMappingRecord, error) {
	if r.client == nil {
		return SupplierImportMappingRecord{}, errors.New("spanner unavailable")
	}

	if _, err := r.GetSession(ctx, supplierID, sessionID); err != nil {
		return SupplierImportMappingRecord{}, err
	}

	row, err := r.client.Single().ReadRow(
		ctx,
		"SupplierImportMapping",
		spanner.Key{supplierID, sessionID},
		[]string{"supplier_id", "session_id", "mapping_json", "created_at", "updated_at"},
	)
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return SupplierImportMappingRecord{SupplierID: supplierID, SessionID: sessionID}, nil
		}
		return SupplierImportMappingRecord{}, err
	}

	var record SupplierImportMappingRecord
	if err := row.Columns(
		&record.SupplierID,
		&record.SessionID,
		&record.MappingJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return SupplierImportMappingRecord{}, err
	}

	if record.MappingJSON.Valid {
		encoded, marshalErr := json.Marshal(record.MappingJSON.Value)
		if marshalErr != nil {
			return SupplierImportMappingRecord{}, marshalErr
		}
		record.Mapping = encoded
	}
	record.CreatedAt = record.CreatedAt.UTC()
	if record.UpdatedAt.Valid {
		record.UpdatedAtRFC = record.UpdatedAt.Time.UTC().Format(time.RFC3339)
	}

	return record, nil
}

func (r *SupplierImportRepository) ListRows(ctx context.Context, supplierID string, sessionID string, limit int, offset int) ([]SupplierImportStagedRowRecord, bool, error) {
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
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"sessionId":  sessionID,
			"limit":      int64(limit + 1),
			"offset":     int64(offset),
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	rows := make([]SupplierImportStagedRowRecord, 0, limit+1)
	for {
		row, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, false, err
		}

		var record SupplierImportStagedRowRecord
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

// ApplyImportSession atomically applies approved staged rows into live inventory tables.
func (r *SupplierImportRepository) ApplyImportSession(ctx context.Context, supplierID string, sessionID string) (SupplierImportApplySummary, error) {
	summary := SupplierImportApplySummary{
		SessionID: sessionID,
		Status:    "APPLIED",
	}
	if r.client == nil {
		return summary, errors.New("spanner unavailable")
	}

	updatedSummary, err := r.applyImportSessionTxn(ctx, supplierID, sessionID)
	if err == nil {
		return updatedSummary, nil
	}

	if errors.Is(err, errSupplierImportAlreadyApplied) {
		summary.Idempotent = true
		return summary, nil
	}

	if errors.Is(err, errSupplierImportSessionNotFound) ||
		errors.Is(err, errSupplierImportStateConflict) ||
		errors.Is(err, errSupplierImportAccessDenied) {
		return summary, err
	}

	if markErr := r.markApplyFailure(ctx, supplierID, sessionID, err); markErr != nil && !errors.Is(markErr, errSupplierImportSessionNotFound) {
		return summary, fmt.Errorf("apply import session failed: %w (mark failed: %v)", err, markErr)
	}

	if errors.Is(err, errSupplierImportNoApplicableRows) {
		return summary, errSupplierImportStateConflict
	}

	return summary, err
}

func (r *SupplierImportRepository) applyImportSessionTxn(ctx context.Context, supplierID string, sessionID string) (SupplierImportApplySummary, error) {
	summary := SupplierImportApplySummary{
		SessionID: sessionID,
		Status:    "APPLIED",
	}
	auditMetadataColumnExists, auditMetadataColumnType := r.detectInventoryAuditMetadataColumn(ctx)
	importFactTableExists := r.detectImportAnalyticsFactTable(ctx)
	affectedWarehouses := map[string]struct{}{}
	affectedProducts := map[string]struct{}{}
	factAggregates := map[supplierImportFactKey]*supplierImportFactAggregate{}
	seenRowKeys := map[string]struct{}{}
	applyTimestamp := time.Now().UTC()
	applyFactDate := civil.DateOf(applyTimestamp)

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sessionRow, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errSupplierImportSessionNotFound
			}
			return readErr
		}

		var sessionStatus string
		if err := sessionRow.Columns(&sessionStatus); err != nil {
			return err
		}
		sessionStatus = normalizeSupplierImportStatus(sessionStatus)
		switch sessionStatus {
		case "APPLIED":
			return errSupplierImportAlreadyApplied
		case "APPROVED", "APPLYING":
			// Allowed.
		default:
			return errSupplierImportStateConflict
		}

		batchMutations := make([]*spanner.Mutation, 0, supplierImportMutationBatch)
		flush := func() error {
			if len(batchMutations) == 0 {
				return nil
			}
			if err := txn.BufferWrite(batchMutations); err != nil {
				return err
			}
			batchMutations = batchMutations[:0]
			return nil
		}

		if sessionStatus == "APPROVED" {
			batchMutations = append(batchMutations, spanner.Update(
				"SupplierImportSessions",
				[]string{"supplier_id", "session_id", "status", "updated_at"},
				[]interface{}{supplierID, sessionID, "APPLYING", spanner.CommitTimestamp},
			))
		}

		warehouseOwnerCache := make(map[string]string)
		stmt := spanner.Statement{
			SQL: `SELECT row_index, raw_data, cleaned_data, is_new_product
			      FROM SupplierImportStagedRows
			      WHERE supplier_id = @supplierId
			        AND session_id = @sessionId
			        AND (validation_errors IS NULL OR ARRAY_LENGTH(validation_errors) = 0)
			      ORDER BY row_index`,
			Params: map[string]interface{}{
				"supplierId": supplierID,
				"sessionId":  sessionID,
			},
		}

		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("list staged rows for apply: %w", nextErr)
			}

			var rowIndex int64
			var rawDataJSON spanner.NullJSON
			var cleanedDataJSON spanner.NullJSON
			var isNewProduct bool
			if err := row.Columns(&rowIndex, &rawDataJSON, &cleanedDataJSON, &isNewProduct); err != nil {
				return fmt.Errorf("parse staged row for apply: %w", err)
			}

			rawData := supplierImportJSONMap(rawDataJSON)
			cleanedData := supplierImportJSONMap(cleanedDataJSON)

			rowSupplierID := supplierImportStringValue(cleanedData, rawData, "supplier_id")
			if rowSupplierID != "" && rowSupplierID != supplierID {
				return errSupplierImportAccessDenied
			}

			warehouseID := strings.TrimSpace(supplierImportStringValue(cleanedData, rawData, "warehouse_id", "warehouse"))
			if warehouseID == "" {
				return fmt.Errorf("row %d missing warehouse_id", rowIndex)
			}

			warehouseOwner, cached := warehouseOwnerCache[warehouseID]
			if !cached {
				warehouseRow, whErr := txn.ReadRow(ctx, "Warehouses", spanner.Key{warehouseID}, []string{"SupplierId"})
				if whErr != nil {
					if spanner.ErrCode(whErr) == codes.NotFound {
						return errSupplierImportAccessDenied
					}
					return fmt.Errorf("read warehouse owner: %w", whErr)
				}
				if err := warehouseRow.Columns(&warehouseOwner); err != nil {
					return fmt.Errorf("parse warehouse owner: %w", err)
				}
				warehouseOwnerCache[warehouseID] = warehouseOwner
			}
			if warehouseOwner != supplierID {
				return errSupplierImportAccessDenied
			}

			targetSKU := strings.TrimSpace(supplierImportStringValue(cleanedData, rawData, "sku_id", "product_id", "sku", "item_code"))
			if targetSKU == "" {
				if !isNewProduct {
					return fmt.Errorf("row %d missing sku_id", rowIndex)
				}
				targetSKU = supplierImportDefaultSKU(sessionID, rowIndex)
			}

			productExists := false
			productOwnerRow, productErr := txn.ReadRow(ctx, "SupplierProducts", spanner.Key{targetSKU}, []string{"SupplierId"})
			if productErr == nil {
				productExists = true
				var ownerSupplierID string
				if err := productOwnerRow.Columns(&ownerSupplierID); err != nil {
					return fmt.Errorf("parse product owner: %w", err)
				}
				if ownerSupplierID != supplierID {
					return errSupplierImportAccessDenied
				}
			} else if spanner.ErrCode(productErr) != codes.NotFound {
				return fmt.Errorf("read supplier product: %w", productErr)
			}

			if !productExists && !isNewProduct {
				return fmt.Errorf("row %d sku_id not found in supplier catalog", rowIndex)
			}

			if isNewProduct {
				productName := strings.TrimSpace(supplierImportStringValue(cleanedData, rawData, "product_name", "name", "item_name"))
				if productName == "" {
					productName = fmt.Sprintf("Imported SKU %d", rowIndex+1)
				}
				basePrice, ok := supplierImportInt64Value(cleanedData, rawData, "unit_price", "base_price", "price")
				if !ok || basePrice <= 0 {
					basePrice = 1
				}
				minimumOrderQty, ok := supplierImportInt64Value(cleanedData, rawData, "minimum_order_qty", "minimum_order", "moq")
				if !ok || minimumOrderQty <= 0 {
					minimumOrderQty = 1
				}
				stepSize, ok := supplierImportInt64Value(cleanedData, rawData, "step_size", "step")
				if !ok || stepSize <= 0 {
					stepSize = 1
				}
				if minimumOrderQty < stepSize {
					minimumOrderQty = stepSize
				}

				volumetricUnit, ok := supplierImportFloat64Value(cleanedData, rawData, "volumetric_unit", "vu")
				lengthCM, hasLength := supplierImportFloat64Value(cleanedData, rawData, "length_cm")
				widthCM, hasWidth := supplierImportFloat64Value(cleanedData, rawData, "width_cm")
				heightCM, hasHeight := supplierImportFloat64Value(cleanedData, rawData, "height_cm")
				if (!ok || volumetricUnit <= 0) && hasLength && hasWidth && hasHeight {
					volumetricUnit = supplierImportCalculateVU(lengthCM, widthCM, heightCM)
				}
				if volumetricUnit <= 0 {
					volumetricUnit = 1.0
				}

				categoryID := strings.TrimSpace(supplierImportStringValue(cleanedData, rawData, "category_id", "category"))
				description := strings.TrimSpace(supplierImportStringValue(cleanedData, rawData, "description"))
				imageURL := strings.TrimSpace(supplierImportStringValue(cleanedData, rawData, "image_url"))

				productCols := []string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl", "CategoryId", "SellByBlock", "UnitsPerBlock", "BasePrice", "VolumetricUnit", "MinimumOrderQty", "StepSize", "IsActive", "UpdatedAt"}
				productVals := []interface{}{targetSKU, supplierID, productName, description, imageURL, supplierImportNullableString(categoryID), false, int64(1), basePrice, volumetricUnit, minimumOrderQty, stepSize, true, spanner.CommitTimestamp}
				if !productExists {
					productCols = append(productCols, "CreatedAt")
					productVals = append(productVals, spanner.CommitTimestamp)
				}
				if hasLength {
					productCols = append(productCols, "LengthCM")
					productVals = append(productVals, lengthCM)
				}
				if hasWidth {
					productCols = append(productCols, "WidthCM")
					productVals = append(productVals, widthCM)
				}
				if hasHeight {
					productCols = append(productCols, "HeightCM")
					productVals = append(productVals, heightCM)
				}

				batchMutations = append(batchMutations, spanner.InsertOrUpdate("SupplierProducts", productCols, productVals))
				summary.CreatedProducts++
			}

			legacyQty := int64(0)
			legacyRow, legacyErr := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{targetSKU}, []string{"SupplierId", "QuantityAvailable"})
			if legacyErr == nil {
				var ownerSupplierID string
				if err := legacyRow.Columns(&ownerSupplierID, &legacyQty); err != nil {
					return fmt.Errorf("parse legacy inventory owner/qty: %w", err)
				}
				if ownerSupplierID != supplierID {
					return errSupplierImportAccessDenied
				}
			} else if spanner.ErrCode(legacyErr) != codes.NotFound {
				return fmt.Errorf("read legacy inventory row: %w", legacyErr)
			}

			v2Qty := int64(0)
			v2Row, v2Err := txn.ReadRow(ctx, "SupplierInventoryV2", spanner.Key{supplierID, warehouseID, targetSKU}, []string{"QuantityAvailable"})
			if v2Err == nil {
				if err := v2Row.Columns(&v2Qty); err != nil {
					return fmt.Errorf("parse inventory v2 qty: %w", err)
				}
			} else if spanner.ErrCode(v2Err) != codes.NotFound {
				return fmt.Errorf("read inventory v2 row: %w", v2Err)
			}

			quantityAvailable, hasQuantityAvailable := supplierImportInt64Value(cleanedData, rawData, "quantity_available", "quantity", "qty")
			quantityDelta, hasQuantityDelta := supplierImportInt64Value(cleanedData, rawData, "quantity_delta", "delta")
			if !hasQuantityAvailable && !hasQuantityDelta {
				return fmt.Errorf("row %d missing quantity_available or quantity_delta", rowIndex)
			}

			var delta int64
			var newV2Qty int64
			if hasQuantityAvailable {
				if quantityAvailable < 0 {
					return fmt.Errorf("row %d quantity_available cannot be negative", rowIndex)
				}
				newV2Qty = quantityAvailable
				delta = newV2Qty - v2Qty
			} else {
				newV2Qty = v2Qty + quantityDelta
				delta = quantityDelta
			}

			if newV2Qty < 0 {
				return fmt.Errorf("row %d would make warehouse quantity negative", rowIndex)
			}

			newLegacyQty := legacyQty + delta
			if newLegacyQty < 0 {
				return fmt.Errorf("row %d would make supplier quantity negative", rowIndex)
			}

			rowDeterministicKey := fmt.Sprintf("%s|%s|%s|%d", supplierID, warehouseID, targetSKU, rowIndex)
			if _, alreadySeen := seenRowKeys[rowDeterministicKey]; alreadySeen {
				return fmt.Errorf("row %d duplicate deterministic key", rowIndex)
			}
			seenRowKeys[rowDeterministicKey] = struct{}{}

			batchMutations = append(batchMutations,
				spanner.InsertOrUpdate(
					"SupplierInventory",
					[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
					[]interface{}{targetSKU, supplierID, newLegacyQty, spanner.CommitTimestamp},
				),
				spanner.InsertOrUpdate(
					"SupplierInventoryV2",
					[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityAvailable", "UpdatedAt"},
					[]interface{}{supplierID, warehouseID, targetSKU, newV2Qty, spanner.CommitTimestamp},
				),
			)

			auditCols := []string{"AuditId", "ProductId", "SupplierId", "AdjustedBy", "PreviousQty", "NewQty", "Delta", "Reason", "AdjustedAt", "WarehouseId"}
			auditVals := []interface{}{supplierImportAuditID(sessionID, rowIndex), targetSKU, supplierID, supplierID, legacyQty, newLegacyQty, delta, supplierImportReasonBulk, spanner.CommitTimestamp, warehouseID}
			if auditMetadataColumnExists {
				auditMetadata, err := json.Marshal(map[string]interface{}{
					"source":      "SUPPLIER_IMPORT_SANDBOX_APPLY",
					"session_id":  sessionID,
					"batch_index": rowIndex,
				})
				if err != nil {
					return fmt.Errorf("marshal inventory audit metadata: %w", err)
				}

				auditCols = append(auditCols, "Metadata")
				if strings.HasPrefix(auditMetadataColumnType, "JSON") {
					metadataJSON, parseErr := toNullJSON(auditMetadata)
					if parseErr != nil {
						return fmt.Errorf("parse inventory audit metadata json: %w", parseErr)
					}
					auditVals = append(auditVals, metadataJSON)
				} else {
					auditVals = append(auditVals, string(auditMetadata))
				}
			}
			batchMutations = append(batchMutations, spanner.InsertOrUpdate("InventoryAuditLog", auditCols, auditVals))

			if importFactTableExists {
				factKey := supplierImportFactKey{
					WarehouseID: warehouseID,
					SKU:         targetSKU,
					FactDate:    applyFactDate,
				}
				aggregate, exists := factAggregates[factKey]
				if !exists {
					aggregate = &supplierImportFactAggregate{
						SessionCount:  1,
						LastSessionID: sessionID,
						LastAppliedAt: applyTimestamp,
					}
					factAggregates[factKey] = aggregate
				}
				aggregate.AppliedRows++
				aggregate.QuantityDelta += delta
			}

			summary.AppliedRows++
			affectedWarehouses[warehouseID] = struct{}{}
			affectedProducts[targetSKU] = struct{}{}

			if len(batchMutations) >= supplierImportMutationBatch {
				if err := flush(); err != nil {
					return fmt.Errorf("flush apply mutation batch: %w", err)
				}
			}
		}

		if summary.AppliedRows == 0 {
			return errSupplierImportNoApplicableRows
		}

		if importFactTableExists && len(factAggregates) > 0 {
			for factKey, aggregate := range factAggregates {
				existingAppliedRows := int64(0)
				existingQuantityDelta := int64(0)
				existingSessionCount := int64(0)

				factRow, factErr := txn.ReadRow(
					ctx,
					"SupplierImportAnalyticsFacts",
					spanner.Key{supplierID, factKey.WarehouseID, factKey.FactDate, factKey.SKU},
					[]string{"applied_rows", "quantity_delta", "session_count"},
				)
				if factErr == nil {
					if err := factRow.Columns(&existingAppliedRows, &existingQuantityDelta, &existingSessionCount); err != nil {
						return fmt.Errorf("parse import analytics fact row: %w", err)
					}
				} else if spanner.ErrCode(factErr) != codes.NotFound {
					return fmt.Errorf("read import analytics fact row: %w", factErr)
				}

				batchMutations = append(batchMutations, spanner.InsertOrUpdate(
					"SupplierImportAnalyticsFacts",
					[]string{
						"supplier_id",
						"warehouse_id",
						"fact_date",
						"sku_id",
						"applied_rows",
						"quantity_delta",
						"session_count",
						"last_session_id",
						"last_applied_at",
						"updated_at",
					},
					[]interface{}{
						supplierID,
						factKey.WarehouseID,
						factKey.FactDate,
						factKey.SKU,
						existingAppliedRows + aggregate.AppliedRows,
						existingQuantityDelta + aggregate.QuantityDelta,
						existingSessionCount + aggregate.SessionCount,
						aggregate.LastSessionID,
						aggregate.LastAppliedAt,
						spanner.CommitTimestamp,
					},
				))

				if len(batchMutations) >= supplierImportMutationBatch {
					if err := flush(); err != nil {
						return fmt.Errorf("flush import analytics fact mutations: %w", err)
					}
				}
			}
		}

		batchMutations = append(batchMutations, spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
			[]interface{}{supplierID, sessionID, "APPLIED", nil, spanner.CommitTimestamp},
		))

		if err := flush(); err != nil {
			return fmt.Errorf("flush final apply mutations: %w", err)
		}
		return nil
	})

	if err != nil {
		return summary, err
	}

	if len(affectedWarehouses) > 0 {
		summary.WarehouseIDs = make([]string, 0, len(affectedWarehouses))
		for warehouseID := range affectedWarehouses {
			summary.WarehouseIDs = append(summary.WarehouseIDs, warehouseID)
		}
		sort.Strings(summary.WarehouseIDs)
	}

	if len(affectedProducts) > 0 {
		summary.AppliedProductIDs = make([]string, 0, len(affectedProducts))
		for productID := range affectedProducts {
			summary.AppliedProductIDs = append(summary.AppliedProductIDs, productID)
		}
		sort.Strings(summary.AppliedProductIDs)
	}

	summary.AffectedWarehouses = int64(len(summary.WarehouseIDs))
	summary.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	return summary, nil
}

func (r *SupplierImportRepository) markApplyFailure(ctx context.Context, supplierID string, sessionID string, applyErr error) error {
	if r.client == nil {
		return errors.New("spanner unavailable")
	}
	errorSummaryPayload := map[string]interface{}{
		"phase":      "APPLY",
		"session_id": sessionID,
		"error":      applyErr.Error(),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	errorSummaryBytes, marshalErr := json.Marshal(errorSummaryPayload)
	if marshalErr != nil {
		return marshalErr
	}
	errorSummaryJSON, parseErr := toNullJSON(errorSummaryBytes)
	if parseErr != nil {
		return parseErr
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if readErr != nil {
			if spanner.ErrCode(readErr) == codes.NotFound {
				return errSupplierImportSessionNotFound
			}
			return readErr
		}

		var currentStatus string
		if err := row.Columns(&currentStatus); err != nil {
			return err
		}
		if normalizeSupplierImportStatus(currentStatus) == "APPLIED" {
			return nil
		}

		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"SupplierImportSessions",
			[]string{"supplier_id", "session_id", "status", "error_summary", "updated_at"},
			[]interface{}{supplierID, sessionID, "FAILED", errorSummaryJSON, spanner.CommitTimestamp},
		)})
	})

	return err
}

func supplierImportJSONMap(source spanner.NullJSON) map[string]interface{} {
	if !source.Valid {
		return map[string]interface{}{}
	}
	m, ok := source.Value.(map[string]interface{})
	if ok {
		return m
	}
	if source.Value == nil {
		return map[string]interface{}{}
	}
	encoded, err := json.Marshal(source.Value)
	if err != nil {
		return map[string]interface{}{}
	}
	decoded := map[string]interface{}{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return map[string]interface{}{}
	}
	return decoded
}

func supplierImportLookupValue(cleaned map[string]interface{}, raw map[string]interface{}, keys ...string) (interface{}, bool) {
	lookup := func(dataset map[string]interface{}, key string) (interface{}, bool) {
		if value, ok := dataset[key]; ok {
			return value, true
		}
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		for candidate, value := range dataset {
			if strings.ToLower(strings.TrimSpace(candidate)) == normalizedKey {
				return value, true
			}
		}
		return nil, false
	}

	for _, key := range keys {
		if value, ok := lookup(cleaned, key); ok {
			return value, true
		}
		if value, ok := lookup(raw, key); ok {
			return value, true
		}
	}

	return nil, false
}

func supplierImportStringValue(cleaned map[string]interface{}, raw map[string]interface{}, keys ...string) string {
	value, ok := supplierImportLookupValue(cleaned, raw, keys...)
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func supplierImportInt64Value(cleaned map[string]interface{}, raw map[string]interface{}, keys ...string) (int64, bool) {
	value, ok := supplierImportLookupValue(cleaned, raw, keys...)
	if !ok || value == nil {
		return 0, false
	}

	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float32:
		return int64(math.Round(float64(typed))), true
	case float64:
		return int64(math.Round(typed)), true
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed, true
		}
		if parsed, err := typed.Float64(); err == nil {
			return int64(math.Round(parsed)), true
		}
	case string:
		normalized := strings.ReplaceAll(strings.TrimSpace(typed), ",", "")
		if normalized == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseInt(normalized, 10, 64); err == nil {
			return parsed, true
		}
		if parsed, err := strconv.ParseFloat(normalized, 64); err == nil {
			return int64(math.Round(parsed)), true
		}
	}

	return 0, false
}

func supplierImportFloat64Value(cleaned map[string]interface{}, raw map[string]interface{}, keys ...string) (float64, bool) {
	value, ok := supplierImportLookupValue(cleaned, raw, keys...)
	if !ok || value == nil {
		return 0, false
	}

	switch typed := value.(type) {
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err == nil {
			return parsed, true
		}
	case string:
		normalized := strings.ReplaceAll(strings.TrimSpace(typed), ",", "")
		if normalized == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(normalized, 64)
		if err == nil {
			return parsed, true
		}
	}

	return 0, false
}

func supplierImportNullableString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func supplierImportCalculateVU(lengthCM float64, widthCM float64, heightCM float64) float64 {
	if lengthCM <= 0 || widthCM <= 0 || heightCM <= 0 {
		return 0
	}
	return (lengthCM * widthCM * heightCM) / 5000
}

func supplierImportDefaultSKU(sessionID string, rowIndex int64) string {
	seed := fmt.Sprintf("%s:%d", sessionID, rowIndex)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}

func supplierImportAuditID(sessionID string, rowIndex int64) string {
	compact := strings.ReplaceAll(strings.TrimSpace(sessionID), "-", "")
	if len(compact) > 12 {
		compact = compact[:12]
	}
	return fmt.Sprintf("AIM-%s-%d", compact, rowIndex)
}

func registerImportRoutes(r chi.Router, d Deps, supplierRole []string, log Middleware, withRegionScope Middleware, idem Middleware) {
	repo := NewSupplierImportRepository(d.Spanner)

	createHandler := withMethodIdempotency(handleCreateSupplierImportSession(repo), idem, http.MethodPost)
	uploadedHandler := withMethodIdempotency(handlePostSupplierImportUploaded(repo), idem, http.MethodPost)
	mappingHandler := withMethodIdempotency(handlePostSupplierImportMapping(repo), idem, http.MethodPost)
	approveHandler := withMethodIdempotency(handlePostSupplierImportApprove(repo), idem, http.MethodPost)
	applyHandler := withMethodIdempotency(handlePostSupplierImportApply(repo, d.SupplierHub, d.WarehouseHub), idem, http.MethodPost)
	mappingReadHandler := handleGetSupplierImportMapping(repo)
	rowsReadHandler := handleGetSupplierImportRows(repo)
	mappingRouteHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			mappingReadHandler(w, r)
			return
		}
		mappingHandler(w, r)
	}

	r.Route("/v1/supplier/inventory/imports", func(imports chi.Router) {
		imports.HandleFunc("/",
			auth.RequireRole(supplierRole, log(withRegionScope(createHandler))))
		imports.HandleFunc("/{id}",
			auth.RequireRole(supplierRole, log(withRegionScope(handleGetSupplierImportSession(repo)))))
		imports.HandleFunc("/{id}/uploaded",
			auth.RequireRole(supplierRole, log(withRegionScope(uploadedHandler))))
		imports.HandleFunc("/{id}/rows",
			auth.RequireRole(supplierRole, log(withRegionScope(rowsReadHandler))))
		imports.HandleFunc("/{id}/mapping",
			auth.RequireRole(supplierRole, log(withRegionScope(mappingRouteHandler))))
		imports.HandleFunc("/{id}/approve",
			auth.RequireRole(supplierRole, log(withRegionScope(approveHandler))))
		imports.HandleFunc("/{id}/apply",
			auth.RequireRole(supplierRole, log(withRegionScope(applyHandler))))
	})
}

type inventorySyncCompleteFrame struct {
	Type               string   `json:"type"`
	SupplierID         string   `json:"supplier_id"`
	WarehouseID        string   `json:"warehouse_id,omitempty"`
	SessionID          string   `json:"session_id"`
	RowsAffected       int64    `json:"rows_affected"`
	AffectedWarehouses int64    `json:"affected_warehouses"`
	ProductIDs         []string `json:"product_ids,omitempty"`
	Source             string   `json:"source"`
	Timestamp          string   `json:"timestamp"`
}

func broadcastInventorySyncComplete(
	supplierID string,
	summary SupplierImportApplySummary,
	supplierHub *ws.SupplierHub,
	warehouseHub *ws.WarehouseHub,
) {
	timestamp := strings.TrimSpace(summary.Timestamp)
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	baseFrame := inventorySyncCompleteFrame{
		Type:               ws.EventInventorySyncComplete,
		SupplierID:         supplierID,
		SessionID:          summary.SessionID,
		RowsAffected:       summary.AppliedRows,
		AffectedWarehouses: summary.AffectedWarehouses,
		ProductIDs:         summary.AppliedProductIDs,
		Source:             "SUPPLIER_IMPORT_SANDBOX_APPLY",
		Timestamp:          timestamp,
	}

	if supplierHub != nil {
		supplierHub.PushToSupplier(supplierID, baseFrame)
	}

	if warehouseHub == nil {
		return
	}

	for _, warehouseID := range summary.WarehouseIDs {
		if strings.TrimSpace(warehouseID) == "" {
			continue
		}
		frame := baseFrame
		frame.WarehouseID = warehouseID
		warehouseHub.PushToWarehouse(warehouseID, frame)
	}
}

func handleCreateSupplierImportSession(repo *SupplierImportRepository) http.HandlerFunc {
	type request struct {
		FileName      string `json:"file_name"`
		FileSizeBytes int64  `json:"file_size_bytes"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		var payload request
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		payload.FileName = strings.TrimSpace(payload.FileName)
		if payload.FileName == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "file_name is required"})
			return
		}
		if payload.FileSizeBytes <= 0 {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "file_size_bytes is required"})
			return
		}
		if payload.FileSizeBytes > supplierImportMaxUploadSize {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "file exceeds max size (50MB)"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sessionID := uuid.NewString()
		uploadURL, gcsPath, contentType, ticketErr := createSupplierImportUploadTicket(supplierID, sessionID, payload.FileName, payload.FileSizeBytes)
		if ticketErr != nil {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": ticketErr.Error()})
			return
		}

		session, err := repo.CreateImportSession(ctx, supplierID, sessionID, payload.FileName, "INITIALIZED")
		if err != nil {
			writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create import session"})
			return
		}

		writeSupplierImportJSON(w, http.StatusCreated, map[string]interface{}{
			"session_id":          session.SessionID,
			"status":              session.Status,
			"file_name":           session.FileName,
			"upload_url":          uploadURL,
			"gcs_path":            gcsPath,
			"content_type":        contentType,
			"expires_in_seconds":  int64((15 * time.Minute).Seconds()),
			"max_file_size_bytes": supplierImportMaxUploadSize,
			"route_prefix":        supplierImportRoutePrefix,
			"supplier_id":         session.SupplierID,
			"created_at":          session.CreatedAt.Format(time.RFC3339),
			"updated_at":          session.UpdatedAtRFC3339,
			"status_description":  "initialized",
		})
	}
}

func handleGetSupplierImportSession(repo *SupplierImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		session, err := repo.GetSession(ctx, supplierID, sessionID)
		if err != nil {
			if errors.Is(err, errSupplierImportSessionNotFound) {
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
			return
		}

		writeSupplierImportJSON(w, http.StatusOK, map[string]interface{}{
			"supplier_id":   session.SupplierID,
			"session_id":    session.SessionID,
			"status":        session.Status,
			"file_name":     session.FileName,
			"gcs_path":      supplierImportObjectPath(session.SupplierID, session.SessionID),
			"total_rows":    session.TotalRows,
			"error_summary": jsonRawOrNull(session.ErrorSummary),
			"created_at":    session.CreatedAt.Format(time.RFC3339),
			"updated_at":    session.UpdatedAtRFC3339,
		})
	}
}

func handleGetSupplierImportRows(repo *SupplierImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		limit, offset := parseSupplierImportPagination(r)
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rows, hasMore, err := repo.ListRows(ctx, supplierID, sessionID, limit, offset)
		if err != nil {
			if errors.Is(err, errSupplierImportSessionNotFound) {
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load staged rows"})
			return
		}

		writeSupplierImportJSON(w, http.StatusOK, map[string]interface{}{
			"session_id":    sessionID,
			"limit":         limit,
			"offset":        offset,
			"has_more":      hasMore,
			"next_offset":   offset + len(rows),
			"rows_returned": len(rows),
			"data":          rows,
		})
	}
}

func handleGetSupplierImportMapping(repo *SupplierImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		mapping, err := repo.GetMapping(ctx, supplierID, sessionID)
		if err != nil {
			if errors.Is(err, errSupplierImportSessionNotFound) {
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load mapping"})
			return
		}

		createdAt := ""
		if !mapping.CreatedAt.IsZero() {
			createdAt = mapping.CreatedAt.Format(time.RFC3339)
		}

		writeSupplierImportJSON(w, http.StatusOK, map[string]interface{}{
			"supplier_id":  supplierID,
			"session_id":   sessionID,
			"mapping_json": jsonRawOrNull(mapping.Mapping),
			"created_at":   createdAt,
			"updated_at":   mapping.UpdatedAtRFC,
		})
	}
}

func handlePostSupplierImportUploaded(repo *SupplierImportRepository) http.HandlerFunc {
	type request struct {
		GCSPath string `json:"gcs_path"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		expectedGCSPath := supplierImportObjectPath(supplierID, sessionID)

		var payload request
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
				writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
		}
		payload.GCSPath = strings.TrimSpace(payload.GCSPath)
		if payload.GCSPath != "" && payload.GCSPath != expectedGCSPath {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "gcs_path mismatch"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := repo.MarkSessionUploadedAndEmit(ctx, supplierID, sessionID, expectedGCSPath); err != nil {
			switch {
			case errors.Is(err, errSupplierImportSessionNotFound):
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case errors.Is(err, errSupplierImportStateConflict):
				writeSupplierImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for uploaded transition"})
			default:
				writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to mark session uploaded"})
			}
			return
		}

		writeSupplierImportJSON(w, http.StatusAccepted, map[string]interface{}{
			"session_id": sessionID,
			"status":     "UPLOADED",
			"gcs_path":   expectedGCSPath,
			"event_type": kafkaEvents.EventInventoryImportUploaded,
			"topic":      kafkaEvents.TopicInventoryImportEvents,
		})
	}
}

func handlePostSupplierImportMapping(repo *SupplierImportRepository) http.HandlerFunc {
	type wrapped struct {
		MappingJSON json.RawMessage `json:"mapping_json"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		body = []byte(strings.TrimSpace(string(body)))
		if len(body) == 0 {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "mapping payload is required"})
			return
		}

		mapping := json.RawMessage(body)
		var wrappedBody wrapped
		if err := json.Unmarshal(body, &wrappedBody); err == nil && len(wrappedBody.MappingJSON) > 0 {
			mapping = wrappedBody.MappingJSON
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := repo.SaveMapping(ctx, supplierID, sessionID, mapping); err != nil {
			if errors.Is(err, errSupplierImportSessionNotFound) {
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save mapping"})
			return
		}

		writeSupplierImportJSON(w, http.StatusAccepted, map[string]interface{}{
			"session_id": sessionID,
			"status":     "MAPPING_REQUIRED",
		})
	}
}

func handlePostSupplierImportApprove(repo *SupplierImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := repo.UpdateSessionStatus(ctx, supplierID, sessionID, "APPROVED"); err != nil {
			switch {
			case errors.Is(err, errSupplierImportSessionNotFound):
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case errors.Is(err, errSupplierImportStateConflict):
				writeSupplierImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for approve"})
			case errors.Is(err, errSupplierImportInvalidStatus):
				writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status transition"})
			default:
				writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to approve session"})
			}
			return
		}

		writeSupplierImportJSON(w, http.StatusAccepted, map[string]interface{}{
			"session_id": sessionID,
			"status":     "APPROVED",
			"next_phase": "apply transition is triggered in phase 6",
		})
	}
}

func handlePostSupplierImportApply(repo *SupplierImportRepository, supplierHub *ws.SupplierHub, warehouseHub *ws.WarehouseHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		supplierID, err := supplierIDFromContext(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeSupplierImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeSupplierImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		summary, err := repo.ApplyImportSession(ctx, supplierID, sessionID)
		if err != nil {
			switch {
			case errors.Is(err, errSupplierImportSessionNotFound):
				writeSupplierImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case errors.Is(err, errSupplierImportAccessDenied):
				writeSupplierImportJSON(w, http.StatusForbidden, map[string]string{"error": "session does not belong to supplier"})
			case errors.Is(err, errSupplierImportStateConflict):
				writeSupplierImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for apply"})
			default:
				writeSupplierImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to apply session"})
			}
			return
		}

		if !summary.Idempotent {
			broadcastInventorySyncComplete(supplierID, summary, supplierHub, warehouseHub)
		}

		writeSupplierImportJSON(w, http.StatusOK, map[string]interface{}{
			"session_id":          summary.SessionID,
			"status":              summary.Status,
			"idempotent":          summary.Idempotent,
			"applied_rows":        summary.AppliedRows,
			"created_products":    summary.CreatedProducts,
			"affected_warehouses": summary.AffectedWarehouses,
			"warehouse_ids":       summary.WarehouseIDs,
			"product_ids":         summary.AppliedProductIDs,
			"timestamp":           summary.Timestamp,
			"source":              "SUPPLIER_IMPORT_SANDBOX_APPLY",
			"journal_reason":      supplierImportReasonBulk,
		})
	}
}

func supplierIDFromContext(r *http.Request) (string, error) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims == nil {
		return "", errors.New("missing claims")
	}
	supplierID := strings.TrimSpace(claims.ResolveSupplierID())
	if supplierID == "" {
		return "", errors.New("missing supplier scope")
	}
	return supplierID, nil
}

func createSupplierImportUploadTicket(supplierID string, sessionID string, fileName string, fileSizeBytes int64) (uploadURL string, objectPath string, contentType string, err error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))), ".")
	if ext == "" {
		ext = "xlsx"
	}
	mime, ok := supplierImportAllowedUploadExtensions[ext]
	if !ok {
		return "", "", "", errors.New("unsupported file extension: use xlsx|xls")
	}
	if fileSizeBytes <= 0 || fileSizeBytes > supplierImportMaxUploadSize {
		return "", "", "", errors.New("file exceeds max size (50MB)")
	}

	now := time.Now().UTC()
	objectPath = supplierImportObjectPath(supplierID, sessionID)
	uploadURL = fmt.Sprintf("https://local.invalid/upload/%s", objectPath)
	contentType = mime

	if storage.Client != nil && storage.BucketName != "" {
		opts := &gcs.SignedURLOptions{
			Scheme:      gcs.SigningSchemeV4,
			Method:      "PUT",
			Expires:     now.Add(15 * time.Minute),
			ContentType: mime,
			Headers:     []string{fmt.Sprintf("content-length:%d", fileSizeBytes)},
		}
		signedURL, signErr := storage.Client.Bucket(storage.BucketName).SignedURL(objectPath, opts)
		if signErr != nil {
			return "", "", "", fmt.Errorf("failed to generate signed upload url: %w", signErr)
		}
		uploadURL = signedURL
	}

	return uploadURL, objectPath, contentType, nil
}

func supplierImportObjectPath(supplierID string, sessionID string) string {
	return fmt.Sprintf("imports/%s/%s/raw.xlsx", supplierID, sessionID)
}

func normalizeSupplierImportStatus(status string) string {
	return strings.ToUpper(strings.TrimSpace(status))
}

func toNullJSON(raw json.RawMessage) (spanner.NullJSON, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return spanner.NullJSON{}, nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return spanner.NullJSON{}, err
	}
	return spanner.NullJSON{Value: value, Valid: true}, nil
}

func jsonRawOrNull(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func parseSupplierImportPagination(r *http.Request) (int, int) {
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

func writeSupplierImportJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
