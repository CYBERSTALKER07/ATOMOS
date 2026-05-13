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

	"backend-go/auth"
	"backend-go/storage"

	"cloud.google.com/go/spanner"
	gcs "cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

const inventoryImportBasePath = "/v1/supplier/inventory/import"

const (
	defaultImportLimit = 25
	maxImportLimit     = 500
)

var allowedImportUploadExtensions = map[string]string{
	"csv":  "text/csv",
	"tsv":  "text/tab-separated-values",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"xls":  "application/vnd.ms-excel",
	"json": "application/json",
}

var (
	errImportSessionNotFound     = errors.New("inventory import session not found")
	errImportSessionAccessDenied = errors.New("inventory import session access denied")
	errImportStateConflict       = errors.New("inventory import session state conflict")
)

// ImportSessionState represents the lifecycle state of a staged inventory import.
type ImportSessionState string

const (
	ImportSessionStateUploaded        ImportSessionState = "UPLOADED"
	ImportSessionStateDiscovering     ImportSessionState = "DISCOVERING"
	ImportSessionStateMappingRequired ImportSessionState = "MAPPING_REQUIRED"
	ImportSessionStateReadyForReview  ImportSessionState = "READY_FOR_REVIEW"
	ImportSessionStateApproved        ImportSessionState = "APPROVED"
	ImportSessionStateApplying        ImportSessionState = "APPLYING"
	ImportSessionStateApplied         ImportSessionState = "APPLIED"
	ImportSessionStateFailed          ImportSessionState = "FAILED"
	ImportSessionStateExpired         ImportSessionState = "EXPIRED"
)

// Valid reports whether the session state is part of the locked import contract.
func (state ImportSessionState) Valid() bool {
	switch state {
	case ImportSessionStateUploaded,
		ImportSessionStateDiscovering,
		ImportSessionStateMappingRequired,
		ImportSessionStateReadyForReview,
		ImportSessionStateApproved,
		ImportSessionStateApplying,
		ImportSessionStateApplied,
		ImportSessionStateFailed,
		ImportSessionStateExpired:
		return true
	default:
		return false
	}
}

// ImportRowStatus represents the per-row state inside a staged import session.
type ImportRowStatus string

const (
	ImportRowStatusUnmapped       ImportRowStatus = "UNMAPPED"
	ImportRowStatusMappedExisting ImportRowStatus = "MAPPED_EXISTING"
	ImportRowStatusPendingCreate  ImportRowStatus = "PENDING_CREATION"
	ImportRowStatusInvalid        ImportRowStatus = "INVALID"
	ImportRowStatusReadyForReview ImportRowStatus = "READY_FOR_REVIEW"
	ImportRowStatusApproved       ImportRowStatus = "APPROVED"
	ImportRowStatusApplied        ImportRowStatus = "APPLIED"
	ImportRowStatusFailed         ImportRowStatus = "FAILED"
)

// Valid reports whether the row status is part of the locked import contract.
func (status ImportRowStatus) Valid() bool {
	switch status {
	case ImportRowStatusUnmapped,
		ImportRowStatusMappedExisting,
		ImportRowStatusPendingCreate,
		ImportRowStatusInvalid,
		ImportRowStatusReadyForReview,
		ImportRowStatusApproved,
		ImportRowStatusApplied,
		ImportRowStatusFailed:
		return true
	default:
		return false
	}
}

// MappingSuggestion captures a proposed spreadsheet-column-to-field mapping.
type MappingSuggestion struct {
	SourceColumn string  `json:"source_column"`
	TargetField  string  `json:"target_field"`
	Confidence   float64 `json:"confidence"`
}

// ImportSession is the stable contract for a staged inventory import session.
type ImportSession struct {
	SessionID           string             `json:"session_id"`
	SupplierID          string             `json:"supplier_id,omitempty"`
	WarehouseID         string             `json:"warehouse_id,omitempty"`
	FileName            string             `json:"file_name,omitempty"`
	ContentType         string             `json:"content_type,omitempty"`
	ObjectPath          string             `json:"object_path,omitempty"`
	State               ImportSessionState `json:"state"`
	TotalRows           int64              `json:"total_rows"`
	ProcessedRows       int64              `json:"processed_rows"`
	FailedRows          int64              `json:"failed_rows"`
	PendingCreationRows int64              `json:"pending_creation_rows"`
	CreatedAt           string             `json:"created_at,omitempty"`
	UpdatedAt           string             `json:"updated_at,omitempty"`
}

// StagedImportRow is the stable contract for one row in a staged import session.
type StagedImportRow struct {
	RowID            string              `json:"row_id"`
	SessionID        string              `json:"session_id"`
	RowNumber        int64               `json:"row_number"`
	Status           ImportRowStatus     `json:"status"`
	Source           map[string]string   `json:"source,omitempty"`
	SkuID            string              `json:"sku_id,omitempty"`
	ProductName      string              `json:"product_name,omitempty"`
	CategoryID       string              `json:"category_id,omitempty"`
	WarehouseID      string              `json:"warehouse_id,omitempty"`
	Currency         string              `json:"currency,omitempty"`
	BasePrice        int64               `json:"base_price,omitempty"`
	QuantityDelta    int64               `json:"quantity_delta,omitempty"`
	MinimumOrderQty  int64               `json:"minimum_order_qty,omitempty"`
	StepSize         int64               `json:"step_size,omitempty"`
	VolumetricUnit   float64             `json:"volumetric_unit,omitempty"`
	LengthCM         *float64            `json:"length_cm,omitempty"`
	WidthCM          *float64            `json:"width_cm,omitempty"`
	HeightCM         *float64            `json:"height_cm,omitempty"`
	Errors           []string            `json:"errors,omitempty"`
	Suggestions      []MappingSuggestion `json:"suggestions,omitempty"`
	ValidationStatus string              `json:"validation_status,omitempty"`
}

