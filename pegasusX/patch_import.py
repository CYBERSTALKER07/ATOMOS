import re

with open("apps/backend-go/supplier/import_sessions_apply.go", "r") as f:
    content = f.read()

replacement = """
			var existingReserved int64 = 0
			invStmt := spanner.Statement{
				SQL: `SELECT 
						(SELECT InventoryId FROM InventoryLevels WHERE WarehouseId = @warehouseId AND ProductId = @productId LIMIT 1),
						(SELECT QuantityReserved FROM SupplierInventoryV2 WHERE SupplierId = @supplierId AND WarehouseId = @warehouseId AND ProductId = @productId LIMIT 1)
				`,
				Params: map[string]any{
					"supplierId":  supplierID,
					"warehouseId": warehouseID,
					"productId":   productID,
				},
			}
			invIter := txn.Query(ctx, invStmt)
			invRow, invErr := invIter.Next()
			invIter.Stop()
			if invErr == nil {
				var existingInvID spanner.NullString
				var existingRes spanner.NullInt64
				if err := invRow.Columns(&existingInvID, &existingRes); err != nil {
					return fmt.Errorf("parse inventory state: %w", err)
				}
				if existingInvID.Valid {
					existingInventoryID = existingInvID.StringVal
				}
				if existingRes.Valid {
					existingReserved = existingRes.Int64
				}
			} else if invErr != iterator.Done {
				return fmt.Errorf("lookup inventory level: %w", invErr)
			}
			if existingInventoryID != "" {
"""

pattern = re.compile(r'\n\t\t\tvar existingReserved int64 = 0\n\t\t\tinvStmt := spanner\.Statement\{.*?\n\t\t\tif existingInventoryID != "" \{', re.DOTALL)
content = pattern.sub(replacement, content)

with open("apps/backend-go/supplier/import_sessions_apply.go", "w") as f:
    f.write(content)

