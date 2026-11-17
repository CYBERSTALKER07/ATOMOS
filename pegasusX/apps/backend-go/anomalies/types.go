// Package anomalies implements ATOMOS §7.7 Anomaly & Fraud Detector Catalogue
// with 5 deterministic integer-scaled detectors over live Cloud Spanner tables,
// deduplicated AnomalyCases persistence, outbox emission, immutable audit trails,
// and optional escalation to governed L1 Proposals.
package anomalies

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

// Canonical §7.7 detector rule codes.
const (
	RuleCashGeofenceMismatch    = "CASH_GEOFENCE_MISMATCH"
	RuleBasketSpikeSharedINN    = "BASKET_SPIKE_SHARED_INN"
	RuleStockCountVariance      = "STOCK_COUNT_VARIANCE"
	RulePricingDiscountAbuse    = "PRICING_DISCOUNT_ABUSE"
	RuleManualJournalAfterHours = "MANUAL_JOURNAL_AFTER_HOURS_OR_SPLIT"
)

// Anomaly categories.
const (
	CategoryCashFraud      = "CASH_FRAUD"
	CategoryCollusionFraud = "COLLUSION_FRAUD"
	CategoryInventoryLoss  = "INVENTORY_LOSS"
	CategoryPricingFraud   = "PRICING_FRAUD"
	CategoryGLControls     = "GL_CONTROLS"
)

// Severities.
const (
	SeverityLow      = "LOW"
	SeverityMedium   = "MEDIUM"
	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"
)

// Anomaly case lifecycle statuses.
const (
	StatusOpen      = "OPEN"
	StatusResolved  = "RESOLVED"
	StatusDismissed = "DISMISSED"
	StatusEscalated = "ESCALATED"
)

// Bounds and limits.
const (
	DefaultListLimit = 50
	MaxListLimit     = 200
	ScanQueryCap     = 1000
	MaxScoreBps      = int64(10000)
)

// Domain sentinel errors.
var (
	ErrInvalidInput        = errors.New("anomalies: invalid input")
	ErrFloatForbidden      = errors.New("anomalies: floating-point numbers are forbidden; use int64 minor units or bps")
	ErrNotFound            = errors.New("anomalies: case not found")
	ErrConflict            = errors.New("anomalies: conflict")
	ErrForbidden           = errors.New("anomalies: forbidden")
	ErrRepoUnavailable     = errors.New("anomalies: repository unavailable")
	ErrProposalUnavailable = errors.New("anomalies: proposal creator unavailable")
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

// ProposalEvidenceRef links a created proposal back to an anomaly case or source object.
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

// CreatedProposal summarizes an L1 governed Proposal created via ProposalCreator.
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

// ServiceConfig wires optional bridges into Service.
type ServiceConfig struct {
	Audit     AuditRecorder
	Proposals ProposalCreator
	Now       func() time.Time
}

// CatalogueRule describes one deterministic anomaly & fraud detector rule.
type CatalogueRule struct {
	RuleCode              string   `json:"rule_code"`
	Name                  string   `json:"name"`
	Category              string   `json:"category"`
	DefaultSeverity       string   `json:"default_severity"`
	Description           string   `json:"description"`
	DefaultThresholdBps   int64    `json:"default_threshold_bps,omitempty"`
	DefaultThresholdMinor int64    `json:"default_threshold_minor,omitempty"`
	PlaybookAction        string   `json:"playbook_action,omitempty"`
	SourceTables          []string `json:"source_tables,omitempty"`
}

// AnomalyCase models one persisted AnomalyCases row.
type AnomalyCase struct {
	TenantID       string         `json:"tenant_id"`
	CaseID         string         `json:"case_id"`
	RuleCode       string         `json:"rule_code"`
	Category       string         `json:"category"`
	Severity       string         `json:"severity"`
	Status         string         `json:"status"`
	ObjectType     string         `json:"object_type"`
	ObjectID       string         `json:"object_id"`
	ExposureMinor  int64          `json:"exposure_minor"`
	ScoreBps       int64          `json:"score_bps"`
	EvidenceJSON   string         `json:"evidence_json,omitempty"`
	Evidence       map[string]any `json:"evidence,omitempty"`
	PlaybookAction string         `json:"playbook_action,omitempty"`
	ProposalID     string         `json:"proposal_id,omitempty"`
	AssignedTo     string         `json:"assigned_to,omitempty"`
	ResolutionNote string         `json:"resolution_note,omitempty"`
	ResolvedBy     string         `json:"resolved_by,omitempty"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
	DetectedAt     time.Time      `json:"detected_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ScanInput triggers deterministic anomaly scanning over live Spanner tables.
type ScanInput struct {
	TenantID       string   `json:"tenant_id,omitempty"`
	EntityID       string   `json:"entity_id,omitempty"`
	RuleCodes      []string `json:"rule_codes,omitempty"`
	ThresholdBps   *int64   `json:"threshold_bps,omitempty"`
	ThresholdMinor *int64   `json:"threshold_minor,omitempty"`
	ActorID        string   `json:"-"`
	ActorType      string   `json:"-"`
	ActorRole      string   `json:"-"`
	TraceID        string   `json:"-"`
	IP             string   `json:"-"`
	DeviceID       string   `json:"-"`
}

// ScanResult reports persisted deduplicated anomaly cases from a scan.
type ScanResult struct {
	TenantID          string        `json:"tenant_id"`
	Cases             []AnomalyCase `json:"cases"`
	CreatedCount      int           `json:"created_count"`
	DeduplicatedCount int           `json:"deduplicated_count"`
	MatchedCount      int           `json:"matched_count"`
	ScannedRules      []string      `json:"scanned_rules"`
	ScanAt            time.Time     `json:"scan_at"`
}

// ListFilter scopes GET /v1/anomalies/cases queries.
type ListFilter struct {
	TenantID string
	Status   string
	Severity string
	Category string
	RuleCode string
	Limit    int
}

// ResolveInput resolves, dismisses, or escalates an AnomalyCase.
type ResolveInput struct {
	TenantID       string `json:"tenant_id,omitempty"`
	CaseID         string `json:"-"`
	Status         string `json:"status"`
	ResolutionNote string `json:"resolution_note"`
	PlaybookAction string `json:"playbook_action,omitempty"`
	AssignedTo     string `json:"assigned_to,omitempty"`
	CreateProposal bool   `json:"create_proposal,omitempty"`
	ActorID        string `json:"-"`
	ActorType      string `json:"-"`
	ActorRole      string `json:"-"`
	IdempotencyKey string `json:"-"`
	TraceID        string `json:"-"`
	IP             string `json:"-"`
	DeviceID       string `json:"-"`
}

// ResolveResult returns the updated case and optional L1 proposal on escalation.
type ResolveResult struct {
	Case       AnomalyCase      `json:"case"`
	Proposal   *CreatedProposal `json:"proposal,omitempty"`
	ProposalID string           `json:"proposal_id,omitempty"`
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

func decodeEvidenceMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil || m == nil {
		return nil
	}
	return m
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
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

func clampBps(v int64) int64 {
	if v < 0 {
		return 0
	}
	if v > MaxScoreBps {
		return MaxScoreBps
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
