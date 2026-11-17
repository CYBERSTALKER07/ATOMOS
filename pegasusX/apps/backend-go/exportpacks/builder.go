package exportpacks

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type builtArtifact struct {
	Meta ArtifactMeta
	Data []byte
}

type builtBundle struct {
	Manifest      *PackManifest
	ManifestBytes []byte
	Artifacts     map[string]builtArtifact
}

func formatWatermark(actor, tenantID, packID string, expiresAt time.Time) string {
	a := strings.TrimSpace(actor)
	if a == "" {
		a = "authorized-user"
	}
	return fmt.Sprintf("Prepared for %s | Tenant %s | Pack %s | Expires %s",
		a, strings.TrimSpace(tenantID), strings.TrimSpace(packID), expiresAt.UTC().Format(time.RFC3339))
}

func buildPackBundle(pack *ExportPack, ds *PackDatasets, signingSecret string) (*builtBundle, error) {
	if ds == nil {
		ds = &PackDatasets{}
	}
	stats := ds.Stats()
	var bankData *BankCreditScheduleData
	if pack != nil && pack.PackType == PackTypeBank {
		bankData = ComputeBankCreditPack(pack, ds)
		stats.BankCreditTapeRows = int64(len(bankData.Tape))
		stats.DpdBucketRows = int64(len(bankData.DPDBuckets))
		stats.DebtorConcentrationRows = int64(len(bankData.Concentration))
		stats.LossRecoveryCurveRows = int64(len(bankData.LossRecoveryCurves))
	}

	xlsxBytes, err := buildPackXLSX(pack, ds, stats, bankData)
	if err != nil {
		return nil, fmt.Errorf("build pack xlsx: %w", err)
	}
	perTableCSV, combinedCSVBytes, err := buildPackCSVs(pack, ds, bankData)
	if err != nil {
		return nil, fmt.Errorf("build pack csv: %w", err)
	}
	xmlBytes, err := buildOneCXML(pack, ds)
	if err != nil {
		return nil, fmt.Errorf("build pack 1c xml: %w", err)
	}

	var fileRefs []ManifestFileRef
	addRef := func(path, format string, data []byte) {
		sum := sha256.Sum256(data)
		fileRefs = append(fileRefs, ManifestFileRef{
			Path:        path,
			Format:      format,
			ContentType: ContentTypeForFormat(format),
			SizeBytes:   int64(len(data)),
			Sha256Hex:   hex.EncodeToString(sum[:]),
		})
	}
	addRef("pack.xlsx", FormatXLSX, xlsxBytes)
	addRef("pack.csv", FormatCSV, combinedCSVBytes)
	for _, tf := range perTableCSV {
		addRef(tf.path, FormatCSV, tf.data)
	}
	addRef("1c_journals.xml", FormatXML, xmlBytes)

	manifest := &PackManifest{
		SchemaVersion: "atomos.export_pack.v1",
		PackID:        pack.PackID,
		PackType:      pack.PackType,
		TenantID:      pack.TenantID,
		EntityID:      pack.EntityID,
		PeriodID:      pack.PeriodID,
		FromDate:      pack.FromDate.UTC(),
		ToDate:        pack.ToDate.UTC(),
		WatermarkText: pack.WatermarkText,
		CreatedBy:     pack.CreatedBy,
		CreatedAt:     pack.CreatedAt.UTC(),
		ExpiresAt:     pack.ExpiresAt.UTC(),
		TotalRows:     stats.Total(),
		Datasets:      stats,
		Files:         fileRefs,
	}
	if bankData != nil {
		sumCopy := bankData.Summary
		manifest.BankCredit = &sumCopy
	}

	if err := signManifest(manifest, signingSecret); err != nil {
		return nil, err
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}

	zipFiles := make([]zipEntry, 0, len(perTableCSV)+4)
	zipFiles = append(zipFiles,
		zipEntry{name: "manifest.json", data: manifestBytes},
		zipEntry{name: "pack.xlsx", data: xlsxBytes},
		zipEntry{name: "pack.csv", data: combinedCSVBytes},
		zipEntry{name: "1c_journals.xml", data: xmlBytes},
	)
	for _, tf := range perTableCSV {
		zipFiles = append(zipFiles, zipEntry{name: tf.path, data: tf.data})
	}
	zipBytes, err := buildZipArchive(zipFiles, pack.WatermarkText)
	if err != nil {
		return nil, fmt.Errorf("build pack zip: %w", err)
	}

	artifacts := map[string]builtArtifact{
		FormatZip:  makeArtifact(pack.PackID, FormatZip, zipBytes),
		FormatXLSX: makeArtifact(pack.PackID, FormatXLSX, xlsxBytes),
		FormatCSV:  makeArtifact(pack.PackID, FormatCSV, combinedCSVBytes),
		FormatXML:  makeArtifact(pack.PackID, FormatXML, xmlBytes),
		FormatJSON: makeArtifact(pack.PackID, FormatJSON, manifestBytes),
	}

	return &builtBundle{
		Manifest:      manifest,
		ManifestBytes: manifestBytes,
		Artifacts:     artifacts,
	}, nil
}

