package periods

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"google.golang.org/api/iterator"
)

// SnapshotSink persists and retrieves immutable JSON+CSV period close snapshots.
type SnapshotSink interface {
	WriteSnapshot(ctx context.Context, snap CloseSnapshot, jsonBytes, csvBytes []byte) (uri string, err error)
	ReadSnapshot(ctx context.Context, uri string) ([]byte, error)
}

// GCSSnapshotSink stores frozen close snapshots in Google Cloud Storage with DoesNotExist precondition.
type GCSSnapshotSink struct {
	client *storage.Client
	bucket string
	prefix string
}

// NewGCSSnapshotSink constructs a create-only GCS sink for frozen close snapshots.
func NewGCSSnapshotSink(client *storage.Client, bucket, prefix string) (*GCSSnapshotSink, error) {
	bucket = strings.TrimSpace(bucket)
	if client == nil || bucket == "" {
		return nil, fmt.Errorf("%w: GCS client and bucket are required", ErrSnapshotWormRequired)
	}
	return &GCSSnapshotSink{
		client: client,
		bucket: bucket,
		prefix: strings.Trim(strings.TrimSpace(prefix), "/"),
	}, nil
}

func (g *GCSSnapshotSink) WriteSnapshot(ctx context.Context, snap CloseSnapshot, jsonBytes, csvBytes []byte) (string, error) {
	if g == nil || g.client == nil || g.bucket == "" {
		return "", ErrSnapshotWormRequired
	}
	shortHash := snap.SnapshotHash
	if len(shortHash) > 12 {
		shortHash = shortHash[:12]
	}
	ts := snap.GeneratedAt.UTC().Format("20060102T150405Z")
	base := fmt.Sprintf("period-snapshots/%s/%s-%s-%s",
		sanitizePathSeg(snap.EntityID),
		sanitizePathSeg(snap.PeriodID),
		ts,
		shortHash,
	)
	if g.prefix != "" {
		base = g.prefix + "/" + base
	}
	jsonObj := base + ".json"
	csvObj := base + ".csv"
	if err := g.writeCreateOnly(ctx, jsonObj, "application/json", jsonBytes); err != nil {
		return "", err
	}
	if err := g.writeCreateOnly(ctx, csvObj, "text/csv", csvBytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("gs://%s/%s", g.bucket, jsonObj), nil
}

func (g *GCSSnapshotSink) writeCreateOnly(ctx context.Context, objectName, contentType string, data []byte) error {
	obj := g.client.Bucket(g.bucket).Object(objectName).If(storage.Conditions{DoesNotExist: true})
	w := obj.NewWriter(ctx)
	w.ContentType = contentType
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("gcs write snapshot %s: %w", objectName, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("gcs close snapshot %s: %w", objectName, err)
	}
	return nil
}

func (g *GCSSnapshotSink) ReadSnapshot(ctx context.Context, uri string) ([]byte, error) {
	if g == nil || g.client == nil {
		return nil, ErrSnapshotWormRequired
	}
	trimmed := strings.TrimPrefix(strings.TrimSpace(uri), "gs://")
	slash := strings.IndexByte(trimmed, '/')
	if slash <= 0 {
		return nil, fmt.Errorf("%w: invalid GCS snapshot URI %q", ErrSnapshotNotFound, uri)
	}
	bucket := trimmed[:slash]
	objName := trimmed[slash+1:]
	rc, err := g.client.Bucket(bucket).Object(objName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: read GCS snapshot %s: %v", ErrSnapshotNotFound, uri, err)
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 64<<20))
}

// LocalSnapshotSink persists immutable 0444 JSON+CSV close snapshots in a local directory (non-production only).
type LocalSnapshotSink struct {
	rootDir string
}

// NewLocalSnapshotSink initializes a local read-only snapshot sink rooted at dir.
// Refuses initialization in production environments where auth.IsProduction() is true.
func NewLocalSnapshotSink(dir string) (*LocalSnapshotSink, error) {
	if auth.IsProduction() {
		return nil, fmt.Errorf("%w: local close snapshot storage disallowed in production", ErrSnapshotWormRequired)
	}
	clean := strings.TrimSpace(dir)
	if clean == "" {
		clean = filepath.Join(os.TempDir(), "pegasusx-period-snapshots")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return nil, fmt.Errorf("resolve local snapshot dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir local snapshot dir: %w", err)
	}
	return &LocalSnapshotSink{rootDir: abs}, nil
}

