package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func importSupplierIDFromContext(r *http.Request) (string, error) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return "", errors.New("missing claims")
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		return "", errors.New("missing supplier scope")
	}
	return supplierID, nil
}

func writeImportJSON(w http.ResponseWriter, statusCode int, payload any) {
	writeJSON(w, statusCode, payload)
}

func handleCreateImportSession(repo *ImportRepository, svc *Service) http.HandlerFunc {
	type request struct {
		FileName      string `json:"file_name"`
		FileSizeBytes int64  `json:"file_size_bytes"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		body, ok := readMutationBody(w, r, 64*1024)
		if !ok {
			return
		}
		var idemKey string
		if svc != nil {
			var handled bool
			idemKey, handled = svc.guardMutationReplay(w, r, body)
			if handled {
				return
			}
		}

		var payload request
		if err := json.Unmarshal(body, &payload); err != nil {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		payload.FileName = strings.TrimSpace(payload.FileName)
		if payload.FileName == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "file_name is required"})
			return
		}
		if payload.FileSizeBytes <= 0 {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "file_size_bytes is required"})
			return
		}
		if payload.FileSizeBytes > supplierImportMaxUploadSize {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "file exceeds max size (50MB)"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sessionID := uuid.NewString()
		uploadURL, gcsPath, contentType, ticketErr := createImportUploadTicket(supplierID, sessionID, payload.FileName, payload.FileSizeBytes)
		if ticketErr != nil {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": ticketErr.Error()})
			return
		}

		session, err := repo.CreateImportSession(ctx, supplierID, sessionID, payload.FileName, "INITIALIZED")
		if err != nil {
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create import session"})
			return
		}

		resp := map[string]any{
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
		}
		respBytes, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(respBytes)
		if svc != nil {
			svc.storeMutationReplay(r.Context(), idemKey, body, http.StatusCreated, respBytes)
		}
	}
}

func handleGetImportSession(repo *ImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		session, err := repo.GetSession(ctx, supplierID, sessionID)
		if err != nil {
			if errors.Is(err, errImportSessionNotFound) {
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
			return
		}

		writeImportJSON(w, http.StatusOK, map[string]any{
			"supplier_id":   session.SupplierID,
			"session_id":    session.SessionID,
			"status":        session.Status,
			"file_name":     session.FileName,
			"gcs_path":      importObjectPath(session.SupplierID, session.SessionID, session.FileName),
			"total_rows":    session.TotalRows,
			"error_summary": importJSONRawOrNull(session.ErrorSummary),
			"created_at":    session.CreatedAt.Format(time.RFC3339),
			"updated_at":    session.UpdatedAtRFC3339,
		})
	}
}

func handleGetImportRows(repo *ImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		limit, offset := parseImportPagination(r)
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rows, hasMore, err := repo.ListRows(ctx, supplierID, sessionID, limit, offset)
		if err != nil {
			if errors.Is(err, errImportSessionNotFound) {
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load staged rows"})
			return
		}

		writeImportJSON(w, http.StatusOK, map[string]any{
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

func handleGetImportMapping(repo *ImportRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		mapping, err := repo.GetMapping(ctx, supplierID, sessionID)
		if err != nil {
			if errors.Is(err, errImportSessionNotFound) {
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load mapping"})
			return
		}

		createdAt := ""
		if !mapping.CreatedAt.IsZero() {
			createdAt = mapping.CreatedAt.Format(time.RFC3339)
		}

		writeImportJSON(w, http.StatusOK, map[string]any{
			"supplier_id":  supplierID,
			"session_id":   sessionID,
			"mapping_json": importJSONRawOrNull(mapping.Mapping),
			"created_at":   createdAt,
			"updated_at":   mapping.UpdatedAtRFC,
		})
	}
}

func loadImportWarehouseIDs(ctx context.Context, repo *ImportRepository, svc *Service, supplierID string) (map[string]struct{}, error) {
	if svc != nil && svc.repo != nil {
		topology, err := svc.repo.GetTopology(ctx, supplierID)
		if err != nil {
			return nil, err
		}
		return warehouseIDSet(topology), nil
	}
	return repo.LoadWarehouseIDSet(ctx, supplierID)
}

func handlePostImportUploaded(repo *ImportRepository, svc *Service) http.HandlerFunc {
	type request struct {
		GCSPath string `json:"gcs_path"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		session, sessErr := repo.GetSession(r.Context(), supplierID, sessionID)
		if sessErr != nil {
			if errors.Is(sessErr, errImportSessionNotFound) {
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
			return
		}
		expectedGCSPath := importObjectPath(supplierID, sessionID, session.FileName)

		var payload request
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
				writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
		}
		payload.GCSPath = strings.TrimSpace(payload.GCSPath)
		if payload.GCSPath != "" && payload.GCSPath != expectedGCSPath {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "gcs_path mismatch"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := repo.MarkSessionUploadedAndEmit(ctx, supplierID, sessionID, expectedGCSPath); err != nil {
			switch {
			case errors.Is(err, errImportSessionNotFound):
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case errors.Is(err, errImportStateConflict):
				writeImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for uploaded transition"})
			default:
				writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to mark session uploaded"})
			}
			return
		}

		// Sandbox/emulator: discover immediately when the object is already on disk.
		// Worker remains the GCS path; MarkSessionDiscovering is idempotent.
		if opener, oerr := NewImportObjectOpenerFromEnv(r.Context()); oerr == nil && opener != nil && opener.localRoot != "" {
			discoverCtx, discoverCancel := context.WithTimeout(r.Context(), 30*time.Second)
			warehouses, whErr := loadImportWarehouseIDs(discoverCtx, repo, svc, supplierID)
			if whErr != nil {
				slog.ErrorContext(r.Context(), "import local discover warehouse load failed",
					"err", whErr, "session_id", sessionID, "supplier_id", supplierID)
			}
			if procErr := repo.ProcessImportUploaded(discoverCtx, opener, supplierID, sessionID, expectedGCSPath, warehouses); procErr != nil {
				slog.ErrorContext(r.Context(), "import local discover after uploaded failed",
					"err", procErr, "session_id", sessionID, "supplier_id", supplierID)
			}
			discoverCancel()
			_ = opener.Close()
		}

		writeImportJSON(w, http.StatusAccepted, map[string]any{
			"session_id": sessionID,
			"status":     "UPLOADED",
			"gcs_path":   expectedGCSPath,
			"event_type": events.EventInventoryImportUploaded,
			"topic":      events.TopicInventoryImportEvents,
		})
	}
}

func handlePostImportIngest(repo *ImportRepository, svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		body, ok := readMutationBody(w, r, supplierImportMaxUploadSize)
		if !ok {
			return
		}
		if len(body) == 0 {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "csv body required"})
			return
		}
		var idemKey string
		if svc != nil {
			var handled bool
			idemKey, handled = svc.guardMutationReplay(w, r, body)
			if handled {
				return
			}
		}

		delimiter := ','
		if strings.EqualFold(strings.TrimSpace(r.Header.Get("Content-Type")), "text/tab-separated-values") {
			delimiter = '\t'
		} else if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "tab-separated") {
			delimiter = '\t'
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		session, err := repo.GetSession(ctx, supplierID, sessionID)
		if err != nil {
			if errors.Is(err, errImportSessionNotFound) {
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
			return
		}
		status := normalizeImportStatus(session.Status)
		if status != "INITIALIZED" && status != "UPLOADED" {
			writeImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for ingest"})
			return
		}

		warehouseIDs, topoErr := loadImportWarehouseIDs(ctx, repo, svc, supplierID)
		if topoErr != nil {
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_supplier_topology_failed"})
			return
		}

		if status == "INITIALIZED" {
			if err := repo.UpdateSessionStatus(ctx, supplierID, sessionID, "UPLOADED"); err != nil && !errors.Is(err, errImportStateConflict) {
				writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to advance session"})
				return
			}
		}

		summary, err := repo.IngestImportBytes(ctx, supplierID, sessionID, body, delimiter, warehouseIDs)
		if err != nil {
			switch {
			case errors.Is(err, errImportSessionNotFound):
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case strings.Contains(err.Error(), "import_empty"), strings.Contains(err.Error(), "import_too_many_rows"), strings.Contains(err.Error(), "no headers"):
				writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			default:
				writeImportJSON(w, http.StatusInternalServerError, map[string]string{
					"error":  "failed to stage rows",
					"detail": err.Error(),
				})
			}
			return
		}

		if svc != nil && svc.cache != nil {
			svc.cache.Invalidate(ctx, "supplier:inventory:"+supplierID)
		}

		resp := map[string]any{
			"session_id":         sessionID,
			"status":             summary.Status,
			"rows_staged":        summary.RowsStaged,
			"suggested_mappings": summary.SuggestedMappings,
			"valid_rows":         summary.ValidRows,
			"invalid_rows":       summary.InvalidRows,
			"discovery_model":    summary.DiscoveryModel,
			"source":             "SUPPLIER_IMPORT_SYNC_INGEST",
		}
		respBytes, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBytes)
		if svc != nil {
			svc.storeMutationReplay(r.Context(), idemKey, body, http.StatusOK, respBytes)
		}
	}
}

