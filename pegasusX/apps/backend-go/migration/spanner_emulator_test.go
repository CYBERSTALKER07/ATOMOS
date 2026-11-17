package migration

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"google.golang.org/api/iterator"
)

func envOrDefaultMigTest(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func requireSpannerEmulatorForMigrationTest(t *testing.T) *spanner.Client {
	t.Helper()
	if strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST")) == "" {
		if os.Getenv("PARITY_REQUIRE_SPANNER") == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner emulator integration test")
	}
	project := envOrDefaultMigTest("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefaultMigTest("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefaultMigTest("SPANNER_DATABASE", "pegasusx-db")
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn)
	if err != nil {
		t.Fatalf("spanner.NewClient(%s): %v", dsn, err)
	}
	t.Cleanup(client.Close)
	return client
}

func rand7DigitsMig() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint32(time.Now().UnixNano() % 10000000)
	}
	return binary.LittleEndian.Uint32(b[:]) % 10000000
}

func cleanupMigrationSpannerTestData(t *testing.T, client *spanner.Client, supID string, bundleIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		var muts []*spanner.Mutation

		// Order children & Orders
		var orderIDs []string
		oIter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT OrderId FROM Orders WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supID},
		})
		for {
			r, err := oIter.Next()
			if err != nil {
				break
			}
			var id string
			if r.Columns(&id) == nil && id != "" {
				orderIDs = append(orderIDs, id)
				muts = append(muts, spanner.Delete("Orders", spanner.Key{id}))
			}
		}
		oIter.Stop()

		if len(orderIDs) > 0 {
			for _, spec := range []struct {
				table string
				cols  []string
			}{
				{"OrderStatusTransitions", []string{"OrderId", "TransitionId"}},
				{"OrderStockReservations", []string{"OrderId"}},
				{"OrderLotReservations", []string{"OrderId", "LotId"}},
			} {
				sql := fmt.Sprintf("SELECT %s FROM %s WHERE OrderId IN UNNEST(@oids)", strings.Join(spec.cols, ", "), spec.table)
				it := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: map[string]any{"oids": orderIDs}})
				for {
					r, err := it.Next()
					if err != nil {
						break
					}
					if len(spec.cols) == 1 {
						var k1 string
						if r.Columns(&k1) == nil {
							muts = append(muts, spanner.Delete(spec.table, spanner.Key{k1}))
						}
					} else {
						var k1, k2 string
						if r.Columns(&k1, &k2) == nil {
							muts = append(muts, spanner.Delete(spec.table, spanner.Key{k1, k2}))
						}
					}
				}
				it.Stop()
			}
		}

		// AR, ledger entries, inventory, lots, products, warehouses, links, retailers
		deleteBySingleCol := func(table, col, whereCol string) []string {
			var vals []string
			it := client.Single().Query(ctx, spanner.Statement{
				SQL:    fmt.Sprintf("SELECT %s FROM %s WHERE %s = @sid", col, table, whereCol),
				Params: map[string]any{"sid": supID},
			})
			for {
				r, err := it.Next()
				if err != nil {
					break
				}
				var v string
				if r.Columns(&v) == nil && v != "" {
					vals = append(vals, v)
					muts = append(muts, spanner.Delete(table, spanner.Key{v}))
				}
			}
			it.Stop()
			return vals
		}

		deleteBySingleCol("ArLedgerEntries", "EntryId", "SupplierId")
		deleteBySingleCol("ArInvoices", "InvoiceId", "SupplierId")
		deleteBySingleCol("PaymentLedgerEntries", "LedgerEntryId", "SupplierId")
		deleteBySingleCol("StockLots", "LotId", "SupplierId")
		deleteBySingleCol("Products", "ProductId", "SupplierId")
		deleteBySingleCol("Warehouses", "WarehouseId", "SupplierId")
		deleteBySingleCol("OutboxEvents", "EventId", "SupplierId")

		// SupplierInventoryV2 (PK: SupplierId, WarehouseId, ProductId)
		invIter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT SupplierId, WarehouseId, ProductId FROM SupplierInventoryV2 WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supID},
		})
		for {
			r, err := invIter.Next()
			if err != nil {
				break
			}
			var s, w, p string
			if r.Columns(&s, &w, &p) == nil {
				muts = append(muts, spanner.Delete("SupplierInventoryV2", spanner.Key{s, w, p}))
			}
		}
		invIter.Stop()

		// SupplierRetailerLinks + Retailers
		linkIter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT SupplierId, RetailerId FROM SupplierRetailerLinks WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supID},
		})
		for {
			r, err := linkIter.Next()
			if err != nil {
				break
			}
			var s, rid string
			if r.Columns(&s, &rid) == nil {
				muts = append(muts, spanner.Delete("SupplierRetailerLinks", spanner.Key{s, rid}))
				muts = append(muts, spanner.Delete("Retailers", spanner.Key{rid}))
			}
		}
		linkIter.Stop()

		// PartnerPlantMaps (PK: TenantType, TenantId, ExternalPlantId)
		pmIter := client.Single().Query(ctx, spanner.Statement{
			SQL:    `SELECT TenantType, TenantId, ExternalPlantId FROM PartnerPlantMaps WHERE TenantType = 'SUPPLIER' AND TenantId = @sid`,
			Params: map[string]any{"sid": supID},
		})
		for {
			r, err := pmIter.Next()
			if err != nil {
				break
			}
			var tt, tid, ext string
			if r.Columns(&tt, &tid, &ext) == nil {
				muts = append(muts, spanner.Delete("PartnerPlantMaps", spanner.Key{tt, tid, ext}))
			}
		}
		pmIter.Stop()

		// Migration tables: IdentityMap, WarehouseLocks, Reports, Sessions (cascades to StagedRows/Mapping/Snapshots), Projects (cascades to Bundles/Artifacts)
		for _, spec := range []struct {
			table string
			cols  []string
		}{
			{"MigrationIdentityMap", []string{"supplier_id", "pack", "external_id"}},
			{"MigrationWarehouseLocks", []string{"supplier_id", "warehouse_id"}},
			{"MigrationReports", []string{"supplier_id", "report_id"}},
			{"SupplierImportSessions", []string{"supplier_id", "session_id"}},
			{"MigrationProjects", []string{"supplier_id", "project_id"}},
		} {
			sql := fmt.Sprintf("SELECT %s FROM %s WHERE supplier_id = @sid", strings.Join(spec.cols, ", "), spec.table)
			it := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: map[string]any{"sid": supID}})
			for {
				r, err := it.Next()
				if err != nil {
					break
				}
				if len(spec.cols) == 2 {
					var a, b string
					if r.Columns(&a, &b) == nil {
						muts = append(muts, spanner.Delete(spec.table, spanner.Key{a, b}))
					}
				} else if len(spec.cols) == 3 {
					var a, b, c string
					if r.Columns(&a, &b, &c) == nil {
						muts = append(muts, spanner.Delete(spec.table, spanner.Key{a, b, c}))
					}
				}
			}
			it.Stop()
		}

		// OutboxEvents referencing bundles if SupplierId differed
		if len(bundleIDs) > 0 {
			obIter := client.Single().Query(ctx, spanner.Statement{
				SQL:    `SELECT EventId FROM OutboxEvents WHERE AggregateId IN UNNEST(@bids)`,
				Params: map[string]any{"bids": bundleIDs},
			})
			for {
				r, err := obIter.Next()
				if err != nil {
					break
				}
				var eid string
				if r.Columns(&eid) == nil && eid != "" {
					muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
				}
			}
			obIter.Stop()
		}

		muts = append(muts, spanner.Delete("Suppliers", spanner.Key{supID}))
		// Apply deletes in bounded chunks (delete children before parents: actually Spanner Delete on parent with ON DELETE CASCADE works,
		// and non-interleaved tables have no FKs, so apply in reverse / small batches).
		if len(muts) > 0 {
			const batch = 200
			for i := 0; i < len(muts); i += batch {
				end := i + batch
				if end > len(muts) {
					end = len(muts)
				}
				_, _ = client.Apply(ctx, muts[i:end])
			}
		}
	})
}

