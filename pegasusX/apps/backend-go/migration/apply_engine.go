package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

const (
	// DefaultApplyChunkSize bounds rows committed per Spanner ReadWriteTransaction so 25k+ files never hit txn mutation limits.
	DefaultApplyChunkSize = 250
)

var (
	// ErrUnitChangeWithNonZeroStock blocks catalog unit mutations when live warehouse stock > 0.
	ErrUnitChangeWithNonZeroStock = errors.New("unit_change_with_non_zero_stock")
	// ErrOnHandBelowReserved blocks stock overwrites that would reduce QuantityOnHand below QuantityReserved.
	ErrOnHandBelowReserved = errors.New("on_hand_below_reserved")
	// ErrSessionNotApproved indicates ApplySession requires an active APPROVED (or resumable FAILED) state.
	ErrSessionNotApproved = errors.New("session_not_approved")
)

// StoredWarehouse mirrors a Warehouses row across Spanner and memory apply stores.
type StoredWarehouse struct {
	WarehouseID string `json:"WarehouseId"`
	SupplierID  string `json:"SupplierId"`
	Name        string `json:"Name"`
	Address     string `json:"Address,omitempty"`
	IsActive    bool   `json:"IsActive"`
	IsOnShift   bool   `json:"IsOnShift"`
}

// StoredProduct mirrors a Products row across Spanner and memory apply stores.
type StoredProduct struct {
	ProductID         string `json:"ProductId"`
	SupplierID        string `json:"SupplierId"`
	CategoryID        string `json:"CategoryId"`
	Name              string `json:"Name"`
	Description       string `json:"Description,omitempty"`
	PriceMinor        int64  `json:"PriceMinor"`
	Currency          string `json:"Currency"`
	Unit              string `json:"Unit"`
	Barcode           string `json:"Barcode,omitempty"`
	HandlingClass     string `json:"HandlingClass"`
	IsPerishable      bool   `json:"IsPerishable"`
	RequiresColdChain bool   `json:"RequiresColdChain"`
	MinShelfLifeDays  int64  `json:"MinShelfLifeDays,omitempty"`
	IsActive          bool   `json:"IsActive"`
	Version           int64  `json:"Version"`
}

// StoredCustomer mirrors a resolved Retailers + SupplierRetailerLinks customer entity.
type StoredCustomer struct {
	RetailerID   string `json:"RetailerId"`
	Phone        string `json:"Phone"`
	Name         string `json:"Name"`
	CountryCode  string `json:"CountryCode"`
	DeliveryAddr string `json:"DeliveryAddress,omitempty"`
	Link         StoredCustomerLink
}

// StoredCustomerLink mirrors a SupplierRetailerLinks row.
type StoredCustomerLink struct {
	SupplierID      string `json:"SupplierId"`
	RetailerID      string `json:"RetailerId"`
	TaxIdTIN        string `json:"TaxIdTIN,omitempty"`
	NormalizedPhone string `json:"NormalizedPhone,omitempty"`
	LegalName       string `json:"LegalName,omitempty"`
	DedupClusterID  string `json:"DedupClusterId,omitempty"`
	DedupStatus     string `json:"DedupStatus"`
}

// StoredStock mirrors a SupplierInventoryV2 row.
type StoredStock struct {
	SupplierID       string `json:"SupplierId"`
	WarehouseID      string `json:"WarehouseId"`
	ProductID        string `json:"ProductId"`
	QuantityOnHand   int64  `json:"QuantityOnHand"`
	QuantityReserved int64  `json:"QuantityReserved"`
	ReorderThreshold int64  `json:"ReorderThreshold"`
}

// StoredLot mirrors a StockLots row.
type StoredLot struct {
	LotID            string     `json:"LotId"`
	SupplierID       string     `json:"SupplierId"`
	WarehouseID      string     `json:"WarehouseId"`
	ProductID        string     `json:"ProductId"`
	LocationID       string     `json:"LocationId"`
	LotCode          string     `json:"LotCode"`
	ExpiryDate       *time.Time `json:"ExpiryDate,omitempty"`
	ManufacturedDate *time.Time `json:"ManufacturedDate,omitempty"`
	QuantityOnHand   int64      `json:"QuantityOnHand"`
	QuantityReserved int64      `json:"QuantityReserved"`
	Status           string     `json:"Status"`
}

