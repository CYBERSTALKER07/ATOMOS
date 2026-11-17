// Package exportpacks builds M1.10 auditor/bank/tax financial packs from immutable
// ledger, subledger, ESF, cash, approval, and reconciliation tables inside Keyset-paginated
// ReadOnlyTransaction queries and stores watermarked, checksummed artifacts with expiry
// enforcement and Outbox/AuditEvents trails.
package exportpacks

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// Supported pack types.
const (
	PackTypeAudit = "AUDIT"
	PackTypeBank  = "BANK"
	PackTypeTax   = "TAX"
)

// Supported output formats.
const (
	FormatZip  = "zip"
	FormatXLSX = "xlsx"
	FormatCSV  = "csv"
	FormatXML  = "xml"
	FormatJSON = "json"
)

// Pack statuses.
const (
	StatusReady   = "READY"
	StatusExpired = "EXPIRED"
)

// DefaultTTLHours is used when ttl_hours is omitted or 0.
const DefaultTTLHours int64 = 72

// Canonical error sentinels mapped to HTTP status codes by handlers.
var (
	ErrInvalidInput         = errors.New("invalid_export_pack_input")
	ErrForbidden            = errors.New("export_pack_forbidden")
	ErrNotFound             = errors.New("export_pack_not_found")
	ErrPackExpired          = errors.New("export_pack_expired")
	ErrInvalidDownloadToken = errors.New("invalid_download_token")
	ErrStorageUnavailable   = errors.New("export_pack_storage_unavailable")
	ErrRepoUnavailable      = errors.New("export_pack_repo_unavailable")
)

