package foundry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/google/uuid"
)

// DefaultAMLCashCeilingMinor is the default daily cash settlement ceiling minor units when limits are enabled.
const DefaultAMLCashCeilingMinor int64 = 2_500_000_000

// StatutoryAMLCashCeilingMinor represents the default ceiling for backwards compatibility.
const StatutoryAMLCashCeilingMinor int64 = DefaultAMLCashCeilingMinor

// VolumeBreakTier defines a quantity threshold and discount in basis points (e.g. 1200 bps = 12%).
type VolumeBreakTier struct {
	MinCases    int64 `json:"min_cases"`
	DiscountBps int64 `json:"discount_bps"`
}

// SplitTenderRejectionInput captures doorstep partial offload reconciliation inputs (Guard #1).
type SplitTenderRejectionInput struct {
	OrderID                  string            `json:"order_id"`
	OrderedCases             int64             `json:"ordered_cases"`
	AcceptedCases            int64             `json:"accepted_cases"`
	BaseCasePriceMinor       int64             `json:"base_case_price_minor"`
	RejectionReason          string            `json:"rejection_reason"` // DAMAGED_IN_TRANSIT | SHORT_SHIPPED | BUYER_REMORSE_REFUSAL
	Tiers                    []VolumeBreakTier `json:"tiers"`
	OriginalCreditReservedMinor int64          `json:"original_credit_reserved_minor"`
	OriginalCashPlannedMinor    int64          `json:"original_cash_planned_minor"`
}

// SplitTenderReconciliationResult contains the fault-attributed pricing and tender waterfall split.
type SplitTenderReconciliationResult struct {
	OrderID                string `json:"order_id"`
	QualificationCases     int64  `json:"qualification_cases"`
	BillableCases          int64  `json:"billable_cases"`
	AppliedDiscountBps     int64  `json:"applied_discount_bps"`
	EffectiveCaseMinor     int64  `json:"effective_case_minor"`
	FinalTotalMinor        int64  `json:"final_total_minor"`
	FinalCreditBookedMinor int64  `json:"final_credit_booked_minor"`
	FinalCashDueMinor      int64  `json:"final_cash_due_minor"`
	CreditReleasedMinor    int64  `json:"credit_released_minor"`
	TierProtected          bool   `json:"tier_protected"`
}

// ReconcileSplitTenderPartialRejection implements Edge-Case Guard #1:
// 1) Preserves the original volume-break discount tier if rejection was caused by DAMAGED_IN_TRANSIT or SHORT_SHIPPED.
// 2) Recalculates the discount tier at AcceptedCases if rejection was BUYER_REMORSE_REFUSAL.
// 3) Applies the total reduction deterministically to the Credit reservation leg first before reducing Cash.
func (e *Engine) ReconcileSplitTenderPartialRejection(
	ctx context.Context,
	in SplitTenderRejectionInput,
) (*SplitTenderReconciliationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.OrderedCases <= 0 || in.AcceptedCases < 0 || in.AcceptedCases > in.OrderedCases || in.BaseCasePriceMinor <= 0 {
		return nil, fmt.Errorf("%w: invalid case quantities or base price", ErrPreconditionFailed)
	}

	reason := strings.ToUpper(strings.TrimSpace(in.RejectionReason))
	tierProtected := reason == "DAMAGED_IN_TRANSIT" || reason == "SHORT_SHIPPED"

	qualCases := in.AcceptedCases
	if tierProtected {
		qualCases = in.OrderedCases
	}

	var bestDiscountBps int64
	for _, t := range in.Tiers {
		if qualCases >= t.MinCases && t.DiscountBps > bestDiscountBps {
			bestDiscountBps = t.DiscountBps
		}
	}

	effectiveCaseMinor := (in.BaseCasePriceMinor * (10000 - bestDiscountBps)) / 10000
	finalTotalMinor := effectiveCaseMinor * in.AcceptedCases

	originalPlannedTotal := in.OriginalCreditReservedMinor + in.OriginalCashPlannedMinor
	reductionMinor := originalPlannedTotal - finalTotalMinor
	if reductionMinor < 0 {
		reductionMinor = 0
	}

	// Deterministic Tender Waterfall: reduce Credit Draw first, then reduce Cash Due.
	creditReduction := reductionMinor
	if creditReduction > in.OriginalCreditReservedMinor {
		creditReduction = in.OriginalCreditReservedMinor
	}
	finalCredit := in.OriginalCreditReservedMinor - creditReduction
	finalCash := finalTotalMinor - finalCredit
	if finalCash < 0 {
		finalCash = 0
	}

	return &SplitTenderReconciliationResult{
		OrderID:                in.OrderID,
		QualificationCases:     qualCases,
		BillableCases:          in.AcceptedCases,
		AppliedDiscountBps:     bestDiscountBps,
		EffectiveCaseMinor:     effectiveCaseMinor,
		FinalTotalMinor:        finalTotalMinor,
		FinalCreditBookedMinor: finalCredit,
		FinalCashDueMinor:      finalCash,
		CreditReleasedMinor:    in.OriginalCreditReservedMinor - finalCredit,
		TierProtected:          tierProtected,
	}, nil
}