// LotMigrationOp describes a putaway or re-import delta adjustment onto StockLots.
type LotMigrationOp struct {
	WarehouseID      string
	ProductID        string
	LocationID       string
	LotCode          string
	TargetQuantity   int64
	ExpiryDate       *time.Time
	ManufacturedDate *time.Time
	NewLotID         string
}

// LotMigrationResult describes whether a lot was inserted, incremented/adjusted via delta, or skipped.
type LotMigrationResult struct {
	LotID      string
	DeltaQty   int64
	Action     string // created | updated | skipped
	FinalOnHand int64
}

// ApplyTx exposes transactional entity read/write primitives inside one chunk ReadWriteTransaction.
type ApplyTx interface {
	Now() time.Time
	SupplierID() string
	SessionID() string
	ReadSupplierMeta(ctx context.Context) (homeCell, currency string, err error)

	LookupIdentity(ctx context.Context, pack Pack, extID string) (string, bool, error)
	PutIdentity(ctx context.Context, pack Pack, extID, internalID string) error
	RecordSnapshot(ctx context.Context, targetTable string, keyObj any, opKind string, beforeObj any) error

	FindWarehouse(ctx context.Context, extID, name string) (StoredWarehouse, bool, error)
	SaveWarehouse(ctx context.Context, wh StoredWarehouse, extPlantID string, before *StoredWarehouse) error

	FindProduct(ctx context.Context, internalID, sku, barcode, name string) (StoredProduct, bool, error)
	TotalProductOnHand(ctx context.Context, productID string) (int64, error)
	SaveProduct(ctx context.Context, p StoredProduct, before *StoredProduct) error

	FindCustomer(ctx context.Context, homeCell, extID, inn, phone string) (StoredCustomer, bool, error)
	SaveCustomer(ctx context.Context, ret StoredCustomer, createRetailer bool, beforeLink *StoredCustomerLink) error

	GetInventory(ctx context.Context, warehouseID, productID string) (StoredStock, bool, error)
	UpsertStock(ctx context.Context, stk StoredStock, before *StoredStock) error

	FindLot(ctx context.Context, warehouseID, productID, lotCode string) (StoredLot, bool, error)
	PutawayOrAdjustLot(ctx context.Context, op LotMigrationOp, existing *StoredLot, beforeStock *StoredStock) (LotMigrationResult, error)

	ResolveCustomerForFinancials(ctx context.Context, homeCell, extID, inn, phone, name string) (StoredCustomer, bool, error)
	ResolveWarehouseForOrder(ctx context.Context, whHint, defaultWarehouseID string) (StoredWarehouse, bool, error)
	ImportHistoricalOrder(ctx context.Context, req order.HistoricalOrderRequest) (created bool, updated bool, orderID string, err error)
	ImportMigratedOpenOrder(ctx context.Context, req order.MigratedOpenOrderRequest) (created bool, updated bool, orderID string, err error)
	OpenMigrationNasiyaAR(ctx context.Context, req ar.OpenFromMigrationRequest) (res *ar.OpenFromMigrationResult, err error)
}

// EntityApplyStore executes atomic chunk transactions across Spanner or in-memory test doubles.
type EntityApplyStore interface {
	RunChunkTx(ctx context.Context, supplierID, sessionID string, fn func(tx ApplyTx) error) error
}