// ArtifactMeta describes a generated pack file stored in GCS or local storage.
type ArtifactMeta struct {
	Format      string `json:"format"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Sha256Hex   string `json:"sha256_hex"`
	StorageURI  string `json:"storage_uri"`
}

// PackDatasetStats captures exact collected row counts across all 15 audit sources (int64 only)
// plus dedicated Bank Credit Tape schedule counts when PackType == BANK.
type PackDatasetStats struct {
	GlAccounts                int64 `json:"gl_accounts"`
	JournalEntries            int64 `json:"journal_entries"`
	JournalPostings           int64 `json:"journal_postings"`
	ArInvoices                int64 `json:"ar_invoices"`
	InventoryCostLayers       int64 `json:"inventory_cost_layers"`
	InventoryMovements        int64 `json:"inventory_movements"`
	EsfDocuments              int64 `json:"esf_documents"`
	CashierBlindCountSessions int64 `json:"cashier_blind_count_sessions"`
	DepotVaultLedgerEntries   int64 `json:"depot_vault_ledger_entries"`
	PaymentLedgerEntries      int64 `json:"payment_ledger_entries"`
	ApprovalRequests          int64 `json:"approval_requests"`
	ApprovalDecisions         int64 `json:"approval_decisions"`
	AuditChainAnchors         int64 `json:"audit_chain_anchors"`
	ReconciliationRuns        int64 `json:"reconciliation_runs"`
	ReconciliationBreaks      int64 `json:"reconciliation_breaks"`
	BankCreditTapeRows        int64 `json:"bank_credit_tape_rows,omitempty"`
	DpdBucketRows             int64 `json:"dpd_bucket_rows,omitempty"`
	DebtorConcentrationRows   int64 `json:"debtor_concentration_rows,omitempty"`
	LossRecoveryCurveRows     int64 `json:"loss_recovery_curve_rows,omitempty"`
}

// Total returns the total collected row count across all datasets.
func (s PackDatasetStats) Total() int64 {
	return s.GlAccounts +
		s.JournalEntries +
		s.JournalPostings +
		s.ArInvoices +
		s.InventoryCostLayers +
		s.InventoryMovements +
		s.EsfDocuments +
		s.CashierBlindCountSessions +
		s.DepotVaultLedgerEntries +
		s.PaymentLedgerEntries +
		s.ApprovalRequests +
		s.ApprovalDecisions +
		s.AuditChainAnchors +
		s.ReconciliationRuns +
		s.ReconciliationBreaks +
		s.BankCreditTapeRows +
		s.DpdBucketRows +
		s.DebtorConcentrationRows +
		s.LossRecoveryCurveRows
}

// ManifestFileRef describes an individual file inside the generated pack bundle.
type ManifestFileRef struct {
	Path        string `json:"path"`
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Sha256Hex   string `json:"sha256_hex"`
}

// PackManifest is the signed manifest.json bundled inside every auditor/tax/bank pack.
type PackManifest struct {
	SchemaVersion string             `json:"schema_version"`
	PackID        string             `json:"pack_id"`
	PackType      string             `json:"pack_type"`
	TenantID      string             `json:"tenant_id"`
	EntityID      string             `json:"entity_id"`
	PeriodID      string             `json:"period_id,omitempty"`
	FromDate      time.Time          `json:"from_date"`
	ToDate        time.Time          `json:"to_date"`
	WatermarkText string             `json:"watermark_text"`
	CreatedBy     string             `json:"created_by"`
	CreatedAt     time.Time          `json:"created_at"`
	ExpiresAt     time.Time          `json:"expires_at"`
	TotalRows     int64              `json:"total_rows"`
	Datasets      PackDatasetStats   `json:"datasets"`
	BankCredit    *BankCreditSummary `json:"bank_credit_summary,omitempty"`
	Files         []ManifestFileRef  `json:"files"`
	SignatureAlg  string             `json:"signature_alg,omitempty"`
	SignatureHex  string             `json:"signature_hex,omitempty"`
}

// ExportPack mirrors the ExportPacks Spanner table and JSON API representation.
type ExportPack struct {
	TenantID      string         `json:"tenant_id"`
	PackID        string         `json:"pack_id"`
	PackType      string         `json:"pack_type"`
	EntityID      string         `json:"entity_id"`
	PeriodID      string         `json:"period_id,omitempty"`
	FromDate      time.Time      `json:"from_date"`
	ToDate        time.Time      `json:"to_date"`
	Format        string         `json:"format"`
	Status        string         `json:"status"`
	Manifest      *PackManifest  `json:"manifest,omitempty"`
	ManifestJSON  string         `json:"manifest_json,omitempty"`
	Artifacts     []ArtifactMeta `json:"artifacts,omitempty"`
	ArtifactsJSON string         `json:"artifacts_json,omitempty"`
	DownloadToken string         `json:"download_token,omitempty"`
	WatermarkText string         `json:"watermark_text"`
	RowCount      int64          `json:"row_count"`
	Sha256Hex     string         `json:"sha256_hex"`
	CreatedBy     string         `json:"created_by"`
	ExpiresAt     time.Time      `json:"expires_at"`
	CreatedAt     time.Time      `json:"created_at"`
	Expired       bool           `json:"expired"`
	DownloadURL   string         `json:"download_url,omitempty"`
}

// CreatePackInput configures pack generation on POST /v1/finance/packs.
type CreatePackInput struct {
	TenantID  string
	PackType  string
	EntityID  string
	PeriodID  string
	FromDate  *time.Time
	ToDate    *time.Time
	Format    string
	TTLHours  int64
	ActorID   string
	ActorRole string
	ActorType string
	TraceID   string
	IP        string
	DeviceID  string
}

// DownloadPackInput identifies a pack download request.
type DownloadPackInput struct {
	TenantID      string
	EntityID      string
	PackID        string
	Format        string
	DownloadToken string
	ActorID       string
	ActorRole     string
	ActorType     string
	TraceID       string
	IP            string
	DeviceID      string
}

// DownloadResult returns verified pack bytes and HTTP response headers.
type DownloadResult struct {
	Pack        *ExportPack
	Artifact    ArtifactMeta
	Data        []byte
	ContentType string
	Filename    string
	Watermark   string
	Sha256Hex   string
}

// AuditRecordInput is a package-local DTO bridging pack actions into hash-chained AuditEvents.
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
	TraceID    string
	IP         string
	DeviceID   string
}

// AuditRecorder persists immutable hash-chained audit records.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
	RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev AuditRecordInput) error
}

// DatasetCollector gathers unlimited rows across the 15 financial/control datasets.
type DatasetCollector interface {
	Collect(ctx context.Context, entityID, periodID string, from, to time.Time) (*PackDatasets, error)
}

// BankDatasetCollector optionally collects additional credit, terms, and AR ledger datasets for BANK packs.
type BankDatasetCollector interface {
	CollectBank(ctx context.Context, entityID, periodID string, from, to time.Time) (*PackDatasets, error)
}

// PackDatasets holds all 15 source datasets plus optional Bank Credit Tape inputs (zero float64).
type PackDatasets struct {
	GlAccounts                []GlAccountRow
	JournalEntries            []JournalEntryRow
	JournalPostings           []JournalPostingRow
	ArInvoices                []ArInvoiceRow
	InventoryCostLayers       []InventoryCostLayerRow
	InventoryMovements        []InventoryMovementRow
	EsfDocuments              []EsfDocumentRow
	CashierBlindCountSessions []CashierBlindCountRow
	DepotVaultLedgerEntries   []DepotVaultLedgerRow
	PaymentLedgerEntries      []PaymentLedgerRow
	ApprovalRequests          []ApprovalRequestRow
	ApprovalDecisions         []ApprovalDecisionRow
	AuditChainAnchors         []AuditChainAnchorRow
	ReconciliationRuns        []ReconciliationRunRow
	ReconciliationBreaks      []ReconciliationBreakRow

	// Optional Bank Credit Tape inputs (populated by CollectBank or tests when PackType == BANK).
	BankArInvoices         []ArInvoiceRow
	RetailerPaymentTerms   []RetailerPaymentTermsRow
	RetailerCreditProfiles []RetailerCreditProfileRow
	ArLedgerEntries        []ArLedgerEntryRow
	CreditScoreHistory     []CreditScoreHistoryRow
}

// Stats returns row counts across all collected datasets.
func (d *PackDatasets) Stats() PackDatasetStats {
	if d == nil {
		return PackDatasetStats{}
	}
	return PackDatasetStats{
		GlAccounts:                int64(len(d.GlAccounts)),
		JournalEntries:            int64(len(d.JournalEntries)),
		JournalPostings:           int64(len(d.JournalPostings)),
		ArInvoices:                int64(len(d.ArInvoices)),
		InventoryCostLayers:       int64(len(d.InventoryCostLayers)),
		InventoryMovements:        int64(len(d.InventoryMovements)),
		EsfDocuments:              int64(len(d.EsfDocuments)),
		CashierBlindCountSessions: int64(len(d.CashierBlindCountSessions)),
		DepotVaultLedgerEntries:   int64(len(d.DepotVaultLedgerEntries)),
		PaymentLedgerEntries:      int64(len(d.PaymentLedgerEntries)),
		ApprovalRequests:          int64(len(d.ApprovalRequests)),
		ApprovalDecisions:         int64(len(d.ApprovalDecisions)),
		AuditChainAnchors:         int64(len(d.AuditChainAnchors)),
		ReconciliationRuns:        int64(len(d.ReconciliationRuns)),
		ReconciliationBreaks:      int64(len(d.ReconciliationBreaks)),
	}
}

type GlAccountRow struct {
	EntityID    string     `json:"entity_id"`
	AccountCode string     `json:"account_code"`
	Class       string     `json:"class"`
	Name        string     `json:"name"`
	ParentCode  string     `json:"parent_code,omitempty"`
	Currency    string     `json:"currency"`
	IsControl   bool       `json:"is_control"`
	CountryPack string     `json:"country_pack"`
	ActiveFrom  *time.Time `json:"active_from,omitempty"`
	ActiveTo    *time.Time `json:"active_to,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type JournalEntryRow struct {
	EntityID         string    `json:"entity_id"`
	EntryID          string    `json:"entry_id"`
	PeriodID         string    `json:"period_id"`
	OriginalPeriodID string    `json:"original_period_id,omitempty"`
	SourceEventID    string    `json:"source_event_id,omitempty"`
	SourceType       string    `json:"source_type"`
	SourceID         string    `json:"source_id"`
	OrderID          string    `json:"order_id,omitempty"`
	Memo             string    `json:"memo,omitempty"`
	Currency         string    `json:"currency"`
	CountryPack      string    `json:"country_pack"`
	ReversesEntryID  string    `json:"reverses_entry_id,omitempty"`
	PrevHash         string    `json:"prev_hash"`
	EntryHash        string    `json:"entry_hash"`
	PostedBy         string    `json:"posted_by"`
	OccurredAt       time.Time `json:"occurred_at"`
	PostedAt         time.Time `json:"posted_at"`
}

