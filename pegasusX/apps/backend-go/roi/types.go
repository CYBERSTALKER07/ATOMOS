// Package roi implements ATOMOS M1.8 ROI v1 baseline capture, savings
// attribution events across 6 value categories, holdout-vs-treatment tracking,
// platform fee comparison (in basis points), and QBR JSON/CSV reporting.
package roi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Canonical ROI savings categories.
const (
	CategoryBadDebtAvoided      = "BAD_DEBT_AVOIDED"
	CategoryStockoutAvoided     = "STOCKOUT_AVOIDED"
	CategoryExpirySaved         = "EXPIRY_SAVED"
	CategoryLogisticsSaved      = "LOGISTICS_SAVED"
	CategoryLabourSaved         = "LABOUR_SAVED"
	CategoryWorkingCapitalSaved = "WORKING_CAPITAL_SAVED"
)

var validCategories = map[string]bool{
	CategoryBadDebtAvoided:      true,
	CategoryStockoutAvoided:     true,
	CategoryExpirySaved:         true,
	CategoryLogisticsSaved:      true,
	CategoryLabourSaved:         true,
	CategoryWorkingCapitalSaved: true,
}

// AllCategories returns the canonical ordered slice of ROI savings categories.
func AllCategories() []string {
	return []string{
		CategoryBadDebtAvoided,
		CategoryStockoutAvoided,
		CategoryExpirySaved,
		CategoryLogisticsSaved,
		CategoryLabourSaved,
		CategoryWorkingCapitalSaved,
	}
}

// NormalizeCategory uppercases and validates a category name.
func NormalizeCategory(raw string) (string, bool) {
	c := strings.ToUpper(strings.TrimSpace(raw))
	return c, validCategories[c]
}

// DefaultMethodology returns the deterministic attribution methodology string for a category.
func DefaultMethodology(category string) string {
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case CategoryBadDebtAvoided:
		return "Overdue AR recovery and automated credit-hold prevention measured against 90-day baseline bad-debt exposure"
	case CategoryStockoutAvoided:
		return "Protected gross margin on prevented stockouts comparing treatment SKUs vs holdout control SKUs"
	case CategoryExpirySaved:
		return "Near-expiry lot and cost-layer value preserved via FEFO rebalance or markdown before expiration"
	case CategoryLogisticsSaved:
		return "VRP route optimization distance and fleet capacity savings vs pre-platform baseline cost"
	case CategoryLabourSaved:
		return "Estimated operator workflow hours saved multiplied by onboarding survey hourly wage (survey estimate)"
	case CategoryWorkingCapitalSaved:
		return "Working capital freed via DSO and safety-stock reduction valued at prorated annual cost of capital"
	default:
		return "Governed operational value attribution"
	}
}

// DefaultConfidenceBps returns the default attribution confidence in basis points (0..10000).
func DefaultConfidenceBps(category string) int64 {
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case CategoryExpirySaved:
		return 9000
	case CategoryBadDebtAvoided:
		return 8500
	case CategoryLogisticsSaved:
		return 8200
	case CategoryStockoutAvoided:
		return 8000
	case CategoryWorkingCapitalSaved:
		return 7500
	case CategoryLabourSaved:
		return 6500
	default:
		return 7500
	}
}

// Domain sentinel errors.
var (
	ErrInvalidInput    = errors.New("roi: invalid input")
	ErrFloatForbidden  = errors.New("roi: floating-point values are forbidden; use int64 minor units or bps")
	ErrRepoUnavailable = errors.New("roi: repository unavailable")
	ErrForbidden       = errors.New("roi: forbidden")
)

