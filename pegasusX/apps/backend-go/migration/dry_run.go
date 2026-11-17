package migration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

var (
	// ErrDryRunExpired indicates the dry-run preview exceeded the 24h freshness window.
	ErrDryRunExpired = errors.New("dry_run_report_expired")
	// ErrDryRunStale indicates uploaded file SHA256, mapping hash, or apply_mode changed since dry-run.
	ErrDryRunStale = errors.New("dry_run_fingerprint_stale")
	// ErrDryRunBlockingConflicts indicates blocking validation/financial errors remain unresolved.
	ErrDryRunBlockingConflicts = errors.New("dry_run_has_blocking_conflicts")
	// ErrSetOnHandConfirmationRequired indicates explicit double-confirmation is needed for SET_ON_HAND stock mode.
	ErrSetOnHandConfirmationRequired = errors.New("set_on_hand_confirmation_required")
)

const dryRunRowPageSize = 500

// LiveWarehouse represents an existing warehouse in Spanner or virtual preceding pack.
type LiveWarehouse struct {
	WarehouseID string
	ExternalID  string
	Name        string
	IsDefault   bool
}

// LiveProduct represents an existing catalog SKU in Spanner or virtual preceding pack.
type LiveProduct struct {
	ProductID         string
	SKU               string
	Barcode           string
	Name              string
	Unit              string
	PriceMinor        int64
	Currency          string
	RequiresColdChain bool
	IsPerishable      bool
}

// LiveCustomer represents an existing linked retailer or counterparty in Spanner or virtual overlay.
type LiveCustomer struct {
	RetailerID   string
	ExternalID   string
	INN          string
	Phone        string
	CustomerName string
	Currency     string
}

// LiveStock captures live warehouse x product inventory on hand & reserved counts.
type LiveStock struct {
	WarehouseID string
	ProductID   string
	OnHand      int64
	Reserved    int64
	Unit        string
}

// LiveLot captures a live or staged lot balance.
type LiveLot struct {
	LotID       string
	WarehouseID string
	ProductID   string
	LotCode     string
	ExpiryDate  time.Time
	OnHand      int64
	Reserved    int64
}

// LiveOrder captures an existing order keyed by ExternalOrderNo.
type LiveOrder struct {
	OrderID         string
	ExternalOrderNo string
	OrderSource     string
	Status          string
}

// LiveARInvoice captures an existing opening/AR invoice keyed by ExternalId.
type LiveARInvoice struct {
	InvoiceID    string
	ExternalID   string
	BalanceMinor int64
	Currency     string
	Status       string
}

// LiveOverlaySnapshot holds read-only state loaded from live Spanner before dry-run simulation.
type LiveOverlaySnapshot struct {
	SupplierCurrency   string
	HomeCell           string
	DefaultWarehouseID string
	WarehousesByExt    map[string]LiveWarehouse
	WarehousesByName   map[string]LiveWarehouse
	ProductsByID       map[string]LiveProduct
	ProductsBySKU      map[string]LiveProduct
	ProductsByBarcode  map[string]LiveProduct
	ProductsByName     map[string]LiveProduct
	CustomersByExt     map[string]LiveCustomer
	CustomersByINN     map[string]LiveCustomer
	CustomersByPhone   map[string]LiveCustomer
	CustomersByName    map[string]LiveCustomer
	StockByWHProduct   map[string]LiveStock
	StockTotalByProd   map[string]int64
	LotsByKey          map[string]LiveLot
	OrdersByExtNo      map[string]LiveOrder
	ArInvoicesByExt    map[string]LiveARInvoice
	WarehouseLocks     []WarehouseLock
}

// NewEmptyLiveOverlay returns an initialized non-nil LiveOverlaySnapshot with default UZS currency.
func NewEmptyLiveOverlay() LiveOverlaySnapshot {
	return LiveOverlaySnapshot{
		SupplierCurrency:  "UZS",
		WarehousesByExt:   make(map[string]LiveWarehouse),
		WarehousesByName:  make(map[string]LiveWarehouse),
		ProductsByID:      make(map[string]LiveProduct),
		ProductsBySKU:     make(map[string]LiveProduct),
		ProductsByBarcode: make(map[string]LiveProduct),
		ProductsByName:    make(map[string]LiveProduct),
		CustomersByExt:    make(map[string]LiveCustomer),
		CustomersByINN:    make(map[string]LiveCustomer),
		CustomersByPhone:  make(map[string]LiveCustomer),
		CustomersByName:   make(map[string]LiveCustomer),
		StockByWHProduct:  make(map[string]LiveStock),
		StockTotalByProd:  make(map[string]int64),
		LotsByKey:         make(map[string]LiveLot),
		OrdersByExtNo:     make(map[string]LiveOrder),
		ArInvoicesByExt:   make(map[string]LiveARInvoice),
	}
}

func (s *LiveOverlaySnapshot) clone() LiveOverlaySnapshot {
	out := NewEmptyLiveOverlay()
	if strings.TrimSpace(s.SupplierCurrency) != "" {
		out.SupplierCurrency = strings.ToUpper(strings.TrimSpace(s.SupplierCurrency))
	}
	out.HomeCell = s.HomeCell
	out.DefaultWarehouseID = s.DefaultWarehouseID
	for k, v := range s.WarehousesByExt {
		out.WarehousesByExt[k] = v
	}
	for k, v := range s.WarehousesByName {
		out.WarehousesByName[k] = v
	}
	for k, v := range s.ProductsByID {
		out.ProductsByID[k] = v
	}
	for k, v := range s.ProductsBySKU {
		out.ProductsBySKU[k] = v
	}
	for k, v := range s.ProductsByBarcode {
		out.ProductsByBarcode[k] = v
	}
	for k, v := range s.ProductsByName {
		out.ProductsByName[k] = v
	}
	for k, v := range s.CustomersByExt {
		out.CustomersByExt[k] = v
	}
	for k, v := range s.CustomersByINN {
		out.CustomersByINN[k] = v
	}
	for k, v := range s.CustomersByPhone {
		out.CustomersByPhone[k] = v
	}
	for k, v := range s.CustomersByName {
		out.CustomersByName[k] = v
	}
	for k, v := range s.StockByWHProduct {
		out.StockByWHProduct[k] = v
	}
	for k, v := range s.StockTotalByProd {
		out.StockTotalByProd[k] = v
	}
	for k, v := range s.LotsByKey {
		out.LotsByKey[k] = v
	}
	for k, v := range s.OrdersByExtNo {
		out.OrdersByExtNo[k] = v
	}
	for k, v := range s.ArInvoicesByExt {
		out.ArInvoicesByExt[k] = v
	}
	out.WarehouseLocks = append([]WarehouseLock(nil), s.WarehouseLocks...)
	return out
}

// DryRunLiveOverlayReader loads read-only live Spanner entities for virtual dry-run evaluation.
type DryRunLiveOverlayReader interface {
	LoadLiveOverlay(ctx context.Context, supplierID string) (LiveOverlaySnapshot, error)
}

// DryRunOptions configures project-level or session-level non-mutating virtual dry-run evaluation.
type DryRunOptions struct {
	SupplierID string
	ProjectID  string
	SessionID  string // empty evaluates the entire project across topological packs
	ApplyMode  string // MERGE (default) | SET_ON_HAND
	Now        time.Time
	NewID      func() string
	LiveReader DryRunLiveOverlayReader
}

