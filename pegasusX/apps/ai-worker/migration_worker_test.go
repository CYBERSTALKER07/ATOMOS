package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration"
)

type aiWorkerTestRepo struct {
	mu          sync.Mutex
	bundles     map[string]migration.Bundle
	artifacts   map[string][]migration.Artifact
	sessions    map[string]migration.Session
	stagedRows  map[string][]migration.StagedRow
	sessionMaps map[string]migration.SessionMappingDocument
	library     map[string]migration.MappingLibraryEntry
}

func newAIWorkerTestRepo() *aiWorkerTestRepo {
	return &aiWorkerTestRepo{
		bundles:     make(map[string]migration.Bundle),
		artifacts:   make(map[string][]migration.Artifact),
		sessions:    make(map[string]migration.Session),
		stagedRows:  make(map[string][]migration.StagedRow),
		sessionMaps: make(map[string]migration.SessionMappingDocument),
		library:     make(map[string]migration.MappingLibraryEntry),
	}
}

func (m *aiWorkerTestRepo) CreateProject(_ context.Context, p migration.Project) (migration.Project, error) {
	return p, nil
}
func (m *aiWorkerTestRepo) GetProject(context.Context, string, string) (migration.Project, error) {
	return migration.Project{}, nil
}
func (m *aiWorkerTestRepo) ListProjects(context.Context, string, int) ([]migration.Project, error) {
	return nil, nil
}
func (m *aiWorkerTestRepo) UpdateProjectStatus(context.Context, string, string, migration.ProjectStatus) error {
	return nil
}
func (m *aiWorkerTestRepo) CreateBundleWithArtifacts(_ context.Context, b migration.Bundle, arts []migration.Artifact) (migration.Bundle, []migration.Artifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bundles[b.BundleID] = b
	m.artifacts[b.BundleID] = arts
	return b, arts, nil
}
func (m *aiWorkerTestRepo) GetBundle(_ context.Context, _, _, bundleID string) (migration.Bundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bundles[bundleID]
	if !ok {
		return migration.Bundle{}, migration.ErrNotFound
	}
	return b, nil
}
func (m *aiWorkerTestRepo) ListBundlesByProject(context.Context, string, string) ([]migration.Bundle, error) {
	return nil, nil
}
func (m *aiWorkerTestRepo) ListArtifactsByBundle(_ context.Context, _, _, bundleID string) ([]migration.Artifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.artifacts[bundleID], nil
}
func (m *aiWorkerTestRepo) MarkBundleUploadedAndEmit(context.Context, string, string, string, map[string]migration.UploadedArtifactMeta) (migration.Bundle, error) {
	return migration.Bundle{}, nil
}
func (m *aiWorkerTestRepo) MarkBundleFailed(_ context.Context, _, _, bundleID, msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bundles[bundleID]
	b.Status = migration.BundleStatusFailed
	b.ErrorMsg = msg
	m.bundles[bundleID] = b
	return nil
}
func (m *aiWorkerTestRepo) UpsertSortedBundleSessions(_ context.Context, _, _, bundleID string, next migration.BundleStatus, summary json.RawMessage, _ map[string]json.RawMessage, sessions []migration.Session) ([]migration.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bundles[bundleID]
	b.Status = next
	b.SortSummary = summary
	m.bundles[bundleID] = b
	for _, s := range sessions {
		m.sessions[s.SessionID] = s
	}
	return sessions, nil
}
func (m *aiWorkerTestRepo) GetSession(_ context.Context, _, id string) (migration.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id], nil
}
func (m *aiWorkerTestRepo) ListSessionsByProject(context.Context, string, string) ([]migration.Session, error) {
	return nil, nil
}
func (m *aiWorkerTestRepo) ListSessionsByBundle(_ context.Context, _, _, bundleID string) ([]migration.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []migration.Session
	for _, s := range m.sessions {
		if s.BundleID == bundleID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *aiWorkerTestRepo) UpdateSessionStatus(context.Context, string, string, migration.SessionStatus, json.RawMessage) (migration.Session, error) {
	return migration.Session{}, nil
}
func (m *aiWorkerTestRepo) SaveStagedRowsChunk(_ context.Context, _, sessionID string, rows []migration.StagedRow) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stagedRows[sessionID] = append(m.stagedRows[sessionID], rows...)
	s := m.sessions[sessionID]
	s.TotalRows = int64(len(m.stagedRows[sessionID]))
	m.sessions[sessionID] = s
	return s.TotalRows, nil
}
func (m *aiWorkerTestRepo) ListStagedRows(_ context.Context, _, sessionID string, limit, _ int) ([]migration.StagedRow, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := m.stagedRows[sessionID]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, int64(len(m.stagedRows[sessionID])), nil
}
func (m *aiWorkerTestRepo) UpdateApplyProgress(context.Context, string, string, int64, string) error {
	return nil
}
func (m *aiWorkerTestRepo) AcquireWarehouseLock(context.Context, string, string, string, time.Duration) (migration.WarehouseLock, error) {
	return migration.WarehouseLock{}, nil
}
func (m *aiWorkerTestRepo) RenewWarehouseLock(context.Context, string, string, string, time.Duration) (migration.WarehouseLock, error) {
	return migration.WarehouseLock{}, nil
}
func (m *aiWorkerTestRepo) ReleaseWarehouseLock(context.Context, string, string, string) error {
	return nil
}
func (m *aiWorkerTestRepo) GetSessionMapping(_ context.Context, _, sessionID string) (migration.SessionMappingDocument, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.sessionMaps[sessionID]
	return d, ok, nil
}
func (m *aiWorkerTestRepo) SaveSessionMapping(_ context.Context, _, sessionID string, nextStatus migration.SessionStatus, doc migration.SessionMappingDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionMaps[sessionID] = doc
	s := m.sessions[sessionID]
	s.Status = nextStatus
	s.AIModel = doc.AIModel
	s.AIStatus = doc.AIStatus
	m.sessions[sessionID] = s
	return nil
}
func (m *aiWorkerTestRepo) GetMappingLibrary(_ context.Context, supplierID string, pack migration.Pack, fp string) (*migration.MappingLibraryEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.library[fmt.Sprintf("%s::%s::%s", supplierID, pack, fp)]
	if !ok {
		return nil, nil
	}
	return &e, nil
}
func (m *aiWorkerTestRepo) UpsertMappingLibrary(_ context.Context, entry migration.MappingLibraryEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.library[fmt.Sprintf("%s::%s::%s", entry.SupplierID, entry.Pack, entry.HeaderFingerprint)] = entry
	return nil
}
func (m *aiWorkerTestRepo) SaveMigrationReport(context.Context, migration.MigrationReport) error {
	return nil
}
func (m *aiWorkerTestRepo) GetMigrationReport(context.Context, string, string) (migration.MigrationReport, error) {
	return migration.MigrationReport{}, migration.ErrNotFound
}
func (m *aiWorkerTestRepo) MarkSessionDryRunReady(context.Context, string, string, string, string, string) error {
	return nil
}
func (m *aiWorkerTestRepo) ApproveSession(context.Context, string, string, string, bool) error {
	return nil
}
func (m *aiWorkerTestRepo) MarkSessionApplied(context.Context, string, string, time.Time) error {
	return nil
}
func (m *aiWorkerTestRepo) SaveSnapshotsChunk(context.Context, string, string, []migration.MigrationSnapshotRecord) error {
	return nil
}
func (m *aiWorkerTestRepo) ListSnapshotsBySession(context.Context, string, string) ([]migration.MigrationSnapshotRecord, error) {
	return nil, nil
}
func (m *aiWorkerTestRepo) ExecuteSessionRollback(context.Context, string, string, []migration.MigrationSnapshotRecord, time.Time) (migration.RollbackResult, error) {
	return migration.RollbackResult{}, nil
}

