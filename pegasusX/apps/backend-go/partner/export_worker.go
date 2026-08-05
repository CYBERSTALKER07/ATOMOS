package partner

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/storage"
	pxstorage "github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"google.golang.org/api/iterator"
)

// PartnerExportsEnabled gates the export worker and APIs.
func PartnerExportsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PARTNER_EXPORTS_ENABLED")))
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// PartnerSFTPEnabled gates SFTP upload after export.
func PartnerSFTPEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PARTNER_SFTP_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func partnerExportLocalRoot() string {
	return strings.TrimSpace(os.Getenv("PARTNER_EXPORT_LOCAL_ROOT"))
}

// ExportWorker processes PENDING PartnerExportJobs.
type ExportWorker struct {
	exports ExportRepository
	sftp    SftpConfigRepository
	spanner *spanner.Client
	log     *slog.Logger
	now     func() time.Time
	// SecretLoader resolves SecretRef → password/private key material.
	SecretLoader func(secretRef string) (string, error)
	// Uploader optional; defaults to UploadSFTP.
	Uploader func(ctx context.Context, cfg SftpConfig, secret, localPath, remoteName string) error
}

func NewExportWorker(exports ExportRepository, sftp SftpConfigRepository, client *spanner.Client, log *slog.Logger) *ExportWorker {
	if log == nil {
		log = slog.Default()
	}
	return &ExportWorker{
		exports:      exports,
		sftp:         sftp,
		spanner:      client,
		log:          log,
		now:          func() time.Time { return time.Now().UTC() },
		SecretLoader: LoadSecretRef,
		Uploader:     UploadSFTP,
	}
}

// Start loops until cancel.
func (w *ExportWorker) Start(ctx context.Context, interval time.Duration) {
	if w == nil || !PartnerExportsEnabled() {
		return
	}
	if interval <= 0 {
		interval = 20 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				w.log.Warn("partner export worker tick failed", "err", err)
			}
		}
	}
}

// RunOnce processes up to N pending jobs.
func (w *ExportWorker) RunOnce(ctx context.Context) (int, error) {
	if w == nil || w.exports == nil || !PartnerExportsEnabled() {
		return 0, nil
	}
	jobs, err := w.exports.ListPending(ctx, 10)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, j := range jobs {
		if err := w.processJob(ctx, j); err != nil {
			w.log.Warn("partner export job failed", "job_id", j.JobID, "err", err)
		} else {
			n++
		}
	}
	return n, nil
}

func (w *ExportWorker) processJob(ctx context.Context, j ExportJob) error {
	j.Status = ExportStatusRunning
	_ = w.exports.UpdateJob(ctx, j)

	rows, err := w.buildRows(ctx, j)
	if err != nil {
		return w.failJob(ctx, j, err.Error())
	}
	body, contentType, err := encodeExport(rows, j.Format)
	if err != nil {
		return w.failJob(ctx, j, err.Error())
	}
	ext := exportFileExt(j.Format)
	objectPath := fmt.Sprintf("partner-exports/%s/%s/%s.%s",
		strings.ToLower(j.TenantType), j.TenantID, j.JobID, ext)

	if err := w.writeObject(ctx, objectPath, body, contentType); err != nil {
		return w.failJob(ctx, j, "write_failed:"+err.Error())
	}

	j.ObjectPath = objectPath
	j.RowCount = int64(len(rows))
	j.SftpStatus = SftpStatusSkipped

	if PartnerSFTPEnabled() && w.sftp != nil {
		cfg, ok, err := w.sftp.Get(ctx, j.TenantType, j.TenantID)
		if err == nil && ok && cfg.IsActive {
			secret, err := w.SecretLoader(cfg.SecretRef)
			if err != nil || secret == "" {
				j.SftpStatus = SftpStatusFailed
				j.Error = "sftp_secret_unavailable"
			} else {
				localPath, err := w.ensureLocalFile(objectPath, body)
				if err != nil {
					j.SftpStatus = SftpStatusFailed
					j.Error = "sftp_local_copy_failed"
				} else {
					remoteName := filepath.Base(objectPath)
					up := w.Uploader
					if up == nil {
						up = UploadSFTP
					}
					if err := up(ctx, cfg, secret, localPath, remoteName); err != nil {
						j.SftpStatus = SftpStatusFailed
						j.Error = "sftp_upload_failed:" + err.Error()
						w.log.Warn("sftp upload failed", "job_id", j.JobID, "err", err)
					} else {
						j.SftpStatus = SftpStatusUploaded
					}
				}
			}
		}
	}

	now := w.now()
	j.Status = ExportStatusSucceeded
	j.FinishedAt = &now
	return w.exports.UpdateJob(ctx, j)
}

