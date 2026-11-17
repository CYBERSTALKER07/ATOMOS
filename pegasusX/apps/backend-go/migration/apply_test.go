package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

type m5TestRepo struct {
	*m4TestRepo
	store   *MemoryApplyStore
	lockMu  sync.Mutex
	locks   map[string]WarehouseLock
	nowTime time.Time
}

func newM5TestEnv(t *testing.T, supplierID string, now time.Time) (*m5TestRepo, *MemoryApplyStore, Project) {
	t.Helper()
	store := NewMemoryApplyStore()
	store.NowFn = func() time.Time { return now }
	store.Supplier[supplierID] = MemorySupplierMeta{
		HomeCell: "8928308280fffff",
		Currency: "UZS",
	}
	base := newM4TestRepo()
	repo := &m5TestRepo{
		m4TestRepo: base,
		store:      store,
		locks:      make(map[string]WarehouseLock),
		nowTime:    now,
	}
	proj := Project{
		SupplierID: supplierID,
		ProjectID:  "proj_m5",
		Name:       "M5 Enterprise Cutover",
		Mode:       ProjectModeCutover,
		Status:     ProjectStatusDraft,
		CreatedBy:  "usr-ops",
	}
	_, err := repo.CreateProject(context.Background(), proj)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return repo, store, proj
}

func (r *m5TestRepo) LoadLiveOverlay(ctx context.Context, supplierID string) (LiveOverlaySnapshot, error) {
	return r.store.LoadLiveOverlay(ctx, supplierID)
}

func (r *m5TestRepo) UpdateApplyProgress(_ context.Context, supplierID, sessionID string, resumeRowIndex int64, lastChunkID string) error {
	r.m4TestRepo.mu.Lock()
	defer r.m4TestRepo.mu.Unlock()
	s, ok := r.m4TestRepo.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return ErrNotFound
	}
	s.ResumeRowIndex = resumeRowIndex
	s.LastChunkID = lastChunkID
	r.m4TestRepo.sessions[sessionID] = s
	return nil
}

func (r *m5TestRepo) AcquireWarehouseLock(_ context.Context, supplierID, warehouseID, sessionID string, ttl time.Duration) (WarehouseLock, error) {
	r.lockMu.Lock()
	defer r.lockMu.Unlock()
	k := supplierID + "|" + warehouseID
	if cur, ok := r.locks[k]; ok && cur.SessionID != sessionID && cur.ExpiresAt.After(r.nowTime) {
		return WarehouseLock{}, ErrWarehouseLockBusy
	}
	wl := WarehouseLock{
		SupplierID:  supplierID,
		WarehouseID: warehouseID,
		SessionID:   sessionID,
		AcquiredAt:  r.nowTime,
		ExpiresAt:   r.nowTime.Add(ttl),
	}
	r.locks[k] = wl
	return wl, nil
}

func (r *m5TestRepo) RenewWarehouseLock(ctx context.Context, supplierID, warehouseID, sessionID string, ttl time.Duration) (WarehouseLock, error) {
	return r.AcquireWarehouseLock(ctx, supplierID, warehouseID, sessionID, ttl)
}

func (r *m5TestRepo) ReleaseWarehouseLock(_ context.Context, supplierID, warehouseID, sessionID string) error {
	r.lockMu.Lock()
	defer r.lockMu.Unlock()
	k := supplierID + "|" + warehouseID
	if cur, ok := r.locks[k]; ok && cur.SessionID == sessionID {
		delete(r.locks, k)
	}
	return nil
}

func (r *m5TestRepo) ListSnapshotsBySession(_ context.Context, _, sessionID string) ([]MigrationSnapshotRecord, error) {
	return r.store.ListSnapshotsForSession(sessionID), nil
}

func (r *m5TestRepo) ExecuteSessionRollback(_ context.Context, supplierID, sessionID string, _ []MigrationSnapshotRecord, _ time.Time) (RollbackResult, error) {
	res, err := r.store.ExecuteRollbackInMemory(supplierID, sessionID)
	if err != nil {
		return RollbackResult{}, err
	}
	r.m4TestRepo.mu.Lock()
	defer r.m4TestRepo.mu.Unlock()
	s, ok := r.m4TestRepo.sessions[sessionID]
	if !ok || s.SupplierID != supplierID {
		return RollbackResult{}, ErrNotFound
	}
	s.Status = SessionStatusRolledBack
	r.m4TestRepo.sessions[sessionID] = s
	res.Session = s
	return res, nil
}