func TestMigrationBundleProcessor_ProcessEvent(t *testing.T) {
	repo := newAIWorkerTestRepo()
	csvBytes := []byte("customer_name,inn,phone\nBaraka LLC,301234567,+998901234567\n")
	sum := sha256.Sum256(csvBytes)
	_, _, _ = repo.CreateBundleWithArtifacts(context.Background(), migration.Bundle{
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		BundleID:   "bnd-1",
		Status:     migration.BundleStatusClassifying,
	}, []migration.Artifact{
		{
			SupplierID:    "sup-1",
			ProjectID:     "proj-1",
			BundleID:      "bnd-1",
			ArtifactID:    "art-1",
			FileName:      "customers.csv",
			ByteSize:      int64(len(csvBytes)),
			ContentSHA256: hex.EncodeToString(sum[:]),
			GCSPath:       "gcs/customers.csv",
		},
	})

	proc := &migrationBundleProcessor{
		repo: repo,
		readObject: func(context.Context, string) ([]byte, error) {
			return csvBytes, nil
		},
		mapper: migration.NewAIMapper(migration.GeminiMapperConfig{Enabled: false}),
		logger: slog.Default(),
	}

	payload, _ := json.Marshal(bundleUploadedEnvelope{
		Type:       migration.EventMigrationBundleUploaded,
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		BundleID:   "bnd-1",
		HomeCell:   "cell-uz",
	})
	if detectEnvelopeType(payload) != "MIGRATION_BUNDLE_UPLOADED" {
		t.Fatalf("unexpected envelope detection")
	}

	termErr, transErr := proc.ProcessEvent(context.Background(), payload)
	if termErr != nil || transErr != nil {
		t.Fatalf("expected clean ProcessEvent, got term=%v trans=%v", termErr, transErr)
	}
	if len(repo.sessions) != 1 || len(repo.sessionMaps) != 1 {
		t.Fatalf("expected 1 sorted+mapped session, got sessions=%d maps=%d", len(repo.sessions), len(repo.sessionMaps))
	}
}
