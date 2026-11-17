package migrationroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration"
)

type memRepo struct {
	projects    map[string]migration.Project
	bundles     map[string]migration.Bundle
	arts        map[string][]migration.Artifact
	sessions    map[string]migration.Session
	rows        map[string][]migration.StagedRow
	sessionMaps map[string]migration.SessionMappingDocument
	reports     map[string]migration.MigrationReport
	snapshots   map[string][]migration.MigrationSnapshotRecord
}

func newMemRepo() *memRepo {
	return &memRepo{
		projects:    map[string]migration.Project{},
		bundles:     map[string]migration.Bundle{},
		arts:        map[string][]migration.Artifact{},
		sessions:    map[string]migration.Session{},
		rows:        map[string][]migration.StagedRow{},
		sessionMaps: map[string]migration.SessionMappingDocument{},
		reports:     map[string]migration.MigrationReport{},
		snapshots:   map[string][]migration.MigrationSnapshotRecord{},
	}
}

func (m *memRepo) CreateProject(_ context.Context, p migration.Project) (migration.Project, error) {
	now := time.Now().UTC()
	p.CreatedAt, p.UpdatedAt = now, now
	m.projects[p.ProjectID] = p
	return p, nil
}
func (m *memRepo) GetProject(_ context.Context, supplierID, projectID string) (migration.Project, error) {
	p, ok := m.projects[projectID]
	if !ok || p.SupplierID != supplierID {
		return migration.Project{}, migration.ErrNotFound
	}
	return p, nil
}
func (m *memRepo) ListProjects(_ context.Context, supplierID string, _ int) ([]migration.Project, error) {
	var out []migration.Project
	for _, p := range m.projects {
		if p.SupplierID == supplierID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *memRepo) UpdateProjectStatus(_ context.Context, supplierID, projectID string, to migration.ProjectStatus) error {
	p, err := m.GetProject(context.Background(), supplierID, projectID)
	if err != nil {
		return err
	}
	p.Status = to
	m.projects[projectID] = p
	return nil
}
func (m *memRepo) CreateBundleWithArtifacts(_ context.Context, b migration.Bundle, artifacts []migration.Artifact) (migration.Bundle, []migration.Artifact, error) {
	if _, err := m.GetProject(context.Background(), b.SupplierID, b.ProjectID); err != nil {
		return migration.Bundle{}, nil, err
	}
	m.bundles[b.BundleID] = b
	m.arts[b.BundleID] = artifacts
	return b, artifacts, nil
}
func (m *memRepo) GetBundle(_ context.Context, supplierID, projectID, bundleID string) (migration.Bundle, error) {
	b, ok := m.bundles[bundleID]
	if !ok || b.SupplierID != supplierID || b.ProjectID != projectID {
		return migration.Bundle{}, migration.ErrNotFound
	}
	return b, nil
}
func (m *memRepo) ListBundlesByProject(_ context.Context, supplierID, projectID string) ([]migration.Bundle, error) {
	var out []migration.Bundle
	for _, b := range m.bundles {
		if b.SupplierID == supplierID && b.ProjectID == projectID {
			out = append(out, b)
		}
	}
	return out, nil
}
func (m *memRepo) ListArtifactsByBundle(_ context.Context, _, _, bundleID string) ([]migration.Artifact, error) {
	return m.arts[bundleID], nil
}
func (m *memRepo) MarkBundleUploadedAndEmit(_ context.Context, supplierID, projectID, bundleID string, _ map[string]migration.UploadedArtifactMeta) (migration.Bundle, error) {
	b, err := m.GetBundle(context.Background(), supplierID, projectID, bundleID)
	if err != nil {
		return migration.Bundle{}, err
	}
	b.Status = migration.BundleStatusClassifying
	m.bundles[bundleID] = b
	return b, nil
}
func (m *memRepo) MarkBundleFailed(context.Context, string, string, string, string) error {
	return nil
}
func (m *memRepo) UpsertSortedBundleSessions(_ context.Context, _ string, _ string, _ string, _ migration.BundleStatus, _ json.RawMessage, _ map[string]json.RawMessage, sessions []migration.Session) ([]migration.Session, error) {
	for _, s := range sessions {
		m.sessions[s.SessionID] = s
	}
	return sessions, nil
}
func (m *memRepo) GetSession(_ context.Context, supplierID, sessionID string) (migration.Session, error) {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.Session{}, migration.ErrNotFound
	}
	return s, nil
}
func (m *memRepo) ListSessionsByProject(_ context.Context, supplierID, projectID string) ([]migration.Session, error) {
	var out []migration.Session
	for _, s := range m.sessions {
		if s.SupplierID == supplierID && s.ProjectID == projectID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *memRepo) ListSessionsByBundle(_ context.Context, supplierID, projectID, bundleID string) ([]migration.Session, error) {
	var out []migration.Session
	for _, s := range m.sessions {
		if s.SupplierID == supplierID && s.ProjectID == projectID && s.BundleID == bundleID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *memRepo) UpdateSessionStatus(_ context.Context, supplierID, sessionID string, next migration.SessionStatus, errSummary json.RawMessage) (migration.Session, error) {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.Session{}, migration.ErrNotFound
	}
	s.Status = next
	if len(errSummary) > 0 {
		s.ErrorSummary = errSummary
	}
	m.sessions[sessionID] = s
	return s, nil
}
func (m *memRepo) SaveStagedRowsChunk(context.Context, string, string, []migration.StagedRow) (int64, error) {
	return 0, nil
}
func (m *memRepo) ListStagedRows(_ context.Context, _, sessionID string, limit, offset int) ([]migration.StagedRow, int64, error) {
	all := m.rows[sessionID]
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}
func (m *memRepo) UpdateApplyProgress(_ context.Context, supplierID, sessionID string, resumeRowIndex int64, lastChunkID string) error {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.ErrNotFound
	}
	s.ResumeRowIndex = resumeRowIndex
	s.LastChunkID = lastChunkID
	m.sessions[sessionID] = s
	return nil
}
func (m *memRepo) AcquireWarehouseLock(context.Context, string, string, string, time.Duration) (migration.WarehouseLock, error) {
	return migration.WarehouseLock{}, nil
}
func (m *memRepo) RenewWarehouseLock(context.Context, string, string, string, time.Duration) (migration.WarehouseLock, error) {
	return migration.WarehouseLock{}, nil
}
func (m *memRepo) ReleaseWarehouseLock(context.Context, string, string, string) error {
	return nil
}
func (m *memRepo) GetSessionMapping(_ context.Context, _, sessionID string) (migration.SessionMappingDocument, bool, error) {
	d, ok := m.sessionMaps[sessionID]
	return d, ok, nil
}
func (m *memRepo) SaveSessionMapping(_ context.Context, _, sessionID string, next migration.SessionStatus, doc migration.SessionMappingDocument) error {
	m.sessionMaps[sessionID] = doc
	if s, ok := m.sessions[sessionID]; ok {
		s.Status = next
		m.sessions[sessionID] = s
	}
	return nil
}
func (m *memRepo) GetMappingLibrary(context.Context, string, migration.Pack, string) (*migration.MappingLibraryEntry, error) {
	return nil, nil
}
func (m *memRepo) UpsertMappingLibrary(context.Context, migration.MappingLibraryEntry) error {
	return nil
}
func (m *memRepo) SaveMigrationReport(_ context.Context, rep migration.MigrationReport) error {
	m.reports[rep.ReportID] = rep
	return nil
}
func (m *memRepo) GetMigrationReport(_ context.Context, supplierID, reportID string) (migration.MigrationReport, error) {
	rep, ok := m.reports[reportID]
	if !ok || rep.SupplierID != supplierID {
		return migration.MigrationReport{}, migration.ErrNotFound
	}
	return rep, nil
}
func (m *memRepo) MarkSessionDryRunReady(_ context.Context, supplierID, sessionID, reportID, fingerprint, applyMode string) error {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.ErrNotFound
	}
	s.Status = migration.SessionStatusDryRunReady
	s.DryRunReportID = reportID
	s.DryRunFingerprint = fingerprint
	s.ApplyMode = migration.NormalizeApplyMode(applyMode)
	m.sessions[sessionID] = s
	return nil
}
func (m *memRepo) ApproveSession(_ context.Context, supplierID, sessionID, applyMode string, setOnHandConfirmed bool) error {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.ErrNotFound
	}
	s.Status = migration.SessionStatusApproved
	s.ApplyMode = migration.NormalizeApplyMode(applyMode)
	s.SetOnHandConfirmed = setOnHandConfirmed
	m.sessions[sessionID] = s
	return nil
}
func (m *memRepo) MarkSessionApplied(_ context.Context, supplierID, sessionID string, rollbackUntil time.Time) error {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.ErrNotFound
	}
	t := rollbackUntil.UTC()
	s.Status = migration.SessionStatusApplied
	s.RollbackUntil = &t
	m.sessions[sessionID] = s
	return nil
}
func (m *memRepo) SaveSnapshotsChunk(_ context.Context, _, sessionID string, snaps []migration.MigrationSnapshotRecord) error {
	m.snapshots[sessionID] = append(m.snapshots[sessionID], snaps...)
	return nil
}
func (m *memRepo) ListSnapshotsBySession(_ context.Context, _, sessionID string) ([]migration.MigrationSnapshotRecord, error) {
	return m.snapshots[sessionID], nil
}
func (m *memRepo) ExecuteSessionRollback(_ context.Context, supplierID, sessionID string, _ []migration.MigrationSnapshotRecord, _ time.Time) (migration.RollbackResult, error) {
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return migration.RollbackResult{}, migration.ErrNotFound
	}
	s.Status = migration.SessionStatusRolledBack
	m.sessions[sessionID] = s
	return migration.RollbackResult{Session: s}, nil
}