func stageDryRunAndApprove(
	t *testing.T,
	repo *m5TestRepo,
	supplierID, projectID, sessionID string,
	pack Pack,
	rows []map[string]any,
	colMap map[string]string,
	applyMode string,
	setOnHandConfirmed bool,
	now time.Time,
) Session {
	t.Helper()
	ctx := context.Background()

	var props []ColumnMappingProposal
	for hdr, cf := range colMap {
		props = append(props, ColumnMappingProposal{
			SourceColumn: hdr,
			TargetField:  cf,
			Confidence:   0.99,
			Source:       "manual",
			Confirmed:    true,
		})
	}
	doc := SessionMappingDocument{
		SessionID:         sessionID,
		Pack:              pack,
		Mappings:          props,
		AllRequiredMapped: true,
	}
	doc.MappingHash = ComputeMappingHash(doc)

	sess := Session{
		SupplierID:    supplierID,
		SessionID:     sessionID,
		ProjectID:     projectID,
		BundleID:      "bnd_m5",
		Pack:          pack,
		Status:        SessionStatusDiscovered,
		ContentSHA256: "sha_" + sessionID,
		TotalRows:     int64(len(rows)),
	}
	repo.m4TestRepo.mu.Lock()
	repo.m4TestRepo.sessions[sessionID] = sess
	var staged []StagedRow
	for i, raw := range rows {
		b, _ := json.Marshal(raw)
		staged = append(staged, StagedRow{
			SupplierID: supplierID,
			SessionID:  sessionID,
			RowIndex:   int64(i + 1),
			RawData:    b,
		})
	}
	repo.m4TestRepo.stagedRows[sessionID] = staged
	repo.m4TestRepo.sessionMaps[sessionID] = doc
	repo.m4TestRepo.mu.Unlock()

	rep, summary, err := EvaluateSessionDryRun(ctx, repo, DryRunOptions{
		SupplierID: supplierID,
		ProjectID:  projectID,
		SessionID:  sessionID,
		ApplyMode:  applyMode,
		Now:        now,
		LiveReader: repo,
	})
	if err != nil {
		t.Fatalf("dry-run %s (%s): %v", sessionID, pack, err)
	}
	if !summary.ReadyToApprove || rep.BlockingCount > 0 {
		t.Fatalf("unexpected dry-run blocking issues for %s (%s): %s", sessionID, pack, string(rep.SummaryJSON))
	}
	approved, err := ApproveSession(ctx, repo, supplierID, sessionID, ApproveSessionRequest{
		ApplyMode:          applyMode,
		SetOnHandConfirmed: setOnHandConfirmed,
		ExpectedReportID:   rep.ReportID,
	}, now)
	if err != nil {
		t.Fatalf("approve %s: %v", sessionID, err)
	}
	return approved
}

