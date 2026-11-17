package migration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

type memIngestRepo struct {
	mu             sync.Mutex
	bundles        map[string]Bundle
	artifacts      map[string][]Artifact
	sessions       map[string]Session
	stagedRows     map[string][]StagedRow
	sessionMaps    map[string]SessionMappingDocument
	mappingLibrary map[string]MappingLibraryEntry
	chunkWrites    int
	maxChunkLen    int
}

func newMemIngestRepo() *memIngestRepo {
	return &memIngestRepo{
		bundles:        make(map[string]Bundle),
		artifacts:      make(map[string][]Artifact),
		sessions:       make(map[string]Session),
		stagedRows:     make(map[string][]StagedRow),
		sessionMaps:    make(map[string]SessionMappingDocument),
		mappingLibrary: make(map[string]MappingLibraryEntry),
	}
}

func (m *memIngestRepo) CreateProject(_ context.Context, p Project) (Project, error) {
	return p, nil
}
func (m *memIngestRepo) GetProject(context.Context, string, string) (Project, error) {
	return Project{}, nil
}
func (m *memIngestRepo) ListProjects(context.Context, string, int) ([]Project, error) {
	return nil, nil
}
func (m *memIngestRepo) UpdateProjectStatus(context.Context, string, string, ProjectStatus) error {
	return nil
}
func (m *memIngestRepo) CreateBundleWithArtifacts(_ context.Context, b Bundle, arts []Artifact) (Bundle, []Artifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bundles[b.BundleID] = b
	m.artifacts[b.BundleID] = arts
	return b, arts, nil
}
func (m *memIngestRepo) GetBundle(_ context.Context, _, _, bundleID string) (Bundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bundles[bundleID]
	if !ok {
		return Bundle{}, ErrNotFound
	}
	return b, nil
}
func (m *memIngestRepo) ListBundlesByProject(context.Context, string, string) ([]Bundle, error) {
	return nil, nil
}
func (m *memIngestRepo) ListArtifactsByBundle(_ context.Context, _, _, bundleID string) ([]Artifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.artifacts[bundleID], nil
}
func (m *memIngestRepo) MarkBundleUploadedAndEmit(context.Context, string, string, string, map[string]UploadedArtifactMeta) (Bundle, error) {
	return Bundle{}, nil
}
func (m *memIngestRepo) MarkBundleFailed(_ context.Context, _, _, bundleID, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.bundles[bundleID]
	b.Status = BundleStatusFailed
	b.ErrorMsg = errMsg
	m.bundles[bundleID] = b
	return nil
}
func (m *memIngestRepo) UpsertSortedBundleSessions(_ context.Context, _, _, bundleID string, next BundleStatus, summary json.RawMessage, _ map[string]json.RawMessage, sessions []Session) ([]Session, error) {
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
func (m *memIngestRepo) GetSession(_ context.Context, _, sessionID string) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[sessionID], nil
}
func (m *memIngestRepo) ListSessionsByProject(context.Context, string, string) ([]Session, error) {
	return nil, nil
}
func (m *memIngestRepo) ListSessionsByBundle(_ context.Context, _, _, bundleID string) ([]Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Session
	for _, s := range m.sessions {
		if s.BundleID == bundleID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *memIngestRepo) UpdateSessionStatus(context.Context, string, string, SessionStatus, json.RawMessage) (Session, error) {
	return Session{}, nil
}
func (m *memIngestRepo) SaveStagedRowsChunk(_ context.Context, supplierID, sessionID string, rows []StagedRow) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunkWrites++
	if len(rows) > m.maxChunkLen {
		m.maxChunkLen = len(rows)
	}
	m.stagedRows[sessionID] = append(m.stagedRows[sessionID], rows...)
	s := m.sessions[sessionID]
	s.TotalRows = int64(len(m.stagedRows[sessionID]))
	m.sessions[sessionID] = s
	_ = supplierID
	return s.TotalRows, nil
}
func (m *memIngestRepo) ListStagedRows(_ context.Context, _, sessionID string, _, _ int) ([]StagedRow, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := m.stagedRows[sessionID]
	return rows, int64(len(rows)), nil
}
func (m *memIngestRepo) UpdateApplyProgress(context.Context, string, string, int64, string) error {
	return nil
}
func (m *memIngestRepo) AcquireWarehouseLock(context.Context, string, string, string, time.Duration) (WarehouseLock, error) {
	return WarehouseLock{}, nil
}
func (m *memIngestRepo) RenewWarehouseLock(context.Context, string, string, string, time.Duration) (WarehouseLock, error) {
	return WarehouseLock{}, nil
}
func (m *memIngestRepo) ReleaseWarehouseLock(context.Context, string, string, string) error {
	return nil
}
func (m *memIngestRepo) GetSessionMapping(_ context.Context, _, sessionID string) (SessionMappingDocument, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.sessionMaps[sessionID]
	return d, ok, nil
}
func (m *memIngestRepo) SaveSessionMapping(_ context.Context, _, sessionID string, nextStatus SessionStatus, doc SessionMappingDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionMaps[sessionID] = doc
	if s, ok := m.sessions[sessionID]; ok {
		s.Status = nextStatus
		s.AIModel = doc.AIModel
		s.AIStatus = doc.AIStatus
		m.sessions[sessionID] = s
	}
	return nil
}
func (m *memIngestRepo) GetMappingLibrary(_ context.Context, supplierID string, pack Pack, fp string) (*MappingLibraryEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s::%s::%s", supplierID, pack, fp)
	entry, ok := m.mappingLibrary[key]
	if !ok {
		return nil, nil
	}
	cp := entry
	return &cp, nil
}
func (m *memIngestRepo) UpsertMappingLibrary(_ context.Context, entry MappingLibraryEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s::%s::%s", entry.SupplierID, entry.Pack, entry.HeaderFingerprint)
	m.mappingLibrary[key] = entry
	return nil
}
func (m *memIngestRepo) SaveMigrationReport(context.Context, MigrationReport) error { return nil }
func (m *memIngestRepo) GetMigrationReport(context.Context, string, string) (MigrationReport, error) {
	return MigrationReport{}, ErrNotFound
}
func (m *memIngestRepo) MarkSessionDryRunReady(context.Context, string, string, string, string, string) error {
	return nil
}
func (m *memIngestRepo) ApproveSession(context.Context, string, string, string, bool) error {
	return nil
}
func (m *memIngestRepo) MarkSessionApplied(context.Context, string, string, time.Time) error {
	return nil
}
func (m *memIngestRepo) SaveSnapshotsChunk(context.Context, string, string, []MigrationSnapshotRecord) error {
	return nil
}
func (m *memIngestRepo) ListSnapshotsBySession(context.Context, string, string) ([]MigrationSnapshotRecord, error) {
	return nil, nil
}
func (m *memIngestRepo) ExecuteSessionRollback(context.Context, string, string, []MigrationSnapshotRecord, time.Time) (RollbackResult, error) {
	return RollbackResult{}, nil
}

