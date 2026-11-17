// Package periods implements ATOMOS M1.1 Period Close & Hard-Close Locks:
// timezone-aware month cutoffs, operational rollover with OriginalPeriodId,
// Spanner-verified close checklists, immutable close snapshots (JSON+CSV+SHA256),
// and dual-approver period reopen workflows.
package periods

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

// Accounting period lifecycle statuses persisted in Periods.Status.
const (
	StatusOpen       = "OPEN"
	StatusSoftClosed = "SOFT_CLOSED"
	StatusClosed     = "CLOSED"
	StatusReopened   = "REOPENED"
)

var (
	ErrInvalidInput            = errors.New("periods: invalid input")
	ErrPeriodNotFound          = errors.New("periods: period not found")
	ErrPeriodAlreadyExists     = errors.New("periods: period already exists")
	ErrInvalidStatusTransition = errors.New("periods: invalid status transition")
	ErrChecklistFailed         = errors.New("periods: close checklist has unresolved blockers")
	ErrApprovalRequired        = errors.New("periods: reopen requires two distinct human approvers")
	ErrPayloadMismatch         = errors.New("periods: approval payload hash mismatch")
	ErrSoDViolation            = errors.New("periods: segregation of duties violation")
	ErrSnapshotNotFound        = errors.New("periods: close snapshot not found")
	ErrSnapshotWormRequired    = errors.New("periods: immutable snapshot storage required")
	ErrRepoUnavailable         = errors.New("periods: repository unavailable")
)

// IsPeriodClosedError reports whether err indicates a closed accounting period lock.
func IsPeriodClosedError(err error) bool {
	return errors.Is(err, gl.ErrPeriodClosed)
}

// Period represents one entity-scoped monthly accounting period record.
type Period struct {
	EntityID       string     `json:"entity_id"`
	PeriodID       string     `json:"period_id"`
	Status         string     `json:"status"`
	Timezone       string     `json:"timezone"`
	PeriodStartUTC time.Time  `json:"period_start_utc"`
	PeriodEndUTC   time.Time  `json:"period_end_utc"`
	CutoffLocal    string     `json:"cutoff_local"`
	SoftClosedBy   string     `json:"soft_closed_by,omitempty"`
	SoftClosedAt   *time.Time `json:"soft_closed_at,omitempty"`
	ClosedBy       string     `json:"closed_by,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	ReopenedBy     string     `json:"reopened_by,omitempty"`
	ReopenedAt     *time.Time `json:"reopened_at,omitempty"`
	ReopenReason   string     `json:"reopen_reason,omitempty"`
	SnapshotURI    string     `json:"snapshot_uri,omitempty"`
	SnapshotHash   string     `json:"snapshot_hash,omitempty"`
	ChecklistJSON  string     `json:"checklist_json,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ChecklistItem captures one live Spanner readiness check for closing a period.
type ChecklistItem struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Passed    bool   `json:"passed"`
	Blocking  bool   `json:"blocking"`
	OpenCount int64  `json:"open_count"`
	Detail    string `json:"detail"`
}

// ChecklistReport aggregates all period close readiness checks across live tables.
type ChecklistReport struct {
	EntityID       string          `json:"entity_id"`
	PeriodID       string          `json:"period_id"`
	Timezone       string          `json:"timezone"`
	PeriodStartUTC time.Time       `json:"period_start_utc"`
	PeriodEndUTC   time.Time       `json:"period_end_utc"`
	ReadyToClose   bool            `json:"ready_to_close"`
	BlockingCount  int             `json:"blocking_count"`
	EvaluatedAt    time.Time       `json:"evaluated_at"`
	Items          []ChecklistItem `json:"items"`
}

// SnapshotARSummary summarizes entity Accounts Receivable balances at period close.
type SnapshotARSummary struct {
	TotalInvoices   int64  `json:"total_invoices"`
	OpenInvoices    int64  `json:"open_invoices"`
	OpenAmountMinor int64  `json:"open_amount_minor"`
	OverdueInvoices int64  `json:"overdue_invoices"`
	Currency        string `json:"currency"`
}

// SnapshotValuationLine summarizes remaining FIFO inventory cost layers for one SKU at one warehouse.
type SnapshotValuationLine struct {
	WarehouseID string `json:"warehouse_id"`
	ProductID   string `json:"product_id"`
	Quantity    int64  `json:"quantity"`
	ValueMinor  int64  `json:"value_minor"`
	Currency    string `json:"currency"`
}