func TestApply_WarehousesAndPartnerPlantMaps_AndCatalogUnitGuard(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m5-cat"
	repo, store, proj := newM5TestEnv(t, sup, now)

	// 1. Apply Warehouses & verify PartnerPlantMaps + default warehouse resolution.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-wh-1", PackWarehouses, []map[string]any{
		{"external_warehouse_id": "1C-WH-01", "name": "Tashkent Central", "address": "Chilonzor 5", "is_default": "true"},
	}, map[string]string{
		"external_warehouse_id": "external_warehouse_id",
		"name":                  "name",
		"address":               "address",
		"is_default":            "is_default",
	}, ApplyModeMerge, false, now)

	whRes, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-wh-1", Now: now})
	if err != nil {
		t.Fatalf("apply warehouses: %v", err)
	}
	if whRes.Created != 1 || whRes.Session.Status != SessionStatusApplied || whRes.Session.RollbackUntil == nil {
		t.Fatalf("unexpected warehouse apply result: %+v", whRes)
	}
	resolvedWH, ok := partner.ResolvePlantWarehouse(partner.TenantSupplier, sup, "1C-WH-01")
	if !ok || resolvedWH == "" {
		t.Fatalf("expected PartnerPlantMaps memory resolution hit, got ok=%v wh=%q", ok, resolvedWH)
	}
	if _, ok := store.Warehouses[resolvedWH]; !ok {
		t.Fatalf("expected warehouse %s persisted in store", resolvedWH)
	}

	// 2. Apply Catalog creates new SKU + updates existing SKU; preserves old Product before-image for rollback.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-cat-1", PackProductsPrices, []map[string]any{
		{"sku": "SKU-TEA-1", "name": "Black Tea 250g", "price": "18500", "currency": "UZS", "unit": "PCS", "barcode": "47800001"},
		{"sku": "SKU-MILK-1", "name": "Cold Milk 1L", "price": "12000", "currency": "UZS", "unit": "PCS", "category": "cold dairy", "shelf_life_days": "14"},
	}, map[string]string{
		"sku":             "sku",
		"name":            "name",
		"price":           "price",
		"currency":        "currency",
		"unit":            "unit",
		"barcode":         "barcode",
		"category":        "category",
		"shelf_life_days": "shelf_life_days",
	}, ApplyModeMerge, false, now)

	catRes, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-cat-1", Now: now})
	if err != nil {
		t.Fatalf("apply catalog: %v", err)
	}
	if catRes.Created != 2 {
		t.Fatalf("expected 2 created products, got %+v", catRes)
	}
	milkID := store.IdentityMap[identityKey(sup, PackProductsPrices, "SKU-MILK-1")].InternalID
	if milkID == "" || !store.Products[milkID].RequiresColdChain || !store.Products[milkID].IsPerishable || store.Products[milkID].MinShelfLifeDays != 14 {
		t.Fatalf("expected cold-chain & perishable metadata on %q, got %+v", milkID, store.Products[milkID])
	}

	// Unit change allowed when stock is 0: PCS -> PACK on SKU-MILK-1.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-cat-upd-ok", PackProductsPrices, []map[string]any{
		{"sku": "SKU-MILK-1", "name": "Cold Milk 1L", "price": "13000", "currency": "UZS", "unit": "PACK"},
	}, map[string]string{
		"sku":      "sku",
		"name":     "name",
		"price":    "price",
		"currency": "currency",
		"unit":     "unit",
	}, ApplyModeMerge, false, now)
	updRes, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-cat-upd-ok", Now: now})
	if err != nil {
		t.Fatalf("unit change with 0 stock should succeed: %v", err)
	}
	if updRes.Updated != 1 || store.Products[milkID].PriceMinor != 1300000 || store.Products[milkID].Unit != "PACK" || store.Products[milkID].Version != 2 {
		t.Fatalf("unexpected updated product state: %+v", store.Products[milkID])
	}

	// Now seed non-zero stock on SKU-TEA-1 and prove unit change PCS -> KG is blocked with ErrUnitChangeWithNonZeroStock.
	teaID := store.IdentityMap[identityKey(sup, PackProductsPrices, "SKU-TEA-1")].InternalID
	store.Inventory[inventoryKey(sup, resolvedWH, teaID)] = StoredStock{
		SupplierID:     sup,
		WarehouseID:    resolvedWH,
		ProductID:      teaID,
		QuantityOnHand: 25,
	}
	sessDirect := Session{
		SupplierID:     sup,
		SessionID:      "sess-cat-block",
		ProjectID:      proj.ProjectID,
		Pack:           PackProductsPrices,
		Status:         SessionStatusApproved,
		ApplyMode:      ApplyModeMerge,
		DryRunReportID: "rep-manual",
		ContentSHA256:  "sha-block",
	}
	docDirect := SessionMappingDocument{
		SessionID:         "sess-cat-block",
		Pack:              PackProductsPrices,
		AllRequiredMapped: true,
		Mappings: []ColumnMappingProposal{
			{SourceColumn: "sku", TargetField: "sku", Confidence: 0.99, Confirmed: true},
			{SourceColumn: "name", TargetField: "name", Confidence: 0.99, Confirmed: true},
			{SourceColumn: "unit", TargetField: "unit", Confidence: 0.99, Confirmed: true},
		},
	}
	docDirect.MappingHash = ComputeMappingHash(docDirect)
	fp := ComputeSessionDryRunFingerprint(sessDirect, docDirect, ApplyModeMerge)
	sessDirect.DryRunFingerprint = fp
	repo.m4TestRepo.mu.Lock()
	repo.m4TestRepo.sessions["sess-cat-block"] = sessDirect
	repo.m4TestRepo.sessionMaps["sess-cat-block"] = docDirect
	rawRow, _ := json.Marshal(map[string]any{"sku": "SKU-TEA-1", "name": "Black Tea 250g", "unit": "KG"})
	repo.m4TestRepo.stagedRows["sess-cat-block"] = []StagedRow{{SupplierID: sup, SessionID: "sess-cat-block", RowIndex: 1, RawData: rawRow}}
	repo.m4TestRepo.reports["rep-manual"] = MigrationReport{
		ReportID:           "rep-manual",
		SupplierID:         sup,
		SessionID:          "sess-cat-block",
		ContentFingerprint: fp,
		ExpiresAt:          now.Add(24 * time.Hour),
	}
	repo.m4TestRepo.mu.Unlock()

	_, err = ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-cat-block", Now: now})
	if !errors.Is(err, ErrUnitChangeWithNonZeroStock) {
		t.Fatalf("expected ErrUnitChangeWithNonZeroStock, got %v", err)
	}

	// Verify 72h Rollback of sess-cat-upd-ok restores Version=1, Price=12000, Unit=PCS on SKU-MILK-1.
	rbRes, err := RollbackSession(ctx, repo, sup, "sess-cat-upd-ok", now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("rollback sess-cat-upd-ok: %v", err)
	}
	if rbRes.Session.Status != SessionStatusRolledBack {
		t.Fatalf("expected ROLLED_BACK status, got %s", rbRes.Session.Status)
	}
	restored := store.Products[milkID]
	if restored.PriceMinor != 1200000 || restored.Unit != "PCS" || restored.Version != 1 {
		t.Fatalf("expected pre-update SKU-MILK-1 restored, got %+v", restored)
	}
}