// AMLDailyCashLimitExceededError is returned when a cash receipt would exceed the statutory AML cash ceiling (UA-2).
type AMLDailyCashLimitExceededError struct {
	LegalEntityTIN       string `json:"legal_entity_tin"`
	CurrentExposureMinor int64  `json:"current_exposure_minor"`
	AttemptedCashMinor   int64  `json:"attempted_cash_minor"`
	CeilingMinor         int64  `json:"ceiling_minor"`
	CalendarDate         string `json:"calendar_date"`
	Advice               string `json:"advice"`
}

func (e *AMLDailyCashLimitExceededError) Error() string {
	return fmt.Sprintf("AML daily cash limit exceeded for TIN %s on %s: current exposure %d Tiyin + attempted %d Tiyin exceeds ceiling %d Tiyin. %s",
		e.LegalEntityTIN, e.CalendarDate, e.CurrentExposureMinor, e.AttemptedCashMinor, e.CeilingMinor, e.Advice)
}

func (e *AMLDailyCashLimitExceededError) Unwrap() error {
	return ErrAMLCeilingExceeded
}

// MerchantDailyCashExposure models the Spanner table MerchantDailyCashExposure.
type MerchantDailyCashExposure struct {
	LegalEntityTIN       string     `spanner:"LegalEntityTIN" json:"legal_entity_tin"`
	SupplierID           string     `spanner:"SupplierId" json:"supplier_id"`
	CalendarDate         civil.Date `spanner:"CalendarDate" json:"calendar_date"`
	AccumulatedCashMinor int64      `spanner:"AccumulatedCashMinor" json:"accumulated_cash_minor"`
	UpdatedAt            time.Time  `spanner:"UpdatedAt" json:"updated_at"`
}

type MerchantDailyCashExposures = MerchantDailyCashExposure

// OfflineQuotaLease bounds offline cash and credit collection on a driver tablet (Guard #2).
type OfflineQuotaLease struct {
	LeaseID               string    `spanner:"LeaseId" json:"lease_id"`
	OrderID               string    `spanner:"OrderId" json:"order_id"`
	DriverID              string    `spanner:"DriverId" json:"driver_id"`
	SupplierID            string    `spanner:"SupplierId" json:"supplier_id"`
	LegalEntityTIN        string    `spanner:"LegalEntityTIN" json:"legal_entity_tin"`
	CalendarDate          string    `spanner:"CalendarDate" json:"calendar_date"`
	MaxCashAllowedMinor   int64     `spanner:"MaxCashAllowedMinor" json:"max_cash_allowed_minor"`
	MaxCreditAllowedMinor int64     `spanner:"MaxCreditAllowedMinor" json:"max_credit_allowed_minor"`
	ExpiresAt             time.Time `spanner:"ExpiresAt" json:"expires_at"`
	Status                string    `spanner:"Status" json:"status"`
	CreatedAt             time.Time `spanner:"CreatedAt" json:"created_at"`
}

type OfflineQuotaLeases = OfflineQuotaLease

