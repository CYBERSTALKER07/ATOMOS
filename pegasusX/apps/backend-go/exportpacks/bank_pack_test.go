package exportpacks

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestBankPack_CreditTape_DPDBuckets_Concentration_LossCurves_AllFormats(t *testing.T) {
	t.Parallel()
	const eid = "sup-bank-unit-1"
	asOf := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	ds := &PackDatasets{
		GlAccounts: []GlAccountRow{{
			EntityID: eid, AccountCode: "1210", Class: "ASSET", Name: "Accounts Receivable", Currency: "UZS", CreatedAt: asOf, UpdatedAt: asOf,
		}},
		ArInvoices: []ArInvoiceRow{
			// ret-1: 15 DPD (0-30), 100k open out of 300k principal (200k recovered), vintage 2026-10
			{
				InvoiceID: "inv-b1", SupplierID: eid, RetailerID: "ret-1", OrderID: "ord-b1", Status: "PARTIAL",
				PrincipalMinor: 300_000, BalanceMinor: 100_000, Currency: "UZS",
				CreditLeaveAt: asOf.AddDate(0, -1, 0), DueAt: asOf.Add(-15 * 24 * time.Hour),
				TermsDays: 15, GracePeriodDays: 0, CreatedAt: asOf.AddDate(0, -1, 0), UpdatedAt: asOf,
			},
			// ret-1: 45 DPD (31-60), 300k open, vintage 2026-09
			{
				InvoiceID: "inv-b2", SupplierID: eid, RetailerID: "ret-1", OrderID: "ord-b2", Status: "OPEN",
				PrincipalMinor: 300_000, BalanceMinor: 300_000, Currency: "UZS",
				CreditLeaveAt: asOf.AddDate(0, -2, 0), DueAt: asOf.Add(-45 * 24 * time.Hour),
				TermsDays: 15, GracePeriodDays: 0, CreatedAt: asOf.AddDate(0, -2, 0), UpdatedAt: asOf,
			},
			// ret-2: 75 DPD (61-90), 200k open, vintage 2026-08
			{
				InvoiceID: "inv-b3", SupplierID: eid, RetailerID: "ret-2", OrderID: "ord-b3", Status: "OPEN",
				PrincipalMinor: 200_000, BalanceMinor: 200_000, Currency: "UZS",
				CreditLeaveAt: asOf.AddDate(0, -3, 0), DueAt: asOf.Add(-75 * 24 * time.Hour),
				TermsDays: 15, GracePeriodDays: 0, CreatedAt: asOf.AddDate(0, -3, 0), UpdatedAt: asOf,
			},
			// ret-3: 110 DPD (90+), 400k open, vintage 2026-07
			{
				InvoiceID: "inv-b4", SupplierID: eid, RetailerID: "ret-3", OrderID: "ord-b4", Status: "OPEN",
				PrincipalMinor: 400_000, BalanceMinor: 400_000, Currency: "UZS",
				CreditLeaveAt: asOf.AddDate(0, -4, 0), DueAt: asOf.Add(-110 * 24 * time.Hour),
				TermsDays: 15, GracePeriodDays: 0, CreatedAt: asOf.AddDate(0, -4, 0), UpdatedAt: asOf,
			},
			// ret-2: written-off invoice in 2026-07 vintage (0 open balance, 100k written off)
			{
				InvoiceID: "inv-b5", SupplierID: eid, RetailerID: "ret-2", OrderID: "ord-b5", Status: "VOID",
				PrincipalMinor: 100_000, BalanceMinor: 0, Currency: "UZS",
				CreditLeaveAt: asOf.AddDate(0, -4, 0), DueAt: asOf.Add(-100 * 24 * time.Hour),
				TermsDays: 15, GracePeriodDays: 0, CreatedAt: asOf.AddDate(0, -4, 0), UpdatedAt: asOf,
			},
		},
		ArLedgerEntries: []ArLedgerEntryRow{
			{EntryID: "le-1", InvoiceID: "inv-b1", SupplierID: eid, RetailerID: "ret-1", EntryType: "OPEN", AmountMinor: 300_000, IdempotencyKey: "k1", CreatedAt: asOf.AddDate(0, -1, 0)},
			{EntryID: "le-2", InvoiceID: "inv-b1", SupplierID: eid, RetailerID: "ret-1", EntryType: "PAYMENT", AmountMinor: -200_000, IdempotencyKey: "k2", CreatedAt: asOf.AddDate(0, 0, -10)},
			{EntryID: "le-3", InvoiceID: "inv-b5", SupplierID: eid, RetailerID: "ret-2", EntryType: "WRITE_OFF", AmountMinor: -100_000, IdempotencyKey: "k3", CreatedAt: asOf.AddDate(0, -2, 0)},
		},
		RetailerPaymentTerms: []RetailerPaymentTermsRow{
			{RetailerID: "ret-1", SupplierID: eid, CreditEnabled: true, TermsDays: 30, GracePeriodDays: 3, CreditLimitMinor: 1_000_000, ConsentVerified: true, UpdatedAt: asOf},
			{RetailerID: "ret-2", SupplierID: eid, CreditEnabled: true, TermsDays: 30, GracePeriodDays: 0, CreditLimitMinor: 500_000, ConsentVerified: false, UpdatedAt: asOf},
			{RetailerID: "ret-3", SupplierID: eid, CreditEnabled: true, TermsDays: 15, GracePeriodDays: 0, CreditLimitMinor: 500_000, ConsentVerified: true, UpdatedAt: asOf},
		},
		RetailerCreditProfiles: []RetailerCreditProfileRow{
			{RetailerID: "ret-1", SupplierID: eid, CreditLimitMinor: 1_000_000, CurrentBalanceMinor: 400_000, RiskScore: 82, DelinquencyCount: 1, Status: "ACTIVE", CreatedAt: asOf, UpdatedAt: asOf},
			{RetailerID: "ret-2", SupplierID: eid, CreditLimitMinor: 500_000, CurrentBalanceMinor: 200_000, RiskScore: 60, DelinquencyCount: 2, Status: "ACTIVE", CreatedAt: asOf, UpdatedAt: asOf},
			{RetailerID: "ret-3", SupplierID: eid, CreditLimitMinor: 500_000, CurrentBalanceMinor: 400_000, RiskScore: 40, DelinquencyCount: 4, Status: "HOLD", PrepayOnlyLocked: true, CreatedAt: asOf, UpdatedAt: asOf},
		},
		CreditScoreHistory: []CreditScoreHistoryRow{
			{RetailerID: "ret-1", ScoreID: "sc-r1", Scope: "RELATIONSHIP", SupplierID: eid, ConsentVerified: true, Score: 84, PdBps: 180, RiskTier: "LOW", ComputedAt: asOf},
			{RetailerID: "ret-1", ScoreID: "sc-n1", Scope: "NETWORK", ConsentVerified: true, Score: 88, PdBps: 120, RiskTier: "LOW", ComputedAt: asOf},
			// ret-2 has ConsentVerified=false so network score MUST be suppressed (0)!
			{RetailerID: "ret-2", ScoreID: "sc-r2", Scope: "RELATIONSHIP", SupplierID: eid, ConsentVerified: false, Score: 58, PdBps: 900, RiskTier: "MEDIUM", ComputedAt: asOf},
			{RetailerID: "ret-2", ScoreID: "sc-n2", Scope: "NETWORK", ConsentVerified: false, Score: 70, PdBps: 450, RiskTier: "MEDIUM", ComputedAt: asOf},
			{RetailerID: "ret-3", ScoreID: "sc-r3", Scope: "RELATIONSHIP", SupplierID: eid, ConsentVerified: true, Score: 35, PdBps: 2400, RiskTier: "HIGH", ComputedAt: asOf},
			{RetailerID: "ret-3", ScoreID: "sc-n3", Scope: "NETWORK", ConsentVerified: true, Score: 42, PdBps: 2100, RiskTier: "HIGH", ComputedAt: asOf},
		},
	}

	repo := NewMemoryRepository()
	storage := NewMemoryPackStorage()
	svc := NewServiceWithDeps(repo, &MemoryCollector{Datasets: ds}, storage)
	svc.SetSigningSecret("bank-pack-unit-secret-2026")
	svc.now = func() time.Time { return asOf }

	from := asOf.AddDate(-1, 0, 0)
	to := asOf
	pack, err := svc.CreatePack(context.Background(), CreatePackInput{
		TenantID:  eid,
		EntityID:  eid,
		PackType:  PackTypeBank,
		PeriodID:  "2026-11",
		FromDate:  &from,
		ToDate:    &to,
		Format:    FormatZip,
		TTLHours:  24,
		ActorID:   "fm-bank-1",
		ActorRole: auth.SupplierRoleFinanceManager,
		ActorType: auth.ActorTypeHuman,
	})
	if err != nil {
		t.Fatalf("CreatePack(BANK): %v", err)
	}
	if pack.Manifest == nil || pack.Manifest.BankCredit == nil {
		t.Fatalf("expected BankCreditSummary in pack manifest: %+v", pack.Manifest)
	}

	bc := pack.Manifest.BankCredit
	if bc.TotalInvoices != 5 || bc.TotalRetailers != 3 || bc.ConsentVerifiedRetailers != 2 {
		t.Fatalf("unexpected counts: %+v", bc)
	}
	if bc.TotalExposureMinor != 1_000_000 || bc.TotalPrincipalMinor != 1_300_000 || bc.TotalRecoveredMinor != 200_000 || bc.TotalWrittenOffMinor != 100_000 {
		t.Fatalf("unexpected totals: %+v", bc)
	}
	// Top-1 debtor is tied at 400k (ret-1 and ret-3 -> 4000 bps each), top-5 = 10000 bps.
	if bc.Top1ConcentrationBps != 4000 || bc.Top5ConcentrationBps != 10000 {
		t.Fatalf("concentration bps got top1=%d top5=%d, want 4000 & 10000", bc.Top1ConcentrationBps, bc.Top5ConcentrationBps)
	}
	// Check DPD buckets sum to 10000 bps and exact exposures: 100k, 300k, 200k, 400k.
	wantBucketExp := map[string]int64{"0-30": 100_000, "31-60": 300_000, "61-90": 200_000, "90+": 400_000}
	wantBucketBps := map[string]int64{"0-30": 1000, "31-60": 3000, "61-90": 2000, "90+": 4000}
	if len(bc.DPDBuckets) != 4 {
		t.Fatalf("len(DPDBuckets)=%d want=4", len(bc.DPDBuckets))
	}
	for _, b := range bc.DPDBuckets {
		if b.ExposureMinor != wantBucketExp[b.Bucket] || b.ShareBps != wantBucketBps[b.Bucket] {
			t.Fatalf("bucket %s mismatch: %+v", b.Bucket, b)
		}
	}
	// Check ret-2 network score is suppressed (0) because ConsentVerified=false.
	for _, dc := range bc.DebtorConcentration {
		if dc.RetailerID == "ret-2" && (dc.ConsentVerified || dc.NetworkScore != 0) {
			t.Fatalf("ret-2 unconsented network score leaked: %+v", dc)
		}
		if dc.RetailerID == "ret-1" && (!dc.ConsentVerified || dc.RelationshipScore != 84 || dc.NetworkScore != 88) {
			t.Fatalf("ret-1 score breakdown mismatch: %+v", dc)
		}
	}

	// Verify ZIP includes all 4 Bank Credit CSVs + manifest.json with zero floats.
	dlZip, err := svc.DownloadPack(context.Background(), DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatZip, ActorID: "aud-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack zip: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(dlZip.Data), int64(len(dlZip.Data)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	zipEntries := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		zipEntries[f.Name] = b
	}
	for _, reqCSV := range []string{
		"csv/bank_credit_tape.csv",
		"csv/dpd_buckets.csv",
		"csv/debtor_concentration.csv",
		"csv/loss_recovery_curves.csv",
	} {
		if len(zipEntries[reqCSV]) == 0 {
			t.Fatalf("missing %s in BANK ZIP", reqCSV)
		}
	}

	// Verify XLSX has the 4 Bank schedules (16 base + 4 bank = 20 sheets).
	dlXlsx, err := svc.DownloadPack(context.Background(), DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatXLSX, ActorID: "aud-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack xlsx: %v", err)
	}
	xl, err := excelize.OpenReader(bytes.NewReader(dlXlsx.Data))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	defer func() { _ = xl.Close() }()
	sheets := xl.GetSheetList()
	if len(sheets) != 20 {
		t.Fatalf("BANK xlsx sheet count=%d (%v), want 20", len(sheets), sheets)
	}
	for _, name := range []string{"Bank_Credit_Tape", "DPD_Buckets", "Debtor_Concentration", "Loss_LossRecovery_Curves"} {
		rows, rErr := xl.GetRows(name)
		if rErr != nil || len(rows) < 2 {
			t.Fatalf("sheet %s rows=(%d, %v)", name, len(rows), rErr)
		}
	}

	// Verify combined CSV contains bank rows.
	dlCsv, err := svc.DownloadPack(context.Background(), DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatCSV, ActorID: "aud-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack csv: %v", err)
	}
	csvRecords, err := csv.NewReader(bytes.NewReader(dlCsv.Data)).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	hasTapeRow := false
	for _, r := range csvRecords {
		if len(r) > 0 && r[0] == "bank_credit_tape" {
			hasTapeRow = true
			break
		}
	}
	if !hasTapeRow {
		t.Fatalf("combined pack.csv missing bank_credit_tape rows")
	}

	// Verify JSON artifact parses with zero floats in bank_credit_summary.
	dlJSON, err := svc.DownloadPack(context.Background(), DownloadPackInput{
		TenantID: eid, PackID: pack.PackID, Format: FormatJSON, ActorID: "aud-1", ActorRole: auth.SupplierRoleAuditor,
	})
	if err != nil {
		t.Fatalf("DownloadPack json: %v", err)
	}
	var manifestMap map[string]any
	dec := json.NewDecoder(bytes.NewReader(dlJSON.Data))
	dec.UseNumber()
	if err := dec.Decode(&manifestMap); err != nil {
		t.Fatalf("decode manifest json: %v", err)
	}
	if err := assertJSONNoFloatLiterals(manifestMap); err != nil {
		t.Fatalf("BANK manifest JSON contains float literal: %v", err)
	}
}

