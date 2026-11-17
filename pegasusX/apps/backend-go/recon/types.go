package recon

import (
	"errors"
	"time"
)

// Dimension identifiers for the four-way financial reconciliation engine.
const (
	DimensionOrdersAR  = "ORDERS_AR"
	DimensionStockGL   = "STOCK_GL"
	DimensionCashPSP   = "CASH_PSP"
	DimensionTaxFiscal = "TAX_FISCAL"
)

// AllDimensions enumerates the canonical order of the four reconciliation dimensions.
var AllDimensions = []string{
	DimensionOrdersAR,
	DimensionStockGL,
	DimensionCashPSP,
	DimensionTaxFiscal,
}

// Run modes, statuses, break statuses, actions, severities, and object types.
const (
	ModeManual    = "MANUAL"
	ModeNightly   = "NIGHTLY"
	ModeTriggered = "TRIGGERED"

	RunStatusGreen       = "GREEN"
	RunStatusBreaksFound = "BREAKS_FOUND"
	RunStatusFailed      = "FAILED"

	DimStatusGreen = "GREEN"
	DimStatusBreak = "BREAK"

	BreakStatusOpen         = "OPEN"
	BreakStatusAcknowledged = "ACKNOWLEDGED"
	BreakStatusResolved     = "RESOLVED"
	BreakStatusWaived       = "WAIVED"

	ActionAcknowledge = "ACKNOWLEDGE"
	ActionResolve     = "RESOLVE"
	ActionWaive       = "WAIVE"

	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"

	ObjectOrder            = "ORDER"
	ObjectDimension        = "DIMENSION"
	ObjectStockLot         = "STOCK_LOT"
	ObjectStockLotUncosted = "STOCK_LOT_UNCOSTED"
	ObjectGLAccount        = "GL_ACCOUNT"
	ObjectCashRecon        = "CASH_RECON"
)

// Default owner roles by dimension.
const (
	OwnerFinanceController = "FINANCE_CONTROLLER"
	OwnerWarehouseManager  = "WAREHOUSE_MANAGER"
	OwnerTreasuryOps       = "TREASURY_OPS"
	OwnerTaxCompliance     = "TAX_COMPLIANCE"
)

// Sentinel errors returned by Service and Repository operations.
var (
	ErrInvalidInput      = errors.New("recon: invalid input")
	ErrNotFound          = errors.New("recon: not found")
	ErrInvalidTransition = errors.New("recon: invalid break transition")
	ErrForbidden         = errors.New("recon: forbidden")
)

// DimensionCheckSummary captures tie-out figures and break rollup for one dimension.
type DimensionCheckSummary struct {
	Dimension     string           `json:"dimension"`
	Status        string           `json:"status"`
	ExpectedMinor int64            `json:"expected_minor"`
	ActualMinor   int64            `json:"actual_minor"`
	DeltaMinor    int64            `json:"delta_minor"`
	BreakCount    int64            `json:"break_count"`
	ExposureMinor int64            `json:"exposure_minor"`
	Currency      string           `json:"currency"`
	Metrics       map[string]int64 `json:"metrics,omitempty"`
}

// RunSummary aggregates pass/fail counts across dimensions and severities.
type RunSummary struct {
	GreenDimensions int64 `json:"green_dimensions"`
	BreakDimensions int64 `json:"break_dimensions"`
	TotalBreaks     int64 `json:"total_breaks"`
	CriticalBreaks  int64 `json:"critical_breaks"`
	HighBreaks      int64 `json:"high_breaks"`
	MediumBreaks    int64 `json:"medium_breaks"`
	LowBreaks       int64 `json:"low_breaks"`
}

// ReconciliationRun is the persisted run envelope returned to API callers and UI.
type ReconciliationRun struct {
	RunID         string                           `json:"run_id"`
	EntityID      string                           `json:"entity_id"`
	PeriodID      string                           `json:"period_id"`
	Mode          string                           `json:"mode"`
	Status        string                           `json:"status"`
	BreakCount    int64                            `json:"break_count"`
	ExposureMinor int64                            `json:"exposure_minor"`
	Currency      string                           `json:"currency"`
	Checks        map[string]DimensionCheckSummary `json:"checks"`
	Summary       RunSummary                       `json:"summary"`
	TriggeredBy   string                           `json:"triggered_by"`
	StartedAt     time.Time                        `json:"started_at"`
	CompletedAt   time.Time                        `json:"completed_at"`
}

// ReconciliationBreak represents a persisted discrepancy item and its Control Tower escalation link.
type ReconciliationBreak struct {
	BreakID                 string     `json:"break_id"`
	RunID                   string     `json:"run_id"`
	EntityID                string     `json:"entity_id"`
	Dimension               string     `json:"dimension"`
	ObjectType              string     `json:"object_type"`
	ObjectID                string     `json:"object_id"`
	ExpectedMinor           int64      `json:"expected_minor"`
	ActualMinor             int64      `json:"actual_minor"`
	DeltaMinor              int64      `json:"delta_minor"`
	Currency                string     `json:"currency"`
	Severity                string     `json:"severity"`
	Status                  string     `json:"status"`
	OwnerID                 string     `json:"owner_id"`
	SlaDueAt                *time.Time `json:"sla_due_at,omitempty"`
	ResolutionNote          string     `json:"resolution_note,omitempty"`
	ControlTowerExceptionID string     `json:"control_tower_exception_id,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// RunRequest parameterizes a four-way reconciliation sweep for one entity.
type RunRequest struct {
	EntityID    string   `json:"entity_id,omitempty"`
	PeriodID    string   `json:"period_id,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Dimensions  []string `json:"dimensions,omitempty"`
	CountryPack string   `json:"-"`
	TriggeredBy string   `json:"-"`
}