// ImportSessionCreateRequest captures the session-create input after object upload.
type ImportSessionCreateRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	ObjectPath  string `json:"object_path"`
	WarehouseID string `json:"warehouse_id,omitempty"`
}

// MappingUpdateRequest captures row-level mapping edits before approval.
type MappingUpdateRequest struct {
	Rows []StagedImportRow `json:"rows"`
}

// ImportApplySummary captures the terminal summary for an applied import.
type ImportApplySummary struct {
	SessionID          string `json:"session_id"`
	AppliedRows        int64  `json:"applied_rows"`
	CreatedProducts    int64  `json:"created_products"`
	FailedRows         int64  `json:"failed_rows"`
	AffectedWarehouses int64  `json:"affected_warehouses"`
	State              string `json:"state"`
}

// ImportProgressFrame is the websocket-safe progress envelope for supplier import updates.
type ImportProgressFrame struct {
	Type          string             `json:"type"`
	SessionID     string             `json:"session_id"`
	State         ImportSessionState `json:"state"`
	ProcessedRows int64              `json:"processed_rows"`
	TotalRows     int64              `json:"total_rows"`
	FailedRows    int64              `json:"failed_rows"`
	WarehouseID   string             `json:"warehouse_id,omitempty"`
}

// HandleInventoryImports locks the Phase 1 supplier inventory import contract.
func HandleInventoryImports(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		supplierID := claims.ResolveSupplierID()
		suffix := strings.TrimPrefix(r.URL.Path, inventoryImportBasePath)

		switch {
		case suffix == "" || suffix == "/":
			handleInventoryImportCollection(w, r, client, supplierID)
		case suffix == "/upload-ticket":
			handleInventoryImportUploadTicket(w, r, supplierID)
		default:
			handleInventoryImportSessionRoute(w, r, client, supplierID, strings.TrimPrefix(suffix, "/"))
		}
	}
}

func handleInventoryImportCollection(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string) {
	if client == nil {
		writeInventoryImportError(w, http.StatusServiceUnavailable, "inventory import storage unavailable")
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleListInventoryImportSessions(w, r, client, supplierID)
	case http.MethodPost:
		handleCreateInventoryImportSession(w, r, client, supplierID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func handleInventoryImportUploadTicket(w http.ResponseWriter, r *http.Request, supplierID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	ext := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("ext")))
	if ext == "" {
		ext = "csv"
	}
	contentType, ok := allowedImportUploadExtensions[ext]
	if !ok {
		writeInventoryImportError(w, http.StatusBadRequest, "unsupported extension: use csv|tsv|xlsx|xls|json")
		return
	}

	now := time.Now().UTC()
	objectPath := fmt.Sprintf("inventory-import/%s/%d-%s.%s", supplierID, now.UnixNano(), uuid.NewString(), ext)
	uploadURL := fmt.Sprintf("https://local.invalid/upload/%s", objectPath)

	if storage.Client != nil && storage.BucketName != "" {
		opts := &gcs.SignedURLOptions{
			Scheme:      gcs.SigningSchemeV4,
			Method:      "PUT",
			Expires:     now.Add(15 * time.Minute),
			ContentType: contentType,
		}
		signedURL, err := storage.Client.Bucket(storage.BucketName).SignedURL(objectPath, opts)
		if err != nil {
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to generate upload ticket")
			return
		}
		uploadURL = signedURL
	}

	writeInventoryImportJSON(w, http.StatusOK, map[string]interface{}{
		"upload_url":         uploadURL,
		"object_path":        objectPath,
		"content_type":       contentType,
		"expires_in_seconds": int64((15 * time.Minute).Seconds()),
	})
}

func handleInventoryImportSessionRoute(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, suffix string) {
	if client == nil {
		writeInventoryImportError(w, http.StatusServiceUnavailable, "inventory import storage unavailable")
		return
	}

	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeInventoryImportError(w, http.StatusBadRequest, "session_id required")
		return
	}

	sessionID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handleGetInventoryImportSession(w, r, client, supplierID, sessionID)
		return
	}

	action := parts[1]
	switch action {
	case "rows":
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handleListInventoryImportRows(w, r, client, supplierID, sessionID)
	case "mapping":
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handlePatchInventoryImportMapping(w, r, client, supplierID, sessionID)
	case "approve":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handleApproveInventoryImportSession(w, r, client, supplierID, sessionID)
	case "apply":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handleApplyInventoryImportSession(w, r, client, supplierID, sessionID)
	case "status":
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handleGetInventoryImportStatus(w, r, client, supplierID, sessionID)
	default:
		http.NotFound(w, r)
	}
}

