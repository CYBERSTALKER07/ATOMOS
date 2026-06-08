import re

with open("apps/backend-go/factory/service_test.go", "r") as f:
    content = f.read()

content = content.replace("type factoryRepoSpy struct {", "type factoryRepoSpy struct {\n\tsvc *Service")

content = re.sub(r'func \(d \*dummyFactoryTx\) ListManifests\(.*?\}', 
r'''func (d *dummyFactoryTx) ListManifests(ctx context.Context) ([]ManifestRow, error) { return append([]ManifestRow(nil), d.svc.manifests...), nil }''', content)

content = re.sub(r'func \(d \*dummyFactoryTx\) ListTransfers\(.*?\}', 
r'''func (d *dummyFactoryTx) ListTransfers(ctx context.Context) ([]TransferRow, error) { return append([]TransferRow(nil), d.svc.transfers...), nil }''', content)

content = content.replace("type dummyFactoryTx struct{}", "type dummyFactoryTx struct{ svc *Service }")
content = content.replace("return fn(ctx, &dummyFactoryTx{})", "return fn(ctx, &dummyFactoryTx{svc: r.svc})")

# Fix all repo initializations to include svc
content = re.sub(r'repo := &factoryRepoSpy\{\}', r'repo := &factoryRepoSpy{}\n\t// svc will be assigned later', content)

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(content)


with open("apps/backend-go/payload/service_test.go", "r") as f:
    content = f.read()

content = content.replace("type payloadRepoSpy struct {", "type payloadRepoSpy struct {\n\tsvc *Service")

content = re.sub(r'func \(d \*dummyPayloadTx\) ListManifests\(.*?\}', 
r'''func (d *dummyPayloadTx) ListManifests(ctx context.Context) ([]ManifestRow, error) { return append([]ManifestRow(nil), d.svc.manifests...), nil }''', content)

content = re.sub(r'func \(d \*dummyPayloadTx\) ListManifestOrders\(.*?\}', 
r'''func (d *dummyPayloadTx) ListManifestOrders(ctx context.Context, mid string) ([]ManifestOrder, error) { return d.svc.manifestOrders[mid], nil }''', content)

content = re.sub(r'func \(d \*dummyPayloadTx\) ListExceptions\(.*?\}', 
r'''func (d *dummyPayloadTx) ListExceptions(ctx context.Context) ([]ManifestException, error) { return append([]ManifestException(nil), d.svc.exceptions...), nil }''', content)

content = content.replace("type dummyPayloadTx struct{}", "type dummyPayloadTx struct{ svc *Service }")
content = content.replace("return fn(ctx, &dummyPayloadTx{})", "return fn(ctx, &dummyPayloadTx{svc: r.svc})")

with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(content)

