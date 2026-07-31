package creditnote

import (
	"context"
)

type Repository interface {
	SaveCreditNote(ctx context.Context, cn CreditNote, eventType string) error
	GetCreditNote(ctx context.Context, id string) (*CreditNote, error)
	ListCreditNotesByOrder(ctx context.Context, orderId string) ([]CreditNote, error)
	ListBySupplier(ctx context.Context, supplierID string, status CreditNoteStatus, limit int) ([]CreditNote, error)
	GetDeliveredOrderLines(ctx context.Context, orderId string) ([]CreditNoteLine, error)
	GetClaimOrder(ctx context.Context, claimID string) (orderID string, amountMinor int64, ok bool, err error)
	SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask, eventType string) error
	GetReverseLogisticsTask(ctx context.Context, taskID string) (*ReverseLogisticsTask, error)
	ReceiveReverseLogisticsTask(ctx context.Context, taskID string, warehouseID string, receivedJSON []byte, actor string) error
}
