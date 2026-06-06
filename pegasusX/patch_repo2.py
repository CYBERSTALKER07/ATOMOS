import re

with open("apps/backend-go/order/external_payment.go", "r") as f:
    content = f.read()

content = content.replace("orderRecord.PaymentMethod = gateway\n\t", "")

with open("apps/backend-go/order/external_payment.go", "w") as f:
    f.write(content)

with open("apps/backend-go/order/repository_spanner.go", "r") as f:
    content = f.read()

content = content.replace("orderCols", "orderSelectColumns")
content = content.replace("scanOrder", "scanOrderRow")

with open("apps/backend-go/order/repository_spanner.go", "w") as f:
    f.write(content)

with open("apps/backend-go/order/service_test.go", "r") as f:
    content = f.read()

content = re.sub(
    r"func \(r \*testRepo\) ListDueAutoConfirmOrders\(ctx context\.Context, before time\.Time, limit int\) \(\[\]Order, error\) \{\n\treturn nil, nil\n\}",
    r"""func (r *testRepo) ListDueAutoConfirmOrders(ctx context.Context, before time.Time, limit int) ([]Order, error) {
	return nil, nil
}

func (r *testRepo) ListManifestOrders(ctx context.Context, manifestID string) ([]Order, error) {
	return nil, nil
}""",
    content
)

with open("apps/backend-go/order/service_test.go", "w") as f:
    f.write(content)

