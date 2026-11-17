// Package forecastv2 implements ATOMOS P1 M1.5 hybrid demand forecasting:
// classic per-series champion models (SES, Croston SBA, Holt-Winters via planning/forecast),
// a challenger global seasonal/calendar regressor model (Navruz, Ramadan, Hayits,
// SchoolStart, Payday, price & promo regressors), rolling WAPE backtesting,
// quantile intervals (P10/P50/P90 in milli-units), Gemini & structural classic
// explainability, and Forecast Value Added (FVA) tracking for approved adjustments.
package forecastv2

import (
	"errors"
	"time"

	"cloud.google.com/go/civil"
)

// Model identifiers persisted on ForecastRuns and ForecastValues.
const (
	ModelClassicSES         = "classic_ses"
	ModelClassicCroston     = "classic_croston"
	ModelClassicHoltWinters = "classic_holt_winters"
	ModelGlobalSeasonal     = "global_seasonal_calendar"
	StatusCompleted         = "COMPLETED"
	StatusFailed            = "FAILED"
	ExplStatusProposed      = "PROPOSED"
	ExplStatusApproved      = "APPROVED"
	ExplStatusRejected      = "REJECTED"
	ExplSourceGemini        = "GEMINI"
	ExplSourceClassicStruct = "CLASSIC_STRUCTURAL"
	DefaultHorizonDays      = 14
	MaxHorizonDays          = 90
	MilliScale              = int64(1000)
	BpsScale                = int64(10000)
)

// Domain sentinel errors.
var (
	ErrInvalidInput                 = errors.New("forecastv2: invalid input")
	ErrFloatForbidden               = errors.New("forecastv2: floating-point numbers are forbidden; use int64 milli or bps")
	ErrRepoUnavailable              = errors.New("forecastv2: repository unavailable")
	ErrRunNotFound                  = errors.New("forecastv2: forecast run not found")
	ErrExplanationNotFound          = errors.New("forecastv2: forecast explanation not found")
	ErrExplanationAlreadyDecided    = errors.New("forecastv2: forecast explanation already decided")
	ErrAgentForbidden               = errors.New("forecastv2: AGENT actor cannot approve or reject demand adjustments")
	ErrAuditorReadOnly              = errors.New("forecastv2: AUDITOR role is strictly read-only")
	ErrSoDViolation                 = errors.New("forecastv2: proposer cannot approve own demand adjustment")
	ErrGeminiExplainerNotConfigured = errors.New("forecastv2: GEMINI_EXPLAINER_REQUIRED=true but Gemini endpoint/credentials are unconfigured")
	ErrGeminiExplainerFailed        = errors.New("forecastv2: Gemini explainer request failed")
)

// RegressorSignal captures one calendar, holiday, payday, price, or promo regressor
// applied by the challenger model (multiplier in basis points; 10000 = 1.00x).
type RegressorSignal struct {
	Name          string `json:"name"`
	Category      string `json:"category"` // CALENDAR | HOLIDAY | PAYDAY | PROMO | PRICE | SEASONAL_OVERRIDE
	MultiplierBps int64  `json:"multiplier_bps"`
	Description   string `json:"description,omitempty"`
}

// EvidenceCitation anchors an explanation to concrete system records or observed metrics.
type EvidenceCitation struct {
	RefType   string `json:"ref_type"` // SERIES | WAPE_BACKTEST | CALENDAR | HOLIDAY | PROMO | PRICE
	RefID     string `json:"ref_id"`
	Label     string `json:"label"`
	ValueText string `json:"value_text"`
}

// SeasonalWindow represents a supplier custom seasonal override from Spanner.
type SeasonalWindow struct {
	OverrideID    string
	TemplateID    string
	Name          string
	StartDate     civil.Date
	EndDate       civil.Date
	MultiplierBps int64
}

