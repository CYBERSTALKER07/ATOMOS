package order

// CheckoutOrderContext carries order-scoped payment routing metadata.
type CheckoutOrderContext struct {
	TotalMinor  int64
	Currency    string
	SupplierID  string
	WarehouseID string
}
