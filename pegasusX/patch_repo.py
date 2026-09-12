import re

with open("apps/backend-go/supplier/repository_spanner.go", "r") as f:
    content = f.read()

replacement = """
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		// Pre-flight check: prevent removing warehouses with active inventory or orders
		whIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT WarehouseId FROM Warehouses WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": supplierID},
		})
		var existingWarehouses []string
		if err := whIter.Do(func(row *spanner.Row) error {
			var wid string
			if err := row.Columns(&wid); err == nil {
				existingWarehouses = append(existingWarehouses, wid)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("query existing warehouses: %w", err)
		}

		newWarehouses := make(map[string]bool)
		for _, w := range topology.Warehouses {
			newWarehouses[w.WarehouseID] = true
		}

		var removedWarehouses []string
		for _, wid := range existingWarehouses {
			if !newWarehouses[wid] {
				removedWarehouses = append(removedWarehouses, wid)
			}
		}

		if len(removedWarehouses) > 0 {
			// Check inventory
			invIter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT WarehouseId FROM SupplierInventoryV2
				      WHERE SupplierId = @sid AND WarehouseId IN UNNEST(@removed)
				      AND (QuantityOnHand > 0 OR QuantityReserved > 0) LIMIT 1`,
				Params: map[string]any{
					"sid": supplierID,
					"removed": removedWarehouses,
				},
			})
			var hasInv bool
			if err := invIter.Do(func(*spanner.Row) error { hasInv = true; return nil }); err != nil {
				return fmt.Errorf("check removed warehouse inventory: %w", err)
			}
			if hasInv {
				return errors.New("cannot remove warehouse with active inventory")
			}

			// Check orders
			ordIter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT WarehouseId FROM Orders
				      WHERE SupplierId = @sid AND WarehouseId IN UNNEST(@removed)
				      AND Status NOT IN ('COMPLETED', 'CANCELLED', 'REJECTED') LIMIT 1`,
				Params: map[string]any{
					"sid": supplierID,
					"removed": removedWarehouses,
				},
			})
			var hasOrd bool
			if err := ordIter.Do(func(*spanner.Row) error { hasOrd = true; return nil }); err != nil {
				return fmt.Errorf("check removed warehouse orders: %w", err)
			}
			if hasOrd {
				return errors.New("cannot remove warehouse with active orders")
			}
		}

		if _, err := txn.Update(ctx, spanner.Statement{
"""

pattern = re.compile(r'\n\t_, err := r\.client\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\tbuf := &spannerTxnBuffer\{\}\n\t\tif emit != nil \{\n\t\t\tif err := emit\(buf\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n\t\t\}\n\n\t\tif _, err := txn\.Update\(ctx, spanner\.Statement\{', re.DOTALL)

content = pattern.sub(replacement, content)

with open("apps/backend-go/supplier/repository_spanner.go", "w") as f:
    f.write(content)