func TestMigrationRoutes_AuthIdempotencyAndBundleLifecycle(t *testing.T) {
	t.Setenv("ALLOW_AUTH_BYPASS", "false")
	const secret = "test-secret-32-chars-minimum-len!"

	repo := newMemRepo()
	idSeq := 0
	h := migration.NewHandlers(migration.HandlerDeps{
		Repo: repo,
		NewID: func() string {
			idSeq++
			return "id-" + string(rune('0'+idSeq))
		},
		SignURL: func(bucket, objectName string, opts *gcs.SignedURLOptions) (string, error) {
			return "https://signed.example/" + objectName, nil
		},
	})

	r := chi.NewRouter()
	r.Use(auth.SessionAuth(secret))
	RegisterRoutes(r, Deps{Handlers: h, JWTSecret: secret, AllowAuthBypass: false})

	token, err := auth.Issue(auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
		HomeCell:   "cell-uz",
	}, auth.IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Missing Idempotency-Key -> 400 idempotency_key_required
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects", bytes.NewBufferString(`{"name":"Q4 Cutover"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing Idempotency-Key got %d want 400 body=%s", rec.Code, rec.Body.String())
	}

	// 2. Create project -> 201 Created
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects", bytes.NewBufferString(`{"name":"Q4 Cutover"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "proj-1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project got %d want 201 body=%s", rec.Code, rec.Body.String())
	}
	var proj migration.Project
	_ = json.Unmarshal(rec.Body.Bytes(), &proj)
	if proj.ProjectID == "" || proj.Mode != migration.ProjectModeCutover {
		t.Fatalf("unexpected project: %+v", proj)
	}

	// 3. Create bundle with multi-format files -> 201 + signed tickets
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects/"+proj.ProjectID+"/bundles",
		bytes.NewBufferString(`{"files":[{"file_name":"all_dump.xlsx","byte_size":2048},{"file_name":"onec.xml","byte_size":62914560}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", "bnd-1")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create bundle got %d want 201 body=%s", rec.Code, rec.Body.String())
	}

	// 4. Bypass active -> 403 auth_bypass_forbidden
	t.Setenv("ALLOW_AUTH_BYPASS", "true")
	req = httptest.NewRequest(http.MethodGet, "/v1/supplier/migrations/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("bypass enabled got %d want 403 body=%s", rec.Code, rec.Body.String())
	}
}

func TestMigrationRoutes_M4DryRunApproveRollbackAuthz(t *testing.T) {
	t.Setenv("ALLOW_AUTH_BYPASS", "false")
	const secret = "test-secret-32-chars-minimum-len!"
	repo := newMemRepo()
	now := time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC)

	repo.projects["proj-m4"] = migration.Project{
		SupplierID: "sup-1", ProjectID: "proj-m4", Name: "M4 Test", Mode: migration.ProjectModeCutover, Status: migration.ProjectStatusDraft,
	}
	repo.sessions["sess-wh"] = migration.Session{
		SupplierID: "sup-1", ProjectID: "proj-m4", SessionID: "sess-wh", Pack: migration.PackWarehouses, Status: migration.SessionStatusDiscovered, ContentSHA256: "sha-wh", TotalRows: 1,
	}
	repo.sessionMaps["sess-wh"] = migration.SessionMappingDocument{
		SessionID: "sess-wh",
		Pack:      migration.PackWarehouses,
		Mappings:  []migration.ColumnMappingProposal{{SourceColumn: "name", TargetField: "name", Confidence: 1, Confirmed: true}},
	}
	rawWH, _ := json.Marshal(map[string]any{"name": "Main Hub"})
	repo.rows["sess-wh"] = []migration.StagedRow{{SupplierID: "sup-1", SessionID: "sess-wh", RowIndex: 1, RawData: rawWH}}

	seq := 0
	h := migration.NewHandlers(migration.HandlerDeps{
		Repo: repo,
		Now:  func() time.Time { return now },
		NewID: func() string {
			seq++
			return "m4-" + string(rune('0'+seq))
		},
	})
	r := chi.NewRouter()
	r.Use(auth.SessionAuth(secret))
	RegisterRoutes(r, Deps{Handlers: h, JWTSecret: secret, AllowAuthBypass: false})

	adminToken, _ := auth.Issue(auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1", HomeCell: "cell-uz"}, auth.IssueOptions{Secret: secret, TTL: time.Hour})
	whAdminToken, _ := auth.Issue(auth.Claims{Subject: "wh-1", Role: auth.RoleAdmin, SupplierRole: "WAREHOUSE_ADMIN", SupplierID: "sup-1", HomeCell: "cell-uz"}, auth.IssueOptions{Secret: secret, TTL: time.Hour})

	// 1. Project dry-run -> 200 OK
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects/proj-m4/dry-run", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "dry-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dry-run got %d want 200 body=%s", rec.Code, rec.Body.String())
	}
	var drResp struct {
		Report migration.MigrationReport `json:"report"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &drResp)
	if drResp.Report.ReportID == "" || drResp.Report.BlockingCount != 0 {
		t.Fatalf("unexpected dry-run response: %s", rec.Body.String())
	}

	// 2. Get report -> 200 OK
	req = httptest.NewRequest(http.MethodGet, "/v1/supplier/migrations/reports/"+drResp.Report.ReportID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get report got %d want 200 body=%s", rec.Code, rec.Body.String())
	}

	// 3. Sub-admin blocked from approving SET_ON_HAND -> 403 financial_permission_required
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/sessions/sess-wh/approve", bytes.NewBufferString(`{"apply_mode":"SET_ON_HAND","set_on_hand_confirmed":true}`))
	req.Header.Set("Authorization", "Bearer "+whAdminToken)
	req.Header.Set("Idempotency-Key", "app-deny")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sub-admin approve SET_ON_HAND got %d want 403 body=%s", rec.Code, rec.Body.String())
	}

	// 4. Org admin approves -> 200 APPROVED
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/sessions/sess-wh/approve", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "app-ok")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve got %d want 200 body=%s", rec.Code, rec.Body.String())
	}

	// 5. Simulate applied + Rollback within 72h: sub-admin blocked 403, org admin succeeds 200
	until := now.Add(72 * time.Hour)
	s := repo.sessions["sess-wh"]
	s.Status = migration.SessionStatusApplied
	s.RollbackUntil = &until
	repo.sessions["sess-wh"] = s

	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/sessions/sess-wh/rollback", nil)
	req.Header.Set("Authorization", "Bearer "+whAdminToken)
	req.Header.Set("Idempotency-Key", "rb-deny")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sub-admin rollback got %d want 403 body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/sessions/sess-wh/rollback", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "rb-ok")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin rollback got %d want 200 body=%s", rec.Code, rec.Body.String())
	}
}