// Baseline represents an entity's 90-day historical + onboarding survey baseline in RoiBaselines.
type Baseline struct {
	TenantID               string         `json:"tenant_id"`
	BaselineID             string         `json:"baseline_id"`
	WindowDays             int64          `json:"window_days"`
	BaselineArMinor        int64          `json:"baseline_ar_minor"`
	BaselineBadDebtMinor   int64          `json:"baseline_bad_debt_minor"`
	BaselineExpiredMinor   int64          `json:"baseline_expired_minor"`
	BaselineStockoutMinor  int64          `json:"baseline_stockout_minor"`
	BaselineLogisticsMinor int64          `json:"baseline_logistics_minor"`
	BaselineLabourHours    int64          `json:"baseline_labour_hours"`
	SurveyHourlyWageMinor  int64          `json:"survey_hourly_wage_minor"`
	SurveyNotes            string         `json:"survey_notes,omitempty"`
	SurveyJSON             string         `json:"survey_json,omitempty"`
	Survey                 map[string]any `json:"survey,omitempty"`
	Currency               string         `json:"currency"`
	CapturedBy             string         `json:"captured_by"`
	PeriodStart            time.Time      `json:"period_start"`
	PeriodEnd              time.Time      `json:"period_end"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

// RoiEvent represents one value attribution record persisted in RoiEvents.
type RoiEvent struct {
	TenantID        string    `json:"tenant_id"`
	EventID         string    `json:"event_id"`
	Category        string    `json:"category"`
	SavingsMinor    int64     `json:"savings_minor"`
	Currency        string    `json:"currency"`
	Methodology     string    `json:"methodology"`
	ConfidenceBps   int64     `json:"confidence_bps"`
	IsEstimate      bool      `json:"is_estimate"`
	HoldoutControl  bool      `json:"holdout_control"`
	ProposalID      string    `json:"proposal_id,omitempty"`
	EvidenceRefType string    `json:"evidence_ref_type,omitempty"`
	EvidenceRefID   string    `json:"evidence_ref_id,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	RecordedBy      string    `json:"recorded_by"`
	IdempotencyKey  string    `json:"idempotency_key,omitempty"`
	OccurredAt      time.Time `json:"occurred_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// CategorySummary aggregates ROI events for one category within a summary window.
type CategorySummary struct {
	Category            string `json:"category"`
	EventCount          int64  `json:"event_count"`
	TotalSavingsMinor   int64  `json:"total_savings_minor"`
	RealizedMinor       int64  `json:"realized_minor"`
	EstimatedMinor      int64  `json:"estimated_minor"`
	HoldoutSavingsMinor int64  `json:"holdout_savings_minor"`
	AvgConfidenceBps    int64  `json:"avg_confidence_bps"`
	Methodology         string `json:"methodology"`
}

// HoldoutStats summarizes treatment vs holdout-control experimentation metrics.
type HoldoutStats struct {
	TreatmentEventCount   int64 `json:"treatment_event_count"`
	TreatmentSavingsMinor int64 `json:"treatment_savings_minor"`
	HoldoutEventCount     int64 `json:"holdout_event_count"`
	HoldoutSavingsMinor   int64 `json:"holdout_savings_minor"`
	IncrementalLiftMinor  int64 `json:"incremental_lift_minor"`
	LiftBps               int64 `json:"lift_bps"`
}

// Summary aggregates realized & estimated ROI savings against BillingMeterEvents platform fees.
type Summary struct {
	TenantID              string            `json:"tenant_id"`
	Currency              string            `json:"currency"`
	WindowDays            int64             `json:"window_days"`
	PeriodStart           time.Time         `json:"period_start"`
	PeriodEnd             time.Time         `json:"period_end"`
	TotalSavingsMinor     int64             `json:"total_savings_minor"`
	RealizedSavingsMinor  int64             `json:"realized_savings_minor"`
	EstimatedSavingsMinor int64             `json:"estimated_savings_minor"`
	HoldoutSavingsMinor   int64             `json:"holdout_savings_minor"`
	PlatformFeesMinor     int64             `json:"platform_fees_minor"`
	NetSavingsMinor       int64             `json:"net_savings_minor"`
	RoiMultipleBps        int64             `json:"roi_multiple_bps"`
	Categories            []CategorySummary `json:"categories"`
	HoldoutStats          HoldoutStats      `json:"holdout_stats"`
	Baseline              *Baseline         `json:"baseline,omitempty"`
}

// QBRReport bundles the ROI summary and supporting event ledger for Quarterly Business Reviews.
type QBRReport struct {
	GeneratedAt time.Time  `json:"generated_at"`
	Summary     Summary    `json:"summary"`
	Events      []RoiEvent `json:"events"`
}

// CaptureBaselineInput configures baseline capture from 90d historical tables plus onboarding survey.
type CaptureBaselineInput struct {
	TenantID               string         `json:"tenant_id,omitempty"`
	WindowDays             int64          `json:"window_days,omitempty"`
	BaselineArMinor        *int64         `json:"baseline_ar_minor,omitempty"`
	BaselineBadDebtMinor   *int64         `json:"baseline_bad_debt_minor,omitempty"`
	BaselineExpiredMinor   *int64         `json:"baseline_expired_minor,omitempty"`
	BaselineStockoutMinor  int64          `json:"baseline_stockout_minor,omitempty"`
	BaselineLogisticsMinor int64          `json:"baseline_logistics_minor,omitempty"`
	BaselineLabourHours    int64          `json:"baseline_labour_hours,omitempty"`
	SurveyHourlyWageMinor  int64          `json:"survey_hourly_wage_minor,omitempty"`
	SurveyNotes            string         `json:"survey_notes,omitempty"`
	Survey                 map[string]any `json:"survey,omitempty"`
	Currency               string         `json:"currency,omitempty"`
	CapturedBy             string         `json:"-"`
}

// RecordEventInput creates a persisted ROI event and emits roi.event.recorded.
type RecordEventInput struct {
	TenantID        string    `json:"tenant_id,omitempty"`
	Category        string    `json:"category"`
	SavingsMinor    int64     `json:"savings_minor"`
	Currency        string    `json:"currency,omitempty"`
	Methodology     string    `json:"methodology,omitempty"`
	ConfidenceBps   int64     `json:"confidence_bps,omitempty"`
	IsEstimate      *bool     `json:"is_estimate,omitempty"`
	HoldoutControl  bool      `json:"holdout_control,omitempty"`
	ProposalID      string    `json:"proposal_id,omitempty"`
	EvidenceRefType string    `json:"evidence_ref_type,omitempty"`
	EvidenceRefID   string    `json:"evidence_ref_id,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	RecordedBy      string    `json:"-"`
	IdempotencyKey  string    `json:"idempotency_key,omitempty"`
	OccurredAt      time.Time `json:"occurred_at,omitempty"`
}

