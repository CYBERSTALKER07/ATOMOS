package partner

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestExportWindow_PreservesMultiYearAndUnlimited(t *testing.T) {
	t.Parallel()
	fixedNow := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	w := &ExportWorker{now: func() time.Time { return fixedNow }}

	// 1. Omitted from/to and unlimited=false -> safe default 7d window.
	fromDef, toDef, unlDef := w.exportWindow(ExportJob{})
	if unlDef {
		t.Fatal("expected unlimited=false when omitted")
	}
	if gotDays := int(toDef.Sub(fromDef).Hours() / 24); gotDays != 7 {
		t.Fatalf("default window days = %d, want 7", gotDays)
	}

	// 2. Explicit multi-year from/to -> NOT clamped to 90 days.
	from2020 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to2026 := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	fromGot, toGot, unlExp := w.exportWindow(ExportJob{FromDate: &from2020, ToDate: &to2026})
	if !unlExp {
		t.Fatal("expected explicit from/to to enable unlimited pagination")
	}
	if !fromGot.Equal(from2020) {
		t.Fatalf("from = %v, want %v (no 90-day clamp)", fromGot, from2020)
	}
	if toGot.Year() != 2026 || toGot.Month() != 11 || toGot.Day() != 17 || toGot.Hour() != 23 {
		t.Fatalf("to = %v, expected end of 2026-11-17", toGot)
	}

	// 3. unlimited=true without from/to -> 1970-01-01..now, unlimited=true.
	fromUnl, toUnl, unlFlag := w.exportWindow(ExportJob{Unlimited: true})
	if !unlFlag || fromUnl.Year() != 1970 || !toUnl.Equal(fixedNow) {
		t.Fatalf("unlimited window got (%v, %v, %v)", fromUnl, toUnl, unlFlag)
	}
}