func TestMigrationRoutes_M6ApplyReleaseCutoverAndRBAC(t *testing.T) {
	t.Setenv("ALLOW_AUTH_BYPASS", "false")
	const secret = "test-secret-32-chars-minimum-len!"
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

	repo := newMemRepo()
	store := migration.NewMemoryApplyStore()
	store.NowFn = func() time.Time { return now }
	store.Supplier["sup-1"] = migration.MemorySupplierMeta{HomeCell: "cell-uz", Currency: "UZS"}
	store.Warehouses["wh-1"] = migration.StoredWarehouse{WarehouseID: "wh-1", SupplierID: "sup-1", Name: "Main", IsActive: true}
	store.Products["prod-tea"] = migration.StoredProduct{ProductID: "prod-tea", SupplierID: "sup-1", Name: "Tea", PriceMinor: 20000, Currency: "UZS", Unit: "PCS", IsActive: true}
	store.Inventory["sup-1|wh-1|prod-tea"] = migration.StoredStock{SupplierID: "sup-1", WarehouseID: "wh-1", ProductID: "prod-tea", QuantityOnHand: 10}
	store.Retailers["ret-1"] = migration.StoredCustomer{RetailerID: "ret-1", Phone: "+998901112233", Name: "Baraka"}
	store.Links["sup-1|ret-1"] = migration.StoredCustomerLink{SupplierID: "sup-1", RetailerID: "ret-1", TaxIdTIN: "301234567"}

	repo.projects["proj-m6"] = migration.Project{
		SupplierID: "sup-1", ProjectID: "proj-m6", Name: "M6 HTTP Cutover", Mode: migration.ProjectModeCutover, Status: migration.ProjectStatusDryRunReady,
	}

	seedSession := func(sid string, pack migration.Pack, rows []map[string]any, cols map[string]string) {
		var props []migration.ColumnMappingProposal
		for k, v := range cols {
			props = append(props, migration.ColumnMappingProposal{SourceColumn: k, TargetField: v, Confidence: 1, Confirmed: true})
		}
		doc := migration.SessionMappingDocument{SessionID: sid, Pack: pack, Mappings: props, AllRequiredMapped: true}
		doc.MappingHash = migration.ComputeMappingHash(doc)
		repo.sessions[sid] = migration.Session{
			SupplierID: "sup-1", ProjectID: "proj-m6", SessionID: sid, Pack: pack,
			Status: migration.SessionStatusDiscovered, ContentSHA256: "sha-" + sid, TotalRows: int64(len(rows)),
		}
		repo.sessionMaps[sid] = doc
		for i, row := range rows {
			b, _ := json.Marshal(row)
			repo.rows[sid] = append(repo.rows[sid], migration.StagedRow{SupplierID: "sup-1", SessionID: sid, RowIndex: int64(i + 1), RawData: b})
		}
		rep, summary, err := migration.EvaluateSessionDryRun(context.Background(), repo, migration.DryRunOptions{
			SupplierID: "sup-1", ProjectID: "proj-m6", SessionID: sid, Now: now, LiveReader: store,
		})
		if err != nil || !summary.ReadyToApprove {
			t.Fatalf("dry-run %s failed: err=%v summary=%+v", sid, err, summary)
		}
		if _, err := migration.ApproveSession(context.Background(), repo, "sup-1", sid, migration.ApproveSessionRequest{
			ExpectedReportID: rep.ReportID,
		}, now); err != nil {
			t.Fatalf("approve %s: %v", sid, err)
		}
	}

	seedSession("sess-nasiya-http", migration.PackNasiyaAR, []map[string]any{
		{"external_ar_id": "HTTP-AR-1", "customer_inn": "301234567", "balance_due": "500000", "currency": "UZS"},
	}, map[string]string{
		"external_ar_id": "external_ar_id", "customer_inn": "customer_inn", "balance_due": "balance_due", "currency": "currency",
	})
	seedSession("sess-open-http", migration.PackOpenOrders, []map[string]any{
		{"external_order_no": "HTTP-ORD-1", "customer_inn": "301234567", "warehouse_id": "wh-1", "sku": "prod-tea", "quantity": "3", "delivery_date": "2026-11-20", "total_amount": "60000"},
	}, map[string]string{
		"external_order_no": "external_order_no", "customer_inn": "customer_inn", "warehouse_id": "warehouse_id",
		"sku": "sku", "quantity": "quantity", "delivery_date": "delivery_date", "total_amount": "total_amount",
	})

	h := migration.NewHandlers(migration.HandlerDeps{
		Repo:       repo,
		LiveReader: store,
		ApplyStore: store,
		Now:        func() time.Time { return now },
	})
	r := chi.NewRouter()
	r.Use(auth.SessionAuth(secret))
	RegisterRoutes(r, Deps{Handlers: h, JWTSecret: secret, AllowAuthBypass: false})

	adminToken, _ := auth.Issue(auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1", HomeCell: "cell-uz"}, auth.IssueOptions{Secret: secret, TTL: time.Hour})
	whAdminToken, _ := auth.Issue(auth.Claims{Subject: "wh-1", Role: auth.RoleAdmin, SupplierRole: "WAREHOUSE_ADMIN", SupplierID: "sup-1", HomeCell: "cell-uz"}, auth.IssueOptions{Secret: secret, TTL: time.Hour})

	// 1. Sub-admin blocked from financial endpoints (nasiya apply, open-orders release, cutover) -> 403.
	for _, tc := range []struct {
		url string
		key string
	}{
		{"/v1/supplier/migrations/sessions/sess-nasiya-http/apply", "rbac-ar"},
		{"/v1/supplier/migrations/projects/proj-m6/open-orders/release", "rbac-rel"},
		{"/v1/supplier/migrations/projects/proj-m6/cutover", "rbac-cut"},
	} {
		req := httptest.NewRequest(http.MethodPost, tc.url, bytes.NewBufferString(`{}`))
		req.Header.Set("Authorization", "Bearer "+whAdminToken)
		req.Header.Set("Idempotency-Key", tc.key)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("sub-admin %s got %d want 403 body=%s", tc.url, rec.Code, rec.Body.String())
		}
	}

	// 2. Cutover before all sessions are applied -> 409 cutover_checklist_failed with checklist items.
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects/proj-m6/cutover", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "cut-fail")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("incomplete cutover got %d want 409 body=%s", rec.Code, rec.Body.String())
	}

	// 3. Admin applies both sessions -> 200 OK.
	for _, sid := range []string{"sess-nasiya-http", "sess-open-http"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/sessions/"+sid+"/apply", bytes.NewBufferString(`{}`))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Idempotency-Key", "apply-"+sid)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("apply %s got %d want 200 body=%s", sid, rec.Code, rec.Body.String())
		}
	}

	// 4. Release MIGRATION_OPEN orders via HTTP -> 200 OK with 1 released order.
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects/proj-m6/open-orders/release", bytes.NewBufferString(`{"target_status":"SCHEDULED"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "rel-ok")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("open-orders release got %d want 200 body=%s", rec.Code, rec.Body.String())
	}
	var relBody struct {
		ReleasedOrderIDs []string `json:"released_order_ids"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &relBody)
	if len(relBody.ReleasedOrderIDs) != 1 {
		t.Fatalf("expected 1 released order, got %s", rec.Body.String())
	}

	// 5. Cutover succeeds -> 200 CUTOVER_COMPLETE.
	req = httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects/proj-m6/cutover", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Idempotency-Key", "cut-ok")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cutover got %d want 200 body=%s", rec.Code, rec.Body.String())
	}
	if repo.projects["proj-m6"].Status != migration.ProjectStatusCutoverComplete {
		t.Fatalf("expected CUTOVER_COMPLETE, got %s", repo.projects["proj-m6"].Status)
	}
}

func TestHandleBundleUploaded_AcceptsSyncTrueAndSync1(t *testing.T) {
	const secret = "routes-sync-test-secret-0123456789abcdef"
	now := time.Date(2026, 11, 15, 11, 0, 0, 0, time.UTC)
	csvData := []byte("sku,name,price_minor,currency\nTEA-1,Green Tea,12000,UZS\n")

	for _, qs := range []string{"sync=true", "sync=1"} {
		repo := newMemRepo()
		_, _ = repo.CreateProject(context.Background(), migration.Project{
			SupplierID: "sup-1",
			ProjectID:  "proj-sync",
			Name:       "Sync Test",
			Mode:       migration.ProjectModeCutover,
			Status:     migration.ProjectStatusDraft,
			CreatedBy:  "admin-1",
		})
		_, _, _ = repo.CreateBundleWithArtifacts(context.Background(), migration.Bundle{
			SupplierID: "sup-1",
			ProjectID:  "proj-sync",
			BundleID:   "bnd-sync",
			Status:     migration.BundleStatusUploading,
			CreatedBy:  "admin-1",
		}, []migration.Artifact{{
			SupplierID:   "sup-1",
			ProjectID:    "proj-sync",
			BundleID:     "bnd-sync",
			ArtifactID:   "art-1",
			FileName:     "catalog.csv",
			ByteSize:     int64(len(csvData)),
			DeclaredMIME: "text/csv",
			GCSPath:      "migrations/cell-uz/sup-1/projects/proj-sync/bundles/bnd-sync/art-1/catalog.csv",
		}})

		h := migration.NewHandlers(migration.HandlerDeps{
			Repo: repo,
			ReadArtifact: func(_ context.Context, _ string) ([]byte, error) {
				return csvData, nil
			},
			Now: func() time.Time { return now },
		})
		r := chi.NewRouter()
		r.Use(auth.SessionAuth(secret))
		RegisterRoutes(r, Deps{Handlers: h, JWTSecret: secret, AllowAuthBypass: false})

		adminToken, _ := auth.Issue(auth.Claims{
			Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1", HomeCell: "cell-uz",
		}, auth.IssueOptions{Secret: secret, TTL: time.Hour})

		req := httptest.NewRequest(http.MethodPost, "/v1/supplier/migrations/projects/proj-sync/bundles/bnd-sync/uploaded?"+qs, bytes.NewBufferString(`{}`))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Idempotency-Key", "idem-"+qs)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d want 200 body=%s", qs, rec.Code, rec.Body.String())
		}
		var out struct {
			Sessions []migration.Session `json:"sessions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s decode: %v", qs, err)
		}
		if len(out.Sessions) == 0 {
			t.Fatalf("%s expected staged sessions synchronously, got body=%s", qs, rec.Body.String())
		}
	}
}

