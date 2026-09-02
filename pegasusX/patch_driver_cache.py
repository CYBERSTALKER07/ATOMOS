import re

with open("apps/backend-go/driver/service_test.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(c \*driverCacheBackendSpy\) Set\(ctx context\.Context, key string, value \[\]byte, ttl time\.Duration\) error \{\n\tc\.mu\.Lock\(\)\n\tdefer c\.mu\.Unlock\(\)\n\tc\.kvs\[key\] = value\n\treturn nil\n\}')
replacement = r"""func (c *driverCacheBackendSpy) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kvs[key] = value
	return nil
}

func (c *driverCacheBackendSpy) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (c *driverCacheBackendSpy) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}"""

content = pattern.sub(replacement, content)
with open("apps/backend-go/driver/service_test.go", "w") as f:
    f.write(content)