// ApproveSessionRequest captures explicit operator approval parameters for POST .../approve.
type ApproveSessionRequest struct {
	ApplyMode          string `json:"apply_mode,omitempty"`
	SetOnHandConfirmed bool   `json:"set_on_hand_confirmed,omitempty"`
	ExpectedReportID   string `json:"expected_report_id,omitempty"`
}

// EvaluateProjectDryRun simulates every staged session in project DAG topological order against Live U Staged Preceding Packs,
// persisting immutable MigrationReports with 24h TTL while making zero mutations to business tables.
func EvaluateProjectDryRun(ctx context.Context, repo Repository, opts DryRunOptions) (MigrationReport, DryRunSummary, error) {
	opts.SessionID = ""
	return evaluateDryRunInternal(ctx, repo, opts)
}

// EvaluateSessionDryRun simulates one target child session (along with virtual overlay from preceding staged packs),
// enforcing DAG upstream dependency presence and saving a fresh 24h dry-run report.
func EvaluateSessionDryRun(ctx context.Context, repo Repository, opts DryRunOptions) (MigrationReport, DryRunSummary, error) {
	if strings.TrimSpace(opts.SessionID) == "" {
		return MigrationReport{}, DryRunSummary{}, fmt.Errorf("%w: sessionId is required", ErrInvalidInput)
	}
	return evaluateDryRunInternal(ctx, repo, opts)
}

