import re

with open("apps/backend-go/factory/service_test.go", "r") as f:
    factory_test = f.read()

factory_test = re.sub(r'func \(r \*factoryRepoSpy\) RunTx\(.*?\}\n', 
r'''func (r *factoryRepoSpy) RunTx(ctx context.Context, fn func(ctx context.Context, tx FactoryTx) error, emit func(outbox.TxnBuffer) error) error {
	r.RunTxCalls++
	return nil
}

func (r *factoryRepoSpy) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}
''', factory_test, count=1, flags=re.DOTALL)

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(factory_test)

with open("apps/backend-go/payload/service_test.go", "r") as f:
    payload_test = f.read()

payload_test = re.sub(r'func \(r \*payloadRepoSpy\) RunTx\(.*?\}\n', 
r'''func (r *payloadRepoSpy) RunTx(ctx context.Context, fn func(ctx context.Context, tx PayloadTx) error, emit func(outbox.TxnBuffer) error) error {
	r.RunTxCalls++
	return nil
}

func (r *payloadRepoSpy) Hydrate(ctx context.Context, supplierID string, s *Service) error {
	return nil
}
''', payload_test, count=1, flags=re.DOTALL)

with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(payload_test)

