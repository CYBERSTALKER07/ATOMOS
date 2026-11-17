// Package whatif implements ATOMOS §7.9 Full What-If Simulator E2E across
// PlanningScenarios/ScenarioObjectPatches, ForecastValues, SupplierInventoryV2,
// Orders/RoutePerformanceAnalytics, and ArInvoices/RetailerCreditProfiles in pure
// int64 minor/bps arithmetic, persisting WhatIfSimulations and turning recommended
// actions into governed L1 Proposals (never mutating domain tables directly).
package whatif

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// Simulation lifecycle statuses persisted in WhatIfSimulations.Status.
const (
	StatusSimulated = "SIMULATED"
	StatusCommitted = "COMMITTED"
)

// Canonical limits and defaults.
const (
	DefaultHorizonDays int64 = 30
	MinHorizonDays     int64 = 1
	MaxHorizonDays     int64 = 365
	MinDeltaBps        int64 = -9999
	MaxDeltaBps        int64 = 100000
	DefaultListLimit   int   = 50
	MaxListLimit       int   = 200
)

// Domain sentinel errors.
var (
	ErrInvalidInput        = errors.New("whatif: invalid input")
	ErrFloatForbidden      = errors.New("whatif: floating-point numbers are forbidden; use int64 minor units or bps")
	ErrNotFound            = errors.New("whatif: simulation not found")
	ErrConflict            = errors.New("whatif: conflict")
	ErrForbidden           = errors.New("whatif: forbidden")
	ErrRepoUnavailable     = errors.New("whatif: repository unavailable")
	ErrProposalUnavailable = errors.New("whatif: proposal creator unavailable")
)

// AuditRecordInput mirrors audit.EventInput without importing audit.
type AuditRecordInput struct {
	TenantID   string `json:"tenant_id"`
	ActorID    string `json:"actor_id"`
	ActorType  string `json:"actor_type"`
	Role       string `json:"role,omitempty"`
	Action     string `json:"action"`
	ObjectType string `json:"object_type,omitempty"`
	ObjectID   string `json:"object_id,omitempty"`
	BeforeJSON string `json:"before_json,omitempty"`
	AfterJSON  string `json:"after_json,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	IP         string `json:"ip,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
}

// AuditRecorder persists immutable hash-chained audit events both standalone and in-txn.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
	RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev AuditRecordInput) error
}

// ProposalEvidenceRef links a proposal to the simulation or underlying live object.
type ProposalEvidenceRef struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// ProposalCreateInput mirrors proposals.CreateProposalInput without importing proposals.
type ProposalCreateInput struct {
	TenantID             string                `json:"tenant_id"`
	AgentID              string                `json:"agent_id,omitempty"`
	ProposalType         string                `json:"proposal_type"`
	TargetObjectType     string                `json:"target_object_type"`
	TargetObjectID       string                `json:"target_object_id"`
	Title                string                `json:"title"`
	Summary              string                `json:"summary,omitempty"`
	Rationale            string                `json:"rationale,omitempty"`
	Payload              map[string]any        `json:"payload,omitempty"`
	EvidenceRefs         []ProposalEvidenceRef `json:"evidence_refs,omitempty"`
	ConfidenceBps        int64                 `json:"confidence_bps,omitempty"`
	EstimatedImpactMinor int64                 `json:"estimated_impact_minor,omitempty"`
	Currency             string                `json:"currency,omitempty"`
	RoiCategory          string                `json:"roi_category,omitempty"`
	CreatedBy            string                `json:"created_by,omitempty"`
	ActorType            string                `json:"actor_type,omitempty"`
	IdempotencyKey       string                `json:"idempotency_key,omitempty"`
	ExpiresInHours       int                   `json:"expires_in_hours,omitempty"`
}

// CreatedProposal summarizes an L1 governed Proposal returned by ProposalCreator.
type CreatedProposal struct {
	TenantID             string `json:"tenant_id"`
	ProposalID           string `json:"proposal_id"`
	AgentID              string `json:"agent_id"`
	ProposalType         string `json:"proposal_type"`
	TargetObjectType     string `json:"target_object_type"`
	TargetObjectID       string `json:"target_object_id"`
	Title                string `json:"title"`
	Summary              string `json:"summary,omitempty"`
	Status               string `json:"status"`
	ApprovalRequestID    string `json:"approval_request_id,omitempty"`
	PayloadHash          string `json:"payload_hash,omitempty"`
	EstimatedImpactMinor int64  `json:"estimated_impact_minor"`
	ConfidenceBps        int64  `json:"confidence_bps"`
	Currency             string `json:"currency"`
	CreatedBy            string `json:"created_by"`
}