func (w *ExportWorker) failJob(ctx context.Context, j ExportJob, msg string) error {
	now := w.now()
	j.Status = ExportStatusFailed
	j.Error = msg
	j.FinishedAt = &now
	_ = w.exports.UpdateJob(ctx, j)
	return fmt.Errorf("%s", msg)
}

func (w *ExportWorker) writeObject(ctx context.Context, objectPath string, body []byte, contentType string) error {
	if root := partnerExportLocalRoot(); root != "" {
		full := filepath.Join(root, filepath.FromSlash(objectPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, body, 0o644)
	}
	if pxstorage.Client != nil && pxstorage.BucketName != "" {
		wc := pxstorage.Client.Bucket(pxstorage.BucketName).Object(objectPath).NewWriter(ctx)
		wc.ContentType = contentType
		if _, err := wc.Write(body); err != nil {
			_ = wc.Close()
			return err
		}
		return wc.Close()
	}
	// Fallback: temp under /tmp for memory-mode tests
	full := filepath.Join(os.TempDir(), "pegasusx-partner-exports", filepath.FromSlash(objectPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, body, 0o644)
}

func (w *ExportWorker) ensureLocalFile(objectPath string, body []byte) (string, error) {
	if root := partnerExportLocalRoot(); root != "" {
		return filepath.Join(root, filepath.FromSlash(objectPath)), nil
	}
	full := filepath.Join(os.TempDir(), "pegasusx-partner-exports", filepath.FromSlash(objectPath))
	if _, err := os.Stat(full); err == nil {
		return full, nil
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		return "", err
	}
	return full, nil
}

// SignDownloadURL returns a short-lived GET URL for a succeeded export object.
func SignDownloadURL(objectPath string) (string, error) {
	objectPath = strings.TrimSpace(objectPath)
	if objectPath == "" {
		return "", fmt.Errorf("missing_object")
	}
	if root := partnerExportLocalRoot(); root != "" {
		return "file://" + filepath.Join(root, filepath.FromSlash(objectPath)), nil
	}
	if pxstorage.Client != nil && pxstorage.BucketName != "" {
		opts := &storage.SignedURLOptions{
			Scheme:  storage.SigningSchemeV4,
			Method:  "GET",
			Expires: time.Now().Add(15 * time.Minute),
		}
		return pxstorage.Client.Bucket(pxstorage.BucketName).SignedURL(objectPath, opts)
	}
	full := filepath.Join(os.TempDir(), "pegasusx-partner-exports", filepath.FromSlash(objectPath))
	return "file://" + full, nil
}

func exportFileExt(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case ExportFormatJSON:
		return ExportFormatJSON
	case ExportFormatXML:
		return ExportFormatXML
	default:
		return ExportFormatCSV
	}
}

func encodeExport(rows []map[string]any, format string) ([]byte, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case ExportFormatJSON:
		b, err := json.MarshalIndent(map[string]any{"rows": rows}, "", "  ")
		return b, "application/json", err
	case ExportFormatXML:
		return encodeExportXML(rows)
	default:
		return encodeExportCSV(rows)
	}
}

func encodeExportCSV(rows []map[string]any) ([]byte, string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	headers := csvHeadersForRows(rows)
	if err := w.Write(headers); err != nil {
		return nil, "", err
	}
	for _, row := range rows {
		rec := make([]string, len(headers))
		for i, h := range headers {
			rec[i] = fmt.Sprint(row[h])
		}
		if err := w.Write(rec); err != nil {
			return nil, "", err
		}
	}
	w.Flush()
	return buf.Bytes(), "text/csv", w.Error()
}

func csvHeadersForRows(rows []map[string]any) []string {
	// Prefer stable journal column order when present.
	if len(rows) > 0 {
		if _, ok := rows[0]["debit_account"]; ok {
			if _, ok2 := rows[0]["credit_account"]; ok2 {
				return append([]string(nil), journalColumnOrder...)
			}
		}
	}
	headers := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	if len(headers) == 0 {
		headers = []string{"empty"}
	}
	return headers
}