func TestProcessBundleSortAndStage_25kRowsAndCompositeSplit(t *testing.T) {
	// Build a 25,000-row XLSX stock workbook (>10k cap!)
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_ = f.SetSheetName("Sheet1", "Stock")
	_ = f.SetCellValue("Stock", "A1", "sku")
	_ = f.SetCellValue("Stock", "B1", "warehouse_id")
	_ = f.SetCellValue("Stock", "C1", "quantity_on_hand")
	for i := 1; i <= 25000; i++ {
		_ = f.SetCellValue("Stock", fmt.Sprintf("A%d", i+1), fmt.Sprintf("SKU-%05d", i))
		_ = f.SetCellValue("Stock", fmt.Sprintf("B%d", i+1), "WH-MAIN")
		_ = f.SetCellValue("Stock", fmt.Sprintf("C%d", i+1), i)
	}
	var xlsxBuf bytes.Buffer
	if err := f.Write(&xlsxBuf); err != nil {
		t.Fatal(err)
	}

	// Also build a mixed Prodaji CSV with 2 DELIVERED and 1 OPEN order + customer data.
	csvData := []byte("customer_name;inn;phone;external_order_no;order_date;delivery_date;total_amount;status\n" +
		"Baraka Market;301234567;+998901234567;ORD-1;2026-10-01;;150000;DELIVERED\n" +
		"Alpomish Store;309876543;+998907654321;ORD-2;2026-11-12;2026-11-16;280000;OPEN\n" +
		"Samarqand Savdo;305556667;+998931112233;ORD-3;2026-10-15;;95000;COMPLETED\n")

	sumXLSX := sha256.Sum256(xlsxBuf.Bytes())
	sumCSV := sha256.Sum256(csvData)

	files := map[string][]byte{
		"gcs/big_stock.xlsx": xlsxBuf.Bytes(),
		"gcs/prodaji.csv":    csvData,
	}
	reader := func(_ context.Context, path string) ([]byte, error) {
		b, ok := files[path]
		if !ok {
			return nil, ErrNotFound
		}
		return b, nil
	}

	repo := newMemIngestRepo()
	_, _, _ = repo.CreateBundleWithArtifacts(context.Background(), Bundle{
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		BundleID:   "bnd-big",
		Status:     BundleStatusClassifying,
	}, []Artifact{
		{SupplierID: "sup-1", ProjectID: "proj-1", BundleID: "bnd-big", ArtifactID: "a-xlsx", FileName: "big_stock.xlsx", ByteSize: int64(xlsxBuf.Len()), ContentSHA256: hex.EncodeToString(sumXLSX[:]), GCSPath: "gcs/big_stock.xlsx"},
		{SupplierID: "sup-1", ProjectID: "proj-1", BundleID: "bnd-big", ArtifactID: "a-csv", FileName: "prodaji.csv", ByteSize: int64(len(csvData)), ContentSHA256: hex.EncodeToString(sumCSV[:]), GCSPath: "gcs/prodaji.csv"},
	})

	dec, err := ProcessBundleSortAndStage(context.Background(), repo, reader, "sup-1", "proj-1", "bnd-big", nil, seqIDGen("sess"))
	if err != nil {
		t.Fatalf("ProcessBundleSortAndStage: %v", err)
	}
	if dec.BundleStatus != BundleStatusSorted {
		t.Fatalf("expected SORTED, got %s", dec.BundleStatus)
	}

	countsBySplit := make(map[string]int)
	for _, s := range dec.Sessions {
		key := fmt.Sprintf("%s:%s", s.Pack, s.StreamSplitKey)
		countsBySplit[key] = len(repo.stagedRows[s.SessionID])
	}
	if countsBySplit["stock:stock"] != 25000 {
		t.Fatalf("expected 25,000 staged stock rows without 10k cap, got %d (maxChunk=%d)", countsBySplit["stock:stock"], repo.maxChunkLen)
	}
	if countsBySplit["historical_orders:orders_historical"] != 2 {
		t.Fatalf("expected 2 historical rows in split session, got %d", countsBySplit["historical_orders:orders_historical"])
	}
	if countsBySplit["open_orders:orders_open"] != 1 {
		t.Fatalf("expected 1 open row in split session, got %d", countsBySplit["open_orders:orders_open"])
	}
	if repo.maxChunkLen > 5000 {
		t.Fatalf("expected chunks capped at 5000, got maxChunkLen=%d", repo.maxChunkLen)
	}
}

