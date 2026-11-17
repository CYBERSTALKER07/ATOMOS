package esf

import (
	"context"
	"time"
)

// Repository defines durable persistence for ESF documents, status timelines, VAT tie-out, and IKPU audits.
type Repository interface {
	LoadOrderFiscalSource(ctx context.Context, entityID, orderID string) (OrderFiscalSource, error)
	SaveDocumentWithHistory(ctx context.Context, doc EsfDocument, hist EsfStatusHistoryEntry, eventType string, glVatReversal *VATReversalSpec) error
	UpdateDocumentStatusWithHistory(ctx context.Context, entityID, esfID, expectedPrevStatus, nextStatus string, updates StatusUpdateFields, hist EsfStatusHistoryEntry, eventType string, glVatReversal *VATReversalSpec) (EsfDocument, error)
	GetDocument(ctx context.Context, entityID, esfID string) (EsfDocument, []EsfStatusHistoryEntry, error)
	FindLatestByOrderOrOperatorRef(ctx context.Context, entityID, orderID, operatorRef string) (*EsfDocument, error)
	ListDocuments(ctx context.Context, filter ListFilter) ([]EsfDocument, error)
	GetVATSummary(ctx context.Context, entityID, periodID string) (VATSummaryReport, error)
	ListMissingIKPU(ctx context.Context, supplierID string) ([]MissingIKPUItem, error)
}

// StatusUpdateFields holds optional column mutations applied during status update or resend.
type StatusUpdateFields struct {
	Operator         string
	OperatorRef      string
	SignedPayloadURI string
	PayloadHash      string
	FailureReason    *string
	IssuedAt         *time.Time
	AcceptedAt       *time.Time
	UpdatedAt        time.Time
}

// VATReversalSpec requests an atomic GL DR TAX:VAT:OUTPUT reversal in the same transaction as esf.rejected.
type VATReversalSpec struct {
	EntityID       string
	EsfID          string
	OrderID        string
	RetailerID     string
	Currency       string
	VatMinor       int64
	ActorID        string
	IdempotencyKey string
	OccurredAt     time.Time
	GlEntryIDOut   *string
}