func encodeExportXML(rows []map[string]any) ([]byte, string, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<Journal version="1" dialect="1c">` + "\n")
	attrs := csvHeadersForRows(rows)
	if len(attrs) == 1 && attrs[0] == "empty" {
		attrs = append([]string(nil), journalColumnOrder...)
	}
	for _, row := range rows {
		buf.WriteString("  <Entry")
		for _, k := range attrs {
			buf.WriteString(" ")
			buf.WriteString(k)
			buf.WriteString(`="`)
			buf.WriteString(xmlAttrEscape(fmt.Sprint(row[k])))
			buf.WriteString(`"`)
		}
		buf.WriteString(" />\n")
	}
	buf.WriteString("</Journal>\n")
	return buf.Bytes(), "application/xml", nil
}

func xmlAttrEscape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func (w *ExportWorker) buildRows(ctx context.Context, j ExportJob) ([]map[string]any, error) {
	from, to := exportWindow(j)
	switch j.Resource {
	case ExportResourceOrders:
		return w.exportOrders(ctx, j.TenantType, j.TenantID, from, to)
	case ExportResourceInvoices:
		return w.exportInvoices(ctx, j.TenantType, j.TenantID, from, to)
	case ExportResourceInventory:
		return w.exportInventory(ctx, j.TenantType, j.TenantID)
	case ExportResourceLedger:
		return w.exportLedger(ctx, j.TenantType, j.TenantID, from, to)
	case ExportResourceJournals:
		return w.exportJournals(ctx, j.TenantType, j.TenantID, from, to)
	default:
		return nil, fmt.Errorf("invalid_resource")
	}
}

func exportWindow(j ExportJob) (time.Time, time.Time) {
	now := time.Now().UTC()
	to := now
	from := now.AddDate(0, 0, -7)
	if j.ToDate != nil {
		to = j.ToDate.UTC()
	}
	if j.FromDate != nil {
		from = j.FromDate.UTC()
	}
	if to.Before(from) {
		from, to = to, from
	}
	if to.Sub(from) > time.Duration(MaxExportWindowDays)*24*time.Hour {
		from = to.AddDate(0, 0, -MaxExportWindowDays)
	}
	return from, to
}

