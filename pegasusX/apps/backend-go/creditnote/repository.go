package creditnote

import (
	"context"
)

type Repository interface {
	SaveCreditNote(ctx context.Context, cn CreditNote) error
	GetCreditNote(ctx context.Context, id string) (*CreditNote, error)
	ListCreditNotesByOrder(ctx context.Context, orderId string) ([]CreditNote, error)
	SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask) error
}
