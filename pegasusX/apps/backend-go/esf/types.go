package esf

import (
	"errors"
	"strings"
	"time"
)

// ESF directions, statuses, actions, and operators.
const (
	DirectionOutbound = "OUTBOUND"
	DirectionInbound  = "INBOUND"

	StatusDraft     = "DRAFT"
	StatusSubmitted = "SUBMITTED"
	StatusAccepted  = "ACCEPTED"
	StatusRejected  = "REJECTED"
	StatusCancelled = "CANCELLED"
	StatusFailed    = "FAILED"

	ActionAccept = "ACCEPT"
	ActionReject = "REJECT"
	ActionCancel = "CANCEL"

	OperatorDirect  = "direct"
	OperatorDidox   = "didox"
	OperatorFaktura = "faktura"
)

// Domain sentinel errors.
var (
	ErrInvalidInput       = errors.New("esf: invalid input")
	ErrNotFound           = errors.New("esf: document not found")
	ErrOrderNotFound      = errors.New("esf: order not found")
	ErrOrderForbidden     = errors.New("esf: order does not belong to entity")
	ErrInvalidTransition  = errors.New("esf: invalid status transition")
	ErrRepoUnavailable    = errors.New("esf: repository unavailable")
	ErrAlreadyExists      = errors.New("esf: document already exists")
	ErrUpstreamSubmission = errors.New("esf: soliq submission failed")
)