func TestProcessBundleSortAndStage_DigestMismatchFailsClosed(t *testing.T) {
	repo := newMemIngestRepo()
	_, _, _ = repo.CreateBundleWithArtifacts(context.Background(), Bundle{
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		BundleID:   "bnd-bad",
		Status:     BundleStatusClassifying,
	}, []Artifact{
		{
			SupplierID:    "sup-1",
			ProjectID:     "proj-1",
			BundleID:      "bnd-bad",
			ArtifactID:    "a-1",
			FileName:      "x.csv",
			ByteSize:      10,
			ContentSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			GCSPath:       "gcs/x.csv",
		},
	})
	reader := func(context.Context, string) ([]byte, error) {
		return []byte("sku,name\n1,A\n"), nil
	}
	_, err := ProcessBundleSortAndStage(context.Background(), repo, reader, "sup-1", "proj-1", "bnd-bad", nil, seqIDGen("s"))
	if !errors.Is(err, ErrArtifactDigestMismatch) {
		t.Fatalf("expected ErrArtifactDigestMismatch, got %v", err)
	}
	b, _ := repo.GetBundle(context.Background(), "sup-1", "proj-1", "bnd-bad")
	if b.Status != BundleStatusFailed {
		t.Fatalf("expected bundle FAILED, got %s", b.Status)
	}
	_ = parser.FormatCSV
}

