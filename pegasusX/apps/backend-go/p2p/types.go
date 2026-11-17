// Package p2p implements ATOMOS §6.6 Procure-to-Pay (Purchase Orders, Goods Receipts,
// Vendor Invoices, 3-Way Match with GL clearing), shipped Uzbekistan process templates,
// process instances, and live Spanner-backed Process Mining.
package p2p

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// Process template domains.
const (
	DomainO2C     = "O2C"
	DomainP2P     = "P2P"
	DomainReturns = "RETURNS"
)

// Built-in shipped Uzbekistan process template codes.
const (
	TemplateO2CDirectUZ       = "O2C-DIRECT-UZ"
	TemplateO2CPreorderUZ     = "O2C-PREORDER-UZ"
	TemplateP2PStdUZ          = "P2P-STD-UZ"
	TemplateReturnsSupplierUZ = "RETURNS-SUPPLIER-UZ"
)

// Template and instance lifecycle statuses.
const (
	TemplateStatusActive   = "ACTIVE"
	TemplateStatusArchived = "ARCHIVED"

	InstanceStatusActive    = "ACTIVE"
	InstanceStatusCompleted = "COMPLETED"
	InstanceStatusCancelled = "CANCELLED"
)

// Purchase Order, Goods Receipt, and Vendor Invoice statuses.
const (
	POStatusPendingApproval = "PENDING_APPROVAL"
	POStatusApproved        = "APPROVED"
	POStatusPartialReceived = "PARTIAL_RECEIVED"
	POStatusReceived        = "RECEIVED"
	POStatusClosed          = "CLOSED"
	POStatusCancelled       = "CANCELLED"

	ReceiptStatusPosted = "POSTED"

	MatchStatusPending     = "PENDING"
	MatchStatusMatched3Way = "MATCHED_3WAY"
	MatchStatusDiscrepancy = "DISCREPANCY"
)

// Canonical GL Chart of Accounts codes matching gl/coa.go.
const (
	AccountInventory     = "INVENTORY"
	AccountGRNI          = "GRNI"
	AccountVatInput      = "TAX:VAT:INPUT"
	AccountVendorPayable = "AP:VENDOR"
	AccountPriceVariance = "COGS"

	DirectionDebit  = "DEBIT"
	DirectionCredit = "CREDIT"
)

// Canonical event types and outbox aggregate names for §6.6 P2P.
const (
	EventP2PPOCreated             = events.EventP2PPOCreated
	EventP2PReceiptPosted         = events.EventP2PReceiptPosted
	EventP2PThreeWayMatchComplete = events.EventP2PThreeWayMatchCompleted

	AggregatePurchaseOrder   = events.AggregatePurchaseOrder
	AggregateGoodsReceipt    = events.AggregateGoodsReceipt
	AggregateVendorInvoice   = events.AggregateVendorInvoice
	AggregateProcessTemplate = events.AggregateProcessTemplate
	AggregateProcessInstance = events.AggregateProcessInstance
)

// Default tolerances and bounds (pure integers, zero floats).
const (
	DefaultVatRateBps     int64 = 1200
	MaxToleranceBps       int64 = 5000
	DefaultListLimit      int   = 50
	MaxListLimit          int   = 200
	MiningQueryRowCap     int   = 5000
	DefaultMiningSLASec   int64 = 86400
	DefaultApproveAction        = "approve"
	DefaultOverrideAction       = "override_discrepancy"
)

// Domain sentinel errors mapped cleanly to HTTP status codes.
var (
	ErrInvalidInput        = errors.New("p2p: invalid input")
	ErrNotFound            = errors.New("p2p: record not found")
	ErrConflict            = errors.New("p2p: conflict")
	ErrRepoUnavailable     = errors.New("p2p: repository unavailable")
	ErrSoDViolation        = errors.New("p2p: segregation of duties violation (creator cannot verify or approve own document)")
	ErrForbidden           = errors.New("p2p: forbidden")
	ErrApprovalRequired    = errors.New("p2p: approval required")
	ErrApprovalFailed      = errors.New("p2p: approval verification failed")
	ErrQtyExceeded         = errors.New("p2p: received quantity exceeds ordered quantity")
	ErrPONotApproved       = errors.New("p2p: purchase order is pending approval or cancelled")
	ErrGLPosterUnavailable = errors.New("p2p: gl poster unavailable")
)

