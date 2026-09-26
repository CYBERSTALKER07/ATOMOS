import re

with open("apps/backend-go/order/service_test.go", "r") as f:
    content = f.read()

replacement = """
func (r *testRepo) CreateOrderWithBackorder(ctx context.Context, o *Order, bo *Order, emit func(outbox.TxnBuffer) error, stockOpts StockReservationOpts) error {
	if r.createErr != nil {
		return r.createErr
	}
	if o == nil {
		return errors.New("nil order")
	}
	r.createCalls++
	if r.retailerWindowOpen != "" || r.retailerWindowClose != "" {
		if err := SnapshotReceivingWindowsOnOrder(o, r.retailerWindowOpen, r.retailerWindowClose); err != nil {
			return err
		}
	}
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
		r.lastEvents = append(r.lastEvents, buf.events...)
	}
	r.created = *o
	return nil
}
"""

pattern = re.compile(r'\nfunc \(r \*testRepo\) CreateOrderWithBackorder\(.*?\n\}\n', re.DOTALL)
content = pattern.sub(replacement, content)

with open("apps/backend-go/order/service_test.go", "w") as f:
    f.write(content)