func evaluateDryRunInternal(ctx context.Context, repo Repository, opts DryRunOptions) (MigrationReport, DryRunSummary, error) {
	if repo == nil {
		return MigrationReport{}, DryRunSummary{}, fmt.Errorf("migration repository is required")
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	newID := opts.NewID
	if newID == nil {
		newID = func() string { return uuid.NewString() }
	}
	applyMode := NormalizeApplyMode(opts.ApplyMode)

	proj, sessions, live, err := loadDryRunInputs(ctx, repo, opts, now)
	if err != nil {
		return MigrationReport{}, DryRunSummary{}, err
	}
	if _, err := EvaluateProjectDAG(sessions); err != nil {
		return MigrationReport{}, DryRunSummary{}, err
	}

	var targetSession *Session
	if opts.SessionID != "" {
		for i := range sessions {
			if sessions[i].SessionID == opts.SessionID {
				targetSession = &sessions[i]
				break
			}
		}
		if targetSession == nil {
			return MigrationReport{}, DryRunSummary{}, ErrNotFound
		}
		if err := ValidatePackDryRunAllowed(targetSession.Pack, sessions); err != nil {
			return MigrationReport{}, DryRunSummary{}, err
		}
	}

	overlay := live.clone()
	if overlay.DefaultWarehouseID == "" && strings.TrimSpace(proj.DefaultWarehouseID) != "" {
		overlay.DefaultWarehouseID = strings.TrimSpace(proj.DefaultWarehouseID)
	}

	ordered := sortSessionsTopologically(sessions)
	summary := DryRunSummary{
		ProjectID:             opts.ProjectID,
		OpeningBalanceAccount: "EQUITY:OPENING_BALANCE",
		BalancedJournalCheck:  true,
		EvaluatedAt:           now,
		ExpiresAt:             now.Add(DryRunReportTTL),
	}

	var sessionFingerprints []string
	var targetReport *MigrationReport

	for _, sess := range ordered {
		st := NormalizeSessionStatus(string(sess.Status))
		if st == SessionStatusRolledBack {
			continue
		}
		// When dry-running a single child session, preceding packs that are already
		// applied/cutover (or quarantined) are already reflected in LoadLiveOverlay;
		// re-scanning tens of thousands of already-applied staged rows adds O(N) work for no delta.
		if targetSession != nil && sess.SessionID != targetSession.SessionID {
			if st == SessionStatusApplied || st == SessionStatusCutoverComplete {
				continue
			}
		}
		doc, hasDoc, err := repo.GetSessionMapping(ctx, opts.SupplierID, sess.SessionID)
		if err != nil {
			return MigrationReport{}, DryRunSummary{}, err
		}
		if !hasDoc {
			doc = SessionMappingDocument{SessionID: sess.SessionID, Pack: sess.Pack}
		}

		packRep, conflicts, debitDelta, creditDelta, err := simulateSessionPack(ctx, repo, opts.SupplierID, sess, doc, applyMode, &overlay, now)
		if err != nil {
			return MigrationReport{}, DryRunSummary{}, err
		}
		summary.JournalDebitMinor += debitDelta
		summary.JournalCreditMinor += creditDelta
		if summary.JournalDebitMinor != summary.JournalCreditMinor {
			summary.BalancedJournalCheck = false
		}

		if targetSession != nil && sess.SessionID != targetSession.SessionID {
			// Preceding pack contributed to virtual overlay only; do not emit final report row for sibling pack.
			continue
		}

		sessFP := ComputeSessionDryRunFingerprint(sess, doc, applyMode)
		sessionFingerprints = append(sessionFingerprints, sessFP)
		summary.Packs = append(summary.Packs, packRep)
		summary.Conflicts = append(summary.Conflicts, conflicts...)
		summary.ColdLotWarningCount += packRep.ColdLotWithoutLiveCount
		summary.OnHandBelowReservedHits += packRep.OnHandBelowReservedCount

		if targetSession == nil || sess.SessionID == targetSession.SessionID {
			rep, err := persistSessionDryRunReport(ctx, repo, opts.SupplierID, sess, doc, packRep, conflicts, sessFP, applyMode, summary, now, newID)
			if err != nil {
				return MigrationReport{}, DryRunSummary{}, err
			}
			if targetSession != nil && sess.SessionID == targetSession.SessionID {
				targetReport = &rep
			}
		}
	}

	if targetSession != nil {
		if targetReport == nil {
			return MigrationReport{}, DryRunSummary{}, ErrNotFound
		}
		summary.ContentFingerprint = targetReport.ContentFingerprint
		summary.ReadyToApprove = targetReport.BlockingCount == 0 && summary.BalancedJournalCheck
		return *targetReport, summary, nil
	}

	projFP := ComputeProjectDryRunFingerprint(sessionFingerprints)
	summary.ContentFingerprint = projFP
	var totalRows, totalValid, totalInvalid, totalCreate, totalUpdate, totalSkip, totalMoney, totalBlock, totalWarn int64
	for _, p := range summary.Packs {
		totalRows += p.RowsTotal
		totalValid += p.RowsValid
		totalInvalid += p.RowsInvalid
		totalCreate += p.WouldCreate
		totalUpdate += p.WouldUpdate
		totalSkip += p.WouldSkip
		totalMoney += p.MoneyImpactMinor
		totalBlock += p.BlockingCount
		totalWarn += p.WarningCount
	}
	summary.ReadyToApprove = totalBlock == 0 && summary.BalancedJournalCheck

	rawSummary, err := json.Marshal(summary)
	if err != nil {
		return MigrationReport{}, DryRunSummary{}, err
	}
	projReport := MigrationReport{
		SupplierID:         opts.SupplierID,
		ReportID:           newID(),
		ProjectID:          opts.ProjectID,
		ReportKind:         ReportKindProjectDryRun,
		ContentFingerprint: projFP,
		RowsTotal:          totalRows,
		RowsValid:          totalValid,
		RowsInvalid:        totalInvalid,
		WouldCreate:        totalCreate,
		WouldUpdate:        totalUpdate,
		WouldSkip:          totalSkip,
		MoneyImpactMinor:   totalMoney,
		Currency:           overlay.SupplierCurrency,
		BlockingCount:      totalBlock,
		WarningCount:       totalWarn,
		SummaryJSON:        rawSummary,
		ExpiresAt:          now.Add(DryRunReportTTL),
		CreatedAt:          now,
	}
	if err := repo.SaveMigrationReport(ctx, projReport); err != nil {
		return MigrationReport{}, DryRunSummary{}, err
	}
	if summary.ReadyToApprove && proj.Status != ProjectStatusCutoverComplete {
		if proj.Status == ProjectStatusDraft {
			_ = repo.UpdateProjectStatus(ctx, opts.SupplierID, opts.ProjectID, ProjectStatusStaging)
		}
		_ = repo.UpdateProjectStatus(ctx, opts.SupplierID, opts.ProjectID, ProjectStatusDryRunReady)
	}
	return projReport, summary, nil
}

func loadDryRunInputs(ctx context.Context, repo Repository, opts DryRunOptions, now time.Time) (Project, []Session, LiveOverlaySnapshot, error) {
	if opts.SessionID != "" && opts.ProjectID == "" {
		s, err := repo.GetSession(ctx, opts.SupplierID, opts.SessionID)
		if err != nil {
			return Project{}, nil, LiveOverlaySnapshot{}, err
		}
		opts.ProjectID = s.ProjectID
	}
	proj, err := repo.GetProject(ctx, opts.SupplierID, opts.ProjectID)
	if err != nil {
		return Project{}, nil, LiveOverlaySnapshot{}, err
	}
	sessions, err := repo.ListSessionsByProject(ctx, opts.SupplierID, opts.ProjectID)
	if err != nil {
		return Project{}, nil, LiveOverlaySnapshot{}, err
	}

	reader := opts.LiveReader
	if reader == nil {
		if r, ok := repo.(DryRunLiveOverlayReader); ok {
			reader = r
		}
	}
	var live LiveOverlaySnapshot
	if reader != nil {
		live, err = reader.LoadLiveOverlay(ctx, opts.SupplierID)
		if err != nil {
			return Project{}, nil, LiveOverlaySnapshot{}, err
		}
	} else {
		live = NewEmptyLiveOverlay()
	}
	if strings.TrimSpace(live.SupplierCurrency) == "" {
		live.SupplierCurrency = "UZS"
	}
	_ = now
	return proj, sessions, live, nil
}

func sortSessionsTopologically(sessions []Session) []Session {
	orderIdx := make(map[Pack]int, len(TopologicalPackOrder))
	for i, p := range TopologicalPackOrder {
		orderIdx[p] = i
	}
	out := append([]Session(nil), sessions...)
	for i := 1; i < len(out); i++ {
		cur := out[i]
		curRank := rankForPack(cur.Pack, orderIdx)
		j := i - 1
		for j >= 0 && rankForPack(out[j].Pack, orderIdx) > curRank {
			out[j+1] = out[j]
			j--
		}
		out[j+1] = cur
	}
	return out
}

func rankForPack(p Pack, idx map[Pack]int) int {
	if r, ok := idx[p]; ok {
		return r
	}
	return len(TopologicalPackOrder)
}

func persistSessionDryRunReport(
	ctx context.Context,
	repo Repository,
	supplierID string,
	sess Session,
	doc SessionMappingDocument,
	packRep PackDryRunReport,
	conflicts []DryRunConflict,
	sessFP, applyMode string,
	parentSummary DryRunSummary,
	now time.Time,
	newID func() string,
) (MigrationReport, error) {
	sessSummary := DryRunSummary{
		ProjectID:               sess.ProjectID,
		ContentFingerprint:      sessFP,
		Packs:                   []PackDryRunReport{packRep},
		Conflicts:               conflicts,
		BalancedJournalCheck:    parentSummary.BalancedJournalCheck,
		JournalDebitMinor:       parentSummary.JournalDebitMinor,
		JournalCreditMinor:      parentSummary.JournalCreditMinor,
		OpeningBalanceAccount:   "EQUITY:OPENING_BALANCE",
		ReadyToApprove:          packRep.BlockingCount == 0 && parentSummary.BalancedJournalCheck,
		ColdLotWarningCount:     packRep.ColdLotWithoutLiveCount,
		OnHandBelowReservedHits: packRep.OnHandBelowReservedCount,
		EvaluatedAt:             now,
		ExpiresAt:               now.Add(DryRunReportTTL),
	}
	raw, err := json.Marshal(sessSummary)
	if err != nil {
		return MigrationReport{}, err
	}
	rep := MigrationReport{
		SupplierID:         supplierID,
		ReportID:           newID(),
		ProjectID:          sess.ProjectID,
		SessionID:          sess.SessionID,
		ReportKind:         ReportKindDryRun,
		ContentFingerprint: sessFP,
		RowsTotal:          packRep.RowsTotal,
		RowsValid:          packRep.RowsValid,
		RowsInvalid:        packRep.RowsInvalid,
		WouldCreate:        packRep.WouldCreate,
		WouldUpdate:        packRep.WouldUpdate,
		WouldSkip:          packRep.WouldSkip,
		MoneyImpactMinor:   packRep.MoneyImpactMinor,
		Currency:           packRep.Currency,
		BlockingCount:      packRep.BlockingCount,
		WarningCount:       packRep.WarningCount,
		SummaryJSON:        raw,
		ExpiresAt:          now.Add(DryRunReportTTL),
		CreatedAt:          now,
	}
	if err := repo.SaveMigrationReport(ctx, rep); err != nil {
		return MigrationReport{}, err
	}

	curStatus := NormalizeSessionStatus(string(sess.Status))
	if curStatus != SessionStatusApplied && curStatus != SessionStatusCutoverComplete && curStatus != SessionStatusApproved {
		if err := repo.MarkSessionDryRunReady(ctx, supplierID, sess.SessionID, rep.ReportID, sessFP, applyMode); err != nil {
			return MigrationReport{}, err
		}
	}
	_ = doc
	return rep, nil
}

type stagedRowCursorReader interface {
	ListStagedRowsAfter(ctx context.Context, supplierID, sessionID string, afterRowIndex int64, limit int) ([]StagedRow, error)
}

func streamAllStagedRows(ctx context.Context, repo Repository, supplierID, sessionID string, afterRowIndex int64, pageSize int, fn func([]StagedRow) error) error {
	if pageSize <= 0 {
		pageSize = dryRunRowPageSize
	}
	if cr, ok := repo.(stagedRowCursorReader); ok {
		cursor := afterRowIndex
		for {
			rows, err := cr.ListStagedRowsAfter(ctx, supplierID, sessionID, cursor, pageSize)
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				return nil
			}
			if err := fn(rows); err != nil {
				return err
			}
			cursor = rows[len(rows)-1].RowIndex
			if len(rows) < pageSize {
				return nil
			}
		}
	}
	offset := 0
	for {
		rows, total, err := repo.ListStagedRows(ctx, supplierID, sessionID, pageSize, offset)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		if err := fn(rows); err != nil {
			return err
		}
		offset += len(rows)
		if int64(offset) >= total || len(rows) < pageSize {
			return nil
		}
	}
}

