package notifications

import "fmt"

// FormatNotification returns a human-readable title and body for a notification event.
// Used by the notification dispatcher consumer to populate the Notifications inbox
// and format Telegram messages.

type FormattedNotification struct {
	Title       string
	Body        string
	TitleKey    string
	BodyKey     string
	MessageArgs map[string]string
}

func newFormattedNotification(title, body, titleKey, bodyKey string, args map[string]string) FormattedNotification {
	return FormattedNotification{
		Title:       title,
		Body:        body,
		TitleKey:    titleKey,
		BodyKey:     bodyKey,
		MessageArgs: args,
	}
}

// NewFormattedNotification creates a notification payload with localization
// metadata while preserving the current rendered title/body fallback.
func NewFormattedNotification(title, body, titleKey, bodyKey string, args map[string]string) FormattedNotification {
	return newFormattedNotification(title, body, titleKey, bodyKey, args)
}

func FormatOrderDispatched(routeID string, orderCount int) FormattedNotification {
	return newFormattedNotification(
		"Order Dispatched",
		fmt.Sprintf("Your order has been dispatched on route %s. %d order(s) are on the way.", routeID, orderCount),
		"notification.order_dispatched.title",
		"notification.order_dispatched.body",
		map[string]string{"route_id": routeID, "order_count": fmt.Sprintf("%d", orderCount)},
	)
}

func FormatDriverDispatched(routeID string, orderCount int) FormattedNotification {
	return newFormattedNotification(
		"New Dispatch Assignment",
		fmt.Sprintf("You have been assigned route %s with %d order(s). Check your route for details.", routeID, orderCount),
		"notification.driver_dispatched.title",
		"notification.driver_dispatched.body",
		map[string]string{"route_id": routeID, "order_count": fmt.Sprintf("%d", orderCount)},
	)
}

func FormatDriverArrived(orderID string) FormattedNotification {
	return newFormattedNotification(
		"Driver Has Arrived",
		fmt.Sprintf("The driver has arrived for order %s. Please prepare for delivery.", orderID),
		"notification.driver_arrived.title",
		"notification.driver_arrived.body",
		map[string]string{"order_id": orderID},
	)
}

func FormatOrderStatusChanged(orderID, oldState, newState string) FormattedNotification {
	return newFormattedNotification(
		"Order Status Updated",
		fmt.Sprintf("Order %s status changed: %s → %s", orderID, oldState, newState),
		"notification.order_status_changed.title",
		"notification.order_status_changed.body",
		map[string]string{"order_id": orderID, "old_state": oldState, "new_state": newState},
	)
}

func FormatPayloadReadyToSeal(routeID string, orderCount int) FormattedNotification {
	return newFormattedNotification(
		"Orders Ready to Seal",
		fmt.Sprintf("Route %s has %d order(s) ready for payload sealing. Proceed to the loading dock.", routeID, orderCount),
		"notification.payload_ready_to_seal.title",
		"notification.payload_ready_to_seal.body",
		map[string]string{"route_id": routeID, "order_count": fmt.Sprintf("%d", orderCount)},
	)
}

func FormatPayloadSealed(orderID, terminalID string) FormattedNotification {
	return newFormattedNotification(
		"Payload Sealed",
		fmt.Sprintf("Order %s has been sealed by terminal %s and is ready for dispatch.", orderID, terminalID),
		"notification.payload_sealed.title",
		"notification.payload_sealed.body",
		map[string]string{"order_id": orderID, "terminal_id": terminalID},
	)
}

func FormatPaymentSettled(orderID, gateway string, amount int64) FormattedNotification {
	return newFormattedNotification(
		"Payment Received",
		fmt.Sprintf("Payment of %d received for order %s via %s.", amount, orderID, gateway),
		"notification.payment_settled.title",
		"notification.payment_settled.body",
		map[string]string{"order_id": orderID, "gateway": gateway, "amount_minor": fmt.Sprintf("%d", amount)},
	)
}

