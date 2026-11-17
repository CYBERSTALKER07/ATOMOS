// Package proposals implements ATOMOS M1.7 Governed Proposals Inbox and
// three L1 autonomous operational agents (Reorder, Collections, Expiry Rebalance)
// routed strictly through M1.2 CEL approvals, payload-hash verification,
// per-target concurrency locking, fresh precondition checks, and ROI attribution.
package proposals

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Canonical L1 Agent IDs.
const (
	AgentReorder         = "REORDER_AGENT"
	AgentCollections     = "COLLECTIONS_AGENT"
	AgentExpiryRebalance = "EXPIRY_REBALANCE_AGENT"
	AgentHuman           = "HUMAN"
)

// Canonical Proposal Types.
const (
	TypeReorder         = "REORDER"
	TypeCollections     = "COLLECTIONS_NUDGE"
	TypeExpiryRebalance = "EXPIRY_REBALANCE"
	TypeCreditHold      = "CREDIT_HOLD"
	TypeSupplyRequest   = "SUPPLY_REQUEST"
)

// Proposal lifecycle statuses persisted in Proposals.Status.
const (
	StatusPending    = "PENDING"
	StatusApproved   = "APPROVED"
	StatusRejected   = "REJECTED"
	StatusExecuted   = "EXECUTED"
	StatusFailed     = "FAILED"
	StatusExpired    = "EXPIRED"
	StatusSuperseded = "SUPERSEDED"
)

// Autonomy levels for governed proposals (L1 = human-approved, L2 = soak-gated auto-executed).
const (
	AutonomyLevelL1 = "L1"
	AutonomyLevelL2 = "L2"
)

// Domain sentinel errors.
var (
	ErrInvalidInput            = errors.New("proposals: invalid input")
	ErrFloatForbidden          = errors.New("proposals: floating-point numbers are forbidden; use int64 minor units or bps")
	ErrProposalNotFound        = errors.New("proposals: proposal not found")
	ErrAgentApprovalForbidden  = errors.New("proposals: AGENT actor cannot approve, reject, or execute proposals")
	ErrAuditorReadOnly         = errors.New("proposals: AUDITOR role is strictly read-only")
	ErrSoDViolation            = errors.New("proposals: segregation of duties violation (creator cannot approve own proposal)")
	ErrPayloadHashMismatch     = errors.New("proposals: payload hash mismatch")
	ErrInvalidTransition       = errors.New("proposals: invalid proposal status transition")
	ErrAlreadyExecuted         = errors.New("proposals: proposal already executed")
	ErrTargetExecutionConflict = errors.New("proposals: overlapping proposal already executed or locked for target")
	ErrPreconditionFailed      = errors.New("proposals: fresh execution preconditions failed")
	ErrRepoUnavailable         = errors.New("proposals: repository unavailable")
)

// EvidenceRef links a proposal to a live Spanner domain object backing the recommendation.
type EvidenceRef struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// Proposal represents one persisted agent or human operational proposal in Proposals.
type Proposal struct {
	TenantID             string         `json:"tenant_id"`
	ProposalID           string         `json:"proposal_id"`
	AgentID              string         `json:"agent_id"`
	ProposalType         string         `json:"proposal_type"`
	TargetObjectType     string         `json:"target_object_type"`
	TargetObjectID       string         `json:"target_object_id"`
	Title                string         `json:"title"`
	Summary              string         `json:"summary"`
	Rationale            string         `json:"rationale"`
	PayloadJSON          string         `json:"payload_json"`
	Payload              map[string]any `json:"payload,omitempty"`
	PayloadHash          string         `json:"payload_hash"`
	EvidenceRefsJSON     string         `json:"-"`
	EvidenceRefs         []EvidenceRef  `json:"evidence_refs"`
	ConfidenceBps        int64          `json:"confidence_bps"`
	EstimatedImpactMinor int64          `json:"estimated_impact_minor"`
	Currency             string         `json:"currency"`
	RoiCategory          string         `json:"roi_category,omitempty"`
	AutonomyLevel        string         `json:"autonomy_level,omitempty"`
	Status               string         `json:"status"`
	ApprovalRequestID    string         `json:"approval_request_id,omitempty"`
	CreatedBy            string         `json:"created_by"`
	ActorType            string         `json:"actor_type"`
	DecidedBy            string         `json:"decided_by,omitempty"`
	DecisionComment      string         `json:"decision_comment,omitempty"`
	ExecutedBy           string         `json:"executed_by,omitempty"`
	ExecutionResultJSON  string         `json:"-"`
	ExecutionResult      map[string]any `json:"execution_result,omitempty"`
	ExecutionError       string         `json:"execution_error,omitempty"`
	RoiEventID           string         `json:"roi_event_id,omitempty"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty"`
	DecidedAt            *time.Time     `json:"decided_at,omitempty"`
	ExecutedAt           *time.Time     `json:"executed_at,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	L2SoakContext        *L2SoakMetrics `json:"-"`
}

// ListFilter scopes GET /v1/proposals queries.
type ListFilter struct {
	TenantID   string
	AgentID    string
	Status     string
	TargetType string
	TargetID   string
	Limit      int
}

// CreateProposalInput captures an agent or human proposal submission.
type CreateProposalInput struct {
	TenantID             string         `json:"tenant_id,omitempty"`
	AgentID              string         `json:"agent_id,omitempty"`
	ProposalType         string         `json:"proposal_type"`
	TargetObjectType     string         `json:"target_object_type"`
	TargetObjectID       string         `json:"target_object_id"`
	Title                string         `json:"title"`
	Summary              string         `json:"summary"`
	Rationale            string         `json:"rationale"`
	Payload              map[string]any `json:"payload,omitempty"`
	PayloadJSON          string         `json:"payload_json,omitempty"`
	EvidenceRefs         []EvidenceRef  `json:"evidence_refs,omitempty"`
	ConfidenceBps        int64          `json:"confidence_bps,omitempty"`
	EstimatedImpactMinor int64          `json:"estimated_impact_minor,omitempty"`
	Currency             string         `json:"currency,omitempty"`
	RoiCategory          string         `json:"roi_category,omitempty"`
	CreatedBy            string         `json:"-"`
	ActorType            string         `json:"-"`
	ApproverRole         string         `json:"-"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	ExpiresInHours       int            `json:"expires_in_hours,omitempty"`
}