func simulateSessionPack(
	ctx context.Context,
	repo Repository,
	supplierID string,
	sess Session,
	doc SessionMappingDocument,
	applyMode string,
	overlay *LiveOverlaySnapshot,
	now time.Time,
) (PackDryRunReport, []DryRunConflict, int64, int64, error) {
	rep := PackDryRunReport{
		SessionID:      sess.SessionID,
		Pack:           sess.Pack,
		StreamSplitKey: sess.StreamSplitKey,
		Currency:       overlay.SupplierCurrency,
	}
	var conflicts []DryRunConflict
	addConflict := func(rowIdx int64, code, severity, field, msg, suggestion string) {
		conflicts = append(conflicts, DryRunConflict{
			SessionID:  sess.SessionID,
			Pack:       sess.Pack,
			RowIndex:   rowIdx,
			Code:       code,
			Severity:   severity,
			Field:      field,
			Message:    msg,
			Suggestion: suggestion,
		})
		if severity == ConflictSeverityBlocking {
			rep.BlockingCount++
		} else {
			rep.WarningCount++
		}
	}

	if err := ValidateMappingsReadyForApply(doc); err != nil {
		addConflict(0, "incomplete_required_mappings", ConflictSeverityBlocking, "", err.Error(), "Confirm all required column mappings >= 80% before dry-run approval.")
	}
	if sess.Pack == PackLots && applyMode == ApplyModeSetOnHand {
		addConflict(0, "lot_set_on_hand_forbidden", ConflictSeverityBlocking, "apply_mode", "SET_ON_HAND overwrite is forbidden on lot-tracked stock; use MERGE.", "Switch inventory apply_mode to MERGE.")
	}
	for _, lock := range overlay.WarehouseLocks {
		if lock.SessionID != sess.SessionID && lock.ExpiresAt.After(now) {
			addConflict(0, "warehouse_lock_busy", ConflictSeverityBlocking, "warehouse_id",
				fmt.Sprintf("warehouse %s is locked by active session %s until %s", lock.WarehouseID, lock.SessionID, lock.ExpiresAt.Format(time.RFC3339)),
				"Wait for the active warehouse apply job to finish before applying.")
		}
	}

	var debitTotal, creditTotal int64
	seenCustomerKeys := make(map[string]bool)
	seenOrderKeys := make(map[string]bool)

	if err := streamAllStagedRows(ctx, repo, supplierID, sess.SessionID, -1, dryRunRowPageSize, func(rows []StagedRow) error {
		for _, sr := range rows {
			rep.RowsTotal++
			fields := ExtractCanonicalRow(sr, doc)
			okRow := simulatePackRow(supplierID, sess, fields, sr.RowIndex, applyMode, overlay, seenCustomerKeys, seenOrderKeys, &rep, &debitTotal, &creditTotal, addConflict)
			if okRow {
				rep.RowsValid++
			} else {
				rep.RowsInvalid++
			}
		}
		return nil
	}); err != nil {
		return PackDryRunReport{}, nil, 0, 0, err
	}

	return rep, conflicts, debitTotal, creditTotal, nil
}

func simulatePackRow(
	supplierID string,
	sess Session,
	fields map[string]string,
	rowIdx int64,
	applyMode string,
	overlay *LiveOverlaySnapshot,
	seenCustomers, seenOrders map[string]bool,
	rep *PackDryRunReport,
	debitTotal, creditTotal *int64,
	addConflict func(rowIdx int64, code, severity, field, msg, suggestion string),
) bool {
	switch sess.Pack {
	case PackWarehouses:
		return simWarehouseRow(fields, rowIdx, overlay, rep, addConflict)
	case PackProductsPrices:
		return simProductRow(fields, rowIdx, overlay, rep, addConflict)
	case PackCustomers:
		return simCustomerRow(fields, rowIdx, overlay, seenCustomers, rep, addConflict)
	case PackStock:
		return simStockRow(fields, rowIdx, applyMode, overlay, rep, addConflict)
	case PackLots:
		return simLotRow(fields, rowIdx, applyMode, overlay, rep, addConflict)
	case PackHistoricalOrders:
		return simHistoricalOrderRow(fields, rowIdx, overlay, seenOrders, rep, addConflict)
	case PackNasiyaAR:
		return simNasiyaRow(supplierID, fields, rowIdx, overlay, rep, debitTotal, creditTotal, addConflict)
	case PackOpenOrders:
		return simOpenOrderRow(fields, rowIdx, overlay, seenOrders, rep, addConflict)
	default:
		addConflict(rowIdx, "unsupported_pack", ConflictSeverityBlocking, "pack", fmt.Sprintf("unsupported pack %q", sess.Pack), "")
		return false
	}
}

func simWarehouseRow(fields map[string]string, rowIdx int64, overlay *LiveOverlaySnapshot, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	name := strings.TrimSpace(fields["name"])
	if name == "" {
		addConflict(rowIdx, "missing_warehouse_name", ConflictSeverityBlocking, "name", "warehouse name is required", "")
		return false
	}
	ext := strings.TrimSpace(fields["external_warehouse_id"])
	normName := strings.ToLower(name)
	_, byExt := overlay.WarehousesByExt[strings.ToLower(ext)]
	_, byName := overlay.WarehousesByName[normName]
	if (ext != "" && byExt) || byName {
		rep.WouldUpdate++
	} else {
		rep.WouldCreate++
		whID := "vwh_" + CleanIDToken(firstNonEmpty(ext, normName))
		w := LiveWarehouse{WarehouseID: whID, ExternalID: ext, Name: name, IsDefault: parseBoolLike(fields["is_default"])}
		if ext != "" {
			overlay.WarehousesByExt[strings.ToLower(ext)] = w
		}
		overlay.WarehousesByName[normName] = w
		if w.IsDefault || overlay.DefaultWarehouseID == "" {
			overlay.DefaultWarehouseID = whID
		}
	}
	return true
}