func FormatPaymentFailed(orderID, gateway, reason string) FormattedNotification {
	return newFormattedNotification(
		"Payment Failed",
		fmt.Sprintf("Payment for order %s via %s failed: %s. Please try again.", orderID, gateway, reason),
		"notification.payment_failed.title",
		"notification.payment_failed.body",
		map[string]string{"order_id": orderID, "gateway": gateway, "reason": reason},
	)
}

func FormatDriverOffline(driverName, reason string) FormattedNotification {
	return newFormattedNotification(
		"Driver Went Offline",
		fmt.Sprintf("Driver %s has gone offline. Reason: %s", driverName, reason),
		"notification.driver_offline.title",
		"notification.driver_offline.body",
		map[string]string{"driver_name": driverName, "reason": reason},
	)
}

func FormatDriverOnline(driverName string) FormattedNotification {
	return newFormattedNotification(
		"Driver Back Online",
		fmt.Sprintf("Driver %s is back online and available for dispatch.", driverName),
		"notification.driver_online.title",
		"notification.driver_online.body",
		map[string]string{"driver_name": driverName},
	)
}

func FormatOrderReassignedRemoved(orderID string) FormattedNotification {
	return newFormattedNotification(
		"Order Removed from Route",
		fmt.Sprintf("Order %s has been reassigned to another driver and removed from your route.", orderID),
		"notification.order_reassigned_removed.title",
		"notification.order_reassigned_removed.body",
		map[string]string{"order_id": orderID},
	)
}

func FormatOrderReassignedAdded(orderID string) FormattedNotification {
	return newFormattedNotification(
		"New Order Added to Route",
		fmt.Sprintf("Order %s has been reassigned to your route. Check route updates.", orderID),
		"notification.order_reassigned_added.title",
		"notification.order_reassigned_added.body",
		map[string]string{"order_id": orderID},
	)
}

// FormatTelegram returns a Markdown-formatted Telegram message from a notification.
func FormatTelegram(n FormattedNotification) string {
	return fmt.Sprintf("*%s*\n%s", n.Title, n.Body)
}

// FormatOrderAmended formats a notification for partial-delivery amendment.
func FormatOrderAmended(orderID string, refunded int64) FormattedNotification {
	if refunded > 0 {
		return newFormattedNotification(
			"Order Amended",
			fmt.Sprintf("Order %s has been amended. Refund of %d applied for returned items.", orderID, refunded),
			"notification.order_amended.title",
			"notification.order_amended.refund_body",
			map[string]string{"order_id": orderID, "refunded_minor": fmt.Sprintf("%d", refunded)},
		)
	}
	return newFormattedNotification(
		"Order Amended",
		fmt.Sprintf("Order %s has been amended. No refund required.", orderID),
		"notification.order_amended.title",
		"notification.order_amended.no_refund_body",
		map[string]string{"order_id": orderID},
	)
}

// FormatFleetDispatched notifies a supplier admin that a truck manifest has been committed.
func FormatFleetDispatched(routeID string, orderCount int) FormattedNotification {
	return newFormattedNotification(
		"Fleet Dispatched",
		fmt.Sprintf("Route %s committed with %d order(s).", routeID, orderCount),
		"notification.fleet_dispatched.title",
		"notification.fleet_dispatched.body",
		map[string]string{"route_id": routeID, "order_count": fmt.Sprintf("%d", orderCount)},
	)
}

// FormatDispatchLockAcquired notifies a supplier admin that a dispatch lock has been acquired.
func FormatDispatchLockAcquired(lockType, lockedBy string) FormattedNotification {
	return newFormattedNotification(
		"Dispatch Lock Acquired",
		fmt.Sprintf("%s lock activated by %s. New orders are queued until the lock is released.", lockType, lockedBy),
		"notification.dispatch_lock_acquired.title",
		"notification.dispatch_lock_acquired.body",
		map[string]string{"lock_type": lockType, "locked_by": lockedBy},
	)
}

