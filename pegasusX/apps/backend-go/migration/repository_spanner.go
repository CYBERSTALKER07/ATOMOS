package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// TopicMigrationEvents reuses the inventory/migration import Kafka topic consumed by ai-worker.
var TopicMigrationEvents = events.TopicInventoryImportEvents

const (
	EventMigrationBundleUploaded = "MIGRATION_BUNDLE_UPLOADED"
	EventMigrationSorted         = "MIGRATION_SORTED"
	EventMigrationSessionUpdated = "MIGRATION_SESSION_STATUS_UPDATE"
)

var (
	// ErrNotFound indicates the requested project, bundle, artifact, or session does not exist for the tenant.
	ErrNotFound = errors.New("migration_resource_not_found")
	// ErrInvalidInput indicates malformed request data.
	ErrInvalidInput = errors.New("invalid_migration_input")
)

// UploadedArtifactMeta carries finalized size/hash metadata from client bundle upload completion.
type UploadedArtifactMeta struct {
	ByteSize      int64  `json:"byte_size,omitempty"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
	Encoding      string `json:"encoding,omitempty"`
}

// Repository persists and queries migration projects, bundles, artifacts, child sessions,
// staged rows, and per-warehouse apply locks.
type Repository interface {
	CreateProject(ctx context.Context, p Project) (Project, error)
	GetProject(ctx context.Context, supplierID, projectID string) (Project, error)
	ListProjects(ctx context.Context, supplierID string, limit int) ([]Project, error)
	UpdateProjectStatus(ctx context.Context, supplierID, projectID string, to ProjectStatus) error

	CreateBundleWithArtifacts(ctx context.Context, b Bundle, artifacts []Artifact) (Bundle, []Artifact, error)
	GetBundle(ctx context.Context, supplierID, projectID, bundleID string) (Bundle, error)
	ListBundlesByProject(ctx context.Context, supplierID, projectID string) ([]Bundle, error)
	ListArtifactsByBundle(ctx context.Context, supplierID, projectID, bundleID string) ([]Artifact, error)
	MarkBundleUploadedAndEmit(ctx context.Context, supplierID, projectID, bundleID string, meta map[string]UploadedArtifactMeta) (Bundle, error)
	MarkBundleFailed(ctx context.Context, supplierID, projectID, bundleID, errMsg string) error
	UpsertSortedBundleSessions(ctx context.Context, supplierID, projectID, bundleID string, nextBundleStatus BundleStatus, sortSummary json.RawMessage, artifactStreams map[string]json.RawMessage, sessions []Session) ([]Session, error)

	GetSession(ctx context.Context, supplierID, sessionID string) (Session, error)
	ListSessionsByProject(ctx context.Context, supplierID, projectID string) ([]Session, error)
	ListSessionsByBundle(ctx context.Context, supplierID, projectID, bundleID string) ([]Session, error)
	UpdateSessionStatus(ctx context.Context, supplierID, sessionID string, next SessionStatus, errorSummary json.RawMessage) (Session, error)

	SaveStagedRowsChunk(ctx context.Context, supplierID, sessionID string, rows []StagedRow) (int64, error)
	ListStagedRows(ctx context.Context, supplierID, sessionID string, limit, offset int) ([]StagedRow, int64, error)
	UpdateApplyProgress(ctx context.Context, supplierID, sessionID string, resumeRowIndex int64, lastChunkID string) error

	AcquireWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string, ttl time.Duration) (WarehouseLock, error)
	RenewWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string, ttl time.Duration) (WarehouseLock, error)
	ReleaseWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string) error

	GetSessionMapping(ctx context.Context, supplierID, sessionID string) (SessionMappingDocument, bool, error)
	SaveSessionMapping(ctx context.Context, supplierID, sessionID string, nextStatus SessionStatus, doc SessionMappingDocument) error
	GetMappingLibrary(ctx context.Context, supplierID string, pack Pack, headerFingerprint string) (*MappingLibraryEntry, error)
	UpsertMappingLibrary(ctx context.Context, entry MappingLibraryEntry) error

	SaveMigrationReport(ctx context.Context, rep MigrationReport) error
	GetMigrationReport(ctx context.Context, supplierID, reportID string) (MigrationReport, error)
	MarkSessionDryRunReady(ctx context.Context, supplierID, sessionID, reportID, fingerprint, applyMode string) error
	ApproveSession(ctx context.Context, supplierID, sessionID, applyMode string, setOnHandConfirmed bool) error
	MarkSessionApplied(ctx context.Context, supplierID, sessionID string, rollbackUntil time.Time) error
	SaveSnapshotsChunk(ctx context.Context, supplierID, sessionID string, snapshots []MigrationSnapshotRecord) error
	ListSnapshotsBySession(ctx context.Context, supplierID, sessionID string) ([]MigrationSnapshotRecord, error)
	ExecuteSessionRollback(ctx context.Context, supplierID, sessionID string, snapshots []MigrationSnapshotRecord, now time.Time) (RollbackResult, error)
}

// SpannerRepository implements Repository on Cloud Spanner with transactional outbox writes.
type SpannerRepository struct {
	client *spanner.Client
	now    func() time.Time
}

// NewSpannerRepository constructs a Cloud Spanner migration repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{
		client: client,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (r *SpannerRepository) requireClient() error {
	if r == nil || r.client == nil {
		return errors.New("spanner unavailable")
	}
	return nil
}

type migrationBundleUploadedEvent struct {
	events.BaseEvent
	SupplierID string `json:"supplier_id"`
	ProjectID  string `json:"project_id"`
	BundleID   string `json:"bundle_id"`
	Status     string `json:"status"`
}

// CreateProject inserts a new MigrationProjects row.
func (r *SpannerRepository) CreateProject(ctx context.Context, p Project) (Project, error) {
	if err := r.requireClient(); err != nil {
		return Project{}, err
	}
	mode := strings.TrimSpace(p.Mode)
	if mode == "" {
		mode = ProjectModeCutover
	}
	if mode != ProjectModeCutover {
		return Project{}, fmt.Errorf("%w: unsupported mode %q", ErrInvalidInput, mode)
	}
	status := p.Status
	if status == "" {
		status = ProjectStatusDraft
	}
	checklistJSON, err := toNullJSON(p.Checklist)
	if err != nil {
		return Project{}, fmt.Errorf("checklist_json: %w", err)
	}
	now := r.now()

	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("MigrationProjects", map[string]any{
				"supplier_id":          p.SupplierID,
				"project_id":           p.ProjectID,
				"name":                 strings.TrimSpace(p.Name),
				"mode":                 mode,
				"status":               string(status),
				"checklist_json":       checklistJSON,
				"default_warehouse_id": toNullString(p.DefaultWarehouseID),
				"created_by":           p.CreatedBy,
				"cutover_at":           toNullTime(p.CutoverAt),
				"created_at":           spanner.CommitTimestamp,
				"updated_at":           spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		return Project{}, err
	}
	p.Mode = mode
	p.Status = status
	p.CreatedAt = now
	p.UpdatedAt = now
	return p, nil
}

// GetProject reads a single MigrationProjects row by tenant and project ID.
func (r *SpannerRepository) GetProject(ctx context.Context, supplierID, projectID string) (Project, error) {
	if err := r.requireClient(); err != nil {
		return Project{}, err
	}
	row, err := r.client.Single().ReadRow(ctx, "MigrationProjects", spanner.Key{supplierID, projectID}, []string{
		"supplier_id", "project_id", "name", "mode", "status", "checklist_json",
		"default_warehouse_id", "created_by", "cutover_at", "created_at", "updated_at",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return scanProjectRow(row)
}

// ListProjects returns projects belonging to supplierID ordered newest first.
func (r *SpannerRepository) ListProjects(ctx context.Context, supplierID string, limit int) ([]Project, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT supplier_id, project_id, name, mode, status, checklist_json,
		             default_warehouse_id, created_by, cutover_at, created_at, updated_at
		      FROM MigrationProjects
		      WHERE supplier_id = @supplierId
		      ORDER BY updated_at DESC
		      LIMIT @limit`,
		Params: map[string]any{"supplierId": supplierID, "limit": int64(limit)},
	})
	defer iter.Stop()

	out := make([]Project, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := scanProjectRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// UpdateProjectStatus validates and persists a MigrationProjects status transition.
func (r *SpannerRepository) UpdateProjectStatus(ctx context.Context, supplierID, projectID string, to ProjectStatus) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "MigrationProjects", spanner.Key{supplierID, projectID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var current string
		if err := row.Columns(&current); err != nil {
			return err
		}
		if err := ValidateProjectTransition(ProjectStatus(current), to); err != nil {
			return err
		}
		cols := map[string]any{
			"supplier_id": supplierID,
			"project_id":  projectID,
			"status":      string(to),
			"updated_at":  spanner.CommitTimestamp,
		}
		if to == ProjectStatusCutoverComplete {
			cols["cutover_at"] = spanner.CommitTimestamp
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("MigrationProjects", cols)})
	})
	return err
}