func simProductRow(fields map[string]string, rowIdx int64, overlay *LiveOverlaySnapshot, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	name := strings.TrimSpace(fields["name"])
	sku := strings.TrimSpace(fields["sku"])
	barcode := strings.TrimSpace(fields["barcode"])
	if name == "" || (sku == "" && barcode == "") {
		addConflict(rowIdx, "missing_product_identity", ConflictSeverityBlocking, "sku", "product requires name and sku or barcode", "")
		return false
	}
	var priceMinor int64
	if rawP := strings.TrimSpace(fields["price"]); rawP != "" {
		p, err := ParseMoneyMinor(rawP)
		if err != nil || p < 0 {
			addConflict(rowIdx, "invalid_price", ConflictSeverityBlocking, "price", fmt.Sprintf("invalid price %q", rawP), "")
			return false
		}
		priceMinor = p
	}
	if cur := strings.ToUpper(strings.TrimSpace(fields["currency"])); cur != "" && cur != overlay.SupplierCurrency {
		addConflict(rowIdx, "currency_mismatch", ConflictSeverityBlocking, "currency", fmt.Sprintf("product currency %s differs from supplier currency %s", cur, overlay.SupplierCurrency), "")
		return false
	}
	unit := strings.TrimSpace(fields["unit"])
	existing, found := lookupVirtualProduct(overlay, sku, barcode, name)
	if found {
		if unit != "" && existing.Unit != "" && !strings.EqualFold(unit, existing.Unit) && overlay.StockTotalByProd[existing.ProductID] > 0 {
			addConflict(rowIdx, "unit_change_with_non_zero_stock", ConflictSeverityBlocking, "unit",
				fmt.Sprintf("cannot change SKU unit from %q to %q while stock > 0", existing.Unit, unit),
				"Adjust stock to 0 before changing base unit of measure.")
			return false
		}
		if priceMinor > 0 && existing.PriceMinor > 0 && (priceMinor >= existing.PriceMinor*5 || existing.PriceMinor >= priceMinor*5) {
			addConflict(rowIdx, "anomalous_price_change", ConflictSeverityWarning, "price",
				fmt.Sprintf("price changed sharply from %d to %d minor units", existing.PriceMinor, priceMinor),
				"Verify missing zeroes or tin/som unit scale before approval.")
		}
		if unit != "" {
			existing.Unit = unit
		}
		if priceMinor > 0 {
			existing.PriceMinor = priceMinor
		}
		registerVirtualProduct(overlay, existing)
		rep.WouldUpdate++
		return true
	}

	prodID := "vprod_" + CleanIDToken(firstNonEmpty(sku, barcode, name))
	p := LiveProduct{
		ProductID:  prodID,
		SKU:        sku,
		Barcode:    barcode,
		Name:       name,
		Unit:       firstNonEmpty(unit, "pcs"),
		PriceMinor: priceMinor,
		Currency:   overlay.SupplierCurrency,
	}
	registerVirtualProduct(overlay, p)
	rep.WouldCreate++
	return true
}

func simCustomerRow(fields map[string]string, rowIdx int64, overlay *LiveOverlaySnapshot, seen map[string]bool, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	name := strings.TrimSpace(fields["customer_name"])
	inn := NormalizeINN(fields["inn"])
	phone := NormalizePhoneUZ(fields["phone"])
	if name == "" || (inn == "" && phone == "") {
		addConflict(rowIdx, "missing_customer_identity", ConflictSeverityBlocking, "inn", "customer requires name and INN or phone", "")
		return false
	}
	if rawINN := strings.TrimSpace(fields["inn"]); rawINN != "" && len(inn) != 9 && len(inn) != 14 {
		addConflict(rowIdx, "invalid_inn_format", ConflictSeverityBlocking, "inn", fmt.Sprintf("INN %q must be 9-digit STIR or 14-digit PINFL", rawINN), "")
		return false
	}
	if inn == "" && phone != "" {
		addConflict(rowIdx, "informal_shop_phone_only", ConflictSeverityWarning, "inn",
			fmt.Sprintf("customer %q has phone %s without INN; will link as INFORMAL_PHONE", name, phone),
			"Verify Tax ID if corporate B2B invoicing is required.")
	}
	if rawCL := strings.TrimSpace(fields["credit_limit"]); rawCL != "" {
		if cl, err := ParseMoneyMinor(rawCL); err != nil || cl < 0 {
			addConflict(rowIdx, "invalid_credit_limit", ConflictSeverityBlocking, "credit_limit", fmt.Sprintf("invalid credit_limit %q", rawCL), "")
			return false
		}
	}

	dedupKey := firstNonEmpty(inn, phone)
	if seen[dedupKey] {
		addConflict(rowIdx, "duplicate_customer_in_file", ConflictSeverityWarning, "inn", fmt.Sprintf("duplicate customer identity %s skipped", dedupKey), "")
		rep.WouldSkip++
		return true
	}
	seen[dedupKey] = true

	if _, ok := lookupVirtualCustomer(overlay, fields["external_customer_id"], inn, phone, name); ok {
		rep.WouldUpdate++
		return true
	}
	c := LiveCustomer{
		RetailerID:   "vret_" + CleanIDToken(dedupKey),
		ExternalID:   strings.TrimSpace(fields["external_customer_id"]),
		INN:          inn,
		Phone:        phone,
		CustomerName: name,
		Currency:     overlay.SupplierCurrency,
	}
	registerVirtualCustomer(overlay, c)
	rep.WouldCreate++
	return true
}

func simStockRow(fields map[string]string, rowIdx int64, applyMode string, overlay *LiveOverlaySnapshot, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	sku := strings.TrimSpace(fields["sku"])
	prod, ok := lookupVirtualProduct(overlay, sku, sku, sku)
	if !ok {
		addConflict(rowIdx, "unresolved_sku", ConflictSeverityBlocking, "sku", fmt.Sprintf("SKU %q not found in live catalog or preceding products_prices pack", sku), "")
		return false
	}
	wh, ok := resolveVirtualWarehouse(overlay, fields["warehouse_id"])
	if !ok {
		addConflict(rowIdx, "unresolved_warehouse", ConflictSeverityBlocking, "warehouse_id", fmt.Sprintf("warehouse %q could not be resolved", fields["warehouse_id"]), "")
		return false
	}
	qty, err := ParseQuantityInt(fields["quantity_on_hand"])
	if err != nil || qty < 0 {
		addConflict(rowIdx, "invalid_quantity", ConflictSeverityBlocking, "quantity_on_hand", fmt.Sprintf("invalid stock quantity %q", fields["quantity_on_hand"]), "")
		return false
	}
	if unit := strings.TrimSpace(fields["unit"]); unit != "" && prod.Unit != "" && !strings.EqualFold(unit, prod.Unit) && overlay.StockTotalByProd[prod.ProductID] > 0 {
		addConflict(rowIdx, "unit_change_with_non_zero_stock", ConflictSeverityBlocking, "unit",
			fmt.Sprintf("stock unit %q conflicts with catalog unit %q while stock > 0", unit, prod.Unit), "")
		return false
	}

	key := wh.WarehouseID + "|" + prod.ProductID
	cur, exists := overlay.StockByWHProduct[key]
	if !exists {
		cur = LiveStock{WarehouseID: wh.WarehouseID, ProductID: prod.ProductID, Unit: prod.Unit}
	}
	var newOnHand int64
	if applyMode == ApplyModeSetOnHand {
		newOnHand = qty
	} else {
		newOnHand = cur.OnHand + qty
	}
	if newOnHand < cur.Reserved {
		rep.OnHandBelowReservedCount++
		addConflict(rowIdx, "on_hand_below_reserved", ConflictSeverityBlocking, "quantity_on_hand",
			fmt.Sprintf("new on_hand %d < live reserved %d for %s at %s", newOnHand, cur.Reserved, sku, wh.Name),
			"Increase imported quantity or release live reservations first.")
		return false
	}
	delta := newOnHand - cur.OnHand
	cur.OnHand = newOnHand
	overlay.StockByWHProduct[key] = cur
	overlay.StockTotalByProd[prod.ProductID] += delta
	if exists {
		rep.WouldUpdate++
	} else {
		rep.WouldCreate++
	}
	return true
}