func TestProcessBundleSortAndStage_IdempotentOnceSortedDoesNotClobberState(t *testing.T) {
	t.Parallel()
	repo := newMemIngestRepo()
	_, _ = repo.CreateProject(context.Background(), Project{
		SupplierID: "sup-1", ProjectID: "proj-1", Name: "P", Mode: ProjectModeCutover, Status: ProjectStatusDraft,
	})
	_, _, _ = repo.CreateBundleWithArtifacts(context.Background(), Bundle{
		SupplierID: "sup-1", ProjectID: "proj-1", BundleID: "bnd-sorted", Status: BundleStatusSorted,
	}, []Artifact{{
		ArtifactID: "art-1", FileName: "warehouses.csv", ByteSize: 16, GCSPath: "mem://wh.csv",
	}})
	repo.sessions["sess-wh"] = Session{
		SupplierID:        "sup-1",
		ProjectID:         "proj-1",
		BundleID:          "bnd-sorted",
		SessionID:         "sess-wh",
		Pack:              PackWarehouses,
		Status:            SessionStatusDryRunReady,
		DryRunReportID:    "rep-1",
		DryRunFingerprint: "fp-1",
		ApplyMode:         ApplyModeMerge,
	}

	calledReader := false
	reader := func(context.Context, string) ([]byte, error) {
		calledReader = true
		return nil, errors.New("reader should not be called when already sorted")
	}
	dec, err := ProcessBundleSortAndStage(context.Background(), repo, reader, "sup-1", "proj-1", "bnd-sorted", nil, seqIDGen("s"))
	if err != nil {
		t.Fatalf("ProcessBundleSortAndStage idempotent: %v", err)
	}
	if calledReader {
		t.Fatal("expected already-sorted bundle not to re-download/re-parse artifacts")
	}
	if len(dec.Sessions) != 1 || dec.Sessions[0].DryRunReportID != "rep-1" || dec.Sessions[0].Status != SessionStatusDryRunReady {
		t.Fatalf("expected existing session state preserved, got %+v", dec.Sessions)
	}
}

func TestMergeExistingSessionLifecyclePreservesDryRunAndStatus(t *testing.T) {
	t.Parallel()
	dst := Session{
		SessionID: "sess-wh",
		Status:    SessionStatusMappingProposed,
	}
	prev := Session{
		SessionID:         "sess-wh",
		Status:            SessionStatusDryRunReady,
		DryRunReportID:    "rep-1",
		DryRunFingerprint: "fp-1",
		ApplyMode:         ApplyModeMerge,
		ResumeRowIndex:    250,
		LastChunkID:       "chk-1",
	}
	mergeExistingSessionLifecycle(&dst, prev)
	if dst.Status != SessionStatusDryRunReady || dst.DryRunReportID != "rep-1" || dst.DryRunFingerprint != "fp-1" || dst.ApplyMode != ApplyModeMerge || dst.LastChunkID != "chk-1" {
		t.Fatalf("expected downstream lifecycle preserved, got %+v", dst)
	}
}