// CreateBundleWithArtifacts atomically inserts an intake bundle and its declared upload artifacts,
// advancing the parent project from DRAFT -> STAGING when appropriate.
func (r *SpannerRepository) CreateBundleWithArtifacts(ctx context.Context, b Bundle, artifacts []Artifact) (Bundle, []Artifact, error) {
	if err := r.requireClient(); err != nil {
		return Bundle{}, nil, err
	}
	now := r.now()
	if b.Status == "" {
		b.Status = BundleStatusUploading
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		pRow, err := txn.ReadRow(ctx, "MigrationProjects", spanner.Key{b.SupplierID, b.ProjectID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var pStatus string
		if err := pRow.Columns(&pStatus); err != nil {
			return err
		}

		muts := make([]*spanner.Mutation, 0, len(artifacts)+2)
		if ProjectStatus(pStatus) == ProjectStatusDraft {
			muts = append(muts, spanner.UpdateMap("MigrationProjects", map[string]any{
				"supplier_id": b.SupplierID,
				"project_id":  b.ProjectID,
				"status":      string(ProjectStatusStaging),
				"updated_at":  spanner.CommitTimestamp,
			}))
		}

		sortJSON, err := toNullJSON(b.SortSummary)
		if err != nil {
			return err
		}
		muts = append(muts, spanner.InsertMap("MigrationBundles", map[string]any{
			"supplier_id":       b.SupplierID,
			"project_id":        b.ProjectID,
			"bundle_id":         b.BundleID,
			"status":            string(b.Status),
			"sort_summary_json": sortJSON,
			"error_message":     toNullString(b.ErrorMsg),
			"created_by":        b.CreatedBy,
			"created_at":        spanner.CommitTimestamp,
			"updated_at":        spanner.CommitTimestamp,
		}))

		for i := range artifacts {
			a := &artifacts[i]
			streamsJSON, err := toNullJSON(a.Streams)
			if err != nil {
				return err
			}
			muts = append(muts, spanner.InsertMap("MigrationArtifacts", map[string]any{
				"supplier_id":    a.SupplierID,
				"project_id":     a.ProjectID,
				"bundle_id":      a.BundleID,
				"artifact_id":    a.ArtifactID,
				"file_name":      a.FileName,
				"byte_size":      a.ByteSize,
				"declared_mime":  toNullString(a.DeclaredMIME),
				"detected_mime":  toNullString(a.DetectedMIME),
				"content_sha256": a.ContentSHA256,
				"encoding":       toNullString(a.Encoding),
				"gcs_path":       a.GCSPath,
				"streams_json":   streamsJSON,
				"created_at":     spanner.CommitTimestamp,
				"updated_at":     spanner.CommitTimestamp,
			}))
			a.CreatedAt = now
			a.UpdatedAt = now
		}

		return txn.BufferWrite(muts)
	})
	if err != nil {
		return Bundle{}, nil, err
	}
	b.CreatedAt = now
	b.UpdatedAt = now
	return b, artifacts, nil
}

// GetBundle reads a MigrationBundles row.
func (r *SpannerRepository) GetBundle(ctx context.Context, supplierID, projectID, bundleID string) (Bundle, error) {
	if err := r.requireClient(); err != nil {
		return Bundle{}, err
	}
	row, err := r.client.Single().ReadRow(ctx, "MigrationBundles", spanner.Key{supplierID, projectID, bundleID}, []string{
		"supplier_id", "project_id", "bundle_id", "status", "sort_summary_json", "error_message", "created_by", "created_at", "updated_at",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Bundle{}, ErrNotFound
		}
		return Bundle{}, err
	}
	return scanBundleRow(row)
}

// ListBundlesByProject returns all bundles attached to projectID.
func (r *SpannerRepository) ListBundlesByProject(ctx context.Context, supplierID, projectID string) ([]Bundle, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT supplier_id, project_id, bundle_id, status, sort_summary_json, error_message, created_by, created_at, updated_at
		      FROM MigrationBundles
		      WHERE supplier_id = @supplierId AND project_id = @projectId
		      ORDER BY created_at DESC`,
		Params: map[string]any{"supplierId": supplierID, "projectId": projectID},
	})
	defer iter.Stop()

	out := make([]Bundle, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		b, err := scanBundleRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// ListArtifactsByBundle returns all MigrationArtifacts rows belonging to a bundle.
func (r *SpannerRepository) ListArtifactsByBundle(ctx context.Context, supplierID, projectID, bundleID string) ([]Artifact, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT supplier_id, project_id, bundle_id, artifact_id, file_name, byte_size,
		             declared_mime, detected_mime, content_sha256, encoding, gcs_path, streams_json,
		             created_at, updated_at
		      FROM MigrationArtifacts
		      WHERE supplier_id = @supplierId AND project_id = @projectId AND bundle_id = @bundleId
		      ORDER BY created_at ASC`,
		Params: map[string]any{"supplierId": supplierID, "projectId": projectID, "bundleId": bundleID},
	})
	defer iter.Stop()

	out := make([]Artifact, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		a, err := scanArtifactRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// MarkBundleUploadedAndEmit marks a bundle CLASSIFYING (idempotent if already CLASSIFYING/SORTED/NEEDS_REVIEW)
// and atomically writes a MIGRATION_BUNDLE_UPLOADED outbox event in the same Spanner transaction.
func (r *SpannerRepository) MarkBundleUploadedAndEmit(ctx context.Context, supplierID, projectID, bundleID string, meta map[string]UploadedArtifactMeta) (Bundle, error) {
	if err := r.requireClient(); err != nil {
		return Bundle{}, err
	}
	var result Bundle
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "MigrationBundles", spanner.Key{supplierID, projectID, bundleID}, []string{
			"supplier_id", "project_id", "bundle_id", "status", "sort_summary_json", "error_message", "created_by", "created_at", "updated_at",
		})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		b, err := scanBundleRow(row)
		if err != nil {
			return err
		}
		if b.Status == BundleStatusClassifying || b.Status == BundleStatusSorted || b.Status == BundleStatusNeedsReview {
			result = b
			return nil
		}
		if err := ValidateBundleTransition(b.Status, BundleStatusClassifying); err != nil {
			return err
		}

		muts := make([]*spanner.Mutation, 0, len(meta)+1)
		muts = append(muts, spanner.UpdateMap("MigrationBundles", map[string]any{
			"supplier_id": supplierID,
			"project_id":  projectID,
			"bundle_id":   bundleID,
			"status":      string(BundleStatusClassifying),
			"updated_at":  spanner.CommitTimestamp,
		}))

		for artifactID, m := range meta {
			upd := map[string]any{
				"supplier_id": supplierID,
				"project_id":  projectID,
				"bundle_id":   bundleID,
				"artifact_id": artifactID,
				"updated_at":  spanner.CommitTimestamp,
			}
			changed := false
			if m.ByteSize > 0 {
				upd["byte_size"] = m.ByteSize
				changed = true
			}
			if sha := strings.TrimSpace(m.ContentSHA256); sha != "" {
				upd["content_sha256"] = sha
				changed = true
			}
			if enc := strings.TrimSpace(m.Encoding); enc != "" {
				upd["encoding"] = enc
				changed = true
			}
			if changed {
				muts = append(muts, spanner.UpdateMap("MigrationArtifacts", upd))
			}
		}

		if err := txn.BufferWrite(muts); err != nil {
			return err
		}

		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, "MigrationBundle", bundleID, TopicMigrationEvents, migrationBundleUploadedEvent{
			BaseEvent:  events.BaseEvent{Type: EventMigrationBundleUploaded, Version: 1},
			SupplierID: supplierID,
			ProjectID:  projectID,
			BundleID:   bundleID,
			Status:     string(BundleStatusClassifying),
		}); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}

		b.Status = BundleStatusClassifying
		b.UpdatedAt = r.now()
		result = b
		return nil
	})
	return result, err
}

