import re

with open("apps/backend-go/factory/service_test.go", "r") as f:
    factory_test = f.read()

factory_test = factory_test + """
type dummyFactoryTx struct{}
func (d *dummyFactoryTx) ListManifests(ctx context.Context) ([]ManifestRow, error) { return nil, nil }
func (d *dummyFactoryTx) SaveManifest(ctx context.Context, m ManifestRow) error { return nil }
func (d *dummyFactoryTx) ListTransfers(ctx context.Context) ([]TransferRow, error) { return nil, nil }
func (d *dummyFactoryTx) SaveTransfer(ctx context.Context, t TransferRow) error { return nil }
"""

factory_test = factory_test.replace("return fn(ctx, nil)", "return fn(ctx, &dummyFactoryTx{})")

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(factory_test)


with open("apps/backend-go/payload/service_test.go", "r") as f:
    payload_test = f.read()

payload_test = payload_test + """
type dummyPayloadTx struct{}
func (d *dummyPayloadTx) ListManifests(ctx context.Context) ([]ManifestRow, error) { return nil, nil }
func (d *dummyPayloadTx) SaveManifest(ctx context.Context, m ManifestRow) error { return nil }
func (d *dummyPayloadTx) ListManifestOrders(ctx context.Context, mid string) ([]ManifestOrder, error) { return nil, nil }
func (d *dummyPayloadTx) SaveManifestOrder(ctx context.Context, mo ManifestOrder, seq int64) error { return nil }
func (d *dummyPayloadTx) ListExceptions(ctx context.Context) ([]ManifestException, error) { return nil, nil }
func (d *dummyPayloadTx) SaveException(ctx context.Context, e ManifestException) error { return nil }
"""

payload_test = payload_test.replace("return fn(ctx, nil)", "return fn(ctx, &dummyPayloadTx{})")

with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(payload_test)

