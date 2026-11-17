package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/innlookup"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

// goldenRepo bridges m5TestRepo with bundle sort persistence and chunk write tracking
// so end-to-end golden fixture tests exercise ingestion -> dry-run -> apply -> cutover -> rollback.
type goldenRepo struct {
	*m5TestRepo
	chunkWrites int
	maxChunkLen int
}

func newGoldenRepo(t *testing.T, supplierID, projectID string, now time.Time) (*goldenRepo, *MemoryApplyStore) {
	t.Helper()
	m5, store, _ := newM5TestEnv(t, supplierID, now)
	if projectID != "proj_m5" {
		_, err := m5.CreateProject(context.Background(), Project{
			SupplierID: supplierID,
			ProjectID:  projectID,
			Name:       "Golden Fixture Cutover",
			Mode:       ProjectModeCutover,
			Status:     ProjectStatusDraft,
			CreatedBy:  "usr-golden",
		})
		if err != nil {
			t.Fatalf("create project %s: %v", projectID, err)
		}
	}
	return &goldenRepo{m5TestRepo: m5}, store
}

func (g *goldenRepo) UpsertSortedBundleSessions(
	_ context.Context,
	supplierID, projectID, bundleID string,
	bundleStatus BundleStatus,
	summaryJSON json.RawMessage,
	_ map[string]json.RawMessage,
	sessions []Session,
) ([]Session, error) {
	g.m4TestRepo.mu.Lock()
	defer g.m4TestRepo.mu.Unlock()
	if b, ok := g.m4TestRepo.bundles[bundleID]; ok {
		b.Status = bundleStatus
		b.SortSummary = summaryJSON
		g.m4TestRepo.bundles[bundleID] = b
	}
	out := make([]Session, len(sessions))
	for i, s := range sessions {
		s.SupplierID = supplierID
		s.ProjectID = projectID
		s.BundleID = bundleID
		if s.Status == "" {
			s.Status = SessionStatusMappingRequired
		}
		g.m4TestRepo.sessions[s.SessionID] = s
		out[i] = s
	}
	return out, nil
}

func (g *goldenRepo) ListSessionsByBundle(_ context.Context, supplierID, projectID, bundleID string) ([]Session, error) {
	g.m4TestRepo.mu.Lock()
	defer g.m4TestRepo.mu.Unlock()
	var out []Session
	for _, s := range g.m4TestRepo.sessions {
		if s.SupplierID == supplierID && s.ProjectID == projectID && s.BundleID == bundleID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (g *goldenRepo) SaveStagedRowsChunk(_ context.Context, _, sessionID string, rows []StagedRow) (int64, error) {
	g.m4TestRepo.mu.Lock()
	defer g.m4TestRepo.mu.Unlock()
	g.chunkWrites++
	if len(rows) > g.maxChunkLen {
		g.maxChunkLen = len(rows)
	}
	g.m4TestRepo.stagedRows[sessionID] = append(g.m4TestRepo.stagedRows[sessionID], rows...)
	total := int64(len(g.m4TestRepo.stagedRows[sessionID]))
	if s, ok := g.m4TestRepo.sessions[sessionID]; ok {
		s.TotalRows = total
		g.m4TestRepo.sessions[sessionID] = s
	}
	return total, nil
}

func packOrderRank(target Pack) int {
	for idx, p := range TopologicalPackOrder {
		if p == target {
			return idx
		}
	}
	return len(TopologicalPackOrder)
}

func loadGoldenFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "..", "testdata", "migrations", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", p, err)
	}
	return b
}

func registerBundleArtifact(t *testing.T, files map[string][]byte, sup, projID, bundleID, artID, fileName string, raw []byte) Artifact {
	t.Helper()
	gcsPath := "fixtures/" + artID + "/" + fileName
	files[gcsPath] = raw
	sum := sha256.Sum256(raw)
	return Artifact{
		SupplierID:    sup,
		ProjectID:     projID,
		BundleID:      bundleID,
		ArtifactID:    artID,
		FileName:      fileName,
		ByteSize:      int64(len(raw)),
		ContentSHA256: hex.EncodeToString(sum[:]),
		GCSPath:       gcsPath,
	}
}

