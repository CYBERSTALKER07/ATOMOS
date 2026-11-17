package migration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

func decodeJSONWithNumbers(raw []byte, dst *map[string]any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	for k, v := range *dst {
		if num, ok := v.(json.Number); ok {
			if iv, err := num.Int64(); err == nil {
				(*dst)[k] = iv
			}
		}
	}
	return nil
}

type m4TestRepo struct {
	mu             sync.Mutex
	projects       map[string]Project
	bundles        map[string]Bundle
	artifacts      map[string][]Artifact
	sessions       map[string]Session
	stagedRows     map[string][]StagedRow
	sessionMaps    map[string]SessionMappingDocument
	mappingLibrary map[string]MappingLibraryEntry
	reports        map[string]MigrationReport
	snapshots      map[string][]MigrationSnapshotRecord

	// Simulated live business tables for non-mutating overlay & rollback verification
	live            LiveOverlaySnapshot
	warehousesTable map[string]map[string]any
	productsTable   map[string]map[string]any
	inventoryTable  map[string]map[string]any
	ordersTable     map[string]map[string]any
	arInvoicesTable map[string]map[string]any
	arLedger        []map[string]any
	journalEntries  []*payment.JournalEntry
	mutationCount   int
}

func newM4TestRepo() *m4TestRepo {
	return &m4TestRepo{
		projects:        make(map[string]Project),
		bundles:         make(map[string]Bundle),
		artifacts:       make(map[string][]Artifact),
		sessions:        make(map[string]Session),
		stagedRows:      make(map[string][]StagedRow),
		sessionMaps:     make(map[string]SessionMappingDocument),
		mappingLibrary:  make(map[string]MappingLibraryEntry),
		reports:         make(map[string]MigrationReport),
		snapshots:       make(map[string][]MigrationSnapshotRecord),
		live:            NewEmptyLiveOverlay(),
		warehousesTable: make(map[string]map[string]any),
		productsTable:   make(map[string]map[string]any),
		inventoryTable:  make(map[string]map[string]any),
		ordersTable:     make(map[string]map[string]any),
		arInvoicesTable: make(map[string]map[string]any),
	}
}

