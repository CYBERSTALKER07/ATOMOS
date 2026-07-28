package claims

import (
	"errors"
	"time"
)

// ClaimType classifies the logistics exception.
type ClaimType string

const (
	ClaimTypeDamaged         ClaimType = "DAMAGED"
	ClaimTypeMissing         ClaimType = "MISSING"
	ClaimTypeConcealedDamage ClaimType = "CONCEALED_DAMAGE"
	ClaimTypeTemperature     ClaimType = "TEMPERATURE"
	ClaimTypeTamper          ClaimType = "TAMPER"
	ClaimTypeOther           ClaimType = "OTHER"
)

// Valid returns true for known claim types.
func (t ClaimType) Valid() bool {
	switch t {
	case ClaimTypeDamaged, ClaimTypeMissing, ClaimTypeConcealedDamage,
		ClaimTypeTemperature, ClaimTypeTamper, ClaimTypeOther:
		return true
	}
	return false
}

// Status is the claim lifecycle state machine.
type Status string

const (
	StatusOpen        Status = "OPEN"
	StatusUnderReview Status = "UNDER_REVIEW"
	StatusApproved    Status = "APPROVED"
	StatusRejected    Status = "REJECTED"
	StatusResolved    Status = "RESOLVED"
)

// Valid returns true for known statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusUnderReview, StatusApproved, StatusRejected, StatusResolved:
		return true
	}
	return false
}

// Source is who/what opened the claim.
type Source string

const (
	SourceRetailerClaim    Source = "RETAILER_CLAIM"
	SourceDriverException  Source = "DRIVER_EXCEPTION"
	SourceTelemetry        Source = "TELEMETRY"
)

// EvidenceType classifies supporting proof.
type EvidenceType string

const (
	EvidencePhoto     EvidenceType = "PHOTO"
	EvidenceSignature EvidenceType = "SIGNATURE"
	EvidenceTempLog   EvidenceType = "TEMP_LOG"
	EvidenceSealBreak EvidenceType = "SEAL_BREAK"
	EvidenceNote      EvidenceType = "NOTE"
)

// Claim is the liability/resolution aggregate for a post-delivery exception.
type Claim struct {
	ClaimID         string         `json:"claim_id"`
	OrderID         string         `json:"order_id"`
	SupplierID      string         `json:"supplier_id"`
	RetailerID      string         `json:"retailer_id"`
	FiledBy         string         `json:"filed_by"`
	FiledByRole     string         `json:"filed_by_role"`
	ClaimType       ClaimType      `json:"claim_type"`
	Status          Status         `json:"status"`
	Description     string         `json:"description,omitempty"`
	AmountMinor     int64          `json:"amount_minor,omitempty"`
	Currency        string         `json:"currency,omitempty"`
	LineItems       []ClaimLine    `json:"line_items,omitempty"`
	Evidences       []Evidence     `json:"evidences,omitempty"`
	ResolutionNote  string         `json:"resolution_note,omitempty"`
	ResolvedBy      string         `json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty"`
	Source          Source         `json:"source"`
	TraceID         string         `json:"trace_id,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ClaimLine is one SKU on a claim.
// UnitPriceMinor / AmountMinor are filled from the order catalog price at file time
// so chargeback never depends on retailer-supplied amounts.
type ClaimLine struct {
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	Reason         string `json:"reason,omitempty"`
	UnitPriceMinor int64  `json:"unit_price_minor,omitempty"`
	AmountMinor    int64  `json:"amount_minor,omitempty"` // quantity * unit_price_minor
}

// Evidence is one proof artifact attached to a claim.
type Evidence struct {
	EvidenceID   string       `json:"evidence_id"`
	ClaimID      string       `json:"claim_id"`
	EvidenceType EvidenceType `json:"evidence_type"`
	URI          string       `json:"uri"`
	MimeType     string       `json:"mime_type,omitempty"`
	CapturedAt   *time.Time   `json:"captured_at,omitempty"`
	CapturedBy   string       `json:"captured_by,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

// FileClaimRequest is the retailer post-delivery claim body.
type FileClaimRequest struct {
	ClaimType   ClaimType   `json:"claim_type"`
	Description string      `json:"description"`
	AmountMinor int64       `json:"amount_minor"`
	Currency    string      `json:"currency"`
	LineItems   []ClaimLine `json:"line_items"`
	Evidences   []struct {
		EvidenceType EvidenceType `json:"evidence_type"`
		URI          string       `json:"uri"`
		MimeType     string       `json:"mime_type"`
	} `json:"evidences"`
}

// ApproveClaimRequest is admin/supplier adjudication body.
type ApproveClaimRequest struct {
	ResolutionNote string `json:"resolution_note"`
	// AmountMinor overrides priced total only for admin exception (capped at order total).
	AmountMinor int64 `json:"amount_minor,omitempty"`
	// SkipGatewayRefund forces ledger-only settlement (cash orders / manual refund).
	SkipGatewayRefund bool `json:"skip_gateway_refund,omitempty"`
}

// RejectClaimRequest is admin/supplier rejection body.
type RejectClaimRequest struct {
	ResolutionNote string `json:"resolution_note"`
}

// SettlementResult is returned after chargeback adjudication.
type SettlementResult struct {
	ChargebackID    string `json:"chargeback_id,omitempty"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	Gateway         string `json:"gateway,omitempty"`
	GatewayRefunded bool   `json:"gateway_refunded"`
	ProviderRef     string `json:"provider_ref,omitempty"`
	Mode            string `json:"mode"` // LEDGER_ONLY | LEDGER_AND_GATEWAY_REFUND | IDEMPOTENT_REPLAY
	Idempotent      bool   `json:"idempotent,omitempty"`
}

// Domain errors.
var (
	ErrOrderNotFound       = errors.New("order_not_found")
	ErrClaimNotFound       = errors.New("claim_not_found")
	ErrClaimNotAllowed     = errors.New("claim_not_allowed")
	ErrClaimWindowExpired  = errors.New("claim_window_expired")
	ErrEvidenceRequired    = errors.New("photo_evidence_required")
	ErrInvalidClaimType    = errors.New("invalid_claim_type")
	ErrInvalidLineItems    = errors.New("invalid_line_items")
	ErrInvalidClaimState   = errors.New("invalid_claim_state")
	ErrForbidden           = errors.New("forbidden")
	ErrPricingFailed       = errors.New("claim_pricing_failed")
	ErrAlreadySettled      = errors.New("claim_already_settled")
)
