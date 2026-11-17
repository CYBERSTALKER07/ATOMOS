package approvals

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

// AuditRecordInput carries hash-chained audit metadata recorded alongside approval transitions.
type AuditRecordInput struct {
	TenantID   string
	ActorID    string
	ActorType  string
	Role       string
	Action     string
	ObjectType string
	ObjectID   string
	BeforeJSON string
	AfterJSON  string
	ReasonCode string
	ApprovalID string
	TraceID    string
	IP         string
	DeviceID   string
}

// AuditRecorder persists immutable AuditEvents (inside the Spanner RW txn when available).
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
	RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev AuditRecordInput) error
}

// SoDChecker enforces four-eyes / auditor / agent invariants and logs SodViolations.
type SoDChecker interface {
	CheckFourEyes(ctx context.Context, in SoDCheckInput) error
}

// MFAVerifier validates optional TOTP codes during step-up approval.
type MFAVerifier interface {
	Verify(ctx context.Context, subject, code string) error
}

// TelegramApproverResolver maps a Telegram chat/user ID to a supplier actor identity.
type TelegramApproverResolver interface {
	ResolveTelegramApprover(ctx context.Context, chatID, username string) (tenantID, actorID, actorRole, actorType string, ok bool, err error)
}

// Repository defines durable storage for approval policies, requests, decisions, and delegations.
type Repository interface {
	CreatePolicy(ctx context.Context, p Policy) error
	UpdatePolicy(ctx context.Context, p Policy) error
	GetPolicy(ctx context.Context, tenantID, policyID string) (Policy, bool, error)
	ListPolicies(ctx context.Context, tenantID, objectType, action string, enabledOnly bool) ([]Policy, error)

	FindRequestByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (ApprovalRequest, bool, error)
	CreateRequest(ctx context.Context, req ApprovalRequest, auditRec AuditRecorder) error
	GetRequest(ctx context.Context, tenantID, requestID string) (ApprovalRequest, []ApprovalDecision, bool, error)
	GetRequestByID(ctx context.Context, requestID string) (ApprovalRequest, []ApprovalDecision, bool, error)
	ListInbox(ctx context.Context, f InboxFilter) ([]ApprovalRequest, error)
	ApplyDecision(
		ctx context.Context,
		tenantID, requestID string,
		mutateFn func(req ApprovalRequest, existingDecisions []ApprovalDecision, policy *Policy) (ApprovalRequest, ApprovalDecision, error),
		auditRec AuditRecorder,
		traceID string,
	) (ApprovalRequest, ApprovalDecision, error)

	CreateDelegation(ctx context.Context, d ApprovalDelegation) error
	GetDelegation(ctx context.Context, tenantID, delegationID string) (ApprovalDelegation, bool, error)
	ListDelegations(ctx context.Context, tenantID, actorID string) ([]ApprovalDelegation, error)
	FindActiveDelegationsForDelegate(ctx context.Context, tenantID, delegateID string, at time.Time) ([]ApprovalDelegation, error)
	SetDelegationEnabled(ctx context.Context, tenantID, delegationID string, enabled bool) (ApprovalDelegation, error)
}