// RunOutput bundles the persisted run and any breaks opened during evaluation.
type RunOutput struct {
	Run    ReconciliationRun     `json:"run"`
	Breaks []ReconciliationBreak `json:"breaks"`
}

// ListRunsFilter constrains run history queries.
type ListRunsFilter struct {
	EntityID string
	PeriodID string
	Status   string
	Limit    int
}

// ListBreaksFilter constrains break queue queries.
type ListBreaksFilter struct {
	EntityID  string
	RunID     string
	Dimension string
	Status    string
	Severity  string
	Limit     int
}

// PatchBreakRequest describes an operator action against a reconciliation break.
type PatchBreakRequest struct {
	Action         string `json:"action"`
	Note           string `json:"note,omitempty"`
	ResolutionNote string `json:"resolution_note,omitempty"`
	OwnerID        string `json:"owner_id,omitempty"`
	ActorID        string `json:"-"`
}

// Snapshot carries strongly consistent subledger + GL figures collected at one read timestamp.
type Snapshot struct {
	EntityID string
	PeriodID string
	Currency string
	AsOf     time.Time

	// ORDERS_AR inputs
	DeliveredCreditOrderCount int64
	DeliveredCreditOrderMinor int64
	DeliveredCreditAnomalies  []OrderARAnomaly
	OpenARInvoiceCount        int64
	OpenARBalanceMinor        int64
	GLARRetailerNetMinor      int64 // DebitMinor - CreditMinor on AR:RETAILER

	// STOCK_GL inputs
	WarehousesChecked      int64
	MissingWarehouseLots   []MissingWarehouseLot
	V2Mismatches           []StockV2Mismatch
	UncostedLots           []UncostedLot
	CostedStockValuedMinor int64
	HasCostLayers          bool
	CostLayersCount        int64
	CostLayersQtyRemaining int64
	CostLayersValuedMinor  int64
	GLInventoryNetMinor    int64 // DebitMinor - CreditMinor on INVENTORY

	// CASH_PSP inputs
	SettledPSPSessionsMinor                  int64
	FallbackCapturedCardLegsMinor            int64
	PaymentChargebacksMinor                  int64
	DriverCashCollectedMinor                 int64
	CashReconAcceptedShortfallRepaidMinor    int64
	CashReconPendingOrDisputedShortfallMinor int64
	CashReconWrittenOffShortfallMinor        int64
	CashReconOverageMinor                    int64
	DisputedCashRecons                       []DisputedCashRecon
	VerifiedARCardGLMinor                    int64
	VerifiedARCashOfficeGLMinor              int64
	VerifiedARCashDriverGLMinor              int64
	VerifiedARBankGLMinor                    int64
	VerifiedCreditTopupBankMinor             int64
	BankPaidPayoutsMinor                     int64
	GLPSPGatewayNetMinor                     int64
	GLCashDriverNetMinor                     int64
	GLCashOfficeNetMinor                     int64
	GLBankOperatingNetMinor                  int64

	// TAX_FISCAL inputs
	FiscalizedOrderCount   int64
	SnapshotVatMinor       int64
	CreditNoteVatMinor     int64
	MissingSnapshotOrders  []MissingFiscalSnapshotOrder
	GLVatOutputCreditMinor int64 // CreditMinor - DebitMinor on TAX:VAT:OUTPUT
}

// OrderARAnomaly flags a delivered credit order whose AR invoice is missing or mismatched.
type OrderARAnomaly struct {
	OrderID               string
	OrderTotalMinor       int64
	InvoicePrincipalMinor int64
	InvoiceStatus         string
}

// MissingWarehouseLot identifies an active lot whose WarehouseId is blank.
type MissingWarehouseLot struct {
	LotID         string
	ProductID     string
	QuantityTotal int64
	UnitCostMinor int64
}

// StockV2Mismatch captures a SKU-level delta from stocklots.ReconcileInventoryV2 plus resolved unit cost.
type StockV2Mismatch struct {
	WarehouseID     string
	ProductID       string
	MissingV2       bool
	LotAvailable    int64
	LotReserved     int64
	LotQuarantine   int64
	V2Available     int64
	V2Reserved      int64
	V2Quarantine    int64
	DeltaAvailable  int64
	DeltaReserved   int64
	DeltaQuarantine int64
	UnitCostMinor   int64
}

// UncostedLot flags an active on-hand StockLot without lot cost or catalog product price.
type UncostedLot struct {
	LotID       string
	WarehouseID string
	ProductID   string
	Quantity    int64
}

// DisputedCashRecon flags a CashReconciliations row awaiting variance resolution.
type DisputedCashRecon struct {
	ReconciliationID  string
	DriverID          string
	ExpectedCashMinor int64
	DeclaredCashMinor int64
	DifferenceMinor   int64
}

// MissingFiscalSnapshotOrder flags a fiscalized non-zero order without OrderLineFiscalSnapshots.
type MissingFiscalSnapshotOrder struct {
	OrderID    string
	TotalMinor int64
}