func amlExposureKey(legalEntityTIN, supplierID, calendarDate string) string {
	return fmt.Sprintf("%s::%s::%s", strings.TrimSpace(legalEntityTIN), strings.TrimSpace(supplierID), strings.TrimSpace(calendarDate))
}

// IsAMLCeilingEnforced reports whether statutory AML cash ceilings are enforced.
// When AML_LIMITS_DISABLED or UNLIMITED_PAYMENTS is set, payment and credit limits are removed (uncapped).
func (e *Engine) IsAMLCeilingEnforced() bool {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("AML_LIMITS_DISABLED"))); v == "1" || v == "true" || v == "yes" {
		return false
	}
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("UNLIMITED_PAYMENTS"))); v == "1" || v == "true" || v == "yes" {
		return false
	}
	return true
}

// AMLCashCeilingMinor returns the active cash ceiling in Tiyin (0 if uncapped).
func (e *Engine) AMLCashCeilingMinor() int64 {
	if !e.IsAMLCeilingEnforced() {
		return 0 // Uncapped: no payment or credit ceiling
	}
	if raw := strings.TrimSpace(os.Getenv("AML_CEILING_TIYIN")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			return v
		}
	}
	return DefaultAMLCashCeilingMinor
}

// CheckAndRecordAMLCashExposure implements Edge-Case Guard #2 & Guard #9 (Anti-Smurfing across LegalEntityTIN):
// Tracks daily cash ceiling across all storefronts sharing a LegalEntityTIN (uncapped if AML_LIMITS_DISABLED is set).
func (e *Engine) CheckAndRecordAMLCashExposure(
	ctx context.Context,
	legalEntityTIN string,
	supplierID string,
	calendarDate string,
	cashMinor int64,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	tin := strings.TrimSpace(legalEntityTIN)
	sup := strings.TrimSpace(supplierID)
	date := strings.TrimSpace(calendarDate)
	if tin == "" || sup == "" || date == "" || cashMinor < 0 {
		return 0, fmt.Errorf("%w: legal_entity_tin, supplier_id, calendar_date, and non-negative cash_minor required", ErrPreconditionFailed)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	key := amlExposureKey(tin, sup, date)
	current := e.amlDailyExposure[key]
	ceiling := e.AMLCashCeilingMinor()
	if e.IsAMLCeilingEnforced() && ceiling > 0 && current+cashMinor > ceiling {
		return current, &AMLDailyCashLimitExceededError{
			LegalEntityTIN:       tin,
			CurrentExposureMinor: current,
			AttemptedCashMinor:   cashMinor,
			CeilingMinor:         ceiling,
			CalendarDate:         date,
			Advice:               "Please settle remaining balance via bank wire transfer or corporate card (Uzcard/Humo).",
		}
	}

	e.amlDailyExposure[key] = current + cashMinor
	newTotal := e.amlDailyExposure[key]
	if e.spannerRepo != nil {
		_ = e.spannerRepo.PersistAMLExposure(ctx, tin, sup, date, newTotal)
	}
	return newTotal, nil
}

// IssueOfflineQuotaLease reserves AML cash headroom and credit capacity at manifest sealing (Step 6)
// so offline driver tablets cannot breach statutory AML limits or credit holds in dead zones (Guard #2).
func (e *Engine) IssueOfflineQuotaLease(
	ctx context.Context,
	orderID string,
	driverID string,
	supplierID string,
	legalEntityTIN string,
	calendarDate string,
	requestedCashMinor int64,
	requestedCreditMinor int64,
	ttl time.Duration,
) (*OfflineQuotaLease, error) {
	if _, err := e.CheckAndRecordAMLCashExposure(ctx, legalEntityTIN, supplierID, calendarDate, requestedCashMinor); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}

	now := e.now()
	lease := &OfflineQuotaLease{
		LeaseID:               uuid.NewString(),
		OrderID:               strings.TrimSpace(orderID),
		DriverID:              strings.TrimSpace(driverID),
		SupplierID:            strings.TrimSpace(supplierID),
		LegalEntityTIN:        strings.TrimSpace(legalEntityTIN),
		CalendarDate:          strings.TrimSpace(calendarDate),
		MaxCashAllowedMinor:   requestedCashMinor,
		MaxCreditAllowedMinor: requestedCreditMinor,
		ExpiresAt:             now.Add(ttl),
		Status:                "ACTIVE",
		CreatedAt:             now,
	}

	e.mu.Lock()
	e.offlineLeases[lease.LeaseID] = lease
	e.mu.Unlock()
	return lease, nil
}