// ApproveOrEditInput approves a pending proposal, optionally editing its payload before approval.
type ApproveOrEditInput struct {
	TenantID          string         `json:"tenant_id,omitempty"`
	ProposalID        string         `json:"proposal_id"`
	ApproverID        string         `json:"-"`
	ActorType         string         `json:"-"`
	ApproverRole      string         `json:"-"`
	Comment           string         `json:"comment,omitempty"`
	ExpectedHash      string         `json:"payload_hash,omitempty"`
	EditedPayload     map[string]any `json:"edited_payload,omitempty"`
	EditedPayloadJSON string         `json:"edited_payload_json,omitempty"`
	MFAVerified       bool           `json:"mfa_verified,omitempty"`
	TOTPCode          string         `json:"totp_code,omitempty"`
	BreakGlass        bool           `json:"break_glass,omitempty"`
	BreakGlassReason  string         `json:"break_glass_reason,omitempty"`
	RoutePath         string         `json:"-"`
	TraceID           string         `json:"-"`
	IP                string         `json:"-"`
	DeviceID          string         `json:"-"`
}

// RejectInput rejects a pending proposal.
type RejectInput struct {
	TenantID     string `json:"tenant_id,omitempty"`
	ProposalID   string `json:"proposal_id"`
	ReviewerID   string `json:"-"`
	ActorType    string `json:"-"`
	ReviewerRole string `json:"-"`
	Comment      string `json:"comment,omitempty"`
	RoutePath    string `json:"-"`
	TraceID      string `json:"-"`
	IP           string `json:"-"`
	DeviceID     string `json:"-"`
}

// ExecuteInput executes an APPROVED proposal under target lock and fresh precondition checks.
type ExecuteInput struct {
	TenantID     string `json:"tenant_id,omitempty"`
	ProposalID   string `json:"proposal_id"`
	ExecutorID   string `json:"-"`
	ActorType    string `json:"-"`
	ExecutorRole string `json:"-"`
	ExpectedHash string `json:"payload_hash,omitempty"`
}

// ExecuteResult returns the executed proposal, execution outcome map, and attributed ROI event ID.
type ExecuteResult struct {
	Proposal        Proposal       `json:"proposal"`
	ExecutionResult map[string]any `json:"execution_result"`
	RoiEvent        any            `json:"roi_event,omitempty"`
}

// CanonicalizePayload converts a map payload into deterministic canonical JSON and SHA-256 hex.
func CanonicalizePayload(payload map[string]any) (canonicalJSON string, sha256Hex string, err error) {
	if payload == nil {
		payload = map[string]any{}
	}
	if err := validateNoFloats(payload); err != nil {
		return "", "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("%w: marshal payload: %v", ErrInvalidInput, err)
	}
	return CanonicalizeJSONBytes(raw)
}

// CanonicalizeJSONBytes normalizes raw JSON into sorted-key canonical JSON and SHA-256 hex, rejecting floats.
func CanonicalizeJSONBytes(raw []byte) (canonicalJSON string, sha256Hex string, err error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		trimmed = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var val any
	if err := dec.Decode(&val); err != nil {
		return "", "", fmt.Errorf("%w: invalid json: %v", ErrInvalidInput, err)
	}
	if err := validateNoFloats(val); err != nil {
		return "", "", err
	}
	norm, err := json.Marshal(val)
	if err != nil {
		return "", "", fmt.Errorf("%w: canonical marshal: %v", ErrInvalidInput, err)
	}
	sum := sha256.Sum256(norm)
	return string(norm), hex.EncodeToString(sum[:]), nil
}

func decodePayloadMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil || m == nil {
		return map[string]any{}
	}
	return m
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
			return fmt.Errorf("%w: unsupported value %T: %v", ErrInvalidInput, tv, err)
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