func handleListInventoryImportSessions(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	limit, offset := parseImportPagination(r)
	stateFilter := strings.TrimSpace(r.URL.Query().Get("state"))

	params := map[string]interface{}{"supplierId": supplierID}
	query := `SELECT SessionId, SupplierId, COALESCE(WarehouseId, ''), FileName,
	                 COALESCE(ContentType, ''), ObjectPath, State,
	                 TotalRows, ProcessedRows, FailedRows, PendingCreationRows,
	                 CreatedAt, UpdatedAt
	          FROM InventoryImportSessions
	          WHERE SupplierId = @supplierId`
	if stateFilter != "" {
		candidate := ImportSessionState(stateFilter)
		if !candidate.Valid() {
			writeInventoryImportError(w, http.StatusBadRequest, "invalid state filter")
			return
		}
		query += " AND State = @state"
		params["state"] = stateFilter
	}
	query += fmt.Sprintf(" ORDER BY UpdatedAt DESC LIMIT %d OFFSET %d", limit+1, offset)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: query, Params: params})
	defer iter.Stop()

	sessions := make([]ImportSession, 0, limit+1)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to list import sessions")
			return
		}
		session, parseErr := parseImportSessionRow(row)
		if parseErr != nil {
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to parse import session")
			return
		}
		sessions = append(sessions, session)
	}

	hasMore := len(sessions) > limit
	if hasMore {
		sessions = sessions[:limit]
	}

	writeInventoryImportJSON(w, http.StatusOK, map[string]interface{}{
		"data":        sessions,
		"limit":       limit,
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": offset + len(sessions),
	})
}

func handleCreateInventoryImportSession(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string) {
	var req ImportSessionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInventoryImportError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.FileName = strings.TrimSpace(req.FileName)
	req.ContentType = strings.TrimSpace(req.ContentType)
	req.ObjectPath = strings.TrimSpace(req.ObjectPath)
	req.WarehouseID = strings.TrimSpace(req.WarehouseID)
	if req.FileName == "" || req.ObjectPath == "" {
		writeInventoryImportError(w, http.StatusBadRequest, "file_name and object_path are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	scopedWarehouseID := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context()))
	if scopedWarehouseID != "" {
		if req.WarehouseID != "" && req.WarehouseID != scopedWarehouseID {
			writeInventoryImportError(w, http.StatusForbidden, "warehouse scope mismatch")
			return
		}
		req.WarehouseID = scopedWarehouseID
	}

	if req.WarehouseID != "" {
		if err := validateImportWarehouse(ctx, client, supplierID, req.WarehouseID); err != nil {
			if errors.Is(err, errImportSessionAccessDenied) {
				writeInventoryImportError(w, http.StatusForbidden, "warehouse not in supplier scope")
				return
			}
			writeInventoryImportError(w, http.StatusBadRequest, "invalid warehouse_id")
			return
		}
	}

	sessionID := uuid.NewString()
	expiresAt := time.Now().UTC().Add(72 * time.Hour)
	insertCols := []string{
		"SessionId", "SupplierId", "WarehouseId", "FileName", "ContentType", "ObjectPath",
		"State", "TotalRows", "ProcessedRows", "FailedRows", "PendingCreationRows",
		"ExpiresAt", "CreatedAt", "UpdatedAt",
	}
	insertVals := []interface{}{
		sessionID, supplierID, nullableString(req.WarehouseID), req.FileName, nullableString(req.ContentType), req.ObjectPath,
		string(ImportSessionStateUploaded), int64(0), int64(0), int64(0), int64(0),
		expiresAt, spanner.CommitTimestamp, spanner.CommitTimestamp,
	}

	if _, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.Insert("InventoryImportSessions", insertCols, insertVals)})
	}); err != nil {
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to create import session")
		return
	}

	session, err := getImportSession(ctx, client, supplierID, sessionID)
	if err != nil {
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to fetch created session")
		return
	}

	writeInventoryImportJSON(w, http.StatusCreated, session)
}

func handleGetInventoryImportSession(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, sessionID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	session, err := getImportSession(ctx, client, supplierID, sessionID)
	if err != nil {
		if errors.Is(err, errImportSessionNotFound) {
			writeInventoryImportError(w, http.StatusNotFound, "session not found")
			return
		}
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to load import session")
		return
	}

	writeInventoryImportJSON(w, http.StatusOK, session)
}

func handleListInventoryImportRows(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, sessionID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if _, err := getImportSession(ctx, client, supplierID, sessionID); err != nil {
		if errors.Is(err, errImportSessionNotFound) {
			writeInventoryImportError(w, http.StatusNotFound, "session not found")
			return
		}
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to validate session")
		return
	}

	limit, offset := parseImportPagination(r)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	params := map[string]interface{}{"supplierId": supplierID, "sessionId": sessionID}
	query := `SELECT RowId, SessionId, RowNumber, Status,
	                 COALESCE(SourceJson, '{}'), COALESCE(SkuId, ''), COALESCE(ProductName, ''),
	                 COALESCE(CategoryId, ''), COALESCE(WarehouseId, ''), COALESCE(Currency, ''),
	                 IFNULL(BasePrice, 0), IFNULL(QuantityDelta, 0), IFNULL(MinimumOrderQty, 0),
	                 IFNULL(StepSize, 0), IFNULL(VolumetricUnit, 0), LengthCM, WidthCM, HeightCM,
	                 COALESCE(ErrorsJson, '[]'), COALESCE(SuggestionsJson, '[]'), COALESCE(ValidationStatus, '')
	          FROM InventoryImportRows
	          WHERE SessionId = @sessionId AND SupplierId = @supplierId`
	if statusFilter != "" {
		candidate := ImportRowStatus(statusFilter)
		if !candidate.Valid() {
			writeInventoryImportError(w, http.StatusBadRequest, "invalid status filter")
			return
		}
		query += " AND Status = @status"
		params["status"] = statusFilter
	}
	query += fmt.Sprintf(" ORDER BY RowNumber LIMIT %d OFFSET %d", limit+1, offset)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: query, Params: params})
	defer iter.Stop()

	rows := make([]StagedImportRow, 0, limit+1)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to list staged rows")
			return
		}
		parsed, parseErr := parseStagedImportRow(row)
		if parseErr != nil {
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to parse staged row")
			return
		}
		rows = append(rows, parsed)
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	writeInventoryImportJSON(w, http.StatusOK, map[string]interface{}{
		"data":        rows,
		"limit":       limit,
		"offset":      offset,
		"has_more":    hasMore,
		"next_offset": offset + len(rows),
	})
}

