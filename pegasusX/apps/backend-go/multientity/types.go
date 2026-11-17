// Package multientity implements §6.4 Multi-Entity Legal Entity management,
// Intercompany Transfers, Month-End FX Revaluation (integer BigInt 1e8 scale,
// fail-closed on missing rates), and Multi-Entity Financial Consolidation with
// intercompany eliminations.
package multientity

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

const (
	EntityStatusActive   = "ACTIVE"
	EntityStatusInactive = "INACTIVE"
	TransferStatusPosted = "POSTED"
)

// Domain errors mapped cleanly to HTTP status codes by handlers.
var (
	ErrRepoUnavailable    = errors.New("multientity: repository unavailable")
	ErrInvalidInput       = errors.New("multientity: invalid input")
	ErrEntityNotFound     = errors.New("multientity: legal entity not found")
	ErrEntityExists       = errors.New("multientity: legal entity already exists")
	ErrEntityInactive     = errors.New("multientity: legal entity is not active")
	ErrSameEntityTransfer = errors.New("multientity: intercompany transfer requires distinct from and to entities")
	ErrFxRateMissing      = errors.New("multientity: fx rate missing (fail closed)")
	ErrInvalidCurrency    = errors.New("multientity: invalid 3-letter ISO currency")
	ErrInvalidPeriod      = errors.New("multientity: period must be formatted YYYY-MM")
	ErrPeriodClosed       = errors.New("multientity: accounting period is closed")
	ErrTransferNotFound   = errors.New("multientity: intercompany transfer not found")
)

