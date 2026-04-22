package notifications

import "fmt"

// FormatNotification returns a human-readable title and body for a notification event.
// Used by the notification dispatcher consumer to populate the Notifications inbox
// and format Telegram messages.

type FormattedNotification struct {
	Title string
	Body  string
}

func FormatOrderDispatched(routeID string, orderCount int) FormattedNotification {
	return FormattedNotification{
		Title: "Order Dispatched",
		Body:  fmt.Sprintf("Your order has been dispatched on route %s. %d order(s) are on the way.", routeID, orderCount),
	}
}

func FormatDriverDispatched(routeID string, orderCount int) FormattedNotification {
	return FormattedNotification{
		Title: "New Dispatch Assignment",
		Body:  fmt.Sprintf("You have been assigned route %s with %d order(s). Check your route for details.", routeID, orderCount),
	}
}

func FormatDriverArrived(orderID string) FormattedNotification {
	return FormattedNotification{
		Title: "Driver Has Arrived",
		Body:  fmt.Sprintf("The driver has arrived for order %s. Please prepare for delivery.", orderID),
	}
}

func FormatOrderStatusChanged(orderID, oldState, newState string) FormattedNotification {
	return FormattedNotification{
		Title: "Order Status Updated",
		Body:  fmt.Sprintf("Order %s status changed: %s → %s", orderID, oldState, newState),
	}
}

func FormatPayloadReadyToSeal(routeID string, orderCount int) FormattedNotification {
	return FormattedNotification{
		Title: "Orders Ready to Seal",
		Body:  fmt.Sprintf("Route %s has %d order(s) ready for payload sealing. Proceed to the loading dock.", routeID, orderCount),
	}
}

func FormatPayloadSealed(orderID, terminalID string) FormattedNotification {
	return FormattedNotification{
		Title: "Payload Sealed",
		Body:  fmt.Sprintf("Order %s has been sealed by terminal %s and is ready for dispatch.", orderID, terminalID),
	}
}

func FormatPaymentSettled(orderID, gateway string, amount int64) FormattedNotification {
	return FormattedNotification{
		Title: "Payment Received",
		Body:  fmt.Sprintf("Payment of %d received for order %s via %s.", amount, orderID, gateway),
	}
}

func FormatPaymentFailed(orderID, gateway, reason string) FormattedNotification {
	return FormattedNotification{
		Title: "Payment Failed",
		Body:  fmt.Sprintf("Payment for order %s via %s failed: %s. Please try again.", orderID, gateway, reason),
	}
}

func FormatDriverOffline(driverName, reason string) FormattedNotification {
	return FormattedNotification{
		Title: "Driver Went Offline",
		Body:  fmt.Sprintf("Driver %s has gone offline. Reason: %s", driverName, reason),
	}
}

func FormatDriverOnline(driverName string) FormattedNotification {
	return FormattedNotification{
		Title: "Driver Back Online",
		Body:  fmt.Sprintf("Driver %s is back online and available for dispatch.", driverName),
	}
}

func FormatOrderReassignedRemoved(orderID string) FormattedNotification {
	return FormattedNotification{
		Title: "Order Removed from Route",
		Body:  fmt.Sprintf("Order %s has been reassigned to another driver and removed from your route.", orderID),
	}
}

func FormatOrderReassignedAdded(orderID string) FormattedNotification {
	return FormattedNotification{
		Title: "New Order Added to Route",
		Body:  fmt.Sprintf("Order %s has been reassigned to your route. Check route updates.", orderID),
	}
}

// FormatTelegram returns a Markdown-formatted Telegram message from a notification.
func FormatTelegram(n FormattedNotification) string {
	return fmt.Sprintf("*%s*\n%s", n.Title, n.Body)
}

// FormatOrderAmended formats a notification for partial-delivery amendment.
func FormatOrderAmended(orderID string, refunded int64) FormattedNotification {
	if refunded > 0 {
		return FormattedNotification{
			Title: "Order Amended",
			Body:  fmt.Sprintf("Order %s has been amended. Refund of %d applied for returned items.", orderID, refunded),
		}
	}
	return FormattedNotification{
		Title: "Order Amended",
		Body:  fmt.Sprintf("Order %s has been amended. No refund required.", orderID),
	}
}

// FormatFleetDispatched notifies a supplier admin that a truck manifest has been committed.
func FormatFleetDispatched(routeID string, orderCount int) FormattedNotification {
	return FormattedNotification{
		Title: "Fleet Dispatched",
		Body:  fmt.Sprintf("Route %s committed with %d order(s).", routeID, orderCount),
	}
}

// FormatDispatchLockAcquired notifies a supplier admin that a dispatch lock has been acquired.
func FormatDispatchLockAcquired(lockType, lockedBy string) FormattedNotification {
	return FormattedNotification{
		Title: "Dispatch Lock Acquired",
		Body:  fmt.Sprintf("%s lock activated by %s. New orders are queued until the lock is released.", lockType, lockedBy),
	}
}

// FormatDispatchLockReleased notifies a supplier admin that a dispatch lock has been released.
func FormatDispatchLockReleased(lockType, lockedBy string) FormattedNotification {
	return FormattedNotification{
		Title: "Dispatch Lock Released",
		Body:  fmt.Sprintf("%s lock released by %s. Queued orders are available for dispatch.", lockType, lockedBy),
	}
}