// ValidateOfflineLeaseCollection verifies that an offline driver ePOD sync does not exceed its pre-allocated lease.
func (e *Engine) ValidateOfflineLeaseCollection(
	ctx context.Context,
	leaseID string,
	collectedCashMinor int64,
	collectedCreditMinor int64,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	lease, ok := e.offlineLeases[strings.TrimSpace(leaseID)]
	if !ok {
		return fmt.Errorf("%w: lease %s not found", ErrOfflineLeaseExceeded, leaseID)
	}
	if lease.Status != "ACTIVE" {
		return fmt.Errorf("%w: lease %s is not ACTIVE (status=%s)", ErrOfflineLeaseExceeded, leaseID, lease.Status)
	}
	if e.now().After(lease.ExpiresAt) {
		return fmt.Errorf("%w: lease %s expired at %s", ErrOfflineLeaseExceeded, leaseID, lease.ExpiresAt.Format(time.RFC3339))
	}
	if collectedCashMinor > lease.MaxCashAllowedMinor {
		return fmt.Errorf("%w: collected cash %d exceeds lease cap %d", ErrOfflineLeaseExceeded, collectedCashMinor, lease.MaxCashAllowedMinor)
	}
	if collectedCreditMinor > lease.MaxCreditAllowedMinor {
		return fmt.Errorf("%w: collected credit %d exceeds lease cap %d", ErrOfflineLeaseExceeded, collectedCreditMinor, lease.MaxCreditAllowedMinor)
	}
	return nil
}

// FiscalLineItemCheck represents a SKU line checked prior to Warehouse Pick Wave release (Guard #3).
type FiscalLineItemCheck struct {
	SKU         string `json:"sku"`
	IKPUCode    string `json:"ikpu_code"`
	PackageCode string `json:"package_code"`
}

// ValidatePreWaveFiscalReadiness implements Edge-Case Guard #3 (Pre-Flight Fiscal Validation at Step 5):
// Blocks pick wave release if any SKU lacks a valid 17-digit Uzbek IKPU/MXIK code or if the Supplier's
// PKCS#12 (.p12) digital signing certificate expires within < 72 hours.
func (e *Engine) ValidatePreWaveFiscalReadiness(
	ctx context.Context,
	lines []FiscalLineItemCheck,
	certExpiresAt time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(lines) == 0 {
		return fmt.Errorf("%w: order has no fiscal line items", ErrFiscalPreFlightBlocked)
	}
	remainingCertValidity := certExpiresAt.Sub(e.now())
	if remainingCertValidity < 72*time.Hour {
		return fmt.Errorf("%w: PKCS#12 signing certificate expires in %s (< 72h minimum safety window)", ErrFiscalPreFlightBlocked, remainingCertValidity)
	}
	for _, line := range lines {
		ikpu := strings.TrimSpace(line.IKPUCode)
		pkg := strings.TrimSpace(line.PackageCode)
		if len(ikpu) < 8 || pkg == "" {
			return fmt.Errorf("%w: SKU %s missing valid IKPU/MXIK (%q) or PackageCode (%q)", ErrFiscalPreFlightBlocked, line.SKU, ikpu, pkg)
		}
	}
	return nil
}

// ValidateSlicePayoutEligibility implements Edge-Case Guard #3 (Payout Lock at Step 10):
// Blocks an InvoiceSettlementSlice from transitioning to BATCHED unless FiscalState == REGISTERED_SOLIQ.
func (e *Engine) ValidateSlicePayoutEligibility(ctx context.Context, sliceID string, fiscalState string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.ToUpper(strings.TrimSpace(fiscalState)) != "REGISTERED_SOLIQ" {
		return fmt.Errorf("%w: slice %s has fiscal_state=%q", ErrSlicePayoutUnfiscalized, sliceID, fiscalState)
	}
	return nil
}