// UpsertSortedBundleSessions persists stream classification metadata on artifacts,
// creates/updates child SupplierImportSessions, and moves the bundle to SORTED or NEEDS_REVIEW.
func (r *SpannerRepository) UpsertSortedBundleSessions(
	ctx context.Context,
	supplierID, projectID, bundleID string,
	nextBundleStatus BundleStatus,
	sortSummary json.RawMessage,
	artifactStreams map[string]json.RawMessage,
	sessions []Session,
) ([]Session, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	now := r.now()

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		bRow, err := txn.ReadRow(ctx, "MigrationBundles", spanner.Key{supplierID, projectID, bundleID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var curStatus string
		if err := bRow.Columns(&curStatus); err != nil {
			return err
		}
		if err := ValidateBundleTransition(BundleStatus(curStatus), nextBundleStatus); err != nil {
			return err
		}

		sortJSON, err := toNullJSON(sortSummary)
		if err != nil {
			return err
		}
		muts := make([]*spanner.Mutation, 0, len(artifactStreams)+len(sessions)+1)
		muts = append(muts, spanner.UpdateMap("MigrationBundles", map[string]any{
			"supplier_id":       supplierID,
			"project_id":        projectID,
			"bundle_id":         bundleID,
			"status":            string(nextBundleStatus),
			"sort_summary_json": sortJSON,
			"updated_at":        spanner.CommitTimestamp,
		}))

		for artifactID, rawStreams := range artifactStreams {
			sJSON, err := toNullJSON(rawStreams)
			if err != nil {
				return err
			}
			muts = append(muts, spanner.UpdateMap("MigrationArtifacts", map[string]any{
				"supplier_id":  supplierID,
				"project_id":   projectID,
				"bundle_id":    bundleID,
				"artifact_id":  artifactID,
				"streams_json": sJSON,
				"updated_at":   spanner.CommitTimestamp,
			}))
		}

		existingByID := make(map[string]Session, len(sessions))
		eIter := txn.Query(ctx, spanner.Statement{
			SQL: "SELECT " + strings.Join(sessionColumns(), ", ") + `
			      FROM SupplierImportSessions
			      WHERE supplier_id = @sid AND bundle_id = @bnd`,
			Params: map[string]any{"sid": supplierID, "bnd": bundleID},
		})
		for {
			row, qErr := eIter.Next()
			if qErr == iterator.Done {
				break
			}
			if qErr != nil {
				eIter.Stop()
				return qErr
			}
			prev, sErr := scanSessionRow(row)
			if sErr != nil {
				eIter.Stop()
				return sErr
			}
			existingByID[prev.SessionID] = prev
		}
		eIter.Stop()

		for i := range sessions {
			s := &sessions[i]
			if s.Status == "" {
				s.Status = SessionStatusInitialized
			}
			prev, exists := existingByID[s.SessionID]
			if exists {
				mergeExistingSessionLifecycle(s, prev)
			}
			errJSON, err := toNullJSON(s.ErrorSummary)
			if err != nil {
				return err
			}
			cols := map[string]any{
				"supplier_id":           supplierID,
				"session_id":            s.SessionID,
				"status":                string(NormalizeSessionStatus(string(s.Status))),
				"file_name":             s.FileName,
				"total_rows":            s.TotalRows,
				"error_summary":         errJSON,
				"project_id":            toNullString(projectID),
				"bundle_id":             toNullString(bundleID),
				"artifact_id":           toNullString(s.ArtifactID),
				"stream_key":            toNullString(s.StreamKey),
				"stream_split_key":      toNullString(s.StreamSplitKey),
				"pack":                  toNullString(string(s.Pack)),
				"sheet_name":            toNullString(s.SheetName),
				"gcs_path":              toNullString(s.GCSPath),
				"content_sha256":        toNullString(s.ContentSHA256),
				"detected_mime":         toNullString(s.DetectedMIME),
				"encoding":              toNullString(s.Encoding),
				"ai_model":              toNullString(s.AIModel),
				"ai_status":             toNullString(s.AIStatus),
				"dry_run_report_id":     toNullString(s.DryRunReportID),
				"dry_run_fingerprint":   toNullString(s.DryRunFingerprint),
				"apply_mode":            toNullString(s.ApplyMode),
				"set_on_hand_confirmed": spanner.NullBool{Bool: s.SetOnHandConfirmed, Valid: s.SetOnHandConfirmed},
				"resume_row_index":      spanner.NullInt64{Int64: s.ResumeRowIndex, Valid: s.ResumeRowIndex > 0},
				"last_chunk_id":         toNullString(s.LastChunkID),
				"rollback_until":        toNullTime(s.RollbackUntil),
				"updated_at":            spanner.CommitTimestamp,
			}
			s.SupplierID = supplierID
			s.ProjectID = projectID
			s.BundleID = bundleID
			s.UpdatedAt = now
			if exists {
				s.CreatedAt = prev.CreatedAt
				muts = append(muts, spanner.UpdateMap("SupplierImportSessions", cols))
			} else {
				cols["created_at"] = spanner.CommitTimestamp
				s.CreatedAt = now
				muts = append(muts, spanner.InsertMap("SupplierImportSessions", cols))
			}
		}

		return txn.BufferWrite(muts)
	})
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func mergeExistingSessionLifecycle(dst *Session, prev Session) {
	if sessionLifecycleRank(prev.Status) > sessionLifecycleRank(dst.Status) {
		dst.Status = prev.Status
	}
	if dst.DryRunReportID == "" {
		dst.DryRunReportID = prev.DryRunReportID
	}
	if dst.DryRunFingerprint == "" {
		dst.DryRunFingerprint = prev.DryRunFingerprint
	}
	if dst.ApplyMode == "" {
		dst.ApplyMode = prev.ApplyMode
	}
	if !dst.SetOnHandConfirmed && prev.SetOnHandConfirmed {
		dst.SetOnHandConfirmed = true
	}
	if dst.ResumeRowIndex <= 0 && prev.ResumeRowIndex > 0 {
		dst.ResumeRowIndex = prev.ResumeRowIndex
	}
	if dst.LastChunkID == "" {
		dst.LastChunkID = prev.LastChunkID
	}
	if (dst.RollbackUntil == nil || dst.RollbackUntil.IsZero()) && prev.RollbackUntil != nil && !prev.RollbackUntil.IsZero() {
		t := *prev.RollbackUntil
		dst.RollbackUntil = &t
	}
	if dst.AIModel == "" {
		dst.AIModel = prev.AIModel
	}
	if dst.AIStatus == "" {
		dst.AIStatus = prev.AIStatus
	}
}

func sessionLifecycleRank(status SessionStatus) int {
	switch NormalizeSessionStatus(string(status)) {
	case SessionStatusInitialized:
		return 1
	case SessionStatusUploaded, SessionStatusDiscovering:
		return 2
	case SessionStatusMappingRequired:
		return 3
	case SessionStatusMappingProposed:
		return 4
	case SessionStatusDiscovered:
		return 5
	case SessionStatusValidating:
		return 6
	case SessionStatusDryRunReady:
		return 7
	case SessionStatusApproved:
		return 8
	case SessionStatusSnapshotting:
		return 9
	case SessionStatusApplying:
		return 10
	case SessionStatusApplied:
		return 11
	case SessionStatusRollingBack:
		return 12
	case SessionStatusRolledBack:
		return 13
	case SessionStatusCutoverComplete:
		return 14
	default:
		return 0
	}
}

// MarkBundleFailed transitions a bundle to FAILED with a structured error summary.
func (r *SpannerRepository) MarkBundleFailed(ctx context.Context, supplierID, projectID, bundleID, errMsg string) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		bRow, err := txn.ReadRow(ctx, "MigrationBundles", spanner.Key{supplierID, projectID, bundleID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var cur string
		if err := bRow.Columns(&cur); err != nil {
			return err
		}
		if BundleStatus(cur) != BundleStatusFailed {
			if err := ValidateBundleTransition(BundleStatus(cur), BundleStatusFailed); err != nil {
				return err
			}
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("MigrationBundles", map[string]any{
			"supplier_id":   supplierID,
			"project_id":    projectID,
			"bundle_id":     bundleID,
			"status":        string(BundleStatusFailed),
			"error_message": toNullString(errMsg),
			"updated_at":    spanner.CommitTimestamp,
		})})
	})
	return err
}