func (m *m4TestRepo) LoadLiveOverlay(_ context.Context, _ string) (LiveOverlaySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.live.clone(), nil
}
func (m *m4TestRepo) CreateProject(_ context.Context, p Project) (Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[p.ProjectID] = p
	return p, nil
}
func (m *m4TestRepo) GetProject(_ context.Context, supplierID, projectID string) (Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[projectID]
	if !ok || p.SupplierID != supplierID {
		return Project{}, ErrNotFound
	}
	return p, nil
}
func (m *m4TestRepo) ListProjects(_ context.Context, supplierID string, _ int) ([]Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Project
	for _, p := range m.projects {
		if p.SupplierID == supplierID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *m4TestRepo) UpdateProjectStatus(_ context.Context, supplierID, projectID string, to ProjectStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[projectID]
	if !ok || p.SupplierID != supplierID {
		return ErrNotFound
	}
	if err := ValidateProjectTransition(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	m.projects[projectID] = p
	return nil
}
func (m *m4TestRepo) CreateBundleWithArtifacts(_ context.Context, b Bundle, arts []Artifact) (Bundle, []Artifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bundles[b.BundleID] = b
	m.artifacts[b.BundleID] = arts
	return b, arts, nil
}
func (m *m4TestRepo) GetBundle(_ context.Context, _, _, bundleID string) (Bundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bundles[bundleID]
	if !ok {
		return Bundle{}, ErrNotFound
	}
	return b, nil
}
func (m *m4TestRepo) ListBundlesByProject(context.Context, string, string) ([]Bundle, error) {
	return nil, nil
}
func (m *m4TestRepo) ListArtifactsByBundle(_ context.Context, _, _, bundleID string) ([]Artifact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.artifacts[bundleID], nil
}
func (m *m4TestRepo) MarkBundleUploadedAndEmit(context.Context, string, string, string, map[string]UploadedArtifactMeta) (Bundle, error) {
	return Bundle{}, nil
}
func (m *m4TestRepo) MarkBundleFailed(context.Context, string, string, string, string) error {
	return nil
}
func (m *m4TestRepo) UpsertSortedBundleSessions(context.Context, string, string, string, BundleStatus, json.RawMessage, map[string]json.RawMessage, []Session) ([]Session, error) {
	return nil, nil
}
func (m *m4TestRepo) GetSession(_ context.Context, supplierID, sessionID string) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return Session{}, ErrNotFound
	}
	return s, nil
}
func (m *m4TestRepo) ListSessionsByProject(_ context.Context, supplierID, projectID string) ([]Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Session
	for _, s := range m.sessions {
		if s.SupplierID == supplierID && s.ProjectID == projectID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *m4TestRepo) ListSessionsByBundle(context.Context, string, string, string) ([]Session, error) {
	return nil, nil
}
func (m *m4TestRepo) UpdateSessionStatus(_ context.Context, supplierID, sessionID string, next SessionStatus, errSummary json.RawMessage) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return Session{}, ErrNotFound
	}
	if err := ValidateSessionTransition(s.Status, next); err != nil {
		return Session{}, err
	}
	s.Status = next
	if len(errSummary) > 0 {
		s.ErrorSummary = errSummary
	}
	m.sessions[sessionID] = s
	return s, nil
}
func (m *m4TestRepo) SaveStagedRowsChunk(_ context.Context, _, sessionID string, rows []StagedRow) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stagedRows[sessionID] = append(m.stagedRows[sessionID], rows...)
	return int64(len(m.stagedRows[sessionID])), nil
}
func (m *m4TestRepo) ListStagedRows(_ context.Context, _, sessionID string, limit, offset int) ([]StagedRow, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := m.stagedRows[sessionID]
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
func (m *m4TestRepo) UpdateApplyProgress(context.Context, string, string, int64, string) error {
	return nil
}
func (m *m4TestRepo) AcquireWarehouseLock(context.Context, string, string, string, time.Duration) (WarehouseLock, error) {
	return WarehouseLock{}, nil
}
func (m *m4TestRepo) RenewWarehouseLock(context.Context, string, string, string, time.Duration) (WarehouseLock, error) {
	return WarehouseLock{}, nil
}
func (m *m4TestRepo) ReleaseWarehouseLock(context.Context, string, string, string) error {
	return nil
}
func (m *m4TestRepo) GetSessionMapping(_ context.Context, _, sessionID string) (SessionMappingDocument, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.sessionMaps[sessionID]
	return d, ok, nil
}
func (m *m4TestRepo) SaveSessionMapping(_ context.Context, _, sessionID string, next SessionStatus, doc SessionMappingDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionMaps[sessionID] = doc
	if s, ok := m.sessions[sessionID]; ok {
		s.Status = next
		m.sessions[sessionID] = s
	}
	return nil
}
func (m *m4TestRepo) GetMappingLibrary(context.Context, string, Pack, string) (*MappingLibraryEntry, error) {
	return nil, nil
}
func (m *m4TestRepo) UpsertMappingLibrary(context.Context, MappingLibraryEntry) error { return nil }
func (m *m4TestRepo) SaveMigrationReport(_ context.Context, rep MigrationReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[rep.ReportID] = rep
	return nil
}
func (m *m4TestRepo) GetMigrationReport(_ context.Context, supplierID, reportID string) (MigrationReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rep, ok := m.reports[reportID]
	if !ok || rep.SupplierID != supplierID {
		return MigrationReport{}, ErrNotFound
	}
	return rep, nil
}
func (m *m4TestRepo) MarkSessionDryRunReady(_ context.Context, supplierID, sessionID, reportID, fingerprint, applyMode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return ErrNotFound
	}
	if err := ValidateSessionTransition(s.Status, SessionStatusDryRunReady); err != nil {
		return err
	}
	s.Status = SessionStatusDryRunReady
	s.DryRunReportID = reportID
	s.DryRunFingerprint = fingerprint
	s.ApplyMode = NormalizeApplyMode(applyMode)
	m.sessions[sessionID] = s
	return nil
}
func (m *m4TestRepo) ApproveSession(_ context.Context, supplierID, sessionID, applyMode string, setOnHandConfirmed bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return ErrNotFound
	}
	if err := ValidateSessionTransition(s.Status, SessionStatusApproved); err != nil {
		return err
	}
	s.Status = SessionStatusApproved
	s.ApplyMode = NormalizeApplyMode(applyMode)
	s.SetOnHandConfirmed = setOnHandConfirmed
	m.sessions[sessionID] = s
	return nil
}

