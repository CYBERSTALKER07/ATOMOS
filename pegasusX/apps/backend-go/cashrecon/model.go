package cashrecon

import (
	"time"
)

type ReconciliationStatus string

const (
	ReconciliationStatusPending  ReconciliationStatus = "PENDING"
	ReconciliationStatusAccepted ReconciliationStatus = "ACCEPTED"
	ReconciliationStatusWriteOff ReconciliationStatus = "WRITE_OFF"
	ReconciliationStatusDisputed ReconciliationStatus = "DISPUTED"
)

type CashReconciliation struct {
	ReconciliationId  string
	SupplierId        string
	DriverId          string
	RouteId           *string
	ShiftDate         time.Time // stored as DATE in Spanner
	ExpectedCashMinor int64
	DeclaredCashMinor int64
	DifferenceMinor   int64
	Status            ReconciliationStatus
	DriverNote        *string
	FinanceNote       *string
	CreatedAt         time.Time
	ResolvedAt        *time.Time
	ResolvedBy        *string
}

type SubmitReconciliationRequest struct {
	SupplierId        string
	DriverId          string
	RouteId           *string
	ShiftDate         time.Time
	ExpectedCashMinor int64
	DeclaredCashMinor int64
	DriverNote        *string
}

