package credit

import (
	"context"
	"errors"
	"time"
)

// Credit score scopes persisted in CreditScoreHistory.
const (
	ScoreScopeRelationship = "RELATIONSHIP"
	ScoreScopeNetwork      = "NETWORK"
	ScorerSourceScorecard  = "SCORECARD"
	ScorerSourceMLSidecar  = "ML_SIDECAR"
)

// V2 sentinel errors.
var (
	ErrFloatForbidden          = errors.New("credit: floating-point numbers are forbidden; use int64 minor or bps")
	ErrInvalidInput            = errors.New("credit: invalid input")
	ErrMLScorerNotConfigured   = errors.New("credit: CREDIT_ML_SCORER_REQUIRED=true but CREDIT_ML_SCORER_URL is unset")
	ErrMLScorerFailed          = errors.New("credit: ML scorer request failed")
	ErrAgentApprovalForbidden  = errors.New("credit: AGENT actor cannot approve or modify credit limits")
	ErrAuditorReadOnly         = errors.New("credit: AUDITOR role is strictly read-only")
	ErrSoDViolation            = errors.New("credit: segregation of duties violation")
	ErrApprovalRequired        = errors.New("credit: credit limit change requires approval")
	ErrApprovalPayloadMismatch = errors.New("credit: approval payload hash mismatch")
)

// ReasonCode explains one concrete scoring or limit decision factor.
type ReasonCode struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	ImpactBps int64  `json:"impact_bps"`
	Detail    string `json:"detail"`
	Source    string `json:"source"` // SCORECARD | ML_SIDECAR | DISPUTE_ISOLATION | CONSENT
}

// CreditScoreRecord mirrors CreditScoreHistory in Spanner (zero floats).
type CreditScoreRecord struct {
	ScoreID               string       `json:"score_id"`
	RetailerID            string       `json:"retailer_id"`
	SupplierID            string       `json:"supplier_id,omitempty"`
	Scope                 string       `json:"scope"` // RELATIONSHIP | NETWORK
	ConsentVerified       bool         `json:"consent_verified"`
	ConsentScope          string       `json:"consent_scope,omitempty"`
	ModelVersion          string       `json:"model_version"`
	ScorerSource          string       `json:"scorer_source"`
	Score                 int64        `json:"score"`
	PdBps                 int64        `json:"pd_bps"`
	RiskTier              RiskTier     `json:"risk_tier"`
	SuggestedLimitMinor   int64        `json:"suggested_limit_minor"`
	CurrentLimitMinor     int64        `json:"current_limit_minor"`
	ReasonCodes           []ReasonCode `json:"reason_codes"`
	ReasonCodesJSON       string       `json:"reason_codes_json,omitempty"`
	FactorsJSON           string       `json:"factors_json,omitempty"`
	FeaturesHash          string       `json:"features_hash"`
	DisputesIsolatedCount int64        `json:"disputes_isolated_count"`
	WindowStart           time.Time    `json:"window_start"`
	WindowEnd             time.Time    `json:"window_end"`
	ComputedAt            time.Time    `json:"computed_at"`
	CreatedAt             time.Time    `json:"created_at,omitempty"`
}

// RetailerScoreView bundles relationship score, network score, and recent CreditScoreHistory.
type RetailerScoreView struct {
	RetailerID        string              `json:"retailer_id"`
	RelationshipScore *CreditScoreRecord  `json:"relationship_score,omitempty"`
	NetworkScore      *CreditScoreRecord  `json:"network_score,omitempty"`
	History           []CreditScoreRecord `json:"history"`
}

// PortfolioSummary aggregates portfolio exposure and scoring metrics for a supplier.
type PortfolioSummary struct {
	TotalRetailers       int64 `json:"total_retailers"`
	TotalExposureMinor   int64 `json:"total_exposure_minor"`
	TotalLimitMinor      int64 `json:"total_limit_minor"`
	UtilizationBps       int64 `json:"utilization_bps"`
	FrozenCount          int64 `json:"frozen_count"`
	HighRiskCount        int64 `json:"high_risk_count"`
	AvgScore             int64 `json:"avg_score"`
	ConsentVerifiedCount int64 `json:"consent_verified_count"`
}

// PortfolioItem represents one shop row on GET /v1/credit/portfolio.
type PortfolioItem struct {
	RetailerID           string       `json:"retailer_id"`
	RetailerName         string       `json:"retailer_name,omitempty"`
	CreditLimitMinor     int64        `json:"credit_limit_minor"`
	CurrentBalanceMinor  int64        `json:"current_balance_minor"`
	ReservedMinor        int64        `json:"reserved_minor"`
	AvailableCreditMinor int64        `json:"available_credit_minor"`
	UtilizationBps       int64        `json:"utilization_bps"`
	RiskScore            int64        `json:"risk_score"`
	PdBps                int64        `json:"pd_bps"`
	RiskTier             RiskTier     `json:"risk_tier"`
	Status               Status       `json:"status"`
	PrepayOnlyLocked     bool         `json:"prepay_only_locked"`
	ConsentVerified      bool         `json:"consent_verified"`
	SuggestedLimitMinor  int64        `json:"suggested_limit_minor"`
	ReasonCodes          []ReasonCode `json:"reason_codes"`
	LastEvaluatedAt      string       `json:"last_evaluated_at,omitempty"`
}

// PortfolioResponse is returned by GET /v1/credit/portfolio.
type PortfolioResponse struct {
	SupplierID string           `json:"supplier_id"`
	Summary    PortfolioSummary `json:"summary"`
	Items      []PortfolioItem  `json:"items"`
}

// LimitChangeApprovalInput passes credit limit change metadata to M1.2 CEL approvals.
type LimitChangeApprovalInput struct {
	TenantID       string
	RetailerID     string
	SupplierID     string
	OldLimitMinor  int64
	NewLimitMinor  int64
	Reason         string
	ReasonCodes    []ReasonCode
	ActorID        string
	ActorType      string
	ActorRole      string
	ApprovalID     string
	IdempotencyKey string
	Context        map[string]any
	Payload        map[string]any
}

// LimitChangeApprovalResult describes the CEL approval gate result for a limit change.
type LimitChangeApprovalResult struct {
	Required          bool
	Approved          bool
	ApprovalRequestID string
	PolicyID          string
	PolicyVersion     int64
	PayloadHash       string
	Status            string
}

// LimitApprovalBridge connects credit limit changes to M1.2 CEL Approvals without circular imports.
type LimitApprovalBridge interface {
	EvaluateOrVerifyLimitChange(ctx context.Context, in LimitChangeApprovalInput) (LimitChangeApprovalResult, error)
}
