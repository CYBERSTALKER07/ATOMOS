import re

with open('apps/backend-go/order/warehouse_ops.go', 'r') as f:
    content = f.read()

# Add spanner import
if '"cloud.google.com/go/spanner"' not in content:
    content = content.replace(
        '"github.com/go-chi/chi/v5"',
        '"cloud.google.com/go/spanner"\n\t"github.com/go-chi/chi/v5"'
    )

old_block = """	prevStatus := current.Status
	current.Status = nextStatus
	current.UpdatedAt = s.now()
	if clearAssignment {
		current.ManifestID = ""
		current.DriverID = ""
		current.VehicleID = ""
		current.RouteID = ""
	}

	actorID := ops.Subject
	if actorID == "" {
		actorID = ops.WarehouseID
	}

	err = s.repo.UpdateOrder(ctx, current, nil, func(txn outbox.TxnBuffer) error {"""

new_block = """	prevStatus := current.Status
	current.Status = nextStatus
	current.UpdatedAt = s.now()

	var prevManifestID string
	if clearAssignment {
		prevManifestID = current.ManifestID
		current.ManifestID = ""
		current.DriverID = ""
		current.VehicleID = ""
		current.RouteID = ""
	}

	actorID := ops.Subject
	if actorID == "" {
		actorID = ops.WarehouseID
	}

	inTxn := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if prevManifestID != "" {
			stmt := spanner.Statement{
				SQL: `UPDATE SupplierTruckManifests 
				      SET StopCount = StopCount - 1,
				          TotalVolumeVU = TotalVolumeVU - COALESCE((SELECT VolumeVU FROM ManifestOrders WHERE ManifestId = @mid AND OrderId = @oid), 0)
				      WHERE ManifestId = @mid`,
				Params: map[string]any{"mid": prevManifestID, "oid": current.OrderID},
			}
			if _, err := txn.Update(ctx, stmt); err != nil {
				return err
			}
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Delete("ManifestOrders", spanner.Key{prevManifestID, current.OrderID}),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	err = s.repo.UpdateOrderWithTxn(ctx, current, nil, inTxn, func(txn outbox.TxnBuffer) error {"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/order/warehouse_ops.go', 'w') as f:
        f.write(content)
    print("Patched warehouse_ops.go successfully.")
else:
    print("Could not find the target block to replace.")