func simLotRow(fields map[string]string, rowIdx int64, applyMode string, overlay *LiveOverlaySnapshot, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	if applyMode == ApplyModeSetOnHand {
		return false
	}
	sku := strings.TrimSpace(fields["sku"])
	prod, ok := lookupVirtualProduct(overlay, sku, sku, sku)
	if !ok {
		addConflict(rowIdx, "unresolved_sku", ConflictSeverityBlocking, "sku", fmt.Sprintf("lot SKU %q not found", sku), "")
		return false
	}
	wh, ok := resolveVirtualWarehouse(overlay, fields["warehouse_id"])
	if !ok {
		addConflict(rowIdx, "unresolved_warehouse", ConflictSeverityBlocking, "warehouse_id", fmt.Sprintf("warehouse %q could not be resolved", fields["warehouse_id"]), "")
		return false
	}
	lotCode := strings.TrimSpace(fields["lot_number"])
	if lotCode == "" {
		addConflict(rowIdx, "missing_lot_number", ConflictSeverityBlocking, "lot_number", "lot_number is required", "")
		return false
	}
	exp, err := ParseFlexibleDate(fields["expiry_date"])
	if err != nil {
		addConflict(rowIdx, "invalid_expiry_date", ConflictSeverityBlocking, "expiry_date", fmt.Sprintf("invalid expiry_date %q", fields["expiry_date"]), "")
		return false
	}
	if prodDateStr := strings.TrimSpace(fields["production_date"]); prodDateStr != "" {
		mfg, pErr := ParseFlexibleDate(prodDateStr)
		if pErr != nil || mfg.After(exp) {
			addConflict(rowIdx, "invalid_lot_dates", ConflictSeverityBlocking, "production_date", "production_date invalid or after expiry_date", "")
			return false
		}
	}
	qty, err := ParseQuantityInt(fields["quantity_on_hand"])
	if err != nil || qty < 0 {
		addConflict(rowIdx, "invalid_quantity", ConflictSeverityBlocking, "quantity_on_hand", fmt.Sprintf("invalid lot quantity %q", fields["quantity_on_hand"]), "")
		return false
	}

	if prod.RequiresColdChain || strings.Contains(strings.ToLower(fields["temperature_mode"]), "cold") {
		rep.ColdLotWithoutLiveCount++
		addConflict(rowIdx, "cold_lot_migrated_without_live_baseline", ConflictSeverityWarning, "lot_number",
			fmt.Sprintf("cold-chain lot %s (%s) migrated without live IoT baseline", lotCode, sku),
			"Warehouse team must confirm sensor baseline during putaway.")
	}

	lotKey := wh.WarehouseID + "|" + prod.ProductID + "|" + strings.ToLower(lotCode)
	curLot, exists := overlay.LotsByKey[lotKey]
	if exists {
		curLot.OnHand += qty
		overlay.LotsByKey[lotKey] = curLot
		rep.WouldUpdate++
	} else {
		overlay.LotsByKey[lotKey] = LiveLot{
			LotID:       "vlot_" + CleanIDToken(lotKey),
			WarehouseID: wh.WarehouseID,
			ProductID:   prod.ProductID,
			LotCode:     lotCode,
			ExpiryDate:  exp,
			OnHand:      qty,
		}
		rep.WouldCreate++
	}
	stkKey := wh.WarehouseID + "|" + prod.ProductID
	stk := overlay.StockByWHProduct[stkKey]
	stk.WarehouseID = wh.WarehouseID
	stk.ProductID = prod.ProductID
	stk.OnHand += qty
	overlay.StockByWHProduct[stkKey] = stk
	overlay.StockTotalByProd[prod.ProductID] += qty
	return true
}

func simHistoricalOrderRow(fields map[string]string, rowIdx int64, overlay *LiveOverlaySnapshot, seenOrders map[string]bool, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	extNo := strings.TrimSpace(fields["external_order_no"])
	if extNo == "" {
		addConflict(rowIdx, "missing_external_order_no", ConflictSeverityBlocking, "external_order_no", "external_order_no is required", "")
		return false
	}
	if _, err := ParseFlexibleDate(fields["order_date"]); err != nil {
		addConflict(rowIdx, "invalid_order_date", ConflictSeverityBlocking, "order_date", fmt.Sprintf("invalid order_date %q", fields["order_date"]), "")
		return false
	}
	amtMinor, err := ParseMoneyMinor(fields["total_amount"])
	if err != nil || amtMinor < 0 {
		addConflict(rowIdx, "invalid_total_amount", ConflictSeverityBlocking, "total_amount", fmt.Sprintf("invalid total_amount %q", fields["total_amount"]), "")
		return false
	}
	key := strings.ToLower(extNo)
	if _, exists := overlay.OrdersByExtNo[key]; exists {
		rep.WouldSkip++
		return true
	}
	if !seenOrders[key] {
		seenOrders[key] = true
		overlay.OrdersByExtNo[key] = LiveOrder{ExternalOrderNo: extNo, OrderSource: "MIGRATION", Status: "DELIVERED"}
		rep.WouldCreate++
		rep.MoneyImpactMinor += amtMinor
	} else {
		// Multi-line order aggregation
		rep.WouldUpdate++
	}
	return true
}

func simNasiyaRow(supplierID string, fields map[string]string, rowIdx int64, overlay *LiveOverlaySnapshot, rep *PackDryRunReport, debitTotal, creditTotal *int64, addConflict func(int64, string, string, string, string, string)) bool {
	balMinor, err := ParseMoneyMinor(fields["balance_due"])
	if err != nil || balMinor <= 0 {
		addConflict(rowIdx, "invalid_opening_ar_balance", ConflictSeverityBlocking, "balance_due", fmt.Sprintf("nasiya balance_due must be positive; got %q", fields["balance_due"]), "")
		return false
	}
	cur := strings.ToUpper(strings.TrimSpace(fields["currency"]))
	if cur == "" {
		cur = overlay.SupplierCurrency
	}
	if cur != overlay.SupplierCurrency {
		addConflict(rowIdx, "currency_mismatch", ConflictSeverityBlocking, "currency", fmt.Sprintf("nasiya currency %s differs from supplier currency %s", cur, overlay.SupplierCurrency), "")
		return false
	}

	inn := NormalizeINN(fields["customer_inn"])
	phone := NormalizePhoneUZ(fields["customer_phone"])
	custName := strings.TrimSpace(fields["customer_name"])
	cust, ok := lookupVirtualCustomer(overlay, "", inn, phone, custName)
	if !ok {
		addConflict(rowIdx, "unresolved_debtor", ConflictSeverityBlocking, "customer_name",
			fmt.Sprintf("debtor %q (INN=%q phone=%q) not found in live links or staged customers pack", custName, inn, phone),
			"Include the debtor in customers pack first.")
		return false
	}

	extInv := strings.TrimSpace(fields["external_invoice_id"])
	if extInv != "" {
		if _, exists := overlay.ArInvoicesByExt[strings.ToLower(extInv)]; exists {
			rep.WouldSkip++
			return true
		}
		overlay.ArInvoicesByExt[strings.ToLower(extInv)] = LiveARInvoice{
			ExternalID:   extInv,
			BalanceMinor: balMinor,
			Currency:     cur,
			Status:       "OPEN",
		}
	}

	// Prove balanced double-entry journal invariant during dry-run.
	je := &payment.JournalEntry{
		EntryID:       fmt.Sprintf("dryrun_nasiya_%d", rowIdx),
		TransactionID: fmt.Sprintf("dryrun_%s_%d", supplierID, rowIdx),
		SupplierID:    supplierID,
		RetailerID:    cust.RetailerID,
		Currency:      cur,
		Memo:          "Dry-run opening nasiya AR preview",
		OccurredAt:    time.Now().UTC(),
		Postings: []payment.Posting{
			{
				AccountCode: fmt.Sprintf("AR:%s", supplierID),
				AccountType: payment.AccountClassAsset,
				Direction:   payment.PostingDebit,
				AmountMinor: balMinor,
				Currency:    cur,
				PartyID:     cust.RetailerID,
			},
			{
				AccountCode: "EQUITY:OPENING_BALANCE",
				AccountType: payment.AccountClassEquity,
				Direction:   payment.PostingCredit,
				AmountMinor: balMinor,
				Currency:    cur,
				PartyID:     supplierID,
			},
		},
	}
	if err := je.Validate(); err != nil {
		addConflict(rowIdx, "unbalanced_opening_journal", ConflictSeverityBlocking, "balance_due", err.Error(), "")
		return false
	}

	*debitTotal += balMinor
	*creditTotal += balMinor
	rep.MoneyImpactMinor += balMinor
	rep.WouldCreate++
	return true
}