func (m *m4TestRepo) MarkSessionApplied(_ context.Context, supplierID, sessionID string, rollbackUntil time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return ErrNotFound
	}
	if err := ValidateSessionTransition(s.Status, SessionStatusApplied); err != nil {
		return err
	}
	s.Status = SessionStatusApplied
	t := rollbackUntil
	s.RollbackUntil = &t
	m.sessions[sessionID] = s
	return nil
}
func (m *m4TestRepo) SaveSnapshotsChunk(_ context.Context, _, sessionID string, snaps []MigrationSnapshotRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[sessionID] = append(m.snapshots[sessionID], snaps...)
	return nil
}
func (m *m4TestRepo) ListSnapshotsBySession(_ context.Context, _, sessionID string) ([]MigrationSnapshotRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := append([]MigrationSnapshotRecord(nil), m.snapshots[sessionID]...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].SnapshotSeq > cp[j].SnapshotSeq })
	return cp, nil
}
func (m *m4TestRepo) ExecuteSessionRollback(_ context.Context, supplierID, sessionID string, snapshots []MigrationSnapshotRecord, now time.Time) (RollbackResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := RollbackResult{}
	for _, snap := range snapshots {
		res.RevertedSnapshots++
		switch snap.TargetTable {
		case "Warehouses":
			id, _ := extractPKField(snap.RowKeyJSON, "WarehouseId")
			if snap.OpKind == SnapshotOpInsert {
				delete(m.warehousesTable, id)
			} else if len(snap.BeforeJSON) > 0 {
				var row map[string]any
				_ = json.Unmarshal(snap.BeforeJSON, &row)
				m.warehousesTable[id] = row
			}
		case "Products":
			id, _ := extractPKField(snap.RowKeyJSON, "ProductId")
			if snap.OpKind == SnapshotOpInsert {
				delete(m.productsTable, id)
			} else if len(snap.BeforeJSON) > 0 {
				var row map[string]any
				_ = json.Unmarshal(snap.BeforeJSON, &row)
				m.productsTable[id] = row
			}
		case "SupplierInventoryV2":
			whID, _ := extractPKField(snap.RowKeyJSON, "WarehouseId")
			prodID, _ := extractPKField(snap.RowKeyJSON, "ProductId")
			key := whID + "|" + prodID
			if snap.OpKind == SnapshotOpInsert {
				delete(m.inventoryTable, key)
			} else if len(snap.BeforeJSON) > 0 {
				var row map[string]any
				_ = decodeJSONWithNumbers(snap.BeforeJSON, &row)
				m.inventoryTable[key] = row
			}
		case "Orders":
			id, _ := extractPKField(snap.RowKeyJSON, "OrderId")
			if o, ok := m.ordersTable[id]; ok {
				o["SupersededByRollback"] = true
				if o["Status"] == "MIGRATION_OPEN" {
					o["Status"] = "CANCELLED"
				}
				res.SupersededOrders++
			}
		case "ArInvoices":
			invID, _ := extractPKField(snap.RowKeyJSON, "InvoiceId")
			if inv, ok := m.arInvoicesTable[invID]; ok && inv["Status"] != "ROLLED_BACK" {
				bal, _ := inv["BalanceMinor"].(int64)
				retID, _ := inv["RetailerId"].(string)
				ordID, _ := inv["OrderId"].(string)
				cur, _ := inv["Currency"].(string)
				inv["Status"] = "ROLLED_BACK"
				inv["BalanceMinor"] = int64(0)
				res.ReversedInvoices++
				if bal > 0 {
					m.arLedger = append(m.arLedger, map[string]any{
						"EntryId":     "ar_rev_" + invID,
						"InvoiceId":   invID,
						"EntryType":   "MIGRATION_ROLLBACK",
						"AmountMinor": -bal,
					})
					je, err := BuildReversalJournalEntry(supplierID, sessionID, invID, ordID, retID, cur, bal, now)
					if err != nil {
						return RollbackResult{}, err
					}
					m.journalEntries = append(m.journalEntries, je)
					res.ReversedAmountMinor += bal
					res.ReversalEntryIDs = append(res.ReversalEntryIDs, je.EntryID)
				}
			}
		}
	}
	s := m.sessions[sessionID]
	s.Status = SessionStatusRolledBack
	m.sessions[sessionID] = s
	res.Session = s
	return res, nil
}

