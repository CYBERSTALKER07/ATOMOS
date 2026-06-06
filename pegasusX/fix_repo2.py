import re

with open("apps/backend-go/warehouse/repository_spanner.go", "r") as f:
    lines = f.readlines()

with open("apps/backend-go/warehouse/repository_spanner.go", "w") as f:
    for i, line in enumerate(lines):
        if 54 <= i <= 73:
            continue
        if "outboxMutations redeclared" in line:
            pass
        f.write(line)

with open("apps/backend-go/warehouse/repository_spanner.go", "r") as f:
    content = f.read()

content = content.replace('spanner.Insert("OutboxEvents", row)', 'spanner.InsertMap("OutboxEvents", row)')
content = content.replace("var created, updated time.Time", "var created time.Time")
content = content.replace("func (m *inMemoryRepository) DeleteLock", "func (m *inMemoryRepository) DeleteLock") # ensure it exists
if "func (m *inMemoryRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {" not in content:
    stub = """
func (m *inMemoryRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	return nil
}
"""
    content = content.replace("func (m *inMemoryRepository) UpsertLock", stub + "func (m *inMemoryRepository) UpsertLock")

with open("apps/backend-go/warehouse/repository_spanner.go", "w") as f:
    f.write(content)

