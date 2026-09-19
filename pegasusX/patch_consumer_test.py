import re

with open("apps/backend-go/order/consumer_test.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*consumerRepoStub\) CreateOrderWithBackorder\(ctx context\.Context, o \*Order, backorder \*Order, emit func\(outbox\.TxnBuffer\) error, stockOpts StockReservationOpts\) error \{')
replacement = r'func (r *consumerRepoStub) CreateOrderWithBackorder(ctx context.Context, o *Order, backorder *Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {'
content = pattern.sub(replacement, content)

with open("apps/backend-go/order/consumer_test.go", "w") as f:
    f.write(content)