func simOpenOrderRow(fields map[string]string, rowIdx int64, overlay *LiveOverlaySnapshot, seenOrders map[string]bool, rep *PackDryRunReport, addConflict func(int64, string, string, string, string, string)) bool {
	extNo := strings.TrimSpace(fields["external_order_no"])
	if extNo == "" {
		addConflict(rowIdx, "missing_external_order_no", ConflictSeverityBlocking, "external_order_no", "external_order_no is required", "")
		return false
	}
	if _, err := ParseFlexibleDate(fields["delivery_date"]); err != nil {
		addConflict(rowIdx, "invalid_delivery_date", ConflictSeverityBlocking, "delivery_date", fmt.Sprintf("invalid delivery_date %q", fields["delivery_date"]), "")
		return false
	}
	sku := strings.TrimSpace(fields["sku"])
	prod, ok := lookupVirtualProduct(overlay, sku, sku, sku)
	if !ok {
		addConflict(rowIdx, "unresolved_sku", ConflictSeverityBlocking, "sku", fmt.Sprintf("open order SKU %q not found", sku), "")
		return false
	}
	wh, ok := resolveVirtualWarehouse(overlay, fields["warehouse_id"])
	if !ok {
		addConflict(rowIdx, "unresolved_warehouse", ConflictSeverityBlocking, "warehouse_id", fmt.Sprintf("warehouse %q could not be resolved", fields["warehouse_id"]), "")
		return false
	}
	qty, err := ParseQuantityInt(fields["quantity"])
	if err != nil || qty <= 0 {
		addConflict(rowIdx, "invalid_quantity", ConflictSeverityBlocking, "quantity", fmt.Sprintf("invalid open order quantity %q", fields["quantity"]), "")
		return false
	}
	if _, exists := overlay.OrdersByExtNo[strings.ToLower(extNo)]; exists {
		rep.WouldSkip++
		return true
	}

	stkKey := wh.WarehouseID + "|" + prod.ProductID
	stk := overlay.StockByWHProduct[stkKey]
	avail := stk.OnHand - stk.Reserved
	if avail < qty {
		addConflict(rowIdx, "insufficient_stock_for_held_open_order", ConflictSeverityWarning, "quantity",
			fmt.Sprintf("open order %s needs %d of %s at %s but available=%d; will hold as MIGRATION_OPEN", extNo, qty, sku, wh.Name, avail),
			"Restock before releasing held open orders.")
	} else {
		stk.Reserved += qty
		overlay.StockByWHProduct[stkKey] = stk
	}

	unitPrice := prod.PriceMinor
	if rawUP := strings.TrimSpace(fields["unit_price"]); rawUP != "" {
		if p, err := ParseMoneyMinor(rawUP); err == nil && p >= 0 {
			unitPrice = p
		}
	}
	rep.MoneyImpactMinor += qty * unitPrice
	key := strings.ToLower(extNo)
	if !seenOrders[key] {
		seenOrders[key] = true
		rep.WouldCreate++
	} else {
		rep.WouldUpdate++
	}
	return true
}

// ApproveSession enforces 24h dry-run freshness, zero blocking conflicts, exact content+mapping fingerprint parity,
// and explicit SET_ON_HAND confirmation before transitioning a session to APPROVED.
func ApproveSession(
	ctx context.Context,
	repo Repository,
	supplierID, sessionID string,
	req ApproveSessionRequest,
	now time.Time,
) (Session, error) {
	if repo == nil {
		return Session{}, fmt.Errorf("migration repository is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	sess, err := repo.GetSession(ctx, supplierID, sessionID)
	if err != nil {
		return Session{}, err
	}
	if sess.DryRunReportID == "" {
		return Session{}, fmt.Errorf("%w: run dry-run before approval", ErrDryRunStale)
	}
	if req.ExpectedReportID != "" && req.ExpectedReportID != sess.DryRunReportID {
		return Session{}, fmt.Errorf("%w: report %q does not match latest dry-run %q", ErrDryRunStale, req.ExpectedReportID, sess.DryRunReportID)
	}

	rep, err := repo.GetMigrationReport(ctx, supplierID, sess.DryRunReportID)
	if err != nil {
		return Session{}, err
	}
	if !now.UTC().Before(rep.ExpiresAt.UTC()) {
		return Session{}, fmt.Errorf("%w: report expired at %s", ErrDryRunExpired, rep.ExpiresAt.Format(time.RFC3339))
	}
	if rep.BlockingCount > 0 {
		return Session{}, fmt.Errorf("%w: %d blocking conflict(s) remain", ErrDryRunBlockingConflicts, rep.BlockingCount)
	}

	applyMode := NormalizeApplyMode(firstNonEmpty(req.ApplyMode, sess.ApplyMode))
	if sess.Pack == PackLots && applyMode == ApplyModeSetOnHand {
		return Session{}, fmt.Errorf("%w: SET_ON_HAND is forbidden for lots pack", ErrInvalidInput)
	}
	if applyMode == ApplyModeSetOnHand && !req.SetOnHandConfirmed {
		return Session{}, ErrSetOnHandConfirmationRequired
	}

	doc, hasDoc, err := repo.GetSessionMapping(ctx, supplierID, sessionID)
	if err != nil {
		return Session{}, err
	}
	if !hasDoc {
		doc = SessionMappingDocument{SessionID: sessionID, Pack: sess.Pack}
	}
	if err := ValidateMappingsReadyForApply(doc); err != nil {
		return Session{}, err
	}
	curFP := ComputeSessionDryRunFingerprint(sess, doc, applyMode)
	if curFP != rep.ContentFingerprint || (sess.DryRunFingerprint != "" && curFP != sess.DryRunFingerprint) {
		return Session{}, fmt.Errorf("%w: session content or mapping changed since dry-run", ErrDryRunStale)
	}

	if err := repo.ApproveSession(ctx, supplierID, sessionID, applyMode, req.SetOnHandConfirmed); err != nil {
		return Session{}, err
	}
	return repo.GetSession(ctx, supplierID, sessionID)
}

// ExtractCanonicalRow projects StagedRow RawData/CleanedData through confirmed mapping proposals into canonical fields.
func ExtractCanonicalRow(sr StagedRow, doc SessionMappingDocument) map[string]string {
	out := make(map[string]string)
	loadJSONMap := func(raw json.RawMessage, override bool) {
		if len(raw) == 0 {
			return
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return
		}
		for k, v := range m {
			valStr := formatAnyString(v)
			if valStr == "" {
				continue
			}
			if _, exists := out[k]; !exists || override {
				out[k] = valStr
			}
		}
	}
	rawVals := make(map[string]string)
	if len(sr.RawData) > 0 {
		var m map[string]any
		if err := json.Unmarshal(sr.RawData, &m); err == nil {
			for k, v := range m {
				rawVals[k] = formatAnyString(v)
			}
		}
	}
	// First apply cleaned_data canonical keys if already normalized during staging.
	loadJSONMap(sr.CleanedData, true)
	// Next project raw columns via confirmed/proposed mapping table.
	for _, prop := range doc.Mappings {
		if prop.TargetField == "" {
			continue
		}
		v, ok := rawVals[prop.SourceColumn]
		if !ok || strings.TrimSpace(v) == "" {
			continue
		}
		out[prop.TargetField] = applyFieldTransform(v, prop.Transform)
	}
	// Finally fill any canonical key already directly present in raw_data if unmapped.
	loadJSONMap(sr.RawData, false)
	return out
}

func applyFieldTransform(val, transform string) string {
	v := strings.TrimSpace(val)
	switch strings.ToLower(strings.TrimSpace(transform)) {
	case "uppercase":
		return strings.ToUpper(v)
	case "lowercase":
		return strings.ToLower(v)
	case "inn", "digits_only":
		return NormalizeINN(v)
	case "phone", "phone_uz":
		return NormalizePhoneUZ(v)
	default:
		return v
	}
}

// NormalizeINN extracts digits for Uzbekistan 9-digit STIR or 14-digit PINFL.
func NormalizeINN(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizePhoneUZ normalizes Uzbekistan phone numbers into canonical +998XXXXXXXXX form when possible.
func NormalizePhoneUZ(raw string) string {
	digits := NormalizeINN(raw)
	if digits == "" {
		return ""
	}
	if len(digits) == 9 {
		digits = "998" + digits
	} else if len(digits) == 10 && strings.HasPrefix(digits, "0") {
		digits = "998" + digits[1:]
	}
	return "+" + digits
}

// ParseFlexibleDate parses ISO, Russian DD.MM.YYYY, DD/MM/YYYY, or RFC3339 date strings.
func ParseFlexibleDate(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"02.01.2006",
		"02/01/2006",
		"02-01-2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date %q", raw)
}

// ParseQuantityInt parses numeric quantities into whole INT64 units, rejecting NaN/Inf/fractional garbage.
func ParseQuantityInt(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty quantity")
	}
	s = strings.NewReplacer(" ", "", "\u00a0", "", ",", ".").Replace(s)
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid quantity %q", raw)
	}
	return int64(math.Round(f)), nil
}