func assertJSONNoFloatLiterals(v any) error {
	switch tv := v.(type) {
	case float64, float32:
		return fmt.Errorf("unexpected float %T", tv)
	case json.Number:
		if strings.ContainsAny(tv.String(), ".eE") {
			return fmt.Errorf("decimal json number %q", tv.String())
		}
	case map[string]any:
		for k, elem := range tv {
			if err := assertJSONNoFloatLiterals(elem); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
	case []any:
		for i, elem := range tv {
			if err := assertJSONNoFloatLiterals(elem); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	}
	return nil
}

func TestSpannerBankPack_CreditTapeBucketsConcentrationAndCurves(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner BANK pack test")
	}

	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOrDefault("SPANNER_PROJECT", "pegasusx-local"),
		envOrDefault("SPANNER_INSTANCE", "pegasusx-instance"),
		envOrDefault("SPANNER_DATABASE", "pegasusx-db"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath, option.WithEndpoint(host), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		t.Fatalf("spanner connect: %v", err)
	}
	t.Cleanup(client.Close)

	if _, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "CreditScoreHistory", spanner.Key{"__probe__", "__probe__"}, []string{"ScoreId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		t.Fatalf("CreditScoreHistory unavailable: %v", probeErr)
	}

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	eid := "sup-bank-spn-" + tag
	ret1 := "ret1-bank-" + tag
	ret2 := "ret2-bank-" + tag
	inv1 := "inv1-bank-" + tag
	inv2 := "inv2-bank-" + tag
	le1 := "le1-bank-" + tag
	le2 := "le2-bank-" + tag
	now := time.Now().UTC().Add(-time.Hour)

	muts := []*spanner.Mutation{
		spanner.InsertMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": ret1, "SupplierId": eid, "CreditEnabled": true, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(2_000_000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": ret2, "SupplierId": eid, "CreditEnabled": true, "TermsDays": int64(15),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(1_000_000), "UseGlobalDefaults": false,
			"ConsentVerified": false, "Version": int64(1), "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": ret1, "SupplierId": eid, "CreditLimitMinor": int64(2_000_000),
			"CurrentBalanceMinor": int64(750_000), "ReservedMinor": int64(0), "AvailableCreditMinor": int64(1_250_000),
			"RiskScore": int64(85), "DelinquencyCount": int64(0), "Status": "ACTIVE",
			"PrepayOnlyLocked": false, "Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": ret2, "SupplierId": eid, "CreditLimitMinor": int64(1_000_000),
			"CurrentBalanceMinor": int64(250_000), "ReservedMinor": int64(0), "AvailableCreditMinor": int64(750_000),
			"RiskScore": int64(42), "DelinquencyCount": int64(3), "Status": "ACTIVE",
			"PrepayOnlyLocked": true, "Version": int64(1), "CreatedAt": now, "UpdatedAt": now,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": inv1, "SupplierId": eid, "RetailerId": ret1, "OrderId": "ord1-" + tag,
			"Status": "PARTIAL", "PrincipalMinor": int64(1_000_000), "BalanceMinor": int64(750_000),
			"Currency": "UZS", "CreditLeaveAt": now.AddDate(0, -1, 0), "DueAt": now.Add(-12 * 24 * time.Hour),
			"TermsDays": int64(30), "GracePeriodDays": int64(0), "CreatedAt": now.AddDate(0, -1, 0), "UpdatedAt": now,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": inv2, "SupplierId": eid, "RetailerId": ret2, "OrderId": "ord2-" + tag,
			"Status": "OPEN", "PrincipalMinor": int64(250_000), "BalanceMinor": int64(250_000),
			"Currency": "UZS", "CreditLeaveAt": now.AddDate(0, -4, 0), "DueAt": now.Add(-95 * 24 * time.Hour),
			"TermsDays": int64(15), "GracePeriodDays": int64(0), "CreatedAt": now.AddDate(0, -4, 0), "UpdatedAt": now,
		}),
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId": le1, "InvoiceId": inv1, "SupplierId": eid, "RetailerId": ret1,
			"EntryType": "PAYMENT", "AmountMinor": int64(-250_000), "IdempotencyKey": "ik-le1-" + tag, "CreatedAt": now,
		}),
		spanner.InsertMap("ArLedgerEntries", map[string]any{
			"EntryId": le2, "InvoiceId": inv2, "SupplierId": eid, "RetailerId": ret2,
			"EntryType": "OPEN", "AmountMinor": int64(250_000), "IdempotencyKey": "ik-le2-" + tag, "CreatedAt": now.AddDate(0, -4, 0),
		}),
		spanner.InsertMap("CreditScoreHistory", map[string]any{
			"RetailerId": ret1, "ScoreId": "rel-" + tag, "Scope": "RELATIONSHIP", "SupplierId": eid,
			"ConsentVerified": true, "ModelVersion": "v2", "ScorerSource": "SCORECARD",
			"Score": int64(86), "PdBps": int64(140), "RiskTier": "LOW",
			"SuggestedLimitMinor": int64(2_200_000), "CurrentLimitMinor": int64(2_000_000),
			"ReasonCodesJson": "[]", "FeaturesHash": "fh1", "DisputesIsolatedCount": int64(0),
			"WindowStart": now.AddDate(0, -3, 0), "WindowEnd": now, "ComputedAt": now, "CreatedAt": now,
		}),
		spanner.InsertMap("CreditScoreHistory", map[string]any{
			"RetailerId": ret1, "ScoreId": "net-" + tag, "Scope": "NETWORK", "SupplierId": nil,
			"ConsentVerified": true, "ModelVersion": "v2", "ScorerSource": "SCORECARD",
			"Score": int64(91), "PdBps": int64(90), "RiskTier": "LOW",
			"SuggestedLimitMinor": int64(2_500_000), "CurrentLimitMinor": int64(2_000_000),
			"ReasonCodesJson": "[]", "FeaturesHash": "fh2", "DisputesIsolatedCount": int64(0),
			"WindowStart": now.AddDate(0, -3, 0), "WindowEnd": now, "ComputedAt": now, "CreatedAt": now,
		}),
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed bank Spanner tables: %v", err)
	}
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cctx, []*spanner.Mutation{
			spanner.Delete("CreditScoreHistory", spanner.KeyRange{Start: spanner.Key{ret1}, End: spanner.Key{ret1 + "~"}}),
			spanner.Delete("ArLedgerEntries", spanner.Key{le1}),
			spanner.Delete("ArLedgerEntries", spanner.Key{le2}),
			spanner.Delete("ArInvoices", spanner.Key{inv1}),
			spanner.Delete("ArInvoices", spanner.Key{inv2}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{ret1, eid}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{ret2, eid}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{ret1, eid}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{ret2, eid}),
			spanner.Delete("ExportPacks", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
			spanner.Delete("OutboxEvents", spanner.KeyRange{Start: spanner.Key{eid}, End: spanner.Key{eid + "~"}}),
		})
	})

	t.Setenv("EXPORT_PACKS_GCS_BUCKET", "")
	t.Setenv("EXPORT_PACKS_LOCAL_ROOT", t.TempDir())

	svc := NewService(client)
	svc.SetBatchSize(1) // Exercise keyset pagination on bank tables too!
	svc.SetSigningSecret("live-bank-pack-secret")

	from := now.AddDate(-1, 0, 0)
	to := now.Add(time.Minute)
	pack, err := svc.CreatePack(ctx, CreatePackInput{
		TenantID:  eid,
		EntityID:  eid,
		PackType:  PackTypeBank,
		PeriodID:  "2026-11",
		FromDate:  &from,
		ToDate:    &to,
		Format:    FormatZip,
		ActorID:   "owner-bank-spn",
		ActorRole: auth.SupplierRoleOwner,
		ActorType: auth.ActorTypeHuman,
	})
	if err != nil {
		t.Fatalf("Spanner CreatePack(BANK): %v", err)
	}
	if pack.Manifest == nil || pack.Manifest.BankCredit == nil {
		t.Fatalf("missing BankCreditSummary in Spanner BANK pack manifest")
	}
	bc := pack.Manifest.BankCredit
	if bc.TotalInvoices != 2 || bc.TotalExposureMinor != 1_000_000 || bc.TotalRecoveredMinor != 250_000 ||
		bc.Top1ConcentrationBps != 7500 || bc.ConsentVerifiedRetailers != 1 || bc.AvgNetworkScore != 91 {
		t.Fatalf("unexpected Spanner BankCreditSummary: %+v", bc)
	}
}