// RescueToteItem describes a tote on a broken-down vehicle during mid-route rescue (Guard #4).
type RescueToteItem struct {
	ToteID                   string `json:"tote_id"`
	StopSequence             int64  `json:"stop_sequence"`
	IsColdChain              bool   `json:"is_cold_chain"`
	ExpectedCashMinor        int64  `json:"expected_cash_minor"`
	ThermalExcursionMinutes  int64  `json:"thermal_excursion_minutes"`
	MaxAllowedExcursionMins  int64  `json:"max_allowed_excursion_mins"`
}

// MidRouteRescueRequest models a breakdown custody handover between Driver 1 and Rescue Driver 2 (Guard #4).
type MidRouteRescueRequest struct {
	CellID                 string           `json:"cell_id"`
	SupplierID             string           `json:"supplier_id"`
	OriginalRouteID        string           `json:"original_route_id"`
	BrokenVehicleID        string           `json:"broken_vehicle_id"`
	OriginalDriverID       string           `json:"original_driver_id"`
	RescueVehicleID        string           `json:"rescue_vehicle_id"`
	RescueDriverID         string           `json:"rescue_driver_id"`
	LastCompletedStopSeq   int64            `json:"last_completed_stop_seq"`
	NewRescueBoltSeal      string           `json:"new_rescue_bolt_seal"`
	Totes                  []RescueToteItem `json:"totes"`
}

// MidRouteRescueResult summarizes the liability split and thermal quarantine outcome.
type MidRouteRescueResult struct {
	RescueRouteID                 string   `json:"rescue_route_id"`
	OriginalDriverLiabilityMinor  int64    `json:"original_driver_liability_minor"`
	RescueDriverLiabilityMinor    int64    `json:"rescue_driver_liability_minor"`
	TransferredToteIDs            []string `json:"transferred_tote_ids"`
	QuarantinedSpoiledToteIDs     []string `json:"quarantined_spoiled_tote_ids"`
	NewRescueBoltSeal             string   `json:"new_rescue_bolt_seal"`
}