// ApplyWarning surfaces non-blocking anomalies discovered during apply (e.g. cold lots without IoT baseline).
type ApplyWarning struct {
	RowIndex int64  `json:"row_index"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// ApplyOptions configures ApplySession execution.
type ApplyOptions struct {
	SupplierID        string
	SessionID         string
	ApplyMode         string
	ChunkSize         int
	LockTTL           time.Duration
	Now               time.Time
	ProgressEmit      func(sessionID string, processed, total int64, chunkID string)
	FailAfterChunk    int                                                                // optional fault-injection hook in tests for resume verification
	JournalOverrideFn func(in ar.MigrationJournalInput) (*payment.JournalEntry, error) // optional hook to test fail-closed journal invariants
}

// ApplyResult summarizes committed entities, chunks, warnings, and 72h rollback deadline.
type ApplyResult struct {
	Session        Session                `json:"session"`
	AlreadyApplied bool                   `json:"already_applied,omitempty"`
	Created        int64                  `json:"created"`
	Updated        int64                  `json:"updated"`
	Skipped        int64                  `json:"skipped"`
	ChunksApplied  int64                  `json:"chunks_applied"`
	LastRowIndex   int64                  `json:"last_row_index"`
	Warnings       []ApplyWarning         `json:"warnings,omitempty"`
	JournalEntries []*payment.JournalEntry `json:"journal_entries,omitempty"`
}

type applyChunkRow struct {
	RowIndex int64
	Fields   map[string]string
}

type applyRowContext struct {
	Session              Session
	DefaultWarehouseID   string
	ApplyMode            string
	HomeCell             string
	SupplierCurrency     string
	SeenKeys             map[string]bool
	NasiyaRetailerCounts map[string]int
	WarehouseLotsEnabled map[string]bool
	ChunkJournalEntries  []*payment.JournalEntry
	JournalOverrideFn    func(in ar.MigrationJournalInput) (*payment.JournalEntry, error)
	Result               *ApplyResult
}

func (c *applyRowContext) warehouseUsesLots(ctx context.Context, warehouseID string) bool {
	if c.WarehouseLotsEnabled != nil {
		if v, ok := c.WarehouseLotsEnabled[warehouseID]; ok {
			return v
		}
	} else {
		c.WarehouseLotsEnabled = make(map[string]bool)
	}
	v := stocklots.EffectiveLots(ctx, warehouseID, c.Session.SupplierID)
	c.WarehouseLotsEnabled[warehouseID] = v
	return v
}

func (c *applyRowContext) addWarning(rowIdx int64, code, msg string) {
	c.Result.Warnings = append(c.Result.Warnings, ApplyWarning{
		RowIndex: rowIdx,
		Code:     code,
		Message:  msg,
	})
}

// ApplySession commits an approved (or crash-resumed FAILED) session pack to production entities in chunked transactions,
// capturing pre-apply snapshots, acquiring per-warehouse locks for inventory packs, and setting a 72h rollback window.
func ApplySession(ctx context.Context, repo Repository, store EntityApplyStore, opts ApplyOptions) (ApplyResult, error) {
	if repo == nil || store == nil {
		return ApplyResult{}, fmt.Errorf("migration repository and apply store are required")
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = DefaultApplyChunkSize
	}
	lockTTL := opts.LockTTL
	if lockTTL <= 0 {
		lockTTL = DefaultWarehouseLockTTL
	}

	sess, err := repo.GetSession(ctx, opts.SupplierID, opts.SessionID)
	if err != nil {
		return ApplyResult{}, err
	}
	curStatus := NormalizeSessionStatus(string(sess.Status))
	if curStatus == SessionStatusApplied || curStatus == SessionStatusCutoverComplete {
		return ApplyResult{Session: sess, AlreadyApplied: true, LastRowIndex: sess.ResumeRowIndex}, nil
	}
	if curStatus != SessionStatusApproved && curStatus != SessionStatusFailed {
		return ApplyResult{}, fmt.Errorf("%w: session status %s cannot apply", ErrSessionNotApproved, curStatus)
	}
	if err := ValidateSessionTransition(curStatus, SessionStatusApplying); err != nil {
		return ApplyResult{}, err
	}

	proj, err := repo.GetProject(ctx, opts.SupplierID, sess.ProjectID)
	if err != nil {
		return ApplyResult{}, err
	}
	sessions, err := repo.ListSessionsByProject(ctx, opts.SupplierID, sess.ProjectID)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := ValidatePackApplyAllowed(sess.Pack, sessions); err != nil {
		return ApplyResult{}, err
	}

	doc, hasDoc, err := repo.GetSessionMapping(ctx, opts.SupplierID, opts.SessionID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !hasDoc {
		doc = SessionMappingDocument{SessionID: sess.SessionID, Pack: sess.Pack}
	}
	if err := ValidateMappingsReadyForApply(doc); err != nil {
		return ApplyResult{}, err
	}

	applyMode := NormalizeApplyMode(firstNonEmpty(opts.ApplyMode, sess.ApplyMode))
	if err := verifyApprovedSessionFreshness(ctx, repo, sess, doc, applyMode, now); err != nil {
		return ApplyResult{}, err
	}

	if _, err := repo.UpdateSessionStatus(ctx, opts.SupplierID, opts.SessionID, SessionStatusApplying, nil); err != nil {
		return ApplyResult{}, err
	}

	// Track per-warehouse locks acquired during inventory packs and release when ApplySession exits.
	heldLocks := make(map[string]bool)
	defer func() {
		for whID := range heldLocks {
			_ = repo.ReleaseWarehouseLock(context.Background(), opts.SupplierID, whID, opts.SessionID)
		}
	}()

	res := ApplyResult{LastRowIndex: sess.ResumeRowIndex}
	rctx := &applyRowContext{
		Session:              sess,
		DefaultWarehouseID:   strings.TrimSpace(proj.DefaultWarehouseID),
		ApplyMode:            applyMode,
		SeenKeys:             make(map[string]bool),
		NasiyaRetailerCounts: make(map[string]int),
		JournalOverrideFn:    opts.JournalOverrideFn,
		Result:               &res,
	}

	if err := executeSessionChunks(ctx, repo, store, rctx, doc, chunkSize, lockTTL, opts.FailAfterChunk, opts.ProgressEmit, heldLocks); err != nil {
		errRaw, _ := jsonMarshalErr(err)
		_, _ = repo.UpdateSessionStatus(ctx, opts.SupplierID, opts.SessionID, SessionStatusFailed, errRaw)
		return ApplyResult{}, err
	}

	rollbackUntil := now.Add(RollbackWindowTTL)
	if err := repo.MarkSessionApplied(ctx, opts.SupplierID, opts.SessionID, rollbackUntil); err != nil {
		return ApplyResult{}, err
	}
	finalSess, err := repo.GetSession(ctx, opts.SupplierID, opts.SessionID)
	if err == nil {
		res.Session = finalSess
	}
	return res, nil
}

func verifyApprovedSessionFreshness(ctx context.Context, repo Repository, sess Session, doc SessionMappingDocument, applyMode string, now time.Time) error {
	if sess.Pack == PackLots && applyMode == ApplyModeSetOnHand {
		return fmt.Errorf("%w: SET_ON_HAND forbidden on lots pack", ErrInvalidInput)
	}
	if applyMode == ApplyModeSetOnHand && !sess.SetOnHandConfirmed {
		return ErrSetOnHandConfirmationRequired
	}
	if strings.TrimSpace(sess.DryRunReportID) == "" {
		return fmt.Errorf("%w: missing approved dry_run_report_id", ErrDryRunStale)
	}
	rep, err := repo.GetMigrationReport(ctx, sess.SupplierID, sess.DryRunReportID)
	if err != nil {
		return err
	}
	if !now.UTC().Before(rep.ExpiresAt.UTC()) {
		return fmt.Errorf("%w: report %s expired at %s", ErrDryRunExpired, rep.ReportID, rep.ExpiresAt.Format(time.RFC3339))
	}
	if rep.BlockingCount > 0 {
		return fmt.Errorf("%w: report %s still has %d blocking conflict(s)", ErrDryRunBlockingConflicts, rep.ReportID, rep.BlockingCount)
	}
	fp := ComputeSessionDryRunFingerprint(sess, doc, applyMode)
	if fp != rep.ContentFingerprint || (sess.DryRunFingerprint != "" && fp != sess.DryRunFingerprint) {
		return fmt.Errorf("%w: content or mapping hash changed since approval", ErrDryRunStale)
	}
	return nil
}

func executeSessionChunks(
	ctx context.Context,
	repo Repository,
	store EntityApplyStore,
	rctx *applyRowContext,
	doc SessionMappingDocument,
	chunkSize int,
	lockTTL time.Duration,
	failAfterChunk int,
	progressEmit func(string, int64, int64, string),
	heldLocks map[string]bool,
) error {
	var allPending []StagedRow
	hasStarted := strings.TrimSpace(rctx.Session.LastChunkID) != "" || rctx.Session.ResumeRowIndex > 0
	startAfter := int64(-1)
	if hasStarted {
		startAfter = rctx.Session.ResumeRowIndex
	}
	if err := streamAllStagedRows(ctx, repo, rctx.Session.SupplierID, rctx.Session.SessionID, startAfter, dryRunRowPageSize, func(rows []StagedRow) error {
		for _, sr := range rows {
			// Staged rows are 0-based (ingest_worker) or 1-based; only skip already-checkpointed prefixes once apply has started.
			if !hasStarted || sr.RowIndex > rctx.Session.ResumeRowIndex {
				allPending = append(allPending, sr)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	chunkIdx := 0
	for start := 0; start < len(allPending); start += chunkSize {
		chunkIdx++
		if failAfterChunk > 0 && chunkIdx > failAfterChunk {
			return fmt.Errorf("injected_chunk_failure_after_%d", failAfterChunk)
		}
		end := start + chunkSize
		if end > len(allPending) {
			end = len(allPending)
		}
		chunk := allPending[start:end]
		chunkID := fmt.Sprintf("chk_%s_%d_%d", rctx.Session.SessionID, chunk[0].RowIndex, chunk[len(chunk)-1].RowIndex)

		if rctx.Session.Pack == PackStock || rctx.Session.Pack == PackLots {
			if err := lockWarehousesForChunk(ctx, repo, rctx, doc, chunk, lockTTL, heldLocks); err != nil {
				return err
			}
		}

		chunkFields := make([]applyChunkRow, len(chunk))
		for i, sr := range chunk {
			chunkFields[i] = applyChunkRow{
				RowIndex: sr.RowIndex,
				Fields:   ExtractCanonicalRow(sr, doc),
			}
		}

		baseCounts := make(map[string]int, len(rctx.NasiyaRetailerCounts))
		for k, v := range rctx.NasiyaRetailerCounts {
			baseCounts[k] = v
		}
		baseSeen := make(map[string]bool, len(rctx.SeenKeys))
		for k, v := range rctx.SeenKeys {
			baseSeen[k] = v
		}
		baseCreated := rctx.Result.Created
		baseUpdated := rctx.Result.Updated
		baseSkipped := rctx.Result.Skipped
		baseWarnLen := len(rctx.Result.Warnings)
		baseDefaultWH := rctx.DefaultWarehouseID

		err := store.RunChunkTx(ctx, rctx.Session.SupplierID, rctx.Session.SessionID, func(tx ApplyTx) error {
			// Reset all in-memory accumulators on every Spanner ReadWriteTransaction retry
			// so aborted attempts cannot leak SeenKeys skips, duplicate warnings, or inflated counts.
			rctx.ChunkJournalEntries = nil
			rctx.Result.Created = baseCreated
			rctx.Result.Updated = baseUpdated
			rctx.Result.Skipped = baseSkipped
			rctx.Result.Warnings = rctx.Result.Warnings[:baseWarnLen]
			rctx.DefaultWarehouseID = baseDefaultWH
			rctx.SeenKeys = make(map[string]bool, len(baseSeen))
			for k, v := range baseSeen {
				rctx.SeenKeys[k] = v
			}
			rctx.NasiyaRetailerCounts = make(map[string]int, len(baseCounts))
			for k, v := range baseCounts {
				rctx.NasiyaRetailerCounts[k] = v
			}
			hc, cur, err := tx.ReadSupplierMeta(ctx)
			if err != nil {
				return err
			}
			rctx.HomeCell = hc
			if cur != "" {
				rctx.SupplierCurrency = strings.ToUpper(cur)
			} else if rctx.SupplierCurrency == "" {
				rctx.SupplierCurrency = "UZS"
			}
			if preloader, ok := tx.(interface {
				PreloadChunk(ctx context.Context, pack Pack, homeCell, defaultWarehouseID string, rows []applyChunkRow) error
			}); ok {
				if err := preloader.PreloadChunk(ctx, rctx.Session.Pack, rctx.HomeCell, rctx.DefaultWarehouseID, chunkFields); err != nil {
					return err
				}
			}
			for _, cr := range chunkFields {
				if err := applyPackRow(ctx, tx, rctx, cr.RowIndex, cr.Fields); err != nil {
					return fmt.Errorf("row %d: %w", cr.RowIndex, err)
				}
			}
			return nil
		})
		if err != nil {
			rctx.ChunkJournalEntries = nil
			return err
		}
		if len(rctx.ChunkJournalEntries) > 0 {
			rctx.Result.JournalEntries = append(rctx.Result.JournalEntries, rctx.ChunkJournalEntries...)
			rctx.ChunkJournalEntries = nil
		}

		lastIdx := chunk[len(chunk)-1].RowIndex
		if err := repo.UpdateApplyProgress(ctx, rctx.Session.SupplierID, rctx.Session.SessionID, lastIdx, chunkID); err != nil {
			return err
		}
		rctx.Result.ChunksApplied++
		rctx.Result.LastRowIndex = lastIdx
		rctx.Session.ResumeRowIndex = lastIdx
		rctx.Session.LastChunkID = chunkID
		if progressEmit != nil {
			progressEmit(rctx.Session.SessionID, lastIdx, rctx.Session.TotalRows, chunkID)
		}
	}
	return nil
}

func lockWarehousesForChunk(
	ctx context.Context,
	repo Repository,
	rctx *applyRowContext,
	doc SessionMappingDocument,
	chunk []StagedRow,
	lockTTL time.Duration,
	heldLocks map[string]bool,
) error {
	for whID := range heldLocks {
		if _, err := repo.RenewWarehouseLock(ctx, rctx.Session.SupplierID, whID, rctx.Session.SessionID, lockTTL); err != nil {
			return err
		}
	}
	// Pre-acquire default warehouse if referenced directly or by omission.
	if rctx.DefaultWarehouseID != "" && !heldLocks[rctx.DefaultWarehouseID] {
		if _, err := repo.AcquireWarehouseLock(ctx, rctx.Session.SupplierID, rctx.DefaultWarehouseID, rctx.Session.SessionID, lockTTL); err != nil {
			return err
		}
		heldLocks[rctx.DefaultWarehouseID] = true
	}
	for _, sr := range chunk {
		fields := ExtractCanonicalRow(sr, doc)
		ref := strings.TrimSpace(fields["warehouse_id"])
		if ref == "" {
			continue
		}
		token := ref
		if !heldLocks[token] {
			if _, err := repo.AcquireWarehouseLock(ctx, rctx.Session.SupplierID, token, rctx.Session.SessionID, lockTTL); err != nil {
				return err
			}
			heldLocks[token] = true
		}
	}
	return nil
}

func applyPackRow(ctx context.Context, tx ApplyTx, rctx *applyRowContext, rowIdx int64, fields map[string]string) error {
	switch rctx.Session.Pack {
	case PackWarehouses:
		return applyWarehouseRow(ctx, tx, rctx, rowIdx, fields)
	case PackProductsPrices:
		return applyCatalogRow(ctx, tx, rctx, rowIdx, fields)
	case PackCustomers:
		return applyCustomerRow(ctx, tx, rctx, rowIdx, fields)
	case PackStock:
		return applyStockRow(ctx, tx, rctx, rowIdx, fields)
	case PackLots:
		return applyLotRow(ctx, tx, rctx, rowIdx, fields)
	case PackHistoricalOrders:
		return applyHistoricalOrderRow(ctx, tx, rctx, rowIdx, fields)
	case PackNasiyaAR:
		return applyNasiyaRow(ctx, tx, rctx, rowIdx, fields)
	case PackOpenOrders:
		return applyOpenOrderRow(ctx, tx, rctx, rowIdx, fields)
	default:
		return fmt.Errorf("%w: apply not yet implemented for pack %q", ErrInvalidInput, rctx.Session.Pack)
	}
}

func deterministicEntityID(prefix, supplierID, pack, key string) string {
	sum := sha256.Sum256([]byte(supplierID + "|" + pack + "|" + strings.ToLower(strings.TrimSpace(key))))
	hexPart := hex.EncodeToString(sum[:12])
	if prefix == "" {
		return hexPart
	}
	id := prefix + "_" + hexPart
	if len(id) > 36 {
		id = id[:36]
	}
	return id
}

func jsonMarshalErr(err error) ([]byte, error) {
	if err == nil {
		return nil, nil
	}
	return []byte(fmt.Sprintf(`{"apply_error":%q}`, err.Error())), nil
}