func autoMapAndConfirmSessions(t *testing.T, repo *goldenRepo, sup, cell string, sessions []Session) {
	t.Helper()
	mapper := NewAIMapper(GeminiMapperConfig{Enabled: false})
	for _, s := range sessions {
		staged, _, err := repo.ListStagedRows(context.Background(), sup, s.SessionID, 20, 0)
		if err != nil {
			t.Fatalf("list staged %s: %v", s.SessionID, err)
		}
		headers, samples := extractHeadersAndSampleRows(staged)
		doc := mapper.ProposeMappings(context.Background(), sup, cell, s.SessionID, s.Pack, headers, samples, nil)
		for i := range doc.Mappings {
			if doc.Mappings[i].TargetField != "" {
				doc.Mappings[i].Confirmed = true
			}
		}
		doc.AllRequiredMapped = true
		doc.MappingHash = ComputeMappingHash(doc)
		if err := repo.SaveSessionMapping(context.Background(), sup, s.SessionID, SessionStatusDiscovered, doc); err != nil {
			t.Fatalf("save mapping %s (%s): %v", s.SessionID, s.Pack, err)
		}
	}
}

func approveAndApplyProjectSessions(t *testing.T, repo *goldenRepo, store *MemoryApplyStore, sup, projID, applyMode string, setOnHandConfirmed bool, now time.Time) map[Pack]ApplyResult {
	t.Helper()
	ctx := context.Background()
	rep, summary, err := EvaluateProjectDryRun(ctx, repo, DryRunOptions{
		SupplierID: sup,
		ProjectID:  projID,
		ApplyMode:  applyMode,
		Now:        now,
		LiveReader: repo,
	})
	if err != nil {
		t.Fatalf("EvaluateProjectDryRun: %v", err)
	}
	if !summary.ReadyToApprove || rep.BlockingCount > 0 {
		t.Fatalf("unexpected dry-run blocking issues: blocking=%d summary=%s", rep.BlockingCount, string(rep.SummaryJSON))
	}

	sessions, err := repo.ListSessionsByProject(ctx, sup, projID)
	if err != nil {
		t.Fatalf("ListSessionsByProject: %v", err)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return packOrderRank(sessions[i].Pack) < packOrderRank(sessions[j].Pack)
	})

	results := make(map[Pack]ApplyResult)
	for _, s := range sessions {
		sRep, _, sErr := EvaluateSessionDryRun(ctx, repo, DryRunOptions{
			SupplierID: sup,
			ProjectID:  projID,
			SessionID:  s.SessionID,
			ApplyMode:  applyMode,
			Now:        now,
			LiveReader: repo,
		})
		if sErr != nil {
			t.Fatalf("EvaluateSessionDryRun %s (%s): %v", s.SessionID, s.Pack, sErr)
		}
		if _, err := ApproveSession(ctx, repo, sup, s.SessionID, ApproveSessionRequest{
			ApplyMode:          applyMode,
			SetOnHandConfirmed: setOnHandConfirmed,
			ExpectedReportID:   sRep.ReportID,
		}, now); err != nil {
			t.Fatalf("ApproveSession %s (%s): %v", s.SessionID, s.Pack, err)
		}
		res, err := ApplySession(ctx, repo, store, ApplyOptions{
			SupplierID: sup,
			SessionID:  s.SessionID,
			ApplyMode:  applyMode,
			Now:        now,
		})
		if err != nil {
			t.Fatalf("ApplySession %s (%s): %v", s.SessionID, s.Pack, err)
		}
		results[s.Pack] = res
	}
	return results
}

