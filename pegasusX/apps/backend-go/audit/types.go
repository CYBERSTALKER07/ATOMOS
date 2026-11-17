// Package audit provides ATOMOS M0.4 cryptographically hash-chained immutable
// audit events, WORM daily root anchors, legal holds, PII role masking, and
// universal HTTP mutating route audit middleware.
package audit

import (
	"errors"
	"strings"
	"time"
)

// ActorType enumerates first-class actor provenance values for AuditEvents.
type ActorType string

const (
	ActorTypeHuman     ActorType = "human"
	ActorTypeAgent     ActorType = "agent"
	ActorTypeConnector ActorType = "connector"
	ActorTypeSystem    ActorType = "system"
)

// Legal hold lifecycle statuses.
const (
	LegalHoldStatusActive   = "ACTIVE"
	LegalHoldStatusReleased = "RELEASED"
)

// Domain errors.
var (
	ErrInvalidAuditInput  = errors.New("audit: invalid audit input")
	ErrInvalidDayBucket   = errors.New("audit: day_bucket must be formatted YYYY-MM-DD")
	ErrAnchorWormRequired = errors.New("audit: WORM anchor storage required in production")
	ErrHoldNotFound       = errors.New("audit: legal hold not found")
	ErrRepoUnavailable    = errors.New("audit: repository unavailable")
	ErrChainBroken        = errors.New("audit: hash chain verification failed")
)

// NormalizeActorType normalizes raw provenance strings into one of the four canonical ActorType values.
func NormalizeActorType(raw string) ActorType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(ActorTypeHuman):
		return ActorTypeHuman
	case string(ActorTypeAgent):
		return ActorTypeAgent
	case string(ActorTypeConnector):
		return ActorTypeConnector
	case string(ActorTypeSystem):
		return ActorTypeSystem
	default:
		return ActorTypeSystem
	}
}

// IsValidActorType reports whether t is a canonical ActorType.
func IsValidActorType(t ActorType) bool {
	switch t {
	case ActorTypeHuman, ActorTypeAgent, ActorTypeConnector, ActorTypeSystem:
		return true
	default:
		return false
	}
}

// EventInput represents a caller request to append one immutable AuditEvents record.
type EventInput struct {
	EventID         string     `json:"event_id,omitempty"`
	TenantID        string     `json:"tenant_id"`
	DayBucket       string     `json:"day_bucket,omitempty"`
	ActorID         string     `json:"actor_id"`
	ActorType       ActorType  `json:"actor_type"`
	Role            string     `json:"role,omitempty"`
	Action          string     `json:"action"`
	ObjectType      string     `json:"object_type,omitempty"`
	ObjectID        string     `json:"object_id,omitempty"`
	BeforeJSON      string     `json:"before_json,omitempty"`
	AfterJSON       string     `json:"after_json,omitempty"`
	DiffGcsURI      string     `json:"diff_gcs_uri,omitempty"`
	ReasonCode      string     `json:"reason_code,omitempty"`
	ApprovalID      string     `json:"approval_id,omitempty"`
	TraceID         string     `json:"trace_id,omitempty"`
	IP              string     `json:"ip,omitempty"`
	DeviceID        string     `json:"device_id,omitempty"`
	LegalHold       bool       `json:"legal_hold,omitempty"`
	LegalHoldReason string     `json:"legal_hold_reason,omitempty"`
	RetentionUntil  *time.Time `json:"retention_until,omitempty"`
}

