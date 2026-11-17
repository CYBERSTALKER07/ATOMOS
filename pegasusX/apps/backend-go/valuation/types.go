package valuation

import (
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/civil"
)

// Canonical movement types persisted in InventoryMovements.
const (
	MovementTypeReceipt          = "RECEIPT"
	MovementTypeIssueCOGS        = "ISSUE_COGS"
	MovementTypeAdjustmentGain   = "ADJUSTMENT_GAIN"
	MovementTypeAdjustmentShrink = "ADJUSTMENT_SHRINK"
	MovementTypeReturn           = "RETURN"
)

// CostingMethod selects cost-layer depletion and valuation calculation behavior.
const (
	CostingMethodFEFO          = "FEFO"
	CostingMethodMovingAverage = "MOVING_AVERAGE"
)

// DefaultExpiryRiskWindowDays is the default near-expiry threshold (30 days) for inventory.expiry.risk events.
const DefaultExpiryRiskWindowDays int64 = 30

// Domain sentinel errors.
var (
	ErrInvalidInput           = errors.New("valuation: invalid input")
	ErrInsufficientCostLayers = errors.New("valuation: insufficient active cost layers")
	ErrRepoUnavailable        = errors.New("valuation: repository unavailable")
	ErrUnknownCreditAccount   = errors.New("valuation: unsupported receipt credit account (must be AP:VENDOR or GRNI)")
)