// ProposalCreator creates governed L1 Proposals (never self-approved).
type ProposalCreator interface {
	CreateProposal(ctx context.Context, in ProposalCreateInput) (CreatedProposal, error)
}

// ServiceConfig configures What-If Simulator dependencies.
type ServiceConfig struct {
	Audit     AuditRecorder
	Proposals ProposalCreator
	Now       func() time.Time
}

// WhatIfShocks holds integer-scaled shock inputs for a simulation.
type WhatIfShocks struct {
	DemandDeltaBps       int64 `json:"demand_delta_bps"`
	FactoryDowntimeHours int64 `json:"factory_downtime_hours"`
	FxRateShockBps       int64 `json:"fx_rate_shock_bps"`
	CreditLimitDeltaBps  int64 `json:"credit_limit_delta_bps"`
	UnitPriceDeltaBps    int64 `json:"unit_price_delta_bps"`
}

// DemandStageBreakdown captures stage 1-2 forecast & revenue deltas in int64 minor/milli units.
type DemandStageBreakdown struct {
	BaselineRevenueMinor int64 `json:"baseline_revenue_minor"`
	ShockedRevenueMinor  int64 `json:"shocked_revenue_minor"`
	ForecastDeltaMinor   int64 `json:"forecast_delta_minor"`
	BaselineUnitsMilli   int64 `json:"baseline_units_milli"`
	ShockedUnitsMilli    int64 `json:"shocked_units_milli"`
	SkuCount             int   `json:"sku_count"`
}

// ReplenishmentStageBreakdown captures stage 3 inventory & FX procurement impact.
type ReplenishmentStageBreakdown struct {
	BaselineCostMinor   int64    `json:"baseline_cost_minor"`
	ShockedCostMinor    int64    `json:"shocked_cost_minor"`
	CostDeltaMinor      int64    `json:"cost_delta_minor"`
	ShortfallUnitsMilli int64    `json:"shortfall_units_milli"`
	AffectedSkus        []string `json:"affected_skus"`
}

// DispatchStageBreakdown captures stage 4 fleet/route cost impact.
type DispatchStageBreakdown struct {
	BaselineCostMinor      int64 `json:"baseline_cost_minor"`
	ShockedCostMinor       int64 `json:"shocked_cost_minor"`
	DispatchCostDeltaMinor int64 `json:"dispatch_cost_delta_minor"`
	RouteCount             int   `json:"route_count"`
	OrderCount             int   `json:"order_count"`
}

// CreditStageBreakdown captures stage 5 AR & credit exposure impact.
type CreditStageBreakdown struct {
	BaselineExposureMinor    int64 `json:"baseline_exposure_minor"`
	ShockedExposureMinor     int64 `json:"shocked_exposure_minor"`
	CreditExposureDeltaMinor int64 `json:"credit_exposure_delta_minor"`
	OverLimitRetailers       int   `json:"over_limit_retailers"`
	RiskCostDeltaMinor       int64 `json:"risk_cost_delta_minor"`
}

// PnLStageBreakdown captures stage 6 net profit & loss summary.
type PnLStageBreakdown struct {
	CogsDeltaMinor        int64 `json:"cogs_delta_minor"`
	GrossMarginDeltaMinor int64 `json:"gross_margin_delta_minor"`
	NetPnLDeltaMinor      int64 `json:"net_pnl_delta_minor"`
}

// RecommendedAction represents one governed L1 proposal candidate derived from a simulation.
type RecommendedAction struct {
	ProposalType         string         `json:"proposal_type"`
	AgentID              string         `json:"agent_id"`
	TargetObjectType     string         `json:"target_object_type"`
	TargetObjectID       string         `json:"target_object_id"`
	Title                string         `json:"title"`
	Summary              string         `json:"summary"`
	EstimatedImpactMinor int64          `json:"estimated_impact_minor"`
	ConfidenceBps        int64          `json:"confidence_bps"`
	Payload              map[string]any `json:"payload"`
}

// WhatIfBreakdown stores the stage-by-stage breakdown and candidate actions in BreakdownJson.
type WhatIfBreakdown struct {
	Currency             string                      `json:"currency"`
	ScenarioApplied      bool                        `json:"scenario_applied"`
	PatchCount           int                         `json:"patch_count"`
	Demand               DemandStageBreakdown        `json:"demand"`
	Replenishment        ReplenishmentStageBreakdown `json:"replenishment"`
	Dispatch             DispatchStageBreakdown      `json:"dispatch"`
	Credit               CreditStageBreakdown        `json:"credit"`
	PnL                  PnLStageBreakdown           `json:"pnl"`
	RecommendedActions   []RecommendedAction         `json:"recommended_actions"`
	CommitIdempotencyKey string                      `json:"commit_idempotency_key,omitempty"`
}

