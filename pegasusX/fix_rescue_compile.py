import re

with open("apps/backend-go/driver/rescue.go", "r") as f:
    content = f.read()

# Fix 1: HandleRescueRequest
# It is defined properly there: var outWarehouseID string is just before _, err := spannerRepo.client.ReadWriteTransaction

# Fix 2: HandleRescueRespond
pattern = re.compile(r'var supplierID, warehouseID string\n\t\t\tlookupStmt := spanner\.Statement\{\n\t\t\t\tSQL:    `SELECT SupplierId, HomeNodeId FROM Drivers WHERE DriverId = @id`,\n\t\t\t\tParams: map\[string\]any\{"id": driverID\},\n\t\t\t\}\n\t\t\titerLookup := txn\.Query\(ctx, lookupStmt\)\n\t\t\tdefer iterLookup\.Stop\(\)\n\t\t\trow, err := iterLookup\.Next\(\)\n\t\t\tif err == nil \{\n\t\t\t\tvar sp, wh spanner\.NullString\n\t\t\t\tif err := row\.Columns\(&sp, &wh\); err == nil \{\n\t\t\t\t\tsupplierID = sp\.StringVal\n\t\t\t\t\twarehouseID = wh\.StringVal\n\t\t\t\t\toutWarehouseID = warehouseID\n\t\t\t\t\}\n\t\t\t\}')

replacement = r"""var supplierID, warehouseID string
			lookupStmt := spanner.Statement{
				SQL:    `SELECT SupplierId, HomeNodeId FROM Drivers WHERE DriverId = @id`,
				Params: map[string]any{"id": driverID},
			}
			iterLookup := txn.Query(ctx, lookupStmt)
			defer iterLookup.Stop()
			row, err := iterLookup.Next()
			if err == nil {
				var sp, wh spanner.NullString
				if err := row.Columns(&sp, &wh); err == nil {
					supplierID = sp.StringVal
					warehouseID = wh.StringVal
				}
			}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/driver/rescue.go", "w") as f:
    f.write(content)
