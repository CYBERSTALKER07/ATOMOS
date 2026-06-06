import re

with open("apps/backend-go/warehouse/repository_spanner.go", "r") as f:
    content = f.read()

content = content.replace(
"""		buf := &spannerTxnBuffer{mutations: mutations}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		return txn.BufferWrite(buf.mutations)""",
"""		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		
		// apply outbox/audits
		for _, e := range buf.events {
			mutations = append(mutations, outbox.InsertEventMutation(e))
		}
		for _, a := range buf.audits {
			mutations = append(mutations, outbox.InsertAuditMutation(a))
		}
		
		return txn.BufferWrite(mutations)""")

content = content.replace("&l.Reason, &created, &updated", "&l.Reason, &created")
content = content.replace("UpdatedAt     TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),", "")
content = content.replace("CreatedAt, UpdatedAt", "CreatedAt")
content = content.replace("l.UpdatedAt = updated.Format(time.RFC3339)", "")
content = content.replace("\"UpdatedAt\":   spanner.CommitTimestamp,", "")

# also add stub implementations to inMemoryRepository
stub = """
func (m *inMemoryRepository) GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error) {
	return nil, nil
}
func (m *inMemoryRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error) {
	return nil, nil
}
func (m *inMemoryRepository) UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *inMemoryRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	return nil
}
"""

# append stub before EOF or somewhere near inMemoryRepository
content = re.sub(
    r"(func \(m \*inMemoryRepository\) Apply)",
    stub + r"\1",
    content
)

with open("apps/backend-go/warehouse/repository_spanner.go", "w") as f:
    f.write(content)

