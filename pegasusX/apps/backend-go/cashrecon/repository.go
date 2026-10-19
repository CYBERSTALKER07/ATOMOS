package cashrecon

import (
	"context"
	"time"
)

type Repository interface {
	SaveReconciliation(ctx context.Context, cr CashReconciliation, eventType string) error
	GetReconciliation(ctx context.Context, id string) (*CashReconciliation, error)
	ListReconciliationsByStatus(ctx context.Context, status ReconciliationStatus) ([]CashReconciliation, error)
	ListByDriver(ctx context.Context, driverID string, shiftDate time.Time) ([]CashReconciliation, error)
	ListBySupplier(ctx context.Context, supplierID string, status ReconciliationStatus, limit int) ([]CashReconciliation, error)
	ResolveDriverSupplierID(ctx context.Context, driverID string) (string, error)
}

// ExpectedCashComputer computes authoritative expected cash from payment legs.
type ExpectedCashComputer interface {
	ComputeExpectedCashMinor(ctx context.Context, driverID string, shiftDate time.Time, routeID *string) (int64, error)
}

// ReconciliationGate checks whether a driver may close their shift.
type ReconciliationGate interface {
	HasAcceptedReconciliation(ctx context.Context, driverID string, shiftDate time.Time) (bool, error)
}