// FormatFreezeLockAcquired notifies a supplier admin that the AI worker has been frozen on a scope.
func FormatFreezeLockAcquired(scope string) FormattedNotification {
	return FormattedNotification{
		Title: "AI Freeze Lock Engaged",
		Body:  fmt.Sprintf("AI worker paused for scope %s during manual dispatch.", scope),
	}
}

// FormatFreezeLockReleased notifies a supplier admin that the AI worker freeze has lifted.
func FormatFreezeLockReleased(scope string) FormattedNotification {
	return FormattedNotification{
		Title: "AI Freeze Lock Released",
		Body:  fmt.Sprintf("AI worker resumed for scope %s. Dispatch automation back online.", scope),
	}
}

// FormatDriverCreated notifies a supplier admin that a new driver has been registered.
func FormatDriverCreated(name, phone string) FormattedNotification {
	return FormattedNotification{
		Title: "Driver Registered",
		Body:  fmt.Sprintf("Driver %s (%s) has been added to your fleet.", name, phone),
	}
}

// FormatVehicleCreated notifies a supplier admin that a new vehicle has been registered.
func FormatVehicleCreated(label, licensePlate string) FormattedNotification {
	return FormattedNotification{
		Title: "Vehicle Registered",
		Body:  fmt.Sprintf("Vehicle %s (%s) has been added to your fleet.", label, licensePlate),
	}
}

// FormatManifestRebalanced notifies the supplier admin that a manifest's
// transfer composition was changed by a manual or automated rebalance.
func FormatManifestRebalanced(manifestID string, transferCount, _ int) FormattedNotification {
	return FormattedNotification{
		Title: "Manifest Rebalanced",
		Body:  fmt.Sprintf("Manifest %s rebalanced: %d transfers moved.", manifestID, transferCount),
	}
}

// FormatManifestCancelled notifies the supplier admin that a manifest (or batch
// of transfers) was cancelled — typically by Kill Switch or factory override.
func FormatManifestCancelled(manifestID, reason string, releasedCount int) FormattedNotification {
	return FormattedNotification{
		Title: "Manifest Cancelled",
		Body:  fmt.Sprintf("Manifest %s cancelled (%s). %d orders released.", manifestID, reason, releasedCount),
	}
}

// FormatForceSealAlert notifies the supplier admin that a force-seal override
// has crossed the warning threshold (3+ in 24h).
func FormatForceSealAlert(manifestID string, count24h, quota int64) FormattedNotification {
	return FormattedNotification{
		Title: "FORCE-SEAL Threshold",
		Body:  fmt.Sprintf("Override quota at %d/%d in 24h on manifest %s. Investigate loading bay.", count24h, quota, manifestID),
	}
}

// FormatOrderDelayed notifies the retailer that their order is delayed.
func FormatOrderDelayed(orderID, reason string) FormattedNotification {
	return FormattedNotification{
		Title: "Order Delayed",
		Body:  fmt.Sprintf("Order %s delayed: %s.", orderID, reason),
	}
}

// FormatManifestOrderReassigned notifies the retailer that their order was moved
// from one manifest to another.
func FormatManifestOrderReassigned(orderID string) FormattedNotification {
	return FormattedNotification{
		Title: "Order Reassigned",
		Body:  fmt.Sprintf("Order %s has been moved to a new manifest. ETA may shift.", orderID),
	}
}

// FormatManifestDispatched notifies the supplier admin that the driver has
// departed and the manifest entered the DISPATCHED phase. Atomic with the
// driver-depart Spanner mutation.
func FormatManifestDispatched(manifestID string, stopCount int) FormattedNotification {
	return FormattedNotification{
		Title: "Manifest Dispatched",
		Body:  fmt.Sprintf("Manifest %s departed the gate (%d stops).", manifestID, stopCount),
	}
}

// FormatManifestCompleted notifies the supplier admin that every stop on the
// manifest has reached COMPLETED — the manifest lifecycle is terminal.
func FormatManifestCompleted(manifestID string, stopCount int) FormattedNotification {
	return FormattedNotification{
		Title: "Manifest Completed",
		Body:  fmt.Sprintf("Manifest %s closed: all %d stops delivered.", manifestID, stopCount),
	}
}

// FormatManifestSettled notifies the supplier admin that the financial settlement
// for a manifest has been finalised — all per-order ledger splits are reconciled.
func FormatManifestSettled(manifestID string, supplierPayout int64, currency string) FormattedNotification {
	return FormattedNotification{
		Title: "Payout Settled",
		Body:  fmt.Sprintf("Manifest %s settled. Supplier payout: %.2f %s.", manifestID, float64(supplierPayout)/100.0, currency),
	}
}

// FormatOrderCancelledByOrigin — hard-kill notification: admin cancelled the order
// before dispatch. Goes 3-way: warehouse + supplier + retailer.
func FormatOrderCancelledByOrigin(orderID, reason string) FormattedNotification {
	return FormattedNotification{
		Title: "Order Cancelled",
		Body:  fmt.Sprintf("Order %s cancelled by origin: %s. Pending payment voided.", orderID, reason),
	}
}

// FormatPayloadOverflow — soft-stop notification: order didn't fit the truck.
// Returns to unassigned pool for redispatch.
func FormatPayloadOverflow(orderID, manifestID string) FormattedNotification {
	return FormattedNotification{
		Title: "Payload Overflow",
		Body:  fmt.Sprintf("Order %s removed from manifest %s (payload overflow). Queued for redispatch.", orderID, manifestID),
	}
}
