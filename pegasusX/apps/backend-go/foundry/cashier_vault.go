package foundry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	CashierStatusAwaitingBlindCount1 = "AWAITING_BLIND_COUNT_1"
	CashierStatusBlindRecountReq     = "BLIND_RECOUNT_REQUIRED"
	CashierStatusBalanced            = "BALANCED"
	CashierStatusToleranceVariance   = "TOLERANCE_VARIANCE"
	CashierStatusDiscrepantQuarantine = "DISCREPANT_QUARANTINE"
	CashierStatusRegulatoryAMLHold   = "REGULATORY_AML_HOLD"

	VaultEntryDriverDropInflow = "DRIVER_DROP_INFLOW"
	VaultEntryCITOutflow       = "CIT_ARMORED_OUTFLOW"
)

// Valid Uzbek Som (UZS) physical banknote denominations.
var validUZSDenominations = map[int64]bool{
	200000: true,
	100000: true,
	50000:  true,
	20000:  true,
	10000:  true,
	5000:   true,
	2000:   true,
	1000:   true,
}

// RecountRequiredError is returned when Attempt #1 blind count fails variance check (UA-1).
type RecountRequiredError struct {
	Session *CashierSessionView `json:"session"`
}

func (e *RecountRequiredError) Error() string {
	return "variance exceeds tolerance; blind recount #2 required"
}

func (e *RecountRequiredError) Unwrap() error {
	return ErrBlindRecountRequired
}

// CashierBlindCountSession stores internal state for Role 7 (Depot Cashier).
type CashierBlindCountSession struct {
	SessionID                 string          `json:"session_id" spanner:"SessionId"`
	SupplierID                string          `json:"supplier_id" spanner:"SupplierId"`
	DepotID                   string          `json:"depot_id" spanner:"DepotId"`
	DriverID                  string          `json:"driver_id" spanner:"DriverId"`
	RouteID                   string          `json:"route_id" spanner:"RouteId"`
	BagSealBarcode            string          `json:"bag_seal_barcode" spanner:"BagSealBarcode"`
	AttemptNumber             int64           `json:"attempt_number" spanner:"AttemptNumber"`
	DenominationsUZS          map[int64]int64 `json:"denominations_uzs,omitempty" spanner:"-"`
	DenominationBreakdownJson string          `json:"denomination_breakdown_json,omitempty" spanner:"DenominationBreakdownJson"`
	CountedCashMinor          int64           `json:"counted_cash_minor" spanner:"CountedCashMinor"`
	ExpectedCashMinor         int64           `json:"-" spanner:"ExpectedCashMinor"` // Never serialized directly to blind cashier view!
	VarianceMinor             int64           `json:"-" spanner:"VarianceMinor"`
	Status                    string          `json:"status" spanner:"Status"`
	ClearanceTokenHash        *string         `json:"clearance_token_hash,omitempty" spanner:"ClearanceTokenHash"`
	CreatedAt                 time.Time       `json:"created_at" spanner:"CreatedAt"`
	ResolvedAt                *time.Time      `json:"resolved_at,omitempty" spanner:"ResolvedAt"`
	CellID                    string          `json:"cell_id" spanner:"-"`
}

type CashierBlindCountSessions = CashierBlindCountSession

func (c *CashierBlindCountSession) PrepareForSpanner() error {
	if c.DenominationBreakdownJson == "" && c.DenominationsUZS != nil {
		b, err := json.Marshal(c.DenominationsUZS)
		if err != nil {
			return err
		}
		c.DenominationBreakdownJson = string(b)
	} else if c.DenominationBreakdownJson != "" && c.DenominationsUZS == nil {
		var m map[int64]int64
		if err := json.Unmarshal([]byte(c.DenominationBreakdownJson), &m); err == nil {
			c.DenominationsUZS = m
		}
	}
	return nil
}