// GetSession reads a widened SupplierImportSessions row.
func (r *SpannerRepository) GetSession(ctx context.Context, supplierID, sessionID string) (Session, error) {
	if err := r.requireClient(); err != nil {
		return Session{}, err
	}
	row, err := r.client.Single().ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, sessionColumns())
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Session{}, ErrNotFound
		}
		return Session{}, err
	}
	return scanSessionRow(row)
}

// ListSessionsByProject lists all SupplierImportSessions linked to projectID.
func (r *SpannerRepository) ListSessionsByProject(ctx context.Context, supplierID, projectID string) ([]Session, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + strings.Join(sessionColumns(), ", ") + `
		      FROM SupplierImportSessions
		      WHERE supplier_id = @supplierId AND project_id = @projectId
		      ORDER BY created_at ASC`,
		Params: map[string]any{"supplierId": supplierID, "projectId": projectID},
	}
	return r.querySessions(ctx, stmt)
}

// ListSessionsByBundle lists all SupplierImportSessions linked to bundleID.
func (r *SpannerRepository) ListSessionsByBundle(ctx context.Context, supplierID, projectID, bundleID string) ([]Session, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	stmt := spanner.Statement{
		SQL: `SELECT ` + strings.Join(sessionColumns(), ", ") + `
		      FROM SupplierImportSessions
		      WHERE supplier_id = @supplierId AND project_id = @projectId AND bundle_id = @bundleId
		      ORDER BY created_at ASC`,
		Params: map[string]any{"supplierId": supplierID, "projectId": projectID, "bundleId": bundleID},
	}
	return r.querySessions(ctx, stmt)
}