func handlePatchInventoryImportMapping(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, sessionID string) {
	var req MappingUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInventoryImportError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Rows) == 0 {
		writeInventoryImportError(w, http.StatusBadRequest, "rows payload cannot be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	scopedWarehouseID := strings.TrimSpace(auth.EffectiveWarehouseID(r.Context()))

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sessionState, sessionWarehouseID, err := readImportSessionState(ctx, txn, supplierID, sessionID)
		if err != nil {
			return err
		}
		if sessionState == ImportSessionStateApproved ||
			sessionState == ImportSessionStateApplying ||
			sessionState == ImportSessionStateApplied ||
			sessionState == ImportSessionStateFailed ||
			sessionState == ImportSessionStateExpired {
			return errImportStateConflict
		}

		mutations := make([]*spanner.Mutation, 0, len(req.Rows))
		for idx, row := range req.Rows {
			rowID := strings.TrimSpace(row.RowID)
			if rowID == "" {
				rowID = uuid.NewString()
			}
			rowNumber := row.RowNumber
			if rowNumber <= 0 {
				rowNumber = int64(idx + 1)
			}
			rowWarehouseID := strings.TrimSpace(row.WarehouseID)
			if rowWarehouseID == "" {
				rowWarehouseID = sessionWarehouseID
			}
			if scopedWarehouseID != "" {
				if rowWarehouseID == "" {
					rowWarehouseID = scopedWarehouseID
				} else if rowWarehouseID != scopedWarehouseID {
					return errImportSessionAccessDenied
				}
			}

			status := normalizeImportRowStatus(row)
			sourceJSON := encodeMapJSON(row.Source)
			errorsJSON := encodeStringSliceJSON(row.Errors)
			suggestionsJSON := encodeSuggestionsJSON(row.Suggestions)

			mutations = append(mutations, spanner.InsertOrUpdate("InventoryImportRows",
				[]string{
					"SessionId", "RowId", "SupplierId", "WarehouseId", "RowNumber", "Status",
					"SourceJson", "SkuId", "ProductName", "CategoryId", "Currency", "BasePrice",
					"QuantityDelta", "MinimumOrderQty", "StepSize", "VolumetricUnit",
					"LengthCM", "WidthCM", "HeightCM", "ErrorsJson", "SuggestionsJson",
					"ValidationStatus", "CreatedAt", "UpdatedAt",
				},
				[]interface{}{
					sessionID, rowID, supplierID, nullableString(rowWarehouseID), rowNumber, string(status),
					sourceJSON, nullableString(strings.TrimSpace(row.SkuID)), nullableString(strings.TrimSpace(row.ProductName)),
					nullableString(strings.TrimSpace(row.CategoryID)), nullableString(strings.TrimSpace(row.Currency)), nullableInt64(row.BasePrice),
					row.QuantityDelta, nullableInt64(row.MinimumOrderQty), nullableInt64(row.StepSize), nullableFloat64(row.VolumetricUnit),
					nullableFloat64Ptr(row.LengthCM), nullableFloat64Ptr(row.WidthCM), nullableFloat64Ptr(row.HeightCM),
					errorsJSON, suggestionsJSON, nullableString(strings.TrimSpace(row.ValidationStatus)),
					spanner.CommitTimestamp, spanner.CommitTimestamp,
				},
			))
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer staged row mappings: %w", err)
		}

		totalRows, unmappedRows, failedRows, pendingCreationRows, processedRows, err := aggregateImportRowCounters(ctx, txn, sessionID)
		if err != nil {
			return err
		}

		nextState := deriveSessionStateFromCounters(totalRows, unmappedRows)
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.Update("InventoryImportSessions",
			[]string{"SessionId", "State", "TotalRows", "ProcessedRows", "FailedRows", "PendingCreationRows", "UpdatedAt"},
			[]interface{}{sessionID, string(nextState), totalRows, processedRows, failedRows, pendingCreationRows, spanner.CommitTimestamp},
		)}); err != nil {
			return fmt.Errorf("buffer import session mapping update: %w", err)
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errImportSessionNotFound):
			writeInventoryImportError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, errImportSessionAccessDenied):
			writeInventoryImportError(w, http.StatusForbidden, "warehouse/session scope mismatch")
		case errors.Is(err, errImportStateConflict):
			writeInventoryImportError(w, http.StatusConflict, "session state does not allow mapping updates")
		default:
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to persist mapping updates")
		}
		return
	}

	session, err := getImportSession(ctx, client, supplierID, sessionID)
	if err != nil {
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to load updated session")
		return
	}

	writeInventoryImportJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "MAPPING_UPDATED",
		"session": session,
	})
}

