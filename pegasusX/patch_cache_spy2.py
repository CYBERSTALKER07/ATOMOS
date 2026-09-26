with open("apps/backend-go/factory/service_test.go", "r") as f:
    content = f.read()

replacement = """func (b *factoryCacheBackendSpy) DecrBy(context.Context, string, int64) (int64, error) {
	return 0, nil
}

func (b *factoryCacheBackendSpy) IncrBy(context.Context, string, int64) (int64, error) {
	return 0, nil
}"""

content = content.replace("func (b *factoryCacheBackendSpy) DecrBy(context.Context, string, int64) (int64, error) {\n\treturn 0, nil\n}", replacement)

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(content)