func (w *ExportWorker) exportOrders(ctx context.Context, tenantType, tenantID string, from, to time.Time) ([]map[string]any, error) {
	if w.spanner == nil {
		return []map[string]any{}, nil
	}
	var sql string
	params := map[string]any{"tid": tenantID, "from": from, "to": to, "lim": int64(MaxExportRows)}
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT OrderId, SupplierId, RetailerId, Status, COALESCE(OrderSource, ''), TotalMinor, Currency, CreatedAt
			FROM Orders WHERE SupplierId = @tid AND CreatedAt >= @from AND CreatedAt <= @to
			ORDER BY CreatedAt DESC LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT OrderId, SupplierId, RetailerId, Status, COALESCE(OrderSource, ''), TotalMinor, Currency, CreatedAt
			FROM Orders WHERE RetailerId = @tid AND CreatedAt >= @from AND CreatedAt <= @to
			ORDER BY CreatedAt DESC LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}
	iter := w.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var orderID, supplierID, retailerID, status, source, currency string
		var total int64
		var created time.Time
		if err := row.Columns(&orderID, &supplierID, &retailerID, &status, &source, &total, &currency, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"order_id": orderID, "supplier_id": supplierID, "retailer_id": retailerID,
			"status": status, "order_source": source, "total_minor": total,
			"currency": currency, "created_at": created.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (w *ExportWorker) exportInvoices(ctx context.Context, tenantType, tenantID string, from, to time.Time) ([]map[string]any, error) {
	if w.spanner == nil {
		return []map[string]any{}, nil
	}
	// Best-effort ArInvoices; empty on missing table/rows.
	var sql string
	params := map[string]any{"tid": tenantID, "from": from, "to": to, "lim": int64(MaxExportRows)}
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT InvoiceId, SupplierId, RetailerId, Status, PrincipalMinor, BalanceMinor, Currency, DueAt
			FROM ArInvoices WHERE SupplierId = @tid AND DueAt >= @from AND DueAt <= @to
			LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT InvoiceId, SupplierId, RetailerId, Status, PrincipalMinor, BalanceMinor, Currency, DueAt
			FROM ArInvoices WHERE RetailerId = @tid AND DueAt >= @from AND DueAt <= @to
			LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}
	iter := w.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Table may not exist / schema drift — return headers-only empty.
			return []map[string]any{}, nil
		}
		var invID, supplierID, retailerID, status, currency string
		var principal, balance int64
		var due time.Time
		if err := row.Columns(&invID, &supplierID, &retailerID, &status, &principal, &balance, &currency, &due); err != nil {
			return []map[string]any{}, nil
		}
		out = append(out, map[string]any{
			"invoice_id": invID, "supplier_id": supplierID, "retailer_id": retailerID,
			"status": status, "principal_minor": principal, "balance_minor": balance,
			"currency": currency, "due_at": due.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (w *ExportWorker) exportInventory(ctx context.Context, tenantType, tenantID string) ([]map[string]any, error) {
	if w.spanner == nil {
		return []map[string]any{}, nil
	}
	if tenantType == TenantRetailer {
		iter := w.spanner.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT Sku, COALESCE(SUM(OnHand), 0), COALESCE(SUM(Reserved), 0)
				FROM RetailerStockBalances WHERE RetailerId = @tid
				GROUP BY Sku LIMIT @lim`,
			Params: map[string]any{"tid": tenantID, "lim": int64(MaxExportRows)},
		})
		defer iter.Stop()
		out := make([]map[string]any, 0)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			var sku string
			var onHand, reserved int64
			if err := row.Columns(&sku, &onHand, &reserved); err != nil {
				return nil, err
			}
			out = append(out, map[string]any{"sku": sku, "on_hand": onHand, "reserved": reserved})
		}
		return out, nil
	}
	// Supplier: product catalog stock snapshot
	iter := w.spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProductId, COALESCE(Barcode, ProductId), COALESCE(StockQuantity, 0)
			FROM Products WHERE SupplierId = @tid AND IsActive = true LIMIT @lim`,
		Params: map[string]any{"tid": tenantID, "lim": int64(MaxExportRows)},
	})
	defer iter.Stop()
	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []map[string]any{}, nil
		}
		var pid, sku string
		var stock int64
		if err := row.Columns(&pid, &sku, &stock); err != nil {
			return []map[string]any{}, nil
		}
		out = append(out, map[string]any{"product_id": pid, "sku": sku, "stock_quantity": stock})
	}
	return out, nil
}

func (w *ExportWorker) exportLedger(ctx context.Context, tenantType, tenantID string, from, to time.Time) ([]map[string]any, error) {
	if w.spanner == nil {
		return []map[string]any{}, nil
	}
	var sql string
	params := map[string]any{"tid": tenantID, "from": from, "to": to, "lim": int64(MaxExportRows)}
	switch tenantType {
	case TenantSupplier:
		sql = `SELECT EntryId, InvoiceId, SupplierId, RetailerId, EntryType, AmountMinor, CreatedAt
			FROM ArLedgerEntries WHERE SupplierId = @tid AND CreatedAt >= @from AND CreatedAt <= @to
			ORDER BY CreatedAt DESC LIMIT @lim`
	case TenantRetailer:
		sql = `SELECT EntryId, InvoiceId, SupplierId, RetailerId, EntryType, AmountMinor, CreatedAt
			FROM ArLedgerEntries WHERE RetailerId = @tid AND CreatedAt >= @from AND CreatedAt <= @to
			ORDER BY CreatedAt DESC LIMIT @lim`
	default:
		return nil, fmt.Errorf("invalid_tenant")
	}
	iter := w.spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	out := make([]map[string]any, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return w.exportInvoices(ctx, tenantType, tenantID, from, to)
		}
		var entryID, invID, supplierID, retailerID, entryType string
		var amount int64
		var created time.Time
		if err := row.Columns(&entryID, &invID, &supplierID, &retailerID, &entryType, &amount, &created); err != nil {
			return w.exportInvoices(ctx, tenantType, tenantID, from, to)
		}
		out = append(out, map[string]any{
			"entry_id": entryID, "invoice_id": invID, "supplier_id": supplierID,
			"retailer_id": retailerID, "entry_type": entryType, "amount_minor": amount,
			"created_at": created.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}
