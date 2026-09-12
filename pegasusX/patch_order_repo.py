import re

with open("apps/backend-go/order/service.go", "r") as f:
    content = f.read()

pattern = re.compile(r'CreateOrderWithBackorder\(ctx context\.Context, o \*Order, backorder \*Order, emit func\(outbox\.TxnBuffer\) error, stockOpts StockReservationOpts\) error')
replacement = r'CreateOrderWithBackorder(ctx context.Context, o *Order, backorder *Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error'
content = pattern.sub(replacement, content)

with open("apps/backend-go/order/service.go", "w") as f:
    f.write(content)

with open("apps/backend-go/order/repository_spanner.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*SpannerRepository\) CreateOrderWithBackorder\(ctx context\.Context, o \*Order, backorder \*Order, emit func\(outbox\.TxnBuffer\) error, stockOpts StockReservationOpts\) error \{')
replacement = r'func (r *SpannerRepository) CreateOrderWithBackorder(ctx context.Context, o *Order, backorder *Order, inTxn func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {'
content = pattern.sub(replacement, content)

# inside CreateOrderWithBackorder, add the inTxn call right after emit
pattern2 = re.compile(r'\t\t\t\tif err := emit\(buf\); err != nil \{\n\t\t\t\t\treturn err\n\t\t\t\t\}\n\t\t\t\}')
replacement2 = r"""				if err := emit(buf); err != nil {
					return err
				}
			}
			if inTxn != nil {
				if err := inTxn(ctx, txn); err != nil {
					return err
				}
			}"""
content = pattern2.sub(replacement2, content)

with open("apps/backend-go/order/repository_spanner.go", "w") as f:
    f.write(content)

