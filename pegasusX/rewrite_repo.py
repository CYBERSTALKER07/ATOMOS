import re

with open("apps/backend-go/warehouse/repository_spanner.go", "r") as f:
    content = f.read()

repo_methods = """
	GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error)
	UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error
	GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error)
	UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error
	DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error
"""
content = content.replace("Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error", "Apply(ctx context.Context, mutate func() error, emit func(outbox.TxnBuffer) error) error\n" + repo_methods)

buffer_impl = """
func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	row := map[string]any{
		"EventId":       e.EventID,
		"AggregateType": e.AggregateType,
		"AggregateId":   e.AggregateID,
		"TopicName":     e.TopicName,
		"Payload":       e.Payload,
		"CreatedAt":     spanner.CommitTimestamp,
	}
	b.events = append(b.events, e)
	// Actually we are accumulating them to insert at the end.
	return nil
}
"""

in_memory_methods = """
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
content += "\n" + in_memory_methods

spanner_methods = """
func (r *SpannerRepository) GetInventoryList(ctx context.Context, warehouseID string) (map[string]InventoryRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, QuantityOnHand, UpdatedAt
			  FROM InventoryLevels
			  WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string]InventoryRow)
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var pid string
		var qty int64
		var updated time.Time
		if err := row.Columns(&pid, &qty, &updated); err == nil {
			out[pid] = InventoryRow{
				SKU:         pid,
				ProductName: pid,
				Quantity:    int(qty),
				UpdatedAt:   updated.Format(time.RFC3339),
			}
		}
	}
	return out, nil
}

func (r *SpannerRepository) UpdateInventoryQuantity(ctx context.Context, warehouseID, productID string, quantity int64, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("InventoryLevels", map[string]any{
				"WarehouseId":    warehouseID,
				"ProductId":      productID,
				"QuantityOnHand": quantity,
				"UpdatedAt":      spanner.CommitTimestamp,
			}),
		}
		
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}

func (r *SpannerRepository) GetLocks(ctx context.Context, warehouseID string) (map[string]DispatchLock, error) {
	stmt := spanner.Statement{
		SQL: `SELECT LockId, EntityType, EntityId, Reason, CreatedAt
			  FROM WarehouseDispatchLocks
			  WHERE WarehouseId = @wid`,
		Params: map[string]any{"wid": warehouseID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make(map[string]DispatchLock)
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var lockID, eType, eID, reason string
		var created time.Time
		if err := row.Columns(&lockID, &eType, &eID, &reason, &created); err == nil {
			out[lockID] = DispatchLock{
				LockID:     lockID,
				EntityType: eType,
				EntityID:   eID,
				Reason:     reason,
				CreatedAt:  created.Format(time.RFC3339),
			}
		}
	}
	return out, nil
}

func (r *SpannerRepository) UpsertLock(ctx context.Context, warehouseID string, lock DispatchLock, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("WarehouseDispatchLocks", map[string]any{
				"WarehouseId": warehouseID,
				"LockId":      lock.LockID,
				"EntityType":  lock.EntityType,
				"EntityId":    lock.EntityID,
				"Reason":      lock.Reason,
				"CreatedAt":   spanner.CommitTimestamp,
				"UpdatedAt":   spanner.CommitTimestamp,
			}),
		}
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}

func (r *SpannerRepository) DeleteLock(ctx context.Context, warehouseID, lockID string, emit func(outbox.TxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.Delete("WarehouseDispatchLocks", spanner.Key{warehouseID, lockID}),
		}
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		mutations = append(mutations, outboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	return err
}
"""

content += "\n" + spanner_methods

with open("apps/backend-go/warehouse/repository_spanner.go", "w") as f:
    f.write(content)

