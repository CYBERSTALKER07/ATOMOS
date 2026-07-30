package cashrecon

import (
	"context"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) SaveReconciliation(ctx context.Context, cr CashReconciliation) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts, err := r.saveReconciliationMutations(cr)
		if err != nil {
			return err
		}
		return txn.BufferWrite(muts)
	})
	return err
}

func (r *SpannerRepository) saveReconciliationMutations(cr CashReconciliation) ([]*spanner.Mutation, error) {
	crRow := map[string]interface{}{
		"ReconciliationId":  cr.ReconciliationId,
		"DriverId":          cr.DriverId,
		"RouteId":           spanner.NullString{StringVal: derefStr(cr.RouteId), Valid: cr.RouteId != nil},
		"ShiftDate":         civil.DateOf(cr.ShiftDate),
		"ExpectedCashMinor": cr.ExpectedCashMinor,
		"DeclaredCashMinor": cr.DeclaredCashMinor,
		"DifferenceMinor":   cr.DifferenceMinor,
		"Status":            string(cr.Status),
		"DriverNote":        spanner.NullString{StringVal: derefStr(cr.DriverNote), Valid: cr.DriverNote != nil},
		"FinanceNote":       spanner.NullString{StringVal: derefStr(cr.FinanceNote), Valid: cr.FinanceNote != nil},
		"CreatedAt":         cr.CreatedAt,
		"ResolvedAt":        spanner.NullTime{Time: derefTime(cr.ResolvedAt), Valid: cr.ResolvedAt != nil},
		"ResolvedBy":        spanner.NullString{StringVal: derefStr(cr.ResolvedBy), Valid: cr.ResolvedBy != nil},
	}

	return []*spanner.Mutation{spanner.InsertOrUpdateMap("CashReconciliations", crRow)}, nil
}

func (r *SpannerRepository) GetReconciliation(ctx context.Context, id string) (*CashReconciliation, error) {
	// Stub
	return nil, nil
}

func (r *SpannerRepository) ListReconciliationsByStatus(ctx context.Context, status ReconciliationStatus) ([]CashReconciliation, error) {
	// Stub
	return nil, nil
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