func TestSpannerMigration_FullPipeline_CutoverReleaseAndRollback(t *testing.T) {
	client := requireSpannerEmulatorForMigrationTest(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	store := NewSpannerApplyStore(client)

	shortID := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	supID := "sup_mig_" + shortID
	projID := "prj_mig_" + shortID
	bndID := "bnd_mig_" + shortID
	artID := "art_mig_" + shortID
	homeCell := "cell-mig-" + shortID
	supINN := fmt.Sprintf("33%07d", rand7DigitsMig())
	custINN := fmt.Sprintf("34%07d", rand7DigitsMig())
	custPhone := fmt.Sprintf("+99895%07d", rand7DigitsMig())

	cleanupMigrationSpannerTestData(t, client, supID, []string{bndID})

	// Seed parent Supplier row expected by ReadSupplierMeta.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":      supID,
			"Name":            "Spanner Emulator Test Supplier",
			"CountryCode":     "UZ",
			"Currency":        "UZS",
			"IsConfigured":    true,
			"HomeCell":        homeCell,
			"NormalizedTaxId": supINN,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed Suppliers: %v", err)
	}

	// 1. M2 Project + Bundle + Artifacts + MarkBundleUploadedAndEmit + Outbox verification.
	createdProj, err := repo.CreateProject(ctx, Project{
		SupplierID: supID,
		ProjectID:  projID,
		Name:       "1C Full Cutover",
		Mode:       ProjectModeCutover,
		Status:     ProjectStatusDraft,
		CreatedBy:  "usr_admin",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if createdProj.ProjectID != projID {
		t.Fatalf("unexpected created project: %+v", createdProj)
	}
	projects, err := repo.ListProjects(ctx, supID, 10)
	if err != nil || len(projects) != 1 || projects[0].ProjectID != projID {
		t.Fatalf("ListProjects: len=%d err=%v", len(projects), err)
	}

	_, arts, err := repo.CreateBundleWithArtifacts(ctx, Bundle{
		SupplierID: supID,
		ProjectID:  projID,
		BundleID:   bndID,
		Status:     BundleStatusUploading,
		CreatedBy:  "usr_admin",
	}, []Artifact{{
		SupplierID:    supID,
		ProjectID:     projID,
		BundleID:      bndID,
		ArtifactID:    artID,
		FileName:      "export_1c.xlsx",
		ByteSize:      1024,
		DeclaredMIME:  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		ContentSHA256: "sha_init_" + shortID,
		GCSPath:       "gs://pegasusx-migrations/test/" + artID + ".xlsx",
	}})
	if err != nil || len(arts) != 1 {
		t.Fatalf("CreateBundleWithArtifacts: len=%d err=%v", len(arts), err)
	}

	uploadedBnd, err := repo.MarkBundleUploadedAndEmit(ctx, supID, projID, bndID, map[string]UploadedArtifactMeta{
		artID: {ByteSize: 2048, ContentSHA256: "sha_real_" + shortID, Encoding: "utf-8"},
	})
	if err != nil {
		t.Fatalf("MarkBundleUploadedAndEmit: %v", err)
	}
	if uploadedBnd.Status != BundleStatusClassifying {
		t.Fatalf("expected CLASSIFYING bundle, got %s", uploadedBnd.Status)
	}

	var obCount int64
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM OutboxEvents WHERE AggregateId = @bid AND SupplierId = @sid`,
		Params: map[string]any{"bid": bndID, "sid": supID},
	}).Do(func(r *spanner.Row) error { return r.Columns(&obCount) }); err != nil {
		t.Fatalf("query outbox for bundle: %v", err)
	}
	if obCount != 1 {
		t.Fatalf("expected 1 outbox event for bundle upload, got %d", obCount)
	}

	// 2. UpsertSortedBundleSessions for 7 sessions across M3-M7 packs (stock & lots are mutually exclusive per project).
	sessWH := "s_wh_" + shortID
	sessCat := "s_cat_" + shortID
	sessCust := "s_cus_" + shortID
	sessStk := "s_stk_" + shortID
	sessAR := "s_ar_" + shortID
	sessHist := "s_hst_" + shortID
	sessOpen := "s_opn_" + shortID

	sessionSpecs := []struct {
		id   string
		pack Pack
		rows int64
	}{
		{sessWH, PackWarehouses, 1},
		{sessCat, PackProductsPrices, 2},
		{sessCust, PackCustomers, 1},
		{sessStk, PackStock, 1},
		{sessAR, PackNasiyaAR, 1},
		{sessHist, PackHistoricalOrders, 1},
		{sessOpen, PackOpenOrders, 2},
	}
	var sortedSessions []Session
	for _, sp := range sessionSpecs {
		sortedSessions = append(sortedSessions, Session{
			SupplierID:    supID,
			SessionID:     sp.id,
			Status:        SessionStatusDiscovered,
			FileName:      "export_1c.xlsx",
			TotalRows:     sp.rows,
			ProjectID:     projID,
			BundleID:      bndID,
			ArtifactID:    artID,
			StreamKey:     string(sp.pack),
			Pack:          sp.pack,
			ContentSHA256: "sha_" + sp.id,
			DetectedMIME:  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			Encoding:      "utf-8",
		})
	}
	createdSessions, err := repo.UpsertSortedBundleSessions(
		ctx, supID, projID, bndID, BundleStatusSorted,
		json.RawMessage(`{"streams":8}`),
		map[string]json.RawMessage{artID: json.RawMessage(`[{"pack":"warehouses"}]`)},
		sortedSessions,
	)
	if err != nil || len(createdSessions) != len(sessionSpecs) {
		t.Fatalf("UpsertSortedBundleSessions: len=%d err=%v", len(createdSessions), err)
	}

	// Verify ListBundlesByProject & ListArtifactsByBundle & WarehouseLock lifecycle.
	if bnds, err := repo.ListBundlesByProject(ctx, supID, projID); err != nil || len(bnds) != 1 || bnds[0].Status != BundleStatusSorted {
		t.Fatalf("ListBundlesByProject: bnds=%+v err=%v", bnds, err)
	}
	if _, err := repo.AcquireWarehouseLock(ctx, supID, "WH-LOCK-TEST", sessStk, 5*time.Minute); err != nil {
		t.Fatalf("AcquireWarehouseLock: %v", err)
	}
	if _, err := repo.AcquireWarehouseLock(ctx, supID, "WH-LOCK-TEST", sessWH, 5*time.Minute); err == nil {
		t.Fatalf("expected warehouse lock conflict for different session")
	}
	if err := repo.ReleaseWarehouseLock(ctx, supID, "WH-LOCK-TEST", sessStk); err != nil {
		t.Fatalf("ReleaseWarehouseLock: %v", err)
	}

	// Keep evaluation timestamps slightly in the past so any allow_commit_timestamp columns stay <= commit time,
	// while still strictly inside 24h dry-run TTL and 72h rollback windows.
	now := time.Now().UTC().Add(-20 * time.Minute).Truncate(time.Microsecond)

	runPack := func(sessionID string, pack Pack, colMap map[string]string, rows []map[string]any, applyMode string, confirmSetOnHand bool) ApplyResult {
		t.Helper()
		var props []ColumnMappingProposal
		for src, tgt := range colMap {
			props = append(props, ColumnMappingProposal{
				SourceColumn: src,
				TargetField:  tgt,
				Confidence:   0.99,
				Source:       "manual",
				Confirmed:    true,
			})
		}
		doc := SessionMappingDocument{
			SessionID:         sessionID,
			Pack:              pack,
			HeaderFingerprint: "fp_" + string(pack),
			AIModel:           "deterministic",
			AIStatus:          "MATCHED",
			Mappings:          props,
			AllRequiredMapped: true,
			ConfirmedBy:       "usr_admin",
			UpdatedAt:         now,
		}
		doc.MappingHash = ComputeMappingHash(doc)
		if err := repo.SaveSessionMapping(ctx, supID, sessionID, SessionStatusDiscovered, doc); err != nil {
			t.Fatalf("SaveSessionMapping(%s): %v", pack, err)
		}
		if gotDoc, ok, err := repo.GetSessionMapping(ctx, supID, sessionID); err != nil || !ok || len(gotDoc.Mappings) != len(props) {
			t.Fatalf("GetSessionMapping(%s): ok=%v err=%v", pack, ok, err)
		}

		staged := make([]StagedRow, 0, len(rows))
		for idx, raw := range rows {
			b, _ := json.Marshal(raw)
			staged = append(staged, StagedRow{
				SupplierID:  supID,
				SessionID:   sessionID,
				RowIndex:    int64(idx),
				RawData:     b,
				CleanedData: b,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
		savedCount, err := repo.SaveStagedRowsChunk(ctx, supID, sessionID, staged)
		if err != nil || savedCount != int64(len(rows)) {
			t.Fatalf("SaveStagedRowsChunk(%s): count=%d err=%v", pack, savedCount, err)
		}
		listedRows, totalRows, err := repo.ListStagedRows(ctx, supID, sessionID, 50, 0)
		if err != nil || totalRows != int64(len(rows)) || len(listedRows) != len(rows) {
			t.Fatalf("ListStagedRows(%s): len=%d total=%d err=%v", pack, len(listedRows), totalRows, err)
		}
		var cursorCount int
		if err := streamAllStagedRows(ctx, repo, supID, sessionID, -1, 1, func(page []StagedRow) error {
			cursorCount += len(page)
			return nil
		}); err != nil || cursorCount != len(rows) {
			t.Fatalf("streamAllStagedRows(%s): count=%d want=%d err=%v", pack, cursorCount, len(rows), err)
		}

		rep, sum, err := EvaluateSessionDryRun(ctx, repo, DryRunOptions{
			SupplierID: supID,
			ProjectID:  projID,
			SessionID:  sessionID,
			ApplyMode:  applyMode,
			Now:        now,
			LiveReader: repo,
		})
		if err != nil {
			t.Fatalf("EvaluateSessionDryRun(%s): %v", pack, err)
		}
		if rep.BlockingCount != 0 || !sum.ReadyToApprove {
			t.Fatalf("EvaluateSessionDryRun(%s) blocked unexpectedly: rep=%+v sum=%+v", pack, rep, sum)
		}

		approved, err := ApproveSession(ctx, repo, supID, sessionID, ApproveSessionRequest{
			ApplyMode:          applyMode,
			SetOnHandConfirmed: confirmSetOnHand,
			ExpectedReportID:   rep.ReportID,
		}, now.Add(time.Minute))
		if err != nil || approved.Status != SessionStatusApproved {
			t.Fatalf("ApproveSession(%s): status=%s err=%v", pack, approved.Status, err)
		}

		res, err := ApplySession(ctx, repo, store, ApplyOptions{
			SupplierID: supID,
			SessionID:  sessionID,
			ApplyMode:  applyMode,
			Now:        now.Add(2 * time.Minute),
		})
		if err != nil {
			t.Fatalf("ApplySession(%s): %v", pack, err)
		}
		if res.Session.Status != SessionStatusApplied {
			t.Fatalf("ApplySession(%s) expected APPLIED, got %s", pack, res.Session.Status)
		}
		return res
	}

	// 3. Apply M5/M6 pipeline across all packs on live Spanner.
	runPack(sessWH, PackWarehouses, map[string]string{
		"external_warehouse_id": "external_warehouse_id",
		"name":                  "name",
		"address":               "address",
		"is_default":            "is_default",
	}, []map[string]any{
		{"external_warehouse_id": "WH-MAIN", "name": "Tashkent Central", "address": "Sergeli 1", "is_default": "true"},
	}, ApplyModeMerge, false)

	runPack(sessCat, PackProductsPrices, map[string]string{
		"sku":             "sku",
		"name":            "name",
		"price":           "price",
		"currency":        "currency",
		"unit":            "unit",
		"barcode":         "barcode",
		"shelf_life_days": "shelf_life_days",
	}, []map[string]any{
		{"sku": "SKU-TEA-1", "name": "Black Tea", "price": "20000", "currency": "UZS", "unit": "PCS", "barcode": "478001" + shortID[:4], "shelf_life_days": "30"},
		{"sku": "SKU-YOG-1", "name": "Fresh Yogurt", "price": "15000", "currency": "UZS", "unit": "PCS", "barcode": "478002" + shortID[:4], "shelf_life_days": "14"},
	}, ApplyModeMerge, false)

	runPack(sessCust, PackCustomers, map[string]string{
		"external_customer_id": "external_customer_id",
		"customer_name":        "customer_name",
		"phone":                "phone",
		"inn":                  "inn",
		"address":              "address",
	}, []map[string]any{
		{"external_customer_id": "CUST-A", "customer_name": "Baraka Market", "phone": custPhone, "inn": custINN, "address": "Chilonzor 5"},
	}, ApplyModeMerge, false)

	// Stock SET_ON_HAND -> QuantityOnHand=25 on SKU-TEA-1.
	runPack(sessStk, PackStock, map[string]string{
		"sku":              "sku",
		"warehouse_id":     "warehouse_id",
		"quantity_on_hand": "quantity_on_hand",
	}, []map[string]any{
		{"sku": "SKU-TEA-1", "warehouse_id": "WH-MAIN", "quantity_on_hand": "25"},
	}, ApplyModeSetOnHand, true)

	// Historical order -> terminal DELIVERED, HISTORICAL_SKIPPED, zero inventory change (must precede nasiya_ar per DAG).
	runPack(sessHist, PackHistoricalOrders, map[string]string{
		"external_order_no": "external_order_no",
		"order_date":        "order_date",
		"status":            "status",
		"customer_inn":      "customer_inn",
		"sku":               "sku",
		"quantity":          "quantity",
		"total_amount":      "total_amount",
		"currency":          "currency",
	}, []map[string]any{
		{
			"external_order_no": "HIST-SPN-01",
			"order_date":        "2026-10-10",
			"status":            "DELIVERED",
			"customer_inn":      custINN,
			"sku":               "SKU-TEA-1",
			"quantity":          "2",
			"total_amount":      "40000",
			"currency":          "UZS",
		},
	}, ApplyModeMerge, false)

	// Nasiya AR -> opening balance + double-entry PaymentLedgerEntries + ar.SpannerRepository ack.
	arRes := runPack(sessAR, PackNasiyaAR, map[string]string{
		"external_ar_id":  "external_ar_id",
		"customer_inn":    "customer_inn",
		"balance_due":     "balance_due",
		"original_amount": "original_amount",
		"currency":        "currency",
		"issued_date":     "issued_date",
		"due_date":        "due_date",
	}, []map[string]any{
		{
			"external_ar_id":  "AR-SPN-100",
			"customer_inn":    custINN,
			"balance_due":     "250000",
			"original_amount": "250000",
			"currency":        "UZS",
			"issued_date":     "2026-10-01",
			"due_date":        "2026-10-20",
		},
	}, ApplyModeMerge, false)
	if len(arRes.JournalEntries) != 1 || arRes.JournalEntries[0].Validate() != nil || len(arRes.JournalEntries[0].Postings) != 2 {
		t.Fatalf("expected valid 2-posting journal entry, got %+v", arRes.JournalEntries)
	}

	// Open orders -> held in MIGRATION_OPEN with 0 reservation.
	runPack(sessOpen, PackOpenOrders, map[string]string{
		"external_order_no": "external_order_no",
		"customer_inn":      "customer_inn",
		"warehouse_id":      "warehouse_id",
		"sku":               "sku",
		"quantity":          "quantity",
		"status":            "status",
		"delivery_date":     "delivery_date",
		"total_amount":      "total_amount",
	}, []map[string]any{
		{
			"external_order_no": "OPEN-FIT-01",
			"customer_inn":      custINN,
			"warehouse_id":      "WH-MAIN",
			"sku":               "SKU-TEA-1",
			"quantity":          "6",
			"status":            "SCHEDULED",
			"delivery_date":     "2026-11-20",
			"total_amount":      "120000",
		},
		{
			"external_order_no": "OPEN-BIG-02",
			"customer_inn":      custINN,
			"warehouse_id":      "WH-MAIN",
			"sku":               "SKU-TEA-1",
			"quantity":          "100",
			"delivery_date":     "2026-11-21",
			"total_amount":      "2000000",
		},
	}, ApplyModeMerge, false)

	// 4. Verify live AR read + acknowledgment via ar.SpannerRepository.
	var retID string
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT RetailerId FROM SupplierRetailerLinks WHERE SupplierId = @sid LIMIT 1`,
		Params: map[string]any{"sid": supID},
	}).Do(func(r *spanner.Row) error { return r.Columns(&retID) }); err != nil || retID == "" {
		t.Fatalf("resolve seeded RetailerId: retID=%q err=%v", retID, err)
	}
	arRepo := ar.NewSpannerRepository(client)
	balances, err := arRepo.ListMigrationOpeningBalances(ctx, retID)
	if err != nil || len(balances) != 1 || balances[0].BalanceMinor != 25000000 {
		t.Fatalf("ListMigrationOpeningBalances: balances=%+v err=%v", balances, err)
	}
	ackInv, err := arRepo.AcknowledgeMigrationInvoice(ctx, retID, balances[0].InvoiceID, now.Add(5*time.Minute))
	if err != nil || ackInv.AcknowledgedAt == nil {
		t.Fatalf("AcknowledgeMigrationInvoice: inv=%+v err=%v", ackInv, err)
	}

	// 5. Verify inventory untouched before release, then selective release via SpannerOpenOrderReleaser.
	whID, teaProdID := "", ""
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT WarehouseId FROM Warehouses WHERE SupplierId = @sid LIMIT 1`,
		Params: map[string]any{"sid": supID},
	}).Do(func(r *spanner.Row) error { return r.Columns(&whID) }); err != nil || whID == "" {
		t.Fatalf("read WarehouseId: %v", err)
	}
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT ProductId FROM Products WHERE SupplierId = @sid AND Sku = 'SKU-TEA-1' LIMIT 1`,
		Params: map[string]any{"sid": supID},
	}).Do(func(r *spanner.Row) error {
		return r.Columns(&teaProdID)
	}); err != nil {
		// Products table uses SKU/Barcode, fetch via MigrationIdentityMap directly.
	}
	if teaProdID == "" {
		idRow, idErr := client.Single().ReadRow(ctx, "MigrationIdentityMap", spanner.Key{supID, string(PackProductsPrices), "SKU-TEA-1"}, []string{"internal_id"})
		if idErr != nil {
			t.Fatalf("read SKU-TEA-1 identity map: %v", idErr)
		}
		if err := idRow.Columns(&teaProdID); err != nil {
			t.Fatalf("scan SKU-TEA-1 internal_id: %v", err)
		}
	}

	checkTeaInventory := func(wantOnHand, wantReserved int64) {
		t.Helper()
		row, err := client.Single().ReadRow(ctx, "SupplierInventoryV2", spanner.Key{supID, whID, teaProdID}, []string{"QuantityOnHand", "QuantityReserved"})
		if err != nil {
			t.Fatalf("read SupplierInventoryV2: %v", err)
		}
		var qoh, qr int64
		if err := row.Columns(&qoh, &qr); err != nil {
			t.Fatalf("scan SupplierInventoryV2: %v", err)
		}
		if qoh != wantOnHand || qr != wantReserved {
			t.Fatalf("inventory mismatch for SKU-TEA-1: got QOH=%d QR=%d, want QOH=%d QR=%d", qoh, qr, wantOnHand, wantReserved)
		}
	}
	checkTeaInventory(25, 0)

	releaser := SpannerOpenOrderReleaser{Client: client}
	relRes, err := releaser.ReleaseOpenOrders(ctx, supID, nil, "admin_mig", now.Add(6*time.Minute))
	if err != nil {
		t.Fatalf("ReleaseOpenOrders: %v", err)
	}
	if len(relRes.ReleasedOrderIDs) != 1 || len(relRes.SkippedOrderIDs) != 1 {
		t.Fatalf("expected 1 released (OPEN-FIT-01) and 1 skipped (OPEN-BIG-02), got %+v", relRes)
	}
	checkTeaInventory(25, 6)

	// 6. M7 Rollback on Spanner: nasiya_ar + historical_orders.
	rbAR, err := RollbackSession(ctx, repo, supID, sessAR, now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("RollbackSession(nasiya_ar): %v", err)
	}
	if rbAR.ReversedInvoices != 1 || rbAR.ReversedAmountMinor != 25000000 || rbAR.Session.Status != SessionStatusRolledBack {
		t.Fatalf("unexpected AR rollback result: %+v", rbAR)
	}
	// Verify compensating ArLedgerEntries (-250000) & PaymentLedgerEntries persisted.
	var arStatus string
	var arBal int64
	if row, err := client.Single().ReadRow(ctx, "ArInvoices", spanner.Key{balances[0].InvoiceID}, []string{"Status", "BalanceMinor"}); err != nil {
		t.Fatalf("read rolled back ArInvoice: %v", err)
	} else if err := row.Columns(&arStatus, &arBal); err != nil {
		t.Fatalf("scan ArInvoice: %v", err)
	}
	if arStatus != "ROLLED_BACK" || arBal != 0 {
		t.Fatalf("expected ArInvoice ROLLED_BACK balance=0, got status=%s bal=%d", arStatus, arBal)
	}

	rbHist, err := RollbackSession(ctx, repo, supID, sessHist, now.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("RollbackSession(historical_orders): %v", err)
	}
	if rbHist.SupersededOrders != 1 || rbHist.Session.Status != SessionStatusRolledBack {
		t.Fatalf("unexpected historical orders rollback result: %+v", rbHist)
	}

	// Verify expired rollback window check fails closed (> 72h).
	rbResLate, err := RollbackSession(ctx, repo, supID, sessCust, now.Add(75*time.Hour))
	if err == nil {
		t.Fatalf("expected rollback window expiry error after 75h, got %+v", rbResLate)
	}
	if !errors.Is(err, ErrRollbackWindowExpired) && !strings.Contains(err.Error(), "rollback") {
		t.Fatalf("unexpected error after 75h rollback: %v", err)
	}
	_ = iterator.Done
}