// SnapshotValuationSummary summarizes entity inventory valuation at period close.
type SnapshotValuationSummary struct {
	LayerCount      int64                   `json:"layer_count"`
	TotalQuantity   int64                   `json:"total_quantity"`
	TotalValueMinor int64                   `json:"total_value_minor"`
	Currency        string                  `json:"currency"`
	Lines           []SnapshotValuationLine `json:"lines"`
}

// SnapshotVATSummary summarizes GL VAT vs accepted ESF VAT at period close.
type SnapshotVATSummary struct {
	PeriodID            string `json:"period_id"`
	Currency            string `json:"currency"`
	GLOutputVATMinor    int64  `json:"gl_output_vat_minor"`
	GLInputVATMinor     int64  `json:"gl_input_vat_minor"`
	GLNetVATMinor       int64  `json:"gl_net_vat_minor"`
	ESFOutboundVATMinor int64  `json:"esf_outbound_vat_minor"`
	ESFInboundVATMinor  int64  `json:"esf_inbound_vat_minor"`
	ESFNetVATMinor      int64  `json:"esf_net_vat_minor"`
	VarianceMinor       int64  `json:"variance_minor"`
	ESFCount            int64  `json:"esf_count"`
}

// CloseSnapshot is the immutable close snapshot persisted to WORM storage.
type CloseSnapshot struct {
	EntityID         string                   `json:"entity_id"`
	PeriodID         string                   `json:"period_id"`
	Timezone         string                   `json:"timezone"`
	PeriodStartUTC   time.Time                `json:"period_start_utc"`
	PeriodEndUTC     time.Time                `json:"period_end_utc"`
	Currency         string                   `json:"currency"`
	ClosedBy         string                   `json:"closed_by"`
	GeneratedAt      time.Time                `json:"generated_at"`
	SnapshotURI      string                   `json:"snapshot_uri,omitempty"`
	SnapshotHash     string                   `json:"snapshot_hash,omitempty"`
	TrialBalance     gl.TrialBalanceReport    `json:"trial_balance"`
	ARSummary        SnapshotARSummary        `json:"ar_summary"`
	ValuationSummary SnapshotValuationSummary `json:"valuation_summary"`
	VATSummary       SnapshotVATSummary       `json:"vat_summary"`
}

// CloseResult bundles the updated CLOSED period, checklist, and written WORM snapshot.
type CloseResult struct {
	Period    Period          `json:"period"`
	Checklist ChecklistReport `json:"checklist"`
	Snapshot  CloseSnapshot   `json:"snapshot"`
}

// ReopenInput captures a dual-approver period reopen request.
type ReopenInput struct {
	EntityID     string
	PeriodID     string
	Reason       string
	ActorID      string
	ActorType    string
	ActorRole    string
	ApprovalID   string
	CoApproverID string
	RoutePath    string
	TraceID      string
	IP           string
	DeviceID     string
}

// ReopenResult reports either pending dual approval (202) or the completed REOPENED period (200).
type ReopenResult struct {
	PendingApproval   bool    `json:"pending_approval"`
	ApprovalRequestID string  `json:"approval_request_id,omitempty"`
	RequiredApprovers int64   `json:"required_approvers,omitempty"`
	PayloadHash       string  `json:"payload_hash,omitempty"`
	Period            *Period `json:"period,omitempty"`
}

// ReopenApprovalResult is returned by ApprovalBridge when evaluating/verifying period reopen approvals.
type ReopenApprovalResult struct {
	Approved          bool
	ApprovalRequestID string
	RequiredApprovers int64
	PayloadHash       string
	ApproverIDs       []string
}

// ApprovalBridge connects period reopen workflows to M1.2 CEL Approvals without circular imports.
type ApprovalBridge interface {
	EvaluateOrVerifyPeriodReopen(
		ctx context.Context,
		tenantID, periodID, reason, actorID, actorType, actorRole, approvalID string,
		canonPayload map[string]any,
	) (ReopenApprovalResult, error)
}

// AuditRecordInput carries immutable AuditEvents metadata recorded alongside period transitions.
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

// AuditRecorder persists hash-chained AuditEvents inside the period transition transaction when possible.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
	RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev AuditRecordInput) error
}