func TestApply_Customers_MergeByINNAndPhone_WithoutCredFabrication(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m5-cust"
	repo, store, proj := newM5TestEnv(t, sup, now)

	// Pre-seed another supplier's linked retailer sharing INN 301234567 to prove cross-supplier (home_cell, inn) merge.
	store.Retailers["ret-existing-inn"] = StoredCustomer{
		RetailerID:  "ret-existing-inn",
		Phone:       "+998901112233",
		Name:        "Samarkand Mart LLC",
		CountryCode: "UZ",
	}
	store.Links[supplierLinkKey("other-sup", "ret-existing-inn")] = StoredCustomerLink{
		SupplierID:  "other-sup",
		RetailerID:  "ret-existing-inn",
		TaxIdTIN:    "301234567",
		DedupStatus: "CLEAN",
	}
	// Pre-seed another retailer matching normalized phone only.
	store.Retailers["ret-existing-phone"] = StoredCustomer{
		RetailerID:  "ret-existing-phone",
		Phone:       "+998909998877",
		Name:        "Anhor Market",
		CountryCode: "UZ",
	}

	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-cust-1", PackCustomers, []map[string]any{
		{"external_customer_id": "1C-CUST-1", "customer_name": "Samarkand Mart LLC", "inn": "301234567", "phone": "+998901112233", "address": "Registan 1"},
		{"external_customer_id": "1C-CUST-2", "customer_name": "Anhor Market", "inn": "", "phone": "998 (90) 999-88-77", "address": "Navoi 10"},
		{"external_customer_id": "1C-CUST-DUP", "customer_name": "Dup INN Row", "inn": "301234567", "phone": "+998901112233"},
		{"external_customer_id": "1C-CUST-NEW", "customer_name": "Fresh Shop INN Only", "inn": "309876543", "phone": ""},
	}, map[string]string{
		"external_customer_id": "external_customer_id",
		"customer_name":        "customer_name",
		"inn":                  "inn",
		"phone":                "phone",
		"address":              "address",
	}, ApplyModeMerge, false, now)

	res, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-cust-1", Now: now})
	if err != nil {
		t.Fatalf("apply customers: %v", err)
	}
	if res.Updated != 2 || res.Created != 1 || res.Skipped != 1 {
		t.Fatalf("expected Updated=2 Created=1 Skipped=1, got %+v", res)
	}
	if _, ok := store.Links[supplierLinkKey(sup, "ret-existing-inn")]; !ok {
		t.Fatalf("expected link created onto existing INN retailer ret-existing-inn")
	}
	if _, ok := store.Links[supplierLinkKey(sup, "ret-existing-phone")]; !ok {
		t.Fatalf("expected link created onto existing Phone retailer ret-existing-phone")
	}
	// Verify synthetic INN-only counterparty uses non-dialable +000... placeholder without fabricating login credentials.
	var foundNew bool
	for _, r := range store.Retailers {
		if r.Name == "Fresh Shop INN Only" {
			foundNew = true
			if r.Phone != "+000309876543" {
				t.Fatalf("expected non-dialable +000... placeholder without fabricating login credentials, got %q", r.Phone)
			}
		}
	}
	if !foundNew {
		t.Fatalf("expected newly created INN-only retailer in store")
	}

	// Verify rollback deletes this supplier's links & identity maps without deleting shared cross-supplier Retailer rows.
	if _, err := RollbackSession(ctx, repo, sup, "sess-cust-1", now.Add(time.Hour)); err != nil {
		t.Fatalf("rollback customers: %v", err)
	}
	if _, ok := store.Links[supplierLinkKey(sup, "ret-existing-inn")]; ok {
		t.Fatalf("expected supplier link deleted on rollback")
	}
	if _, ok := store.Retailers["ret-existing-inn"]; !ok {
		t.Fatalf("expected pre-existing shared Retailer row preserved across rollback")
	}
}