func (r *SpannerRepository) querySessions(ctx context.Context, stmt spanner.Statement) ([]Session, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make([]Session, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		s, err := scanSessionRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// UpdateSessionStatus validates and persists a session status transition.
func (r *SpannerRepository) UpdateSessionStatus(ctx context.Context, supplierID, sessionID string, next SessionStatus, errorSummary json.RawMessage) (Session, error) {
	if err := r.requireClient(); err != nil {
		return Session{}, err
	}
	var updated Session
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, sessionColumns())
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		s, err := scanSessionRow(row)
		if err != nil {
			return err
		}
		target := NormalizeSessionStatus(string(next))
		if err := ValidateSessionTransition(s.Status, target); err != nil {
			return err
		}
		cols := map[string]any{
			"supplier_id": supplierID,
			"session_id":  sessionID,
			"status":      string(target),
			"updated_at":  spanner.CommitTimestamp,
		}
		if len(errorSummary) > 0 {
			errJSON, err := toNullJSON(errorSummary)
			if err != nil {
				return err
			}
			cols["error_summary"] = errJSON
			s.ErrorSummary = errorSummary
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("SupplierImportSessions", cols)}); err != nil {
			return err
		}
		s.Status = target
		s.UpdatedAt = r.now()
		updated = s
		return nil
	})
	return updated, err
}

// SaveStagedRowsChunk writes up to 5,000 staged rows per transaction with no 10k row ceiling
// and updates SupplierImportSessions.total_rows monotonically.
func (r *SpannerRepository) SaveStagedRowsChunk(ctx context.Context, supplierID, sessionID string, rows []StagedRow) (int64, error) {
	if err := r.requireClient(); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		s, err := r.GetSession(ctx, supplierID, sessionID)
		return s.TotalRows, err
	}
	const maxChunkSize = 5000
	var finalTotal int64
	for start := 0; start < len(rows); start += maxChunkSize {
		end := start + maxChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]
		_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			row, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"total_rows"})
			if err != nil {
				if spanner.ErrCode(err) == codes.NotFound {
					return ErrNotFound
				}
				return err
			}
			var existingTotal int64
			if err := row.Columns(&existingTotal); err != nil {
				return err
			}
			maxTotal := existingTotal
			muts := make([]*spanner.Mutation, 0, len(chunk)+1)
			for _, sr := range chunk {
				if sr.RowIndex+1 > maxTotal {
					maxTotal = sr.RowIndex + 1
				}
				rawJSON, err := toNullJSON(sr.RawData)
				if err != nil {
					return fmt.Errorf("row %d raw_data: %w", sr.RowIndex, err)
				}
				cleanedJSON, err := toNullJSON(sr.CleanedData)
				if err != nil {
					return fmt.Errorf("row %d cleaned_data: %w", sr.RowIndex, err)
				}
				muts = append(muts, spanner.InsertOrUpdateMap("SupplierImportStagedRows", map[string]any{
					"supplier_id":       supplierID,
					"session_id":        sessionID,
					"row_index":         sr.RowIndex,
					"raw_data":          rawJSON,
					"cleaned_data":      cleanedJSON,
					"validation_errors": sr.ValidationErrors,
					"is_new_product":    sr.IsNewProduct,
					"external_id":       toNullString(sr.ExternalID),
					"stream_split_key":  toNullString(sr.StreamSplitKey),
					"apply_action":      toNullString(sr.ApplyAction),
					"apply_chunk":       spanner.NullInt64{Int64: sr.ApplyChunk, Valid: sr.ApplyChunk > 0},
					"is_new_entity":     spanner.NullBool{Bool: sr.IsNewEntity, Valid: sr.ApplyAction != ""},
					"created_at":        spanner.CommitTimestamp,
					"updated_at":        spanner.CommitTimestamp,
				}))
			}
			muts = append(muts, spanner.UpdateMap("SupplierImportSessions", map[string]any{
				"supplier_id": supplierID,
				"session_id":  sessionID,
				"total_rows":  maxTotal,
				"updated_at":  spanner.CommitTimestamp,
			}))
			finalTotal = maxTotal
			return txn.BufferWrite(muts)
		})
		if err != nil {
			return 0, err
		}
	}
	return finalTotal, nil
}

const stagedRowSelectColumns = `supplier_id, session_id, row_index, raw_data, cleaned_data, validation_errors,
		             is_new_product, external_id, stream_split_key, apply_action, apply_chunk, is_new_entity,
		             created_at, updated_at`