func handleApproveInventoryImportSession(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, sessionID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sessionState, _, err := readImportSessionState(ctx, txn, supplierID, sessionID)
		if err != nil {
			return err
		}
		if sessionState == ImportSessionStateApplied ||
			sessionState == ImportSessionStateApplying ||
			sessionState == ImportSessionStateFailed ||
			sessionState == ImportSessionStateExpired {
			return errImportStateConflict
		}

		totalRows, unmappedRows, failedRows, pendingCreationRows, processedRows, err := aggregateImportRowCounters(ctx, txn, sessionID)
		if err != nil {
			return err
		}
		if totalRows == 0 {
			return errImportStateConflict
		}
		if unmappedRows > 0 {
			return errImportStateConflict
		}

		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT RowId, Status FROM InventoryImportRows WHERE SessionId = @sessionId`,
			Params: map[string]interface{}{"sessionId": sessionID},
		})
		defer iter.Stop()

		mutations := make([]*spanner.Mutation, 0, totalRows+1)
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("list staged rows for approval: %w", nextErr)
			}
			var rowID string
			var statusRaw string
			if err := row.Columns(&rowID, &statusRaw); err != nil {
				return fmt.Errorf("parse staged row for approval: %w", err)
			}
			status := ImportRowStatus(statusRaw)
			if status == ImportRowStatusMappedExisting ||
				status == ImportRowStatusPendingCreate ||
				status == ImportRowStatusReadyForReview {
				mutations = append(mutations, spanner.Update("InventoryImportRows",
					[]string{"SessionId", "RowId", "Status", "UpdatedAt"},
					[]interface{}{sessionID, rowID, string(ImportRowStatusApproved), spanner.CommitTimestamp},
				))
			}
		}

		mutations = append(mutations, spanner.Update("InventoryImportSessions",
			[]string{"SessionId", "State", "TotalRows", "ProcessedRows", "FailedRows", "PendingCreationRows", "ApprovedBy", "ApprovedAt", "UpdatedAt"},
			[]interface{}{sessionID, string(ImportSessionStateApproved), totalRows, processedRows, failedRows, pendingCreationRows, supplierID, spanner.CommitTimestamp, spanner.CommitTimestamp},
		))

		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("approve import session: %w", err)
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errImportSessionNotFound):
			writeInventoryImportError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, errImportStateConflict):
			writeInventoryImportError(w, http.StatusConflict, "session not ready for approval")
		default:
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to approve import session")
		}
		return
	}

	session, err := getImportSession(ctx, client, supplierID, sessionID)
	if err != nil {
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to load approved session")
		return
	}

	writeInventoryImportJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "SESSION_APPROVED",
		"session": session,
	})
}

func handleApplyInventoryImportSession(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, sessionID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	summary := ImportApplySummary{SessionID: sessionID, State: string(ImportSessionStateApplied)}
	affectedWarehouse := map[string]struct{}{}

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sessionState, sessionWarehouseID, err := readImportSessionState(ctx, txn, supplierID, sessionID)
		if err != nil {
			return err
		}
		if sessionState != ImportSessionStateApproved {
			return errImportStateConflict
		}

		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT RowId, RowNumber, Status, COALESCE(SkuId, ''), COALESCE(ProductName, ''),
			             COALESCE(CategoryId, ''), COALESCE(WarehouseId, ''), IFNULL(BasePrice, 0),
			             IFNULL(QuantityDelta, 0), IFNULL(MinimumOrderQty, 0), IFNULL(StepSize, 0),
			             IFNULL(VolumetricUnit, 0), LengthCM, WidthCM, HeightCM
			      FROM InventoryImportRows
			      WHERE SessionId = @sessionId
			        AND Status IN ('APPROVED', 'READY_FOR_REVIEW', 'MAPPED_EXISTING', 'PENDING_CREATION')
			      ORDER BY RowNumber`,
			Params: map[string]interface{}{"sessionId": sessionID},
		})
		defer iter.Stop()

		mutations := make([]*spanner.Mutation, 0)
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("load staged rows for apply: %w", nextErr)
			}

			var rowID string
			var rowNumber int64
			var statusRaw string
			var skuID string
			var productName string
			var categoryID string
			var rowWarehouseID string
			var basePrice int64
			var quantityDelta int64
			var minimumOrderQty int64
			var stepSize int64
			var volumetricUnit float64
			var lengthCM spanner.NullFloat64
			var widthCM spanner.NullFloat64
			var heightCM spanner.NullFloat64

			if err := row.Columns(
				&rowID,
				&rowNumber,
				&statusRaw,
				&skuID,
				&productName,
				&categoryID,
				&rowWarehouseID,
				&basePrice,
				&quantityDelta,
				&minimumOrderQty,
				&stepSize,
				&volumetricUnit,
				&lengthCM,
				&widthCM,
				&heightCM,
			); err != nil {
				return fmt.Errorf("parse staged row for apply: %w", err)
			}

			rowStatus := ImportRowStatus(statusRaw)
			targetWarehouseID := strings.TrimSpace(rowWarehouseID)
			if targetWarehouseID == "" {
				targetWarehouseID = sessionWarehouseID
			}

			markFailed := func(reason string) {
				summary.FailedRows++
				mutations = append(mutations, spanner.Update("InventoryImportRows",
					[]string{"SessionId", "RowId", "Status", "ErrorsJson", "ValidationStatus", "UpdatedAt"},
					[]interface{}{sessionID, rowID, string(ImportRowStatusFailed), encodeStringSliceJSON([]string{reason}), "FAILED", spanner.CommitTimestamp},
				))
			}

			targetSKU := strings.TrimSpace(skuID)
			if targetSKU == "" && rowStatus != ImportRowStatusPendingCreate {
				markFailed("sku_id is required for mapped rows")
				continue
			}

			if targetSKU == "" {
				targetSKU = uuid.NewString()
				if strings.TrimSpace(productName) == "" {
					productName = fmt.Sprintf("Imported SKU %d", rowNumber)
				}
				if basePrice <= 0 {
					basePrice = 1
				}
				if stepSize <= 0 {
					stepSize = 1
				}
				if minimumOrderQty <= 0 {
					minimumOrderQty = stepSize
				}
				if minimumOrderQty < stepSize {
					minimumOrderQty = stepSize
				}
				if volumetricUnit <= 0 && lengthCM.Valid && widthCM.Valid && heightCM.Valid {
					computed := CalculateVU(lengthCM.Float64, widthCM.Float64, heightCM.Float64)
					if computed > 0 {
						volumetricUnit = computed
					}
				}
				if volumetricUnit <= 0 {
					volumetricUnit = 1.0
				}

				productCols := []string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl", "CategoryId", "SellByBlock", "UnitsPerBlock", "BasePrice", "VolumetricUnit", "MinimumOrderQty", "StepSize", "IsActive", "CreatedAt"}
				productVals := []interface{}{targetSKU, supplierID, productName, "", "", nullableString(strings.TrimSpace(categoryID)), false, int64(1), basePrice, volumetricUnit, minimumOrderQty, stepSize, true, spanner.CommitTimestamp}
				if lengthCM.Valid && widthCM.Valid && heightCM.Valid {
					productCols = append(productCols, "LengthCM", "WidthCM", "HeightCM")
					productVals = append(productVals, lengthCM.Float64, widthCM.Float64, heightCM.Float64)
				}
				mutations = append(mutations, spanner.Insert("SupplierProducts", productCols, productVals))
				summary.CreatedProducts++
			} else {
				ownerRow, ownerErr := txn.ReadRow(ctx, "SupplierProducts", spanner.Key{targetSKU}, []string{"SupplierId"})
				if ownerErr != nil {
					if spanner.ErrCode(ownerErr) == codes.NotFound {
						markFailed("sku_id not found in supplier catalog")
						continue
					}
					return fmt.Errorf("read supplier product during apply: %w", ownerErr)
				}
				var ownerSupplierID string
				if err := ownerRow.Columns(&ownerSupplierID); err != nil {
					return fmt.Errorf("parse supplier product owner during apply: %w", err)
				}
				if ownerSupplierID != supplierID {
					markFailed("sku_id belongs to another supplier")
					continue
				}
			}

			legacyQty := int64(0)
			legacyRow, legacyErr := txn.ReadRow(ctx, "SupplierInventory", spanner.Key{targetSKU}, []string{"SupplierId", "QuantityAvailable"})
			if legacyErr == nil {
				var ownerSupplierID string
				if err := legacyRow.Columns(&ownerSupplierID, &legacyQty); err != nil {
					return fmt.Errorf("parse legacy inventory during import apply: %w", err)
				}
				if ownerSupplierID != supplierID {
					markFailed("inventory row belongs to another supplier")
					continue
				}
			} else if spanner.ErrCode(legacyErr) != codes.NotFound {
				return fmt.Errorf("read legacy inventory during import apply: %w", legacyErr)
			}

			newLegacyQty := legacyQty + quantityDelta
			if newLegacyQty < 0 {
				markFailed("quantity_delta would make supplier inventory negative")
				continue
			}

			mutations = append(mutations, spanner.InsertOrUpdate("SupplierInventory",
				[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{targetSKU, supplierID, newLegacyQty, spanner.CommitTimestamp},
			))

			if targetWarehouseID != "" {
				v2Qty := int64(0)
				v2Row, v2Err := txn.ReadRow(ctx, "SupplierInventoryV2", spanner.Key{supplierID, targetWarehouseID, targetSKU}, []string{"QuantityAvailable"})
				if v2Err == nil {
					if err := v2Row.Columns(&v2Qty); err != nil {
						return fmt.Errorf("parse inventory v2 during import apply: %w", err)
					}
				} else if spanner.ErrCode(v2Err) != codes.NotFound {
					return fmt.Errorf("read inventory v2 during import apply: %w", v2Err)
				}

				newV2Qty := v2Qty + quantityDelta
				if newV2Qty < 0 {
					markFailed("quantity_delta would make warehouse inventory negative")
					continue
				}

				mutations = append(mutations, spanner.InsertOrUpdate("SupplierInventoryV2",
					[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityAvailable", "UpdatedAt"},
					[]interface{}{supplierID, targetWarehouseID, targetSKU, newV2Qty, spanner.CommitTimestamp},
				))
				affectedWarehouse[targetWarehouseID] = struct{}{}
			}

			if quantityDelta != 0 {
				auditCols := []string{"AuditId", "ProductId", "SupplierId", "AdjustedBy", "PreviousQty", "NewQty", "Delta", "Reason", "AdjustedAt"}
				auditVals := []interface{}{fmt.Sprintf("AUD-%s", uuid.NewString()[:8]), targetSKU, supplierID, supplierID, legacyQty, newLegacyQty, quantityDelta, "CORRECTION", spanner.CommitTimestamp}
				if targetWarehouseID != "" {
					auditCols = append(auditCols, "WarehouseId")
					auditVals = append(auditVals, targetWarehouseID)
				}
				mutations = append(mutations, spanner.Insert("InventoryAuditLog", auditCols, auditVals))
			}

			summary.AppliedRows++
			mutations = append(mutations, spanner.Update("InventoryImportRows",
				[]string{"SessionId", "RowId", "Status", "SkuId", "ErrorsJson", "ValidationStatus", "AppliedAt", "UpdatedAt"},
				[]interface{}{sessionID, rowID, string(ImportRowStatusApplied), targetSKU, "[]", "APPLIED", spanner.CommitTimestamp, spanner.CommitTimestamp},
			))
		}

		if summary.AppliedRows == 0 && summary.FailedRows == 0 {
			return errImportStateConflict
		}

		summary.AffectedWarehouses = int64(len(affectedWarehouse))
		if summary.FailedRows > 0 {
			summary.State = string(ImportSessionStateFailed)
		}

		sessionUpdate := spanner.Update("InventoryImportSessions",
			[]string{"SessionId", "State", "ProcessedRows", "FailedRows", "PendingCreationRows", "ApplyStartedAt", "AppliedAt", "UpdatedAt"},
			[]interface{}{sessionID, summary.State, summary.AppliedRows + summary.FailedRows, summary.FailedRows, int64(0), spanner.CommitTimestamp, spanner.CommitTimestamp, spanner.CommitTimestamp},
		)
		mutations = append(mutations, sessionUpdate)

		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer import apply mutations: %w", err)
		}
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, errImportSessionNotFound):
			writeInventoryImportError(w, http.StatusNotFound, "session not found")
		case errors.Is(err, errImportStateConflict):
			writeInventoryImportError(w, http.StatusConflict, "session not ready for apply")
		default:
			writeInventoryImportError(w, http.StatusInternalServerError, "failed to apply import session")
		}
		return
	}

	writeInventoryImportJSON(w, http.StatusOK, summary)
}

