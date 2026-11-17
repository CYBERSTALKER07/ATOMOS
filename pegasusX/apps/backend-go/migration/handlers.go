package migration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// HandlerDeps configures HTTP handlers for Universal Smart Ingestion endpoints.
type HandlerDeps struct {
	Repo                 Repository
	AllowAuthBypass      bool
	JWTSecret            string
	Now                  func() time.Time
	NewID                func() string
	SignURL              SignURLFunc
	ReadArtifact         ArtifactBytesReader
	Mapper               *AIMapper
	LiveReader           DryRunLiveOverlayReader
	ApplyStore           EntityApplyStore
	OpenOrderReleaser    OpenOrderReleaser
	ProgressEmit         func(sessionID string, processed, total int64, chunkID string)
	SoliqHealthPing      func(ctx context.Context, supplierID string) error
	ReleaseLotsGateCheck func(ctx context.Context, supplierID string) error
}

// Handlers serves project, bundle, artifact, session, and cutover endpoints.
type Handlers struct {
	repo                 Repository
	allowAuthBypass      bool
	localUploadSecret    []byte
	now                  func() time.Time
	newID                func() string
	signURL              SignURLFunc
	readArtifact         ArtifactBytesReader
	mapper               *AIMapper
	liveReader           DryRunLiveOverlayReader
	applyStore           EntityApplyStore
	openOrderReleaser    OpenOrderReleaser
	progressEmit         func(sessionID string, processed, total int64, chunkID string)
	soliqHealthPing      func(ctx context.Context, supplierID string) error
	releaseLotsGateCheck func(ctx context.Context, supplierID string) error
}

// NewHandlers constructs HTTP handlers backed by Repository.
func NewHandlers(d HandlerDeps) *Handlers {
	nowFn := d.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	idFn := d.NewID
	if idFn == nil {
		idFn = func() string { return uuid.NewString() }
	}
	mapper := d.Mapper
	if mapper == nil {
		mapper = NewAIMapper(GeminiMapperConfigFromEnv())
	}
	releaser := d.OpenOrderReleaser
	if releaser == nil {
		if r, ok := d.ApplyStore.(OpenOrderReleaser); ok {
			releaser = r
		}
	}
	return &Handlers{
		repo:                 d.Repo,
		allowAuthBypass:      d.AllowAuthBypass,
		localUploadSecret:    resolveLocalUploadSecret(d.JWTSecret),
		now:                  nowFn,
		newID:                idFn,
		signURL:              d.SignURL,
		readArtifact:         d.ReadArtifact,
		mapper:               mapper,
		liveReader:           d.LiveReader,
		applyStore:           d.ApplyStore,
		openOrderReleaser:    releaser,
		progressEmit:         d.ProgressEmit,
		soliqHealthPing:      d.SoliqHealthPing,
		releaseLotsGateCheck: d.ReleaseLotsGateCheck,
	}
}

type createProjectRequest struct {
	Name               string          `json:"name"`
	Mode               string          `json:"mode,omitempty"`
	DefaultWarehouseID string          `json:"default_warehouse_id,omitempty"`
	Checklist          json.RawMessage `json:"checklist,omitempty"`
}

// HandleCreateProject creates a CUTOVER migration project for the authenticated supplier.
func (h *Handlers) HandleCreateProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeMigrationError(w, http.StatusBadRequest, "name_required", "project name is required")
		return
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = ProjectModeCutover
	}
	if mode != ProjectModeCutover {
		writeMigrationError(w, http.StatusBadRequest, "unsupported_mode", "only CUTOVER mode is supported")
		return
	}

	p, err := h.repo.CreateProject(r.Context(), Project{
		SupplierID:         claims.SupplierID,
		ProjectID:          h.newID(),
		Name:               strings.TrimSpace(req.Name),
		Mode:               mode,
		Status:             ProjectStatusDraft,
		Checklist:          req.Checklist,
		DefaultWarehouseID: strings.TrimSpace(req.DefaultWarehouseID),
		CreatedBy:          auth.ActorLabel(claims),
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusCreated, p)
}

// HandleListProjects lists cutover migration projects for the authenticated supplier.
func (h *Handlers) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "read", "")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	projects, err := h.repo.ListProjects(r.Context(), claims.SupplierID, limit)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

