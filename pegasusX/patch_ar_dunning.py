import re

with open("apps/backend-go/ar/service.go", "r") as f:
    content = f.read()

pattern = re.compile(r'var sid, rid string\n\t\tif row, err := txn\.ReadRow\(ctx, "ArInvoices", spanner\.Key\{invoiceID\}, \[\]string\{"SupplierId", "RetailerId"\}\); err == nil \{\n\t\t\t_ = row\.Columns\(&sid, &rid\)\n\t\t\}')

replacement = r"""var sid, rid, status string
		var currentVersion int64
		row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{invoiceID}, []string{"SupplierId", "RetailerId", "Version", "Status"})
		if err != nil {
			return err
		}
		if err := row.Columns(&sid, &rid, &currentVersion, &status); err != nil {
			return err
		}
		if currentVersion != version || status == StatusPaid || status == StatusVoid {
			// Version mismatch or invoice no longer open; silently abort this dunning step
			return nil
		}"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/ar/service.go", "w") as f:
    f.write(content)