func TestPartnerExport_Over50kRows_XLSX_CSV_And_Journals(t *testing.T) {
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	t.Setenv("PARTNER_SFTP_ENABLED", "false")
	root := t.TempDir()
	t.Setenv("PARTNER_EXPORT_LOCAL_ROOT", root)

	const totalRows = 55000
	exports := NewMemoryExportRepository()
	sftp := NewMemorySftpConfigRepository()
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetExportRepos(exports, sftp)
	worker := NewExportWorker(exports, sftp, nil, nil)
	worker.rowsFetcher = func(_ context.Context, j ExportJob) ([]map[string]any, error) {
		rows := make([]map[string]any, totalRows)
		for i := 0; i < totalRows; i++ {
			rows[i] = map[string]any{
				"order_id":     fmt.Sprintf("ord-%06d", i),
				"amount_minor": int64(10000 + i),
				"currency":     "UZS",
				"status":       "COMPLETED",
			}
		}
		return rows, nil
	}

	p := Principal{TenantType: TenantSupplier, TenantID: "sup-55k", Scopes: []string{ScopeExportsRead}}

	// 1. Real OOXML .xlsx with 55,000 rows (> 50,000 cap).
	jobXLSX, err := svc.CreateExportJobWithOptions(context.Background(), p, ExportResourceOrders, ExportFormatXLSX, nil, nil, true)
	if err != nil {
		t.Fatalf("CreateExportJobWithOptions xlsx: %v", err)
	}
	if n, err := worker.RunOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("RunOnce xlsx n=%d err=%v", n, err)
	}
	gotXLSX, _, err := svc.GetExportJob(context.Background(), p, jobXLSX.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if gotXLSX.Status != ExportStatusSucceeded || gotXLSX.RowCount != totalRows {
		t.Fatalf("gotXLSX = %+v, want RowCount=%d", gotXLSX, totalRows)
	}
	xlsxPath := filepath.Join(root, filepath.FromSlash(gotXLSX.ObjectPath))
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		t.Fatalf("excelize.OpenFile: %v", err)
	}
	defer f.Close()
	sheetRows, err := f.GetRows("Export")
	if err != nil {
		t.Fatalf("GetRows(Export): %v", err)
	}
	if len(sheetRows) != totalRows+1 {
		t.Fatalf("xlsx sheet rows = %d, want %d (header + %d rows)", len(sheetRows), totalRows+1, totalRows)
	}
	if sheetRows[0][0] != "amount_minor" || sheetRows[0][2] != "order_id" {
		t.Fatalf("unexpected sorted headers: %v", sheetRows[0])
	}
	lastRow := sheetRows[totalRows]
	if lastRow[0] != strconv.Itoa(10000+totalRows-1) || lastRow[2] != fmt.Sprintf("ord-%06d", totalRows-1) {
		t.Fatalf("unexpected last row: %v", lastRow)
	}

	// 2. CSV export with 55,000 rows.
	jobCSV, err := svc.CreateExportJobWithOptions(context.Background(), p, ExportResourceOrders, ExportFormatCSV, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := worker.RunOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("RunOnce csv n=%d err=%v", n, err)
	}
	gotCSV, _, err := svc.GetExportJob(context.Background(), p, jobCSV.JobID)
	if err != nil || gotCSV.RowCount != totalRows {
		t.Fatalf("gotCSV = (%+v, %v)", gotCSV, err)
	}
	csvRaw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(gotCSV.ObjectPath)))
	if err != nil {
		t.Fatal(err)
	}
	csvRecs, err := csv.NewReader(bytes.NewReader(csvRaw)).ReadAll()
	if err != nil || len(csvRecs) != totalRows+1 {
		t.Fatalf("csv records = %d err=%v, want %d", len(csvRecs), err, totalRows+1)
	}

	// 3. Deduplicate journals with maxRows=0 (>50k items, O(N) fast, zero truncation).
	now := time.Now().UTC()
	arItems := make([]journalExportItem, 52000)
	for i := range arItems {
		id := fmt.Sprintf("ar-%06d", i)
		arItems[i] = journalExportItem{
			Row:        journalRow(map[string]any{"entry_id": id, "source": "ar", "amount_minor": int64(100 + i)}),
			Source:     "ar",
			EntryID:    id,
			EntryType:  "OPEN",
			SupplierID: "sup-55k",
			EntryDate:  now.Add(-time.Duration(i) * time.Second),
		}
	}
	dedup := deduplicateJournalItemsWithLimit(arItems, nil, nil, 0)
	if len(dedup) != 52000 {
		t.Fatalf("unlimited journal dedup = %d, want 52000", len(dedup))
	}
}

func TestPartnerExportHTTP_XLSXAndUnlimited(t *testing.T) {
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	exports := NewMemoryExportRepository()
	svc := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svc.SetExportRepos(exports, NewMemorySftpConfigRepository())
	h := &Handlers{Svc: svc}

	r := chi.NewRouter()
	const jwtSecret = "partner-export-test-secret-32-bytes!!"
	r.Use(auth.SessionAuth(jwtSecret))
	RegisterAdminKeyRoutes(r, h)

	tok, err := auth.Issue(auth.Claims{
		Subject:    "owner-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-http-exp",
	}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"resource":"orders","format":"xlsx","from":"2021-01-01","to":"2026-11-17","unlimited":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/partner-exports", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /v1/supplier/partner-exports = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["format"] != "xlsx" || resp["unlimited"] != true {
		t.Fatalf("unexpected DTO: %+v", resp)
	}
}

func openPartnerSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner partner export test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := "projects/" + envOr("SPANNER_PROJECT", "pegasusx-local") +
		"/instances/" + envOr("SPANNER_INSTANCE", "pegasusx-instance") +
		"/databases/" + envOr("SPANNER_DATABASE", "pegasusx-db")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner dial: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	if _, pErr := c.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "PartnerExportJobs", spanner.Key{"__probe__"}, []string{"JobId"}); pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		c.Close()
		if requireSpanner {
			t.Fatalf("PartnerExportJobs probe: %v", pErr)
		}
		t.Skipf("PartnerExportJobs unavailable: %v", pErr)
	}
	t.Cleanup(c.Close)
	return c
}