func handleGetInventoryImportStatus(w http.ResponseWriter, r *http.Request, client *spanner.Client, supplierID string, sessionID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	session, err := getImportSession(ctx, client, supplierID, sessionID)
	if err != nil {
		if errors.Is(err, errImportSessionNotFound) {
			writeInventoryImportError(w, http.StatusNotFound, "session not found")
			return
		}
		writeInventoryImportError(w, http.StatusInternalServerError, "failed to load session status")
		return
	}

	writeInventoryImportJSON(w, http.StatusOK, ImportProgressFrame{
		Type:          "INVENTORY_IMPORT_PROGRESS",
		SessionID:     session.SessionID,
		State:         session.State,
		ProcessedRows: session.ProcessedRows,
		TotalRows:     session.TotalRows,
		FailedRows:    session.FailedRows,
		WarehouseID:   session.WarehouseID,
	})
}

func parseImportPagination(r *http.Request) (int, int) {
	limit := defaultImportLimit
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			if parsed > 0 && parsed <= maxImportLimit {
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

func writeInventoryImportJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeInventoryImportError(w http.ResponseWriter, statusCode int, message string) {
	writeInventoryImportJSON(w, statusCode, map[string]string{"error": message})
}

func getImportSession(ctx context.Context, client *spanner.Client, supplierID string, sessionID string) (ImportSession, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, SupplierId, COALESCE(WarehouseId, ''), FileName,
		             COALESCE(ContentType, ''), ObjectPath, State,
		             TotalRows, ProcessedRows, FailedRows, PendingCreationRows,
		             CreatedAt, UpdatedAt
		      FROM InventoryImportSessions
		      WHERE SessionId = @sessionId AND SupplierId = @supplierId`,
		Params: map[string]interface{}{"sessionId": sessionID, "supplierId": supplierID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return ImportSession{}, errImportSessionNotFound
		}
		return ImportSession{}, err
	}
	return parseImportSessionRow(row)
}

func parseImportSessionRow(row *spanner.Row) (ImportSession, error) {
	var session ImportSession
	var stateRaw string
	var createdAt time.Time
	var updatedAt spanner.NullTime

	if err := row.Columns(
		&session.SessionID,
		&session.SupplierID,
		&session.WarehouseID,
		&session.FileName,
		&session.ContentType,
		&session.ObjectPath,
		&stateRaw,
		&session.TotalRows,
		&session.ProcessedRows,
		&session.FailedRows,
		&session.PendingCreationRows,
		&createdAt,
		&updatedAt,
	); err != nil {
		return ImportSession{}, err
	}

	session.State = ImportSessionState(stateRaw)
	if !session.State.Valid() {
		session.State = ImportSessionStateUploaded
	}
	session.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if updatedAt.Valid {
		session.UpdatedAt = updatedAt.Time.UTC().Format(time.RFC3339)
	}
	return session, nil
}

func parseStagedImportRow(row *spanner.Row) (StagedImportRow, error) {
	var parsed StagedImportRow
	var statusRaw string
	var sourceJSON string
	var errorsJSON string
	var suggestionsJSON string
	var lengthCM spanner.NullFloat64
	var widthCM spanner.NullFloat64
	var heightCM spanner.NullFloat64

	if err := row.Columns(
		&parsed.RowID,
		&parsed.SessionID,
		&parsed.RowNumber,
		&statusRaw,
		&sourceJSON,
		&parsed.SkuID,
		&parsed.ProductName,
		&parsed.CategoryID,
		&parsed.WarehouseID,
		&parsed.Currency,
		&parsed.BasePrice,
		&parsed.QuantityDelta,
		&parsed.MinimumOrderQty,
		&parsed.StepSize,
		&parsed.VolumetricUnit,
		&lengthCM,
		&widthCM,
		&heightCM,
		&errorsJSON,
		&suggestionsJSON,
		&parsed.ValidationStatus,
	); err != nil {
		return StagedImportRow{}, err
	}

	parsed.Status = ImportRowStatus(statusRaw)
	if !parsed.Status.Valid() {
		parsed.Status = ImportRowStatusUnmapped
	}
	if lengthCM.Valid {
		value := lengthCM.Float64
		parsed.LengthCM = &value
	}
	if widthCM.Valid {
		value := widthCM.Float64
		parsed.WidthCM = &value
	}
	if heightCM.Valid {
		value := heightCM.Float64
		parsed.HeightCM = &value
	}
	parsed.Source = decodeMapJSON(sourceJSON)
	parsed.Errors = decodeStringSliceJSON(errorsJSON)
	parsed.Suggestions = decodeSuggestionsJSON(suggestionsJSON)

	return parsed, nil
}

func normalizeImportRowStatus(row StagedImportRow) ImportRowStatus {
	if row.Status.Valid() {
		return row.Status
	}
	if strings.TrimSpace(row.SkuID) != "" {
		return ImportRowStatusMappedExisting
	}
	if strings.TrimSpace(row.ProductName) != "" {
		return ImportRowStatusPendingCreate
	}
	return ImportRowStatusUnmapped
}

func readImportSessionState(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, sessionID string) (ImportSessionState, string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, COALESCE(WarehouseId, ''), State
		      FROM InventoryImportSessions
		      WHERE SessionId = @sessionId`,
		Params: map[string]interface{}{"sessionId": sessionID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return "", "", errImportSessionNotFound
		}
		return "", "", err
	}

	var ownerSupplierID string
	var warehouseID string
	var stateRaw string
	if err := row.Columns(&ownerSupplierID, &warehouseID, &stateRaw); err != nil {
		return "", "", err
	}
	if ownerSupplierID != supplierID {
		return "", "", errImportSessionAccessDenied
	}
	state := ImportSessionState(stateRaw)
	if !state.Valid() {
		state = ImportSessionStateUploaded
	}
	return state, warehouseID, nil
}

func aggregateImportRowCounters(ctx context.Context, txn *spanner.ReadWriteTransaction, sessionID string) (int64, int64, int64, int64, int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(1),
		             IFNULL(SUM(CASE WHEN Status = 'UNMAPPED' THEN 1 ELSE 0 END), 0),
		             IFNULL(SUM(CASE WHEN Status IN ('INVALID', 'FAILED') THEN 1 ELSE 0 END), 0),
		             IFNULL(SUM(CASE WHEN Status = 'PENDING_CREATION' THEN 1 ELSE 0 END), 0),
		             IFNULL(SUM(CASE WHEN Status IN ('MAPPED_EXISTING', 'PENDING_CREATION', 'READY_FOR_REVIEW', 'APPROVED', 'APPLIED', 'FAILED', 'INVALID') THEN 1 ELSE 0 END), 0)
		      FROM InventoryImportRows
		      WHERE SessionId = @sessionId`,
		Params: map[string]interface{}{"sessionId": sessionID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	var totalRows int64
	var unmappedRows int64
	var failedRows int64
	var pendingCreationRows int64
	var processedRows int64
	if err := row.Columns(&totalRows, &unmappedRows, &failedRows, &pendingCreationRows, &processedRows); err != nil {
		return 0, 0, 0, 0, 0, err
	}

	return totalRows, unmappedRows, failedRows, pendingCreationRows, processedRows, nil
}

func deriveSessionStateFromCounters(totalRows int64, unmappedRows int64) ImportSessionState {
	if totalRows == 0 {
		return ImportSessionStateUploaded
	}
	if unmappedRows > 0 {
		return ImportSessionStateMappingRequired
	}
	return ImportSessionStateReadyForReview
}

func validateImportWarehouse(ctx context.Context, client *spanner.Client, supplierID string, warehouseID string) error {
	stmt := spanner.Statement{
		SQL:    `SELECT SupplierId FROM Warehouses WHERE WarehouseId = @warehouseId`,
		Params: map[string]interface{}{"warehouseId": warehouseID},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return errImportSessionNotFound
		}
		return err
	}

	var ownerSupplierID string
	if err := row.Columns(&ownerSupplierID); err != nil {
		return err
	}
	if ownerSupplierID != supplierID {
		return errImportSessionAccessDenied
	}
	return nil
}

func encodeMapJSON(source map[string]string) string {
	if len(source) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func decodeMapJSON(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}
	}
	decoded := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return map[string]string{}
	}
	return decoded
}

func encodeStringSliceJSON(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeStringSliceJSON(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	values := []string{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func encodeSuggestionsJSON(values []MappingSuggestion) string {
	if len(values) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeSuggestionsJSON(raw string) []MappingSuggestion {
	if strings.TrimSpace(raw) == "" {
		return []MappingSuggestion{}
	}
	values := []MappingSuggestion{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []MappingSuggestion{}
	}
	return values
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableInt64(value int64) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func nullableFloat64(value float64) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func nullableFloat64Ptr(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