func TestApply_StockMergeAndSetOnHand_ChunkCrashResumeAndRollback(t *testing.T) {
	prevLots := stocklots.LotsEnabled()
	stocklots.SetLotsEnabled(false)
	t.Cleanup(func() { stocklots.SetLotsEnabled(prevLots) })

	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m5-stk"
	repo, store, proj := newM5TestEnv(t, sup, now)

	store.Warehouses["WH-MAIN"] = StoredWarehouse{WarehouseID: "WH-MAIN", SupplierID: sup, Name: "Main WH", IsActive: true, IsOnShift: true}
	for _, sku := range []string{"SKU-1", "SKU-2", "SKU-3", "SKU-4"} {
		store.Products[sku] = StoredProduct{
			ProductID: sku, SupplierID: sup, CategoryID: "general", Name: sku, PriceMinor: 1000, Currency: "UZS", Unit: "PCS", IsActive: true, Version: 1,
		}
	}
	store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-1")] = StoredStock{
		SupplierID: sup, WarehouseID: "WH-MAIN", ProductID: "SKU-1", QuantityOnHand: 10, QuantityReserved: 4, ReorderThreshold: 2,
	}

	// Crash after chunk 1 (2 rows per chunk, 4 rows total -> rows 1..2 committed, crash before chunk 2, then resume).
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-stk-merge", PackStock, []map[string]any{
		{"sku": "SKU-1", "warehouse_id": "WH-MAIN", "quantity_on_hand": "15"},
		{"sku": "SKU-2", "warehouse_id": "WH-MAIN", "quantity_on_hand": "20"},
		{"sku": "SKU-3", "warehouse_id": "WH-MAIN", "quantity_on_hand": "30"},
		{"sku": "SKU-4", "warehouse_id": "WH-MAIN", "quantity_on_hand": "40"},
	}, map[string]string{
		"sku":              "sku",
		"warehouse_id":     "warehouse_id",
		"quantity_on_hand": "quantity_on_hand",
	}, ApplyModeMerge, false, now)

	_, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID:     sup,
		SessionID:      "sess-stk-merge",
		ApplyMode:      ApplyModeMerge,
		ChunkSize:      2,
		FailAfterChunk: 1,
		Now:            now,
	})
	if err == nil {
		t.Fatalf("expected injected chunk crash after chunk 1")
	}
	failedSess, _ := repo.GetSession(ctx, sup, "sess-stk-merge")
	if failedSess.Status != SessionStatusFailed || failedSess.ResumeRowIndex != 2 {
		t.Fatalf("expected FAILED with ResumeRowIndex=2, got status=%s resume=%d", failedSess.Status, failedSess.ResumeRowIndex)
	}
	if got := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-1")].QuantityOnHand; got != 25 {
		t.Fatalf("expected chunk 1 SKU-1 merged 10+15=25, got %d", got)
	}
	if _, ok := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-3")]; ok {
		t.Fatalf("chunk 2 SKU-3 must NOT be committed before resume")
	}
	if len(repo.locks) != 0 {
		t.Fatalf("expected warehouse lock released even on chunk crash, still held: %+v", repo.locks)
	}

	// Resume from ResumeRowIndex=2 with zero double-counting of SKU-1/SKU-2.
	var progressEvents int
	resumed, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup,
		SessionID:  "sess-stk-merge",
		ApplyMode:  ApplyModeMerge,
		ChunkSize:  2,
		Now:        now,
		ProgressEmit: func(_ string, _, _ int64, _ string) {
			progressEvents++
		},
	})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if resumed.Session.Status != SessionStatusApplied || resumed.LastRowIndex != 4 || progressEvents != 1 {
		t.Fatalf("unexpected resumed result: %+v (progress=%d)", resumed, progressEvents)
	}
	if stk1 := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-1")]; stk1.QuantityOnHand != 25 || stk1.QuantityReserved != 4 {
		t.Fatalf("SKU-1 must remain 25 (no double-count) and preserve QuantityReserved=4, got %+v", stk1)
	}
	if stk4 := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-4")]; stk4.QuantityOnHand != 40 {
		t.Fatalf("expected SKU-4=40, got %+v", stk4)
	}

	// Idempotent re-apply on APPLIED returns AlreadyApplied=true without mutating stock.
	idem, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-stk-merge", Now: now})
	if err != nil || !idem.AlreadyApplied || store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-1")].QuantityOnHand != 25 {
		t.Fatalf("expected idempotent AlreadyApplied=true with stock unchanged, got %+v err=%v", idem, err)
	}

	// SET_ON_HAND overwrites QuantityOnHand while preserving QuantityReserved.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-stk-set", PackStock, []map[string]any{
		{"sku": "SKU-1", "warehouse_id": "WH-MAIN", "quantity_on_hand": "12"},
	}, map[string]string{
		"sku":              "sku",
		"warehouse_id":     "warehouse_id",
		"quantity_on_hand": "quantity_on_hand",
	}, ApplyModeSetOnHand, true, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup, SessionID: "sess-stk-set", ApplyMode: ApplyModeSetOnHand, Now: now,
	}); err != nil {
		t.Fatalf("SET_ON_HAND failed: %v", err)
	}
	if stk1 := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-1")]; stk1.QuantityOnHand != 12 || stk1.QuantityReserved != 4 {
		t.Fatalf("expected SET_ON_HAND=12 preserving Reserved=4, got %+v", stk1)
	}

	// Atomic chunk rollback: row 1 touches SKU-2, row 2 sets SKU-1 below reserved (3 < 4) -> NEITHER row persists.
	sessAtomic := Session{
		SupplierID: sup, SessionID: "sess-stk-atomic-fail", ProjectID: proj.ProjectID,
		Pack: PackStock, Status: SessionStatusApproved, ApplyMode: ApplyModeSetOnHand, SetOnHandConfirmed: true,
		DryRunReportID: "rep-atomic", ContentSHA256: "sha-atomic",
	}
	docAtomic := SessionMappingDocument{
		SessionID: "sess-stk-atomic-fail", Pack: PackStock, AllRequiredMapped: true,
		Mappings: []ColumnMappingProposal{
			{SourceColumn: "sku", TargetField: "sku", Confidence: 0.99, Confirmed: true},
			{SourceColumn: "warehouse_id", TargetField: "warehouse_id", Confidence: 0.99, Confirmed: true},
			{SourceColumn: "quantity_on_hand", TargetField: "quantity_on_hand", Confidence: 0.99, Confirmed: true},
		},
	}
	docAtomic.MappingHash = ComputeMappingHash(docAtomic)
	fpAtomic := ComputeSessionDryRunFingerprint(sessAtomic, docAtomic, ApplyModeSetOnHand)
	sessAtomic.DryRunFingerprint = fpAtomic
	r1, _ := json.Marshal(map[string]any{"sku": "SKU-2", "warehouse_id": "WH-MAIN", "quantity_on_hand": "999"})
	r2, _ := json.Marshal(map[string]any{"sku": "SKU-1", "warehouse_id": "WH-MAIN", "quantity_on_hand": "3"})
	repo.m4TestRepo.mu.Lock()
	repo.m4TestRepo.sessions["sess-stk-atomic-fail"] = sessAtomic
	repo.m4TestRepo.sessionMaps["sess-stk-atomic-fail"] = docAtomic
	repo.m4TestRepo.stagedRows["sess-stk-atomic-fail"] = []StagedRow{
		{SupplierID: sup, SessionID: "sess-stk-atomic-fail", RowIndex: 1, RawData: r1},
		{SupplierID: sup, SessionID: "sess-stk-atomic-fail", RowIndex: 2, RawData: r2},
	}
	repo.m4TestRepo.reports["rep-atomic"] = MigrationReport{
		ReportID: "rep-atomic", SupplierID: sup, SessionID: "sess-stk-atomic-fail",
		ContentFingerprint: fpAtomic, ExpiresAt: now.Add(24 * time.Hour),
	}
	repo.m4TestRepo.mu.Unlock()

	_, err = ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup, SessionID: "sess-stk-atomic-fail", ApplyMode: ApplyModeSetOnHand, ChunkSize: 10, Now: now,
	})
	if !errors.Is(err, ErrOnHandBelowReserved) {
		t.Fatalf("expected ErrOnHandBelowReserved, got %v", err)
	}
	if store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-2")].QuantityOnHand != 20 {
		t.Fatalf("expected atomic chunk abort preserving SKU-2=20, got %d", store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-2")].QuantityOnHand)
	}

	// M4 72h Rollback reverses sess-stk-set then sess-stk-merge back to exact initial state (SKU-1=10/4, SKU-2..4 deleted).
	if _, err := RollbackSession(ctx, repo, sup, "sess-stk-set", now.Add(1*time.Hour)); err != nil {
		t.Fatalf("rollback sess-stk-set: %v", err)
	}
	if _, err := RollbackSession(ctx, repo, sup, "sess-stk-merge", now.Add(2*time.Hour)); err != nil {
		t.Fatalf("rollback sess-stk-merge: %v", err)
	}
	if stk1 := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-1")]; stk1.QuantityOnHand != 10 || stk1.QuantityReserved != 4 {
		t.Fatalf("expected SKU-1 restored to exact 10/4, got %+v", stk1)
	}
	if _, ok := store.Inventory[inventoryKey(sup, "WH-MAIN", "SKU-4")]; ok {
		t.Fatalf("expected newly inserted SKU-4 inventory row deleted on rollback")
	}
}