func TestSpannerPartnerExport_MultiYearPaginatedXLSX(t *testing.T) {
	client := openPartnerSpannerClient(t)
	ctx := context.Background()
	t.Setenv("PARTNER_EXPORTS_ENABLED", "true")
	t.Setenv("PARTNER_SFTP_ENABLED", "false")
	root := t.TempDir()
	t.Setenv("PARTNER_EXPORT_LOCAL_ROOT", root)

	supID := "sup-px-" + uuid.NewString()[:8]
	retID := "ret-px-" + uuid.NewString()[:8]
	var orderIDs []string
	var muts []*spanner.Mutation
	// Insert 11 orders spanning 2021 through 2026 (> 90 days!) to exercise multi-page keyset pagination.
	for i := 0; i < 11; i++ {
		oid := fmt.Sprintf("ord-px-%s-%02d", supID[len(supID)-6:], i)
		orderIDs = append(orderIDs, oid)
		ts := time.Date(2021+(i%6), time.Month((i%12)+1), 10, 12, 0, 0, 0, time.UTC)
		muts = append(muts, spanner.InsertMap("Orders", map[string]any{
			"OrderId":            oid,
			"SupplierId":         supID,
			"RetailerId":         retID,
			"Status":             "COMPLETED",
			"OrderSource":        "MANUAL",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      []byte(`[]`),
			"TotalMinor":         int64(150000 + i*1000),
			"OriginalTotalMinor": int64(150000 + i*1000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          ts,
			"UpdatedAt":          ts,
		}))
	}
	if _, err := client.Apply(ctx, muts); err != nil {
		t.Fatalf("seed Orders: %v", err)
	}

	exports := NewSpannerExportRepository(client)
	sftp := NewSpannerSftpConfigRepository(client)
	svc := NewService(NewSpannerKeyRepository(client), NewSpannerWebhookRepository(client), nil, nil, nil)
	svc.SetExportRepos(exports, sftp)
	worker := NewExportWorker(exports, sftp, client, nil)
	worker.batchSize = 4 // force 3 keyset-paginated loops in ReadOnlyTransaction

	var jobID string
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var del []*spanner.Mutation
		for _, oid := range orderIDs {
			del = append(del, spanner.Delete("Orders", spanner.Key{oid}))
		}
		if jobID != "" {
			del = append(del, spanner.Delete("PartnerExportJobs", spanner.Key{jobID}))
		}
		_, _ = client.Apply(cleanCtx, del)
	})

	from2020 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to2026 := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	p := Principal{TenantType: TenantSupplier, TenantID: supID, Scopes: []string{ScopeExportsRead}}
	j, err := svc.CreateExportJobWithOptions(ctx, p, ExportResourceOrders, ExportFormatXLSX, &from2020, &to2026, true)
	if err != nil {
		t.Fatalf("CreateExportJobWithOptions: %v", err)
	}
	jobID = j.JobID

	if n, err := worker.RunOnce(ctx); err != nil || n < 1 {
		t.Fatalf("worker.RunOnce n=%d err=%v", n, err)
	}
	got, _, err := svc.GetExportJob(ctx, p, jobID)
	if err != nil {
		t.Fatalf("GetExportJob: %v", err)
	}
	if got.Status != ExportStatusSucceeded || got.RowCount != 11 {
		t.Fatalf("got job = %+v, want SUCCEEDED with RowCount=11 across 2021..2026", got)
	}
	f, err := excelize.OpenFile(filepath.Join(root, filepath.FromSlash(got.ObjectPath)))
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer f.Close()
	rows, err := f.GetRows("Export")
	if err != nil || len(rows) != 12 {
		t.Fatalf("xlsx rows = %d err=%v, want 12 (header + 11 orders)", len(rows), err)
	}
}
