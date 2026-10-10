package payout

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
)

type SettlementSliceStatus string

const (
	SliceStatusUnsettled SettlementSliceStatus = "UNSETTLED"
	SliceStatusBatched   SettlementSliceStatus = "BATCHED"
)

// InvoiceSettlementSlice represents an immutable slice of money captured from an order
// that is owed to the supplier, minus the platform commission.
type InvoiceSettlementSlice struct {
	SliceID         string                `spanner:"SliceId"`
	OrderID         string                `spanner:"OrderId"`
	SupplierID      string                `spanner:"SupplierId"`
	CapturedLegID   string                `spanner:"CapturedLegId"`
	GrossMinor      int64                 `spanner:"GrossMinor"`
	CommissionMinor int64                 `spanner:"CommissionMinor"`
	NetPayoutMinor  int64                 `spanner:"NetPayoutMinor"`
	Currency        string                `spanner:"Currency"`
	PayoutBatchID   spanner.NullString    `spanner:"PayoutBatchId"`
	Status          SettlementSliceStatus `spanner:"Status"`
	CreatedAt       time.Time             `spanner:"CreatedAt"`
}

// BuildSettlementSliceMutation creates a mutation for inserting a new slice.
// Called during the order checkout / capture transaction.
func BuildSettlementSliceMutation(slice InvoiceSettlementSlice) *spanner.Mutation {
	m, _ := spanner.InsertStruct("InvoiceSettlementSlices", slice)
	return m
}

// CommissionResolver defines how to calculate the commission for a captured amount.
type CommissionResolver interface {
	CommissionMinor(ctx context.Context, supplierID string, grossCapturedMinor int64, currency string) (int64, error)
}

// GenerateSettlementSlice calculates the commission and returns the SettlementSlice and its Spanner mutation.
func GenerateSettlementSlice(ctx context.Context, resolver CommissionResolver, newSliceID, orderID, supplierID, capturedLegID string, grossMinor int64, currency string, now time.Time) (InvoiceSettlementSlice, *spanner.Mutation, error) {
	commission, err := resolver.CommissionMinor(ctx, supplierID, grossMinor, currency)
	if err != nil {
		return InvoiceSettlementSlice{}, nil, err
	}
	
	slice := InvoiceSettlementSlice{
		SliceID:         newSliceID,
		OrderID:         orderID,
		SupplierID:      supplierID,
		CapturedLegID:   capturedLegID,
		GrossMinor:      grossMinor,
		CommissionMinor: commission,
		NetPayoutMinor:  grossMinor - commission,
		Currency:        currency,
		Status:          SliceStatusUnsettled,
		CreatedAt:       now,
	}
	return slice, BuildSettlementSliceMutation(slice), nil
}