// FormatDispatchLockReleased notifies a supplier admin that a dispatch lock has been released.
func FormatDispatchLockReleased(lockType, lockedBy string) FormattedNotification {
	return newFormattedNotification(
		"Dispatch Lock Released",
		fmt.Sprintf("%s lock released by %s. Queued orders are available for dispatch.", lockType, lockedBy),
		"notification.dispatch_lock_released.title",
		"notification.dispatch_lock_released.body",
		map[string]string{"lock_type": lockType, "locked_by": lockedBy},
	)
}

// FormatFreezeLockAcquired notifies a supplier admin that the AI worker has been frozen on a scope.
func FormatFreezeLockAcquired(scope string) FormattedNotification {
	return newFormattedNotification(
		"AI Freeze Lock Engaged",
		fmt.Sprintf("AI worker paused for scope %s during manual dispatch.", scope),
		"notification.freeze_lock_acquired.title",
		"notification.freeze_lock_acquired.body",
		map[string]string{"scope": scope},
	)
}

// FormatFreezeLockReleased notifies a supplier admin that the AI worker freeze has lifted.
func FormatFreezeLockReleased(scope string) FormattedNotification {
	return newFormattedNotification(
		"AI Freeze Lock Released",
		fmt.Sprintf("AI worker resumed for scope %s. Dispatch automation back online.", scope),
		"notification.freeze_lock_released.title",
		"notification.freeze_lock_released.body",
		map[string]string{"scope": scope},
	)
}

// FormatDriverCreated notifies a supplier admin that a new driver has been registered.
func FormatDriverCreated(name, phone string) FormattedNotification {
	return newFormattedNotification(
		"Driver Registered",
		fmt.Sprintf("Driver %s (%s) has been added to your fleet.", name, phone),
		"notification.driver_created.title",
		"notification.driver_created.body",
		map[string]string{"driver_name": name, "phone": phone},
	)
}

// FormatVehicleCreated notifies a supplier admin that a new vehicle has been registered.
func FormatVehicleCreated(label, licensePlate string) FormattedNotification {
	return newFormattedNotification(
		"Vehicle Registered",
		fmt.Sprintf("Vehicle %s (%s) has been added to your fleet.", label, licensePlate),
		"notification.vehicle_created.title",
		"notification.vehicle_created.body",
		map[string]string{"vehicle_label": label, "license_plate": licensePlate},
	)
}

// FormatManifestRebalanced notifies the supplier admin that a manifest's
// transfer composition was changed by a manual or automated rebalance.
func FormatManifestRebalanced(manifestID string, transferCount, _ int) FormattedNotification {
	return newFormattedNotification(
		"Manifest Rebalanced",
		fmt.Sprintf("Manifest %s rebalanced: %d transfers moved.", manifestID, transferCount),
		"notification.manifest_rebalanced.title",
		"notification.manifest_rebalanced.body",
		map[string]string{"manifest_id": manifestID, "transfer_count": fmt.Sprintf("%d", transferCount)},
	)
}

// FormatManifestCancelled notifies the supplier admin that a manifest (or batch
// of transfers) was cancelled — typically by Kill Switch or factory override.
func FormatManifestCancelled(manifestID, reason string, releasedCount int) FormattedNotification {
	return newFormattedNotification(
		"Manifest Cancelled",
		fmt.Sprintf("Manifest %s cancelled (%s). %d orders released.", manifestID, reason, releasedCount),
		"notification.manifest_cancelled.title",
		"notification.manifest_cancelled.body",
		map[string]string{"manifest_id": manifestID, "reason": reason, "released_count": fmt.Sprintf("%d", releasedCount)},
	)
}

// FormatForceSealAlert notifies the supplier admin that a force-seal override
// has crossed the warning threshold (3+ in 24h).
func FormatForceSealAlert(manifestID string, count24h, quota int64) FormattedNotification {
	return newFormattedNotification(
		"FORCE-SEAL Threshold",
		fmt.Sprintf("Override quota at %d/%d in 24h on manifest %s. Investigate loading bay.", count24h, quota, manifestID),
		"notification.force_seal_alert.title",
		"notification.force_seal_alert.body",
		map[string]string{"manifest_id": manifestID, "count_24h": fmt.Sprintf("%d", count24h), "quota": fmt.Sprintf("%d", quota)},
	)
}