// CashierSessionView is the safe projection returned to the Cashier UI.
// ExpectedCashMinor and VarianceMinor remain nil until the dual-blind recount ratchet finishes.
type CashierSessionView struct {
	SessionID             string          `json:"session_id"`
	SupplierID            string          `json:"supplier_id"`
	DepotID               string          `json:"depot_id"`
	DriverID              string          `json:"driver_id"`
	RouteID               string          `json:"route_id"`
	BagSealBarcode        string          `json:"bag_seal_barcode"`
	AttemptNumber         int64           `json:"attempt_number"`
	CountedCashMinor      int64           `json:"counted_cash_minor"`
	ExpectedCashMinor     *int64          `json:"expected_cash_minor,omitempty"`
	VarianceMinor         *int64          `json:"variance_minor,omitempty"`
	Status                string          `json:"status"`
	ClearanceTokenHash    *string         `json:"clearance_token_hash,omitempty"`
	MicroSkimmingFlagged  bool            `json:"micro_skimming_flagged,omitempty"`
	ErrorCode             string          `json:"error_code,omitempty"`
	ErrorMessage          string          `json:"error_message,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	ResolvedAt            *time.Time      `json:"resolved_at,omitempty"`
}

// StartCashierSessionRequest initializes a blind count session when scanning a driver's cash bag.
type StartCashierSessionRequest struct {
	CellID            string `json:"cell_id"`
	SupplierID        string `json:"supplier_id"`
	DepotID           string `json:"depot_id"`
	DriverID          string `json:"driver_id"`
	RouteID           string `json:"route_id"`
	BagSealBarcode    string `json:"bag_seal_barcode"`
	ExpectedCashMinor int64  `json:"expected_cash_minor"`
}

// DepotVaultLedgerEntry tracks physical cash balance in the depot safe and CIT handovers.
type DepotVaultLedgerEntry struct {
	VaultEntryID        string    `json:"vault_entry_id" spanner:"VaultEntryId"`
	DepotID             string    `json:"depot_id" spanner:"DepotId"`
	SupplierID          string    `json:"supplier_id" spanner:"SupplierId"`
	SessionID           *string   `json:"session_id,omitempty" spanner:"SessionId"`
	CITManifestID       *string   `json:"cit_manifest_id,omitempty" spanner:"CITManifestId"`
	EntryType           string    `json:"entry_type" spanner:"EntryType"`
	AmountMinor         int64     `json:"amount_minor" spanner:"AmountMinor"`
	RunningBalanceMinor int64     `json:"running_balance_minor" spanner:"RunningBalanceMinor"`
	CourierBadgeID      *string   `json:"courier_badge_id,omitempty" spanner:"CourierBadgeId"`
	SealBarcode         *string   `json:"seal_barcode,omitempty" spanner:"SealBarcode"`
	CreatedAt           time.Time `json:"created_at" spanner:"CreatedAt"`
}

type DepotVaultLedgerEntries = DepotVaultLedgerEntry

// RRNTenderReclassificationRequest reclassifies a mistakenly entered CASH leg to CARD using a verified RRN.
type RRNTenderReclassificationRequest struct {
	SessionID         string    `json:"session_id"`
	OrderID           string    `json:"order_id"`
	DisputedCashMinor int64     `json:"disputed_cash_minor"`
	GatewayRRN        string    `json:"gateway_rrn"`
	RRNAmountMinor    int64     `json:"rrn_amount_minor"`
	EPODCapturedAt    time.Time `json:"epod_captured_at"`
	RRNCapturedAt     time.Time `json:"rrn_captured_at"`
	ActorID           string    `json:"actor_id"`
}

// CITHandoverRequest consolidates cleared depot vault cash for armored courier pickup.
type CITHandoverRequest struct {
	DepotID        string   `json:"depot_id"`
	SupplierID     string   `json:"supplier_id"`
	SessionIDs     []string `json:"session_ids"`
	CourierBadgeID string   `json:"courier_badge_id"`
	CITSealBarcode string   `json:"cit_seal_barcode"`
	AmountMinor    int64    `json:"amount_minor"`
}

func toCashierView(s *CashierBlindCountSession, revealExpected bool, skimmingFlagged bool) *CashierSessionView {
	v := &CashierSessionView{
		SessionID:            s.SessionID,
		SupplierID:           s.SupplierID,
		DepotID:              s.DepotID,
		DriverID:             s.DriverID,
		RouteID:              s.RouteID,
		BagSealBarcode:       s.BagSealBarcode,
		AttemptNumber:        s.AttemptNumber,
		CountedCashMinor:     s.CountedCashMinor,
		Status:               s.Status,
		ClearanceTokenHash:   s.ClearanceTokenHash,
		MicroSkimmingFlagged: skimmingFlagged,
		CreatedAt:            s.CreatedAt,
		ResolvedAt:           s.ResolvedAt,
	}
	if revealExpected {
		exp := s.ExpectedCashMinor
		diff := s.VarianceMinor
		v.ExpectedCashMinor = &exp
		v.VarianceMinor = &diff
	}
	return v
}

// StartBlindCountSession binds a tamper-evident bag seal and opens a blind count session.
// ExpectedCashMinor is stored server-side and omitted from the returned view.
func (e *Engine) StartBlindCountSession(ctx context.Context, req StartCashierSessionRequest) (*CashierSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	seal := strings.TrimSpace(req.BagSealBarcode)
	driverID := strings.TrimSpace(req.DriverID)
	depotID := strings.TrimSpace(req.DepotID)
	if seal == "" || driverID == "" || depotID == "" {
		return nil, fmt.Errorf("%w: bag_seal_barcode, driver_id, and depot_id are required", ErrPreconditionFailed)
	}
	if req.ExpectedCashMinor < 0 {
		return nil, fmt.Errorf("%w: expected_cash_minor cannot be negative", ErrPreconditionFailed)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.bagSeals[seal]; exists {
		return nil, ErrDuplicateBagSeal
	}

	cellID := strings.TrimSpace(req.CellID)
	if cellID == "" {
		cellID = "cell-uz"
	}

	repo := e.spannerRepo
	expectedMinor := req.ExpectedCashMinor
	if repo != nil && strings.TrimSpace(req.RouteID) != "" {
		if dbExpected, err := repo.ResolveRouteExpectedCash(ctx, strings.TrimSpace(req.SupplierID), driverID, strings.TrimSpace(req.RouteID)); err == nil && dbExpected > 0 {
			expectedMinor = dbExpected
		}
	}

	sess := &CashierBlindCountSession{
		SessionID:         uuid.NewString(),
		CellID:            cellID,
		SupplierID:        strings.TrimSpace(req.SupplierID),
		DepotID:           depotID,
		DriverID:          driverID,
		RouteID:           strings.TrimSpace(req.RouteID),
		BagSealBarcode:    seal,
		AttemptNumber:     0,
		DenominationsUZS:  make(map[int64]int64),
		ExpectedCashMinor: expectedMinor,
		Status:            CashierStatusAwaitingBlindCount1,
		CreatedAt:         e.now(),
	}
	e.bagSeals[seal] = sess.SessionID
	e.cashierSessions[sess.SessionID] = sess
	view := toCashierView(sess, false, false)

	if repo != nil {
		_ = repo.PersistCashierSession(ctx, sess)
	}
	return view, nil
}

// SubmitBlindDenominationCount executes the Blind Denomination Counting Ritual and Dual-Blind Recount Ratchet (Guard #11).
func (e *Engine) SubmitBlindDenominationCount(
	ctx context.Context,
	sessionID string,
	denominationsUZS map[int64]int64,
	toleranceThresholdMinor int64,
) (*CashierSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(denominationsUZS) == 0 {
		return nil, fmt.Errorf("%w: denomination counts required", ErrPreconditionFailed)
	}

	var countedMinor int64
	cleanDenoms := make(map[int64]int64, len(denominationsUZS))
	for noteUZS, count := range denominationsUZS {
		if !validUZSDenominations[noteUZS] {
			return nil, fmt.Errorf("%w: invalid UZS banknote denomination %d", ErrPreconditionFailed, noteUZS)
		}
		if count < 0 {
			return nil, fmt.Errorf("%w: negative banknote count for denomination %d", ErrPreconditionFailed, noteUZS)
		}
		// 1 UZS = 100 Tiyin (integer minor units).
		countedMinor += noteUZS * count * 100
		cleanDenoms[noteUZS] = count
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	sess, ok := e.cashierSessions[strings.TrimSpace(sessionID)]
	if !ok {
		return nil, fmt.Errorf("%w: cashier session %s", ErrObjectNotFound, sessionID)
	}
	if sess.Status != CashierStatusAwaitingBlindCount1 && sess.Status != CashierStatusBlindRecountReq {
		return nil, fmt.Errorf("%w: session %s is already finalized in status %s", ErrPreconditionFailed, sessionID, sess.Status)
	}

	sess.AttemptNumber++
	sess.DenominationsUZS = cleanDenoms
	sess.CountedCashMinor = countedMinor
	sess.VarianceMinor = countedMinor - sess.ExpectedCashMinor
	absVariance := sess.VarianceMinor
	if absVariance < 0 {
		absVariance = -absVariance
	}

	// Case 1: Exact match (Zero Variance).
	if sess.VarianceMinor == 0 {
		now := e.now()
		sess.Status = CashierStatusBalanced
		sess.ResolvedAt = &now
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%d", sess.SessionID, sess.DriverID, countedMinor, now.UnixNano())))
		token := hex.EncodeToString(hash[:])
		sess.ClearanceTokenHash = &token
		e.driverShiftVariances[sess.DriverID] = append(e.driverShiftVariances[sess.DriverID], 0)
		if len(e.driverShiftVariances[sess.DriverID]) > 5 {
			e.driverShiftVariances[sess.DriverID] = e.driverShiftVariances[sess.DriverID][len(e.driverShiftVariances[sess.DriverID])-5:]
		}
		e.recordVaultInflowLocked(sess)
		return toCashierView(sess, true, false), nil
	}

	// Case 2: Within minor tolerance threshold. Check for systematic micro-skimming (Guard #11).
	if absVariance <= toleranceThresholdMinor {
		e.driverShiftVariances[sess.DriverID] = append(e.driverShiftVariances[sess.DriverID], sess.VarianceMinor)
		if len(e.driverShiftVariances[sess.DriverID]) > 5 {
			e.driverShiftVariances[sess.DriverID] = e.driverShiftVariances[sess.DriverID][len(e.driverShiftVariances[sess.DriverID])-5:]
		}
		if sess.VarianceMinor < 0 && e.isDriverMicroSkimmingLocked(sess.DriverID) {
			sess.Status = CashierStatusDiscrepantQuarantine
			e.upsertObjectLocked(sess.CellID, "Driver", sess.DriverID, sess.SupplierID, map[string]any{
				"dispatch_locked": true,
				"lock_reason":     "SYSTEMATIC_TOLERANCE_MICRO_SKIMMING",
			}, []string{MarkingMicroSkimmingSuspect, MarkingDiscrepantQuarantine})
			return toCashierView(sess, true, true), nil
		}

		now := e.now()
		sess.Status = CashierStatusToleranceVariance
		sess.ResolvedAt = &now
		e.recordVaultInflowLocked(sess)
		return toCashierView(sess, true, false), nil
	}

	// Case 3: Variance exceeds tolerance on Attempt #1 -> Dual-Blind Recount Ratchet!
	// ExpectedCashMinor MUST remain hidden so the cashier cannot fudge Attempt #2.
	if sess.AttemptNumber == 1 {
		sess.Status = CashierStatusBlindRecountReq
		v := toCashierView(sess, false, false)
		v.ErrorCode = "RECOUNT_REQUIRED"
		v.ErrorMessage = "variance exceeds tolerance; blind recount #2 required"
		return v, &RecountRequiredError{Session: v}
	}

	// Case 4: Attempt #2 still has unacceptable variance -> Lock into DISCREPANT_QUARANTINE and reveal to Supervisor.
	sess.Status = CashierStatusDiscrepantQuarantine
	e.driverShiftVariances[sess.DriverID] = append(e.driverShiftVariances[sess.DriverID], sess.VarianceMinor)
	if len(e.driverShiftVariances[sess.DriverID]) > 5 {
		e.driverShiftVariances[sess.DriverID] = e.driverShiftVariances[sess.DriverID][len(e.driverShiftVariances[sess.DriverID])-5:]
	}
	e.upsertObjectLocked(sess.CellID, "Driver", sess.DriverID, sess.SupplierID, map[string]any{
		"dispatch_locked": true,
		"lock_reason":     "CASH_SHORTAGE_QUARANTINE",
		"variance_minor":  sess.VarianceMinor,
	}, []string{MarkingDiscrepantQuarantine})

	return toCashierView(sess, true, false), nil
}

func (e *Engine) isDriverMicroSkimmingLocked(driverID string) bool {
	history := e.driverShiftVariances[driverID]
	if len(history) < 4 {
		return false
	}
	recent := history
	if len(recent) > 5 {
		recent = recent[len(recent)-5:]
	}
	negCount := 0
	for _, v := range recent {
		if v < 0 {
			negCount++
		}
	}
	// Flag if >= 80% of last 4-5 shifts had negative variance.
	return negCount*100/len(recent) >= 80
}

func (e *Engine) recordVaultInflowLocked(sess *CashierBlindCountSession) {
	newBal := e.vaultBalances[sess.DepotID] + sess.CountedCashMinor
	e.vaultBalances[sess.DepotID] = newBal
	sessID := sess.SessionID
	seal := sess.BagSealBarcode
	entry := &DepotVaultLedgerEntry{
		VaultEntryID:        uuid.NewString(),
		DepotID:             sess.DepotID,
		SupplierID:          sess.SupplierID,
		SessionID:           &sessID,
		EntryType:           VaultEntryDriverDropInflow,
		AmountMinor:         sess.CountedCashMinor,
		RunningBalanceMinor: newBal,
		SealBarcode:         &seal,
		CreatedAt:           e.now(),
	}
	e.vaultLedger[sess.DepotID] = append(e.vaultLedger[sess.DepotID], entry)
	if len(e.vaultLedger[sess.DepotID]) > 1000 {
		e.vaultLedger[sess.DepotID] = e.vaultLedger[sess.DepotID][len(e.vaultLedger[sess.DepotID])-1000:]
	}
}

// ReclassifyCashToCardByRRN resolves a cash shortage without destructive edits when a driver
// accidentally tapped CASH instead of CARD, verified by a payment gateway RRN within ±15m of ePOD (Guard #5).
func (e *Engine) ReclassifyCashToCardByRRN(
	ctx context.Context,
	req RRNTenderReclassificationRequest,
) (*CashierSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GatewayRRN) == "" || req.DisputedCashMinor <= 0 {
		return nil, ErrInvalidRRNProof
	}
	if req.RRNAmountMinor != req.DisputedCashMinor {
		return nil, fmt.Errorf("%w: RRN amount %d does not match disputed cash %d", ErrInvalidRRNProof, req.RRNAmountMinor, req.DisputedCashMinor)
	}
	timeDiff := req.RRNCapturedAt.Sub(req.EPODCapturedAt)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > 15*time.Minute {
		return nil, fmt.Errorf("%w: RRN timestamp drift %s exceeds 15m window", ErrInvalidRRNProof, timeDiff)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	sess, ok := e.cashierSessions[strings.TrimSpace(req.SessionID)]
	if !ok {
		return nil, fmt.Errorf("%w: cashier session %s", ErrObjectNotFound, req.SessionID)
	}

	// Adjust expected cash via compensating ledger adjustment rather than mutating historical payment leg.
	sess.ExpectedCashMinor -= req.DisputedCashMinor
	sess.VarianceMinor = sess.CountedCashMinor - sess.ExpectedCashMinor
	if sess.VarianceMinor == 0 {
		now := e.now()
		sess.Status = CashierStatusBalanced
		sess.ResolvedAt = &now
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:RRN:%s:%d", sess.SessionID, req.GatewayRRN, now.UnixNano())))
		token := hex.EncodeToString(hash[:])
		sess.ClearanceTokenHash = &token
		e.recordVaultInflowLocked(sess)
	}

	return toCashierView(sess, true, false), nil
}

// ExecuteCITArmoredHandover records an armored bank courier pickup from the depot vault.
func (e *Engine) ExecuteCITArmoredHandover(ctx context.Context, req CITHandoverRequest) (*DepotVaultLedgerEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	depotID := strings.TrimSpace(req.DepotID)
	badge := strings.TrimSpace(req.CourierBadgeID)
	seal := strings.TrimSpace(req.CITSealBarcode)
	if depotID == "" || badge == "" || seal == "" || req.AmountMinor <= 0 {
		return nil, fmt.Errorf("%w: depot_id, courier_badge_id, cit_seal_barcode, and positive amount_minor required", ErrPreconditionFailed)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	currBal := e.vaultBalances[depotID]
	if req.AmountMinor > currBal {
		return nil, fmt.Errorf("%w: CIT withdrawal %d exceeds depot vault balance %d", ErrPreconditionFailed, req.AmountMinor, currBal)
	}

	newBal := currBal - req.AmountMinor
	e.vaultBalances[depotID] = newBal
	citManifestID := uuid.NewString()
	entry := &DepotVaultLedgerEntry{
		VaultEntryID:        uuid.NewString(),
		DepotID:             depotID,
		SupplierID:          strings.TrimSpace(req.SupplierID),
		CITManifestID:       &citManifestID,
		EntryType:           VaultEntryCITOutflow,
		AmountMinor:         -req.AmountMinor,
		RunningBalanceMinor: newBal,
		CourierBadgeID:      &badge,
		SealBarcode:         &seal,
		CreatedAt:           e.now(),
	}
	e.vaultLedger[depotID] = append(e.vaultLedger[depotID], entry)
	if len(e.vaultLedger[depotID]) > 1000 {
		e.vaultLedger[depotID] = e.vaultLedger[depotID][len(e.vaultLedger[depotID])-1000:]
	}
	return entry, nil
}