func seedMappedSession(repo *m4TestRepo, supplierID, projID, sessID string, pack Pack, sha string, cols []string, rows []map[string]any) {
	mappings := make([]ColumnMappingProposal, 0, len(cols))
	for _, c := range cols {
		mappings = append(mappings, ColumnMappingProposal{
			SourceColumn: c,
			TargetField:  c,
			Confidence:   1.0,
			Confirmed:    true,
			Source:       "manual",
		})
	}
	doc := SessionMappingDocument{
		SessionID:         sessID,
		Pack:              pack,
		AllRequiredMapped: true,
		Mappings:          mappings,
	}
	doc.MappingHash = ComputeMappingHash(doc)
	repo.sessions[sessID] = Session{
		SupplierID:    supplierID,
		SessionID:     sessID,
		ProjectID:     projID,
		Pack:          pack,
		Status:        SessionStatusDiscovered,
		ContentSHA256: sha,
		TotalRows:     int64(len(rows)),
	}
	repo.sessionMaps[sessID] = doc
	staged := make([]StagedRow, 0, len(rows))
	for i, r := range rows {
		raw, _ := json.Marshal(r)
		staged = append(staged, StagedRow{
			SupplierID: supplierID,
			SessionID:  sessID,
			RowIndex:   int64(i + 1),
			RawData:    raw,
		})
	}
	repo.stagedRows[sessID] = staged
}

