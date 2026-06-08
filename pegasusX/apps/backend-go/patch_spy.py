import re

with open("apps/backend-go/factory/service_test.go", "r") as f:
    factory_test = f.read()

factory_test = re.sub(
r'''func \(r \*factoryRepoSpy\) Apply\(ctx context\.Context, mutate func\(\) error, emit func\(outbox\.TxnBuffer\) error, snapshotFn func\(\) \*PersistenceSnapshot\) error \{.*?\}''',
r'''func (r *factoryRepoSpy) RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error {
	r.ApplyCalls++
	return fn(ctx, nil)
}

func (r *factoryRepoSpy) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}''', factory_test, flags=re.DOTALL)

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(factory_test)


with open("apps/backend-go/payload/service_test.go", "r") as f:
    payload_test = f.read()

payload_test = re.sub(
r'''func \(r \*payloadRepoSpy\) Apply\(ctx context\.Context, mutate func\(\) error, emit func\(outbox\.TxnBuffer\) error, snapshotFn func\(\) \*PersistenceSnapshot\) error \{.*?\}''',
r'''func (r *payloadRepoSpy) RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error {
	r.ApplyCalls++
	return fn(ctx, nil)
}

func (r *payloadRepoSpy) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}''', payload_test, flags=re.DOTALL)

with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(payload_test)