// FormatOrderDelayed notifies the retailer that their order is delayed.
func FormatOrderDelayed(orderID, reason string) FormattedNotification {
	return newFormattedNotification(
		"Order Delayed",
		fmt.Sprintf("Order %s delayed: %s.", orderID, reason),
		"notification.order_delayed.title",
		"notification.order_delayed.body",
		map[string]string{"order_id": orderID, "reason": reason},
	)
}

// FormatManifestOrderReassigned notifies the retailer that their order was moved
// from one manifest to another.
func FormatManifestOrderReassigned(orderID string) FormattedNotification {
	return newFormattedNotification(
		"Order Reassigned",
		fmt.Sprintf("Order %s has been moved to a new manifest. ETA may shift.", orderID),
		"notification.manifest_order_reassigned.title",
		"notification.manifest_order_reassigned.body",
		map[string]string{"order_id": orderID},
	)
}

// FormatManifestDispatched notifies the supplier admin that the driver has
// departed and the manifest entered the DISPATCHED phase. Atomic with the
// driver-depart Spanner mutation.
func FormatManifestDispatched(manifestID string, stopCount int) FormattedNotification {
	return newFormattedNotification(
		"Manifest Dispatched",
		fmt.Sprintf("Manifest %s departed the gate (%d stops).", manifestID, stopCount),
		"notification.manifest_dispatched.title",
		"notification.manifest_dispatched.body",
		map[string]string{"manifest_id": manifestID, "stop_count": fmt.Sprintf("%d", stopCount)},
	)
}

// FormatManifestCompleted notifies the supplier admin that every stop on the
// manifest has reached COMPLETED — the manifest lifecycle is terminal.
func FormatManifestCompleted(manifestID string, stopCount int) FormattedNotification {
	return newFormattedNotification(
		"Manifest Completed",
		fmt.Sprintf("Manifest %s closed: all %d stops delivered.", manifestID, stopCount),
		"notification.manifest_completed.title",
		"notification.manifest_completed.body",
		map[string]string{"manifest_id": manifestID, "stop_count": fmt.Sprintf("%d", stopCount)},
	)
}

// FormatManifestSettled notifies the supplier admin that the financial settlement
// for a manifest has been finalised — all per-order ledger splits are reconciled.
func FormatManifestSettled(manifestID string, supplierPayout int64, currency string) FormattedNotification {
	return newFormattedNotification(
		"Payout Settled",
		fmt.Sprintf("Manifest %s settled. Supplier payout: %.2f %s.", manifestID, float64(supplierPayout)/100.0, currency),
		"notification.manifest_settled.title",
		"notification.manifest_settled.body",
		map[string]string{"manifest_id": manifestID, "supplier_payout_minor": fmt.Sprintf("%d", supplierPayout), "currency": currency},
	)
}

// FormatOrderCancelledByOrigin — hard-kill notification: admin cancelled the order
// before dispatch. Goes 3-way: warehouse + supplier + retailer.
func FormatOrderCancelledByOrigin(orderID, reason string) FormattedNotification {
	return newFormattedNotification(
		"Order Cancelled",
		fmt.Sprintf("Order %s cancelled by origin: %s. Pending payment voided.", orderID, reason),
		"notification.order_cancelled_by_origin.title",
		"notification.order_cancelled_by_origin.body",
		map[string]string{"order_id": orderID, "reason": reason},
	)
}

// FormatPayloadOverflow — soft-stop notification: order didn't fit the truck.
// Returns to unassigned pool for redispatch.
func FormatPayloadOverflow(orderID, manifestID string) FormattedNotification {
	return newFormattedNotification(
		"Payload Overflow",
		fmt.Sprintf("Order %s removed from manifest %s (payload overflow). Queued for redispatch.", orderID, manifestID),
		"notification.payload_overflow.title",
		"notification.payload_overflow.body",
		map[string]string{"order_id": orderID, "manifest_id": manifestID},
	)
}
