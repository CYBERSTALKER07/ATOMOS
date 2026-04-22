# Spanner Discipline — Cloud Spanner Access Patterns & Traps

## Description
Prevents Spanner misuse: silent mutation failures, full-scan queries, race conditions from read-then-write, and dead-table confusion. Activated when writing or reviewing any Go code that reads from or writes to Spanner, modifies DDL, or adds queries.

## Trigger Keywords
spanner, ReadWriteTransaction, Apply, mutation, query, index, DDL, schema, migration, Single(), row.Columns, InsertOrUpdate, BufferWrite

## Anti-Pattern Catalog

### 1. Apply for Multi-Row Mutations (SILENT FAILURE)
```go
// WRONG — Apply does not retry on Spanner abort
_, err := spannerClient.Apply(ctx, []*spanner.Mutation{
    spanner.InsertOrUpdate("Factories", cols, vals1),
    spanner.InsertOrUpdate("LoadingBays", cols, vals2),
    spanner.InsertOrUpdate("FactoryStaff", cols, vals3),
})

// RIGHT — ReadWriteTransaction retries on abort
_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    return txn.BufferWrite([]*spanner.Mutation{
        spanner.InsertOrUpdate("Factories", cols, vals1),
        spanner.InsertOrUpdate("LoadingBays", cols, vals2),
        spanner.InsertOrUpdate("FactoryStaff", cols, vals3),
    })
})
```
**Real violations**: `factory/crud.go` L309, L576 — multi-row entity creation uses `Apply`.

**Rule**: `Apply` does not retry on Spanner transaction abort. Under contention, mutations silently fail with a transient error that's not retried. Use `ReadWriteTransaction` for ALL multi-row mutations. `Apply` is acceptable ONLY for single-row idempotent updates (e.g., heartbeat timestamp).

### 2. Read-Then-Write Outside Transaction (RACE)
```go
// WRONG — another request can change stock between read and write
row, _ := spannerClient.Single().ReadRow(ctx, "Inventory", key, []string{"Stock"})
row.Columns(&stock)
newStock := stock - quantity
spannerClient.Apply(ctx, []*spanner.Mutation{
    spanner.Update("Inventory", []string{"ProductId", "Stock"}, []interface{}{productId, newStock}),
})

// RIGHT — atomic read-decide-write inside transaction
_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    row, err := txn.ReadRow(ctx, "Inventory", key, []string{"Stock"})
    if err != nil { return err }
    var stock int64
    if err := row.Columns(&stock); err != nil { return err }
    if stock < quantity { return ErrInsufficientStock }
    return txn.BufferWrite([]*spanner.Mutation{
        spanner.Update("Inventory", []string{"ProductId", "Stock"}, []interface{}{productId, stock - quantity}),
    })
})
```

**Preferred**: SQL-level atomic operations when possible:
```sql
UPDATE Inventory SET Stock = Stock - @quantity WHERE ProductId = @id AND Stock >= @quantity
```

### 3. Nullable Critical Columns (SILENT DOWNSTREAM FAILURE)
Known nullable columns that handlers often assume are NOT NULL:

| Column | Table | Impact if NULL |
|---|---|---|
| `Amount` | `Orders` | Ledger, payment session, treasurer crash or produce garbage |
| `SupplierId` | `Drivers` | Scope checks fail, driver appears unowned |
| `Phone` | `Drivers` | Auth via phone+PIN fails silently |
| `Latitude/Longitude` | `Retailers` | H3 indexing, geofence completion gate, proximity all fail |
| `Phone` | `Retailers` | SMS notifications silently skip |

**Rule**: Always guard nullable columns with explicit NULL checks:
```go
var amount spanner.NullInt64
if err := row.Columns(&amount); err != nil { return err }
if !amount.Valid {
    return fmt.Errorf("order %s: amount is NULL", orderID)
}
orderAmount := amount.Int64
```

### 4. Products vs SupplierProducts (DEAD TABLE)
**Finding**: `Products` table (`schema/spanner.ddl` L159) has no `SupplierId`, no indexes, uses `NUMERIC` for price, and lacks `CreatedAt`. The active catalog is `SupplierProducts`.

**Rule**: NEVER use `Products` table. Always use `SupplierProducts` for catalog operations. If you see code referencing `Products`, it's dead code or a bug.

