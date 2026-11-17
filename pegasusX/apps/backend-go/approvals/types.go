// Package approvals implements ATOMOS M1.2 CEL-driven multi-step approval
// policies, immutable SHA-256 payload pinning, delegation with expiry checks,
// maker-checker SoD, MFA step-up thresholds, break-glass post-review tracking,
// and Telegram webhook callback approvals.
package approvals

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

// Approval lifecycle statuses matching ApprovalRequests.Status.
const (
	StatusPending   = "PENDING"
	StatusApproved  = "APPROVED"
	StatusRejected  = "REJECTED"
	StatusExpired   = "EXPIRED"
	StatusCancelled = "CANCELLED"
)

// Decision values matching ApprovalDecisions.Decision.
const (
	DecisionApprove = "APPROVE"
	DecisionReject  = "REJECT"
)

// Domain sentinel errors mapped cleanly to HTTP status codes.
var (
	ErrInvalidInput           = errors.New("approvals: invalid input")
	ErrInvalidCEL             = errors.New("approvals: invalid CEL expression")
	ErrFloatForbidden         = errors.New("approvals: floating-point numbers are forbidden; use int64 minor units or bps")
	ErrPolicyNotFound         = errors.New("approvals: approval policy not found")
	ErrRequestNotFound        = errors.New("approvals: approval request not found")
	ErrDelegationNotFound     = errors.New("approvals: approval delegation not found")
	ErrAgentApprovalForbidden = errors.New("approvals: AGENT actor cannot approve or reject requests")
	ErrAuditorReadOnly        = errors.New("approvals: AUDITOR role is strictly read-only")
	ErrSoDViolation           = errors.New("approvals: segregation of duties violation (requester cannot approve own request)")
	ErrRoleNotAuthorized      = errors.New("approvals: caller role not authorized for current approval step")
	ErrMFARequired            = errors.New("approvals: MFA verification required for approval threshold")
	ErrPayloadHashMismatch    = errors.New("approvals: payload hash mismatch (payload mutated since request creation)")
	ErrRequestAlreadyResolved = errors.New("approvals: approval request already resolved")
	ErrRequestExpired         = errors.New("approvals: approval request has expired")
	ErrAlreadyDecidedByActor  = errors.New("approvals: actor has already recorded a decision on this step")
	ErrRepoUnavailable        = errors.New("approvals: repository unavailable")
	ErrWebhookUnauthorized    = errors.New("approvals: unauthorized webhook secret")
)

// ApprovalStep defines one sequential stage in an ApprovalPolicy workflow.
type ApprovalStep struct {
	StepIndex         int      `json:"step_index"`
	Name              string   `json:"name,omitempty"`
	AllowedRoles      []string `json:"allowed_roles,omitempty"`
	RequiredApprovers int      `json:"required_approvers"`
	RequireMFA        bool     `json:"require_mfa,omitempty"`
	MFAThresholdMinor int64    `json:"mfa_threshold_minor,omitempty"`
	TimeoutHours      int      `json:"timeout_hours,omitempty"`
}

