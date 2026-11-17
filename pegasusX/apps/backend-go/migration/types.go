package migration

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ProjectModeCutover is the sole supported migration execution mode in this release.
const ProjectModeCutover = "CUTOVER"

// ProjectStatus represents the overall lifecycle of a supplier cutover project.
type ProjectStatus string

const (
	ProjectStatusDraft           ProjectStatus = "DRAFT"
	ProjectStatusStaging         ProjectStatus = "STAGING"
	ProjectStatusDryRunReady     ProjectStatus = "DRY_RUN_READY"
	ProjectStatusReadyToCutover  ProjectStatus = "READY_TO_CUTOVER"
	ProjectStatusCutoverComplete ProjectStatus = "CUTOVER_COMPLETE"
	ProjectStatusFailed          ProjectStatus = "FAILED"
)

// BundleStatus represents the intake & auto-sort lifecycle of a multi-file upload drop.
type BundleStatus string

const (
	BundleStatusCreated     BundleStatus = "CREATED"
	BundleStatusUploading   BundleStatus = "UPLOADING"
	BundleStatusClassifying BundleStatus = "CLASSIFYING"
	BundleStatusSorted      BundleStatus = "SORTED"
	BundleStatusNeedsReview BundleStatus = "NEEDS_REVIEW"
	BundleStatusFailed      BundleStatus = "FAILED"
)

// SessionStatus represents the per-entity-stream staging, dry-run, apply, and rollback state.
type SessionStatus string

const (
	SessionStatusInitialized     SessionStatus = "INITIALIZED"
	SessionStatusUploaded        SessionStatus = "UPLOADED"
	SessionStatusDiscovering     SessionStatus = "DISCOVERING"
	SessionStatusDiscovered      SessionStatus = "DISCOVERED"
	SessionStatusMappingRequired SessionStatus = "MAPPING_REQUIRED"
	SessionStatusMappingProposed SessionStatus = "MAPPING_PROPOSED"
	SessionStatusValidating      SessionStatus = "VALIDATING"
	SessionStatusDryRunReady     SessionStatus = "DRY_RUN_READY"
	SessionStatusDryRunning      SessionStatus = "DRY_RUNNING"
	SessionStatusApproved        SessionStatus = "APPROVED"
	SessionStatusSnapshotting    SessionStatus = "SNAPSHOTTING"
	SessionStatusApplying        SessionStatus = "APPLYING"
	SessionStatusApplied         SessionStatus = "APPLIED"
	SessionStatusRollingBack     SessionStatus = "ROLLING_BACK"
	SessionStatusRolledBack      SessionStatus = "ROLLED_BACK"
	SessionStatusCutoverComplete SessionStatus = "CUTOVER_COMPLETE"
	SessionStatusFailed          SessionStatus = "FAILED"
)

// NormalizeSessionStatus maps legacy lowercase statuses and mixed-case inputs to canonical uppercase.
func NormalizeSessionStatus(raw string) SessionStatus {
	return SessionStatus(strings.ToUpper(strings.TrimSpace(raw)))
}

// Pack identifies the canonical dependency-ordered entity domain.
type Pack string

const (
	PackWarehouses       Pack = "warehouses"
	PackProductsPrices   Pack = "products_prices"
	PackCustomers        Pack = "customers"
	PackStock            Pack = "stock"
	PackLots             Pack = "lots"
	PackHistoricalOrders Pack = "historical_orders"
	PackNasiyaAR         Pack = "nasiya_ar"
	PackOpenOrders       Pack = "open_orders"
)

// ParsePack normalizes and validates a pack identifier.
func ParsePack(raw string) (Pack, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(PackWarehouses):
		return PackWarehouses, nil
	case string(PackProductsPrices):
		return PackProductsPrices, nil
	case string(PackCustomers):
		return PackCustomers, nil
	case string(PackStock):
		return PackStock, nil
	case string(PackLots):
		return PackLots, nil
	case string(PackHistoricalOrders):
		return PackHistoricalOrders, nil
	case string(PackNasiyaAR):
		return PackNasiyaAR, nil
	case string(PackOpenOrders):
		return PackOpenOrders, nil
	default:
		return "", fmt.Errorf("unknown migration pack %q", raw)
	}
}