func makeArtifact(packID, format string, data []byte) builtArtifact {
	sum := sha256.Sum256(data)
	return builtArtifact{
		Meta: ArtifactMeta{
			Format:      format,
			Filename:    packID + "." + format,
			ContentType: ContentTypeForFormat(format),
			SizeBytes:   int64(len(data)),
			Sha256Hex:   hex.EncodeToString(sum[:]),
		},
		Data: data,
	}
}

func signManifest(m *PackManifest, secret string) error {
	m.SignatureAlg = ""
	m.SignatureHex = ""
	sec := strings.TrimSpace(secret)
	if sec != "" {
		m.SignatureAlg = "HMAC-SHA256"
	} else {
		m.SignatureAlg = "SHA256-DIGEST"
	}
	canonical, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("canonical manifest marshal: %w", err)
	}
	if sec != "" {
		mac := hmac.New(sha256.New, []byte(sec))
		_, _ = mac.Write(canonical)
		m.SignatureHex = hex.EncodeToString(mac.Sum(nil))
		return nil
	}
	sum := sha256.Sum256(canonical)
	m.SignatureHex = hex.EncodeToString(sum[:])
	return nil
}

func writeStreamSheet(f *excelize.File, sheetName, watermark string, headers []string, rows [][]any) error {
	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	if err := f.SetHeaderFooter(sheetName, &excelize.HeaderFooterOptions{
		OddHeader: "&L" + watermark,
	}); err != nil {
		return err
	}
	_ = idx
	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return err
	}
	hdrCells := make([]any, len(headers))
	for i, h := range headers {
		hdrCells[i] = h
	}
	if err := sw.SetRow("A1", hdrCells); err != nil {
		return err
	}
	for rIdx, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, rIdx+2)
		if err != nil {
			return err
		}
		safeRow := make([]any, len(row))
		for cIdx, val := range row {
			if sv, ok := val.(string); ok {
				safeRow[cIdx] = safeSpreadsheetCell(sv)
			} else {
				safeRow[cIdx] = val
			}
		}
		if err := sw.SetRow(cell, safeRow); err != nil {
			return err
		}
	}
	return sw.Flush()
}

// safeSpreadsheetCell neutralizes CSV/XLSX formula injection while leaving negative integers intact.
func safeSpreadsheetCell(s string) string {
	if s == "" {
		return ""
	}
	switch s[0] {
	case '=', '+', '@', '\t', '\r':
		return "'" + s
	case '-':
		if _, err := strconv.ParseInt(s, 10, 64); err == nil {
			return s
		}
		return "'" + s
	default:
		return s
	}
}

func sanitizeCSVRow(row []string) []string {
	out := make([]string, len(row))
	for i, c := range row {
		out[i] = safeSpreadsheetCell(c)
	}
	return out
}

