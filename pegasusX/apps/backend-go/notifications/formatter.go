package notifications

import (
	"fmt"
	"strings"
)

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

// FormatManifestExceptionResolved produces a notification when a factory exception is resolved.
func FormatManifestExceptionResolved(manifestID, resolution string) FormattedNotification {
	body := "Manifest exception resolved on " + manifestID
	if resolution != "" {
		body += ": " + resolution
	}
	return FormattedNotification{
		Title:    "Exception Resolved",
		Body:     body,
		DeepLink: "/manifest-exceptions",
		Priority: "normal",
	}
}

// FormatFactoryStaffCreated produces a notification for factory staff create.
func FormatFactoryStaffCreated(staffID, factoryID string) FormattedNotification {
	body := "Factory staff " + staffID + " created"
	if factoryID != "" {
		body += " at " + factoryID
	}
	return FormattedNotification{
		Title:    "Factory Staff",
		Body:     body,
		DeepLink: "/staff",
		Priority: "normal",
	}
}

// FormatFactoryTransferCreated produces a notification for factory transfer create.
func FormatFactoryTransferCreated(transferID string) FormattedNotification {
	return FormattedNotification{
		Title:    "Transfer Created",
		Body:     "Factory transfer " + transferID + " created",
		DeepLink: "/transfers",
		Priority: "normal",
	}
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

// FormatShopClosedTimeout produces a notification when grace expires and auto-decision runs.
func FormatShopClosedTimeout(orderID, resolution string) FormattedNotification {
	body := "Shop-closed grace ended for order " + orderID
	if resolution != "" {
		body = "Shop-closed timeout resolved as " + resolution + " for order " + orderID
	}
	return FormattedNotification{
		Title:    "Shop Closed Timeout",
		Body:     body,
		DeepLink: "/exceptions/shop-closed",
		Priority: "high",
	}
}

// FormatProximityUnlocked produces a driver/supplier notice when payment modes unlock.
func FormatProximityUnlocked(orderID, method string) FormattedNotification {
	body := "Settlement proximity unlocked for order " + orderID
	if method != "" {
		body = "Settlement proximity unlocked (" + method + ") for order " + orderID
	}
	return FormattedNotification{
		Title:    "Proximity Unlocked",
		Body:     body,
		DeepLink: "/orders/" + orderID,
		Priority: "normal",
	}
}

// FormatPartialOffload produces a notification when driver records partial delivery.
func FormatPartialOffload(orderID string) FormattedNotification {
	return FormattedNotification{
		Title:    "Partial Offload",
		Body:     "Partial delivery recorded for order " + orderID + "; settlement uses delivered portion only",
		DeepLink: "/orders/" + orderID,
		Priority: "high",
	}
}

// FormatCreditLeave produces a notification when goods are left on credit.
func FormatCreditLeave(orderID string) FormattedNotification {
	return FormattedNotification{
		Title:    "Credit Leave",
		Body:     "Order " + orderID + " delivered on credit; fiscal path pending settlement",
		DeepLink: "/orders/" + orderID,
		Priority: "high",
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

// FormatDriverAvailabilityChanged produces warehouse dispatch inbox copy for driver shift changes.
func FormatDriverAvailabilityChanged(driverID string, onShift bool, reason, note string) FormattedNotification {
	label := strings.TrimSpace(driverID)
	if len(label) > 8 {
		label = label[:8]
	}
	if onShift {
		return FormattedNotification{
			Title:    "Driver Available",
			Body:     "Driver " + label + " is back on shift and eligible for dispatch",
			DeepLink: "/dispatch",
			Priority: "normal",
		}
	}
	display := formatAvailabilityReason(reason, note)
	return FormattedNotification{
		Title:    "Driver Offline",
		Body:     "Driver " + label + " went offline — " + display,
		DeepLink: "/dispatch",
		Priority: "high",
	}
}

// FormatVehicleAvailabilityChanged produces warehouse dispatch inbox copy for vehicle holds.
func FormatVehicleAvailabilityChanged(vehicleID string, isActive bool, reason, note string) FormattedNotification {
	label := strings.TrimSpace(vehicleID)
	if len(label) > 8 {
		label = label[:8]
	}
	if isActive {
		return FormattedNotification{
			Title:    "Truck Restored",
			Body:     "Vehicle " + label + " is active and available for dispatch",
			DeepLink: "/dispatch",
			Priority: "normal",
		}
	}
	display := formatAvailabilityReason(reason, note)
	return FormattedNotification{
		Title:    "Truck Unavailable",
		Body:     "Vehicle " + label + " marked unavailable — " + display,
		DeepLink: "/dispatch",
		Priority: "high",
	}
}

func formatAvailabilityReason(reason, note string) string {
	reason = strings.TrimSpace(reason)
	note = strings.TrimSpace(note)
	if reason == "" {
		return "Unavailable"
	}
	if strings.EqualFold(reason, "OTHER") && note != "" {
		return note
	}
	parts := strings.Split(strings.ToLower(reason), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

// FormatClaimFiled produces inbox copy when a retailer files a claim.
func FormatClaimFiled(claimID, orderID, claimType string) FormattedNotification {
	body := "Claim " + claimID + " filed"
	if orderID != "" {
		body = "Claim " + claimID + " filed on order " + orderID
	}
	if claimType != "" {
		body += " (" + claimType + ")"
	}
	return FormattedNotification{
		Title:    "Claim filed",
		Body:     body,
		DeepLink: "/claims/" + claimID,
		Priority: "high",
	}
}

// FormatClaimUnderReview produces inbox copy when a claim enters settlement review.
func FormatClaimUnderReview(claimID, orderID string) FormattedNotification {
	body := "Claim " + claimID + " is under review"
	if orderID != "" {
		body = "Claim " + claimID + " on order " + orderID + " is under review for settlement"
	}
	return FormattedNotification{
		Title:    "Claim under review",
		Body:     body,
		DeepLink: "/claims/" + claimID,
		Priority: "high",
	}
}

// FormatARInvoice produces inbox copy for AR open-item lifecycle events.
func FormatARInvoice(eventType, invoiceID, orderID, status string, balanceMinor int64, currency string) FormattedNotification {
	title := "AR invoice update"
	switch eventType {
	case "AR_INVOICE_OPENED":
		title = "AR invoice opened"
	case "AR_INVOICE_PAYMENT":
		title = "AR payment recorded"
	case "AR_INVOICE_SETTLED":
		title = "AR invoice settled"
	case "AR_INVOICE_DUNNED":
		title = "AR dunning step"
	}
	body := "Invoice " + invoiceID
	if orderID != "" {
		body += " (order " + orderID + ")"
	}
	if status != "" {
		body += " — " + status
	}
	if balanceMinor > 0 && currency != "" {
		body += fmt.Sprintf(" balance %d %s", balanceMinor, currency)
	}
	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: "/finance/ar/" + invoiceID,
		Priority: "normal",
	}
}

// FormatPayoutBatch produces inbox copy for supplier payout lifecycle.
func FormatPayoutBatch(eventType, batchID, status string, netMinor int64, currency string) FormattedNotification {
	title := "Payout batch update"
	switch eventType {
	case "PAYOUT_BATCH_GENERATED":
		title = "Payout batch generated"
	case "PAYOUT_BATCH_EXPORTED":
		title = "Payout bank file exported"
	case "PAYOUT_BATCH_DISPATCHED":
		title = "Payout batch dispatched"
	case "PAYOUT_BATCH_PAID":
		title = "Payout batch paid"
	}
	body := "Batch " + batchID + " — " + status
	if netMinor != 0 && currency != "" {
		body += fmt.Sprintf(" (%d %s)", netMinor, currency)
	}
	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: "/finance/payouts/" + batchID,
		Priority: "normal",
	}
}

// FormatParentOrder produces inbox copy for multi-supplier parent rollup (B3 M-P0-6).
func FormatParentOrder(eventType, parentOrderID, status string, childCount int) FormattedNotification {
	title := "Parent order update"
	switch eventType {
	case "PARENT_ORDER_CREATED":
		title = "Multi-supplier order created"
	case "PARENT_ORDER_UPDATED":
		title = "Multi-supplier order updated"
	}
	body := "Parent " + parentOrderID
	if status != "" {
		body += " — " + status
	}
	if childCount > 0 {
		body += fmt.Sprintf(" (%d suppliers)", childCount)
	}
	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: "/orders/parent/" + parentOrderID,
		Priority: "normal",
	}
}

// FormatCartSyncUpdated produces a soft UX toast for multi-device cart sync (B3 M-P1-3).
func FormatCartSyncUpdated(_ string) FormattedNotification {
	return FormattedNotification{
		Title:    "Cart updated",
		Body:     "Your cart was updated on another device",
		DeepLink: "/cart",
		Priority: "low",
	}
}

// FormatClaimResolved produces inbox copy when a claim is approved/rejected.
func FormatClaimResolved(claimID, orderID, status string) FormattedNotification {
	body := "Claim " + claimID + " resolved"
	if status != "" {
		body = "Claim " + claimID + " is now " + status
	}
	if orderID != "" {
		body += " (order " + orderID + ")"
	}
	return FormattedNotification{
		Title:    "Claim update",
		Body:     body,
		DeepLink: "/claims/" + claimID,
		Priority: "high",
	}
}

// FormatLogisticsException produces inbox copy for OS&D / reverse-logistics events.
func FormatLogisticsException(eventType, orderID, claimID string) FormattedNotification {
	title := "Logistics exception"
	switch eventType {
	case "REVERSE_LOGISTICS_REQUIRED":
		title = "Return required"
	case "LOGISTICS_EXCEPTION_REPORTED":
		title = "Exception reported"
	}
	body := title
	if orderID != "" {
		body = title + " for order " + orderID
	} else if claimID != "" {
		body = title + " for claim " + claimID
	}
	deep := "/exceptions"
	if claimID != "" {
		deep = "/claims/" + claimID
	} else if orderID != "" {
		deep = "/orders/" + orderID
	}
	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: deep,
		Priority: "high",
	}
}

// FormatRetailerPriceOverride produces retailer inbox copy for custom pricing changes.
func FormatRetailerPriceOverride(productID string, priceMinor int64, currency string, created bool) FormattedNotification {
	label := strings.TrimSpace(productID)
	if len(label) > 8 {
		label = label[:8]
	}
	if label == "" {
		label = "product"
	}
	if created {
		body := fmt.Sprintf("Custom price set for %s", label)
		if priceMinor > 0 && strings.TrimSpace(currency) != "" {
			body = fmt.Sprintf("Custom price %d %s set for %s", priceMinor, currency, label)
		}
		return FormattedNotification{
			Title:    "Custom pricing applied",
			Body:     body,
			DeepLink: "/catalog",
			Priority: "normal",
		}
	}
	return FormattedNotification{
		Title:    "Custom pricing removed",
		Body:     fmt.Sprintf("Standard pricing restored for %s", label),
		DeepLink: "/catalog",
		Priority: "normal",
	}
}