func handlePostImportMapping(repo *ImportRepository) http.HandlerFunc {
	type wrapped struct {
		MappingJSON json.RawMessage `json:"mapping_json"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		body = []byte(strings.TrimSpace(string(body)))
		if len(body) == 0 {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "mapping payload is required"})
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
			if errors.Is(err, errImportSessionNotFound) {
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
				return
			}
			writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save mapping"})
			return
		}

		writeImportJSON(w, http.StatusAccepted, map[string]any{
			"session_id": sessionID,
			"status":     "MAPPING_REQUIRED",
		})
	}
}

func handlePostImportApprove(repo *ImportRepository, svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}

		body, ok := readMutationBody(w, r, 4*1024)
		if !ok {
			return
		}
		var idemKey string
		if svc != nil {
			var handled bool
			idemKey, handled = svc.guardMutationReplay(w, r, body)
			if handled {
				return
			}
		}

		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := repo.UpdateSessionStatus(ctx, supplierID, sessionID, "APPROVED"); err != nil {
			switch {
			case errors.Is(err, errImportSessionNotFound):
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case errors.Is(err, errImportStateConflict):
				writeImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for approve"})
			case errors.Is(err, errImportInvalidStatus):
				writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status transition"})
			default:
				writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to approve session"})
			}
			return
		}

		resp := map[string]any{
			"session_id": sessionID,
			"status":     "APPROVED",
			"next_phase": "apply",
		}
		respBytes, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(respBytes)
		if svc != nil {
			svc.storeMutationReplay(r.Context(), idemKey, body, http.StatusAccepted, respBytes)
		}
	}
}

func handlePostImportApply(repo *ImportRepository, svc *Service, supplierHub, warehouseHub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeImportJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		supplierID, err := importSupplierIDFromContext(r)
		if err != nil {
			writeImportJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
		if sessionID == "" {
			writeImportJSON(w, http.StatusBadRequest, map[string]string{"error": "session id is required"})
			return
		}
		if repo.client == nil {
			writeImportJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "import storage unavailable"})
			return
		}

		body, ok := readMutationBody(w, r, 4*1024)
		if !ok {
			return
		}
		var idemKey string
		if svc != nil {
			var handled bool
			idemKey, handled = svc.guardMutationReplay(w, r, body)
			if handled {
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		summary, err := repo.ApplyImportSession(ctx, supplierID, sessionID)
		if err != nil {
			slog.Error("apply import session", "supplier_id", supplierID, "session_id", sessionID, "err", err)
			switch {
			case errors.Is(err, errImportSessionNotFound):
				writeImportJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			case errors.Is(err, errImportAccessDenied):
				writeImportJSON(w, http.StatusForbidden, map[string]string{"error": "session does not belong to supplier"})
			case errors.Is(err, errImportNoApplicableRows):
				writeImportJSON(w, http.StatusConflict, map[string]string{"error": "no_applicable_rows"})
			case errors.Is(err, errImportStateConflict):
				writeImportJSON(w, http.StatusConflict, map[string]string{"error": "session not ready for apply"})
			default:
				writeImportJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to apply session"})
			}
			return
		}

		if !summary.Idempotent {
			broadcastImportInventorySyncComplete(ctx, supplierID, summary, supplierHub, warehouseHub)
			if svc != nil && svc.cache != nil {
				svc.cache.Invalidate(ctx, "supplier:inventory:"+supplierID)
			}
		}

		resp := map[string]any{
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
		}
		respBytes, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBytes)
		if svc != nil {
			svc.storeMutationReplay(r.Context(), idemKey, body, http.StatusOK, respBytes)
		}
	}
}

type importInventorySyncFrame struct {
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

func broadcastImportInventorySyncComplete(ctx context.Context, supplierID string, summary ImportApplySummary, supplierHub, warehouseHub *ws.Hub) {
	timestamp := strings.TrimSpace(summary.Timestamp)
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	baseFrame := importInventorySyncFrame{
		Type:               events.EventInventorySyncComplete,
		SupplierID:         supplierID,
		SessionID:          summary.SessionID,
		RowsAffected:       summary.AppliedRows,
		AffectedWarehouses: summary.AffectedWarehouses,
		ProductIDs:         summary.AppliedProductIDs,
		Source:             "SUPPLIER_IMPORT_SANDBOX_APPLY",
		Timestamp:          timestamp,
	}

	payload, err := json.Marshal(baseFrame)
	if err != nil {
		return
	}

	if supplierHub != nil {
		supplierHub.Broadcast(ctx, "supplier:"+supplierID, payload)
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
		whPayload, marshalErr := json.Marshal(frame)
		if marshalErr != nil {
			continue
		}
		warehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, whPayload)
	}
}

func createImportUploadTicket(supplierID, sessionID, fileName string, fileSizeBytes int64) (uploadURL, objectPath, contentType string, err error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))), ".")
	if ext == "" {
		ext = "csv"
	}
	mime, ok := supplierImportAllowedUploadExtensions[ext]
	if !ok {
		return "", "", "", errors.New("unsupported file extension: use xlsx|xls|csv|tsv")
	}
	if fileSizeBytes <= 0 || fileSizeBytes > supplierImportMaxUploadSize {
		return "", "", "", errors.New("file exceeds max size (50MB)")
	}

	objectPath = importObjectPath(supplierID, sessionID, fileName)
	uploadURL = fmt.Sprintf("https://local.invalid/upload/%s", objectPath)
	contentType = mime

	if storage.Client != nil && storage.BucketName != "" {
		opts := &gcs.SignedURLOptions{
			Scheme:      gcs.SigningSchemeV4,
			Method:      "PUT",
			Expires:     time.Now().UTC().Add(15 * time.Minute),
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

func importObjectPath(supplierID, sessionID, fileName string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))), ".")
	if ext == "" {
		ext = "csv"
	}
	return fmt.Sprintf("imports/%s/%s/raw.%s", supplierID, sessionID, ext)
}