func (r *SpannerRepository) queryStagedRows(ctx context.Context, stmt spanner.Statement) ([]StagedRow, error) {
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make([]StagedRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		sr, err := scanStagedRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
}

// ListStagedRows paginates staged rows for a session along with session total_rows.
func (r *SpannerRepository) ListStagedRows(ctx context.Context, supplierID, sessionID string, limit, offset int) ([]StagedRow, int64, error) {
	if err := r.requireClient(); err != nil {
		return nil, 0, err
	}
	s, err := r.GetSession(ctx, supplierID, sessionID)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	out, err := r.queryStagedRows(ctx, spanner.Statement{
		SQL: `SELECT ` + stagedRowSelectColumns + `
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId AND session_id = @sessionId
		      ORDER BY row_index ASC
		      LIMIT @limit OFFSET @offset`,
		Params: map[string]any{
			"supplierId": supplierID,
			"sessionId":  sessionID,
			"limit":      int64(limit),
			"offset":     int64(offset),
		},
	})
	if err != nil {
		return nil, 0, err
	}
	return out, s.TotalRows, nil
}

// ListStagedRowsAfter streams staged rows using O(N) primary-key cursor pagination
// (row_index > afterRowIndex) rather than O(N^2) OFFSET scans. Pass afterRowIndex < 0
// to start from the first staged row (including row_index = 0).
func (r *SpannerRepository) ListStagedRowsAfter(ctx context.Context, supplierID, sessionID string, afterRowIndex int64, limit int) ([]StagedRow, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1000
	}
	if limit > 2000 {
		limit = 2000
	}
	if afterRowIndex < 0 {
		return r.queryStagedRows(ctx, spanner.Statement{
			SQL: `SELECT ` + stagedRowSelectColumns + `
			      FROM SupplierImportStagedRows
			      WHERE supplier_id = @supplierId AND session_id = @sessionId
			      ORDER BY row_index ASC
			      LIMIT @limit`,
			Params: map[string]any{
				"supplierId": supplierID,
				"sessionId":  sessionID,
				"limit":      int64(limit),
			},
		})
	}
	return r.queryStagedRows(ctx, spanner.Statement{
		SQL: `SELECT ` + stagedRowSelectColumns + `
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId AND session_id = @sessionId AND row_index > @afterRowIndex
		      ORDER BY row_index ASC
		      LIMIT @limit`,
		Params: map[string]any{
			"supplierId":    supplierID,
			"sessionId":     sessionID,
			"afterRowIndex": afterRowIndex,
			"limit":         int64(limit),
		},
	})
}

// UpdateApplyProgress checkpoints chunk resume cursors durably.
func (r *SpannerRepository) UpdateApplyProgress(ctx context.Context, supplierID, sessionID string, resumeRowIndex int64, lastChunkID string) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("SupplierImportSessions", map[string]any{
				"supplier_id":      supplierID,
				"session_id":       sessionID,
				"resume_row_index": resumeRowIndex,
				"last_chunk_id":    toNullString(lastChunkID),
				"updated_at":       spanner.CommitTimestamp,
			}),
		})
	})
	return err
}

// AcquireWarehouseLock acquires or re-entrantly extends an exclusive lease on (supplier_id, warehouse_id).
func (r *SpannerRepository) AcquireWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string, ttl time.Duration) (WarehouseLock, error) {
	if err := r.requireClient(); err != nil {
		return WarehouseLock{}, err
	}
	if ttl <= 0 {
		ttl = DefaultWarehouseLockTTL
	}
	var out WarehouseLock
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		now := r.now()
		existing, err := readWarehouseLockInTxn(ctx, txn, supplierID, warehouseID)
		if err != nil {
			return err
		}
		if !canAcquireLock(existing, sessionID, now) {
			return buildWarehouseLockBusyError(*existing)
		}
		expiresAt := now.Add(ttl)
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("MigrationWarehouseLocks", map[string]any{
				"supplier_id":  supplierID,
				"warehouse_id": warehouseID,
				"session_id":   sessionID,
				"acquired_at":  spanner.CommitTimestamp,
				"expires_at":   expiresAt,
			}),
		}); err != nil {
			return err
		}
		out = WarehouseLock{
			SupplierID:  supplierID,
			WarehouseID: warehouseID,
			SessionID:   sessionID,
			AcquiredAt:  now,
			ExpiresAt:   expiresAt,
		}
		return nil
	})
	return out, err
}

// RenewWarehouseLock extends an existing lock lease owned by sessionID.
func (r *SpannerRepository) RenewWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string, ttl time.Duration) (WarehouseLock, error) {
	if err := r.requireClient(); err != nil {
		return WarehouseLock{}, err
	}
	if ttl <= 0 {
		ttl = DefaultWarehouseLockTTL
	}
	var out WarehouseLock
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		now := r.now()
		existing, err := readWarehouseLockInTxn(ctx, txn, supplierID, warehouseID)
		if err != nil {
			return err
		}
		if existing == nil || strings.TrimSpace(existing.SessionID) != strings.TrimSpace(sessionID) {
			return ErrWarehouseLockNotHeld
		}
		expiresAt := now.Add(ttl)
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("MigrationWarehouseLocks", map[string]any{
				"supplier_id":  supplierID,
				"warehouse_id": warehouseID,
				"expires_at":   expiresAt,
			}),
		}); err != nil {
			return err
		}
		out = *existing
		out.ExpiresAt = expiresAt
		return nil
	})
	return out, err
}

// ReleaseWarehouseLock deletes the lock row when currently held by sessionID.
func (r *SpannerRepository) ReleaseWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		existing, err := readWarehouseLockInTxn(ctx, txn, supplierID, warehouseID)
		if err != nil {
			return err
		}
		if existing == nil {
			return nil
		}
		if strings.TrimSpace(existing.SessionID) != strings.TrimSpace(sessionID) {
			return ErrWarehouseLockNotHeld
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("MigrationWarehouseLocks", spanner.Key{supplierID, warehouseID}),
		})
	})
	return err
}

func readWarehouseLockInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, warehouseID string) (*WarehouseLock, error) {
	row, err := txn.ReadRow(ctx, "MigrationWarehouseLocks", spanner.Key{supplierID, warehouseID}, []string{
		"supplier_id", "warehouse_id", "session_id", "acquired_at", "expires_at",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var lock WarehouseLock
	if err := row.Columns(&lock.SupplierID, &lock.WarehouseID, &lock.SessionID, &lock.AcquiredAt, &lock.ExpiresAt); err != nil {
		return nil, err
	}
	lock.AcquiredAt = lock.AcquiredAt.UTC()
	lock.ExpiresAt = lock.ExpiresAt.UTC()
	return &lock, nil
}

func sessionColumns() []string {
	return []string{
		"supplier_id", "session_id", "status", "file_name", "total_rows", "error_summary",
		"project_id", "bundle_id", "artifact_id", "stream_key", "stream_split_key",
		"pack", "sheet_name", "gcs_path", "content_sha256", "detected_mime", "encoding",
		"ai_model", "ai_status", "dry_run_report_id", "dry_run_fingerprint", "apply_mode",
		"set_on_hand_confirmed", "resume_row_index", "last_chunk_id", "rollback_until",
		"created_at", "updated_at",
	}
}

func scanProjectRow(row *spanner.Row) (Project, error) {
	var p Project
	var status string
	var checklist spanner.NullJSON
	var defaultWh spanner.NullString
	var cutoverAt spanner.NullTime
	if err := row.Columns(
		&p.SupplierID, &p.ProjectID, &p.Name, &p.Mode, &status, &checklist,
		&defaultWh, &p.CreatedBy, &cutoverAt, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Project{}, err
	}
	p.Status = ProjectStatus(status)
	p.Checklist = fromNullJSON(checklist)
	p.DefaultWarehouseID = defaultWh.StringVal
	if cutoverAt.Valid {
		t := cutoverAt.Time.UTC()
		p.CutoverAt = &t
	}
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	return p, nil
}

func scanBundleRow(row *spanner.Row) (Bundle, error) {
	var b Bundle
	var status string
	var sortJSON spanner.NullJSON
	var errMsg spanner.NullString
	if err := row.Columns(
		&b.SupplierID, &b.ProjectID, &b.BundleID, &status, &sortJSON, &errMsg, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return Bundle{}, err
	}
	b.Status = BundleStatus(status)
	b.SortSummary = fromNullJSON(sortJSON)
	b.ErrorMsg = errMsg.StringVal
	b.CreatedAt = b.CreatedAt.UTC()
	b.UpdatedAt = b.UpdatedAt.UTC()
	return b, nil
}

func scanArtifactRow(row *spanner.Row) (Artifact, error) {
	var a Artifact
	var declMIME, detMIME, enc spanner.NullString
	var streams spanner.NullJSON
	if err := row.Columns(
		&a.SupplierID, &a.ProjectID, &a.BundleID, &a.ArtifactID, &a.FileName, &a.ByteSize,
		&declMIME, &detMIME, &a.ContentSHA256, &enc, &a.GCSPath, &streams, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return Artifact{}, err
	}
	a.DeclaredMIME = declMIME.StringVal
	a.DetectedMIME = detMIME.StringVal
	a.Encoding = enc.StringVal
	a.Streams = fromNullJSON(streams)
	a.CreatedAt = a.CreatedAt.UTC()
	a.UpdatedAt = a.UpdatedAt.UTC()
	return a, nil
}

func scanSessionRow(row *spanner.Row) (Session, error) {
	var s Session
	var status string
	var errJSON spanner.NullJSON
	var projectID, bundleID, artifactID, streamKey, splitKey, pack, sheet, gcsPath, sha, detMIME, enc spanner.NullString
	var aiModel, aiStatus, reportID, fp, applyMode, chunkID spanner.NullString
	var setOnHand spanner.NullBool
	var resumeIdx spanner.NullInt64
	var rollbackUntil, updatedAt spanner.NullTime

	if err := row.Columns(
		&s.SupplierID, &s.SessionID, &status, &s.FileName, &s.TotalRows, &errJSON,
		&projectID, &bundleID, &artifactID, &streamKey, &splitKey,
		&pack, &sheet, &gcsPath, &sha, &detMIME, &enc,
		&aiModel, &aiStatus, &reportID, &fp, &applyMode,
		&setOnHand, &resumeIdx, &chunkID, &rollbackUntil,
		&s.CreatedAt, &updatedAt,
	); err != nil {
		return Session{}, err
	}
	s.Status = NormalizeSessionStatus(status)
	s.ErrorSummary = fromNullJSON(errJSON)
	s.ProjectID = projectID.StringVal
	s.BundleID = bundleID.StringVal
	s.ArtifactID = artifactID.StringVal
	s.StreamKey = streamKey.StringVal
	s.StreamSplitKey = splitKey.StringVal
	s.Pack = Pack(pack.StringVal)
	s.SheetName = sheet.StringVal
	s.GCSPath = gcsPath.StringVal
	s.ContentSHA256 = sha.StringVal
	s.DetectedMIME = detMIME.StringVal
	s.Encoding = enc.StringVal
	s.AIModel = aiModel.StringVal
	s.AIStatus = aiStatus.StringVal
	s.DryRunReportID = reportID.StringVal
	s.DryRunFingerprint = fp.StringVal
	s.ApplyMode = applyMode.StringVal
	s.SetOnHandConfirmed = setOnHand.Valid && setOnHand.Bool
	if resumeIdx.Valid {
		s.ResumeRowIndex = resumeIdx.Int64
	}
	s.LastChunkID = chunkID.StringVal
	if rollbackUntil.Valid {
		t := rollbackUntil.Time.UTC()
		s.RollbackUntil = &t
	}
	s.CreatedAt = s.CreatedAt.UTC()
	if updatedAt.Valid {
		s.UpdatedAt = updatedAt.Time.UTC()
	}
	return s, nil
}

func scanStagedRow(row *spanner.Row) (StagedRow, error) {
	var sr StagedRow
	var rawJ, cleanedJ spanner.NullJSON
	var extID, splitKey, action spanner.NullString
	var chunk spanner.NullInt64
	var isNew spanner.NullBool
	var updatedAt spanner.NullTime
	if err := row.Columns(
		&sr.SupplierID, &sr.SessionID, &sr.RowIndex, &rawJ, &cleanedJ, &sr.ValidationErrors,
		&sr.IsNewProduct, &extID, &splitKey, &action, &chunk, &isNew, &sr.CreatedAt, &updatedAt,
	); err != nil {
		return StagedRow{}, err
	}
	sr.RawData = fromNullJSON(rawJ)
	sr.CleanedData = fromNullJSON(cleanedJ)
	sr.ExternalID = extID.StringVal
	sr.StreamSplitKey = splitKey.StringVal
	sr.ApplyAction = action.StringVal
	if chunk.Valid {
		sr.ApplyChunk = chunk.Int64
	}
	sr.IsNewEntity = isNew.Valid && isNew.Bool
	sr.CreatedAt = sr.CreatedAt.UTC()
	if updatedAt.Valid {
		sr.UpdatedAt = updatedAt.Time.UTC()
	}
	return sr, nil
}

// GetSessionMapping reads the persisted SessionMappingDocument from SupplierImportMapping.
func (r *SpannerRepository) GetSessionMapping(ctx context.Context, supplierID, sessionID string) (SessionMappingDocument, bool, error) {
	if err := r.requireClient(); err != nil {
		return SessionMappingDocument{}, false, err
	}
	row, err := r.client.Single().ReadRow(ctx, "SupplierImportMapping", spanner.Key{supplierID, sessionID}, []string{"mapping_json"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return SessionMappingDocument{}, false, nil
		}
		return SessionMappingDocument{}, false, err
	}
	var mJSON spanner.NullJSON
	if err := row.Columns(&mJSON); err != nil {
		return SessionMappingDocument{}, false, err
	}
	raw := fromNullJSON(mJSON)
	if len(raw) == 0 {
		return SessionMappingDocument{}, false, nil
	}
	var doc SessionMappingDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return SessionMappingDocument{}, false, err
	}
	return doc, true, nil
}

// SaveSessionMapping persists mapping proposals/confirmations to SupplierImportMapping and updates session status & AI metadata.
func (r *SpannerRepository) SaveSessionMapping(ctx context.Context, supplierID, sessionID string, nextStatus SessionStatus, doc SessionMappingDocument) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	rawDoc, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	docJSON, err := toNullJSON(rawDoc)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		sRow, err := txn.ReadRow(ctx, "SupplierImportSessions", spanner.Key{supplierID, sessionID}, []string{"status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var curStatusStr string
		if err := sRow.Columns(&curStatusStr); err != nil {
			return err
		}
		cur := NormalizeSessionStatus(curStatusStr)
		target := NormalizeSessionStatus(string(nextStatus))
		if target != "" && target != cur {
			if err := ValidateSessionTransition(cur, target); err != nil {
				return err
			}
		} else {
			target = cur
		}

		upd := map[string]any{
			"supplier_id": supplierID,
			"session_id":  sessionID,
			"status":      string(target),
			"ai_model":    toNullString(doc.AIModel),
			"ai_status":   toNullString(doc.AIStatus),
			"updated_at":  spanner.CommitTimestamp,
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("SupplierImportMapping", map[string]any{
				"supplier_id":  supplierID,
				"session_id":   sessionID,
				"mapping_json": docJSON,
				"created_at":   spanner.CommitTimestamp,
				"updated_at":   spanner.CommitTimestamp,
			}),
			spanner.UpdateMap("SupplierImportSessions", upd),
		})
	})
	return err
}