// SeriesHistoryInput is one historical SKU/location daily series (in milli-units).
type SeriesHistoryInput struct {
	LocationID       string       `json:"location_id"`
	LocationType     string       `json:"location_type,omitempty"`
	SKU              string       `json:"sku"`
	ProductID        string       `json:"product_id,omitempty"`
	DailyUnitsMilli  []int64      `json:"daily_units_milli"`
	Dates            []civil.Date `json:"-"`
	PromoDiscountBps int64        `json:"promo_discount_bps,omitempty"`
	PriceChangeBps   int64        `json:"price_change_bps,omitempty"`
	ActivePromo      bool         `json:"active_promo,omitempty"`
}

// ForecastRun mirrors ForecastRuns in Spanner (zero floats).
type ForecastRun struct {
	EntityID           string            `json:"entity_id"`
	RunID              string            `json:"run_id"`
	SupplierID         string            `json:"supplier_id"`
	HorizonDays        int64             `json:"horizon_days"`
	Status             string            `json:"status"`
	ChampionModel      string            `json:"champion_model"`
	ChallengerModel    string            `json:"challenger_model"`
	WinnerModel        string            `json:"winner_model"`
	ChampionWapeBps    int64             `json:"champion_wape_bps"`
	ChallengerWapeBps  int64             `json:"challenger_wape_bps"`
	ChampionBiasBps    int64             `json:"champion_bias_bps"`
	ChallengerBiasBps  int64             `json:"challenger_bias_bps"`
	WapeImprovementBps int64             `json:"wape_improvement_bps"`
	SkuCount           int64             `json:"sku_count"`
	LocationCount      int64             `json:"location_count"`
	Regressors         []RegressorSignal `json:"regressors,omitempty"`
	RegressorsJSON     string            `json:"regressors_json,omitempty"`
	FeatureSource      string            `json:"feature_source"`
	TriggeredBy        string            `json:"triggered_by"`
	FailureReason      string            `json:"failure_reason,omitempty"`
	StartedAt          time.Time         `json:"started_at"`
	CompletedAt        *time.Time        `json:"completed_at,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// ForecastValue mirrors ForecastValues in Spanner (zero floats).
type ForecastValue struct {
	EntityID         string            `json:"entity_id"`
	RunID            string            `json:"run_id"`
	ForecastDate     string            `json:"forecast_date"` // YYYY-MM-DD
	LocationID       string            `json:"location_id"`
	LocationType     string            `json:"location_type,omitempty"`
	SKU              string            `json:"sku"`
	ProductID        string            `json:"product_id,omitempty"`
	Model            string            `json:"model"`
	IsWinner         bool              `json:"is_winner"`
	DemandClass      string            `json:"demand_class"`
	P10Milli         int64             `json:"p10_milli"`
	P50Milli         int64             `json:"p50_milli"`
	P90Milli         int64             `json:"p90_milli"`
	BaselineP50Milli int64             `json:"baseline_p50_milli"`
	WapeBps          int64             `json:"wape_bps"`
	BiasBps          int64             `json:"bias_bps"`
	ConfidenceBps    int64             `json:"confidence_bps"`
	Regressors       []RegressorSignal `json:"regressors,omitempty"`
	RegressorsJSON   string            `json:"regressors_json,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// ForecastExplanation mirrors ForecastExplanations in Spanner (zero floats).
type ForecastExplanation struct {
	EntityID            string             `json:"entity_id"`
	ExplanationID       string             `json:"explanation_id"`
	RunID               string             `json:"run_id"`
	LocationID          string             `json:"location_id"`
	SKU                 string             `json:"sku"`
	WindowStart         time.Time          `json:"window_start"`
	WindowEnd           time.Time          `json:"window_end"`
	Source              string             `json:"source"`
	ModelName           string             `json:"model_name,omitempty"`
	SummaryText         string             `json:"summary_text"`
	EvidenceRefs        []EvidenceCitation `json:"evidence_refs,omitempty"`
	EvidenceRefsJSON    string             `json:"evidence_refs_json,omitempty"`
	AdjustmentBps       int64              `json:"adjustment_bps"`
	AdjustmentReason    string             `json:"adjustment_reason,omitempty"`
	Status              string             `json:"status"`
	ProposedBy          string             `json:"proposed_by"`
	ApprovedBy          string             `json:"approved_by,omitempty"`
	ApprovedAt          *time.Time         `json:"approved_at,omitempty"`
	ApprovalNote        string             `json:"approval_note,omitempty"`
	ApprovalRequestID   string             `json:"approval_request_id,omitempty"`
	BaselineP50Milli    int64              `json:"baseline_p50_milli"`
	AdjustedP50Milli    int64              `json:"adjusted_p50_milli"`
	ActualQtyMilli      int64              `json:"actual_qty_milli"`
	ForecastValueAddBps int64              `json:"forecast_value_add_bps"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// RunForecastInput triggers a hybrid champion/challenger forecast run.
type RunForecastInput struct {
	EntityID         string               `json:"entity_id,omitempty"`
	SupplierID       string               `json:"supplier_id,omitempty"`
	LocationID       string               `json:"location_id,omitempty"`
	SKU              string               `json:"sku,omitempty"`
	HorizonDays      int                  `json:"horizon_days,omitempty"`
	PromoDiscountBps int64                `json:"promo_discount_bps,omitempty"`
	PriceChangeBps   int64                `json:"price_change_bps,omitempty"`
	ActivePromo      bool                 `json:"active_promo,omitempty"`
	Explain          *bool                `json:"explain,omitempty"`
	TriggeredBy      string               `json:"triggered_by,omitempty"`
	ActorType        string               `json:"actor_type,omitempty"`
	AsOf             time.Time            `json:"-"`
	Series           []SeriesHistoryInput `json:"series,omitempty"`
}

// RunForecastOutput bundles the persisted run, winner/challenger values, and explanations.
type RunForecastOutput struct {
	Run          ForecastRun           `json:"run"`
	Forecasts    []ForecastValue       `json:"forecasts"`
	Explanations []ForecastExplanation `json:"explanations"`
}

// ForecastQuery filters GET /v1/planning/forecast.
type ForecastQuery struct {
	EntityID    string
	RunID       string
	LocationID  string
	SKU         string
	HorizonDays int
	Limit       int
}

// ForecastView is returned by GET /v1/planning/forecast.
type ForecastView struct {
	Run          *ForecastRun          `json:"run,omitempty"`
	Forecasts    []ForecastValue       `json:"forecasts"`
	Explanations []ForecastExplanation `json:"explanations"`
}

// ProposeAdjustmentInput requests a demand adjustment proposal (Gemini or structural).
type ProposeAdjustmentInput struct {
	EntityID         string `json:"entity_id,omitempty"`
	RunID            string `json:"run_id,omitempty"`
	LocationID       string `json:"location_id"`
	SKU              string `json:"sku"`
	AdjustmentBps    *int64 `json:"adjustment_bps,omitempty"`
	AdjustmentReason string `json:"adjustment_reason,omitempty"`
	Notes            string `json:"notes,omitempty"`
	ProposedBy       string `json:"proposed_by,omitempty"`
	ActorType        string `json:"actor_type,omitempty"`
}

// ApproveAdjustmentInput approves or rejects a proposed demand adjustment and measures FVA.
type ApproveAdjustmentInput struct {
	EntityID       string `json:"entity_id,omitempty"`
	ExplanationID  string `json:"explanation_id"`
	Decision       string `json:"decision,omitempty"` // APPROVE (default) | REJECT
	AdjustmentBps  *int64 `json:"adjustment_bps,omitempty"`
	ActualQtyMilli *int64 `json:"actual_qty_milli,omitempty"`
	Comment        string `json:"comment,omitempty"`
	ActorID        string `json:"-"`
	ActorType      string `json:"-"`
	ActorRole      string `json:"-"`
}
