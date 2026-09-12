import re

with open('apps/backend-go/notifications/fcm.go', 'r') as f:
    content = f.read()

old_block = """	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := f.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Delete("DeviceTokens", spanner.Key{token})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {"""

new_block = """	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Use Apply (blind write) instead of ReadWriteTransaction to prevent transaction
	// lock storms during mass token purges (e.g. app uninstalls).
	_, err := f.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.Delete("DeviceTokens", spanner.Key{token}),
	})
	if err != nil {"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/notifications/fcm.go', 'w') as f:
        f.write(content)
    print("Patched purgeStaleToken successfully.")
else:
    print("Could not find the target block to replace.")