func TestGoldenSuite1_AllInOne25kXLSX_EndToEndAndCutover(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	sup, projID, cell := "sup-golden-1", "proj_golden_1", "8928308280fffff"
	repo, store := newGoldenRepo(t, sup, projID, now)

	rawXLSX := loadGoldenFixture(t, "suite1_all_in_one_25k.xlsx")
	files := make(map[string][]byte)
	art := registerBundleArtifact(t, files, sup, projID, "bnd-s1", "art-xlsx", "suite1_all_in_one_25k.xlsx", rawXLSX)
	_, _, _ = repo.CreateBundleWithArtifacts(ctx, Bundle{
		SupplierID: sup, ProjectID: projID, BundleID: "bnd-s1", Status: BundleStatusClassifying,
	}, []Artifact{art})

	reader := func(_ context.Context, path string) ([]byte, error) {
		b, ok := files[path]
		if !ok {
			return nil, ErrNotFound
		}
		return b, nil
	}

	dec, err := ProcessBundleSortAndStage(ctx, repo, reader, sup, projID, "bnd-s1", nil, seqIDGen("s1"))
	if err != nil {
		t.Fatalf("ProcessBundleSortAndStage: %v", err)
	}
	if dec.BundleStatus != BundleStatusSorted {
		t.Fatalf("expected SORTED, got %s (%v)", dec.BundleStatus, dec.Summary.ReviewReasons)
	}
	if len(dec.Sessions) != 6 {
		t.Fatalf("expected 6 DAG sessions from 6 sheets, got %d", len(dec.Sessions))
	}
	var totalStaged int64
	for _, s := range dec.Sessions {
		totalStaged += s.TotalRows
	}
	if totalStaged != 25505 {
		t.Fatalf("expected 25,505 staged rows, got %d", totalStaged)
	}
	if repo.maxChunkLen > 5000 {
		t.Fatalf("expected chunk cap <= 5000, got maxChunkLen=%d", repo.maxChunkLen)
	}

	autoMapAndConfirmSessions(t, repo, sup, cell, dec.Sessions)
	results := approveAndApplyProjectSessions(t, repo, store, sup, projID, ApplyModeMerge, false, now)

	if results[PackWarehouses].Created != 5 {
		t.Fatalf("expected 5 warehouses created, got %+v", results[PackWarehouses])
	}
	if results[PackProductsPrices].Created != 18000 {
		t.Fatalf("expected 18,000 products created, got %+v", results[PackProductsPrices])
	}
	if results[PackCustomers].Created != 1500 {
		t.Fatalf("expected 1,500 customers created, got %+v", results[PackCustomers])
	}
	if results[PackStock].Created != 4000 {
		t.Fatalf("expected 4,000 stock entries created, got %+v", results[PackStock])
	}
	if results[PackNasiyaAR].Created != 1000 || len(store.JournalEntries) != 1000 {
		t.Fatalf("expected 1,000 balanced nasiya AR journals, got apply=%+v journals=%d", results[PackNasiyaAR], len(store.JournalEntries))
	}
	for id, je := range store.JournalEntries {
		if err := je.Validate(); err != nil {
			t.Fatalf("journal %s imbalanced: %v", id, err)
		}
	}
	if results[PackHistoricalOrders].Created != 1000 {
		t.Fatalf("expected 1,000 historical orders created, got %+v", results[PackHistoricalOrders])
	}

	cutRes, err := ExecuteProjectCutover(ctx, repo, CutoverOptions{
		SupplierID: sup, ProjectID: projID, ActorID: "usr-golden", Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ExecuteProjectCutover: %v (checklist=%+v)", err, cutRes.Checklist)
	}
	if cutRes.Project.Status != ProjectStatusCutoverComplete {
		t.Fatalf("expected CUTOVER_COMPLETE, got %s", cutRes.Project.Status)
	}
}

func TestGoldenSuite2_BIFF8XLSAndCP1251BillzCSV_PriceAnomaly(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	sup, projID, cell := "sup-golden-2", "proj_golden_2", "8928308280fffff"
	repo, store := newGoldenRepo(t, sup, projID, now)

	xlsRaw := loadGoldenFixture(t, "suite2_biff8_legacy.xls")
	csvRaw := loadGoldenFixture(t, "suite2_messy_cp1251_billz.csv")
	files := make(map[string][]byte)
	aXLS := registerBundleArtifact(t, files, sup, projID, "bnd-s2", "art-xls", "suite2_biff8_legacy.xls", xlsRaw)
	aCSV := registerBundleArtifact(t, files, sup, projID, "bnd-s2", "art-csv", "suite2_messy_cp1251_billz.csv", csvRaw)
	_, _, _ = repo.CreateBundleWithArtifacts(ctx, Bundle{
		SupplierID: sup, ProjectID: projID, BundleID: "bnd-s2", Status: BundleStatusClassifying,
	}, []Artifact{aXLS, aCSV})

	// Explicit cp1251 override on legacy exports ensures Cyrillic headers decode deterministically.
	overrides := &BundleSortOverrideRequest{
		Overrides: []StreamOverrideItem{
			{ArtifactID: "art-csv", Encoding: "windows-1251"},
		},
	}
	dec, err := ProcessBundleSortAndStage(ctx, repo, func(_ context.Context, p string) ([]byte, error) {
		return files[p], nil
	}, sup, projID, "bnd-s2", overrides, seqIDGen("s2"))
	if err != nil {
		t.Fatalf("ProcessBundleSortAndStage suite2: %v", err)
	}

	var seenBIFF8Stock, seenCSVCatalog bool
	for _, st := range dec.Summary.Streams {
		for _, p := range st.DetectedPacks {
			if st.FileName == "suite2_biff8_legacy.xls" && p == PackStock {
				seenBIFF8Stock = true
			}
			if st.FileName == "suite2_messy_cp1251_billz.csv" && p == PackProductsPrices {
				seenCSVCatalog = true
			}
		}
	}
	if !seenBIFF8Stock || !seenCSVCatalog {
		t.Fatalf("expected BIFF8 stock + CP1251 CSV catalog classifications, got streams=%+v", dec.Summary.Streams)
	}

	store.Warehouses["wh-default-s2"] = StoredWarehouse{WarehouseID: "wh-default-s2", SupplierID: sup, Name: "Default", IsActive: true, IsOnShift: true}

	autoMapAndConfirmSessions(t, repo, sup, cell, dec.Sessions)
	_, summary, err := EvaluateProjectDryRun(ctx, repo, DryRunOptions{
		SupplierID: sup, ProjectID: projID, ApplyMode: ApplyModeMerge, Now: now, LiveReader: repo,
	})
	if err != nil {
		t.Fatalf("dry-run suite2: %v", err)
	}

	var hasPriceAnomaly bool
	for _, c := range summary.Conflicts {
		if c.Code == "anomalous_price_change" {
			hasPriceAnomaly = true
			break
		}
	}
	if !hasPriceAnomaly {
		t.Fatalf("expected anomalous_price_change warning on 10x CSV price jump, got conflicts=%+v", summary.Conflicts)
	}
}

func TestGoldenSuite3_CommerceML1C_XMLSplit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	sup, projID := "sup-golden-3", "proj_golden_3"
	repo, _ := newGoldenRepo(t, sup, projID, now)

	xmlRaw := loadGoldenFixture(t, "suite3_commerceml_1c.xml")
	files := make(map[string][]byte)
	art := registerBundleArtifact(t, files, sup, projID, "bnd-s3", "art-xml", "suite3_commerceml_1c.xml", xmlRaw)
	_, _, _ = repo.CreateBundleWithArtifacts(ctx, Bundle{
		SupplierID: sup, ProjectID: projID, BundleID: "bnd-s3", Status: BundleStatusClassifying,
	}, []Artifact{art})

	dec, err := ProcessBundleSortAndStage(ctx, repo, func(_ context.Context, p string) ([]byte, error) {
		return files[p], nil
	}, sup, projID, "bnd-s3", nil, seqIDGen("s3"))
	if err != nil {
		t.Fatalf("ProcessBundleSortAndStage CommerceML: %v", err)
	}
	if dec.BundleStatus != BundleStatusSorted {
		t.Fatalf("expected SORTED for CommerceML bundle, got %s (%v)", dec.BundleStatus, dec.Summary.ReviewReasons)
	}

	splits := make(map[string]int64)
	for _, s := range dec.Sessions {
		splits[string(s.Pack)+":"+s.StreamSplitKey] = s.TotalRows
	}
	for _, expectedKey := range []string{
		"products_prices:products_prices",
		"products_prices:offers_prices",
		"stock:offers_stock",
		"customers:customers",
		"historical_orders:orders_historical",
		"open_orders:orders_open",
	} {
		if splits[expectedKey] == 0 {
			t.Fatalf("missing CommerceML virtual split %s; got %+v", expectedKey, splits)
		}
	}
}