func TestApply_LotsPutawayDeltaAndRollup_WithColdChainWarningAndFeatureFlagGuard(t *testing.T) {
	prevLots := stocklots.LotsEnabled()
	prevWaves := stocklots.PickWavesEnabled()
	stocklots.SetLotsEnabled(true)
	stocklots.SetPickWavesEnabled(true)
	t.Cleanup(func() {
		stocklots.SetLotsEnabled(prevLots)
		stocklots.SetPickWavesEnabled(prevWaves)
	})

	ctx := context.Background()
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sup := "sup-m5-lots"
	repo, store, proj := newM5TestEnv(t, sup, now)

	store.Warehouses["WH-COLD"] = StoredWarehouse{WarehouseID: "WH-COLD", SupplierID: sup, Name: "Cold WH", IsActive: true, IsOnShift: true}
	store.Products["SKU-VAX"] = StoredProduct{
		ProductID: "SKU-VAX", SupplierID: sup, CategoryID: "pharma", Name: "Vaccine Vial",
		PriceMinor: 50000, Currency: "UZS", Unit: "PCS", HandlingClass: "COLD_CHAIN",
		RequiresColdChain: true, IsPerishable: true, MinShelfLifeDays: 30, IsActive: true, Version: 1,
	}

	// Confirm PackStock is rejected in a separate project when WMS_LOTS_ENABLED=true.
	_, _ = repo.CreateProject(ctx, Project{
		SupplierID: sup, ProjectID: "proj_stk_blocked", Name: "Blocked Stock", Mode: ProjectModeCutover, Status: ProjectStatusDraft,
	})
	stageDryRunAndApprove(t, repo, sup, "proj_stk_blocked", "sess-stk-blocked", PackStock, []map[string]any{
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "quantity_on_hand": "10"},
	}, map[string]string{"sku": "sku", "warehouse_id": "warehouse_id", "quantity_on_hand": "quantity_on_hand"}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-stk-blocked", Now: now}); !errors.Is(err, inventory.ErrLotsEnabledDirectV2) {
		t.Fatalf("expected ErrLotsEnabledDirectV2 on stock pack while lots enabled, got %v", err)
	}

	// 1. Initial apply of TWO lots for the SAME SKU in ONE chunk -> rollup sums both = 30.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-lots-1", PackLots, []map[string]any{
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "lot_number": "LOT-A", "expiry_date": "2027-06-01", "quantity_on_hand": "10"},
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "lot_number": "LOT-B", "expiry_date": "2027-09-01", "quantity_on_hand": "20"},
	}, map[string]string{
		"sku": "sku", "warehouse_id": "warehouse_id", "lot_number": "lot_number", "expiry_date": "expiry_date", "quantity_on_hand": "quantity_on_hand",
	}, ApplyModeMerge, false, now)

	res1, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-lots-1", Now: now})
	if err != nil {
		t.Fatalf("apply lots-1: %v", err)
	}
	if res1.Created != 2 {
		t.Fatalf("expected 2 created lots, got %+v", res1)
	}
	var coldWarnCount int
	for _, w := range res1.Warnings {
		if w.Code == "cold_lot_migrated_without_live_baseline" {
			coldWarnCount++
		}
	}
	if coldWarnCount != 2 {
		t.Fatalf("expected 2 cold_lot_migrated_without_live_baseline warnings, got %+v", res1.Warnings)
	}
	if stk := store.Inventory[inventoryKey(sup, "WH-COLD", "SKU-VAX")]; stk.QuantityOnHand != 30 {
		t.Fatalf("expected rolled-up SupplierInventoryV2=30, got %+v", stk)
	}

	// 2. Re-import lots with +5 positive delta (LOT-A: 10->15), 0 delta (LOT-B: 20->20 skipped), and confirm rollup = 35.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-lots-2", PackLots, []map[string]any{
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "lot_number": "LOT-A", "expiry_date": "2027-06-01", "quantity_on_hand": "15"},
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "lot_number": "LOT-B", "expiry_date": "2027-09-01", "quantity_on_hand": "20"},
	}, map[string]string{
		"sku": "sku", "warehouse_id": "warehouse_id", "lot_number": "lot_number", "expiry_date": "expiry_date", "quantity_on_hand": "quantity_on_hand",
	}, ApplyModeMerge, false, now)

	res2, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-lots-2", Now: now})
	if err != nil {
		t.Fatalf("apply lots-2: %v", err)
	}
	if res2.Updated != 1 || res2.Skipped != 1 {
		t.Fatalf("expected Updated=1 Skipped=1, got %+v", res2)
	}
	if stk := store.Inventory[inventoryKey(sup, "WH-COLD", "SKU-VAX")]; stk.QuantityOnHand != 35 {
		t.Fatalf("expected rolled-up SupplierInventoryV2=35 after delta, got %+v", stk)
	}

	// 3. Negative delta adjustment (LOT-A: 15->8) updates rollup to 28.
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-lots-3", PackLots, []map[string]any{
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "lot_number": "LOT-A", "expiry_date": "2027-06-01", "quantity_on_hand": "8"},
	}, map[string]string{
		"sku": "sku", "warehouse_id": "warehouse_id", "lot_number": "lot_number", "expiry_date": "expiry_date", "quantity_on_hand": "quantity_on_hand",
	}, ApplyModeMerge, false, now)
	res3, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-lots-3", Now: now})
	if err != nil || res3.Updated != 1 {
		t.Fatalf("apply lots-3 negative delta failed: err=%v res=%+v", err, res3)
	}
	if stk := store.Inventory[inventoryKey(sup, "WH-COLD", "SKU-VAX")]; stk.QuantityOnHand != 28 {
		t.Fatalf("expected rolled-up SupplierInventoryV2=28 after downward adjustment, got %+v", stk)
	}

	// Rollback sess-lots-3 restores LOT-A=15 and V2=35.
	if _, err := RollbackSession(ctx, repo, sup, "sess-lots-3", now.Add(time.Hour)); err != nil {
		t.Fatalf("rollback sess-lots-3: %v", err)
	}
	if stk := store.Inventory[inventoryKey(sup, "WH-COLD", "SKU-VAX")]; stk.QuantityOnHand != 35 {
		t.Fatalf("expected V2=35 after lot rollback, got %+v", stk)
	}

	// 4. Downward adjustment below QuantityReserved must fail closed with ErrOnHandBelowReserved.
	lotAID := deterministicEntityID("lot", sup, string(PackLots), "WH-COLD|SKU-VAX|LOT-A")
	lotA := store.Lots[lotAID]
	lotA.QuantityReserved = 10
	store.Lots[lotAID] = lotA
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-lots-below-res", PackLots, []map[string]any{
		{"sku": "SKU-VAX", "warehouse_id": "WH-COLD", "lot_number": "LOT-A", "expiry_date": "2027-06-01", "quantity_on_hand": "6"},
	}, map[string]string{
		"sku": "sku", "warehouse_id": "warehouse_id", "lot_number": "lot_number", "expiry_date": "expiry_date", "quantity_on_hand": "quantity_on_hand",
	}, ApplyModeMerge, false, now)
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: sup, SessionID: "sess-lots-below-res", Now: now}); !errors.Is(err, ErrOnHandBelowReserved) {
		t.Fatalf("expected ErrOnHandBelowReserved when adjusting lot below reserved, got %v", err)
	}

	// 5. SET_ON_HAND mode is forbidden for lots pack at runtime as well.
	if _, err := ApplySession(ctx, repo, store, ApplyOptions{
		SupplierID: sup, SessionID: "sess-lots-below-res", ApplyMode: ApplyModeSetOnHand, Now: now,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for SET_ON_HAND on lots pack, got %v", err)
	}
}

