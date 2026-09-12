with open("apps/backend-go/factory/service_test.go", "r") as f:
    content = f.read()

replacement = """func (b *factoryCacheBackendSpy) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (b *factoryCacheBackendSpy) DecrBy(context.Context, string, int64) (int64, error) {
	return 0, nil
}"""

content = content.replace("func (b *factoryCacheBackendSpy) Set(context.Context, string, []byte, time.Duration) error {\n\treturn nil\n}", replacement)

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(content)

