package exportpacks

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type spyAudit struct {
	mu     sync.Mutex
	events []AuditRecordInput
}

func (s *spyAudit) RecordEvent(_ context.Context, ev AuditRecordInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *spyAudit) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev AuditRecordInput) error {
	return s.RecordEvent(context.Background(), ev)
}

func samplePackDatasets(eid string, baseTime time.Time) *PackDatasets {
	exp := baseTime.Add(30 * 24 * time.Hour)
	return &PackDatasets{
		GlAccounts: []GlAccountRow{{
			EntityID: eid, AccountCode: "1010", Class: "ASSET", Name: "Cash at Bank", Currency: "UZS", IsControl: true, CountryPack: "UZ", CreatedAt: baseTime, UpdatedAt: baseTime,
		}},
		JournalEntries: []JournalEntryRow{{
			EntityID: eid, EntryID: "je-1", PeriodID: "2026-10", SourceType: "ORDER", SourceID: "ord-1", Memo: "Sale UZS", Currency: "UZS", CountryPack: "UZ", PrevHash: "0000", EntryHash: "abcd", PostedBy: "acct-1", OccurredAt: baseTime, PostedAt: baseTime,
		}},
		JournalPostings: []JournalPostingRow{{
			EntityID: eid, EntryID: "je-1", Line: 1, AccountCode: "1010", Direction: "DEBIT", AmountMinor: 250000, Currency: "UZS", FunctionalAmountMinor: 250000, Description: "Receipt",
		}},
		ArInvoices: []ArInvoiceRow{{
			InvoiceID: "inv-1", SupplierID: eid, RetailerID: "ret-1", OrderID: "ord-1", Status: "OPEN", PrincipalMinor: 250000, BalanceMinor: 100000, Currency: "UZS", CreditLeaveAt: baseTime, DueAt: exp, TermsDays: 30, CreatedAt: baseTime, UpdatedAt: baseTime,
		}},
		InventoryCostLayers: []InventoryCostLayerRow{{
			EntityID: eid, WarehouseID: "wh-1", ProductID: "sku-1", LayerID: "lay-1", QtyRemaining: 10, OriginalQty: 20, UnitCostMinor: 5000, Currency: "UZS", ReceivedAt: baseTime, ExpiryDate: "2027-01-01", CreatedAt: baseTime, UpdatedAt: baseTime,
		}},
		InventoryMovements: []InventoryMovementRow{{
			EntityID: eid, MovementID: "mov-1", MovementType: "ISSUE", ReferenceType: "ORDER", ReferenceID: "ord-1", WarehouseID: "wh-1", ProductID: "sku-1", LayerID: "lay-1", QtyDelta: -10, UnitCostMinor: 5000, TotalCostMinor: -50000, Currency: "UZS", GlEntryID: "je-1", CreatedAt: baseTime,
		}},
		EsfDocuments: []EsfDocumentRow{{
			EntityID: eid, EsfID: "esf-1", Direction: "OUTBOUND", OrderID: "ord-1", CounterpartyTIN: "305123456", CounterpartyName: "Korzinka LLC", TotalMinor: 250000, VatMinor: 26786, VatRateBps: 1200, Status: "ACCEPTED", Operator: "DIDOX", PeriodID: "2026-10", IssuedAt: &baseTime, AcceptedAt: &baseTime, CreatedAt: baseTime, UpdatedAt: baseTime,
		}},
		CashierBlindCountSessions: []CashierBlindCountRow{{
			SessionID: "cb-1", SupplierID: eid, DepotID: "dep-1", DriverID: "drv-1", RouteID: "rt-1", BagSealBarcode: "SEAL-1", AttemptNumber: 1, CountedCashMinor: 100000, ExpectedCashMinor: 100000, VarianceMinor: 0, Status: "CLEARED", CreatedAt: baseTime,
		}},
		DepotVaultLedgerEntries: []DepotVaultLedgerRow{{
			VaultEntryID: "dv-1", DepotID: "dep-1", SupplierID: eid, SessionID: "cb-1", EntryType: "DEPOSIT", AmountMinor: 100000, RunningBalanceMinor: 500000, CreatedAt: baseTime,
		}},
		PaymentLedgerEntries: []PaymentLedgerRow{{
			LedgerEntryID: "pay-1", OrderID: "ord-1", SupplierID: eid, RetailerID: "ret-1", Gateway: "CLICK", EntryType: "SETTLEMENT", AmountMinor: 150000, Currency: "UZS", Source: "PSP", OccurredAt: baseTime, CreatedAt: baseTime,
		}},
		ApprovalRequests: []ApprovalRequestRow{{
			TenantID: eid, RequestID: "apr-1", PolicyID: "pol-1", PolicyVersion: 1, ObjectType: "JournalEntry", ObjectID: "je-1", Action: "journal.approve", AmountMinor: 250000, RequestedBy: "acct-1", ActorType: "HUMAN", PayloadHash: "hash-1", Status: "APPROVED", CurrentStep: 1, RequiredApprovers: 1, CreatedAt: baseTime, UpdatedAt: baseTime,
		}},
		ApprovalDecisions: []ApprovalDecisionRow{{
			TenantID: eid, RequestID: "apr-1", DecisionID: "dec-1", StepIndex: 0, ApproverID: "fm-1", Decision: "APPROVE", MfaVerified: true, CreatedAt: baseTime,
		}},
		AuditChainAnchors: []AuditChainAnchorRow{{
			TenantID: eid, DayBucket: "2026-10-15", HeadEventID: "ev-1", EventCount: 42, RootHash: "root-hash-1", AnchoredBy: "system", AnchoredAt: baseTime,
		}},
		ReconciliationRuns: []ReconciliationRunRow{{
			EntityID: eid, RunID: "run-1", PeriodID: "2026-10", Mode: "FULL", Status: "COMPLETED", BreakCount: 1, ExposureMinor: -5000, TriggeredBy: "fm-1", StartedAt: baseTime,
		}},
		ReconciliationBreaks: []ReconciliationBreakRow{{
			EntityID: eid, BreakID: "brk-1", RunID: "run-1", Dimension: "AR", ObjectType: "ArInvoice", ObjectID: "inv-1", ExpectedMinor: 105000, ActualMinor: 100000, DeltaMinor: -5000, Currency: "UZS", Severity: "MEDIUM", Status: "OPEN", CreatedAt: baseTime, UpdatedAt: baseTime,
		}},
	}
}

