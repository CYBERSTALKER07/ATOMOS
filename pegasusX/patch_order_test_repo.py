import re

with open("apps/backend-go/order/service_test.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*testRepo\) CreateOrderWithBackorder\(ctx context\.Context, o \*Order, bo \*Order, emit func\(outbox\.TxnBuffer\) error, stockOpts StockReservationOpts\) error \{')
replacement = r'func (r *testRepo) CreateOrderWithBackorder(ctx context.Context, o *Order, bo *Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {'
content = pattern.sub(replacement, content)

# I also need to add spanner import if it's missing in service_test.go? Actually context is there.
# Let's just check if it compiles after.
with open("apps/backend-go/order/service_test.go", "w") as f:
    f.write(content)