func (l *LocalSnapshotSink) WriteSnapshot(_ context.Context, snap CloseSnapshot, jsonBytes, csvBytes []byte) (string, error) {
	if auth.IsProduction() {
		return "", fmt.Errorf("%w: local close snapshot write disallowed in production", ErrSnapshotWormRequired)
	}
	if l == nil || strings.TrimSpace(l.rootDir) == "" {
		return "", ErrSnapshotWormRequired
	}
	shortHash := snap.SnapshotHash
	if len(shortHash) > 12 {
		shortHash = shortHash[:12]
	}
	ts := snap.GeneratedAt.UTC().Format("20060102T150405Z")
	dir := filepath.Join(l.rootDir, sanitizePathSeg(snap.EntityID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir snapshot entity dir: %w", err)
	}
	base := fmt.Sprintf("%s-%s-%s", sanitizePathSeg(snap.PeriodID), ts, shortHash)
	jsonPath := filepath.Join(dir, base+".json")
	csvPath := filepath.Join(dir, base+".csv")

	if err := writeReadOnlyExclusive(jsonPath, jsonBytes); err != nil {
		return "", err
	}
	if err := writeReadOnlyExclusive(csvPath, csvBytes); err != nil {
		return "", err
	}
	return "file://" + jsonPath, nil
}

func (l *LocalSnapshotSink) ReadSnapshot(_ context.Context, uri string) ([]byte, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(uri), "file://")
	if trimmed == "" {
		return nil, ErrSnapshotNotFound
	}
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSnapshotNotFound, err)
	}
	return raw, nil
}

func writeReadOnlyExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o444)
	if err != nil {
		return fmt.Errorf("create immutable snapshot file %s: %w", path, err)
	}
	_, wErr := f.Write(data)
	cErr := f.Close()
	if wErr != nil {
		return fmt.Errorf("write snapshot file %s: %w", path, wErr)
	}
	if cErr != nil {
		return fmt.Errorf("close snapshot file %s: %w", path, cErr)
	}
	_ = os.Chmod(path, 0o444)
	return nil
}

// FailClosedSnapshotSink fails every write with ErrSnapshotWormRequired (used in prod when GCS is unconfigured).
type FailClosedSnapshotSink struct{}

func (FailClosedSnapshotSink) WriteSnapshot(_ context.Context, _ CloseSnapshot, _, _ []byte) (string, error) {
	return "", fmt.Errorf("%w: GCS snapshot bucket is required in production", ErrSnapshotWormRequired)
}

func (FailClosedSnapshotSink) ReadSnapshot(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrSnapshotNotFound
}

// DefaultSnapshotSink returns an environment-appropriate SnapshotSink:
// GCS if bucket+client are provided, LocalSnapshotSink in dev/test, or fail-closed in production.
func DefaultSnapshotSink(gcsClient *storage.Client) SnapshotSink {
	bucket := strings.TrimSpace(os.Getenv("PERIOD_SNAPSHOT_GCS_BUCKET"))
	prefix := strings.TrimSpace(os.Getenv("PERIOD_SNAPSHOT_GCS_PREFIX"))
	if gcsClient != nil && bucket != "" {
		if sink, err := NewGCSSnapshotSink(gcsClient, bucket, prefix); err == nil {
			return sink
		}
	}
	if auth.IsProduction() {
		return FailClosedSnapshotSink{}
	}
	dir := strings.TrimSpace(os.Getenv("PERIOD_SNAPSHOT_WORM_DIR"))
	sink, err := NewLocalSnapshotSink(dir)
	if err != nil {
		return FailClosedSnapshotSink{}
	}
	return sink
}

