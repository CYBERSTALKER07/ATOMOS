import re

with open("apps/backend-go/payload/repository.go", "r") as f:
    content = f.read()

pattern1 = re.compile(r'SaveManifestOrder\(ctx context\.Context, mo ManifestOrder, seq int64\) error')
replacement1 = r'SaveManifestOrder(ctx context.Context, mo ManifestOrder, seq int64) error\n\tDeleteManifestOrder(ctx context.Context, manifestID, orderID string) error'
content = content.replace(pattern1.pattern, replacement1)
content = re.sub(r'SaveManifestOrder\(ctx context\.Context, mo ManifestOrder, seq int64\) error', replacement1, content)

pattern2 = re.compile(r'func \(emptyPayloadTx\) SaveManifestOrder\(context\.Context, ManifestOrder, int64\) error \{ return nil \}')
replacement2 = r'func (emptyPayloadTx) SaveManifestOrder(context.Context, ManifestOrder, int64) error { return nil }\nfunc (emptyPayloadTx) DeleteManifestOrder(context.Context, string, string) error { return nil }'
content = content.replace(pattern2.pattern, replacement2)
content = re.sub(r'func \(emptyPayloadTx\) SaveManifestOrder\(context\.Context, ManifestOrder, int64\) error \{ return nil \}', replacement2, content)

with open("apps/backend-go/payload/repository.go", "w") as f:
    f.write(content)

with open("apps/backend-go/payload/repository_spanner.go", "r") as f:
    content = f.read()

append_str = """
func (tx *spannerPayloadTx) DeleteManifestOrder(ctx context.Context, manifestID, orderID string) error {
	if tx.txn == nil {
		return fmt.Errorf("delete manifest order: missing transaction")
	}
	m := spanner.Delete("ManifestOrders", spanner.Key{manifestID, orderID})
	return tx.txn.BufferWrite([]*spanner.Mutation{m})
}
"""
content += append_str

with open("apps/backend-go/payload/repository_spanner.go", "w") as f:
    f.write(content)