// EventsFilter scopes GET /v1/roi/events queries.
type EventsFilter struct {
	TenantID string
	Category string
	Since    *time.Time
	Until    *time.Time
	Limit    int
}

// HistoricalMetrics holds 90-day baseline metrics read from live Spanner AR/Inventory tables.
type HistoricalMetrics struct {
	ArMinor      int64
	BadDebtMinor int64
	ExpiredMinor int64
	Currency     string
}

func validateNoFloats(v any) error {
	switch tv := v.(type) {
	case nil, bool, string, int, int32, int64, uint, uint32, uint64, []string, []int, []int64, map[string]string:
		return nil
	case float32, float64:
		return ErrFloatForbidden
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return fmt.Errorf("%w: %q", ErrFloatForbidden, s)
		}
		return nil
	case []any:
		for _, elem := range tv {
			if err := validateNoFloats(elem); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		for _, elem := range tv {
			if err := validateNoFloats(elem); err != nil {
				return err
			}
		}
		return nil
	default:
		raw, err := json.Marshal(tv)
		if err != nil {
			return fmt.Errorf("%w: unsupported payload value %T: %v", ErrInvalidInput, tv, err)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return fmt.Errorf("%w: decode value %T: %v", ErrInvalidInput, tv, err)
		}
		return validateNoFloats(parsed)
	}
}