// HandleGetProject returns the hydrated ProjectTree (bundles, artifacts, sessions, DAG status).
func (h *Handlers) HandleGetProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "read", "")
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	if projectID == "" {
		writeMigrationError(w, http.StatusBadRequest, "project_id_required", "projectId path param is required")
		return
	}
	proj, err := h.repo.GetProject(r.Context(), claims.SupplierID, projectID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	bundles, err := h.repo.ListBundlesByProject(r.Context(), claims.SupplierID, projectID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	trees := make([]BundleTree, 0, len(bundles))
	for _, b := range bundles {
		arts, err := h.repo.ListArtifactsByBundle(r.Context(), claims.SupplierID, projectID, b.BundleID)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		trees = append(trees, BundleTree{Bundle: b, Artifacts: arts})
	}
	sessions, err := h.repo.ListSessionsByProject(r.Context(), claims.SupplierID, projectID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	dag, err := EvaluateProjectDAG(sessions)
	if err != nil {
		dag = DAGStatus{Order: TopologicalPackOrder, BlockedBy: map[Pack][]Pack{}}
	}
	writeMigrationJSON(w, http.StatusOK, ProjectTree{
		Project:  proj,
		Bundles:  trees,
		Sessions: sessions,
		DAG:      dag,
	})
}

type declareBundleFile struct {
	FileName      string `json:"file_name"`
	ByteSize      int64  `json:"byte_size"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
}

type createBundleRequest struct {
	Files []declareBundleFile `json:"files"`
}

// HandleCreateBundle declares 1..N files for upload and returns signed GCS upload tickets.
func (h *Handlers) HandleCreateBundle(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	if projectID == "" {
		writeMigrationError(w, http.StatusBadRequest, "project_id_required", "projectId path param is required")
		return
	}
	var req createBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed request body")
		return
	}
	if len(req.Files) == 0 || len(req.Files) > 50 {
		writeMigrationError(w, http.StatusBadRequest, "invalid_file_count", "provide between 1 and 50 files per bundle")
		return
	}

	bundleID := h.newID()
	homeCell := auth.EffectiveHomeCell(claims)
	now := h.now()

	tickets := make([]UploadTicket, 0, len(req.Files))
	artifacts := make([]Artifact, 0, len(req.Files))
	localBaseURL := deriveLocalUploadBaseURL(r)
	for _, f := range req.Files {
		artID := h.newID()
		ticket, err := CreateArtifactUploadTicketWithLocalFallback(homeCell, claims.SupplierID, projectID, bundleID, ArtifactDescriptor{
			ArtifactID:    artID,
			FileName:      f.FileName,
			ByteSize:      f.ByteSize,
			ContentSHA256: strings.TrimSpace(f.ContentSHA256),
		}, now, h.signURL, localBaseURL, h.localUploadSecret)
		if err != nil {
			writeUploadTicketError(w, err)
			return
		}
		tickets = append(tickets, ticket)
		artifacts = append(artifacts, Artifact{
			SupplierID:    claims.SupplierID,
			ProjectID:     projectID,
			BundleID:      bundleID,
			ArtifactID:    artID,
			FileName:      ticket.FileName,
			ByteSize:      f.ByteSize,
			DeclaredMIME:  ticket.ContentType,
			ContentSHA256: strings.TrimSpace(f.ContentSHA256),
			GCSPath:       ticket.GCSPath,
		})
	}

	bundle, storedArtifacts, err := h.repo.CreateBundleWithArtifacts(r.Context(), Bundle{
		SupplierID: claims.SupplierID,
		ProjectID:  projectID,
		BundleID:   bundleID,
		Status:     BundleStatusUploading,
		CreatedBy:  auth.ActorLabel(claims),
	}, artifacts)
	if err != nil {
		writeRepoError(w, err)
		return
	}

	writeMigrationJSON(w, http.StatusCreated, map[string]any{
		"bundle":         bundle,
		"artifacts":      storedArtifacts,
		"upload_tickets": tickets,
	})
}

type bundleUploadedRequest struct {
	Artifacts map[string]UploadedArtifactMeta `json:"artifacts,omitempty"`
}

// HandleBundleUploaded finalizes client GCS upload and emits MIGRATION_BUNDLE_UPLOADED.
func (h *Handlers) HandleBundleUploaded(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	bundleID := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if projectID == "" || bundleID == "" {
		writeMigrationError(w, http.StatusBadRequest, "missing_path_params", "projectId and bundleId are required")
		return
	}
	var req bundleUploadedRequest
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	bundle, err := h.repo.MarkBundleUploadedAndEmit(r.Context(), claims.SupplierID, projectID, bundleID, req.Artifacts)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	syncVal := strings.TrimSpace(r.URL.Query().Get("sync"))
	if strings.EqualFold(syncVal, "true") || syncVal == "1" {
		decision, sortErr := ProcessBundleSortAndStage(r.Context(), h.repo, h.readArtifact, claims.SupplierID, projectID, bundleID, nil, h.newID)
		if sortErr != nil {
			writeRepoError(w, sortErr)
			return
		}
		_, _ = ProposeMappingsForBundle(r.Context(), h.repo, h.mapper, claims.SupplierID, auth.EffectiveHomeCell(claims), projectID, bundleID)
		if latest, lErr := h.repo.ListSessionsByBundle(r.Context(), claims.SupplierID, projectID, bundleID); lErr == nil && len(latest) > 0 {
			decision.Sessions = latest
		}
		bundle.Status = decision.BundleStatus
		writeMigrationJSON(w, http.StatusOK, map[string]any{
			"bundle":   bundle,
			"summary":  decision.Summary,
			"sessions": decision.Sessions,
		})
		return
	}
	writeMigrationJSON(w, http.StatusAccepted, map[string]any{
		"bundle": bundle,
	})
}

// HandleOverrideBundleSort applies manual pack/encoding/composite overrides and re-stages affected child sessions.
func (h *Handlers) HandleOverrideBundleSort(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	projectID := strings.TrimSpace(chi.URLParam(r, "projectId"))
	bundleID := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if projectID == "" || bundleID == "" {
		writeMigrationError(w, http.StatusBadRequest, "missing_path_params", "projectId and bundleId are required")
		return
	}
	var req BundleSortOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed sort override body")
		return
	}
	decision, err := ProcessBundleSortAndStage(r.Context(), h.repo, h.readArtifact, claims.SupplierID, projectID, bundleID, &req, h.newID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	_, _ = ProposeMappingsForBundle(r.Context(), h.repo, h.mapper, claims.SupplierID, auth.EffectiveHomeCell(claims), projectID, bundleID)
	if latest, lErr := h.repo.ListSessionsByBundle(r.Context(), claims.SupplierID, projectID, bundleID); lErr == nil && len(latest) > 0 {
		decision.Sessions = latest
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"bundle_id":     bundleID,
		"bundle_status": decision.BundleStatus,
		"summary":       decision.Summary,
		"sessions":      decision.Sessions,
	})
}

// HandleGetSession returns a single widened SupplierImportSessions record.
func (h *Handlers) HandleGetSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "read", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId is required")
		return
	}
	session, err := h.repo.GetSession(r.Context(), claims.SupplierID, sessionID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, session)
}

// HandleGetSessionRows paginates staged rows for a session.
func (h *Handlers) HandleGetSessionRows(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "read", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId is required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rows, total, err := h.repo.ListStagedRows(r.Context(), claims.SupplierID, sessionID, limit, offset)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"session_id": sessionID,
		"total_rows": total,
		"rows":       rows,
	})
}

func (h *Handlers) authorize(w http.ResponseWriter, r *http.Request, pack, action, targetSupplierID string) (auth.Claims, bool) {
	if h == nil || h.repo == nil {
		writeMigrationError(w, http.StatusServiceUnavailable, "migration_service_unavailable", "migration repository unavailable")
		return auth.Claims{}, false
	}
	claims, err := GuardRequestAuthorization(r.Context(), AuthzRequest{
		AllowAuthBypass:  h.allowAuthBypass,
		Pack:             pack,
		Action:           action,
		TargetSupplierID: targetSupplierID,
	})
	if err != nil {
		WriteAuthzError(w, err)
		return auth.Claims{}, false
	}
	return claims, true
}

// HandleGetSessionMapping returns current AI/library/confirmed mappings alongside the pack's canonical schema.
func (h *Handlers) HandleGetSessionMapping(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "read", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId is required")
		return
	}
	sess, err := h.repo.GetSession(r.Context(), claims.SupplierID, sessionID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	doc, hasDoc, err := h.repo.GetSessionMapping(r.Context(), claims.SupplierID, sessionID)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	if !hasDoc {
		staged, _, _ := h.repo.ListStagedRows(r.Context(), claims.SupplierID, sessionID, 20, 0)
		headers, samples := extractHeadersAndSampleRows(staged)
		if len(headers) > 0 {
			fp := ComputeHeaderFingerprint(sess.Pack, headers)
			libEntry, _ := h.repo.GetMappingLibrary(r.Context(), claims.SupplierID, sess.Pack, fp)
			doc = h.mapper.ProposeMappings(r.Context(), claims.SupplierID, auth.EffectiveHomeCell(claims), sessionID, sess.Pack, headers, samples, libEntry)
		} else {
			doc = SessionMappingDocument{SessionID: sessionID, Pack: sess.Pack}
		}
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"session":         sess,
		"taxonomy":        PackTaxonomy(sess.Pack),
		"required_groups": PackRequiredGroups(sess.Pack),
		"mapping":         doc,
	})
}

// HandleConfirmSessionMapping applies interactive column overrides or accept_above_confidence bulk confirmation.
func (h *Handlers) HandleConfirmSessionMapping(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authorize(w, r, "", "write", "")
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionId"))
	if sessionID == "" {
		writeMigrationError(w, http.StatusBadRequest, "session_id_required", "sessionId is required")
		return
	}
	var req ConfirmMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMigrationError(w, http.StatusBadRequest, "invalid_json", "malformed mapping request body")
		return
	}
	sess, doc, err := ConfirmSessionMapping(r.Context(), h.repo, claims.SupplierID, sessionID, auth.ActorLabel(claims), req, h.now())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeMigrationJSON(w, http.StatusOK, map[string]any{
		"session": sess,
		"mapping": doc,
	})
}

func writeUploadTicketError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedFileExtension), errors.Is(err, ErrInvalidArtifactSize), errors.Is(err, ErrInvalidArtifactPath):
		writeMigrationError(w, http.StatusBadRequest, "invalid_artifact_descriptor", err.Error())
	case errors.Is(err, ErrUploadStorageUnavailable):
		writeMigrationError(w, http.StatusServiceUnavailable, "upload_storage_unavailable", err.Error())
	default:
		writeMigrationError(w, http.StatusInternalServerError, "upload_ticket_failed", err.Error())
	}
}

func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, parser.ErrPasswordProtectedWorkbook):
		writeMigrationError(w, http.StatusUnprocessableEntity, "password_protected_workbook", err.Error())
	case errors.Is(err, ErrIncompleteRequiredMappings):
		writeMigrationError(w, http.StatusUnprocessableEntity, "incomplete_required_mappings", err.Error())
	case errors.Is(err, ErrDryRunExpired):
		writeMigrationError(w, http.StatusConflict, "dry_run_expired", err.Error())
	case errors.Is(err, ErrDryRunStale):
		writeMigrationError(w, http.StatusConflict, "dry_run_stale", err.Error())
	case errors.Is(err, ErrDryRunBlockingConflicts):
		writeMigrationError(w, http.StatusUnprocessableEntity, "dry_run_blocking_conflicts", err.Error())
	case errors.Is(err, ErrSetOnHandConfirmationRequired):
		writeMigrationError(w, http.StatusBadRequest, "set_on_hand_confirmation_required", err.Error())
	case errors.Is(err, ErrRollbackWindowExpired):
		writeMigrationError(w, http.StatusConflict, "rollback_window_expired", err.Error())
	case errors.Is(err, ErrRollbackNotAllowed):
		writeMigrationError(w, http.StatusConflict, "rollback_not_allowed", err.Error())
	case errors.Is(err, ErrSessionNotApproved):
		writeMigrationError(w, http.StatusConflict, "session_not_approved", err.Error())
	case errors.Is(err, ErrUnitChangeWithNonZeroStock):
		writeMigrationError(w, http.StatusConflict, "unit_change_with_non_zero_stock", err.Error())
	case errors.Is(err, ErrOnHandBelowReserved):
		writeMigrationError(w, http.StatusConflict, "on_hand_below_reserved", err.Error())
	case errors.Is(err, ErrCutoverChecklistFailed):
		writeMigrationError(w, http.StatusConflict, "cutover_checklist_failed", err.Error())
	case errors.Is(err, order.ErrInvalidHistoricalStatus):
		writeMigrationError(w, http.StatusUnprocessableEntity, "invalid_historical_status", err.Error())
	case errors.Is(err, order.ErrOrderNotMigrationHeld):
		writeMigrationError(w, http.StatusConflict, "order_not_migration_held", err.Error())
	case errors.Is(err, ar.ErrMigrationJournalRecorderUnavailable):
		writeMigrationError(w, http.StatusInternalServerError, "migration_journal_recorder_unavailable", err.Error())
	case errors.Is(err, parser.ErrFileFormatMismatch), errors.Is(err, parser.ErrMalformedArtifact), errors.Is(err, ErrArtifactDigestMismatch):
		writeMigrationError(w, http.StatusBadRequest, "invalid_artifact_content", err.Error())
	case errors.Is(err, ErrNotFound):
		writeMigrationError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrInvalidTransition), errors.Is(err, ErrDependencyNotSatisfied),
		errors.Is(err, ErrConflictingInventoryMode), errors.Is(err, ErrWarehouseLockBusy):
		writeMigrationError(w, http.StatusConflict, "state_conflict", err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeMigrationError(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		writeMigrationError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

func writeMigrationError(w http.ResponseWriter, status int, code, msg string) {
	writeMigrationJSON(w, status, map[string]string{
		"error":   code,
		"message": msg,
	})
}

func writeMigrationJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