// LegalEntity represents one legal entity belonging to a supplier/org tenant.
type LegalEntity struct {
	TenantID       string    `json:"tenant_id"`
	EntityID       string    `json:"entity_id"`
	Name           string    `json:"name"`
	TIN            string    `json:"tin,omitempty"`
	CountryCode    string    `json:"country_code"`
	BaseCurrency   string    `json:"base_currency"`
	IsParent       bool      `json:"is_parent"`
	ParentEntityID string    `json:"parent_entity_id,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateEntityInput is the command payload for POST /v1/org/entities.
type CreateEntityInput struct {
	TenantID       string `json:"tenant_id,omitempty"`
	EntityID       string `json:"entity_id,omitempty"`
	Name           string `json:"name"`
	TIN            string `json:"tin,omitempty"`
	CountryCode    string `json:"country_code,omitempty"`
	BaseCurrency   string `json:"base_currency,omitempty"`
	IsParent       bool   `json:"is_parent,omitempty"`
	ParentEntityID string `json:"parent_entity_id,omitempty"`
	ActorID        string `json:"-"`
	ActorType      string `json:"-"`
	Role           string `json:"-"`
	TraceID        string `json:"-"`
}

// PatchEntityInput is the command payload for PATCH /v1/org/entities/{entityID}.
type PatchEntityInput struct {
	TenantID       string  `json:"tenant_id,omitempty"`
	EntityID       string  `json:"entity_id,omitempty"`
	Name           *string `json:"name,omitempty"`
	TIN            *string `json:"tin,omitempty"`
	CountryCode    *string `json:"country_code,omitempty"`
	BaseCurrency   *string `json:"base_currency,omitempty"`
	IsParent       *bool   `json:"is_parent,omitempty"`
	ParentEntityID *string `json:"parent_entity_id,omitempty"`
	Status         *string `json:"status,omitempty"`
	ActorID        string  `json:"-"`
	ActorType      string  `json:"-"`
	Role           string  `json:"-"`
	TraceID        string  `json:"-"`
}

// IntercompanyTransfer records one balanced intercompany posting across two legal entities.
type IntercompanyTransfer struct {
	TenantID        string    `json:"tenant_id"`
	TransferID      string    `json:"transfer_id"`
	FromEntityID    string    `json:"from_entity_id"`
	ToEntityID      string    `json:"to_entity_id"`
	TransferType    string    `json:"transfer_type"`
	AmountMinor     int64     `json:"amount_minor"`
	Currency        string    `json:"currency"`
	PeriodID        string    `json:"period_id"`
	ReferenceType   string    `json:"reference_type,omitempty"`
	ReferenceID     string    `json:"reference_id,omitempty"`
	DebitJournalID  string    `json:"debit_journal_id"`
	CreditJournalID string    `json:"credit_journal_id"`
	Status          string    `json:"status"`
	IdempotencyKey  string    `json:"idempotency_key,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateTransferInput is the command payload for POST /v1/finance/intercompany-transfers.
type CreateTransferInput struct {
	TenantID       string `json:"tenant_id,omitempty"`
	FromEntityID   string `json:"from_entity_id"`
	ToEntityID     string `json:"to_entity_id"`
	TransferType   string `json:"transfer_type"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency,omitempty"`
	PeriodID       string `json:"period_id,omitempty"`
	ReferenceType  string `json:"reference_type,omitempty"`
	ReferenceID    string `json:"reference_id,omitempty"`
	OffsetAccount  string `json:"offset_account,omitempty"`
	Memo           string `json:"memo,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	CreatedBy      string `json:"-"`
	ActorType      string `json:"-"`
	Role           string `json:"-"`
	TraceID        string `json:"-"`
}

// CreateTransferResult wraps the persisted transfer and idempotency replay flag.
type CreateTransferResult struct {
	Transfer      IntercompanyTransfer `json:"transfer"`
	AlreadyPosted bool                 `json:"already_posted"`
}

// FxRevaluationRun records one month-end FX revaluation execution and GL journal.
type FxRevaluationRun struct {
	TenantID                string    `json:"tenant_id"`
	RunID                   string    `json:"run_id"`
	EntityID                string    `json:"entity_id,omitempty"`
	PeriodID                string    `json:"period_id"`
	BaseCurrency            string    `json:"base_currency"`
	UnrealizedGainLossMinor int64     `json:"unrealized_gain_loss_minor"`
	RoundingDeltaMinor      int64     `json:"rounding_delta_minor"`
	RatesJSON               string    `json:"rates_json,omitempty"`
	JournalEntryID          string    `json:"journal_entry_id,omitempty"`
	IdempotencyKey          string    `json:"idempotency_key,omitempty"`
	CreatedBy               string    `json:"created_by"`
	CreatedAt               time.Time `json:"created_at"`
}

// FxRevalInput is the command payload for POST /v1/finance/fx-revaluation.
type FxRevalInput struct {
	TenantID       string `json:"tenant_id,omitempty"`
	EntityID       string `json:"entity_id,omitempty"`
	PeriodID       string `json:"period_id,omitempty"`
	BaseCurrency   string `json:"base_currency,omitempty"`
	AsOf           string `json:"as_of,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	CreatedBy      string `json:"-"`
	ActorType      string `json:"-"`
	Role           string `json:"-"`
	TraceID        string `json:"-"`
}

// FxRevalPosition captures one open foreign-currency balance position evaluated during revaluation.
type FxRevalPosition struct {
	SourceType                string    `json:"source_type"`
	SourceID                  string    `json:"source_id"`
	AccountCode               string    `json:"account_code"`
	Direction                 string    `json:"direction"` // ASSET or LIABILITY
	Currency                  string    `json:"currency"`
	ForeignAmountMinor        int64     `json:"foreign_amount_minor"`
	CarryingFunctionalMinor   int64     `json:"carrying_functional_minor"`
	RevaluedFunctionalMinor   int64     `json:"revalued_functional_minor"`
	UnrealizedGainLossMinor   int64     `json:"unrealized_gain_loss_minor"`
	RateID                    string    `json:"rate_id,omitempty"`
	RateScaled                int64     `json:"rate_scaled"`
	Scale                     int64     `json:"scale"`
	OccurredAt                time.Time `json:"-"`
	HasHistoricalFunctional   bool      `json:"-"`
	PriorRevalFunctionalMinor int64     `json:"-"`
}

// FxRevalResult is the JSON response for POST /v1/finance/fx-revaluation.
type FxRevalResult struct {
	Run           FxRevaluationRun  `json:"run"`
	AlreadyPosted bool              `json:"already_posted"`
	PositionCount int               `json:"position_count"`
	Positions     []FxRevalPosition `json:"positions,omitempty"`
}

// FxRateApplied documents an exact integer-scaled FX rate used in revaluation or consolidation.
type FxRateApplied struct {
	RateID        string    `json:"rate_id,omitempty"`
	BaseCurrency  string    `json:"base_currency"`
	QuoteCurrency string    `json:"quote_currency"`
	RateScaled    int64     `json:"rate_scaled"`
	Scale         int64     `json:"scale"`
	Inverse       bool      `json:"inverse,omitempty"`
	EffectiveAt   time.Time `json:"effective_at,omitempty"`
	Source        string    `json:"source,omitempty"`
}

// ConsolidatedEntitySummary reports one entity's native and reporting-currency trial balance totals.
type ConsolidatedEntitySummary struct {
	EntityID             string `json:"entity_id"`
	Name                 string `json:"name"`
	TIN                  string `json:"tin,omitempty"`
	CountryCode          string `json:"country_code"`
	BaseCurrency         string `json:"base_currency"`
	IsParent             bool   `json:"is_parent"`
	FxRateScaled         int64  `json:"fx_rate_scaled,omitempty"`
	FxScale              int64  `json:"fx_scale,omitempty"`
	FxInverse            bool   `json:"fx_inverse,omitempty"`
	DebitMinor           int64  `json:"debit_minor"`
	CreditMinor          int64  `json:"credit_minor"`
	NetMinor             int64  `json:"net_minor"`
	ConvertedDebitMinor  int64  `json:"converted_debit_minor"`
	ConvertedCreditMinor int64  `json:"converted_credit_minor"`
	ConvertedNetMinor    int64  `json:"converted_net_minor"`
	Balanced             bool   `json:"balanced"`
}

// IntercompanyElimination details one eliminated intercompany transfer or IC balance pair.
type IntercompanyElimination struct {
	TransferID           string `json:"transfer_id,omitempty"`
	AccountCode          string `json:"account_code,omitempty"`
	FromEntityID         string `json:"from_entity_id"`
	ToEntityID           string `json:"to_entity_id"`
	Currency             string `json:"currency,omitempty"`
	AmountMinor          int64  `json:"amount_minor"`
	ConvertedAmountMinor int64  `json:"converted_amount_minor"`
	TransferType         string `json:"transfer_type,omitempty"`
	Description          string `json:"description,omitempty"`
}

// ConsolidatedAccountLine reports pre-elimination, eliminated, and consolidated totals per COA account.
type ConsolidatedAccountLine struct {
	AccountCode               string           `json:"account_code"`
	AccountName               string           `json:"account_name,omitempty"`
	Class                     string           `json:"class,omitempty"`
	Currency                  string           `json:"currency"`
	PreEliminationDebitMinor  int64            `json:"pre_elimination_debit_minor"`
	PreEliminationCreditMinor int64            `json:"pre_elimination_credit_minor"`
	DebitMinor                int64            `json:"debit_minor"`
	CreditMinor               int64            `json:"credit_minor"`
	EliminatedMinor           int64            `json:"eliminated_minor"`
	NetMinor                  int64            `json:"net_minor"`
	ByEntity                  map[string]int64 `json:"by_entity,omitempty"`
}

// ConsolidationReport is the JSON response for GET /v1/finance/consolidation.
type ConsolidationReport struct {
	TenantID               string                      `json:"tenant_id"`
	PeriodID               string                      `json:"period_id"`
	ReportingCurrency      string                      `json:"reporting_currency"`
	EntityIDs              []string                    `json:"entity_ids"`
	Entities               []ConsolidatedEntitySummary `json:"entities"`
	Eliminations           []IntercompanyElimination   `json:"eliminations"`
	Accounts               []ConsolidatedAccountLine   `json:"accounts"`
	TotalDebitsMinor       int64                       `json:"total_debits_minor"`
	TotalCreditsMinor      int64                       `json:"total_credits_minor"`
	TotalEliminationsMinor int64                       `json:"total_eliminations_minor"`
	ICVarianceMinor        int64                       `json:"ic_variance_minor"`
	Balanced               bool                        `json:"balanced"`
	FxRatesApplied         []FxRateApplied             `json:"fx_rates_applied"`
}

// AuditRecordInput is the narrow audit-event payload recorded by multientity mutations.
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

// AuditRecorder persists immutable AuditEvents synchronously or inside a Spanner RW transaction.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
	RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev AuditRecordInput) error
}

// ConsolidationSnapshot holds the raw consistent read snapshot used to compute consolidation.
type ConsolidationSnapshot struct {
	Entities      []LegalEntity
	TrialBalances map[string]gl.TrialBalanceReport
	Transfers     []IntercompanyTransfer
}