type JournalPostingRow struct {
	EntityID              string `json:"entity_id"`
	EntryID               string `json:"entry_id"`
	Line                  int64  `json:"line"`
	AccountCode           string `json:"account_code"`
	Direction             string `json:"direction"`
	AmountMinor           int64  `json:"amount_minor"`
	Currency              string `json:"currency"`
	FunctionalAmountMinor int64  `json:"functional_amount_minor"`
	FxRateID              string `json:"fx_rate_id,omitempty"`
	PartyType             string `json:"party_type,omitempty"`
	PartyID               string `json:"party_id,omitempty"`
	WarehouseID           string `json:"warehouse_id,omitempty"`
	ProductID             string `json:"product_id,omitempty"`
	LotID                 string `json:"lot_id,omitempty"`
	Description           string `json:"description,omitempty"`
}

type ArInvoiceRow struct {
	InvoiceID       string     `json:"invoice_id"`
	SupplierID      string     `json:"supplier_id"`
	RetailerID      string     `json:"retailer_id"`
	OrderID         string     `json:"order_id"`
	Status          string     `json:"status"`
	PrincipalMinor  int64      `json:"principal_minor"`
	BalanceMinor    int64      `json:"balance_minor"`
	Currency        string     `json:"currency"`
	CreditLeaveAt   time.Time  `json:"credit_leave_at"`
	DueAt           time.Time  `json:"due_at"`
	TermsDays       int64      `json:"terms_days"`
	GracePeriodDays int64      `json:"grace_period_days"`
	AgingBucket     string     `json:"aging_bucket,omitempty"`
	LastDunnedAt    *time.Time `json:"last_dunned_at,omitempty"`
	DunningStep     int64      `json:"dunning_step"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type InventoryCostLayerRow struct {
	EntityID       string    `json:"entity_id"`
	WarehouseID    string    `json:"warehouse_id"`
	ProductID      string    `json:"product_id"`
	LayerID        string    `json:"layer_id"`
	LotID          string    `json:"lot_id,omitempty"`
	ReceiptRefType string    `json:"receipt_ref_type,omitempty"`
	ReceiptRefID   string    `json:"receipt_ref_id,omitempty"`
	QtyRemaining   int64     `json:"qty_remaining"`
	OriginalQty    int64     `json:"original_qty"`
	UnitCostMinor  int64     `json:"unit_cost_minor"`
	Currency       string    `json:"currency"`
	FxRateID       string    `json:"fx_rate_id,omitempty"`
	ReceivedAt     time.Time `json:"received_at"`
	ExpiryDate     string    `json:"expiry_date,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type InventoryMovementRow struct {
	EntityID       string    `json:"entity_id"`
	MovementID     string    `json:"movement_id"`
	MovementType   string    `json:"movement_type"`
	ReferenceType  string    `json:"reference_type"`
	ReferenceID    string    `json:"reference_id"`
	WarehouseID    string    `json:"warehouse_id"`
	ProductID      string    `json:"product_id"`
	LotID          string    `json:"lot_id,omitempty"`
	LayerID        string    `json:"layer_id,omitempty"`
	QtyDelta       int64     `json:"qty_delta"`
	UnitCostMinor  int64     `json:"unit_cost_minor"`
	TotalCostMinor int64     `json:"total_cost_minor"`
	Currency       string    `json:"currency"`
	GlEntryID      string    `json:"gl_entry_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type EsfDocumentRow struct {
	EntityID         string     `json:"entity_id"`
	EsfID            string     `json:"esf_id"`
	Direction        string     `json:"direction"`
	OrderID          string     `json:"order_id,omitempty"`
	PoID             string     `json:"po_id,omitempty"`
	CounterpartyTIN  string     `json:"counterparty_tin"`
	CounterpartyName string     `json:"counterparty_name,omitempty"`
	TotalMinor       int64      `json:"total_minor"`
	VatMinor         int64      `json:"vat_minor"`
	VatRateBps       int64      `json:"vat_rate_bps"`
	Status           string     `json:"status"`
	Operator         string     `json:"operator"`
	OperatorRef      string     `json:"operator_ref,omitempty"`
	SignedPayloadURI string     `json:"signed_payload_uri,omitempty"`
	PayloadHash      string     `json:"payload_hash,omitempty"`
	FailureReason    string     `json:"failure_reason,omitempty"`
	PeriodID         string     `json:"period_id"`
	IssuedAt         *time.Time `json:"issued_at,omitempty"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CashierBlindCountRow struct {
	SessionID         string     `json:"session_id"`
	SupplierID        string     `json:"supplier_id"`
	DepotID           string     `json:"depot_id"`
	DriverID          string     `json:"driver_id"`
	RouteID           string     `json:"route_id"`
	BagSealBarcode    string     `json:"bag_seal_barcode"`
	AttemptNumber     int64      `json:"attempt_number"`
	CountedCashMinor  int64      `json:"counted_cash_minor"`
	ExpectedCashMinor int64      `json:"expected_cash_minor"`
	VarianceMinor     int64      `json:"variance_minor"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
}

type DepotVaultLedgerRow struct {
	VaultEntryID        string    `json:"vault_entry_id"`
	DepotID             string    `json:"depot_id"`
	SupplierID          string    `json:"supplier_id"`
	SessionID           string    `json:"session_id,omitempty"`
	CITManifestID       string    `json:"cit_manifest_id,omitempty"`
	EntryType           string    `json:"entry_type"`
	AmountMinor         int64     `json:"amount_minor"`
	RunningBalanceMinor int64     `json:"running_balance_minor"`
	CourierBadgeID      string    `json:"courier_badge_id,omitempty"`
	SealBarcode         string    `json:"seal_barcode,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

type PaymentLedgerRow struct {
	LedgerEntryID string    `json:"ledger_entry_id"`
	SessionID     string    `json:"session_id,omitempty"`
	OrderID       string    `json:"order_id,omitempty"`
	SupplierID    string    `json:"supplier_id,omitempty"`
	RetailerID    string    `json:"retailer_id,omitempty"`
	Gateway       string    `json:"gateway"`
	EntryType     string    `json:"entry_type"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
	ReferenceID   string    `json:"reference_id,omitempty"`
	Source        string    `json:"source"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type ApprovalRequestRow struct {
	TenantID          string     `json:"tenant_id"`
	RequestID         string     `json:"request_id"`
	PolicyID          string     `json:"policy_id,omitempty"`
	PolicyVersion     int64      `json:"policy_version"`
	ObjectType        string     `json:"object_type"`
	ObjectID          string     `json:"object_id"`
	Action            string     `json:"action"`
	AmountMinor       int64      `json:"amount_minor"`
	RequestedBy       string     `json:"requested_by"`
	ActorType         string     `json:"actor_type"`
	PayloadHash       string     `json:"payload_hash"`
	Status            string     `json:"status"`
	CurrentStep       int64      `json:"current_step"`
	RequiredApprovers int64      `json:"required_approvers"`
	BreakGlass        bool       `json:"break_glass"`
	BreakGlassReason  string     `json:"break_glass_reason,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ApprovalDecisionRow struct {
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

type AuditChainAnchorRow struct {
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

type ReconciliationRunRow struct {
	EntityID      string     `json:"entity_id"`
	RunID         string     `json:"run_id"`
	PeriodID      string     `json:"period_id"`
	Mode          string     `json:"mode"`
	Status        string     `json:"status"`
	BreakCount    int64      `json:"break_count"`
	ExposureMinor int64      `json:"exposure_minor"`
	TriggeredBy   string     `json:"triggered_by"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type ReconciliationBreakRow struct {
	EntityID                string     `json:"entity_id"`
	BreakID                 string     `json:"break_id"`
	RunID                   string     `json:"run_id"`
	Dimension               string     `json:"dimension"`
	ObjectType              string     `json:"object_type"`
	ObjectID                string     `json:"object_id"`
	ExpectedMinor           int64      `json:"expected_minor"`
	ActualMinor             int64      `json:"actual_minor"`
	DeltaMinor              int64      `json:"delta_minor"`
	Currency                string     `json:"currency"`
	Severity                string     `json:"severity"`
	Status                  string     `json:"status"`
	OwnerID                 string     `json:"owner_id,omitempty"`
	SlaDueAt                *time.Time `json:"sla_due_at,omitempty"`
	ResolutionNote          string     `json:"resolution_note,omitempty"`
	ControlTowerExceptionID string     `json:"control_tower_exception_id,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// NormalizePackType validates and canonicalizes a requested pack_type.
func NormalizePackType(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", PackTypeAudit:
		return PackTypeAudit, true
	case PackTypeBank:
		return PackTypeBank, true
	case PackTypeTax:
		return PackTypeTax, true
	default:
		return "", false
	}
}

// NormalizeFormat validates and canonicalizes a requested output format.
func NormalizeFormat(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", FormatZip:
		return FormatZip, true
	case FormatXLSX:
		return FormatXLSX, true
	case FormatCSV:
		return FormatCSV, true
	case FormatXML:
		return FormatXML, true
	case FormatJSON:
		return FormatJSON, true
	default:
		return "", false
	}
}

// ContentTypeForFormat returns the MIME Content-Type for a canonical export pack format.
func ContentTypeForFormat(format string) string {
	switch format {
	case FormatZip:
		return "application/zip"
	case FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatCSV:
		return "text/csv; charset=utf-8"
	case FormatXML:
		return "application/xml; charset=utf-8"
	case FormatJSON:
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// RetailerPaymentTermsRow mirrors RetailerPaymentTerms for Bank Credit Tape schedules.
type RetailerPaymentTermsRow struct {
	RetailerID        string     `json:"retailer_id"`
	SupplierID        string     `json:"supplier_id"`
	CreditEnabled     bool       `json:"credit_enabled"`
	EnabledAt         *time.Time `json:"enabled_at,omitempty"`
	DisabledAt        *time.Time `json:"disabled_at,omitempty"`
	TermsDays         int64      `json:"terms_days"`
	GracePeriodDays   int64      `json:"grace_period_days"`
	CreditLimitMinor  int64      `json:"credit_limit_minor"`
	UseGlobalDefaults bool       `json:"use_global_defaults"`
	ConsentVerified   bool       `json:"consent_verified"`
	Version           int64      `json:"version"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// RetailerCreditProfileRow mirrors RetailerCreditProfiles for Bank Credit Tape schedules.
type RetailerCreditProfileRow struct {
	RetailerID           string     `json:"retailer_id"`
	SupplierID           string     `json:"supplier_id"`
	CreditLimitMinor     int64      `json:"credit_limit_minor"`
	CurrentBalanceMinor  int64      `json:"current_balance_minor"`
	ReservedMinor        int64      `json:"reserved_minor"`
	AvailableCreditMinor int64      `json:"available_credit_minor"`
	RiskScore            int64      `json:"risk_score"`
	DelinquencyCount     int64      `json:"delinquency_count"`
	Status               string     `json:"status"`
	PrepayOnlyLocked     bool       `json:"prepay_only_locked"`
	LastEvaluatedAt      *time.Time `json:"last_evaluated_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ArLedgerEntryRow mirrors ArLedgerEntries for Bank Credit Tape recovery and write-off curves.
type ArLedgerEntryRow struct {
	EntryID        string    `json:"entry_id"`
	InvoiceID      string    `json:"invoice_id"`
	SupplierID     string    `json:"supplier_id"`
	RetailerID     string    `json:"retailer_id"`
	EntryType      string    `json:"entry_type"`
	AmountMinor    int64     `json:"amount_minor"`
	IdempotencyKey string    `json:"idempotency_key"`
	RefOrderID     string    `json:"ref_order_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreditScoreHistoryRow mirrors CreditScoreHistory (RELATIONSHIP & NETWORK scopes) with zero float64.
type CreditScoreHistoryRow struct {
	RetailerID            string    `json:"retailer_id"`
	ScoreID               string    `json:"score_id"`
	Scope                 string    `json:"scope"`
	SupplierID            string    `json:"supplier_id,omitempty"`
	ConsentVerified       bool      `json:"consent_verified"`
	ConsentScope          string    `json:"consent_scope,omitempty"`
	ModelVersion          string    `json:"model_version"`
	ScorerSource          string    `json:"scorer_source"`
	Score                 int64     `json:"score"`
	PdBps                 int64     `json:"pd_bps"`
	RiskTier              string    `json:"risk_tier"`
	SuggestedLimitMinor   int64     `json:"suggested_limit_minor"`
	CurrentLimitMinor     int64     `json:"current_limit_minor"`
	DisputesIsolatedCount int64     `json:"disputes_isolated_count"`
	WindowStart           time.Time `json:"window_start"`
	WindowEnd             time.Time `json:"window_end"`
	ComputedAt            time.Time `json:"computed_at"`
}

// BankCreditTapeRow represents one loan/invoice line in the Bank_Credit_Tape schedule (zero float64).
type BankCreditTapeRow struct {
	RetailerID           string `json:"retailer_id"`
	InvoiceID            string `json:"invoice_id"`
	OrderID              string `json:"order_id"`
	Status               string `json:"status"`
	Currency             string `json:"currency"`
	PrincipalMinor       int64  `json:"principal_minor"`
	BalanceMinor         int64  `json:"balance_minor"`
	RecoveredMinor       int64  `json:"recovered_minor"`
	WrittenOffMinor      int64  `json:"written_off_minor"`
	CreditLimitMinor     int64  `json:"credit_limit_minor"`
	UtilizationBps       int64  `json:"utilization_bps"`
	TermsDays            int64  `json:"terms_days"`
	GracePeriodDays      int64  `json:"grace_period_days"`
	DPDDays              int64  `json:"dpd_days"`
	DPDBucket            string `json:"dpd_bucket"`
	ConsentVerified      bool   `json:"consent_verified"`
	RelationshipScore    int64  `json:"relationship_score"`
	RelationshipPdBps    int64  `json:"relationship_pd_bps"`
	RelationshipRiskTier string `json:"relationship_risk_tier,omitempty"`
	NetworkScore         int64  `json:"network_score"`
	NetworkPdBps         int64  `json:"network_pd_bps"`
	NetworkRiskTier      string `json:"network_risk_tier,omitempty"`
	ProfileRiskScore     int64  `json:"profile_risk_score"`
	DelinquencyCount     int64  `json:"delinquency_count"`
	PrepayOnlyLocked     bool   `json:"prepay_only_locked"`
	CreditLeaveAt        string `json:"credit_leave_at"`
	DueAt                string `json:"due_at"`
}

// DPDBucketRow aggregates open AR exposure in one DPD bucket (`0-30`, `31-60`, `61-90`, `90+`).
type DPDBucketRow struct {
	Bucket         string `json:"bucket"`
	InvoiceCount   int64  `json:"invoice_count"`
	RetailerCount  int64  `json:"retailer_count"`
	PrincipalMinor int64  `json:"principal_minor"`
	ExposureMinor  int64  `json:"exposure_minor"`
	ShareBps       int64  `json:"share_bps"`
	Currency       string `json:"currency"`
}

// DebtorConcentrationRow aggregates top-N debtor exposure and concentration in basis points.
type DebtorConcentrationRow struct {
	Rank              int64  `json:"rank"`
	RetailerID        string `json:"retailer_id"`
	InvoiceCount      int64  `json:"invoice_count"`
	ExposureMinor     int64  `json:"exposure_minor"`
	PrincipalMinor    int64  `json:"principal_minor"`
	CreditLimitMinor  int64  `json:"credit_limit_minor"`
	ConcentrationBps  int64  `json:"concentration_bps"`
	CumulativeBps     int64  `json:"cumulative_bps"`
	MaxDPDDays        int64  `json:"max_dpd_days"`
	ConsentVerified   bool   `json:"consent_verified"`
	RelationshipScore int64  `json:"relationship_score"`
	NetworkScore      int64  `json:"network_score"`
	Currency          string `json:"currency"`
}

// LossRecoveryCurveRow summarizes monthly vintage origination, recovery, write-off, and default curves.
type LossRecoveryCurveRow struct {
	VintageMonth         string `json:"vintage_month"`
	InvoiceCount         int64  `json:"invoice_count"`
	OriginationMinor     int64  `json:"origination_minor"`
	RecoveredMinor       int64  `json:"recovered_minor"`
	WrittenOffMinor      int64  `json:"written_off_minor"`
	OutstandingMinor     int64  `json:"outstanding_minor"`
	DefaultExposureMinor int64  `json:"default_exposure_minor"`
	RecoveryRateBps      int64  `json:"recovery_rate_bps"`
	LossRateBps          int64  `json:"loss_rate_bps"`
	DefaultRateBps       int64  `json:"default_rate_bps"`
	Currency             string `json:"currency"`
}

// BankCreditSummary is embedded on PackManifest (`bank_credit_summary`) when PackType == BANK.
type BankCreditSummary struct {
	AsOf                     time.Time                `json:"as_of"`
	Currency                 string                   `json:"currency"`
	TotalInvoices            int64                    `json:"total_invoices"`
	TotalRetailers           int64                    `json:"total_retailers"`
	ConsentVerifiedRetailers int64                    `json:"consent_verified_retailers"`
	TotalPrincipalMinor      int64                    `json:"total_principal_minor"`
	TotalExposureMinor       int64                    `json:"total_exposure_minor"`
	TotalRecoveredMinor      int64                    `json:"total_recovered_minor"`
	TotalWrittenOffMinor     int64                    `json:"total_written_off_minor"`
	Top1ConcentrationBps     int64                    `json:"top_1_concentration_bps"`
	Top5ConcentrationBps     int64                    `json:"top_5_concentration_bps"`
	Top10ConcentrationBps    int64                    `json:"top_10_concentration_bps"`
	AvgRelationshipScore     int64                    `json:"avg_relationship_score"`
	AvgNetworkScore          int64                    `json:"avg_network_score"`
	DPDBuckets               []DPDBucketRow           `json:"dpd_buckets"`
	DebtorConcentration      []DebtorConcentrationRow `json:"debtor_concentration"`
	LossRecoveryCurves       []LossRecoveryCurveRow   `json:"loss_recovery_curves"`
}

// BankCreditScheduleData bundles the full computed Bank Credit Pack schedules and manifest summary.
type BankCreditScheduleData struct {
	Summary            BankCreditSummary
	Tape               []BankCreditTapeRow
	DPDBuckets         []DPDBucketRow
	Concentration      []DebtorConcentrationRow
	LossRecoveryCurves []LossRecoveryCurveRow
}
