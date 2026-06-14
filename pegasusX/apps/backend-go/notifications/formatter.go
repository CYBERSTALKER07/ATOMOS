package notifications

import "fmt"

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
func FormatManifestDispatched(manifestID, driverID string) FormattedNotification {
	body := "Manifest " + manifestID + " dispatched"
	if driverID != "" {
		body = "Manifest " + manifestID + " dispatched with driver " + driverID
	}
	return formatManifestNotification("Manifest Dispatched", body, manifestID, "high")
}

// FormatManifestCompleted produces a notification for a completed manifest.
func FormatManifestCompleted(manifestID string, orderCount int) FormattedNotification {
	body := "Manifest " + manifestID + " completed"
	if orderCount > 0 {
		body = fmt.Sprintf("Manifest %s completed (%d orders)", manifestID, orderCount)
	}
	return formatManifestNotification("Manifest Complete", body, manifestID, "normal")
}

// FormatManifestDraftCreated produces a notification when a manifest draft is created.
func FormatManifestDraftCreated(manifestID string) FormattedNotification {
	return formatManifestNotification(
		"Manifest Draft Created",
		"Manifest "+manifestID+" is ready for loading",
		manifestID,
		"normal",
	)
}

// FormatManifestLoadingStarted produces a notification when loading begins.
func FormatManifestLoadingStarted(manifestID, driverID string) FormattedNotification {
	body := "Loading started for manifest " + manifestID
	if driverID != "" {
		body = "Driver " + driverID + " started loading manifest " + manifestID
	}
	return formatManifestNotification("Manifest Loading", body, manifestID, "normal")
}

// FormatManifestSealed produces a notification when a manifest is sealed.
func FormatManifestSealed(manifestID string) FormattedNotification {
	return formatManifestNotification(
		"Manifest Sealed",
		"Manifest "+manifestID+" sealed and ready for dispatch",
		manifestID,
		"high",
	)
}

// FormatManifestOrderInjected produces a notification when an order is injected into a manifest.
func FormatManifestOrderInjected(manifestID, orderID string) FormattedNotification {
	body := "Order added to manifest " + manifestID
	if orderID != "" {
		body = "Order " + orderID + " injected into manifest " + manifestID
	}
	return formatManifestNotification("Order Injected", body, manifestID, "normal")
}

// FormatManifestOrderException produces a notification for a manifest gate exception.
func FormatManifestOrderException(manifestID, orderID, reason string) FormattedNotification {
	body := "Manifest gate exception on " + manifestID
	if orderID != "" {
		body = "Order " + orderID + " removed from manifest " + manifestID
	}
	if reason != "" {
		body += ": " + reason
	}
	return FormattedNotification{
		Title:    "Manifest Exception",
		Body:     body,
		DeepLink: "/manifest-exceptions",
		Priority: "high",
	}
}

// FormatManifestDLQEscalation produces a notification when a manifest exception escalates.
func FormatManifestDLQEscalation(manifestID, reason string) FormattedNotification {
	body := "Manifest " + manifestID + " exception escalated"
	if reason != "" {
		body += ": " + reason
	}
	return FormattedNotification{
		Title:    "Manifest DLQ Escalation",
		Body:     body,
		DeepLink: "/manifest-exceptions",
		Priority: "high",
	}
}

// FormatManifestRebalanced produces a notification when payload override rebalances manifests.
func FormatManifestRebalanced(manifestID, fromManifestID, toManifestID string) FormattedNotification {
	body := "Manifest " + manifestID + " rebalanced"
	if fromManifestID != "" && toManifestID != "" {
		body = "Transfer moved from manifest " + fromManifestID + " to " + toManifestID
	}
	return formatManifestNotification("Manifest Rebalanced", body, manifestID, "normal")
}

// FormatManifestCancelled produces a notification when a manifest is cancelled.
func FormatManifestCancelled(manifestID, reason string) FormattedNotification {
	body := "Manifest " + manifestID + " cancelled"
	if reason != "" {
		body += ": " + reason
	}
	return formatManifestNotification("Manifest Cancelled", body, manifestID, "normal")
}

func formatManifestNotification(title, body, manifestID, priority string) FormattedNotification {
	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: "/manifests/" + manifestID,
		Priority: priority,
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

// FormatShopClosed produces a supplier-facing notification when a driver reports shop closed.
func FormatShopClosed(orderID, driverID string) FormattedNotification {
	body := "Driver reported shop closed for order " + orderID
	if driverID != "" {
		body = "Driver " + driverID + " reported shop closed for order " + orderID
	}
	return FormattedNotification{
		Title:    "Shop Closed Reported",
		Body:     body,
		DeepLink: "/exceptions/shop-closed",
		Priority: "high",
	}
}

// FormatShopClosedEscalated produces a supplier notification when grace expires.
func FormatShopClosedEscalated(orderID string) FormattedNotification {
	return FormattedNotification{
		Title:    "Shop Closed Escalated",
		Body:     "Order " + orderID + " needs supplier resolution",
		DeepLink: "/exceptions/shop-closed",
		Priority: "high",
	}
}

// FormatShopClosedResolved produces a notification when supplier resolves an attempt.
func FormatShopClosedResolved(orderID, resolution string) FormattedNotification {
	body := "Shop-closed case resolved for order " + orderID
	if resolution != "" {
		body = "Shop-closed case resolved (" + resolution + ") for order " + orderID
	}
	return FormattedNotification{
		Title:    "Shop Closed Resolved",
		Body:     body,
		DeepLink: "/exceptions/shop-closed",
		Priority: "normal",
	}
}

// FormatShopClosedResponse produces a notification when the retailer responds.
func FormatShopClosedResponse(orderID, response string) FormattedNotification {
	body := "Retailer responded to shop-closed report for order " + orderID
	if response != "" {
		body = "Retailer chose " + response + " for order " + orderID
	}
	return FormattedNotification{
		Title:    "Shop Closed Response",
		Body:     body,
		DeepLink: "/orders/" + orderID,
		Priority: "normal",
	}
}

// FormatDriverCreated produces a supplier notification when a driver is onboarded.
func FormatDriverCreated(driverID, homeNodeID string) FormattedNotification {
	body := "Driver " + driverID + " added to fleet"
	if homeNodeID != "" {
		body = "Driver " + driverID + " added at home node " + homeNodeID
	}
	return FormattedNotification{
		Title:    "Driver Added",
		Body:     body,
		DeepLink: "/org-fleet",
		Priority: "normal",
	}
}

// FormatVehicleCreated produces a supplier notification when a vehicle is onboarded.
func FormatVehicleCreated(vehicleID, homeNodeID string) FormattedNotification {
	body := "Vehicle " + vehicleID + " added to fleet"
	if homeNodeID != "" {
		body = "Vehicle " + vehicleID + " added at home node " + homeNodeID
	}
	return FormattedNotification{
		Title:    "Vehicle Added",
		Body:     body,
		DeepLink: "/org-fleet",
		Priority: "normal",
	}
}
