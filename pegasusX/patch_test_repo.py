import re

with open("apps/backend-go/order/service_test.go", "r") as f:
    content = f.read()

replacement = """
func (m *testRepo) CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {
	m.orders[o.OrderID] = *o
	if emit != nil {
		return emit(outbox.NoopBuffer{})
	}
	return nil
}

func (m *testRepo) CreateOrderWithBackorder(ctx context.Context, o *Order, bo *Order, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {
	m.orders[o.OrderID] = *o
	if bo != nil {
		m.orders[bo.OrderID] = *bo
	}
	if emit != nil {
		return emit(outbox.NoopBuffer{})
	}
	return nil
}
"""

pattern = re.compile(r'\nfunc \(m \*testRepo\) CreateOrder\(.*?\n\}\n', re.DOTALL)
content = pattern.sub(replacement, content)

with open("apps/backend-go/order/service_test.go", "w") as f:
    f.write(content)

