package cashrecon

import (
	"context"
)

type Repository interface {
	SaveReconciliation(ctx context.Context, cr CashReconciliation) error
	GetReconciliation(ctx context.Context, id string) (*CashReconciliation, error)
	ListReconciliationsByStatus(ctx context.Context, status ReconciliationStatus) ([]CashReconciliation, error)
}
