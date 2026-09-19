with open("apps/backend-go/driver/rescue.go", "r") as f:
    lines = f.readlines()

for i, line in enumerate(lines):
    if '_, err := spannerRepo.client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {' in line:
        if i < 40: # Only HandleRescueRequest
            lines.insert(i, "\tvar outWarehouseID string\n")
            break

for i, line in enumerate(lines):
    if 'warehouseID = wh.StringVal' in line:
        if i < 80: # Only HandleRescueRequest
            lines.insert(i+1, "\t\t\t\toutWarehouseID = warehouseID\n")
            break

with open("apps/backend-go/driver/rescue.go", "w") as f:
    f.writelines(lines)
