package creditnote

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) SaveCreditNote(ctx context.Context, cn CreditNote) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts, err := r.saveCreditNoteMutations(cn)
		if err != nil {
			return err
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) saveCreditNoteMutations(cn CreditNote) ([]*spanner.Mutation, error) {
	var muts []*spanner.Mutation

	cnRow := map[string]interface{}{
		"CreditNoteId":    cn.CreditNoteId,
		"OrderId":         cn.OrderId,
		"Type":            string(cn.Type),
		"Status":          string(cn.Status),
		"ReasonCode":      cn.ReasonCode,
		"ReasonText":      spanner.NullString{StringVal: derefStr(cn.ReasonText), Valid: cn.ReasonText != nil},
		"TotalNetMinor":   cn.TotalNetMinor,
		"TotalVatMinor":   cn.TotalVatMinor,
		"TotalGrossMinor": cn.TotalGrossMinor,
		"RegimeId":        spanner.NullString{StringVal: derefStr(cn.RegimeId), Valid: cn.RegimeId != nil},
		"OriginalEhfId":   spanner.NullString{StringVal: derefStr(cn.OriginalEhfId), Valid: cn.OriginalEhfId != nil},
		"CorrectiveEhfId": spanner.NullString{StringVal: derefStr(cn.CorrectiveEhfId), Valid: cn.CorrectiveEhfId != nil},
		"CreatedBy":       cn.CreatedBy,
		"CreatedAt":       cn.CreatedAt,
		"IssuedAt":        spanner.NullTime{Time: derefTime(cn.IssuedAt), Valid: cn.IssuedAt != nil},
		"CompletedAt":     spanner.NullTime{Time: derefTime(cn.CompletedAt), Valid: cn.CompletedAt != nil},
	}
	muts = append(muts, spanner.InsertOrUpdateMap("CreditNotes", cnRow))

	for _, line := range cn.Lines {
		m, err := spanner.InsertOrUpdateStruct("CreditNoteLines", line)
		if err != nil {
			return nil, err
		}
		muts = append(muts, m)
	}

	return muts, nil
}

func (r *SpannerRepository) GetCreditNote(ctx context.Context, id string) (*CreditNote, error) {
	// Stub
	return nil, nil
}

func (r *SpannerRepository) ListCreditNotesByOrder(ctx context.Context, orderId string) ([]CreditNote, error) {
	// Stub
	return nil, nil
}

func (r *SpannerRepository) SaveReverseLogisticsTask(ctx context.Context, task ReverseLogisticsTask) error {
	m, err := spanner.InsertOrUpdateStruct("ReverseLogisticsTasks", task)
	if err != nil {
		return err
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
