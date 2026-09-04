import re

with open("apps/backend-go/ar/service.go", "r") as f:
    content = f.read()

pattern = re.compile(r'_\, err := r\.client\.ReadWriteTransaction\(ctx\, func\(ctx context\.Context\, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\t\tif err := txn\.BufferWrite\(\[\]\*spanner\.Mutation\{')

replacement = r"""_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var currentVersion int64
			var status string
			row, err := txn.ReadRow(ctx, "ArInvoices", spanner.Key{rec.id}, []string{"Version", "Status"})
			if err != nil {
				return err
			}
			if err := row.Columns(&currentVersion, &status); err != nil {
				return err
			}
			if currentVersion != rec.ver || status == StatusPaid || status == StatusVoid {
				// Abort silently: concurrent payment modified it since the snapshot read
				return nil
			}

			if err := txn.BufferWrite([]*spanner.Mutation{"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/ar/service.go", "w") as f:
    f.write(content)