// Project corresponds to a MigrationProjects row.
type Project struct {
	SupplierID         string          `json:"supplier_id"`
	ProjectID          string          `json:"project_id"`
	Name               string          `json:"name"`
	Mode               string          `json:"mode"`
	Status             ProjectStatus   `json:"status"`
	Checklist          json.RawMessage `json:"checklist,omitempty"`
	DefaultWarehouseID string          `json:"default_warehouse_id,omitempty"`
	CreatedBy          string          `json:"created_by"`
	CutoverAt          *time.Time      `json:"cutover_at,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// Bundle corresponds to a MigrationBundles row.
type Bundle struct {
	SupplierID  string          `json:"supplier_id"`
	ProjectID   string          `json:"project_id"`
	BundleID    string          `json:"bundle_id"`
	Status      BundleStatus    `json:"status"`
	SortSummary json.RawMessage `json:"sort_summary,omitempty"`
	ErrorMsg    string          `json:"error_message,omitempty"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Artifact corresponds to a MigrationArtifacts row.
type Artifact struct {
	SupplierID    string          `json:"supplier_id"`
	ProjectID     string          `json:"project_id"`
	BundleID      string          `json:"bundle_id"`
	ArtifactID    string          `json:"artifact_id"`
	FileName      string          `json:"file_name"`
	ByteSize      int64           `json:"byte_size"`
	DeclaredMIME  string          `json:"declared_mime,omitempty"`
	DetectedMIME  string          `json:"detected_mime,omitempty"`
	ContentSHA256 string          `json:"content_sha256"`
	Encoding      string          `json:"encoding,omitempty"`
	GCSPath       string          `json:"gcs_path"`
	Streams       json.RawMessage `json:"streams,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// UploadTicket describes a pre-signed GCS upload target returned on bundle declaration.
type UploadTicket struct {
	ArtifactID       string            `json:"artifact_id"`
	FileName         string            `json:"file_name"`
	GCSPath          string            `json:"gcs_path"`
	UploadURL        string            `json:"upload_url"`
	Method           string            `json:"method"` // PUT (<= 50 MiB) or POST (resumable > 50 MiB)
	ContentType      string            `json:"content_type"`
	Resumable        bool              `json:"resumable"`
	RequiredHeaders  map[string]string `json:"required_headers,omitempty"`
	ExpiresAtUTC     time.Time         `json:"expires_at"`
}

// Session corresponds to a widened SupplierImportSessions row.
type Session struct {
	SupplierID         string          `json:"supplier_id"`
	SessionID          string          `json:"session_id"`
	Status             SessionStatus   `json:"status"`
	FileName           string          `json:"file_name"`
	TotalRows          int64           `json:"total_rows"`
	ErrorSummary       json.RawMessage `json:"error_summary,omitempty"`
	ProjectID          string          `json:"project_id,omitempty"`
	BundleID           string          `json:"bundle_id,omitempty"`
	ArtifactID         string          `json:"artifact_id,omitempty"`
	StreamKey          string          `json:"stream_key,omitempty"`
	StreamSplitKey     string          `json:"stream_split_key,omitempty"`
	Pack               Pack            `json:"pack,omitempty"`
	SheetName          string          `json:"sheet_name,omitempty"`
	GCSPath            string          `json:"gcs_path,omitempty"`
	ContentSHA256      string          `json:"content_sha256,omitempty"`
	DetectedMIME       string          `json:"detected_mime,omitempty"`
	Encoding           string          `json:"encoding,omitempty"`
	AIModel            string          `json:"ai_model,omitempty"`
	AIStatus           string          `json:"ai_status,omitempty"`
	DryRunReportID     string          `json:"dry_run_report_id,omitempty"`
	DryRunFingerprint  string          `json:"dry_run_fingerprint,omitempty"`
	ApplyMode          string          `json:"apply_mode,omitempty"`
	SetOnHandConfirmed bool            `json:"set_on_hand_confirmed,omitempty"`
	ResumeRowIndex     int64           `json:"resume_row_index,omitempty"`
	LastChunkID        string          `json:"last_chunk_id,omitempty"`
	RollbackUntil      *time.Time      `json:"rollback_until,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// StagedRow corresponds to a SupplierImportStagedRows row.
type StagedRow struct {
	SupplierID       string          `json:"supplier_id"`
	SessionID        string          `json:"session_id"`
	RowIndex         int64           `json:"row_index"`
	RawData          json.RawMessage `json:"raw_data,omitempty"`
	CleanedData      json.RawMessage `json:"cleaned_data,omitempty"`
	ValidationErrors []string        `json:"validation_errors,omitempty"`
	IsNewProduct     bool            `json:"is_new_product"`
	ExternalID       string          `json:"external_id,omitempty"`
	StreamSplitKey   string          `json:"stream_split_key,omitempty"`
	ApplyAction      string          `json:"apply_action,omitempty"`
	ApplyChunk       int64           `json:"apply_chunk,omitempty"`
	IsNewEntity      bool            `json:"is_new_entity,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// WarehouseLock represents an exclusive active apply lease on (supplier_id, warehouse_id).
type WarehouseLock struct {
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id"`
	SessionID   string    `json:"session_id"`
	AcquiredAt  time.Time `json:"acquired_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// BundleTree groups a bundle with its raw uploaded artifacts for API serialization.
type BundleTree struct {
	Bundle    Bundle     `json:"bundle"`
	Artifacts []Artifact `json:"artifacts"`
}

// DAGStatus summarizes readiness and dependency blockers for the project tree response.
type DAGStatus struct {
	Order        []Pack            `json:"order"`
	RunnableNow  []Pack            `json:"runnable_now"`
	BlockedBy    map[Pack][]Pack   `json:"blocked_by"`
	MissingPacks []Pack            `json:"missing_packs,omitempty"`
}

// ProjectTree is the complete hydrated response for GET /v1/supplier/migrations/projects/{projectId}.
type ProjectTree struct {
	Project  Project      `json:"project"`
	Bundles  []BundleTree `json:"bundles"`
	Sessions []Session    `json:"sessions"`
	DAG      DAGStatus    `json:"dag"`
}