// EsfDocument models one persisted EsfDocuments row.
type EsfDocument struct {
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

// EsfStatusHistoryEntry models one persisted EsfStatusHistory row.
type EsfStatusHistoryEntry struct {
	EntityID            string    `json:"entity_id"`
	EsfID               string    `json:"esf_id"`
	HistoryID           string    `json:"history_id"`
	FromStatus          string    `json:"from_status,omitempty"`
	ToStatus            string    `json:"to_status"`
	ActorID             string    `json:"actor_id"`
	Reason              string    `json:"reason,omitempty"`
	OperatorPayloadJSON string    `json:"operator_payload_json,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// EsfDetailResponse bundles an EsfDocument with its chronological status history.
type EsfDetailResponse struct {
	Document EsfDocument             `json:"document"`
	History  []EsfStatusHistoryEntry `json:"history"`
}

// EsfCanonicalLine is one tax-snapshot-derived line inside the EDS-signed ESF payload (zero floats).
type EsfCanonicalLine struct {
	OrderLineID string `json:"order_line_id"`
	RegimeID    string `json:"regime_id"`
	VatRateBps  int64  `json:"vat_rate_bps"`
	NetMinor    int64  `json:"net_minor"`
	VatMinor    int64  `json:"vat_minor"`
	GrossMinor  int64  `json:"gross_minor"`
}

// EsfCanonicalPayload is the deterministic JSON structure signed via fiscal.AttachSignature.
type EsfCanonicalPayload struct {
	EsfID            string             `json:"esf_id"`
	EntityID         string             `json:"entity_id"`
	Direction        string             `json:"direction"`
	OrderID          string             `json:"order_id,omitempty"`
	PoID             string             `json:"po_id,omitempty"`
	SupplierTIN      string             `json:"supplier_tin,omitempty"`
	CounterpartyTIN  string             `json:"counterparty_tin"`
	CounterpartyName string             `json:"counterparty_name,omitempty"`
	Operator         string             `json:"operator"`
	PeriodID         string             `json:"period_id"`
	Currency         string             `json:"currency"`
	TotalMinor       int64              `json:"total_minor"`
	VatMinor         int64              `json:"vat_minor"`
	VatRateBps       int64              `json:"vat_rate_bps"`
	Lines            []EsfCanonicalLine `json:"lines,omitempty"`
}

// OrderFiscalSource carries Order + OrderLineFiscalSnapshots + TaxRegimeVersions data to build an outbound ESF.
type OrderFiscalSource struct {
	OrderID          string
	SupplierID       string
	RetailerID       string
	SupplierTIN      string
	CounterpartyTIN  string
	CounterpartyName string
	Currency         string
	TotalMinor       int64
	CreatedAt        time.Time
	Lines            []EsfCanonicalLine
	TotalNetMinor    int64
	TotalVatMinor    int64
	TotalGrossMinor  int64
	DominantVatBps   int64
}

// CreateOutboundRequest is the payload for POST /v1/fiscal/esf.
type CreateOutboundRequest struct {
	EntityID         string `json:"entity_id,omitempty"`
	OrderID          string `json:"order_id"`
	Operator         string `json:"operator,omitempty"`
	CounterpartyTIN  string `json:"counterparty_tin,omitempty"`
	CounterpartyName string `json:"counterparty_name,omitempty"`
	Submit           *bool  `json:"submit,omitempty"` // default true
	ActorID          string `json:"-"`
}

// CreateInboundRequest is the payload for POST /v1/fiscal/esf/inbound.
type CreateInboundRequest struct {
	EntityID         string `json:"entity_id,omitempty"`
	EsfID            string `json:"esf_id,omitempty"`
	PoID             string `json:"po_id,omitempty"`
	OrderID          string `json:"order_id,omitempty"`
	CounterpartyTIN  string `json:"counterparty_tin"`
	CounterpartyName string `json:"counterparty_name,omitempty"`
	TotalMinor       int64  `json:"total_minor"`
	VatMinor         int64  `json:"vat_minor"`
	VatRateBps       int64  `json:"vat_rate_bps"`
	Operator         string `json:"operator,omitempty"`
	OperatorRef      string `json:"operator_ref,omitempty"`
	PeriodID         string `json:"period_id,omitempty"`
	Status           string `json:"status,omitempty"` // SUBMITTED (default) or ACCEPTED
	IssuedAt         string `json:"issued_at,omitempty"`
	ActorID          string `json:"-"`
}

// TransitionStatusRequest is the payload for POST /v1/fiscal/esf/{id}/status.
type TransitionStatusRequest struct {
	EntityID        string `json:"entity_id,omitempty"`
	Action          string `json:"action,omitempty"` // ACCEPT | REJECT | CANCEL
	Status          string `json:"status,omitempty"` // alternative: ACCEPTED | REJECTED | CANCELLED
	Reason          string `json:"reason,omitempty"`
	OperatorRef     string `json:"operator_ref,omitempty"`
	OperatorPayload string `json:"operator_payload,omitempty"`
	ActorID         string `json:"-"`
}

// TransitionStatusResult summarizes the ESF transition and any triggered reverse settlement / GL VAT reversal.
type TransitionStatusResult struct {
	Document          EsfDocument             `json:"document"`
	History           []EsfStatusHistoryEntry `json:"history"`
	VatReversedMinor  int64                   `json:"vat_reversed_minor,omitempty"`
	GlReversalEntryID string                  `json:"gl_reversal_entry_id,omitempty"`
	CreditNoteID      string                  `json:"credit_note_id,omitempty"`
}

// ListFilter narrows GET /v1/fiscal/esf queries.
type ListFilter struct {
	EntityID  string
	Direction string
	Status    string
	PeriodID  string
	OrderID   string
	Limit     int
}

// VATRateBreakdown summarizes ESF totals grouped by VAT rate (integer bps).
type VATRateBreakdown struct {
	VatRateBps        int64 `json:"vat_rate_bps"`
	OutboundBaseMinor int64 `json:"outbound_base_minor"`
	OutboundVatMinor  int64 `json:"outbound_vat_minor"`
	InboundBaseMinor  int64 `json:"inbound_base_minor"`
	InboundVatMinor   int64 `json:"inbound_vat_minor"`
	DocCount          int64 `json:"doc_count"`
}

// VATSummaryReport compares ESF output/input VAT against GL TAX:VAT:OUTPUT and TAX:VAT:INPUT.
type VATSummaryReport struct {
	EntityID          string             `json:"entity_id"`
	PeriodID          string             `json:"period_id"`
	Currency          string             `json:"currency"`
	EsfOutputVatMinor int64              `json:"esf_output_vat_minor"`
	EsfInputVatMinor  int64              `json:"esf_input_vat_minor"`
	EsfNetVatMinor    int64              `json:"esf_net_vat_minor"`
	GlOutputVatMinor  int64              `json:"gl_output_vat_minor"`
	GlInputVatMinor   int64              `json:"gl_input_vat_minor"`
	GlNetVatMinor     int64              `json:"gl_net_vat_minor"`
	OutputDiffMinor   int64              `json:"output_diff_minor"`
	InputDiffMinor    int64              `json:"input_diff_minor"`
	DiffMinor         int64              `json:"diff_minor"`
	Reconciled        bool               `json:"reconciled"`
	OutboundCount     int64              `json:"outbound_count"`
	InboundCount      int64              `json:"inbound_count"`
	RejectedCount     int64              `json:"rejected_count"`
	ByRate            []VATRateBreakdown `json:"by_rate"`
}

// MissingIKPUItem describes an active SKU missing Soliq IKPU/MXIK, package, or marking codes.
type MissingIKPUItem struct {
	ProductID     string   `json:"product_id"`
	Name          string   `json:"name"`
	CategoryID    string   `json:"category_id,omitempty"`
	Barcode       string   `json:"barcode,omitempty"`
	IKPUCode      string   `json:"ikpu_code,omitempty"`
	PackageCode   string   `json:"package_code,omitempty"`
	MarkingCode   string   `json:"marking_code,omitempty"`
	MissingFields []string `json:"missing_fields"`
}

// MissingIKPUResponse is returned by GET /v1/fiscal/missing-ikpu.
type MissingIKPUResponse struct {
	SupplierID string            `json:"supplier_id"`
	Items      []MissingIKPUItem `json:"items"`
	Count      int               `json:"count"`
}

func normalizeOperator(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case OperatorDidox:
		return OperatorDidox
	case OperatorFaktura:
		return OperatorFaktura
	default:
		return OperatorDirect
	}
}

func normalizeActionToTargetStatus(action, status string) (string, error) {
	raw := strings.ToUpper(strings.TrimSpace(action))
	if raw == "" {
		raw = strings.ToUpper(strings.TrimSpace(status))
	}
	switch raw {
	case ActionAccept, StatusAccepted:
		return StatusAccepted, nil
	case ActionReject, StatusRejected:
		return StatusRejected, nil
	case ActionCancel, StatusCancelled:
		return StatusCancelled, nil
	default:
		return "", ErrInvalidInput
	}
}
