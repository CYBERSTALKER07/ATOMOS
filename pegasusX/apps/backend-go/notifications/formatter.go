package notifications

// FormattedNotification is the output of a Format* function — pure data,
// zero side effects. Transport decides how to deliver it.
type FormattedNotification struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	DeepLink string `json:"deep_link"`
	Priority string `json:"priority"`
}

// FormatOrderCreated produces a notification for a new order event.
func FormatOrderCreated(orderID, supplierName string, totalMinor int64, currency string) FormattedNotification {
	return FormattedNotification{
		Title:    "New Order Received",
		Body:     "Order " + orderID + " from " + supplierName,
		DeepLink: "/orders/" + orderID,
		Priority: "high",
	}
}

// FormatOrderStatusChanged produces a notification for an order status change.
func FormatOrderStatusChanged(orderID, newStatus string) FormattedNotification {
	return FormattedNotification{
		Title:    "Order Update",
		Body:     "Order " + orderID + " is now " + newStatus,
		DeepLink: "/orders/" + orderID,
		Priority: "normal",
	}
}

// FormatManifestDispatched produces a notification for a dispatched manifest.
func FormatManifestDispatched(manifestID, driverName string) FormattedNotification {
	return FormattedNotification{
		Title:    "Manifest Dispatched",
		Body:     "Manifest " + manifestID + " dispatched with " + driverName,
		DeepLink: "/manifests/" + manifestID,
		Priority: "high",
	}
}

// FormatManifestCompleted produces a notification for a completed manifest.
func FormatManifestCompleted(manifestID string, orderCount int) FormattedNotification {
	return FormattedNotification{
		Title:    "Manifest Complete",
		Body:     "Manifest " + manifestID + " delivered all orders",
		DeepLink: "/manifests/" + manifestID,
		Priority: "normal",
	}
}

// FormatPaymentReceived produces a notification for a successful payment.
func FormatPaymentReceived(orderID string, amountMinor int64, currency string) FormattedNotification {
	return FormattedNotification{
		Title:    "Payment Received",
		Body:     "Payment for order " + orderID + " confirmed",
		DeepLink: "/orders/" + orderID,
		Priority: "normal",
	}
}