// WriteCloseSnapshot reads consistent close figures for (entityID, periodID), seals canonical JSON
// with SHA-256 + companion CSV, and writes immutable snapshot artifacts via sink.
func WriteCloseSnapshot(
	ctx context.Context,
	repo Repository,
	sink SnapshotSink,
	entityID, periodID, tzName, closedBy string,
) (CloseSnapshot, error) {
	if repo == nil {
		return CloseSnapshot{}, ErrRepoUnavailable
	}
	if sink == nil {
		sink = DefaultSnapshotSink(nil)
	}
	snap, err := repo.BuildCloseSnapshotData(ctx, entityID, periodID, tzName, closedBy)
	if err != nil {
		return CloseSnapshot{}, err
	}
	// Hash canonical JSON before setting SnapshotURI so SHA-256 matches exact stored JSON bytes.
	snap.SnapshotURI = ""
	snap.SnapshotHash = ""
	jsonBytes, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return CloseSnapshot{}, fmt.Errorf("marshal close snapshot JSON: %w", err)
	}
	sum := sha256.Sum256(jsonBytes)
	snap.SnapshotHash = hex.EncodeToString(sum[:])

	csvBytes := FormatSnapshotCSV(snap)
	uri, err := sink.WriteSnapshot(ctx, snap, jsonBytes, csvBytes)
	if err != nil {
		return CloseSnapshot{}, err
	}
	snap.SnapshotURI = uri
	return snap, nil
}

// ReadCloseSnapshot retrieves a previously persisted CloseSnapshot JSON artifact and verifies its SHA-256 digest.
func ReadCloseSnapshot(ctx context.Context, sink SnapshotSink, p Period) (CloseSnapshot, error) {
	if strings.TrimSpace(p.SnapshotURI) == "" {
		return CloseSnapshot{}, ErrSnapshotNotFound
	}
	var raw []byte
	var err error
	if sink != nil {
		raw, err = sink.ReadSnapshot(ctx, p.SnapshotURI)
	} else if strings.HasPrefix(p.SnapshotURI, "file://") {
		raw, err = os.ReadFile(strings.TrimPrefix(p.SnapshotURI, "file://"))
	} else {
		return CloseSnapshot{}, ErrSnapshotWormRequired
	}
	if err != nil {
		return CloseSnapshot{}, err
	}
	if expected := strings.TrimSpace(p.SnapshotHash); expected != "" {
		sum := sha256.Sum256(raw)
		actual := hex.EncodeToString(sum[:])
		if !strings.EqualFold(actual, expected) {
			return CloseSnapshot{}, fmt.Errorf("%w: snapshot SHA-256 mismatch (got %s want %s)", ErrSnapshotWormRequired, actual, expected)
		}
	}
	var snap CloseSnapshot
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&snap); err != nil {
		return CloseSnapshot{}, fmt.Errorf("decode snapshot JSON: %w", err)
	}
	snap.SnapshotURI = p.SnapshotURI
	snap.SnapshotHash = p.SnapshotHash
	return snap, nil
}