// WhatIfSimulation models one persisted WhatIfSimulations row.
type WhatIfSimulation struct {
	TenantID                 string           `json:"tenant_id"`
	SimulationID             string           `json:"simulation_id"`
	ScenarioID               string           `json:"scenario_id,omitempty"`
	Name                     string           `json:"name"`
	EntityID                 string           `json:"entity_id,omitempty"`
	HorizonDays              int64            `json:"horizon_days"`
	ShocksJSON               string           `json:"shocks_json,omitempty"`
	Shocks                   WhatIfShocks     `json:"shocks"`
	ForecastDeltaMinor       int64            `json:"forecast_delta_minor"`
	ReplenishmentCostMinor   int64            `json:"replenishment_cost_minor"`
	DispatchCostDeltaMinor   int64            `json:"dispatch_cost_delta_minor"`
	CreditExposureDeltaMinor int64            `json:"credit_exposure_delta_minor"`
	NetPnLDeltaMinor         int64            `json:"net_pnl_delta_minor"`
	BreakdownJSON            string           `json:"breakdown_json,omitempty"`
	Breakdown                *WhatIfBreakdown `json:"breakdown,omitempty"`
	CommittedProposalIdsJSON string           `json:"committed_proposal_ids_json,omitempty"`
	CommittedProposalIDs     []string         `json:"committed_proposal_ids"`
	Status                   string           `json:"status"`
	CreatedBy                string           `json:"created_by"`
	CreatedAt                time.Time        `json:"created_at"`
	UpdatedAt                time.Time        `json:"updated_at"`
}

// SimulateInput triggers an E2E What-If simulation over live Spanner data.
type SimulateInput struct {
	TenantID             string `json:"tenant_id,omitempty"`
	Name                 string `json:"name"`
	ScenarioID           string `json:"scenario_id,omitempty"`
	EntityID             string `json:"entity_id,omitempty"`
	HorizonDays          int64  `json:"horizon_days,omitempty"`
	DemandDeltaBps       int64  `json:"demand_delta_bps"`
	FactoryDowntimeHours int64  `json:"factory_downtime_hours"`
	FxRateShockBps       int64  `json:"fx_rate_shock_bps"`
	CreditLimitDeltaBps  int64  `json:"credit_limit_delta_bps"`
	UnitPriceDeltaBps    int64  `json:"unit_price_delta_bps"`
	Currency             string `json:"currency,omitempty"`
	IdempotencyKey       string `json:"idempotency_key,omitempty"`
	ActorID              string `json:"-"`
	ActorType            string `json:"-"`
	ActorRole            string `json:"-"`
	TraceID              string `json:"-"`
	IP                   string `json:"-"`
	DeviceID             string `json:"-"`
}

// ListFilter scopes GET /v1/whatif/simulations queries.
type ListFilter struct {
	TenantID string
	Status   string
	Limit    int
}

// CommitProposalsInput commits recommended actions from a simulation into L1 Proposals.
type CommitProposalsInput struct {
	TenantID       string   `json:"tenant_id,omitempty"`
	SimulationID   string   `json:"-"`
	ProposalTypes  []string `json:"proposal_types,omitempty"`
	AgentID        string   `json:"agent_id,omitempty"`
	ActorID        string   `json:"-"`
	ActorType      string   `json:"-"`
	ActorRole      string   `json:"-"`
	IdempotencyKey string   `json:"-"`
	TraceID        string   `json:"-"`
	IP             string   `json:"-"`
	DeviceID       string   `json:"-"`
}

// CommitProposalsResult returns the updated simulation and created L1 Proposals.
type CommitProposalsResult struct {
	Simulation           WhatIfSimulation  `json:"simulation"`
	Proposals            []CreatedProposal `json:"proposals"`
	CommittedProposalIDs []string          `json:"committed_proposal_ids"`
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
			return fmt.Errorf("%w: marshal %T: %v", ErrInvalidInput, tv, err)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return fmt.Errorf("%w: decode %T: %v", ErrInvalidInput, tv, err)
		}
		return validateNoFloats(parsed)
	}
}

func mulDivInt64(a, b, denom int64) int64 {
	if denom == 0 || a == 0 || b == 0 {
		return 0
	}
	prod := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	q := new(big.Int).Quo(prod, big.NewInt(denom))
	if q.IsInt64() {
		return q.Int64()
	}
	if q.Sign() < 0 {
		return -1 << 62
	}
	return 1 << 62
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func clampBps(v int64) int64 {
	if v < 0 {
		return 0
	}
	if v > 10000 {
		return 10000
	}
	return v
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return DefaultListLimit
	}
	if limit > MaxListLimit {
		return MaxListLimit
	}
	return limit
}

func nonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