### 5. Missing Index = Full Scan
Before writing a `WHERE` clause, verify the column has an index:
```sql
-- WRONG — no index on LedgerEntries.AccountId → full scan
SELECT * FROM LedgerEntries WHERE AccountId = @accountId

-- Check schema/spanner.ddl for: CREATE INDEX Idx_LedgerEntries_ByAccountId ON LedgerEntries(AccountId)
-- If missing → add the index in a migration BEFORE deploying the query
```

**Known missing indexes**:
- `LedgerEntries.AccountId` — reconciliation by account
- `MasterInvoices.State` — finding pending invoices
- `Orders.H3Cell` — doctrine references `Idx_Orders_ByH3Cell` but no H3Cell column exists on Orders

**Rule**: Every `WHERE` filter MUST hit an index. Add secondary indexes in `schema/` migrations, not inline. No full-scan queries in production.

### 6. SupplierInventory PK Trap
**Finding**: `SupplierInventory` PK is `ProductId` only — not `(SupplierId, ProductId)`. Two suppliers cannot stock the same product.

**Rule**: Be aware of this constraint. If your feature requires per-supplier inventory, this is a schema migration, not a code fix.

### 7. Mutation Cap
**Rule**: Spanner hard limit is 20,000 cell mutations per transaction. Practical ceiling: 1,000 row mutations. For bulk operations:
```go
// Batch into multiple transactions
const batchSize = 500
for i := 0; i < len(mutations); i += batchSize {
    end := min(i+batchSize, len(mutations))
    _, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        return txn.BufferWrite(mutations[i:end])
    })
    if err != nil { return err }
}
```

### 8. Stale Reads for Read-Only Queries
```go
// Dashboard / analytics / list view — use stale read
iter := spannerClient.Single().
    WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).
    Query(ctx, stmt)

// Mutation precondition check — use strong read INSIDE transaction
_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    row, err := txn.ReadRow(ctx, "Orders", key, cols) // strong read
    // ...
})
```

**Rule**: Strong reads (`Single().Query()`) acquire locks and compete with writes. Use stale reads (`ExactStaleness(15s)`) for dashboards, list views, and analytics. Reserve strong reads for mutation precondition checks inside `ReadWriteTransaction`.

### 9. row.Columns Error Must Not Be Swallowed
```go
// WRONG — partial/corrupted data silently used
_ = row.Columns(&id, &name, &amount)

// RIGHT
if err := row.Columns(&id, &name, &amount); err != nil {
    return fmt.Errorf("scan order %s: %w", orderID, err)
}
```
**Real violations**: `analytics/retailer.go` L121, `kafka/notification_dispatcher.go` L443.

### 10. Outbox Emission Inside Transaction
Every entity mutation that needs a downstream event:
```go
_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    // 1. Write business data
    if err := txn.BufferWrite(businessMutations); err != nil {
        return err
    }
    // 2. Emit event (written to OutboxEvents table in SAME transaction)
    return outbox.EmitJSON(txn, "Order", orderID, kafka.TopicMain,
        kafka.OrderCreatedEvent{...},
        telemetry.TraceIDFromContext(ctx),
    )
})
// 3. Post-commit: invalidate cache
if err == nil {
    cache.Invalidate(ctx, "order:"+orderID)
}
```

**Rule**: `outbox.EmitJSON` goes INSIDE the `ReadWriteTransaction`. `cache.Invalidate` goes AFTER the commit. Never reverse this order.

## Canonical Verification
After writing Spanner code, verify:
1. No `Apply` for multi-row mutations
2. No read-then-write outside `ReadWriteTransaction`
3. Every `WHERE` clause hits an index (check `schema/spanner.ddl`)
4. Every `row.Columns` error is handled (not `_ =`)
5. Nullable columns are guarded with `spanner.NullXxx` types
6. Using `SupplierProducts`, never `Products`
7. Bulk mutations batched to ≤1000 per transaction
8. `outbox.EmitJSON` inside txn, `cache.Invalidate` after commit
9. Dashboard reads use stale reads (`ExactStaleness`)

## Cross-References
- `intrusions.md` §3 — Spanner Discipline
- `gemini-instructions.md` §4 High-Performance Code — Spanner Access Patterns
- `gemini-instructions.md` Enterprise Algorithm Patterns §2 — Transactional Outbox
- `.agents/skills/test-with-spanner/SKILL.md` — Running tests with Spanner emulator