// GLLine represents a single balanced debit or credit leg emitted by P2P.
type GLLine struct {
	AccountCode string `json:"account_code"`
	Direction   string `json:"direction"`
	AmountMinor int64  `json:"amount_minor"`
	PartyType   string `json:"party_type,omitempty"`
	PartyID     string `json:"party_id,omitempty"`
	WarehouseID string `json:"warehouse_id,omitempty"`
	ProductID   string `json:"product_id,omitempty"`
	Description string `json:"description,omitempty"`
}

// GLPostInput is the clean package-local payload sent to GLPoster.
type GLPostInput struct {
	TenantID       string    `json:"tenant_id"`
	EntityID       string    `json:"entity_id,omitempty"`
	PeriodID       string    `json:"period_id,omitempty"`
	SourceType     string    `json:"source_type"`
	SourceID       string    `json:"source_id"`
	Memo           string    `json:"memo,omitempty"`
	Currency       string    `json:"currency"`
	IdempotencyKey string    `json:"idempotency_key"`
	PostedBy       string    `json:"posted_by,omitempty"`
	OccurredAt     time.Time `json:"occurred_at,omitempty"`
	Lines          []GLLine  `json:"lines"`
}

// GLPostResult reports the committed GL entry ID and period.
type GLPostResult struct {
	EntryID       string `json:"entry_id"`
	PeriodID      string `json:"period_id,omitempty"`
	AlreadyPosted bool   `json:"already_posted,omitempty"`
}

// GLPoster decouples p2p from gl so bootstrap/wave4a.go wires *gl.Service cleanly.
type GLPoster interface {
	PostJournal(ctx context.Context, in GLPostInput) (GLPostResult, error)
}

// TransactionalGLPoster is optionally implemented by GLPoster adapters that can
// buffer balanced GL mutations directly inside a Spanner ReadWriteTransaction.
type TransactionalGLPoster interface {
	GLPoster
	PostJournalInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, outboxBuf outbox.TxnBuffer, in GLPostInput) (GLPostResult, error)
}

// ApprovalCheckInput carries PO / override details to an optional ApprovalChecker.
type ApprovalCheckInput struct {
	TenantID       string         `json:"tenant_id"`
	ObjectType     string         `json:"object_type"`
	ObjectID       string         `json:"object_id"`
	Action         string         `json:"action"`
	AmountMinor    int64          `json:"amount_minor"`
	RequestedBy    string         `json:"requested_by"`
	ActorType      string         `json:"actor_type"`
	ApproverRole   string         `json:"approver_role,omitempty"`
	ApprovalID     string         `json:"approval_id,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	Context        map[string]any `json:"context,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	ForceCreate    bool           `json:"force_create,omitempty"`
}

// ApprovalCheckResult reports CEL policy evaluation or approval request verification.
type ApprovalCheckResult struct {
	Required          bool     `json:"required"`
	Approved          bool     `json:"approved"`
	ApprovalRequestID string   `json:"approval_request_id,omitempty"`
	PolicyID          string   `json:"policy_id,omitempty"`
	PolicyVersion     int64    `json:"policy_version,omitempty"`
	PayloadHash       string   `json:"payload_hash,omitempty"`
	Status            string   `json:"status,omitempty"`
	ApproverIDs       []string `json:"approver_ids,omitempty"`
}

// ApprovalChecker decouples p2p from approvals without cross-package cycles.
type ApprovalChecker interface {
	CheckOrSubmitApproval(ctx context.Context, in ApprovalCheckInput) (ApprovalCheckResult, error)
}

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

// ProcessStep defines one ordered step in a ProcessTemplate (all timing in int64 seconds).
type ProcessStep struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Sequence     int64  `json:"sequence"`
	SLASeconds   int64  `json:"sla_seconds"`
	RequiredRole string `json:"required_role,omitempty"`
}