func TestExportPacks_BuildVerifyAllFormatsAndExpiry(t *testing.T) {
	t.Parallel()
	const eid = "sup-pack-unit-1"
	const signingSecret = "unit-pack-hmac-secret-2026"
	baseTime := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)

	repo := NewMemoryRepository()
	storage := NewMemoryPackStorage()
	auditSpy := &spyAudit{}
	svc := NewServiceWithDeps(repo, &MemoryCollector{Datasets: samplePackDatasets(eid, baseTime)}, storage)
	svc.SetAuditRecorder(auditSpy)
	svc.SetSigningSecret(signingSecret)

	curNow := baseTime
	svc.now = func() time.Time { return curNow }

	ctx := context.Background()
	from := baseTime.Add(-365 * 24 * time.Hour)
	to := baseTime

	// AUDITOR and AGENT cannot create packs -> ErrForbidden.
	if _, err := svc.CreatePack(ctx, CreatePackInput{
		TenantID: eid, EntityID: eid, PackType: PackTypeAudit, ActorID: "aud-1", ActorRole: auth.SupplierRoleAuditor,
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("AUDITOR CreatePack got %v, want ErrForbidden", err)
	}
	if _, err := svc.CreatePack(ctx, CreatePackInput{
		TenantID: eid, EntityID: eid, PackType: PackTypeTax, ActorID: "agt-1", ActorType: auth.ActorTypeAgent,
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("AGENT CreatePack got %v, want ErrForbidden", err)
	}

	// Finance Manager creates 24h AUDIT pack.
	pack, err := svc.CreatePack(ctx, CreatePackInput{
		TenantID:  eid,
		EntityID:  eid,
		PackType:  PackTypeAudit,
		PeriodID:  "2026-10",
		FromDate:  &from,
		ToDate:    &to,
		Format:    FormatZip,
		TTLHours:  24,
		ActorID:   "fm-unit-1",
		ActorRole: auth.SupplierRoleFinanceManager,
		ActorType: auth.ActorTypeHuman,
		TraceID:   "tr-pack-unit-1",
	})
	if err != nil {
		t.Fatalf("CreatePack: %v", err)
	}
	if pack.RowCount != 15 {
		t.Fatalf("pack.RowCount = %d, want 15", pack.RowCount)
	}
	if !strings.Contains(pack.WatermarkText, "Prepared for fm-unit-1") || !strings.Contains(pack.WatermarkText, pack.PackID) {
		t.Fatalf("unexpected WatermarkText: %q", pack.WatermarkText)
	}
	if len(pack.Artifacts) != 5 {
		t.Fatalf("len(Artifacts) = %d, want 5", len(pack.Artifacts))
	}

	// Download ZIP and inspect all bundled files + HMAC-SHA256 manifest.
	dlZip, err := svc.DownloadPack(ctx, DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatZip, ActorID: "aud-unit-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack zip: %v", err)
	}
	if dlZip.Sha256Hex != pack.Sha256Hex || dlZip.Watermark != pack.WatermarkText {
		t.Fatalf("header mismatch: sha=%s watermark=%s", dlZip.Sha256Hex, dlZip.Watermark)
	}

	zr, err := zip.NewReader(bytes.NewReader(dlZip.Data), int64(len(dlZip.Data)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	zipEntries := make(map[string][]byte)
	for _, f := range zr.File {
		rc, oerr := f.Open()
		if oerr != nil {
			t.Fatalf("zip open %s: %v", f.Name, oerr)
		}
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		zipEntries[f.Name] = b
	}
	for _, reqName := range []string{"manifest.json", "pack.xlsx", "pack.csv", "1c_journals.xml", "csv/gl_accounts.csv", "csv/reconciliation_breaks.csv"} {
		if _, ok := zipEntries[reqName]; !ok {
			t.Fatalf("missing expected entry %q in pack zip", reqName)
		}
	}

	// Verify signed manifest.json signature & per-file SHA-256 checksums.
	var manifest PackManifest
	if err := json.Unmarshal(zipEntries["manifest.json"], &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if manifest.SignatureAlg != "HMAC-SHA256" || manifest.SignatureHex == "" {
		t.Fatalf("expected HMAC-SHA256 signature on manifest, got alg=%s hex=%s", manifest.SignatureAlg, manifest.SignatureHex)
	}
	expectedSig := manifest.SignatureHex
	manifest.SignatureHex = ""
	canonical, _ := json.Marshal(&manifest)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	_, _ = mac.Write(canonical)
	if gotSig := hex.EncodeToString(mac.Sum(nil)); gotSig != expectedSig {
		t.Fatalf("manifest HMAC mismatch: got %s want %s", gotSig, expectedSig)
	}
	for _, ref := range manifest.Files {
		raw, ok := zipEntries[ref.Path]
		if !ok {
			t.Fatalf("manifest file %q missing in zip", ref.Path)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != ref.Sha256Hex || int64(len(raw)) != ref.SizeBytes {
			t.Fatalf("checksum/size mismatch on %s", ref.Path)
		}
	}

	// Open real OOXML .xlsx and verify Overview + all 15 sheets.
	dlXlsx, err := svc.DownloadPack(ctx, DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatXLSX, ActorID: "aud-unit-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack xlsx: %v", err)
	}
	xlFile, err := excelize.OpenReader(bytes.NewReader(dlXlsx.Data))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer func() { _ = xlFile.Close() }()
	if v, _ := xlFile.GetCellValue("Overview", "B1"); v != pack.WatermarkText {
		t.Fatalf("Overview watermark cell = %q, want %q", v, pack.WatermarkText)
	}
	if v, _ := xlFile.GetCellValue("GL_Accounts", "B2"); v != "1010" {
		t.Fatalf("GL_Accounts B2 = %q, want 1010", v)
	}
	if sheets := xlFile.GetSheetList(); len(sheets) != 16 {
		t.Fatalf("len(xlsx sheets) = %d (%v), want 16", len(sheets), sheets)
	}

	// Download CSV and parse with encoding/csv.
	dlCsv, err := svc.DownloadPack(ctx, DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatCSV, ActorID: "aud-unit-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack csv: %v", err)
	}
	csvRows, err := csv.NewReader(bytes.NewReader(dlCsv.Data)).ReadAll()
	if err != nil || len(csvRows) != 16 {
		t.Fatalf("csv rows = (%d, %v), want 16 (1 header + 15 rows)", len(csvRows), err)
	}

	// Download 1C XML and verify structure.
	dlXml, err := svc.DownloadPack(ctx, DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatXML, ActorID: "aud-unit-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack xml: %v", err)
	}
	var parsedXML oneCExportDoc
	if err := xml.Unmarshal(dlXml.Data, &parsedXML); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	if len(parsedXML.JournalEntries) != 1 || len(parsedXML.JournalEntries[0].Postings) != 1 || len(parsedXML.EsfDocuments) != 1 {
		t.Fatalf("unexpected 1C XML structure: %+v", parsedXML)
	}

	// AUDITOR list/get redacts DownloadToken.
	audCtx := auth.WithClaims(ctx, auth.Claims{Subject: "aud-unit-1", Role: auth.RoleAdmin, SupplierID: eid, SupplierRole: auth.SupplierRoleAuditor})
	fetched, err := svc.GetPack(audCtx, eid, pack.PackID)
	if err != nil || fetched.DownloadToken != "" || fetched.Expired {
		t.Fatalf("AUDITOR GetPack got (%+v, %v), want active pack with redacted DownloadToken", fetched, err)
	}

	// Advance clock past 24h -> DownloadPack fails with ErrPackExpired, GetPack reports EXPIRED.
	curNow = baseTime.Add(25 * time.Hour)
	if _, err := svc.DownloadPack(ctx, DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatZip, ActorID: "aud-unit-1",
	}); !errors.Is(err, ErrPackExpired) {
		t.Fatalf("expired DownloadPack got %v, want ErrPackExpired", err)
	}
	expiredView, err := svc.GetPack(audCtx, eid, pack.PackID)
	if err != nil || !expiredView.Expired || expiredView.Status != StatusExpired {
		t.Fatalf("expired GetPack view = (%+v, %v), want Expired=true Status=EXPIRED", expiredView, err)
	}

	// Verify AuditEvents captured EXPORT_PACK_CREATED and EXPORT_PACK_DOWNLOADED.
	auditSpy.mu.Lock()
	var createdCount, downloadCount int
	for _, ev := range auditSpy.events {
		switch ev.Action {
		case "EXPORT_PACK_CREATED":
			createdCount++
		case "EXPORT_PACK_DOWNLOADED":
			downloadCount++
		}
	}
	auditSpy.mu.Unlock()
	if createdCount != 1 || downloadCount != 4 {
		t.Fatalf("audit counts created=%d download=%d, want 1 & 4", createdCount, downloadCount)
	}
}