func TestEvaluateProjectDryRun_CrossPackVirtualOverlayAndNonMutating(t *testing.T) {
	repo := newM4TestRepo()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	_, _ = repo.CreateProject(context.Background(), Project{
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		Name:       "Enterprise Cutover",
		Mode:       ProjectModeCutover,
		Status:     ProjectStatusDraft,
	})

	// Seed live stock so insufficient open order check works across live + staged overlay
	repo.live.ProductsByID["live-cola"] = LiveProduct{
		ProductID: "live-cola", SKU: "LIVE-01", Name: "Live Soda", Unit: "pcs", PriceMinor: 500000, Currency: "UZS",
	}
	registerVirtualProduct(&repo.live, repo.live.ProductsByID["live-cola"])

	// Staged packs WITHOUT applying them yet!
	seedMappedSession(repo, "sup-1", "proj-1", "s-wh", PackWarehouses, "sha-wh",
		[]string{"external_warehouse_id", "name", "is_default"},
		[]map[string]any{{"external_warehouse_id": "WH-CENT", "name": "Tashkent Central", "is_default": "true"}},
	)
	seedMappedSession(repo, "sup-1", "proj-1", "s-prod", PackProductsPrices, "sha-prod",
		[]string{"sku", "name", "unit", "price"},
		[]map[string]any{{"sku": "TEA-100", "name": "Green Tea 100g", "unit": "box", "price": "15000"}},
	)
	seedMappedSession(repo, "sup-1", "proj-1", "s-cust", PackCustomers, "sha-cust",
		[]string{"customer_name", "inn", "phone"},
		[]map[string]any{{"customer_name": "Baraka Market", "inn": "301234567", "phone": "+998901112233"}},
	)
	seedMappedSession(repo, "sup-1", "proj-1", "s-stock", PackStock, "sha-stock",
		[]string{"sku", "warehouse_id", "quantity_on_hand"},
		[]map[string]any{{"sku": "TEA-100", "warehouse_id": "WH-CENT", "quantity_on_hand": "10"}},
	)
	seedMappedSession(repo, "sup-1", "proj-1", "s-nasiya", PackNasiyaAR, "sha-nas",
		[]string{"external_invoice_id", "customer_inn", "customer_name", "balance_due", "currency"},
		[]map[string]any{{"external_invoice_id": "INV-99", "customer_inn": "301234567", "customer_name": "Baraka Market", "balance_due": "250000.50", "currency": "UZS"}},
	)
	seedMappedSession(repo, "sup-1", "proj-1", "s-open", PackOpenOrders, "sha-open",
		[]string{"external_order_no", "delivery_date", "sku", "warehouse_id", "quantity"},
		[]map[string]any{
			{"external_order_no": "ORD-OK", "delivery_date": "2026-11-20", "sku": "TEA-100", "warehouse_id": "WH-CENT", "quantity": "8"},
			{"external_order_no": "ORD-LOW", "delivery_date": "2026-11-21", "sku": "TEA-100", "warehouse_id": "WH-CENT", "quantity": "5"}, // only 2 left -> warning!
		},
	)

	seq := 0
	rep, summary, err := EvaluateProjectDryRun(context.Background(), repo, DryRunOptions{
		SupplierID: "sup-1",
		ProjectID:  "proj-1",
		Now:        now,
		NewID: func() string {
			seq++
			return fmt.Sprintf("rpt-%d", seq)
		},
	})
	if err != nil {
		t.Fatalf("EvaluateProjectDryRun failed: %v", err)
	}
	if !summary.ReadyToApprove || rep.BlockingCount != 0 {
		t.Fatalf("expected ReadyToApprove=true and 0 blocking conflicts, got ready=%v blocking=%d conflicts=%+v",
			summary.ReadyToApprove, rep.BlockingCount, summary.Conflicts)
	}
	if !summary.BalancedJournalCheck || summary.JournalDebitMinor != 25000050 || summary.JournalCreditMinor != 25000050 {
		t.Fatalf("expected balanced journal 25000050/25000050, got balanced=%v dr=%d cr=%d",
			summary.BalancedJournalCheck, summary.JournalDebitMinor, summary.JournalCreditMinor)
	}
	if len(repo.warehousesTable) != 0 || len(repo.productsTable) != 0 || len(repo.arInvoicesTable) != 0 {
		t.Fatalf("dry-run must NEVER mutate business tables")
	}
	// Second open order should have triggered insufficient_stock_for_held_open_order warning (10 stocked - 8 allocated = 2 < 5).
	foundStockWarn := false
	for _, c := range summary.Conflicts {
		if c.Code == "insufficient_stock_for_held_open_order" && c.Severity == ConflictSeverityWarning {
			foundStockWarn = true
		}
	}
	if !foundStockWarn {
		t.Fatalf("expected insufficient_stock_for_held_open_order warning on competing virtual open orders, got %+v", summary.Conflicts)
	}

	// Approval succeeds immediately using generated report, fails if SHA256 changes (stale) or 24h elapses.
	nasiyaSess := repo.sessions["s-nasiya"]
	if nasiyaSess.DryRunReportID == "" || nasiyaSess.Status != SessionStatusDryRunReady {
		t.Fatalf("expected s-nasiya DRY_RUN_READY with DryRunReportID, got %+v", nasiyaSess)
	}

	// 24h TTL check
	_, err = ApproveSession(context.Background(), repo, "sup-1", "s-nasiya", ApproveSessionRequest{}, now.Add(24*time.Hour+time.Minute))
	if !errors.Is(err, ErrDryRunExpired) {
		t.Fatalf("expected ErrDryRunExpired after 24h, got %v", err)
	}

	// Mutating underlying file SHA256 invalidates dry-run fingerprint
	nasiyaSess.ContentSHA256 = "sha-tampered"
	repo.sessions["s-nasiya"] = nasiyaSess
	_, err = ApproveSession(context.Background(), repo, "sup-1", "s-nasiya", ApproveSessionRequest{}, now.Add(time.Hour))
	if !errors.Is(err, ErrDryRunStale) {
		t.Fatalf("expected ErrDryRunStale after SHA256 mutation, got %v", err)
	}

	// Restore valid SHA256 -> Approve succeeds
	nasiyaSess.ContentSHA256 = "sha-nas"
	repo.sessions["s-nasiya"] = nasiyaSess
	approved, err := ApproveSession(context.Background(), repo, "sup-1", "s-nasiya", ApproveSessionRequest{}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("ApproveSession expected success, got %v", err)
	}
	if approved.Status != SessionStatusApproved {
		t.Fatalf("expected APPROVED, got %s", approved.Status)
	}
}