// FormatSnapshotCSV renders CloseSnapshot as deterministic UTF-8 CSV covering Trial Balance,
// AR summary, Inventory Valuation, and VAT reconciliation.
func FormatSnapshotCSV(snap CloseSnapshot) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"section", "key", "subkey", "currency", "qty", "debit_minor", "credit_minor", "net_minor", "metadata"})
	_ = w.Write([]string{
		"META", snap.EntityID, snap.PeriodID, snap.Currency, "",
		strconv.FormatInt(snap.TrialBalance.TotalDebitsMinor, 10),
		strconv.FormatInt(snap.TrialBalance.TotalCreditsMinor, 10),
		"0",
		fmt.Sprintf("timezone=%s;sha256=%s", snap.Timezone, snap.SnapshotHash),
	})
	for _, line := range snap.TrialBalance.Accounts {
		_ = w.Write([]string{
			"TRIAL_BALANCE", line.AccountCode, line.Class, line.Currency, "",
			strconv.FormatInt(line.DebitMinor, 10),
			strconv.FormatInt(line.CreditMinor, 10),
			strconv.FormatInt(line.NetMinor, 10),
			line.AccountName,
		})
	}
	_ = w.Write([]string{
		"AR_SUMMARY", "open_receivables", "", snap.ARSummary.Currency,
		strconv.FormatInt(snap.ARSummary.OpenInvoices, 10),
		strconv.FormatInt(snap.ARSummary.OpenAmountMinor, 10),
		"0",
		strconv.FormatInt(snap.ARSummary.OpenAmountMinor, 10),
		fmt.Sprintf("total=%d;overdue=%d", snap.ARSummary.TotalInvoices, snap.ARSummary.OverdueInvoices),
	})
	_ = w.Write([]string{
		"VALUATION_SUMMARY", "total_inventory", "", snap.ValuationSummary.Currency,
		strconv.FormatInt(snap.ValuationSummary.TotalQuantity, 10),
		strconv.FormatInt(snap.ValuationSummary.TotalValueMinor, 10),
		"0",
		strconv.FormatInt(snap.ValuationSummary.TotalValueMinor, 10),
		fmt.Sprintf("layers=%d", snap.ValuationSummary.LayerCount),
	})
	for _, vl := range snap.ValuationSummary.Lines {
		_ = w.Write([]string{
			"VALUATION_LINE", vl.WarehouseID, vl.ProductID, vl.Currency,
			strconv.FormatInt(vl.Quantity, 10),
			strconv.FormatInt(vl.ValueMinor, 10),
			"0",
			strconv.FormatInt(vl.ValueMinor, 10),
			"",
		})
	}
	_ = w.Write([]string{
		"VAT_SUMMARY", snap.VATSummary.PeriodID, "gl_vs_esf", snap.VATSummary.Currency,
		strconv.FormatInt(snap.VATSummary.ESFCount, 10),
		strconv.FormatInt(snap.VATSummary.GLInputVATMinor, 10),
		strconv.FormatInt(snap.VATSummary.GLOutputVATMinor, 10),
		strconv.FormatInt(snap.VATSummary.VarianceMinor, 10),
		fmt.Sprintf("gl_net=%d;esf_net=%d", snap.VATSummary.GLNetVATMinor, snap.VATSummary.ESFNetVATMinor),
	})
	w.Flush()
	return buf.Bytes()
}

func buildCloseSnapshotSpanner(
	ctx context.Context,
	client *spanner.Client,
	entityID, periodID, tzName, closedBy string,
) (CloseSnapshot, error) {
	if client == nil {
		return CloseSnapshot{}, ErrRepoUnavailable
	}
	entityID = strings.TrimSpace(entityID)
	_, _, normPeriod, err := ValidatePeriodID(periodID)
	if err != nil {
		return CloseSnapshot{}, err
	}
	periodID = normPeriod
	if strings.TrimSpace(tzName) == "" {
		resolvedTZ, _, tzErr := ResolveEntityTimezone(ctx, entityID, "")
		if tzErr != nil {
			return CloseSnapshot{}, tzErr
		}
		tzName = resolvedTZ
	}
	startUTC, endUTC, _, resolvedTZ, err := ComputePeriodWindow(periodID, tzName)
	if err != nil {
		return CloseSnapshot{}, err
	}

	ro := client.ReadOnlyTransaction()
	defer ro.Close()

	// 1. Trial Balance from GlBalances + GlAccounts.
	tb, err := readTrialBalanceRO(ctx, ro, entityID, periodID)
	if err != nil {
		return CloseSnapshot{}, err
	}

	// 2. AR Summary from ArInvoices as of endUTC.
	arSum, err := readARSummaryRO(ctx, ro, entityID, tb.Currency, endUTC)
	if err != nil {
		return CloseSnapshot{}, err
	}

	// 3. Inventory Valuation Summary from InventoryCostLayers (int64 minor units).
	valSum, err := readValuationSummaryRO(ctx, ro, entityID, tb.Currency)
	if err != nil {
		return CloseSnapshot{}, err
	}

	// 4. VAT Summary comparing GL VAT accounts vs accepted EsfDocuments.
	vatSum, err := readVATSummaryRO(ctx, ro, entityID, periodID, tb)
	if err != nil {
		return CloseSnapshot{}, err
	}

	return CloseSnapshot{
		EntityID:         entityID,
		PeriodID:         periodID,
		Timezone:         resolvedTZ,
		PeriodStartUTC:   startUTC,
		PeriodEndUTC:     endUTC,
		Currency:         tb.Currency,
		ClosedBy:         strings.TrimSpace(closedBy),
		GeneratedAt:      time.Now().UTC().Truncate(time.Microsecond),
		TrialBalance:     tb,
		ARSummary:        arSum,
		ValuationSummary: valSum,
		VATSummary:       vatSum,
	}, nil
}

func readTrialBalanceRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, entityID, periodID string) (gl.TrialBalanceReport, error) {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT b.AccountCode, b.Currency, SUM(b.DebitMinor) AS DebitMinor, SUM(b.CreditMinor) AS CreditMinor,
		             a.Class, a.Name
		      FROM GlBalances b
		      LEFT JOIN GlAccounts a ON a.EntityId = b.EntityId AND a.AccountCode = b.AccountCode
		      WHERE b.EntityId = @entityId AND b.PeriodId = @periodId
		      GROUP BY b.AccountCode, b.Currency, a.Class, a.Name
		      ORDER BY b.AccountCode ASC`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	})
	defer iter.Stop()

	lines := make([]gl.TrialBalanceLine, 0)
	var totalDebits, totalCredits int64
	curr := ""
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return gl.TrialBalanceReport{}, fmt.Errorf("snapshot read GlBalances: %w", err)
		}
		var line gl.TrialBalanceLine
		var classNull, nameNull spanner.NullString
		if err := row.Columns(&line.AccountCode, &line.Currency, &line.DebitMinor, &line.CreditMinor, &classNull, &nameNull); err != nil {
			return gl.TrialBalanceReport{}, fmt.Errorf("snapshot scan GlBalances: %w", err)
		}
		if classNull.Valid {
			line.Class = strings.ToUpper(strings.TrimSpace(classNull.StringVal))
		}
		if nameNull.Valid {
			line.AccountName = nameNull.StringVal
		}
		uc := strings.ToUpper(strings.TrimSpace(line.AccountCode))
		if line.Class == "" {
			if def, ok := gl.LookupSystemAccount(uc); ok {
				line.Class = def.Class
				if line.AccountName == "" {
					line.AccountName = def.Name
				}
			}
		}
		if gl.IsDebitNormalClass(line.Class) {
			line.NetMinor = line.DebitMinor - line.CreditMinor
		} else {
			line.NetMinor = line.CreditMinor - line.DebitMinor
		}
		totalDebits += line.DebitMinor
		totalCredits += line.CreditMinor
		if curr == "" {
			curr = line.Currency
		}
		lines = append(lines, line)
	}
	if curr == "" {
		if c, err := auth.CurrencyFromContext(ctx, entityID); err == nil && strings.TrimSpace(c) != "" {
			curr = c
		} else {
			curr = gl.DefaultCurrencyForPack("")
		}
	}
	return gl.TrialBalanceReport{
		EntityID:          entityID,
		PeriodID:          periodID,
		Currency:          curr,
		TotalDebitsMinor:  totalDebits,
		TotalCreditsMinor: totalCredits,
		Balanced:          totalDebits == totalCredits,
		Accounts:          lines,
	}, nil
}

func readARSummaryRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, entityID, defaultCurrency string, asOf time.Time) (SnapshotARSummary, error) {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT Status, BalanceMinor, Currency, DueAt
		      FROM ArInvoices
		      WHERE SupplierId = @entityId AND CreatedAt < @asOf`,
		Params: map[string]any{
			"entityId": entityID,
			"asOf":     asOf,
		},
	})
	defer iter.Stop()

	out := SnapshotARSummary{Currency: defaultCurrency}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return SnapshotARSummary{}, fmt.Errorf("snapshot read ArInvoices: %w", err)
		}
		var st, curr string
		var bal int64
		var dueAt time.Time
		if err := row.Columns(&st, &bal, &curr, &dueAt); err != nil {
			return SnapshotARSummary{}, fmt.Errorf("snapshot scan ArInvoices: %w", err)
		}
		out.TotalInvoices++
		if out.Currency == "" && strings.TrimSpace(curr) != "" {
			out.Currency = strings.ToUpper(strings.TrimSpace(curr))
		}
		st = strings.ToUpper(strings.TrimSpace(st))
		if bal > 0 && st != "PAID" && st != "WRITTEN_OFF" && st != "CANCELLED" {
			out.OpenInvoices++
			out.OpenAmountMinor += bal
			if st == "OVERDUE" || dueAt.Before(asOf) {
				out.OverdueInvoices++
			}
		}
	}
	return out, nil
}

func readValuationSummaryRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, entityID, defaultCurrency string) (SnapshotValuationSummary, error) {
	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT WarehouseId, ProductId, Currency, QtyRemaining, UnitCostMinor
		      FROM InventoryCostLayers
		      WHERE EntityId = @entityId AND QtyRemaining > 0
		      ORDER BY WarehouseId ASC, ProductId ASC`,
		Params: map[string]any{
			"entityId": entityID,
		},
	})
	defer iter.Stop()

	type key struct{ wh, prod, curr string }
	grouped := make(map[key]*SnapshotValuationLine)
	var order []key
	out := SnapshotValuationSummary{
		Currency: defaultCurrency,
		Lines:    make([]SnapshotValuationLine, 0),
	}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return SnapshotValuationSummary{}, fmt.Errorf("snapshot read InventoryCostLayers: %w", err)
		}
		var wh, prod, curr string
		var qty, unitCost int64
		if err := row.Columns(&wh, &prod, &curr, &qty, &unitCost); err != nil {
			return SnapshotValuationSummary{}, fmt.Errorf("snapshot scan InventoryCostLayers: %w", err)
		}
		curr = strings.ToUpper(strings.TrimSpace(curr))
		if curr == "" {
			curr = defaultCurrency
		}
		if out.Currency == "" {
			out.Currency = curr
		}
		valMinor := qty * unitCost
		out.LayerCount++
		out.TotalQuantity += qty
		out.TotalValueMinor += valMinor
		k := key{wh: wh, prod: prod, curr: curr}
		line, exists := grouped[k]
		if !exists {
			line = &SnapshotValuationLine{
				WarehouseID: wh,
				ProductID:   prod,
				Currency:    curr,
			}
			grouped[k] = line
			order = append(order, k)
		}
		line.Quantity += qty
		line.ValueMinor += valMinor
	}
	for _, k := range order {
		out.Lines = append(out.Lines, *grouped[k])
	}
	return out, nil
}

func readVATSummaryRO(ctx context.Context, ro *spanner.ReadOnlyTransaction, entityID, periodID string, tb gl.TrialBalanceReport) (SnapshotVATSummary, error) {
	vat := SnapshotVATSummary{
		PeriodID: periodID,
		Currency: tb.Currency,
	}
	for _, line := range tb.Accounts {
		switch strings.ToUpper(strings.TrimSpace(line.AccountCode)) {
		case gl.AccountVatOutput:
			vat.GLOutputVATMinor += line.CreditMinor - line.DebitMinor
		case gl.AccountVatInput:
			vat.GLInputVATMinor += line.DebitMinor - line.CreditMinor
		}
	}
	vat.GLNetVATMinor = vat.GLOutputVATMinor - vat.GLInputVATMinor

	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT Direction, VatMinor
		      FROM EsfDocuments
		      WHERE EntityId = @entityId
		        AND PeriodId = @periodId
		        AND UPPER(Status) = 'ACCEPTED'`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	})
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return SnapshotVATSummary{}, fmt.Errorf("snapshot read EsfDocuments: %w", err)
		}
		var dir string
		var vatMinor int64
		if err := row.Columns(&dir, &vatMinor); err != nil {
			return SnapshotVATSummary{}, fmt.Errorf("snapshot scan EsfDocuments: %w", err)
		}
		vat.ESFCount++
		switch strings.ToUpper(strings.TrimSpace(dir)) {
		case "OUTBOUND":
			vat.ESFOutboundVATMinor += vatMinor
		case "INBOUND":
			vat.ESFInboundVATMinor += vatMinor
		}
	}
	vat.ESFNetVATMinor = vat.ESFOutboundVATMinor - vat.ESFInboundVATMinor
	vat.VarianceMinor = vat.GLNetVATMinor - vat.ESFNetVATMinor
	return vat, nil
}

func sanitizePathSeg(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
