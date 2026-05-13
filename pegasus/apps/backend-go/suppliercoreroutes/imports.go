package suppliercoreroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/storage"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	gcs "cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

const (
	supplierImportRoutePrefix   = "/v1/supplier/inventory/imports"
	supplierImportMaxUploadSize = int64(50 * 1024 * 1024) // 50MB
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
	errSupplierImportSessionNotFound = errors.New("supplier import session not found")
	errSupplierImportInvalidStatus   = errors.New("invalid supplier import status")
	errSupplierImportStateConflict   = errors.New("supplier import status transition conflict")
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

// SupplierImportRepository encapsulates supplier-scoped import sandbox persistence.
type SupplierImportRepository struct {
	client *spanner.Client
}

func NewSupplierImportRepository(client *spanner.Client) *SupplierImportRepository {
	return &SupplierImportRepository{client: client}
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

func registerImportRoutes(r chi.Router, d Deps, supplierRole []string, log Middleware, withRegionScope Middleware, idem Middleware) {
	repo := NewSupplierImportRepository(d.Spanner)

	createHandler := withMethodIdempotency(handleCreateSupplierImportSession(repo), idem, http.MethodPost)
	uploadedHandler := withMethodIdempotency(handlePostSupplierImportUploaded(repo), idem, http.MethodPost)
	mappingHandler := withMethodIdempotency(handlePostSupplierImportMapping(repo), idem, http.MethodPost)
	approveHandler := withMethodIdempotency(handlePostSupplierImportApprove(repo), idem, http.MethodPost)

	r.Route("/v1/supplier/inventory/imports", func(imports chi.Router) {
		imports.HandleFunc("/",
			auth.RequireRole(supplierRole, log(withRegionScope(createHandler))))
		imports.HandleFunc("/{id}",
			auth.RequireRole(supplierRole, log(withRegionScope(handleGetSupplierImportSession(repo)))))
		imports.HandleFunc("/{id}/uploaded",
			auth.RequireRole(supplierRole, log(withRegionScope(uploadedHandler))))
		imports.HandleFunc("/{id}/mapping",
			auth.RequireRole(supplierRole, log(withRegionScope(mappingHandler))))
		imports.HandleFunc("/{id}/approve",
			auth.RequireRole(supplierRole, log(withRegionScope(approveHandler))))
	})
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

func writeSupplierImportJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