func buildPackXLSX(pack *ExportPack, ds *PackDatasets, stats PackDatasetStats, bankData *BankCreditScheduleData) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", "Overview"); err != nil {
		return nil, err
	}
	_ = f.SetHeaderFooter("Overview", &excelize.HeaderFooterOptions{OddHeader: "&L" + pack.WatermarkText})

	sw, err := f.NewStreamWriter("Overview")
	if err != nil {
		return nil, err
	}
	overviewRows := [][]any{
		{"Watermark", pack.WatermarkText},
		{"PackID", pack.PackID},
		{"PackType", pack.PackType},
		{"TenantID", pack.TenantID},
		{"EntityID", pack.EntityID},
		{"PeriodID", pack.PeriodID},
		{"FromDate", pack.FromDate.UTC().Format(time.RFC3339)},
		{"ToDate", pack.ToDate.UTC().Format(time.RFC3339)},
		{"ExpiresAt", pack.ExpiresAt.UTC().Format(time.RFC3339)},
		{"CreatedBy", pack.CreatedBy},
		{"TotalRows", stats.Total()},
		{"GlAccounts", stats.GlAccounts},
		{"JournalEntries", stats.JournalEntries},
		{"JournalPostings", stats.JournalPostings},
		{"ArInvoices", stats.ArInvoices},
		{"InventoryCostLayers", stats.InventoryCostLayers},
		{"InventoryMovements", stats.InventoryMovements},
		{"EsfDocuments", stats.EsfDocuments},
		{"CashierBlindCountSessions", stats.CashierBlindCountSessions},
		{"DepotVaultLedgerEntries", stats.DepotVaultLedgerEntries},
		{"PaymentLedgerEntries", stats.PaymentLedgerEntries},
		{"ApprovalRequests", stats.ApprovalRequests},
		{"ApprovalDecisions", stats.ApprovalDecisions},
		{"AuditChainAnchors", stats.AuditChainAnchors},
		{"ReconciliationRuns", stats.ReconciliationRuns},
		{"ReconciliationBreaks", stats.ReconciliationBreaks},
	}
	if bankData != nil {
		overviewRows = append(overviewRows,
			[]any{"BankCreditTapeRows", stats.BankCreditTapeRows},
			[]any{"DpdBucketRows", stats.DpdBucketRows},
			[]any{"DebtorConcentrationRows", stats.DebtorConcentrationRows},
			[]any{"LossRecoveryCurveRows", stats.LossRecoveryCurveRows},
			[]any{"BankTotalExposureMinor", bankData.Summary.TotalExposureMinor},
			[]any{"BankTop1ConcentrationBps", bankData.Summary.Top1ConcentrationBps},
			[]any{"BankConsentVerifiedRetailers", bankData.Summary.ConsentVerifiedRetailers},
		)
	}
	for i, row := range overviewRows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return nil, err
		}
		if err := sw.SetRow(cell, row); err != nil {
			return nil, err
		}
	}
	if err := sw.Flush(); err != nil {
		return nil, err
	}

	specs := datasetTableSpecs(ds)
	if bankData != nil {
		specs = append(specs, bankTableSpecs(bankData)...)
	}
	for _, sp := range specs {
		rowsAny := make([][]any, len(sp.rows))
		for i, strRow := range sp.rows {
			r := make([]any, len(strRow))
			for j, val := range strRow {
				r[j] = val
			}
			rowsAny[i] = r
		}
		if err := writeStreamSheet(f, sp.sheetName, pack.WatermarkText, sp.headers, rowsAny); err != nil {
			return nil, fmt.Errorf("sheet %s: %w", sp.sheetName, err)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type tableSpec struct {
	name      string
	sheetName string
	csvPath   string
	headers   []string
	rows      [][]string
}

type csvEntry struct {
	path string
	data []byte
}

func buildPackCSVs(pack *ExportPack, ds *PackDatasets, bankData *BankCreditScheduleData) ([]csvEntry, []byte, error) {
	watermark := pack.WatermarkText
	specs := datasetTableSpecs(ds)
	if bankData != nil {
		specs = append(specs, bankTableSpecs(bankData)...)
	}
	entries := make([]csvEntry, 0, len(specs))

	var combined bytes.Buffer
	cw := csv.NewWriter(&combined)
	if err := cw.Write([]string{"dataset", "record_id", "occurred_at", "entity_id", "account_or_ref", "direction_or_status", "amount_minor", "currency", "details_json", "watermark"}); err != nil {
		return nil, nil, err
	}

	for _, sp := range specs {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		if err := w.Write(sanitizeCSVRow(append(append([]string(nil), sp.headers...), "watermark"))); err != nil {
			return nil, nil, err
		}
		for _, r := range sp.rows {
			if err := w.Write(sanitizeCSVRow(append(append([]string(nil), r...), watermark))); err != nil {
				return nil, nil, err
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, nil, err
		}
		entries = append(entries, csvEntry{path: sp.csvPath, data: buf.Bytes()})
	}

	for _, row := range combinedNormalizedRows(watermark, ds) {
		if err := cw.Write(sanitizeCSVRow(row)); err != nil {
			return nil, nil, err
		}
	}
	if bankData != nil {
		for _, row := range bankCombinedNormalizedRows(pack.EntityID, watermark, bankData) {
			if err := cw.Write(sanitizeCSVRow(row)); err != nil {
				return nil, nil, err
			}
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return nil, nil, err
	}
	return entries, combined.Bytes(), nil
}

func fmtTimeOpt(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func i64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func datasetTableSpecs(ds *PackDatasets) []tableSpec {
	glRows := make([][]string, len(ds.GlAccounts))
	for i, r := range ds.GlAccounts {
		glRows[i] = []string{r.EntityID, r.AccountCode, r.Class, r.Name, r.ParentCode, r.Currency, strconv.FormatBool(r.IsControl), r.CountryPack, fmtTimeOpt(r.ActiveFrom), fmtTimeOpt(r.ActiveTo), r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	jeRows := make([][]string, len(ds.JournalEntries))
	for i, r := range ds.JournalEntries {
		jeRows[i] = []string{r.EntityID, r.EntryID, r.PeriodID, r.OriginalPeriodID, r.SourceType, r.SourceID, r.OrderID, r.Memo, r.Currency, r.CountryPack, r.ReversesEntryID, r.PrevHash, r.EntryHash, r.PostedBy, r.OccurredAt.UTC().Format(time.RFC3339), r.PostedAt.UTC().Format(time.RFC3339)}
	}
	jpRows := make([][]string, len(ds.JournalPostings))
	for i, r := range ds.JournalPostings {
		jpRows[i] = []string{r.EntityID, r.EntryID, i64(r.Line), r.AccountCode, r.Direction, i64(r.AmountMinor), r.Currency, i64(r.FunctionalAmountMinor), r.PartyType, r.PartyID, r.WarehouseID, r.ProductID, r.LotID, r.Description}
	}
	arRows := make([][]string, len(ds.ArInvoices))
	for i, r := range ds.ArInvoices {
		arRows[i] = []string{r.InvoiceID, r.SupplierID, r.RetailerID, r.OrderID, r.Status, i64(r.PrincipalMinor), i64(r.BalanceMinor), r.Currency, r.CreditLeaveAt.UTC().Format(time.RFC3339), r.DueAt.UTC().Format(time.RFC3339), i64(r.TermsDays), i64(r.GracePeriodDays), r.AgingBucket, i64(r.DunningStep), r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	clRows := make([][]string, len(ds.InventoryCostLayers))
	for i, r := range ds.InventoryCostLayers {
		clRows[i] = []string{r.EntityID, r.WarehouseID, r.ProductID, r.LayerID, r.LotID, r.ReceiptRefType, r.ReceiptRefID, i64(r.QtyRemaining), i64(r.OriginalQty), i64(r.UnitCostMinor), r.Currency, r.ExpiryDate, r.ReceivedAt.UTC().Format(time.RFC3339)}
	}
	mvRows := make([][]string, len(ds.InventoryMovements))
	for i, r := range ds.InventoryMovements {
		mvRows[i] = []string{r.EntityID, r.MovementID, r.MovementType, r.ReferenceType, r.ReferenceID, r.WarehouseID, r.ProductID, r.LotID, r.LayerID, i64(r.QtyDelta), i64(r.UnitCostMinor), i64(r.TotalCostMinor), r.Currency, r.GlEntryID, r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	esfRows := make([][]string, len(ds.EsfDocuments))
	for i, r := range ds.EsfDocuments {
		esfRows[i] = []string{r.EntityID, r.EsfID, r.Direction, r.OrderID, r.PoID, r.CounterpartyTIN, r.CounterpartyName, i64(r.TotalMinor), i64(r.VatMinor), i64(r.VatRateBps), r.Status, r.Operator, r.OperatorRef, r.PeriodID, r.PayloadHash, fmtTimeOpt(r.IssuedAt), fmtTimeOpt(r.AcceptedAt), r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	cbRows := make([][]string, len(ds.CashierBlindCountSessions))
	for i, r := range ds.CashierBlindCountSessions {
		cbRows[i] = []string{r.SessionID, r.SupplierID, r.DepotID, r.DriverID, r.RouteID, r.BagSealBarcode, i64(r.AttemptNumber), i64(r.CountedCashMinor), i64(r.ExpectedCashMinor), i64(r.VarianceMinor), r.Status, r.CreatedAt.UTC().Format(time.RFC3339), fmtTimeOpt(r.ResolvedAt)}
	}
	dvRows := make([][]string, len(ds.DepotVaultLedgerEntries))
	for i, r := range ds.DepotVaultLedgerEntries {
		dvRows[i] = []string{r.VaultEntryID, r.DepotID, r.SupplierID, r.SessionID, r.CITManifestID, r.EntryType, i64(r.AmountMinor), i64(r.RunningBalanceMinor), r.CourierBadgeID, r.SealBarcode, r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	pmRows := make([][]string, len(ds.PaymentLedgerEntries))
	for i, r := range ds.PaymentLedgerEntries {
		pmRows[i] = []string{r.LedgerEntryID, r.SessionID, r.OrderID, r.SupplierID, r.RetailerID, r.Gateway, r.EntryType, i64(r.AmountMinor), r.Currency, r.ReferenceID, r.Source, r.OccurredAt.UTC().Format(time.RFC3339)}
	}
	apReqRows := make([][]string, len(ds.ApprovalRequests))
	for i, r := range ds.ApprovalRequests {
		apReqRows[i] = []string{r.TenantID, r.RequestID, r.PolicyID, i64(r.PolicyVersion), r.ObjectType, r.ObjectID, r.Action, i64(r.AmountMinor), r.RequestedBy, r.ActorType, r.Status, i64(r.CurrentStep), i64(r.RequiredApprovers), strconv.FormatBool(r.BreakGlass), r.PayloadHash, r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	apDecRows := make([][]string, len(ds.ApprovalDecisions))
	for i, r := range ds.ApprovalDecisions {
		apDecRows[i] = []string{r.TenantID, r.RequestID, r.DecisionID, i64(r.StepIndex), r.ApproverID, r.DelegatedFrom, r.Decision, r.Comment, strconv.FormatBool(r.MfaVerified), r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	ancRows := make([][]string, len(ds.AuditChainAnchors))
	for i, r := range ds.AuditChainAnchors {
		ancRows[i] = []string{r.TenantID, r.DayBucket, r.HeadEventID, i64(r.EventCount), r.RootHash, r.WormURI, r.SignerKeyID, r.AnchoredBy, r.AnchoredAt.UTC().Format(time.RFC3339)}
	}
	rrRows := make([][]string, len(ds.ReconciliationRuns))
	for i, r := range ds.ReconciliationRuns {
		rrRows[i] = []string{r.EntityID, r.RunID, r.PeriodID, r.Mode, r.Status, i64(r.BreakCount), i64(r.ExposureMinor), r.TriggeredBy, r.StartedAt.UTC().Format(time.RFC3339), fmtTimeOpt(r.CompletedAt)}
	}
	rbRows := make([][]string, len(ds.ReconciliationBreaks))
	for i, r := range ds.ReconciliationBreaks {
		rbRows[i] = []string{r.EntityID, r.BreakID, r.RunID, r.Dimension, r.ObjectType, r.ObjectID, i64(r.ExpectedMinor), i64(r.ActualMinor), i64(r.DeltaMinor), r.Currency, r.Severity, r.Status, r.OwnerID, r.ControlTowerExceptionID, r.CreatedAt.UTC().Format(time.RFC3339)}
	}

	return []tableSpec{
		{name: "gl_accounts", sheetName: "GL_Accounts", csvPath: "csv/gl_accounts.csv", headers: []string{"entity_id", "account_code", "class", "name", "parent_code", "currency", "is_control", "country_pack", "active_from", "active_to", "created_at"}, rows: glRows},
		{name: "journal_entries", sheetName: "Journal_Entries", csvPath: "csv/journal_entries.csv", headers: []string{"entity_id", "entry_id", "period_id", "original_period_id", "source_type", "source_id", "order_id", "memo", "currency", "country_pack", "reverses_entry_id", "prev_hash", "entry_hash", "posted_by", "occurred_at", "posted_at"}, rows: jeRows},
		{name: "journal_postings", sheetName: "Journal_Postings", csvPath: "csv/journal_postings.csv", headers: []string{"entity_id", "entry_id", "line", "account_code", "direction", "amount_minor", "currency", "functional_amount_minor", "party_type", "party_id", "warehouse_id", "product_id", "lot_id", "description"}, rows: jpRows},
		{name: "ar_invoices", sheetName: "AR_Invoices", csvPath: "csv/ar_invoices.csv", headers: []string{"invoice_id", "supplier_id", "retailer_id", "order_id", "status", "principal_minor", "balance_minor", "currency", "credit_leave_at", "due_at", "terms_days", "grace_period_days", "aging_bucket", "dunning_step", "created_at"}, rows: arRows},
		{name: "inventory_cost_layers", sheetName: "Inv_Cost_Layers", csvPath: "csv/inventory_cost_layers.csv", headers: []string{"entity_id", "warehouse_id", "product_id", "layer_id", "lot_id", "receipt_ref_type", "receipt_ref_id", "qty_remaining", "original_qty", "unit_cost_minor", "currency", "expiry_date", "received_at"}, rows: clRows},
		{name: "inventory_movements", sheetName: "Inv_Movements", csvPath: "csv/inventory_movements.csv", headers: []string{"entity_id", "movement_id", "movement_type", "reference_type", "reference_id", "warehouse_id", "product_id", "lot_id", "layer_id", "qty_delta", "unit_cost_minor", "total_cost_minor", "currency", "gl_entry_id", "created_at"}, rows: mvRows},
		{name: "esf_documents", sheetName: "ESF_Documents", csvPath: "csv/esf_documents.csv", headers: []string{"entity_id", "esf_id", "direction", "order_id", "po_id", "counterparty_tin", "counterparty_name", "total_minor", "vat_minor", "vat_rate_bps", "status", "operator", "operator_ref", "period_id", "payload_hash", "issued_at", "accepted_at", "created_at"}, rows: esfRows},
		{name: "cashier_blind_count_sessions", sheetName: "Cash_Blind_Counts", csvPath: "csv/cashier_blind_count_sessions.csv", headers: []string{"session_id", "supplier_id", "depot_id", "driver_id", "route_id", "bag_seal_barcode", "attempt_number", "counted_cash_minor", "expected_cash_minor", "variance_minor", "status", "created_at", "resolved_at"}, rows: cbRows},
		{name: "depot_vault_ledger_entries", sheetName: "Depot_Vault_Ledger", csvPath: "csv/depot_vault_ledger_entries.csv", headers: []string{"vault_entry_id", "depot_id", "supplier_id", "session_id", "cit_manifest_id", "entry_type", "amount_minor", "running_balance_minor", "courier_badge_id", "seal_barcode", "created_at"}, rows: dvRows},
		{name: "payment_ledger_entries", sheetName: "Payment_Ledger", csvPath: "csv/payment_ledger_entries.csv", headers: []string{"ledger_entry_id", "session_id", "order_id", "supplier_id", "retailer_id", "gateway", "entry_type", "amount_minor", "currency", "reference_id", "source", "occurred_at"}, rows: pmRows},
		{name: "approval_requests", sheetName: "Approval_Requests", csvPath: "csv/approval_requests.csv", headers: []string{"tenant_id", "request_id", "policy_id", "policy_version", "object_type", "object_id", "action", "amount_minor", "requested_by", "actor_type", "status", "current_step", "required_approvers", "break_glass", "payload_hash", "created_at"}, rows: apReqRows},
		{name: "approval_decisions", sheetName: "Approval_Decisions", csvPath: "csv/approval_decisions.csv", headers: []string{"tenant_id", "request_id", "decision_id", "step_index", "approver_id", "delegated_from", "decision", "comment", "mfa_verified", "created_at"}, rows: apDecRows},
		{name: "audit_chain_anchors", sheetName: "Audit_Anchors", csvPath: "csv/audit_chain_anchors.csv", headers: []string{"tenant_id", "day_bucket", "head_event_id", "event_count", "root_hash", "worm_uri", "signer_key_id", "anchored_by", "anchored_at"}, rows: ancRows},
		{name: "reconciliation_runs", sheetName: "Recon_Runs", csvPath: "csv/reconciliation_runs.csv", headers: []string{"entity_id", "run_id", "period_id", "mode", "status", "break_count", "exposure_minor", "triggered_by", "started_at", "completed_at"}, rows: rrRows},
		{name: "reconciliation_breaks", sheetName: "Recon_Breaks", csvPath: "csv/reconciliation_breaks.csv", headers: []string{"entity_id", "break_id", "run_id", "dimension", "object_type", "object_id", "expected_minor", "actual_minor", "delta_minor", "currency", "severity", "status", "owner_id", "control_tower_exception_id", "created_at"}, rows: rbRows},
	}
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func combinedNormalizedRows(watermark string, ds *PackDatasets) [][]string {
	var out [][]string
	for _, r := range ds.GlAccounts {
		out = append(out, []string{"gl_accounts", r.AccountCode, r.CreatedAt.UTC().Format(time.RFC3339), r.EntityID, r.AccountCode, r.Class, "0", r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.JournalEntries {
		out = append(out, []string{"journal_entries", r.EntryID, r.OccurredAt.UTC().Format(time.RFC3339), r.EntityID, r.SourceID, r.SourceType, "0", r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.JournalPostings {
		out = append(out, []string{"journal_postings", fmt.Sprintf("%s:%d", r.EntryID, r.Line), "", r.EntityID, r.AccountCode, r.Direction, i64(r.AmountMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.ArInvoices {
		out = append(out, []string{"ar_invoices", r.InvoiceID, r.CreatedAt.UTC().Format(time.RFC3339), r.SupplierID, r.OrderID, r.Status, i64(r.BalanceMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.InventoryCostLayers {
		out = append(out, []string{"inventory_cost_layers", r.LayerID, r.ReceivedAt.UTC().Format(time.RFC3339), r.EntityID, r.ProductID, r.WarehouseID, i64(r.QtyRemaining * r.UnitCostMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.InventoryMovements {
		out = append(out, []string{"inventory_movements", r.MovementID, r.CreatedAt.UTC().Format(time.RFC3339), r.EntityID, r.ProductID, r.MovementType, i64(r.TotalCostMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.EsfDocuments {
		out = append(out, []string{"esf_documents", r.EsfID, r.CreatedAt.UTC().Format(time.RFC3339), r.EntityID, r.CounterpartyTIN, r.Status, i64(r.TotalMinor), "", jsonString(r), watermark})
	}
	for _, r := range ds.CashierBlindCountSessions {
		out = append(out, []string{"cashier_blind_count_sessions", r.SessionID, r.CreatedAt.UTC().Format(time.RFC3339), r.SupplierID, r.BagSealBarcode, r.Status, i64(r.CountedCashMinor), "", jsonString(r), watermark})
	}
	for _, r := range ds.DepotVaultLedgerEntries {
		out = append(out, []string{"depot_vault_ledger_entries", r.VaultEntryID, r.CreatedAt.UTC().Format(time.RFC3339), r.SupplierID, r.DepotID, r.EntryType, i64(r.AmountMinor), "", jsonString(r), watermark})
	}
	for _, r := range ds.PaymentLedgerEntries {
		out = append(out, []string{"payment_ledger_entries", r.LedgerEntryID, r.OccurredAt.UTC().Format(time.RFC3339), r.SupplierID, r.Gateway, r.EntryType, i64(r.AmountMinor), r.Currency, jsonString(r), watermark})
	}
	for _, r := range ds.ApprovalRequests {
		out = append(out, []string{"approval_requests", r.RequestID, r.CreatedAt.UTC().Format(time.RFC3339), r.TenantID, r.ObjectID, r.Status, i64(r.AmountMinor), "", jsonString(r), watermark})
	}
	for _, r := range ds.ApprovalDecisions {
		out = append(out, []string{"approval_decisions", r.DecisionID, r.CreatedAt.UTC().Format(time.RFC3339), r.TenantID, r.RequestID, r.Decision, "0", "", jsonString(r), watermark})
	}
	for _, r := range ds.AuditChainAnchors {
		out = append(out, []string{"audit_chain_anchors", r.DayBucket, r.AnchoredAt.UTC().Format(time.RFC3339), r.TenantID, r.HeadEventID, "ANCHORED", i64(r.EventCount), "", jsonString(r), watermark})
	}
	for _, r := range ds.ReconciliationRuns {
		out = append(out, []string{"reconciliation_runs", r.RunID, r.StartedAt.UTC().Format(time.RFC3339), r.EntityID, r.PeriodID, r.Status, i64(r.ExposureMinor), "", jsonString(r), watermark})
	}
	for _, r := range ds.ReconciliationBreaks {
		out = append(out, []string{"reconciliation_breaks", r.BreakID, r.CreatedAt.UTC().Format(time.RFC3339), r.EntityID, r.ObjectID, r.Status, i64(r.DeltaMinor), r.Currency, jsonString(r), watermark})
	}
	return out
}

type oneCExportDoc struct {
	XMLName        xml.Name      `xml:"AtomosOneCExport"`
	SchemaVersion  string        `xml:"schema_version,attr"`
	PackID         string        `xml:"pack_id,attr"`
	PackType       string        `xml:"pack_type,attr"`
	EntityID       string        `xml:"entity_id,attr"`
	PeriodID       string        `xml:"period_id,attr,omitempty"`
	FromDate       string        `xml:"from_date,attr"`
	ToDate         string        `xml:"to_date,attr"`
	Watermark      string        `xml:"watermark,attr"`
	GlAccounts     []oneCAccount `xml:"GlAccounts>GlAccount"`
	JournalEntries []oneCEntry   `xml:"JournalEntries>JournalEntry"`
	EsfDocuments   []oneCEsfDoc  `xml:"EsfDocuments>EsfDocument"`
}

type oneCAccount struct {
	AccountCode string `xml:"account_code,attr"`
	Class       string `xml:"class,attr"`
	Currency    string `xml:"currency,attr"`
	Name        string `xml:",chardata"`
}

type oneCEntry struct {
	EntryID    string        `xml:"entry_id,attr"`
	PeriodID   string        `xml:"period_id,attr"`
	OccurredAt string        `xml:"occurred_at,attr"`
	SourceType string        `xml:"source_type,attr"`
	SourceID   string        `xml:"source_id,attr"`
	Currency   string        `xml:"currency,attr"`
	EntryHash  string        `xml:"entry_hash,attr"`
	PostedBy   string        `xml:"posted_by,attr"`
	Memo       string        `xml:"Memo,omitempty"`
	Postings   []oneCPosting `xml:"Postings>Posting"`
}

type oneCPosting struct {
	Line                  int64  `xml:"line,attr"`
	AccountCode           string `xml:"account_code,attr"`
	Direction             string `xml:"direction,attr"`
	AmountMinor           int64  `xml:"amount_minor,attr"`
	FunctionalAmountMinor int64  `xml:"functional_amount_minor,attr"`
	Currency              string `xml:"currency,attr"`
	PartyID               string `xml:"party_id,attr,omitempty"`
	Description           string `xml:",chardata"`
}

type oneCEsfDoc struct {
	EsfID           string `xml:"esf_id,attr"`
	Direction       string `xml:"direction,attr"`
	CounterpartyTIN string `xml:"counterparty_tin,attr"`
	Status          string `xml:"status,attr"`
	TotalMinor      int64  `xml:"total_minor,attr"`
	VatMinor        int64  `xml:"vat_minor,attr"`
	PeriodID        string `xml:"period_id,attr"`
}

func buildOneCXML(pack *ExportPack, ds *PackDatasets) ([]byte, error) {
	postingsByEntry := make(map[string][]oneCPosting)
	for _, p := range ds.JournalPostings {
		postingsByEntry[p.EntryID] = append(postingsByEntry[p.EntryID], oneCPosting{
			Line:                  p.Line,
			AccountCode:           p.AccountCode,
			Direction:             p.Direction,
			AmountMinor:           p.AmountMinor,
			FunctionalAmountMinor: p.FunctionalAmountMinor,
			Currency:              p.Currency,
			PartyID:               p.PartyID,
			Description:           p.Description,
		})
	}

	doc := oneCExportDoc{
		SchemaVersion: "atomos.1c.xml.v1",
		PackID:        pack.PackID,
		PackType:      pack.PackType,
		EntityID:      pack.EntityID,
		PeriodID:      pack.PeriodID,
		FromDate:      pack.FromDate.UTC().Format(time.RFC3339),
		ToDate:        pack.ToDate.UTC().Format(time.RFC3339),
		Watermark:     pack.WatermarkText,
	}
	for _, a := range ds.GlAccounts {
		doc.GlAccounts = append(doc.GlAccounts, oneCAccount{
			AccountCode: a.AccountCode,
			Class:       a.Class,
			Currency:    a.Currency,
			Name:        a.Name,
		})
	}
	for _, je := range ds.JournalEntries {
		doc.JournalEntries = append(doc.JournalEntries, oneCEntry{
			EntryID:    je.EntryID,
			PeriodID:   je.PeriodID,
			OccurredAt: je.OccurredAt.UTC().Format(time.RFC3339),
			SourceType: je.SourceType,
			SourceID:   je.SourceID,
			Currency:   je.Currency,
			EntryHash:  je.EntryHash,
			PostedBy:   je.PostedBy,
			Memo:       je.Memo,
			Postings:   postingsByEntry[je.EntryID],
		})
	}
	for _, esf := range ds.EsfDocuments {
		doc.EsfDocuments = append(doc.EsfDocuments, oneCEsfDoc{
			EsfID:           esf.EsfID,
			Direction:       esf.Direction,
			CounterpartyTIN: esf.CounterpartyTIN,
			Status:          esf.Status,
			TotalMinor:      esf.TotalMinor,
			VatMinor:        esf.VatMinor,
			PeriodID:        esf.PeriodID,
		})
	}

	raw, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), raw...), nil
}

type zipEntry struct {
	name string
	data []byte
}

func buildZipArchive(entries []zipEntry, comment string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if comment != "" {
		_ = zw.SetComment(comment)
	}
	for _, e := range entries {
		fh := &zip.FileHeader{
			Name:     e.name,
			Method:   zip.Deflate,
			Modified: time.Now().UTC(),
		}
		w, err := zw.CreateHeader(fh)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write(e.data); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