func TestGoldenSuite4_CompositeJSON_ColdChainLots_FEFOAndCancelRelease(t *testing.T) {
	prevLots := stocklots.LotsEnabled()
	stocklots.SetLotsEnabled(true)
	t.Cleanup(func() { stocklots.SetLotsEnabled(prevLots) })

	ctx := context.Background()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	sup, projID, cell := "sup-golden-4", "proj_golden_4", "8928308280fffff"
	repo, store := newGoldenRepo(t, sup, projID, now)

	jsonRaw := loadGoldenFixture(t, "suite4_composite_coldchain.json")
	files := make(map[string][]byte)
	art := registerBundleArtifact(t, files, sup, projID, "bnd-s4", "art-json", "suite4_composite_coldchain.json", jsonRaw)
	_, _, _ = repo.CreateBundleWithArtifacts(ctx, Bundle{
		SupplierID: sup, ProjectID: projID, BundleID: "bnd-s4", Status: BundleStatusClassifying,
	}, []Artifact{art})

	dec, err := ProcessBundleSortAndStage(ctx, repo, func(_ context.Context, p string) ([]byte, error) {
		return files[p], nil
	}, sup, projID, "bnd-s4", nil, seqIDGen("s4"))
	if err != nil {
		t.Fatalf("ProcessBundleSortAndStage suite4: %v", err)
	}
	if len(dec.Summary.Streams) != 1 || !dec.Summary.Streams[0].IsComposite {
		t.Fatalf("expected 1 composite JSON stream, got %+v", dec.Summary.Streams)
	}
	if len(dec.Sessions) < 4 {
		t.Fatalf("expected composite JSON to split into >=4 DAG sessions, got %d", len(dec.Sessions))
	}

	autoMapAndConfirmSessions(t, repo, sup, cell, dec.Sessions)
	_, summary, err := EvaluateProjectDryRun(ctx, repo, DryRunOptions{
		SupplierID: sup, ProjectID: projID, ApplyMode: ApplyModeMerge, Now: now, LiveReader: repo,
	})
	if err != nil {
		t.Fatalf("dry-run suite4: %v", err)
	}
	if summary.ColdLotWarningCount != 2 {
		t.Fatalf("expected 2 cold_lot_migrated_without_live_baseline warnings, got %d", summary.ColdLotWarningCount)
	}

	results := approveAndApplyProjectSessions(t, repo, store, sup, projID, ApplyModeMerge, false, now)
	if results[PackLots].Created != 2 || results[PackOpenOrders].Created != 2 {
		t.Fatalf("expected 2 lots + 2 held open orders created, got lots=%+v open=%+v", results[PackLots], results[PackOpenOrders])
	}

	whID := store.IdentityMap[identityKey(sup, PackWarehouses, "WH-COLD-01")].InternalID
	prodID := store.IdentityMap[identityKey(sup, PackProductsPrices, "SKU-VAX-UZ")].InternalID
	earlyLotID := deterministicEntityID("lot", sup, string(PackLots), whID+"|"+prodID+"|LOT-FEFO-EARLY")
	lateLotID := deterministicEntityID("lot", sup, string(PackLots), whID+"|"+prodID+"|LOT-FEFO-LATE")
	if stk := store.Inventory[inventoryKey(sup, whID, prodID)]; stk.QuantityOnHand != 45 || stk.QuantityReserved != 0 {
		t.Fatalf("expected pre-release rollup OnHand=45 Reserved=0, got %+v", stk)
	}

	// Release both held MIGRATION_OPEN orders selectively -> FEFO consumes earliest expiry first.
	ord1ID := store.IdentityMap[identityKey(sup, PackOpenOrders, "OPEN-COLD-101")].InternalID
	ord2ID := store.IdentityMap[identityKey(sup, PackOpenOrders, "OPEN-COLD-102")].InternalID
	relRes, err := store.ReleaseOpenOrders(ctx, sup, []order.ReleaseOpenOrderItem{
		{OrderID: ord1ID, TargetStatus: order.StatusPending},
		{OrderID: ord2ID, TargetStatus: order.StatusScheduled},
	}, "usr-golden", now.Add(time.Minute))
	if err != nil || len(relRes.ReleasedOrderIDs) != 2 {
		t.Fatalf("ReleaseOpenOrders: err=%v res=%+v", err, relRes)
	}

	if store.Lots[earlyLotID].QuantityReserved != 15 {
		t.Fatalf("expected earliest-expiry lot fully reserved (15/15), got %d", store.Lots[earlyLotID].QuantityReserved)
	}
	if store.Lots[lateLotID].QuantityReserved != 7 {
		t.Fatalf("expected later-expiry lot reserved=7 (10+12-15), got %d", store.Lots[lateLotID].QuantityReserved)
	}
	if stk := store.Inventory[inventoryKey(sup, whID, prodID)]; stk.QuantityReserved != 22 {
		t.Fatalf("expected total QuantityReserved=22, got %+v", stk)
	}

	// Verify cancel-after-release reservation path:
	// Historical MIGRATION source no-ops cleanly; released open order (OrderSourceMigrationReleased/Manual) does not bypass release.
	histLines := []order.LineItem{{SKU: prodID, Quantity: 10}}
	if err := order.ReleaseReservationsForOrderInTxn(ctx, nil, sup, whID, "ord-hist", order.OrderSourceMigration, histLines); err != nil {
		t.Fatalf("OrderSourceMigration no-op failed: %v", err)
	}
	if err := order.ReleaseReservationsForOrderInTxn(ctx, nil, sup, whID, ord1ID, order.OrderSourceMigrationReleased, histLines); err != nil {
		t.Fatalf("OrderSourceMigrationReleased nil-txn check failed: %v", err)
	}
}