// AuditEvent is one immutable row in AuditEvents (or fallback legacy AuditLog projection).
type AuditEvent struct {
	TenantID        string     `json:"tenant_id"`
	DayBucket       string     `json:"day_bucket"`
	EventID         string     `json:"event_id"`
	SeqNum          int64      `json:"seq_num"`
	ActorID         string     `json:"actor_id"`
	ActorType       ActorType  `json:"actor_type"`
	Role            string     `json:"role,omitempty"`
	Action          string     `json:"action"`
	ObjectType      string     `json:"object_type,omitempty"`
	ObjectID        string     `json:"object_id,omitempty"`
	BeforeJSON      string     `json:"before_json,omitempty"`
	AfterJSON       string     `json:"after_json,omitempty"`
	DiffGcsURI      string     `json:"diff_gcs_uri,omitempty"`
	ReasonCode      string     `json:"reason_code,omitempty"`
	ApprovalID      string     `json:"approval_id,omitempty"`
	TraceID         string     `json:"trace_id,omitempty"`
	IP              string     `json:"ip,omitempty"`
	DeviceID        string     `json:"device_id,omitempty"`
	PrevHash        string     `json:"prev_hash"`
	EntryHash       string     `json:"entry_hash"`
	LegalHold       bool       `json:"legal_hold"`
	LegalHoldReason string     `json:"legal_hold_reason,omitempty"`
	RetentionUntil  *time.Time `json:"retention_until,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	Source          string     `json:"source,omitempty"`
	PIIMasked       bool       `json:"pii_masked,omitempty"`
}

// Anchor represents one daily hash-chain root record persisted in AuditChainAnchors.
type Anchor struct {
	TenantID    string    `json:"tenant_id"`
	DayBucket   string    `json:"day_bucket"`
	HeadEventID string    `json:"head_event_id"`
	EventCount  int64     `json:"event_count"`
	RootHash    string    `json:"root_hash"`
	WormURI     string    `json:"worm_uri,omitempty"`
	SignerKeyID string    `json:"signer_key_id,omitempty"`
	AnchoredBy  string    `json:"anchored_by"`
	AnchoredAt  time.Time `json:"anchored_at"`
}

// LegalHoldInput defines parameters to place an audit legal hold on an object.
type LegalHoldInput struct {
	TenantID   string     `json:"tenant_id"`
	ObjectType string     `json:"object_type"`
	ObjectID   string     `json:"object_id"`
	Reason     string     `json:"reason"`
	CreatedBy  string     `json:"created_by"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// LegalHold represents one row in AuditLegalHolds.
type LegalHold struct {
	TenantID   string     `json:"tenant_id"`
	HoldID     string     `json:"hold_id"`
	ObjectType string     `json:"object_type"`
	ObjectID   string     `json:"object_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	CreatedBy  string     `json:"created_by"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ChainVerifyResult reports cryptographic hash-chain verification for (TenantId, DayBucket).
type ChainVerifyResult struct {
	TenantID           string `json:"tenant_id"`
	DayBucket          string `json:"day_bucket"`
	Valid              bool   `json:"valid"`
	EventCount         int64  `json:"event_count"`
	HeadEventID        string `json:"head_event_id,omitempty"`
	RootHash           string `json:"root_hash"`
	AnchoredRootHash   string `json:"anchored_root_hash,omitempty"`
	AnchoredEventCount int64  `json:"anchored_event_count,omitempty"`
	AnchorMatched      *bool  `json:"anchor_matched,omitempty"`
	BrokenEventID      string `json:"broken_event_id,omitempty"`
	FailureReason      string `json:"failure_reason,omitempty"`
}

// Filter defines query criteria for ListEvents.
type Filter struct {
	TenantID      string     `json:"tenant_id,omitempty"`
	ObjectType    string     `json:"object_type,omitempty"`
	ObjectID      string     `json:"object_id,omitempty"`
	ActorID       string     `json:"actor_id,omitempty"`
	ActorType     ActorType  `json:"actor_type,omitempty"`
	Action        string     `json:"action,omitempty"`
	DayBucket     string     `json:"day_bucket,omitempty"`
	From          *time.Time `json:"from,omitempty"`
	To            *time.Time `json:"to,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	IncludeLegacy bool       `json:"include_legacy,omitempty"`
	ViewerRole    string     `json:"viewer_role,omitempty"`
}

// MutationContext allows HTTP handlers to enrich the universal AuditMiddleware event
// with domain-specific before/after snapshots, object coordinates, reason codes, or approval IDs.
type MutationContext struct {
	TenantID   string `json:"tenant_id,omitempty"`
	Action     string `json:"action,omitempty"`
	ObjectType string `json:"object_type,omitempty"`
	ObjectID   string `json:"object_id,omitempty"`
	BeforeJSON string `json:"before_json,omitempty"`
	AfterJSON  string `json:"after_json,omitempty"`
	ReasonCode string `json:"reason_code,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
}
