package creditnote

import "time"

type CreditNoteType string

const (
	CreditNoteTypeFullReturn  CreditNoteType = "FULL_RETURN"
	CreditNoteTypePartial     CreditNoteType = "PARTIAL"
	CreditNoteTypeClaim       CreditNoteType = "CLAIM"
	CreditNoteTypeBuyerReject CreditNoteType = "BUYER_REJECT"
	CreditNoteTypeManual      CreditNoteType = "MANUAL"
)

type CreditNoteStatus string

const (
	CreditNoteStatusDraft         CreditNoteStatus = "DRAFT"
	CreditNoteStatusIssued        CreditNoteStatus = "ISSUED"
	CreditNoteStatusFiscalPending CreditNoteStatus = "FISCAL_PENDING"
	CreditNoteStatusCompleted     CreditNoteStatus = "COMPLETED"
	CreditNoteStatusCancelled     CreditNoteStatus = "CANCELLED"
)

type ReverseLogisticsStatus string

const (
	ReverseTaskStatusOpen      ReverseLogisticsStatus = "OPEN"
	ReverseTaskStatusAssigned  ReverseLogisticsStatus = "ASSIGNED"
	ReverseTaskStatusInTransit ReverseLogisticsStatus = "IN_TRANSIT"
	ReverseTaskStatusReceived  ReverseLogisticsStatus = "RECEIVED"
	ReverseTaskStatusClosed    ReverseLogisticsStatus = "CLOSED"
)

type CreditNote struct {
	CreditNoteId    string
	OrderId         string
	Type            CreditNoteType
	Status          CreditNoteStatus
	ReasonCode      string
	ReasonText      *string
	TotalNetMinor   int64
	TotalVatMinor   int64
	TotalGrossMinor int64
	RegimeId        *string
	OriginalEhfId   *string
	CorrectiveEhfId *string
	CreatedBy       string
	CreatedAt       time.Time
	IssuedAt        *time.Time
	CompletedAt     *time.Time

	// Relations
	Lines []CreditNoteLine
}

type CreditNoteLine struct {
	CreditNoteId   string
	LineId         string
	OrderLineId    string
	Sku            string
	Qty            int64
	UnitNetMinor   int64
	VatRateBps     int64
	LineNetMinor   int64
	LineVatMinor   int64
	LineGrossMinor int64
}

type ReverseLogisticsTask struct {
	TaskId          string
	CreditNoteId    string
	OrderId         string
	Status          ReverseLogisticsStatus
	WarehouseId     *string
	DriverId        *string
	ExpectedQtyJson []byte // map[string]int64
	ReceivedQtyJson []byte // map[string]int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateManualCreditNoteRequest struct {
	OrderId    string
	Lines      []CreditNoteLineInput
	ReasonCode string
	ReasonText string
}

type CreditNoteLineInput struct {
	OrderLineId string
	Qty         int64
}
