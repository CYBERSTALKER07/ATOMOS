import re

with open("apps/backend-go/warehouse/transfers.go", "r") as f:
    content = f.read()

# Replace CreateTransfer logic
content = re.sub(
    r"transferID := uuid\.NewString\(\)\n\t_, err = s\.spannerClient\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\treturn txn\.BufferWrite\(\[\]\*spanner\.Mutation\{\n\t\t\tspanner\.InsertOrUpdateMap\(\"FactoryInternalTransfers\", map\[string\]any\{\n\t\t\t\t\"TransferId\":    transferID,\n\t\t\t\t\"FactoryId\":     factoryID,\n\t\t\t\t\"SupplierId\":    supplierID,\n\t\t\t\t\"State\":         \"APPROVED\",\n\t\t\t\t\"TotalVolumeVU\": req\.TotalVolumeVU,\n\t\t\t\t\"CreatedAt\":     spanner\.CommitTimestamp,\n\t\t\t\t\"UpdatedAt\":     spanner\.CommitTimestamp,\n\t\t\t\}\),\n\t\t\}\)\n\t\}\)",
    r"""transferID := uuid.NewString()
	err = s.repo.CreateTransfer(ctx, transferID, factoryID, supplierID, req.TotalVolumeVU, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":            events.EventWarehouseTransferCreated,
			"transfer_id":     transferID,
			"factory_id":      factoryID,
			"supplier_id":     supplierID,
			"total_volume_vu": req.TotalVolumeVU,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, whID, events.TopicMain, payload)
	})""",
    content
)

# Replace ForceReceive CreateTransfer logic
content = re.sub(
    r"transferID := uuid\.NewString\(\)\n\t_, err = s\.spannerClient\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\treturn txn\.BufferWrite\(\[\]\*spanner\.Mutation\{\n\t\t\tspanner\.InsertOrUpdateMap\(\"FactoryInternalTransfers\", map\[string\]any\{\n\t\t\t\t\"TransferId\":    transferID,\n\t\t\t\t\"FactoryId\":     factoryID,\n\t\t\t\t\"SupplierId\":    supplierID,\n\t\t\t\t\"State\":         \"RECEIVED\",\n\t\t\t\t\"TotalVolumeVU\": req\.TotalVolumeVU,\n\t\t\t\t\"CreatedAt\":     spanner\.CommitTimestamp,\n\t\t\t\t\"UpdatedAt\":     spanner\.CommitTimestamp,\n\t\t\t\}\),\n\t\t\}\)\n\t\}\)",
    r"""transferID := uuid.NewString()
	err = s.repo.CreateTransfer(ctx, transferID, factoryID, supplierID, req.TotalVolumeVU, nil)
	if err == nil {
		err = s.repo.UpdateTransferState(ctx, transferID, supplierID, "RECEIVED", func(txn outbox.TxnBuffer) error {
			payload := map[string]any{
				"type":            events.EventWarehouseTransferReceived,
				"transfer_id":     transferID,
				"factory_id":      factoryID,
				"supplier_id":     supplierID,
				"total_volume_vu": req.TotalVolumeVU,
				"timestamp":       time.Now().UTC().Format(time.RFC3339),
			}
			return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, whID, events.TopicMain, payload)
		})
	}""",
    content
)

# Replace receiveTransfer logic
content = re.sub(
    r"_, err := s\.spannerClient\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{[\s\S]+?return err\n\}",
    r"""err := s.repo.UpdateTransferState(ctx, transferID, ops.SupplierID, "RECEIVED", func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":        events.EventWarehouseTransferReceived,
			"transfer_id": transferID,
			"supplier_id": ops.SupplierID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		// assuming ops is warehouse scope, we might not have warehouse_id here, but we can emit for aggregate
		return outbox.EmitJSON(ctx, txn, events.AggregateWarehouse, transferID, events.TopicMain, payload)
	})
	return err
}""",
    content
)

with open("apps/backend-go/warehouse/transfers.go", "w") as f:
    f.write(content)