func TestDryRun_GuardsAndBlockingRules(t *testing.T) {
	repo := newM4TestRepo()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	_, _ = repo.CreateProject(context.Background(), Project{
		SupplierID: "sup-1", ProjectID: "proj-1", Name: "Guards", Mode: ProjectModeCutover, Status: ProjectStatusDraft,
	})
	repo.live.WarehousesByExt["wh-1"] = LiveWarehouse{WarehouseID: "wh-1", ExternalID: "WH-1", Name: "Main"}
	repo.live.WarehousesByName["main"] = repo.live.WarehousesByExt["wh-1"]
	prod := LiveProduct{ProductID: "p-1", SKU: "MILK-1L", Name: "Milk 1L", Unit: "liter", PriceMinor: 1200000, Currency: "UZS", RequiresColdChain: true}
	registerVirtualProduct(&repo.live, prod)
	repo.live.StockByWHProduct["wh-1|p-1"] = LiveStock{WarehouseID: "wh-1", ProductID: "p-1", OnHand: 20, Reserved: 15, Unit: "liter"}
	repo.live.StockTotalByProd["p-1"] = 20

	// 1. Unit change on non-zero stock + SET_ON_HAND below Reserved -> blocking
	seedMappedSession(repo, "sup-1", "proj-1", "s-stk", PackStock, "sha-s",
		[]string{"sku", "warehouse_id", "quantity_on_hand", "unit"},
		[]map[string]any{{"sku": "MILK-1L", "warehouse_id": "WH-1", "quantity_on_hand": "10", "unit": "box"}},
	)
	rep, sum, err := EvaluateSessionDryRun(context.Background(), repo, DryRunOptions{
		SupplierID: "sup-1", ProjectID: "proj-1", SessionID: "s-stk", ApplyMode: ApplyModeSetOnHand, Now: now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.BlockingCount == 0 || sum.ReadyToApprove {
		t.Fatalf("expected blocking unit_change_with_non_zero_stock")
	}

	// Fix unit -> still blocked by on_hand (10) < reserved (15) under SET_ON_HAND
	seedMappedSession(repo, "sup-1", "proj-1", "s-stk", PackStock, "sha-s2",
		[]string{"sku", "warehouse_id", "quantity_on_hand", "unit"},
		[]map[string]any{{"sku": "MILK-1L", "warehouse_id": "WH-1", "quantity_on_hand": "10", "unit": "liter"}},
	)
	rep, sum, err = EvaluateSessionDryRun(context.Background(), repo, DryRunOptions{
		SupplierID: "sup-1", ProjectID: "proj-1", SessionID: "s-stk", ApplyMode: ApplyModeSetOnHand, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sum.OnHandBelowReservedHits != 1 || rep.BlockingCount == 0 {
		t.Fatalf("expected on_hand_below_reserved blocking conflict, got %+v", sum)
	}

	// SET_ON_HAND confirmation gate even when valid (qty=30 >= 15)
	seedMappedSession(repo, "sup-1", "proj-1", "s-stk", PackStock, "sha-s3",
		[]string{"sku", "warehouse_id", "quantity_on_hand", "unit"},
		[]map[string]any{{"sku": "MILK-1L", "warehouse_id": "WH-1", "quantity_on_hand": "30", "unit": "liter"}},
	)
	_, _, err = EvaluateSessionDryRun(context.Background(), repo, DryRunOptions{
		SupplierID: "sup-1", ProjectID: "proj-1", SessionID: "s-stk", ApplyMode: ApplyModeSetOnHand, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApproveSession(context.Background(), repo, "sup-1", "s-stk", ApproveSessionRequest{ApplyMode: ApplyModeSetOnHand, SetOnHandConfirmed: false}, now.Add(time.Minute))
	if !errors.Is(err, ErrSetOnHandConfirmationRequired) {
		t.Fatalf("expected ErrSetOnHandConfirmationRequired, got %v", err)
	}

	// 2. Cold-chain lots warn cold_lot_migrated_without_live_baseline & forbid SET_ON_HAND
	repo2 := newM4TestRepo()
	_, _ = repo2.CreateProject(context.Background(), Project{SupplierID: "sup-1", ProjectID: "proj-2", Name: "Lots", Mode: ProjectModeCutover, Status: ProjectStatusDraft})
	repo2.live = repo.live.clone()
	seedMappedSession(repo2, "sup-1", "proj-2", "s-lot", PackLots, "sha-l",
		[]string{"sku", "warehouse_id", "lot_number", "expiry_date", "quantity_on_hand"},
		[]map[string]any{{"sku": "MILK-1L", "warehouse_id": "WH-1", "lot_number": "LOT-A1", "expiry_date": "2026-12-30", "quantity_on_hand": "25"}},
	)
	_, lotSumBad, err := EvaluateSessionDryRun(context.Background(), repo2, DryRunOptions{
		SupplierID: "sup-1", ProjectID: "proj-2", SessionID: "s-lot", ApplyMode: ApplyModeSetOnHand, Now: now,
	})
	if err != nil || lotSumBad.ReadyToApprove {
		t.Fatalf("expected lot_set_on_hand_forbidden blocking conflict, err=%v sum=%+v", err, lotSumBad)
	}
	_, lotSumOK, err := EvaluateSessionDryRun(context.Background(), repo2, DryRunOptions{
		SupplierID: "sup-1", ProjectID: "proj-2", SessionID: "s-lot", ApplyMode: ApplyModeMerge, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !lotSumOK.ReadyToApprove || lotSumOK.ColdLotWarningCount != 1 {
		t.Fatalf("expected ReadyToApprove=true & ColdLotWarningCount=1, got %+v", lotSumOK)
	}
}

func TestRollbackSession_72hWindowAndCompensatingLedgerReversal(t *testing.T) {
	repo := newM4TestRepo()
	appliedAt := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	until := appliedAt.Add(RollbackWindowTTL)

	// Populate simulated pre-rollback applied state
	repo.sessions["s-mix"] = Session{
		SupplierID:    "sup-1",
		SessionID:     "s-mix",
		ProjectID:     "proj-1",
		Pack:          PackNasiyaAR,
		Status:        SessionStatusApplied,
		RollbackUntil: &until,
	}
	repo.inventoryTable["wh-1|p-1"] = map[string]any{"SupplierId": "sup-1", "WarehouseId": "wh-1", "ProductId": "p-1", "QuantityOnHand": int64(50), "QuantityReserved": int64(5)}
	repo.ordersTable["ord-mig-1"] = map[string]any{"OrderId": "ord-mig-1", "Status": "MIGRATION_OPEN", "SupersededByRollback": false}
	repo.arInvoicesTable["inv-mig-1"] = map[string]any{"InvoiceId": "inv-mig-1", "SupplierId": "sup-1", "RetailerId": "ret-1", "OrderId": "ord-mig-1", "BalanceMinor": int64(750000), "Currency": "UZS", "Status": "OPEN"}

	s1, _ := BuildSnapshotRecord("sup-1", "s-mix", 1, "SupplierInventoryV2", map[string]any{"SupplierId": "sup-1", "WarehouseId": "wh-1", "ProductId": "p-1"}, SnapshotOpUpdate,
		map[string]any{"SupplierId": "sup-1", "WarehouseId": "wh-1", "ProductId": "p-1", "QuantityOnHand": int64(20), "QuantityReserved": int64(5)}, appliedAt)
	s2, _ := BuildSnapshotRecord("sup-1", "s-mix", 2, "Orders", map[string]any{"OrderId": "ord-mig-1"}, SnapshotOpInsert, nil, appliedAt)
	s3, _ := BuildSnapshotRecord("sup-1", "s-mix", 3, "ArInvoices", map[string]any{"InvoiceId": "inv-mig-1"}, SnapshotOpInsert,
		RollbackInvoiceBeforeImage{InvoiceID: "inv-mig-1", SupplierID: "sup-1", RetailerID: "ret-1", OrderID: "ord-mig-1", AmountMinor: 750000, Currency: "UZS"}, appliedAt)
	_ = repo.SaveSnapshotsChunk(context.Background(), "sup-1", "s-mix", []MigrationSnapshotRecord{s1, s2, s3})

	// 1. Expired > 72h must be rejected fail-closed
	_, err := RollbackSession(context.Background(), repo, "sup-1", "s-mix", appliedAt.Add(72*time.Hour+time.Second))
	if !errors.Is(err, ErrRollbackWindowExpired) {
		t.Fatalf("expected ErrRollbackWindowExpired after 72h, got %v", err)
	}

	// 2. Within 72h -> restores inventory byte-exact, supersedes Orders, reverses ArInvoices & creates balanced journal
	res, err := RollbackSession(context.Background(), repo, "sup-1", "s-mix", appliedAt.Add(6*time.Hour))
	if err != nil {
		t.Fatalf("RollbackSession failed: %v", err)
	}
	if res.Session.Status != SessionStatusRolledBack || res.RevertedSnapshots != 3 || res.SupersededOrders != 1 || res.ReversedInvoices != 1 || res.ReversedAmountMinor != 750000 {
		t.Fatalf("unexpected rollback result: %+v", res)
	}
	if got := repo.inventoryTable["wh-1|p-1"]["QuantityOnHand"]; got != int64(20) {
		t.Fatalf("expected inventory QuantityOnHand restored to 20, got %T(%v)", got, got)
	}
	if repo.ordersTable["ord-mig-1"]["Status"] != "CANCELLED" || repo.ordersTable["ord-mig-1"]["SupersededByRollback"] != true {
		t.Fatalf("expected order preserved + marked CANCELLED + SupersededByRollback=true, got %+v", repo.ordersTable["ord-mig-1"])
	}
	if repo.arInvoicesTable["inv-mig-1"]["Status"] != "ROLLED_BACK" || repo.arInvoicesTable["inv-mig-1"]["BalanceMinor"] != int64(0) {
		t.Fatalf("expected ArInvoice ROLLED_BACK with 0 balance, got %+v", repo.arInvoicesTable["inv-mig-1"])
	}
	if len(repo.journalEntries) != 1 || repo.journalEntries[0].EntryID != "jentry_migration_reversal_inv-mig-1" {
		t.Fatalf("expected balanced reversal journal entry, got %+v", repo.journalEntries)
	}
	if err := repo.journalEntries[0].Validate(); err != nil {
		t.Fatalf("reversal journal entry failed Validate(): %v", err)
	}
}

func TestEvaluateSessionDryRun_AllowsMappingProposedUpstreamVirtualOverlay(t *testing.T) {
	repo := newM4TestRepo()
	now := time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC)
	_, _ = repo.CreateProject(context.Background(), Project{
		SupplierID: "sup-1",
		ProjectID:  "proj-proposed",
		Name:       "Proposed Upstream Overlay",
		Mode:       ProjectModeCutover,
		Status:     ProjectStatusDraft,
	})

	// Upstream customers & historical_orders finished staging with AI-proposed (unconfirmed) mappings.
	seedMappedSession(repo, "sup-1", "proj-proposed", "s-cust", PackCustomers, "sha-c",
		[]string{"customer_name", "inn", "phone"},
		[]map[string]any{{"customer_name": "Baraka Market", "inn": "301234567", "phone": "+998901112233"}},
	)
	custSess := repo.sessions["s-cust"]
	custSess.Status = SessionStatusMappingProposed
	repo.sessions["s-cust"] = custSess
	custDoc := repo.sessionMaps["s-cust"]
	for i := range custDoc.Mappings {
		custDoc.Mappings[i].Confirmed = false
	}
	repo.sessionMaps["s-cust"] = custDoc

	seedMappedSession(repo, "sup-1", "proj-proposed", "s-ord", PackHistoricalOrders, "sha-o",
		[]string{"external_order_no", "order_date", "total_amount", "customer_inn"},
		[]map[string]any{{"external_order_no": "ORD-10", "order_date": "2026-10-01", "total_amount": "250000", "customer_inn": "301234567"}},
	)
	ordSess := repo.sessions["s-ord"]
	ordSess.Status = SessionStatusMappingProposed
	repo.sessions["s-ord"] = ordSess
	ordDoc := repo.sessionMaps["s-ord"]
	for i := range ordDoc.Mappings {
		ordDoc.Mappings[i].Confirmed = false
	}
	repo.sessionMaps["s-ord"] = ordDoc

	// Target nasiya_ar session is mapped/confirmed (DISCOVERED).
	seedMappedSession(repo, "sup-1", "proj-proposed", "s-ar", PackNasiyaAR, "sha-ar",
		[]string{"external_invoice_id", "customer_inn", "customer_name", "balance_due", "currency"},
		[]map[string]any{{"external_invoice_id": "INV-10", "customer_inn": "301234567", "customer_name": "Baraka Market", "balance_due": "125000", "currency": "UZS"}},
	)

	rep, sum, err := EvaluateSessionDryRun(context.Background(), repo, DryRunOptions{
		SupplierID: "sup-1",
		ProjectID:  "proj-proposed",
		SessionID:  "s-ar",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("EvaluateSessionDryRun failed when upstream is MAPPING_PROPOSED: %v", err)
	}
	if rep.BlockingCount != 0 || !sum.ReadyToApprove || !sum.BalancedJournalCheck {
		t.Fatalf("expected clean ready-to-approve dry-run, got blocking=%d sum=%+v", rep.BlockingCount, sum)
	}
	if repo.sessions["s-ar"].Status != SessionStatusDryRunReady {
		t.Fatalf("expected s-ar transitioned to DRY_RUN_READY, got %s", repo.sessions["s-ar"].Status)
	}
	if repo.sessions["s-cust"].Status != SessionStatusMappingProposed {
		t.Fatalf("expected upstream s-cust left in MAPPING_PROPOSED, got %s", repo.sessions["s-cust"].Status)
	}
}