func TestGoldenSuite5_FinancialAuthBypassINNAnd72hRollbackInvariants(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)

	var fixture struct {
		ImbalancedNasiyaRows []map[string]any `json:"imbalanced_nasiya_rows"`
		DuplicateOrders      []map[string]any `json:"duplicate_historical_orders"`
		INNLookupCases       []struct {
			INN            string `json:"inn"`
			ExpectedStatus string `json:"expected_status"`
			LegalName      string `json:"legal_name"`
		} `json:"inn_lookup_cases"`
		RollbackWindowHours int `json:"rollback_window_hours"`
	}
	raw := loadGoldenFixture(t, "suite5_financial_and_inn_invariants.json")
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode suite5: %v", err)
	}

	// 1. Imbalanced double-entry journal rejection + negative/zero/FX-mismatch nasiya dry-run blocking.
	imbalanced := &payment.JournalEntry{
		EntryID: "je-bad-1", TransactionID: "tx-bad-1", SupplierID: "sup-5", Currency: "UZS", OccurredAt: now,
		Postings: []payment.Posting{
			{AccountCode: "AR:sup-5", AccountType: payment.AccountClassAsset, Direction: payment.PostingDebit, AmountMinor: 100000, Currency: "UZS"},
			{AccountCode: "EQUITY:OPENING_BALANCE", AccountType: payment.AccountClassEquity, Direction: payment.PostingCredit, AmountMinor: 90000, Currency: "UZS"},
		},
	}
	if err := imbalanced.Validate(); !errors.Is(err, payment.ErrUnbalancedLedgerEntry) {
		t.Fatalf("expected ErrUnbalancedLedgerEntry, got %v", err)
	}

	sup, projID := "sup-golden-5", "proj_golden_5"
	repo, store := newGoldenRepo(t, sup, projID, now)
	store.Retailers["ret-bad-test"] = StoredCustomer{
		RetailerID: "ret-bad-test", Phone: "+998909000001", Name: "Zero Debt Store", CountryCode: "UZ",
	}
	store.Links[sup+"|ret-bad-test"] = StoredCustomerLink{
		SupplierID: sup, RetailerID: "ret-bad-test", TaxIdTIN: "309000001", LegalName: "Zero Debt Store", DedupStatus: "LINKED",
	}
	badSession := Session{
		SupplierID: sup, SessionID: "sess-bad-ar", ProjectID: projID, BundleID: "bnd-s5",
		Pack: PackNasiyaAR, Status: SessionStatusDiscovered, ContentSHA256: "sha-bad-ar", TotalRows: int64(len(fixture.ImbalancedNasiyaRows)),
	}
	repo.m4TestRepo.mu.Lock()
	repo.m4TestRepo.sessions[badSession.SessionID] = badSession
	for i, r := range fixture.ImbalancedNasiyaRows {
		b, _ := json.Marshal(r)
		repo.m4TestRepo.stagedRows[badSession.SessionID] = append(repo.m4TestRepo.stagedRows[badSession.SessionID], StagedRow{
			SupplierID: sup, SessionID: badSession.SessionID, RowIndex: int64(i + 1), RawData: b,
		})
	}
	repo.m4TestRepo.sessionMaps[badSession.SessionID] = SessionMappingDocument{
		SessionID: badSession.SessionID, Pack: PackNasiyaAR, AllRequiredMapped: true,
		Mappings: []ColumnMappingProposal{
			{SourceColumn: "external_invoice_id", TargetField: "external_invoice_id", Confirmed: true},
			{SourceColumn: "customer_inn", TargetField: "customer_inn", Confirmed: true},
			{SourceColumn: "customer_name", TargetField: "customer_name", Confirmed: true},
			{SourceColumn: "balance_due", TargetField: "balance_due", Confirmed: true},
			{SourceColumn: "currency", TargetField: "currency", Confirmed: true},
		},
	}
	repo.m4TestRepo.mu.Unlock()

	rep, summary, err := EvaluateSessionDryRun(ctx, repo, DryRunOptions{
		SupplierID: sup, ProjectID: projID, SessionID: badSession.SessionID, ApplyMode: ApplyModeMerge, Now: now, LiveReader: repo,
	})
	if err != nil {
		t.Fatalf("dry-run bad nasiya: %v", err)
	}
	if summary.ReadyToApprove || rep.BlockingCount < 3 {
		t.Fatalf("expected >=3 blocking errors for invalid/FX nasiya rows, got blocking=%d ready=%v", rep.BlockingCount, summary.ReadyToApprove)
	}

	// 2. Duplicate OrderId idempotency (second row with same external_order_no skips rather than duplicating).
	dupProj := "proj_golden_dup"
	_, _ = repo.CreateProject(ctx, Project{SupplierID: sup, ProjectID: dupProj, Name: "Dup", Mode: ProjectModeCutover, Status: ProjectStatusDraft})
	stageDryRunAndApprove(t, repo.m5TestRepo, sup, dupProj, "sess-dup-orders", PackHistoricalOrders, fixture.DuplicateOrders, map[string]string{
		"external_order_no": "external_order_no",
		"order_date":        "order_date",
		"total_amount":      "total_amount",
		"status":            "status",
	}, ApplyModeMerge, false, now)
	dupRes, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-dup-orders", Now: now})
	if err != nil {
		t.Fatalf("apply duplicate order fixture: %v", err)
	}
	if dupRes.Created != 1 || dupRes.Skipped != 1 || len(store.Orders) != 1 {
		t.Fatalf("expected exactly 1 deduplicated historical order (created=1 skipped=1), got %+v orders=%d", dupRes, len(store.Orders))
	}

	// 3. ALLOW_AUTH_BYPASS fail-closed rejection.
	t.Setenv("ALLOW_AUTH_BYPASS", "true")
	adminCtx := auth.WithClaims(ctx, auth.Claims{
		Subject: "usr-admin", SupplierID: sup, Role: auth.RoleAdmin, HomeCell: "8928308280fffff",
	})
	if _, err := GuardRequestAuthorization(adminCtx, AuthzRequest{Action: "apply", Pack: string(PackNasiyaAR), TargetSupplierID: sup}); !errors.Is(err, ErrAuthBypassForbidden) {
		t.Fatalf("expected ErrAuthBypassForbidden even with admin claims, got %v", err)
	}
	t.Setenv("ALLOW_AUTH_BYPASS", "false")

	// 4. INN Found / NotFound / Unavailable / JoinRequest flows.
	stub := innlookup.NewStubAdapter()
	for _, tc := range fixture.INNLookupCases {
		switch tc.ExpectedStatus {
		case "FOUND", "FOUND_WITH_EXISTING_TENANT":
			stub.Register(tc.INN, tc.LegalName, "Tashkent")
		case "UNAVAILABLE":
			stub.RegisterResult(tc.INN, innlookup.LookupResult{Status: innlookup.StatusUnavailable})
		}
	}
	memINN := innlookup.NewMemoryStore()
	memINN.RegisterTenant(innlookup.TenantMatch{
		HomeCell: "cell-uz", TenantType: innlookup.TenantTypeRetailer, TenantID: "ret-existing-409", LegalName: "OOO Navoi Mart Test", NormalizedINN: "301900409",
	})
	innSvc := innlookup.NewService(innlookup.Config{
		Adapter:      stub,
		Cache:        memINN,
		Tenants:      memINN,
		JoinRequests: memINN,
	})

	rFound, err := innSvc.Lookup(ctx, innlookup.LookupOptions{TaxID: "301900001", HomeCell: "cell-uz", ClientKey: "k1"})
	if err != nil || rFound.Status != innlookup.StatusFound {
		t.Fatalf("INN FOUND mismatch: %+v err=%v", rFound, err)
	}
	rNotFound, err := innSvc.Lookup(ctx, innlookup.LookupOptions{TaxID: "301900404", HomeCell: "cell-uz", ClientKey: "k2"})
	if err != nil || rNotFound.Status != innlookup.StatusNotFound {
		t.Fatalf("INN NOT_FOUND mismatch: %+v err=%v", rNotFound, err)
	}
	rUnavail, err := innSvc.Lookup(ctx, innlookup.LookupOptions{TaxID: "301900503", HomeCell: "cell-uz", ClientKey: "k3"})
	if !errors.Is(err, innlookup.ErrRegistryUnavailable) || rUnavail.Status != innlookup.StatusUnavailable {
		t.Fatalf("INN UNAVAILABLE mismatch: %+v err=%v", rUnavail, err)
	}
	rDup, err := innSvc.Lookup(ctx, innlookup.LookupOptions{TaxID: "301900409", HomeCell: "cell-uz", TenantType: innlookup.TenantTypeRetailer, ClientKey: "k4"})
	if err != nil || rDup.Status != innlookup.StatusFound || rDup.ExistingTenantID != "ret-existing-409" || !rDup.ExistingJoinEligible {
		t.Fatalf("INN FOUND_WITH_EXISTING_TENANT mismatch: %+v err=%v", rDup, err)
	}
	jr, err := memINN.CreateJoinRequest(ctx, innlookup.JoinRequest{
		HomeCell:         "cell-uz",
		TargetTenantType: innlookup.TenantTypeRetailer,
		TargetTenantID:   "ret-existing-409",
		NormalizedINN:    "301900409",
		RequesterPhone:   "+998901900409",
		RequesterName:    "Aziz",
		RoleSought:       string(auth.RoleRetailer),
		CreatedAt:        now,
	}, nil)
	if err != nil || jr.Status != innlookup.JoinStatusPending {
		t.Fatalf("JoinRequest creation failed: %+v err=%v", jr, err)
	}
	decided, err := memINN.DecideJoinRequest(ctx, innlookup.TenantTypeRetailer, "ret-existing-409", jr.RequestID, innlookup.JoinStatusApproved, "admin-409", "ok", now.Add(time.Minute), nil)
	if err != nil || decided.Status != innlookup.JoinStatusApproved {
		t.Fatalf("JoinRequest decide failed: %+v err=%v", decided, err)
	}

	// 5. 72h Rollback boundary: expired fails closed, inside window succeeds cleanly.
	win := time.Duration(fixture.RollbackWindowHours) * time.Hour
	if _, err := RollbackSession(ctx, repo, sup, "sess-dup-orders", now.Add(win+time.Minute)); !errors.Is(err, ErrRollbackWindowExpired) {
		t.Fatalf("expected ErrRollbackWindowExpired at %dh+1m, got %v", fixture.RollbackWindowHours, err)
	}
	rbRes, err := RollbackSession(ctx, repo, sup, "sess-dup-orders", now.Add(win-time.Minute))
	if err != nil || rbRes.SupersededOrders != 1 {
		t.Fatalf("expected rollback within %dh to supersede 1 order, got err=%v res=%+v", fixture.RollbackWindowHours, err, rbRes)
	}
}