type abortedOnceApplyStore struct {
	*MemoryApplyStore
	retryChunk int
	calls      int
}

func (s *abortedOnceApplyStore) RunChunkTx(ctx context.Context, supplierID, sessionID string, fn func(tx ApplyTx) error) error {
	s.calls++
	if s.calls == s.retryChunk {
		// Mirror Spanner ReadWriteTransaction Aborted retry: attempt 1 executes fn(tx) against
		// a discarded snapshot (which rolls back storage writes), then re-runs fn(tx) on attempt 2.
		_ = s.MemoryApplyStore.RunChunkTx(ctx, supplierID, sessionID, func(tx ApplyTx) error {
			if err := fn(tx); err != nil {
				return err
			}
			return errors.New("simulated spanner aborted")
		})
	}
	return s.MemoryApplyStore.RunChunkTx(ctx, supplierID, sessionID, fn)
}

func TestApplySession_AbortedRetryDoesNotDuplicateCountsOrSkipCustomers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	sup := "sup-m5-aborted-retry"
	repo, memStore, proj := newM5TestEnv(t, sup, now)
	retryStore := &abortedOnceApplyStore{MemoryApplyStore: memStore, retryChunk: 2}

	custRows := make([]map[string]any, 4)
	for i := range custRows {
		custRows[i] = map[string]any{
			"customer_name": fmt.Sprintf("Retry Store %d", i+1),
			"phone":         fmt.Sprintf("+99890555%04d", 100+i),
			"inn":           fmt.Sprintf("300001%03d", i+1),
		}
	}
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-cust-retry", PackCustomers, custRows, map[string]string{
		"customer_name": "customer_name",
		"phone":         "phone",
		"inn":           "inn",
	}, ApplyModeMerge, false, now)

	custRes, err := ApplySession(ctx, repo, retryStore, ApplyOptions{
		SupplierID: sup,
		SessionID:  "sess-cust-retry",
		ChunkSize:  2,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("apply customers with aborted retry: %v", err)
	}
	if custRes.Created != 4 || custRes.Updated != 0 || custRes.Skipped != 0 {
		t.Fatalf("expected 4 created / 0 updated / 0 skipped after aborted retry, got %+v", custRes)
	}
	if len(memStore.Links) != 4 {
		t.Fatalf("expected 4 persisted customer links after aborted retry, got %d", len(memStore.Links))
	}

	retryStore.calls = 0
	retryStore.retryChunk = 2
	prodRows := make([]map[string]any, 4)
	for i := range prodRows {
		prodRows[i] = map[string]any{
			"sku":      fmt.Sprintf("RETRY-SKU-%02d", i+1),
			"name":     fmt.Sprintf("Retry Product %d", i+1),
			"price":    "12000",
			"currency": "UZS",
			"unit":     "PCS",
		}
	}
	stageDryRunAndApprove(t, repo, sup, proj.ProjectID, "sess-prod-retry", PackProductsPrices, prodRows, map[string]string{
		"sku":      "sku",
		"name":     "name",
		"price":    "price",
		"currency": "currency",
		"unit":     "unit",
	}, ApplyModeMerge, false, now)

	prodRes, err := ApplySession(ctx, repo, retryStore, ApplyOptions{
		SupplierID: sup,
		SessionID:  "sess-prod-retry",
		ChunkSize:  2,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("apply products with aborted retry: %v", err)
	}
	if prodRes.Created != 4 || prodRes.Updated != 0 || prodRes.Skipped != 0 {
		t.Fatalf("expected 4 created / 0 updated / 0 skipped products after aborted retry, got %+v", prodRes)
	}
	if len(memStore.Products) != 4 {
		t.Fatalf("expected 4 persisted products after aborted retry, got %d", len(memStore.Products))
	}
}
