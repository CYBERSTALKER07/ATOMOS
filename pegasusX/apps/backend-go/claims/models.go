package claims

import (
	"time"
)

type ClaimStatus string

const (
	ClaimStatusPending  ClaimStatus = "PENDING"
	ClaimStatusApproved ClaimStatus = "APPROVED"
	ClaimStatusRejected ClaimStatus = "REJECTED"
)

type Claim struct {
	ClaimId             string      `spanner:"ClaimId" json:"claim_id"`
	OrderId             string      `spanner:"OrderId" json:"order_id"`
	RetailerId          string      `spanner:"RetailerId" json:"retailer_id"`
	SupplierId          string      `spanner:"SupplierId" json:"supplier_id"`
	Status              ClaimStatus `spanner:"Status" json:"status"`
	Reason              string      `spanner:"Reason" json:"reason"`
	RequestedAmountMinor int64      `spanner:"RequestedAmountMinor" json:"requested_amount_minor"`
	ApprovedAmountMinor *int64      `spanner:"ApprovedAmountMinor" json:"approved_amount_minor,omitempty"`
	Liability           *string     `spanner:"Liability" json:"liability,omitempty"`
	Notes               *string     `spanner:"Notes" json:"notes,omitempty"`
	CreatedAt           time.Time   `spanner:"CreatedAt" json:"created_at"`
	UpdatedAt           time.Time   `spanner:"UpdatedAt" json:"updated_at"`
}

type ClaimEvidence struct {
	EvidenceId  string    `spanner:"EvidenceId" json:"evidence_id"`
	ClaimId     string    `spanner:"ClaimId" json:"claim_id"`
	FileUrl     string    `spanner:"FileUrl" json:"file_url"`
	ContentType string    `spanner:"ContentType" json:"content_type,omitempty"`
	UploadedBy  string    `spanner:"UploadedBy" json:"uploaded_by"`
	CreatedAt   time.Time `spanner:"CreatedAt" json:"created_at"`
}