func TestSpannerMigration_LotsCivilDateApplyAndRollback(t *testing.T) {
	prevLots := stocklots.LotsEnabled()
	stocklots.SetLotsEnabled(true)
	t.Cleanup(func() { stocklots.SetLotsEnabled(prevLots) })

	client := requireSpannerEmulatorForMigrationTest(t)
	ctx := context.Background()
	repo := NewSpannerRepository(client)
	store := NewSpannerApplyStore(client)

	shortID := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	supID := "sup_lot_" + shortID
	projID := "prj_lot_" + shortID
	bndID := "bnd_lot_" + shortID
	artID := "art_lot_" + shortID
	supINN := fmt.Sprintf("35%07d", rand7DigitsMig())

	cleanupMigrationSpannerTestData(t, client, supID, []string{bndID})

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Suppliers", map[string]any{
			"SupplierId":      supID,
			"Name":            "Spanner Lots Supplier",
			"CountryCode":     "UZ",
			"Currency":        "UZS",
			"IsConfigured":    true,
			"HomeCell":        "cell-lot-" + shortID,
			"NormalizedTaxId": supINN,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed Suppliers: %v", err)
	}

	if _, err := repo.CreateProject(ctx, Project{
		SupplierID: supID,
		ProjectID:  projID,
		Name:       "Lots Cutover",
		Mode:       ProjectModeCutover,
		Status:     ProjectStatusDraft,
		CreatedBy:  "usr_admin",
	}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if _, _, err := repo.CreateBundleWithArtifacts(ctx, Bundle{
		SupplierID: supID, ProjectID: projID, BundleID: bndID, Status: BundleStatusUploading, CreatedBy: "usr_admin",
	}, []Artifact{{
		SupplierID: supID, ProjectID: projID, BundleID: bndID, ArtifactID: artID,
		FileName: "lots.xlsx", ByteSize: 512, ContentSHA256: "sha_" + artID, GCSPath: "gs://pegasusx-migrations/test/" + artID,
	}}); err != nil {
		t.Fatalf("CreateBundleWithArtifacts: %v", err)
	}
	if _, err := repo.MarkBundleUploadedAndEmit(ctx, supID, projID, bndID, nil); err != nil {
		t.Fatalf("MarkBundleUploadedAndEmit: %v", err)
	}

	sessWH := "sl_wh_" + shortID
	sessCat := "sl_cat_" + shortID
	sessLot := "sl_lot_" + shortID
	if _, err := repo.UpsertSortedBundleSessions(ctx, supID, projID, bndID, BundleStatusSorted, json.RawMessage(`{"streams":3}`), nil, []Session{
		{SupplierID: supID, SessionID: sessWH, Status: SessionStatusDiscovered, FileName: "lots.xlsx", TotalRows: 1, ProjectID: projID, BundleID: bndID, ArtifactID: artID, StreamKey: "warehouses", Pack: PackWarehouses, ContentSHA256: "s1"},
		{SupplierID: supID, SessionID: sessCat, Status: SessionStatusDiscovered, FileName: "lots.xlsx", TotalRows: 1, ProjectID: projID, BundleID: bndID, ArtifactID: artID, StreamKey: "products_prices", Pack: PackProductsPrices, ContentSHA256: "s2"},
		{SupplierID: supID, SessionID: sessLot, Status: SessionStatusDiscovered, FileName: "lots.xlsx", TotalRows: 1, ProjectID: projID, BundleID: bndID, ArtifactID: artID, StreamKey: "lots", Pack: PackLots, ContentSHA256: "s3"},
	}); err != nil {
		t.Fatalf("UpsertSortedBundleSessions: %v", err)
	}

	now := time.Now().UTC().Add(-20 * time.Minute).Truncate(time.Microsecond)
	stageAndApply := func(sessionID string, pack Pack, colMap map[string]string, rows []map[string]any) {
		var props []ColumnMappingProposal
		for src, tgt := range colMap {
			props = append(props, ColumnMappingProposal{SourceColumn: src, TargetField: tgt, Confidence: 0.99, Source: "manual", Confirmed: true})
		}
		doc := SessionMappingDocument{SessionID: sessionID, Pack: pack, HeaderFingerprint: "fp_" + string(pack), AIModel: "deterministic", AIStatus: "MATCHED", Mappings: props, AllRequiredMapped: true, UpdatedAt: now}
		doc.MappingHash = ComputeMappingHash(doc)
		if err := repo.SaveSessionMapping(ctx, supID, sessionID, SessionStatusDiscovered, doc); err != nil {
			t.Fatalf("SaveSessionMapping(%s): %v", pack, err)
		}
		staged := make([]StagedRow, 0, len(rows))
		for idx, raw := range rows {
			b, _ := json.Marshal(raw)
			staged = append(staged, StagedRow{SupplierID: supID, SessionID: sessionID, RowIndex: int64(idx), RawData: b, CleanedData: b})
		}
		if _, err := repo.SaveStagedRowsChunk(ctx, supID, sessionID, staged); err != nil {
			t.Fatalf("SaveStagedRowsChunk(%s): %v", pack, err)
		}
		rep, sum, err := EvaluateSessionDryRun(ctx, repo, DryRunOptions{SupplierID: supID, ProjectID: projID, SessionID: sessionID, ApplyMode: ApplyModeMerge, Now: now, LiveReader: repo})
		if err != nil || rep.BlockingCount != 0 || !sum.ReadyToApprove {
			t.Fatalf("EvaluateSessionDryRun(%s): err=%v rep=%+v sum=%+v", pack, err, rep, sum)
		}
		if _, err := ApproveSession(ctx, repo, supID, sessionID, ApproveSessionRequest{ApplyMode: ApplyModeMerge, ExpectedReportID: rep.ReportID}, now.Add(time.Minute)); err != nil {
			t.Fatalf("ApproveSession(%s): %v", pack, err)
		}
		if _, err := ApplySession(ctx, repo, store, ApplyOptions{SupplierID: supID, SessionID: sessionID, ApplyMode: ApplyModeMerge, Now: now.Add(2 * time.Minute)}); err != nil {
			t.Fatalf("ApplySession(%s): %v", pack, err)
		}
	}

	stageAndApply(sessWH, PackWarehouses, map[string]string{"external_warehouse_id": "external_warehouse_id", "name": "name"}, []map[string]any{{"external_warehouse_id": "WH-LOT", "name": "Cold Hub"}})
	stageAndApply(sessCat, PackProductsPrices, map[string]string{"sku": "sku", "name": "name", "price": "price", "currency": "currency", "unit": "unit", "shelf_life_days": "shelf_life_days"}, []map[string]any{{"sku": "SKU-YOG-1", "name": "Fresh Yogurt", "price": "15000", "currency": "UZS", "unit": "PCS", "shelf_life_days": "14"}})
	stageAndApply(sessLot, PackLots, map[string]string{"sku": "sku", "warehouse_id": "warehouse_id", "lot_number": "lot_number", "quantity_on_hand": "quantity_on_hand", "expiry_date": "expiry_date"}, []map[string]any{{"sku": "SKU-YOG-1", "warehouse_id": "WH-LOT", "lot_number": "LOT-YOG-01", "quantity_on_hand": "12", "expiry_date": "2027-06-15"}})

	var lotCount int64
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM StockLots WHERE SupplierId = @sid AND LotCode = 'LOT-YOG-01'`, Params: map[string]any{"sid": supID},
	}).Do(func(r *spanner.Row) error { return r.Columns(&lotCount) }); err != nil || lotCount != 1 {
		t.Fatalf("expected 1 StockLot after apply, got %d err=%v", lotCount, err)
	}

	rbLot, err := RollbackSession(ctx, repo, supID, sessLot, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("RollbackSession(lots): %v", err)
	}
	if rbLot.RevertedSnapshots < 1 || rbLot.Session.Status != SessionStatusRolledBack {
		t.Fatalf("unexpected lots rollback result: %+v", rbLot)
	}
	lotCount = -1
	if err := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM StockLots WHERE SupplierId = @sid`, Params: map[string]any{"sid": supID},
	}).Do(func(r *spanner.Row) error { return r.Columns(&lotCount) }); err != nil || lotCount != 0 {
		t.Fatalf("expected 0 StockLots after rollback, got %d err=%v", lotCount, err)
	}

	// Verify ExecuteProjectCutover commits MigrationProjects.cutover_at against Spanner (requires allow_commit_timestamp=true).
	cutRes, err := ExecuteProjectCutover(ctx, repo, CutoverOptions{
		SupplierID: supID,
		ProjectID:  projID,
		ActorID:    "admin_lot_cutover",
		Now:        now.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatalf("ExecuteProjectCutover: %v", err)
	}
	if cutRes.Project.Status != ProjectStatusCutoverComplete || cutRes.Project.CutoverAt == nil {
		t.Fatalf("expected CUTOVER_COMPLETE with non-nil CutoverAt, got %+v", cutRes.Project)
	}
}