// CleanIDToken converts an arbitrary string into a safe ASCII token for deterministic virtual/entity IDs.
func CleanIDToken(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) > 48 {
		out = out[:48]
	}
	if out == "" {
		return "item"
	}
	return out
}

func formatAnyString(v any) string {
	switch tv := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(tv)
	case float64:
		if math.Mod(tv, 1) == 0 {
			return strconv.FormatInt(int64(tv), 10)
		}
		return strconv.FormatFloat(tv, 'f', -1, 64)
	case bool:
		if tv {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func parseBoolLike(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "да", "ha":
		return true
	default:
		return false
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func lookupVirtualProduct(overlay *LiveOverlaySnapshot, sku, barcode, name string) (LiveProduct, bool) {
	if s := strings.ToLower(strings.TrimSpace(sku)); s != "" {
		if p, ok := overlay.ProductsBySKU[s]; ok {
			return p, true
		}
	}
	if b := strings.ToLower(strings.TrimSpace(barcode)); b != "" {
		if p, ok := overlay.ProductsByBarcode[b]; ok {
			return p, true
		}
	}
	if n := strings.ToLower(strings.TrimSpace(name)); n != "" {
		if p, ok := overlay.ProductsByName[n]; ok {
			return p, true
		}
	}
	return LiveProduct{}, false
}

func registerVirtualProduct(overlay *LiveOverlaySnapshot, p LiveProduct) {
	overlay.ProductsByID[p.ProductID] = p
	if p.SKU != "" {
		overlay.ProductsBySKU[strings.ToLower(strings.TrimSpace(p.SKU))] = p
	}
	if p.Barcode != "" {
		overlay.ProductsByBarcode[strings.ToLower(strings.TrimSpace(p.Barcode))] = p
	}
	if p.Name != "" {
		overlay.ProductsByName[strings.ToLower(strings.TrimSpace(p.Name))] = p
	}
}

func lookupVirtualCustomer(overlay *LiveOverlaySnapshot, extID, inn, phone, name string) (LiveCustomer, bool) {
	if e := strings.ToLower(strings.TrimSpace(extID)); e != "" {
		if c, ok := overlay.CustomersByExt[e]; ok {
			return c, true
		}
	}
	if inn != "" {
		if c, ok := overlay.CustomersByINN[inn]; ok {
			return c, true
		}
	}
	if phone != "" {
		if c, ok := overlay.CustomersByPhone[phone]; ok {
			return c, true
		}
	}
	if n := strings.ToLower(strings.TrimSpace(name)); n != "" {
		if c, ok := overlay.CustomersByName[n]; ok {
			return c, true
		}
	}
	return LiveCustomer{}, false
}

func registerVirtualCustomer(overlay *LiveOverlaySnapshot, c LiveCustomer) {
	if c.ExternalID != "" {
		overlay.CustomersByExt[strings.ToLower(strings.TrimSpace(c.ExternalID))] = c
	}
	if c.INN != "" {
		overlay.CustomersByINN[c.INN] = c
	}
	if c.Phone != "" {
		overlay.CustomersByPhone[c.Phone] = c
	}
	if c.CustomerName != "" {
		overlay.CustomersByName[strings.ToLower(strings.TrimSpace(c.CustomerName))] = c
	}
}

func resolveVirtualWarehouse(overlay *LiveOverlaySnapshot, rawRef string) (LiveWarehouse, bool) {
	ref := strings.ToLower(strings.TrimSpace(rawRef))
	if ref != "" {
		if w, ok := overlay.WarehousesByExt[ref]; ok {
			return w, true
		}
		if w, ok := overlay.WarehousesByName[ref]; ok {
			return w, true
		}
		return LiveWarehouse{}, false
	}
	if overlay.DefaultWarehouseID != "" {
		for _, w := range overlay.WarehousesByExt {
			if w.WarehouseID == overlay.DefaultWarehouseID {
				return w, true
			}
		}
		for _, w := range overlay.WarehousesByName {
			if w.WarehouseID == overlay.DefaultWarehouseID {
				return w, true
			}
		}
		return LiveWarehouse{WarehouseID: overlay.DefaultWarehouseID, Name: overlay.DefaultWarehouseID}, true
	}
	if len(overlay.WarehousesByName) == 1 {
		for _, w := range overlay.WarehousesByName {
			return w, true
		}
	}
	return LiveWarehouse{}, false
}

var _ = stocklots.LotsEnabled
