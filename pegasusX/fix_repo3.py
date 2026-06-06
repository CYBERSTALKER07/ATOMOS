import re

with open("apps/backend-go/warehouse/repository_spanner.go", "r") as f:
    content = f.read()

# Make sure all required inMemoryRepository stubs exist
stubs = """
func (r *inMemoryRepository) GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error) {
	return nil, nil
}
func (r *inMemoryRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *inMemoryRepository) GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error) {
	return nil, nil
}
func (r *inMemoryRepository) UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *inMemoryRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	return nil
}
"""

# if they don't exist, append to the Apply function
if "func (r *inMemoryRepository) GetInventoryList" not in content:
    content = content.replace("func (r *inMemoryRepository) Apply", stubs + "\nfunc (r *inMemoryRepository) Apply")

with open("apps/backend-go/warehouse/repository_spanner.go", "w") as f:
    f.write(content)

