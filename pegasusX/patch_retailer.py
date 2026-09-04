import re

with open("apps/backend-go/retailer/store_stock.go", "r") as f:
    content = f.read()

# Add applyDeltaInTxn
new_func = """
func (s *Service) applyDeltaInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, locationID, bin, sku string, delta int64, moveType, refType, refID, actor, note string) error {
	var onHand, reserved int64
	row, err := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{locationID, bin, sku},
		[]string{"OnHand", "Reserved"})
	if err != nil && !isNotFound(err) {
		return err
	}
	if err == nil {
		_ = row.Columns(&onHand, &reserved)
	}
	next := onHand + delta
	if next < 0 {
		return errors.New("insufficient_stock")
	}
	muts := []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
			"LocationId": locationID,
			"StockBin":   bin,
			"Sku":        sku,
			"RetailerId": retailerID,
			"OnHand":     next,
			"Reserved":   reserved,
			"UpdatedAt":  spanner.CommitTimestamp,
		}),
		spanner.InsertMap("RetailerStockMovements", map[string]any{
			"MovementId":   s.newID(),
			"RetailerId":   retailerID,
			"LocationId":   locationID,
			"StockBin":     bin,
			"Sku":          sku,
			"Qty":          delta,
			"MovementType": moveType,
			"RefType":      nullableStr(refType),
			"RefId":        nullableStr(refID),
			"ActorUserId":  nullableStr(actor),
			"Note":         nullableStr(note),
			"CreatedAt":    spanner.CommitTimestamp,
		}),
	}
	return txn.BufferWrite(muts)
}
"""
if "applyDeltaInTxn" not in content:
    content = content + new_func

# Replace applyDelta's spanner part to use applyDeltaInTxn
pattern = re.compile(r'_, err := s\.spannerClient\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\t// Read current.*?\n\t\t\}\)\n\t\treturn txn\.BufferWrite\(muts\)\n\t\}\)', re.DOTALL)
replacement = """_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return s.applyDeltaInTxn(ctx, txn, retailerID, locationID, bin, sku, delta, moveType, refType, refID, actor, note)
	})"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/retailer/store_stock.go", "w") as f:
    f.write(content)