// Policy represents one versioned CEL rule stored in ApprovalPolicies.
type Policy struct {
	TenantID      string         `json:"tenant_id"`
	PolicyID      string         `json:"policy_id"`
	Name          string         `json:"name"`
	ObjectType    string         `json:"object_type"`
	Action        string         `json:"action"`
	ConditionExpr string         `json:"condition_expr"`
	Steps         []ApprovalStep `json:"steps"`
	Version       int64          `json:"version"`
	Enabled       bool           `json:"enabled"`
	CreatedBy     string         `json:"created_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// ApprovalRequest represents one persisted workflow instance in ApprovalRequests.
type ApprovalRequest struct {
	TenantID          string     `json:"tenant_id"`
	RequestID         string     `json:"request_id"`
	PolicyID          string     `json:"policy_id"`
	PolicyVersion     int64      `json:"policy_version"`
	ObjectType        string     `json:"object_type"`
	ObjectID          string     `json:"object_id"`
	Action            string     `json:"action"`
	AmountMinor       int64      `json:"amount_minor"`
	RequestedBy       string     `json:"requested_by"`
	ActorType         string     `json:"actor_type"`
	PayloadJSON       string     `json:"payload_json"`
	PayloadHash       string     `json:"payload_hash"`
	Status            string     `json:"status"`
	CurrentStep       int64      `json:"current_step"`
	RequiredApprovers int64      `json:"required_approvers"`
	BreakGlass        bool       `json:"break_glass"`
	BreakGlassReason  string     `json:"break_glass_reason,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	IdempotencyKey    string     `json:"idempotency_key,omitempty"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ApprovalDecision represents one step decision recorded in ApprovalDecisions.
type ApprovalDecision struct {
	TenantID      string    `json:"tenant_id"`
	RequestID     string    `json:"request_id"`
	DecisionID    string    `json:"decision_id"`
	StepIndex     int64     `json:"step_index"`
	ApproverID    string    `json:"approver_id"`
	DelegatedFrom string    `json:"delegated_from,omitempty"`
	Decision      string    `json:"decision"`
	Comment       string    `json:"comment,omitempty"`
	MfaVerified   bool      `json:"mfa_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

// ApprovalDelegation represents a time-bounded approval delegation in ApprovalDelegations.
type ApprovalDelegation struct {
	TenantID     string    `json:"tenant_id"`
	DelegationID string    `json:"delegation_id"`
	DelegatorID  string    `json:"delegator_id"`
	DelegateID   string    `json:"delegate_id"`
	RoleScope    string    `json:"role_scope,omitempty"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	Reason       string    `json:"reason"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreatePolicyInput defines fields for creating or updating an approval policy.
type CreatePolicyInput struct {
	TenantID      string         `json:"tenant_id,omitempty"`
	PolicyID      string         `json:"policy_id,omitempty"`
	Name          string         `json:"name"`
	ObjectType    string         `json:"object_type"`
	Action        string         `json:"action"`
	ConditionExpr string         `json:"condition_expr"`
	Steps         []ApprovalStep `json:"steps"`
	Enabled       *bool          `json:"enabled,omitempty"`
	CreatedBy     string         `json:"created_by,omitempty"`
}

// UpdatePolicyInput defines partial or full policy modifications (bumping Version).
type UpdatePolicyInput struct {
	TenantID      string          `json:"tenant_id,omitempty"`
	PolicyID      string          `json:"policy_id"`
	Name          *string         `json:"name,omitempty"`
	ObjectType    *string         `json:"object_type,omitempty"`
	Action        *string         `json:"action,omitempty"`
	ConditionExpr *string         `json:"condition_expr,omitempty"`
	Steps         *[]ApprovalStep `json:"steps,omitempty"`
	Enabled       *bool           `json:"enabled,omitempty"`
	UpdatedBy     string          `json:"updated_by,omitempty"`
}

// SubmitRequestInput evaluates active policies for (ObjectType, Action) and creates an ApprovalRequest
// if any enabled policy's CEL condition matches (or if ForceCreate is true).
type SubmitRequestInput struct {
	TenantID          string         `json:"tenant_id"`
	ObjectType        string         `json:"object_type"`
	ObjectID          string         `json:"object_id"`
	Action            string         `json:"action"`
	AmountMinor       int64          `json:"amount_minor"`
	RequestedBy       string         `json:"requested_by"`
	ActorType         string         `json:"actor_type"`
	Context           map[string]any `json:"context,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
	PayloadJSON       string         `json:"payload_json,omitempty"`
	IdempotencyKey    string         `json:"idempotency_key,omitempty"`
	RequiredApprovers int64          `json:"required_approvers,omitempty"`
	MinApprovers      int64          `json:"min_approvers,omitempty"`
	ForceCreate       bool           `json:"force_create,omitempty"`
	BreakGlass        bool           `json:"break_glass,omitempty"`
	BreakGlassReason  string         `json:"break_glass_reason,omitempty"`
	ApproverRole      string         `json:"approver_role,omitempty"`
	MFAVerified       bool           `json:"mfa_verified,omitempty"`
	ExpiresInHours    int            `json:"expires_in_hours,omitempty"`
}

// SubmitRequestResult reports whether an approval workflow was triggered and its request record.
type SubmitRequestResult struct {
	Required bool             `json:"required"`
	Matched  bool             `json:"matched"`
	Request  *ApprovalRequest `json:"request,omitempty"`
	Policy   *Policy          `json:"policy,omitempty"`
}

// DecideInput captures an approve or reject action against an ApprovalRequest.
type DecideInput struct {
	TenantID         string `json:"tenant_id"`
	RequestID        string `json:"request_id"`
	ApproverID       string `json:"approver_id"`
	ActorType        string `json:"actor_type"`
	ApproverRole     string `json:"approver_role"`
	SupplierPerms    []string
	Decision         string `json:"decision"`
	Comment          string `json:"comment,omitempty"`
	ExpectedHash     string `json:"payload_hash,omitempty"`
	MFAVerified      bool   `json:"mfa_verified,omitempty"`
	TOTPCode         string `json:"totp_code,omitempty"`
	BreakGlass       bool   `json:"break_glass,omitempty"`
	BreakGlassReason string `json:"break_glass_reason,omitempty"`
	RoutePath        string `json:"-"`
	TraceID          string `json:"-"`
	IP               string `json:"-"`
	DeviceID         string `json:"-"`
}

// DecideResult returns the updated request and recorded decision.
type DecideResult struct {
	Request  ApprovalRequest  `json:"request"`
	Decision ApprovalDecision `json:"decision"`
}

// RequestDetail bundles an ApprovalRequest with its decisions and pinned policy snapshot.
type RequestDetail struct {
	Request   ApprovalRequest    `json:"request"`
	Decisions []ApprovalDecision `json:"decisions"`
	Policy    *Policy            `json:"policy,omitempty"`
}

// InboxFilter filters ApprovalRequests for GET /v1/approvals/inbox.
type InboxFilter struct {
	TenantID   string
	Status     string
	ObjectType string
	Action     string
	BreakGlass *bool
	Limit      int
}

// CreateDelegationInput creates a time-bounded approval delegation.
type CreateDelegationInput struct {
	TenantID     string    `json:"tenant_id,omitempty"`
	DelegatorID  string    `json:"delegator_id"`
	DelegateID   string    `json:"delegate_id"`
	RoleScope    string    `json:"role_scope,omitempty"`
	StartsAt     time.Time `json:"starts_at,omitempty"`
	EndsAt       time.Time `json:"ends_at"`
	Reason       string    `json:"reason"`
	ActorType    string    `json:"-"`
	DelegateType string    `json:"-"`
}

// SoDCheckInput carries maker-checker metadata to iam.Service.CheckFourEyes.
type SoDCheckInput struct {
	TenantID    string
	ActorID     string
	ActorType   string
	Role        string
	Action      string
	MakerAction string
	ObjectType  string
	ObjectID    string
	MakerID     string
	RoutePath   string
	TraceID     string
	IP          string
	DeviceID    string
	IsApproval  bool
}

// CanonicalizePayload converts a map payload into deterministic canonical JSON and its SHA-256 hex digest.
// Rejects any floating-point numbers anywhere in the nested payload.
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

// CanonicalizeJSONBytes normalizes raw JSON bytes into sorted-key canonical JSON and SHA-256 hex.
// Rejects malformed JSON and any floating-point numbers.
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
		// Re-encode structs/slices cleanly via JSON UseNumber check so nested structs are also validated.
		raw, err := json.Marshal(tv)
		if err != nil {
			return fmt.Errorf("%w: unsupported payload value %T: %v", ErrInvalidInput, tv, err)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return fmt.Errorf("%w: decode payload %T: %v", ErrInvalidInput, tv, err)
		}
		return validateNoFloats(parsed)
	}
}
