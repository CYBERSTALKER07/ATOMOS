import re

with open("apps/backend-go/payload/service_test.go", "r") as f:
    content = f.read()

pattern1 = re.compile(r'func \(c \*payloadCacheBackendSpy\) Set\(ctx context\.Context, key string, value \[\]byte, ttl time\.Duration\) error \{\n\tc\.mu\.Lock\(\)\n\tdefer c\.mu\.Unlock\(\)\n\tc\.kvs\[key\] = value\n\treturn nil\n\}')
replacement1 = r"""func (c *payloadCacheBackendSpy) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kvs[key] = value
	return nil
}

func (c *payloadCacheBackendSpy) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (c *payloadCacheBackendSpy) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}"""
content = pattern1.sub(replacement1, content)

pattern2 = re.compile(r'func \(tx \*dummyPayloadTx\) UpdateOrderAssignment\(ctx context\.Context, orderID, routeID, driverID string\) error \{\n\treturn nil\n\}')
replacement2 = r"""func (tx *dummyPayloadTx) UpdateOrderAssignment(ctx context.Context, orderID, routeID, driverID string) error {
	return nil
}

func (tx *dummyPayloadTx) DeleteManifestOrder(ctx context.Context, manifestID, orderID string) error {
	return nil
}"""
content = pattern2.sub(replacement2, content)

with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(content)