// InventoryCostLayer models one persisted InventoryCostLayers row plus computed valuation metadata.
type InventoryCostLayer struct {
	EntityID        string    `json:"entity_id"`
	WarehouseID     string    `json:"warehouse_id"`
	ProductID       string    `json:"product_id"`
	LayerID         string    `json:"layer_id"`
	LotID           string    `json:"lot_id,omitempty"`
	ReceiptRefType  string    `json:"receipt_ref_type,omitempty"`
	ReceiptRefID    string    `json:"receipt_ref_id,omitempty"`
	QtyRemaining    int64     `json:"qty_remaining"`
	OriginalQty     int64     `json:"original_qty"`
	UnitCostMinor   int64     `json:"unit_cost_minor"`
	TotalValueMinor int64     `json:"total_value_minor"`
	Currency        string    `json:"currency"`
	FxRateID        string    `json:"fx_rate_id,omitempty"`
	ReceivedAt      time.Time `json:"received_at"`
	ExpiryDate      string    `json:"expiry_date,omitempty"`
	NearExpiry      bool      `json:"near_expiry,omitempty"`
	DaysToExpiry    *int64    `json:"days_to_expiry,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// InventoryMovement models one append-only InventoryMovements audit row.
type InventoryMovement struct {
	EntityID       string    `json:"entity_id"`
	MovementID     string    `json:"movement_id"`
	MovementType   string    `json:"movement_type"`
	ReferenceType  string    `json:"reference_type"`
	ReferenceID    string    `json:"reference_id"`
	WarehouseID    string    `json:"warehouse_id"`
	ProductID      string    `json:"product_id"`
	LotID          string    `json:"lot_id,omitempty"`
	LayerID        string    `json:"layer_id,omitempty"`
	QtyDelta       int64     `json:"qty_delta"`
	UnitCostMinor  int64     `json:"unit_cost_minor"`
	TotalCostMinor int64     `json:"total_cost_minor"`
	Currency       string    `json:"currency"`
	GlEntryID      string    `json:"gl_entry_id,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// SKUValuationSummary aggregates active layers for one (EntityId, WarehouseId, ProductId).
type SKUValuationSummary struct {
	EntityID               string `json:"entity_id"`
	WarehouseID            string `json:"warehouse_id"`
	ProductID              string `json:"product_id"`
	Currency               string `json:"currency"`
	TotalQtyRemaining      int64  `json:"total_qty_remaining"`
	TotalValueMinor        int64  `json:"total_value_minor"`
	MovingAvgUnitCostMinor int64  `json:"moving_avg_unit_cost_minor"`
	LayerCount             int64  `json:"layer_count"`
}

// ValuationSummaryResponse is the response payload for GET /v1/inventory/valuation.
type ValuationSummaryResponse struct {
	EntityID          string                `json:"entity_id"`
	WarehouseID       string                `json:"warehouse_id,omitempty"`
	Currency          string                `json:"currency"`
	TotalQtyRemaining int64                 `json:"total_qty_remaining"`
	TotalValueMinor   int64                 `json:"total_value_minor"`
	ExpiryRiskDays    int64                 `json:"expiry_risk_days"`
	SKUs              []SKUValuationSummary `json:"skus"`
	Layers            []InventoryCostLayer  `json:"layers"`
}

// ValuationFilter narrows GET /v1/inventory/valuation queries.
type ValuationFilter struct {
	EntityID       string
	WarehouseID    string
	ProductID      string
	IncludeZeroQty bool
	ExpiryRiskDays int64
}

// MovementsFilter narrows GET /v1/inventory/movements queries.
type MovementsFilter struct {
	EntityID     string
	WarehouseID  string
	ProductID    string
	MovementType string
	Limit        int
}

// ReceiptInput records a valued inventory receipt creating a cost layer, movement, GL entry, and events.
type ReceiptInput struct {
	EntityID             string    `json:"entity_id,omitempty"`
	WarehouseID          string    `json:"warehouse_id"`
	ProductID            string    `json:"product_id"`
	LotID                string    `json:"lot_id,omitempty"`
	ReceiptRefType       string    `json:"receipt_ref_type,omitempty"`
	ReceiptRefID         string    `json:"receipt_ref_id"`
	Quantity             int64     `json:"quantity"`
	UnitCostMinor        int64     `json:"unit_cost_minor"`
	VatMinor             int64     `json:"vat_minor,omitempty"`
	CreditAccount        string    `json:"credit_account,omitempty"` // AP:VENDOR (default) or GRNI
	VendorID             string    `json:"vendor_id,omitempty"`
	Currency             string    `json:"currency,omitempty"`
	CountryPack          string    `json:"country_pack,omitempty"`
	FxRateID             string    `json:"fx_rate_id,omitempty"`
	ReceivedAt           time.Time `json:"received_at,omitempty"`
	ExpiryDate           string    `json:"expiry_date,omitempty"`
	ExpiryRiskWindowDays int64     `json:"expiry_risk_window_days,omitempty"`
	IdempotencyKey       string    `json:"idempotency_key"`
	ActorID              string    `json:"-"`
}

// ReceiptResult summarizes a persisted or replayed receipt.
type ReceiptResult struct {
	Layer             InventoryCostLayer `json:"layer"`
	Movement          InventoryMovement  `json:"movement"`
	GlEntryID         string             `json:"gl_entry_id,omitempty"`
	TotalCostMinor    int64              `json:"total_cost_minor"`
	VatMinor          int64              `json:"vat_minor,omitempty"`
	ExpiryRiskEmitted bool               `json:"expiry_risk_emitted,omitempty"`
	AlreadyProcessed  bool               `json:"already_processed,omitempty"`
}

// IssueInput consumes inventory layers via FEFO (or moving-average) and posts COGS vs INVENTORY.
type IssueInput struct {
	EntityID       string    `json:"entity_id,omitempty"`
	WarehouseID    string    `json:"warehouse_id"`
	ProductID      string    `json:"product_id"`
	LotID          string    `json:"lot_id,omitempty"`
	ReferenceType  string    `json:"reference_type"`
	ReferenceID    string    `json:"reference_id"`
	OrderID        string    `json:"order_id,omitempty"`
	Quantity       int64     `json:"quantity"`
	CostingMethod  string    `json:"costing_method,omitempty"` // FEFO (default) or MOVING_AVERAGE
	Currency       string    `json:"currency,omitempty"`
	CountryPack    string    `json:"country_pack,omitempty"`
	OccurredAt     time.Time `json:"occurred_at,omitempty"`
	IdempotencyKey string    `json:"idempotency_key"`
	ActorID        string    `json:"-"`
}

// IssueResult summarizes a persisted or replayed stock issue.
type IssueResult struct {
	Movements        []InventoryMovement `json:"movements"`
	TotalQtyIssued   int64               `json:"total_qty_issued"`
	TotalCostMinor   int64               `json:"total_cost_minor"`
	UnitCostAvgMinor int64               `json:"unit_cost_avg_minor"`
	GlEntryID        string              `json:"gl_entry_id,omitempty"`
	AlreadyProcessed bool                `json:"already_processed,omitempty"`
}

// AdjustmentCostInput posts inventory count gain/shrink cost movements and GL EXP:SHRINK vs INVENTORY.
type AdjustmentCostInput struct {
	EntityID       string    `json:"entity_id,omitempty"`
	WarehouseID    string    `json:"warehouse_id"`
	ProductID      string    `json:"product_id"`
	LotID          string    `json:"lot_id,omitempty"`
	AdjustmentID   string    `json:"adjustment_id"`
	DeltaQty       int64     `json:"delta_qty"`
	UnitCostMinor  int64     `json:"unit_cost_minor,omitempty"`
	ReasonCode     string    `json:"reason_code,omitempty"`
	Currency       string    `json:"currency,omitempty"`
	CountryPack    string    `json:"country_pack,omitempty"`
	OccurredAt     time.Time `json:"occurred_at,omitempty"`
	IdempotencyKey string    `json:"idempotency_key"`
	ActorID        string    `json:"-"`
}

// AdjustmentCostResult summarizes a persisted or replayed adjustment valuation.
type AdjustmentCostResult struct {
	Movements        []InventoryMovement `json:"movements"`
	Layer            *InventoryCostLayer `json:"layer,omitempty"`
	TotalCostMinor   int64               `json:"total_cost_minor"`
	GlEntryID        string              `json:"gl_entry_id,omitempty"`
	AlreadyProcessed bool                `json:"already_processed,omitempty"`
}

// ExpiryRiskLayer describes an active cost layer approaching or past expiry.
type ExpiryRiskLayer struct {
	Layer            InventoryCostLayer `json:"layer"`
	ValueAtRiskMinor int64              `json:"value_at_risk_minor"`
	DaysRemaining    int64              `json:"days_remaining"`
}

// ComputeMovingAverageUnitCostMinor computes integer half-up moving-average unit cost in minor units.
// Complexity: O(1), zero floating-point operations.
func ComputeMovingAverageUnitCostMinor(totalValueMinor, totalQty int64) int64 {
	if totalQty <= 0 || totalValueMinor <= 0 {
		return 0
	}
	return (totalValueMinor + totalQty/2) / totalQty
}

// ParseExpiryDate parses YYYY-MM-DD or RFC3339 timestamps into civil.Date.
func ParseExpiryDate(raw string) (*civil.Date, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if d, err := civil.ParseDate(trimmed); err == nil {
		return &d, nil
	}
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		d := civil.DateOf(t.UTC())
		return &d, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		d := civil.DateOf(t.UTC())
		return &d, nil
	}
	return nil, ErrInvalidInput
}

// ComputeExpiryDaysRemaining calculates whole calendar days between ref UTC date and expiry.
func ComputeExpiryDaysRemaining(expiry *civil.Date, ref time.Time) *int64 {
	if expiry == nil {
		return nil
	}
	if ref.IsZero() {
		ref = time.Now().UTC()
	}
	refDate := civil.DateOf(ref.UTC())
	days := int64(expiry.DaysSince(refDate))
	return &days
}