func TestGoldenZeroBasedRowIndexAndResumeAfterFirstChunk(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	sup, projID := "sup-golden-zero", "proj_golden_zero"
	repo, store := newGoldenRepo(t, sup, projID, now)

	sess := Session{
		SupplierID: sup, SessionID: "sess-zero-1", ProjectID: projID, BundleID: "bnd-zero",
		Pack: PackWarehouses, Status: SessionStatusDiscovered, ContentSHA256: "sha-zero", TotalRows: 2,
	}
	repo.m4TestRepo.mu.Lock()
	repo.m4TestRepo.sessions[sess.SessionID] = sess
	for i, name := range []string{"Warehouse Zero", "Warehouse One"} {
		b, _ := json.Marshal(map[string]any{"external_warehouse_id": name, "name": name})
		repo.m4TestRepo.stagedRows[sess.SessionID] = append(repo.m4TestRepo.stagedRows[sess.SessionID], StagedRow{
			SupplierID: sup, SessionID: sess.SessionID, RowIndex: int64(i), RawData: b,
		})
	}
	doc := SessionMappingDocument{
		SessionID: sess.SessionID, Pack: PackWarehouses, AllRequiredMapped: true,
		Mappings: []ColumnMappingProposal{
			{SourceColumn: "external_warehouse_id", TargetField: "external_warehouse_id", Confidence: 0.99, Source: "manual", Confirmed: true},
			{SourceColumn: "name", TargetField: "name", Confidence: 0.99, Source: "manual", Confirmed: true},
		},
	}
	doc.MappingHash = ComputeMappingHash(doc)
	repo.m4TestRepo.sessionMaps[sess.SessionID] = doc
	repo.m4TestRepo.mu.Unlock()

	rep, _, err := EvaluateSessionDryRun(ctx, repo, DryRunOptions{
		SupplierID: sup, ProjectID: projID, SessionID: sess.SessionID, ApplyMode: ApplyModeMerge, Now: now, LiveReader: repo,
	})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if _, err := ApproveSession(ctx, repo, sup, sess.SessionID, ApproveSessionRequest{
		ApplyMode: ApplyModeMerge, ExpectedReportID: rep.ReportID,
	}, now); err != nil {
		t.Fatalf("approve: %v", err)
	}

	if _, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup, SessionID: sess.SessionID, ChunkSize: 1, FailAfterChunk: 1, Now: now,
	}); err == nil {
		t.Fatalf("expected injected chunk failure after chunk 1")
	}
	failedSess, _ := repo.GetSession(ctx, sup, sess.SessionID)
	if failedSess.ResumeRowIndex != 0 || failedSess.LastChunkID != "chk_sess-zero-1_0_0" || len(store.Warehouses) != 1 {
		t.Fatalf("expected row 0 checkpointed (ResumeRowIndex=0 LastChunkID=chk_sess-zero-1_0_0), got sess=%+v warehouses=%d", failedSess, len(store.Warehouses))
	}

	resumed, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup, SessionID: sess.SessionID, ChunkSize: 1, Now: now,
	})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if resumed.Created != 1 || resumed.LastRowIndex != 1 || len(store.Warehouses) != 2 {
		t.Fatalf("expected row 0 skipped on resume and row 1 applied, got res=%+v warehouses=%d", resumed, len(store.Warehouses))
	}
}
