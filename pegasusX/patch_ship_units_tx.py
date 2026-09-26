import re

with open("apps/backend-go/payload/repository.go", "r") as f:
    content = f.read()

content = content.replace("UpdateOrderAssignment(ctx context.Context, orderID, routeID, driverID string) error", "UpdateOrderAssignment(ctx context.Context, orderID, routeID, driverID string) error\n\tSaveShipUnits(ctx context.Context, units []ShipUnit) error")

content = content.replace("func (emptyPayloadTx) DeleteManifestOrder(context.Context, string, string) error { return nil }", "func (emptyPayloadTx) DeleteManifestOrder(context.Context, string, string) error { return nil }\nfunc (emptyPayloadTx) SaveShipUnits(context.Context, []ShipUnit) error { return nil }")

with open("apps/backend-go/payload/repository.go", "w") as f:
    f.write(content)


with open("apps/backend-go/payload/repository_spanner.go", "r") as f:
    content = f.read()

append_str = """
func (tx *spannerPayloadTx) SaveShipUnits(ctx context.Context, units []ShipUnit) error {
	if tx.txn == nil {
		return fmt.Errorf("missing transaction")
	}
	var muts []*spanner.Mutation
	for _, u := range units {
		muts = append(muts, spanner.InsertMap("ManifestShipUnits", map[string]any{
			"ManifestId": u.ManifestID,
			"ShipUnitId": u.ShipUnitID,
			"Sscc":       u.SSCC,
			"OrderId":    u.OrderID,
			"Sequence":   u.Sequence,
			"Gtin":       nullableStr(u.GTIN),
			"CreatedAt":  spanner.CommitTimestamp,
		}))
	}
	if len(muts) > 0 {
		return tx.txn.BufferWrite(muts)
	}
	return nil
}
"""
content += append_str

with open("apps/backend-go/payload/repository_spanner.go", "w") as f:
    f.write(content)