// ProcessTemplate models a persisted or shipped ProcessTemplates row.
type ProcessTemplate struct {
	TenantID   string        `json:"tenant_id"`
	TemplateID string        `json:"template_id"`
	Code       string        `json:"code"`
	Name       string        `json:"name"`
	Domain     string        `json:"domain"`
	Steps      []ProcessStep `json:"steps"`
	IsDefault  bool          `json:"is_default"`
	BuiltIn    bool          `json:"built_in,omitempty"`
	Status     string        `json:"status"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// ProcessTransition records one chronological step transition on a ProcessInstance.
type ProcessTransition struct {
	FromStep   string    `json:"from_step"`
	ToStep     string    `json:"to_step"`
	ActorID    string    `json:"actor_id,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

// ProcessInstance models one persisted ProcessInstances row.
type ProcessInstance struct {
	TenantID     string              `json:"tenant_id"`
	InstanceID   string              `json:"instance_id"`
	TemplateID   string              `json:"template_id"`
	TemplateCode string              `json:"template_code,omitempty"`
	EntityID     string              `json:"entity_id,omitempty"`
	ObjectType   string              `json:"object_type"`
	ObjectID     string              `json:"object_id"`
	CurrentStep  string              `json:"current_step"`
	Status       string              `json:"status"`
	Transitions  []ProcessTransition `json:"transitions"`
	StartedAt    time.Time           `json:"started_at"`
	CompletedAt  *time.Time          `json:"completed_at,omitempty"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// PurchaseOrderLine models one PurchaseOrderLines row (all quantities/amounts/VAT in int64).
type PurchaseOrderLine struct {
	TenantID       string `json:"tenant_id"`
	PoID           string `json:"po_id"`
	LineNum        int64  `json:"line_num"`
	Sku            string `json:"sku"`
	ProductName    string `json:"product_name,omitempty"`
	QtyOrdered     int64  `json:"qty_ordered"`
	QtyReceived    int64  `json:"qty_received"`
	UnitCostMinor  int64  `json:"unit_cost_minor"`
	VatBps         int64  `json:"vat_bps"`
	LineTotalMinor int64  `json:"line_total_minor"`
}

// PurchaseOrder models one PurchaseOrders header plus its ordered lines.
type PurchaseOrder struct {
	TenantID          string              `json:"tenant_id"`
	PoID              string              `json:"po_id"`
	EntityID          string              `json:"entity_id,omitempty"`
	VendorID          string              `json:"vendor_id"`
	VendorName        string              `json:"vendor_name,omitempty"`
	VendorTin         string              `json:"vendor_tin,omitempty"`
	Currency          string              `json:"currency"`
	SubtotalMinor     int64               `json:"subtotal_minor"`
	VatMinor          int64               `json:"vat_minor"`
	TotalMinor        int64               `json:"total_minor"`
	Status            string              `json:"status"`
	ApprovalRequestID string              `json:"approval_request_id,omitempty"`
	IdempotencyKey    string              `json:"idempotency_key,omitempty"`
	ExpectedAt        *time.Time          `json:"expected_at,omitempty"`
	CreatedBy         string              `json:"created_by"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	Lines             []PurchaseOrderLine `json:"lines"`
}

// GoodsReceiptLine models one received line serialized into GoodsReceipts.LinesJson.
type GoodsReceiptLine struct {
	LineNum       int64  `json:"line_num"`
	Sku           string `json:"sku"`
	QtyReceived   int64  `json:"qty_received"`
	UnitCostMinor int64  `json:"unit_cost_minor"`
	VatBps        int64  `json:"vat_bps"`
	LineNetMinor  int64  `json:"line_net_minor"`
}

// GoodsReceipt models one persisted GoodsReceipts row.
type GoodsReceipt struct {
	TenantID           string             `json:"tenant_id"`
	ReceiptID          string             `json:"receipt_id"`
	PoID               string             `json:"po_id"`
	EntityID           string             `json:"entity_id,omitempty"`
	WarehouseID        string             `json:"warehouse_id"`
	Status             string             `json:"status"`
	Lines              []GoodsReceiptLine `json:"lines"`
	TotalReceivedMinor int64              `json:"total_received_minor"`
	JournalEntryID     string             `json:"journal_entry_id,omitempty"`
	IdempotencyKey     string             `json:"idempotency_key,omitempty"`
	ReceivedBy         string             `json:"received_by"`
	ReceivedAt         time.Time          `json:"received_at"`
	CreatedAt          time.Time          `json:"created_at"`
}

// VendorInvoice models one persisted VendorInvoices row.
type VendorInvoice struct {
	TenantID        string    `json:"tenant_id"`
	VendorInvoiceID string    `json:"vendor_invoice_id"`
	PoID            string    `json:"po_id"`
	ReceiptID       string    `json:"receipt_id,omitempty"`
	EsfDocID        string    `json:"esf_doc_id,omitempty"`
	EntityID        string    `json:"entity_id,omitempty"`
	VendorID        string    `json:"vendor_id"`
	Currency        string    `json:"currency"`
	SubtotalMinor   int64     `json:"subtotal_minor"`
	VatMinor        int64     `json:"vat_minor"`
	TotalMinor      int64     `json:"total_minor"`
	MatchStatus     string    `json:"match_status"`
	VarianceMinor   int64     `json:"variance_minor"`
	VarianceReason  string    `json:"variance_reason,omitempty"`
	JournalEntryID  string    `json:"journal_entry_id,omitempty"`
	IdempotencyKey  string    `json:"idempotency_key,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// InboundESFDoc carries the verified EsfDocuments subset used during 3-way match.
type InboundESFDoc struct {
	EntityID        string `json:"entity_id"`
	EsfID           string `json:"esf_id"`
	Direction       string `json:"direction"`
	PoID            string `json:"po_id,omitempty"`
	CounterpartyTin string `json:"counterparty_tin"`
	TotalMinor      int64  `json:"total_minor"`
	VatMinor        int64  `json:"vat_minor"`
	VatRateBps      int64  `json:"vat_rate_bps"`
	Status          string `json:"status"`
}

// ThreeWayMatchResult summarizes the outcome of POST /v1/p2p/three-way-match.
type ThreeWayMatchResult struct {
	VendorInvoice   VendorInvoice    `json:"vendor_invoice"`
	Matched         bool             `json:"matched"`
	MatchStatus     string           `json:"match_status"`
	VarianceMinor   int64            `json:"variance_minor"`
	VarianceReason  string           `json:"variance_reason,omitempty"`
	JournalEntryID  string           `json:"journal_entry_id,omitempty"`
	VerifiedBy      string           `json:"verified_by,omitempty"`
	ProcessInstance *ProcessInstance `json:"process_instance,omitempty"`
}

// MiningEdge describes one observed transition edge computed from real DB rows (zero floats).
type MiningEdge struct {
	FromStep          string `json:"from_step"`
	ToStep            string `json:"to_step"`
	Domain            string `json:"domain"`
	CaseCount         int64  `json:"case_count"`
	TransitionCount   int64  `json:"transition_count"`
	AvgDurationSec    int64  `json:"avg_duration_sec"`
	MedianDurationSec int64  `json:"median_duration_sec"`
	P95DurationSec    int64  `json:"p95_duration_sec"`
	SLATargetSec      int64  `json:"sla_target_sec"`
	SLABreachCount    int64  `json:"sla_breach_count"`
}

// MiningBottleneck identifies the highest-p95/breach stage (nil when empty).
type MiningBottleneck struct {
	FromStep          string `json:"from_step"`
	ToStep            string `json:"to_step"`
	Domain            string `json:"domain"`
	CaseCount         int64  `json:"case_count"`
	TransitionCount   int64  `json:"transition_count"`
	MedianDurationSec int64  `json:"median_duration_sec"`
	P95DurationSec    int64  `json:"p95_duration_sec"`
	SLATargetSec      int64  `json:"sla_target_sec"`
	SLABreachCount    int64  `json:"sla_breach_count"`
}

// ProcessMiningReport is returned by GET /v1/processes/mining.
type ProcessMiningReport struct {
	TenantID         string            `json:"tenant_id"`
	EntityID         string            `json:"entity_id,omitempty"`
	Domain           string            `json:"domain,omitempty"`
	TotalCases       int64             `json:"total_cases"`
	TotalTransitions int64             `json:"total_transitions"`
	SLABreachCount   int64             `json:"sla_breach_count"`
	BottleneckStage  *MiningBottleneck `json:"bottleneck_stage"`
	Empty            bool              `json:"empty"`
	Edges            []MiningEdge      `json:"edges"`
}

// CreateTemplateInput defines fields to create or upsert a tenant ProcessTemplate.
type CreateTemplateInput struct {
	TenantID   string        `json:"tenant_id,omitempty"`
	TemplateID string        `json:"template_id,omitempty"`
	Code       string        `json:"code"`
	Name       string        `json:"name"`
	Domain     string        `json:"domain"`
	Steps      []ProcessStep `json:"steps"`
	IsDefault  bool          `json:"is_default,omitempty"`
	Status     string        `json:"status,omitempty"`
	ActorID    string        `json:"-"`
	ActorType  string        `json:"-"`
	ActorRole  string        `json:"-"`
}

// StartOrAdvanceInstanceInput starts a ProcessInstance or advances an existing one.
type StartOrAdvanceInstanceInput struct {
	TenantID     string `json:"tenant_id,omitempty"`
	InstanceID   string `json:"instance_id,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	TemplateCode string `json:"template_code,omitempty"`
	EntityID     string `json:"entity_id,omitempty"`
	ObjectType   string `json:"object_type"`
	ObjectID     string `json:"object_id"`
	NextStep     string `json:"next_step,omitempty"`
	CurrentStep  string `json:"current_step,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Complete     bool   `json:"complete,omitempty"`
	ActorID      string `json:"-"`
	ActorType    string `json:"-"`
	ActorRole    string `json:"-"`
}

// PurchaseOrderLineInput defines one requested PO line.
type PurchaseOrderLineInput struct {
	Sku           string `json:"sku"`
	ProductName   string `json:"product_name,omitempty"`
	QtyOrdered    int64  `json:"qty_ordered"`
	UnitCostMinor int64  `json:"unit_cost_minor"`
	VatBps        int64  `json:"vat_bps"`
}

// CreatePurchaseOrderInput is the command payload for POST /v1/p2p/purchase-orders.
type CreatePurchaseOrderInput struct {
	TenantID       string                   `json:"tenant_id,omitempty"`
	PoID           string                   `json:"po_id,omitempty"`
	EntityID       string                   `json:"entity_id,omitempty"`
	VendorID       string                   `json:"vendor_id"`
	VendorName     string                   `json:"vendor_name,omitempty"`
	VendorTin      string                   `json:"vendor_tin,omitempty"`
	Currency       string                   `json:"currency,omitempty"`
	ExpectedAt     *time.Time               `json:"expected_at,omitempty"`
	ApprovalID     string                   `json:"approval_id,omitempty"`
	IdempotencyKey string                   `json:"idempotency_key,omitempty"`
	TemplateCode   string                   `json:"template_code,omitempty"`
	Lines          []PurchaseOrderLineInput `json:"lines"`
	CreatedBy      string                   `json:"-"`
	ActorType      string                   `json:"-"`
	ActorRole      string                   `json:"-"`
}

// CreatePurchaseOrderResult wraps the created PO plus optional approval challenge fields.
type CreatePurchaseOrderResult struct {
	PurchaseOrder     PurchaseOrder `json:"purchase_order"`
	ApprovalRequired  bool          `json:"approval_required,omitempty"`
	ApprovalRequestID string        `json:"approval_request_id,omitempty"`
}

// ApprovePurchaseOrderInput approves a PENDING_APPROVAL purchase order enforcing SoD.
type ApprovePurchaseOrderInput struct {
	TenantID   string `json:"tenant_id,omitempty"`
	PoID       string `json:"po_id"`
	ApprovalID string `json:"approval_id,omitempty"`
	ActorID    string `json:"-"`
	ActorType  string `json:"-"`
	ActorRole  string `json:"-"`
}

// GoodsReceiptLineInput specifies one line quantity to receive against a PO.
type GoodsReceiptLineInput struct {
	LineNum     int64  `json:"line_num,omitempty"`
	Sku         string `json:"sku"`
	QtyReceived int64  `json:"qty_received"`
}

// PostGoodsReceiptInput is the command payload for POST /v1/p2p/receipts.
type PostGoodsReceiptInput struct {
	TenantID       string                  `json:"tenant_id,omitempty"`
	ReceiptID      string                  `json:"receipt_id,omitempty"`
	PoID           string                  `json:"po_id"`
	EntityID       string                  `json:"entity_id,omitempty"`
	WarehouseID    string                  `json:"warehouse_id"`
	PeriodID       string                  `json:"period_id,omitempty"`
	IdempotencyKey string                  `json:"idempotency_key,omitempty"`
	ReceivedAt     *time.Time              `json:"received_at,omitempty"`
	Lines          []GoodsReceiptLineInput `json:"lines"`
	ReceivedBy     string                  `json:"-"`
	ActorType      string                  `json:"-"`
	ActorRole      string                  `json:"-"`
}

// PostGoodsReceiptResult bundles the posted receipt, updated PO, and GL entry ID.
type PostGoodsReceiptResult struct {
	Receipt        GoodsReceipt  `json:"receipt"`
	PurchaseOrder  PurchaseOrder `json:"purchase_order"`
	JournalEntryID string        `json:"journal_entry_id"`
}

// CreateVendorInvoiceInput is the command payload for POST /v1/p2p/vendor-invoices.
type CreateVendorInvoiceInput struct {
	TenantID        string `json:"tenant_id,omitempty"`
	VendorInvoiceID string `json:"vendor_invoice_id,omitempty"`
	PoID            string `json:"po_id"`
	ReceiptID       string `json:"receipt_id,omitempty"`
	EsfDocID        string `json:"esf_doc_id,omitempty"`
	EntityID        string `json:"entity_id,omitempty"`
	VendorID        string `json:"vendor_id,omitempty"`
	Currency        string `json:"currency,omitempty"`
	SubtotalMinor   int64  `json:"subtotal_minor"`
	VatMinor        int64  `json:"vat_minor"`
	TotalMinor      int64  `json:"total_minor"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
	CreatedBy       string `json:"-"`
	ActorType       string `json:"-"`
	ActorRole       string `json:"-"`
}

// ThreeWayMatchInput is the command payload for POST /v1/p2p/three-way-match.
type ThreeWayMatchInput struct {
	TenantID            string `json:"tenant_id,omitempty"`
	VendorInvoiceID     string `json:"vendor_invoice_id"`
	PoID                string `json:"po_id,omitempty"`
	ReceiptID           string `json:"receipt_id,omitempty"`
	EsfDocID            string `json:"esf_doc_id,omitempty"`
	PeriodID            string `json:"period_id,omitempty"`
	ToleranceBps        int64  `json:"tolerance_bps,omitempty"`
	ToleranceMinor      int64  `json:"tolerance_minor,omitempty"`
	OverrideDiscrepancy bool   `json:"override_discrepancy,omitempty"`
	ApprovalID          string `json:"approval_id,omitempty"`
	VerifiedBy          string `json:"verified_by,omitempty"`
	ActorType           string `json:"-"`
	ActorRole           string `json:"-"`
}

// ListFilter filters P2P and process queries.
type ListFilter struct {
	TenantID    string
	EntityID    string
	VendorID    string
	PoID        string
	Status      string
	MatchStatus string
	ObjectType  string
	ObjectID    string
	Domain      string
	Limit       int
}

// MiningFilter constrains GET /v1/processes/mining queries.
type MiningFilter struct {
	TenantID string
	EntityID string
	Domain   string
	From     *time.Time
	To       *time.Time
}

// MiningSample is one raw chronological event pair used to compute process mining metrics.
type MiningSample struct {
	CaseID      string
	Domain      string
	FromStep    string
	ToStep      string
	DurationSec int64
	SLATarget   int64
}

func normalizeDomain(d string) string {
	switch strings.ToUpper(strings.TrimSpace(d)) {
	case DomainO2C:
		return DomainO2C
	case DomainP2P:
		return DomainP2P
	case DomainReturns:
		return DomainReturns
	default:
		return strings.ToUpper(strings.TrimSpace(d))
	}
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

func coalesce(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