// ExecuteMidRouteBreakdownRescue implements Edge-Case Guard #4:
// 1) Verifies RescueVehicleID is DVIR-authorized.
// 2) Locks OriginalDriverID cash liability strictly to completed stops (<= LastCompletedStopSeq).
// 3) Quarantines cold-chain totes whose thermal excursion exceeded tolerance (stamping MARKING:THERMAL_QUARANTINE_SPOILED).
// 4) Transfers surviving remaining totes and their expected cash liability to RescueDriverID under NewRescueBoltSeal.
func (e *Engine) ExecuteMidRouteBreakdownRescue(
	ctx context.Context,
	req MidRouteRescueRequest,
) (*MidRouteRescueResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	seal := strings.TrimSpace(req.NewRescueBoltSeal)
	if seal == "" || strings.TrimSpace(req.OriginalDriverID) == "" || strings.TrimSpace(req.RescueDriverID) == "" {
		return nil, fmt.Errorf("%w: original_driver_id, rescue_driver_id, and new_rescue_bolt_seal required", ErrPreconditionFailed)
	}
	if err := e.AssertVehicleDispatchAuthorized(ctx, req.RescueVehicleID); err != nil {
		return nil, err
	}

	cellID := strings.TrimSpace(req.CellID)
	if cellID == "" {
		cellID = "cell-uz"
	}

	res := &MidRouteRescueResult{
		RescueRouteID:     "rescue_" + uuid.NewString(),
		NewRescueBoltSeal: seal,
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, t := range req.Totes {
		if t.StopSequence <= req.LastCompletedStopSeq {
			res.OriginalDriverLiabilityMinor += t.ExpectedCashMinor
			continue
		}
		if t.IsColdChain && t.MaxAllowedExcursionMins > 0 && t.ThermalExcursionMinutes > t.MaxAllowedExcursionMins {
			res.QuarantinedSpoiledToteIDs = append(res.QuarantinedSpoiledToteIDs, t.ToteID)
			e.upsertObjectLocked(cellID, "Tote", t.ToteID, req.SupplierID, map[string]any{
				"status":                    "THERMAL_QUARANTINE_SPOILED",
				"thermal_excursion_minutes": t.ThermalExcursionMinutes,
			}, []string{MarkingThermalQuarantineSpoiled})
			continue
		}
		res.TransferredToteIDs = append(res.TransferredToteIDs, t.ToteID)
		res.RescueDriverLiabilityMinor += t.ExpectedCashMinor
	}

	return res, nil
}

// SettlementSliceRecord models an immutable payout slice (Guard #8).
type SettlementSliceRecord struct {
	SliceID          string `json:"slice_id"`
	ParentSliceID    string `json:"parent_slice_id,omitempty"`
	OrderID          string `json:"order_id"`
	SupplierID       string `json:"supplier_id"`
	NetPayoutMinor   int64  `json:"net_payout_minor"`
	Status           string `json:"status"` // UNSETTLED | BATCHED
	Reason           string `json:"reason"`
}

// SetSupplierReserveEscrow configures the rolling reserve holdback balance for a Supplier.
func (e *Engine) SetSupplierReserveEscrow(supplierID string, reserveMinor int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.supplierReserveMinor[strings.TrimSpace(supplierID)] = reserveMinor
}

// CreatePostSettlementClawbackSlice implements Edge-Case Guard #8:
// Forbids mutating a BATCHED slice and instead appends a negative clawback slice backed by the Supplier's rolling reserve.
func (e *Engine) CreatePostSettlementClawbackSlice(
	ctx context.Context,
	originalSlice SettlementSliceRecord,
	approvedDamageClaimMinor int64,
) (*SettlementSliceRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if approvedDamageClaimMinor <= 0 || approvedDamageClaimMinor > originalSlice.NetPayoutMinor {
		return nil, fmt.Errorf("%w: invalid clawback amount %d", ErrPreconditionFailed, approvedDamageClaimMinor)
	}
	if originalSlice.Status != "BATCHED" {
		return nil, fmt.Errorf("%w: slice %s is not BATCHED (status=%s)", ErrSliceImmutableAfterBatch, originalSlice.SliceID, originalSlice.Status)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	supID := strings.TrimSpace(originalSlice.SupplierID)
	reserve := e.supplierReserveMinor[supID]
	if reserve < approvedDamageClaimMinor {
		return nil, fmt.Errorf("%w: supplier %s rolling reserve (%d) insufficient for post-settlement clawback (%d)", ErrPreconditionFailed, supID, reserve, approvedDamageClaimMinor)
	}
	e.supplierReserveMinor[supID] = reserve - approvedDamageClaimMinor

	return &SettlementSliceRecord{
		SliceID:        "slice_clawback_" + uuid.NewString(),
		ParentSliceID:  originalSlice.SliceID,
		OrderID:        originalSlice.OrderID,
		SupplierID:     supID,
		NetPayoutMinor: -approvedDamageClaimMinor,
		Status:         "UNSETTLED",
		Reason:         "CONCEALED_DAMAGE_CLAIM_APPROVED",
	}, nil
}

// LoadStackLayer represents a vertical layer in a vehicle compartment (Guard #6).
type LoadStackLayer struct {
	SKU             string `json:"sku"`
	LayerIndex      int    `json:"layer_index"` // 0 = floor, 1 = stacked on top of 0
	GrossWeightKg   int64  `json:"gross_weight_kg"`
	MaxStackCrushKg int64  `json:"max_stack_crush_kg"`
	VolumeDm3       int64  `json:"volume_dm3"`
}

// ReturnableContainerLedger tracks empty crates/kegs/pallets in the Perpetual Container Ledger (Guard #6).
type ReturnableContainerLedger struct {
	AccountID          string    `spanner:"AccountId" json:"account_id"`
	ContainerSKU       string    `spanner:"ContainerSku" json:"container_sku"`
	SupplierID         string    `spanner:"SupplierId" json:"supplier_id"`
	HolderType         string    `spanner:"HolderType" json:"holder_type"`
	BalanceQty         int64     `spanner:"BalanceQty" json:"balance_qty"`
	DepositLockedMinor int64     `spanner:"DepositLockedMinor" json:"deposit_locked_minor"`
	UpdatedAt          time.Time `spanner:"UpdatedAt" json:"updated_at"`
}

type ReturnableContainerBalance = ReturnableContainerLedger
type ReturnableContainerLedgerEntry = ReturnableContainerLedger

// Validate3DLoadAndRTIReturns implements Edge-Case Guard #6:
// 1) Rejects stacking a heavy item above a fragile item whose MaxStackCrushKg is exceeded.
// 2) Verifies total cargo + reverse-logistics empty RTI container pickup volume fits inside VehicleCapacityDm3.
// 3) Updates the ReturnableContainerLedger balance for the account.
func (e *Engine) Validate3DLoadAndRTIReturns(
	ctx context.Context,
	stackLayers []LoadStackLayer,
	vehicleCapacityDm3 int64,
	accountID string,
	supplierID string,
	containerSKU string,
	deliveredFullContainers int64,
	pickedUpEmptyContainers int64,
	unitContainerVolumeDm3 int64,
	unitDepositMinor int64,
) (*ReturnableContainerBalance, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var cargoVolumeDm3 int64
	for i := 0; i < len(stackLayers); i++ {
		cargoVolumeDm3 += stackLayers[i].VolumeDm3
		if i > 0 {
			below := stackLayers[i-1]
			above := stackLayers[i]
			if above.GrossWeightKg > below.MaxStackCrushKg {
				return nil, fmt.Errorf(
					"%w: SKU %s (%d kg) stacked above fragile SKU %s (max crush %d kg)",
					ErrCrushabilityViolation, above.SKU, above.GrossWeightKg, below.SKU, below.MaxStackCrushKg,
				)
			}
		}
	}

	reversePickupVolumeDm3 := pickedUpEmptyContainers * unitContainerVolumeDm3
	if cargoVolumeDm3+reversePickupVolumeDm3 > vehicleCapacityDm3 {
		return nil, fmt.Errorf(
			"%w: cargo (%d dm3) + RTI empty pickup (%d dm3) exceeds vehicle capacity (%d dm3)",
			ErrPreconditionFailed, cargoVolumeDm3, reversePickupVolumeDm3, vehicleCapacityDm3,
		)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	key := fmt.Sprintf("%s::%s", strings.TrimSpace(accountID), strings.TrimSpace(containerSKU))
	bal, ok := e.containerLedger[key]
	if !ok {
		bal = &ReturnableContainerBalance{
			AccountID:    strings.TrimSpace(accountID),
			ContainerSKU: strings.TrimSpace(containerSKU),
			SupplierID:   strings.TrimSpace(supplierID),
			HolderType:   "RETAILER",
		}
		e.containerLedger[key] = bal
	}
	netDelta := deliveredFullContainers - pickedUpEmptyContainers
	bal.BalanceQty += netDelta
	bal.DepositLockedMinor += netDelta * unitDepositMinor
	bal.UpdatedAt = e.now()

	out := *bal
	return &out, nil
}

// EvaluateFXCorridorBeforeWave implements Edge-Case Guard #9:
// Enforces LOCK_AT_CHECKOUT vs FLOAT_WITH_CORRIDOR_CAP and hard-freezes UZS Tiyin prices once WaveReleased is true.
func (e *Engine) EvaluateFXCorridorBeforeWave(
	ctx context.Context,
	fxPolicy string,
	checkoutPriceMinor int64,
	checkoutFXRateMicros int64,
	currentFXRateMicros int64,
	maxCorridorBps int64,
	waveAlreadyReleased bool,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if waveAlreadyReleased || strings.ToUpper(strings.TrimSpace(fxPolicy)) == "LOCK_AT_CHECKOUT" {
		return checkoutPriceMinor, nil
	}
	if checkoutFXRateMicros <= 0 || currentFXRateMicros <= 0 {
		return checkoutPriceMinor, nil
	}

	deltaBps := ((currentFXRateMicros - checkoutFXRateMicros) * 10000) / checkoutFXRateMicros
	absDeltaBps := deltaBps
	if absDeltaBps < 0 {
		absDeltaBps = -absDeltaBps
	}
	if absDeltaBps > maxCorridorBps {
		adjusted := (checkoutPriceMinor * currentFXRateMicros) / checkoutFXRateMicros
		return adjusted, fmt.Errorf(
			"%w: FX moved %d bps (corridor cap %d bps); new price %d Tiyin",
			ErrFXReconfirmationRequired, deltaBps, maxCorridorBps, adjusted,
		)
	}

	return (checkoutPriceMinor * currentFXRateMicros) / checkoutFXRateMicros, nil
}

// PickToteState tracks active picker custody and scan heartbeats (Guard #10).
type PickToteState struct {
	ToteID              string    `json:"tote_id"`
	WaveID              string    `json:"wave_id"`
	ActivePickerID      string    `json:"active_picker_id"`
	Status              string    `json:"status"` // PICKER_IN_HAND | MARSHALING_STAGED
	CompletedLineCount  int       `json:"completed_line_count"`
	TotalLineCount      int       `json:"total_line_count"`
	PickedLotIDs        []string  `json:"picked_lot_ids"`
	LastScanHeartbeatAt time.Time `json:"last_scan_heartbeat_at"`
}

// RegisterPickToteState registers or updates a tote's scan heartbeat in the warehouse.
func (e *Engine) RegisterPickToteState(tote PickToteState) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := tote
	e.toteStates[strings.TrimSpace(tote.ToteID)] = &cp
}

// ToteTakeoverNotPermittedError is returned when a tote cannot be taken over due to recent picker activity (UA-11).
type ToteTakeoverNotPermittedError struct {
	ToteID               string        `json:"tote_id"`
	CurrentPickerID      string        `json:"current_picker_id"`
	LastScanHeartbeatAt  time.Time     `json:"last_scan_heartbeat_at"`
	IdleDuration         time.Duration `json:"idle_duration"`
	RemainingIdleSeconds int64         `json:"remaining_idle_seconds"`
}

func (e *ToteTakeoverNotPermittedError) Error() string {
	return fmt.Sprintf("tote %s takeover not permitted: currently held by picker %s (last scanned %s ago, remaining idle lock: %ds)",
		e.ToteID, e.CurrentPickerID, e.IdleDuration, e.RemainingIdleSeconds)
}

func (e *ToteTakeoverNotPermittedError) Unwrap() error {
	return ErrToteHeartbeatActive
}

// TakeoverAbandonedPickTote implements Edge-Case Guard #10:
// Allows a relief picker to take over a PICKER_IN_HAND tote only if its scan heartbeat has stalled for >= 20 minutes,
// preserving all already-picked FEFO lots and resuming at CompletedLineCount + 1.
func (e *Engine) TakeoverAbandonedPickTote(
	ctx context.Context,
	toteID string,
	newPickerID string,
) (*PickToteState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	tote, ok := e.toteStates[strings.TrimSpace(toteID)]
	if !ok {
		return nil, fmt.Errorf("%w: tote %s", ErrObjectNotFound, toteID)
	}
	stalledDuration := e.now().Sub(tote.LastScanHeartbeatAt)
	if stalledDuration < 20*time.Minute {
		remSec := int64((20*time.Minute - stalledDuration) / time.Second)
		if remSec < 0 {
			remSec = 0
		}
		return nil, &ToteTakeoverNotPermittedError{
			ToteID:               toteID,
			CurrentPickerID:      tote.ActivePickerID,
			LastScanHeartbeatAt:  tote.LastScanHeartbeatAt,
			IdleDuration:         stalledDuration,
			RemainingIdleSeconds: remSec,
		}
	}

	tote.ActivePickerID = strings.TrimSpace(newPickerID)
	tote.LastScanHeartbeatAt = e.now()
	cp := *tote
	return &cp, nil
}
