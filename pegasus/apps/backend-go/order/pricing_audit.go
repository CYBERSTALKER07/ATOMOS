package order

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// PricingAuditEntry captures a fee/pricing decision for dispute reconstruction.
type PricingAuditEntry struct {
	OrderID          string
	InvoiceID        string
	SupplierID       string
	ActorID          string
	ActorType        string
	FeePolicyVersion string
	SelectedTierKey  string
	FeeBasisPoints   int64
	FeeCapApplied    bool
	GrossAmount      int64
	FeeAmount        int64
	NetAmount        int64
	BeforeSnapshot   map[string]interface{}
	AfterSnapshot    map[string]interface{}
}

// InsertPricingAuditLog writes a PricingAuditLog row in the caller's transaction.
func InsertPricingAuditLog(txn *spanner.ReadWriteTransaction, entry PricingAuditEntry) error {
	beforeJSON, _ := json.Marshal(entry.BeforeSnapshot)
	afterJSON, _ := json.Marshal(entry.AfterSnapshot)
	return txn.BufferWrite([]*spanner.Mutation{
		spanner.Insert("PricingAuditLog",
			[]string{"AuditId", "OrderId", "InvoiceId", "SupplierId", "ActorId", "ActorType",
				"FeePolicyVersion", "SelectedTierKey", "FeeBasisPoints", "FeeCapApplied",
				"GrossAmount", "FeeAmount", "NetAmount", "BeforeSnapshot", "AfterSnapshot", "CreatedAt"},
			[]interface{}{
				uuid.New().String(),
				nullIfEmpty(entry.OrderID),
				nullIfEmpty(entry.InvoiceID),
				nullIfEmpty(entry.SupplierID),
				nullIfEmpty(entry.ActorID),
				nullIfEmpty(entry.ActorType),
				nullIfEmpty(entry.FeePolicyVersion),
				nullIfEmpty(entry.SelectedTierKey),
				entry.FeeBasisPoints,
				entry.FeeCapApplied,
				entry.GrossAmount,
				entry.FeeAmount,
				entry.NetAmount,
				spanner.NullJSON{Value: beforeJSON, Valid: len(beforeJSON) > 0 && string(beforeJSON) != "null"},
				spanner.NullJSON{Value: afterJSON, Valid: len(afterJSON) > 0 && string(afterJSON) != "null"},
				spanner.CommitTimestamp,
			},
		),
	})
}

func nullIfEmpty(s string) spanner.NullString {
	if s == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}

// CountSupplierOrdersToday returns non-terminal orders created today (Tashkent TZ).
func CountSupplierOrdersToday(ctx context.Context, client *spanner.Client, supplierID string, dayStart, dayEnd time.Time) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM Orders
		      WHERE SupplierId = @supplierId
		        AND CreatedAt >= @start
		        AND CreatedAt < @end
		        AND State NOT IN ('CANCELLED', 'REFUNDED')`,
		Params: map[string]interface{}{
			"supplierId": supplierID,
			"start":      dayStart,
			"end":        dayEnd,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// SupplierDailyOrderCap reads optional per-supplier daily intake cap (0 = unlimited).
func SupplierDailyOrderCap(ctx context.Context, client *spanner.Client, supplierID string) (int64, error) {
	row, err := client.Single().ReadRow(ctx, "Suppliers", spanner.Key{supplierID}, []string{"DailyOrderCap"})
	if err != nil {
		return 0, err
	}
	var cap spanner.NullInt64
	if err := row.Columns(&cap); err != nil {
		return 0, err
	}
	if cap.Valid && cap.Int64 > 0 {
		return cap.Int64, nil
	}
	return 0, nil
}

// OrderHasPartialLineItems returns true when any line item is PARTIAL_DELIVERED.
func OrderHasPartialLineItems(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (bool, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT 1 FROM OrderLineItems WHERE OrderId = @oid AND Status = 'PARTIAL_DELIVERED' LIMIT 1`,
		Params: map[string]interface{}{"oid": orderID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	return err == nil, err
}
