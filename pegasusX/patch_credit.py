import re

with open("apps/backend-go/credit/repository.go", "r") as f:
    content = f.read()

# Add ReserveOrderInTxn
new_func = """
func (r *SpannerRepository) ReserveOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, res OrderReservation) error {
	existing, err := txn.ReadRow(ctx, "OrderCreditReservations", spanner.Key{res.OrderID}, []string{"Status", "AmountMinor"})
	if err == nil {
		var st string
		var amt int64
		_ = existing.Columns(&st, &amt)
		if st == string(ReservationReserved) || st == string(ReservationConverted) {
			return nil // idempotent
		}
	} else if spanner.ErrCode(err) != 5 {
		return err
	}

	row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{res.RetailerID, res.SupplierID},
		[]string{"CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor", "Status", "Version"})
	if err != nil {
		if spanner.ErrCode(err) == 5 {
			return ErrProfileNotFound
		}
		return err
	}
	var limit, balance, reserved, version int64
	var st string
	_ = row.Columns(&limit, &balance, &reserved, &st, &version)
	
	if st != string(StatusActive) {
		return ErrProfileNotActive
	}
	if limit-(balance+reserved) < res.AmountMinor {
		return ErrInsufficientCredit
	}
	
	muts := []*spanner.Mutation{
		spanner.InsertOrUpdateMap("OrderCreditReservations", map[string]any{
			"OrderId":     res.OrderID,
			"RetailerId":  res.RetailerID,
			"SupplierId":  res.SupplierID,
			"AmountMinor": res.AmountMinor,
			"Status":      string(ReservationReserved),
			"CreatedAt":   spanner.CommitTimestamp,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
		spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId":    res.RetailerID,
			"SupplierId":    res.SupplierID,
			"ReservedMinor": reserved + res.AmountMinor,
			"Version":       version + 1,
			"UpdatedAt":     spanner.CommitTimestamp,
		}),
	}
	return txn.BufferWrite(muts)
}
"""

if "ReserveOrderInTxn" not in content:
    content = content + new_func

with open("apps/backend-go/credit/repository.go", "w") as f:
    f.write(content)

with open("apps/backend-go/credit/service.go", "r") as f:
    content = f.read()

new_func = """
func (s *Service) ReserveOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, supplierID, orderID string, amountMinor int64) error {
	res := OrderReservation{
		OrderID:     orderID,
		RetailerID:  retailerID,
		SupplierID:  supplierID,
		AmountMinor: amountMinor,
		Status:      ReservationReserved,
	}
	if r, ok := s.repo.(*SpannerRepository); ok {
		return r.ReserveOrderInTxn(ctx, txn, res)
	}
	return errors.New("not supported")
}
"""

if "ReserveOrderInTxn" not in content:
    content = content + new_func

with open("apps/backend-go/credit/service.go", "w") as f:
    f.write(content)