// GetMappingLibrary fetches a previously confirmed column mapping template by header fingerprint.
func (r *SpannerRepository) GetMappingLibrary(ctx context.Context, supplierID string, pack Pack, headerFingerprint string) (*MappingLibraryEntry, error) {
	if err := r.requireClient(); err != nil {
		return nil, err
	}
	row, err := r.client.Single().ReadRow(ctx, "SupplierImportMappingLibrary", spanner.Key{supplierID, string(pack), headerFingerprint}, []string{
		"mapping_json", "confirmed_by", "updated_at",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var mJSON spanner.NullJSON
	var confirmedBy string
	var updatedAt time.Time
	if err := row.Columns(&mJSON, &confirmedBy, &updatedAt); err != nil {
		return nil, err
	}
	var mappings []ColumnMappingProposal
	if raw := fromNullJSON(mJSON); len(raw) > 0 {
		_ = json.Unmarshal(raw, &mappings)
	}
	return &MappingLibraryEntry{
		SupplierID:        supplierID,
		Pack:              pack,
		HeaderFingerprint: headerFingerprint,
		Mappings:          mappings,
		ConfirmedBy:       confirmedBy,
		UpdatedAt:         updatedAt.UTC(),
	}, nil
}

// UpsertMappingLibrary stores a confirmed column mapping pattern for instant reuse on future files.
func (r *SpannerRepository) UpsertMappingLibrary(ctx context.Context, entry MappingLibraryEntry) error {
	if err := r.requireClient(); err != nil {
		return err
	}
	raw, err := json.Marshal(entry.Mappings)
	if err != nil {
		return err
	}
	mJSON, err := toNullJSON(raw)
	if err != nil {
		return err
	}
	_, err = r.client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("SupplierImportMappingLibrary", map[string]any{
			"supplier_id":        entry.SupplierID,
			"pack":               string(entry.Pack),
			"header_fingerprint": entry.HeaderFingerprint,
			"mapping_json":       mJSON,
			"confirmed_by":       entry.ConfirmedBy,
			"updated_at":         spanner.CommitTimestamp,
		}),
	})
	return err
}

func toNullString(v string) spanner.NullString {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: trimmed, Valid: true}
}

func toNullTime(t *time.Time) spanner.NullTime {
	if t == nil || t.IsZero() {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: t.UTC(), Valid: true}
}

func toNullJSON(raw json.RawMessage) (spanner.NullJSON, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return spanner.NullJSON{}, nil
	}
	var val any
	if err := json.Unmarshal([]byte(trimmed), &val); err != nil {
		return spanner.NullJSON{}, err
	}
	return spanner.NullJSON{Value: val, Valid: true}, nil
}

func fromNullJSON(v spanner.NullJSON) json.RawMessage {
	if !v.Valid || v.Value == nil {
		return nil
	}
	b, err := json.Marshal(v.Value)
	if err != nil {
		return nil
	}
	return b
}