func TestExportPacks_ProductionFailClosedWithoutGCS(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("EXPORT_PACKS_GCS_BUCKET", "")
	if _, err := NewDefaultPackStorage(context.Background()); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("expected ErrStorageUnavailable in production without bucket, got %v", err)
	}
}

func TestSpannerExportPacks_All15TablesKeysetPaginationAndOutbox(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Spanner ExportPacks integration test")
	}
	project := envOrDefault("SPANNER_PROJECT", "pegasusx-local")
	instance := envOrDefault("SPANNER_INSTANCE", "pegasusx-instance")
	database := envOrDefault("SPANNER_DATABASE", "pegasusx-db")
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath, option.WithEndpoint(host), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	t.Cleanup(client.Close)

	if _, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ExportPacks", spanner.Key{"__probe__", "__probe__"}, []string{"PackId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		if requireSpanner {
			t.Fatalf("ExportPacks table unavailable: %v", probeErr)
		}
		t.Skipf("ExportPacks unavailable: %v", probeErr)
	}

	eid := "sup-ep-spn-" + uuid.NewString()[:8]
	now := time.Now().UTC().Add(-time.Hour)

	muts := []*spanner.Mutation{
		// 3 GlAccounts -> tests keyset pagination with batchSize=2
		spanner.InsertMap("GlAccounts", map[string]any{"EntityId": eid, "AccountCode": "1010", "Class": "ASSET", "Name": "Cash", "Currency": "UZS", "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("GlAccounts", map[string]any{"EntityId": eid, "AccountCode": "1210", "Class": "ASSET", "Name": "AR", "Currency": "UZS", "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("GlAccounts", map[string]any{"EntityId": eid, "AccountCode": "4010", "Class": "REVENUE", "Name": "Sales", "Currency": "UZS", "CreatedAt": now, "UpdatedAt": now}),
		// 3 JournalEntries + 4 JournalPostings
		spanner.InsertMap("JournalEntries", map[string]any{"EntityId": eid, "EntryId": "je-1", "PeriodId": "2026-10", "SourceType": "ORDER", "SourceId": "o-1", "Currency": "UZS", "IdempotencyKey": "ik-1-" + eid, "PrevHash": "0", "EntryHash": "h1", "PostedBy": "acct", "OccurredAt": now, "PostedAt": now}),
		spanner.InsertMap("JournalEntries", map[string]any{"EntityId": eid, "EntryId": "je-2", "PeriodId": "2026-10", "SourceType": "ORDER", "SourceId": "o-2", "Currency": "UZS", "IdempotencyKey": "ik-2-" + eid, "PrevHash": "h1", "EntryHash": "h2", "PostedBy": "acct", "OccurredAt": now, "PostedAt": now}),
		spanner.InsertMap("JournalEntries", map[string]any{"EntityId": eid, "EntryId": "je-3", "PeriodId": "2026-10", "SourceType": "ORDER", "SourceId": "o-3", "Currency": "UZS", "IdempotencyKey": "ik-3-" + eid, "PrevHash": "h2", "EntryHash": "h3", "PostedBy": "acct", "OccurredAt": now, "PostedAt": now}),
		spanner.InsertMap("JournalPostings", map[string]any{"EntityId": eid, "EntryId": "je-1", "Line": int64(1), "AccountCode": "1010", "Direction": "DEBIT", "AmountMinor": int64(1000), "Currency": "UZS", "FunctionalAmountMinor": int64(1000)}),
		spanner.InsertMap("JournalPostings", map[string]any{"EntityId": eid, "EntryId": "je-1", "Line": int64(2), "AccountCode": "4010", "Direction": "CREDIT", "AmountMinor": int64(1000), "Currency": "UZS", "FunctionalAmountMinor": int64(1000)}),
		spanner.InsertMap("JournalPostings", map[string]any{"EntityId": eid, "EntryId": "je-2", "Line": int64(1), "AccountCode": "1210", "Direction": "DEBIT", "AmountMinor": int64(2000), "Currency": "UZS", "FunctionalAmountMinor": int64(2000)}),
		spanner.InsertMap("JournalPostings", map[string]any{"EntityId": eid, "EntryId": "je-3", "Line": int64(1), "AccountCode": "1010", "Direction": "DEBIT", "AmountMinor": int64(3000), "Currency": "UZS", "FunctionalAmountMinor": int64(3000)}),
		// Remaining 12 tables
		spanner.InsertMap("ArInvoices", map[string]any{"InvoiceId": "inv-" + eid, "SupplierId": eid, "RetailerId": "ret-1", "OrderId": "o-1", "Status": "OPEN", "PrincipalMinor": int64(5000), "BalanceMinor": int64(5000), "Currency": "UZS", "CreditLeaveAt": now, "DueAt": now.Add(24 * time.Hour), "TermsDays": int64(30), "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("InventoryCostLayers", map[string]any{"EntityId": eid, "WarehouseId": "wh-1", "ProductId": "p-1", "LayerId": "l-1", "QtyRemaining": int64(5), "OriginalQty": int64(5), "UnitCostMinor": int64(400), "ReceivedAt": now, "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("InventoryMovements", map[string]any{"EntityId": eid, "MovementId": "mov-1", "MovementType": "RECEIPT", "ReferenceType": "PO", "ReferenceId": "po-1", "WarehouseId": "wh-1", "ProductId": "p-1", "QtyDelta": int64(5), "UnitCostMinor": int64(400), "TotalCostMinor": int64(2000), "CreatedAt": now}),
		spanner.InsertMap("EsfDocuments", map[string]any{"EntityId": eid, "EsfId": "esf-1", "Direction": "OUTBOUND", "CounterpartyTin": "123456789", "TotalMinor": int64(5000), "VatMinor": int64(600), "VatRateBps": int64(1200), "Status": "ACCEPTED", "Operator": "DIDOX", "PeriodId": "2026-10", "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("CashierBlindCountSessions", map[string]any{"SessionId": "cb-" + eid, "SupplierId": eid, "DepotId": "dep-1", "DriverId": "drv-1", "RouteId": "rt-1", "BagSealBarcode": "BAG-1", "DenominationBreakdownJson": "{}", "CountedCashMinor": int64(2000), "ExpectedCashMinor": int64(2000), "VarianceMinor": int64(0), "Status": "CLEARED", "CreatedAt": now}),
		spanner.InsertMap("DepotVaultLedgerEntries", map[string]any{"VaultEntryId": "dv-" + eid, "DepotId": "dep-1", "SupplierId": eid, "EntryType": "DEPOSIT", "AmountMinor": int64(2000), "RunningBalanceMinor": int64(2000), "CreatedAt": now}),
		spanner.InsertMap("PaymentLedgerEntries", map[string]any{"LedgerEntryId": "ple-" + eid, "SupplierId": eid, "Gateway": "PAYME", "EntryType": "SETTLEMENT", "AmountMinor": int64(3000), "Currency": "UZS", "Source": "PSP", "OccurredAt": now, "CreatedAt": now}),
		spanner.InsertMap("ApprovalRequests", map[string]any{"TenantId": eid, "RequestId": "apr-1", "ObjectType": "JournalEntry", "ObjectId": "je-1", "Action": "journal.approve", "RequestedBy": "acct", "ActorType": "HUMAN", "PayloadJson": "{}", "PayloadHash": "ph-1", "Status": "APPROVED", "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("ApprovalDecisions", map[string]any{"TenantId": eid, "RequestId": "apr-1", "DecisionId": "dec-1", "StepIndex": int64(0), "ApproverId": "fm", "Decision": "APPROVE", "CreatedAt": now}),
		spanner.InsertMap("AuditChainAnchors", map[string]any{"TenantId": eid, "DayBucket": "2026-10-15", "HeadEventId": "ev-1", "EventCount": int64(1), "RootHash": "rh-1", "AnchoredBy": "sys", "AnchoredAt": now}),
		spanner.InsertMap("ReconciliationRuns", map[string]any{"EntityId": eid, "RunId": "run-1", "PeriodId": "2026-10", "Mode": "FULL", "Status": "COMPLETED", "TriggeredBy": "fm", "StartedAt": now}),
		spanner.InsertMap("ReconciliationBreaks", map[string]any{"EntityId": eid, "BreakId": "brk-1", "RunId": "run-1", "Dimension": "AR", "ObjectType": "ArInvoice", "ObjectId": "inv-1", "Currency": "UZS", "Severity": "LOW", "Status": "OPEN", "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("ReconciliationBreaks", map[string]any{"EntityId": eid, "BreakId": "brk-2", "RunId": "run-1", "Dimension": "AR", "ObjectType": "ArInvoice", "ObjectId": "inv-2", "Currency": "UZS", "Severity": "LOW", "Status": "OPEN", "CreatedAt": now, "UpdatedAt": now}),
		spanner.InsertMap("ReconciliationBreaks", map[string]any{"EntityId": eid, "BreakId": "brk-3", "RunId": "run-1", "Dimension": "AR", "ObjectType": "ArInvoice", "ObjectId": "inv-3", "Currency": "UZS", "Severity": "LOW", "Status": "OPEN", "CreatedAt": now, "UpdatedAt": now}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed 15 tables: %v", err)
	}

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("ArInvoices", spanner.Key{"inv-" + eid}),
			spanner.Delete("InventoryCostLayers", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("InventoryMovements", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("EsfDocuments", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("CashierBlindCountSessions", spanner.Key{"cb-" + eid}),
			spanner.Delete("DepotVaultLedgerEntries", spanner.Key{"dv-" + eid}),
			spanner.Delete("PaymentLedgerEntries", spanner.Key{"ple-" + eid}),
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("AuditChainAnchors", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("ReconciliationBreaks", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("ReconciliationRuns", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("ExportPacks", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("OutboxEvents", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
		})
	})

	t.Setenv("EXPORT_PACKS_GCS_BUCKET", "")
	t.Setenv("EXPORT_PACKS_LOCAL_ROOT", t.TempDir())

	svc := NewService(client)
	svc.SetBatchSize(2) // force multiple keyset pagination loops in ReadOnlyTransaction
	svc.SetSigningSecret("spanner-export-pack-hmac-secret")
	auditSpy := &spyAudit{}
	svc.SetAuditRecorder(auditSpy)

	from := now.Add(-2 * time.Hour)
	to := now.Add(2 * time.Hour)
	pack, err := svc.CreatePack(ctx, CreatePackInput{
		TenantID:  eid,
		EntityID:  eid,
		PackType:  PackTypeAudit,
		PeriodID:  "2026-10",
		FromDate:  &from,
		ToDate:    &to,
		Format:    FormatZip,
		TTLHours:  48,
		ActorID:   "owner-spn-1",
		ActorRole: auth.SupplierRoleOwner,
		ActorType: auth.ActorTypeHuman,
		TraceID:   "tr-spn-ep-1",
	})
	if err != nil {
		t.Fatalf("Spanner CreatePack: %v", err)
	}
	if pack.RowCount != 24 {
		t.Fatalf("Spanner pack.RowCount = %d, want 24 across all 15 tables & keyset pages", pack.RowCount)
	}

	// Verify ExportPacks row and ListPacks on Spanner.
	listed, err := svc.ListPacks(ctx, eid, eid, 20)
	if err != nil || len(listed) != 1 || listed[0].PackID != pack.PackID {
		t.Fatalf("Spanner ListPacks = (%+v, %v)", listed, err)
	}

	// Verify OutboxEvents emitted export.pack.created on events.TopicLedger.
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @eid`,
		Params: map[string]any{"eid": eid},
	})
	defer iter.Stop()
	foundOutbox := false
	for {
		row, rerr := iter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr != nil {
			t.Fatalf("query OutboxEvents: %v", rerr)
		}
		var topic string
		var payload []byte
		if serr := row.Columns(&topic, &payload); serr != nil {
			t.Fatalf("scan OutboxEvents: %v", serr)
		}
		if topic == events.TopicLedger && strings.Contains(string(payload), `"export.pack.created"`) && strings.Contains(string(payload), pack.PackID) {
			foundOutbox = true
		}
	}
	if !foundOutbox {
		t.Fatalf("expected export.pack.created on TopicLedger in OutboxEvents for %s", eid)
	}

	// Download via token & verify XLSX contains all 3 GlAccounts rows + checksum + AuditEvent.
	dl, err := svc.DownloadPack(ctx, DownloadPackInput{
		PackID: pack.PackID, Format: FormatXLSX, DownloadToken: pack.DownloadToken, ActorID: "ext-aud-1",
	})
	if err != nil {
		t.Fatalf("Spanner DownloadPack xlsx: %v", err)
	}
	xf, err := excelize.OpenReader(bytes.NewReader(dl.Data))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer func() { _ = xf.Close() }()
	rows, err := xf.GetRows("GL_Accounts")
	if err != nil || len(rows) != 4 {
		t.Fatalf("GL_Accounts sheet rows = (%d, %v), want 4 (1 header + 3 keyset-paged accounts)", len(rows), err)
	}
	auditSpy.mu.Lock()
	var hasCreated, hasDownloaded bool
	for _, ev := range auditSpy.events {
		if ev.Action == "EXPORT_PACK_CREATED" {
			hasCreated = true
		}
		if ev.Action == "EXPORT_PACK_DOWNLOADED" {
			hasDownloaded = true
		}
	}
	auditSpy.mu.Unlock()
	if !hasCreated || !hasDownloaded {
		t.Fatalf("missing AuditEvents (created=%v downloaded=%v)", hasCreated, hasDownloaded)
	}
}

func TestSafeSpreadsheetCell_NeutralizesFormulasPreservesNegatives(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"=CMD|' /C calc'!A0", "'=CMD|' /C calc'!A0"},
		{"+SUM(A1:A2)", "'+SUM(A1:A2)"},
		{"@SUM(1,1)", "'@SUM(1,1)"},
		{"\t=1+1", "'\t=1+1"},
		{"\r=1+1", "'\r=1+1"},
		{"-1+1", "'-1+1"},
		{"-500000", "-500000"},
		{"normal memo", "normal memo"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := safeSpreadsheetCell(tc.in); got != tc.want {
			t.Fatalf("safeSpreadsheetCell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func envOrDefault(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
