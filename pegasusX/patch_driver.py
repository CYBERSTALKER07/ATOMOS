import re

with open("apps/backend-go/driver/repository.go", "r") as f:
    content = f.read()

content = content.replace("UpdateVehicle(ctx context.Context, v Vehicle, emit func(outbox.TxnBuffer) error) error", "UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error")

with open("apps/backend-go/driver/repository.go", "w") as f:
    f.write(content)


with open("apps/backend-go/driver/repository_crud.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*SpannerRepository\) UpdateVehicle\(ctx context\.Context, v Vehicle, emit func\(outbox\.TxnBuffer\) error\) error \{\n\tv\.UpdatedAt = spanner\.CommitTimestamp\n\tm, err := spanner\.UpdateStruct\("Vehicles", v\)\n\tif err != nil \{\n\t\treturn err\n\t\}\n\t_, err = r\.client\.ReadWriteTransaction\(ctx, func\(ctx context\.Context, txn \*spanner\.ReadWriteTransaction\) error \{\n\t\tmuts := \[\]\*spanner\.Mutation\{m\}\n\t\tif emit != nil \{\n\t\t\tbuf := &spannerTxnBuffer\{\}\n\t\t\tif err := emit\(buf\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n\t\t\tmuts = append\(muts, outboxMutations\(buf\.events\)\.\.\.\)\n\t\t\}\n\t\treturn txn\.BufferWrite\(muts\)\n\t\}\)\n\treturn err\n\}')

replacement = r"""func (r *SpannerRepository) UpdateVehicle(ctx context.Context, vehicleID string, updates map[string]any, emit func(outbox.TxnBuffer) error) error {
	updates["VehicleId"] = vehicleID
	updates["UpdatedAt"] = spanner.CommitTimestamp
	m := spanner.UpdateMap("Vehicles", updates)
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{m}
		if emit != nil {
			buf := &spannerTxnBuffer{}
			if err := emit(buf); err != nil {
				return err
			}
			muts = append(muts, outboxMutations(buf.events)...)
		}
		return txn.BufferWrite(muts)
	})
	return err
}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/driver/repository_crud.go", "w") as f:
    f.write(content)

