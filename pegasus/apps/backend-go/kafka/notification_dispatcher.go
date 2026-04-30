package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"backend-go/kafka/workerpool"
	"backend-go/notifications"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	goKafka "github.com/segmentio/kafka-go"
)

// ─── Notification Dispatcher Consumer ──────────────────────────────────────────
// Listens on lab-logistics-events and dispatches notifications for all event types
// to the appropriate recipients via:
//   1. Spanner Notifications table (persistent inbox)
//   2. WebSocket push (real-time toast)
//   3. Telegram (fallback for users with TelegramChatId configured)

// NotificationDeps holds all dependencies for the notification dispatcher.
type NotificationDeps struct {
	RetailerHub   *ws.RetailerHub
	DriverHub     *ws.DriverHub
	PayloaderHub  *ws.PayloaderHub
	FCM           *notifications.FCMClient
	Telegram      *notifications.TelegramClient
	SpannerClient *spanner.Client
}

// StartNotificationDispatcher boots the partition-parallel Kafka consumer that
// dispatches notifications for all event types emitted by the order and payment
// services. Returns immediately; the pool runs until ctx is cancelled.
func StartNotificationDispatcher(ctx context.Context, deps NotificationDeps, brokerAddress string) {
	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    "lab-logistics-events",
		GroupID:  "lab-notification-dispatcher-group",
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "notification-dispatcher",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, m goKafka.Message) error {
			eventType := EventType(m.Headers, m.Key)
			switch eventType {
			case EventOrderDispatched:
				handleOrderDispatched(deps, m.Value)
			case EventDriverArrived:
				handleDriverArrived(deps, m.Value)
			case EventOrderStatusChanged:
				handleOrderStatusChanged(deps, m.Value)
			case EventPayloadReadyToSeal:
				handlePayloadReadyToSeal(deps, m.Value)
			case EventPayloadSealed:
				handlePayloadSealed(deps, m.Value)
			case EventPaymentSettled:
				handlePaymentSettled(deps, m.Value)
			case EventPaymentFailed:
				handlePaymentFailed(deps, m.Value)
			case EventCashCollectionRequired:
				handleCashCollectionRequired(deps, m.Value)
			case EventFulfillmentPaymentCompleted:
				handleFulfillmentPaymentCompleted(deps, m.Value)
			case EventFulfillmentPaid:
				handleFulfillmentPaid(deps, m.Value)
			case EventDriverAvailabilityChanged:
				handleDriverAvailabilityChanged(deps, m.Value)
			case EventOrderReassigned:
				handleOrderReassigned(deps, m.Value)
			case EventOrderModified:
				handleOrderModified(deps, m.Value)
			case EventWarehouseStatusChanged:
				handleWarehouseStatusChanged(deps, m.Value)
			case EventOutOfStock:
				handleOutOfStock(deps, m.Value)
			case EventRetailerPriceOverride:
				handleRetailerPriceOverride(deps, m.Value)
			case EventCancelRequested:
				handleCancelRequested(deps, m.Value)
			case EventCancelApproved:
				handleCancelApproved(deps, m.Value)
			case EventPaymentBypassIssued:
				handlePaymentBypassIssued(deps, m.Value)
			case EventPaymentBypassCompleted:
				handlePaymentBypassCompleted(deps, m.Value)
			case EventOrderCompleted:
				handleOrderCompleted(deps, m.Value)
			case EventOrderCreated:
				handleOrderCreated(deps, m.Value)
			case EventUnifiedCheckoutCompleted:
				handleUnifiedCheckoutCompleted(deps, m.Value)
			case EventStockBackordered:
				handleStockBackordered(deps, m.Value)
			case EventOrderCancelled:
				handleOrderCancelled(deps, m.Value)
			case EventOffloadConfirmed:
				handleOffloadConfirmed(deps, m.Value)
			case EventSmsQuickComplete:
				handleSMSQuickComplete(deps, m.Value)
			case EventEarlyCompleteRequested:
				handleEarlyCompleteRequested(deps, m.Value)
			case EventEarlyCompleteApproved:
				handleEarlyCompleteApproved(deps, m.Value)
			case EventNegotiationProposed:
				handleNegotiationProposed(deps, m.Value)
			case EventNegotiationResolved:
				handleNegotiationResolved(deps, m.Value)
			case EventCreditDeliveryMarked:
				handleCreditDeliveryMarked(deps, m.Value)
			case EventCreditDeliveryResolved:
				handleCreditDeliveryResolved(deps, m.Value)
			case EventMissingItemsReported:
				handleMissingItemsReported(deps, m.Value)
			case EventSplitPaymentCreated:
				handleSplitPaymentCreated(deps, m.Value)
			case EventAiOrderConfirmed:
				handleAiOrderConfirmed(deps, m.Value)
			case EventAiOrderRejected:
				handleAiOrderRejected(deps, m.Value)
			case EventShopClosed:
				handleShopClosed(deps, m.Value)
			case EventShopClosedResponse:
				handleShopClosedResponse(deps, m.Value)
			case EventPowerOutageReported:
				handlePowerOutageReported(deps, m.Value)
			case EventShopClosedEscalated:
				handleShopClosedEscalated(deps, m.Value)
			case EventShopClosedResolved:
				handleShopClosedResolved(deps, m.Value)
			case EventSupplyRequestSubmitted:
				handleSupplyRequestSubmitted(deps, m.Value)
			case EventPreOrderAutoAccepted:
				handlePreOrderAutoAccepted(deps, m.Value)
			case EventPreOrderConfirmed:
				handlePreOrderConfirmed(deps, m.Value)
			case EventPreOrderEdited:
				handlePreOrderEdited(deps, m.Value)
			case EventFleetDispatched:
				handleFleetDispatched(deps, m.Value)
			case EventDispatchLockAcquired:
				handleDispatchLockAcquired(deps, m.Value)
			case EventDispatchLockReleased:
				handleDispatchLockReleased(deps, m.Value)
			case EventFreezeLockAcquired:
				handleFreezeLockAcquired(deps, m.Value)
			case EventFreezeLockReleased:
				handleFreezeLockReleased(deps, m.Value)
			case EventDriverCreated:
				handleDriverCreated(deps, m.Value)
			case EventVehicleCreated:
				handleVehicleCreated(deps, m.Value)
			case EventManifestRebalanced:
				handleManifestRebalanced(deps, m.Value)
			case EventManifestDraftCreated:
				handleManifestDraftCreated(deps, m.Value)
			case EventManifestLoadingStarted:
				handleManifestLoadingStarted(deps, m.Value)
			case EventManifestSealed:
				handleManifestSealed(deps, m.Value)
			case EventManifestOrderException:
				handleManifestOrderException(deps, m.Value)
			case EventManifestOrderInjected:
				handleManifestOrderInjected(deps, m.Value)
			case EventManifestForceSeal:
				handleManifestForceSeal(deps, m.Value)
			case EventManifestDLQEscalation:
				handleManifestDLQEscalation(deps, m.Value)
			case EventManifestCancelled:
				handleManifestCancelled(deps, m.Value)
			case EventManifestDispatched:
				handleManifestDispatched(deps, m.Value)
			case EventManifestCompleted:
				handleManifestCompleted(deps, m.Value)
			case EventManifestSettled:
				handleManifestSettled(deps, m.Value)
			case EventForceSealAlert:
				handleForceSealAlert(deps, m.Value)
			case EventOrderDelayed:
				handleOrderDelayed(deps, m.Value)
			case EventPayloadSync:
				handlePayloadSync(deps, m.Value)
			case EventOrderCancelledByOrigin:
				handleOrderCancelledByOrigin(deps, m.Value)
			case EventPayloadOverflow:
				handlePayloadOverflow(deps, m.Value)
			case EventRouteCreated:
				handleRouteCreated(deps, m.Value)
			case EventFactoryManifestCreated:
				handleFactoryManifestCreated(deps, m.Value)
			case EventOrderAssigned:
				handleOrderAssigned(deps, m.Value)
			case EventRouteFinalized:
				handleRouteFinalized(deps, m.Value)
			case EventWarehouseCreated:
				handleWarehouseCreated(deps, m.Value)
			case EventWarehouseSpatialUpdated:
				handleWarehouseSpatialUpdated(deps, m.Value)
			case EventFactoryCreated:
				handleFactoryCreated(deps, m.Value)
			case EventRetailerRegistered:
				handleRetailerRegistered(deps, m.Value)
			case EventFactorySLABreach:
				handleFactorySLABreach(deps, m.Value)
			case EventInboundFreightUnannounced:
				handleInboundFreightUnannounced(deps, m.Value)
			case EventSupplyLaneTransitUpdated:
				handleSupplyLaneTransitUpdated(deps, m.Value)
			case EventTransferStateChanged:
				handleTransferStateChanged(deps, m.Value)
			case EventTransferApproved:
				handleTransferApproved(deps, m.Value)
			case EventTransferReceived:
				handleTransferReceived(deps, m.Value)
			case EventTransferUnassigned:
				handleTransferUnassigned(deps, m.Value)
			case EventNetworkModeChanged:
				handleNetworkModeChanged(deps, m.Value)
			case EventPullMatrixCompleted:
				handlePullMatrixCompleted(deps, m.Value)
			case EventReplenishmentTransferCreated:
				handleReplenishmentTransferCreated(deps, m.Value)
			case EventInsightApprovedTransferCreated:
				handleInsightApprovedTransferCreated(deps, m.Value)
			}
			return nil
		},
	})
	if err != nil {
		slog.Error("notification_dispatcher: pool init failed", "err", err)
		return
	}
	go func() {
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("notification_dispatcher: pool exited", "err", err)
		}
	}()
	slog.Info("notification dispatcher ONLINE", "topic", "lab-logistics-events")
}

// ─── Event Handlers ────────────────────────────────────────────────────────────

func handleOrderDispatched(deps NotificationDeps, data []byte) {
	var event OrderDispatchedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_DISPATCHED", "err", err)
		return
	}

	// Notify each retailer whose order was dispatched
	retailerMap := notifications.LookupRetailerIDsForOrders(deps.SpannerClient, event.OrderIDs)
	for _, retailerID := range retailerMap {
		notif := notifications.FormatOrderDispatched(event.RouteID, len(event.OrderIDs))
		dispatchToRecipient(deps, retailerID, "RETAILER", EventOrderDispatched, notif)
	}

	// Notify the driver of their new dispatch assignment
	if event.DriverID != "" {
		notif := notifications.FormatDriverDispatched(event.RouteID, len(event.OrderIDs))
		dispatchToRecipient(deps, event.DriverID, "DRIVER", EventOrderDispatched, notif)
	}
}

func handleDriverArrived(deps NotificationDeps, data []byte) {
	var event DriverArrivedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "DRIVER_ARRIVED", "err", err)
		return
	}

	notif := notifications.FormatDriverArrived(event.OrderID)
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventDriverArrived, notif)
}

func handleOrderStatusChanged(deps NotificationDeps, data []byte) {
	var event OrderStatusChangedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_STATUS_CHANGED", "err", err)
		return
	}

	notif := notifications.FormatOrderStatusChanged(event.OrderID, event.OldState, event.NewState)
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOrderStatusChanged, notif)
}

func handlePayloadReadyToSeal(deps NotificationDeps, data []byte) {
	var event PayloadReadyToSealEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYLOAD_READY_TO_SEAL", "err", err)
		return
	}

	// Notify supplier (admin portal) — payload sealing notification
	if event.SupplierID != "" {
		notif := notifications.FormatPayloadReadyToSeal(event.RouteID, len(event.OrderIDs))
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventPayloadReadyToSeal, notif)
	}
}

func handlePayloadSealed(deps NotificationDeps, data []byte) {
	var event PayloadSealedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYLOAD_SEALED", "err", err)
		return
	}

	// Look up supplierID from the order
	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "PAYLOAD_SEALED", "err", err)
		return
	}
	var sid spanner.NullString
	if err := row.Columns(&sid); err != nil || !sid.Valid {
		return
	}

	notif := notifications.FormatPayloadSealed(event.OrderID, event.TerminalID)
	dispatchToRecipient(deps, sid.StringVal, "SUPPLIER", EventPayloadSealed, notif)
}

func handlePaymentSettled(deps NotificationDeps, data []byte) {
	var event PaymentSettledEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYMENT_SETTLED", "err", err)
		return
	}

	// Notify retailer
	notif := notifications.FormatPaymentSettled(event.OrderID, event.Gateway, event.Amount)
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventPaymentSettled, notif)

	// Notify driver
	if event.DriverID != "" {
		driverNotif := notifications.FormattedNotification{
			Title: "Payment Received",
			Body:  "Payment confirmed for your delivery. You may proceed to complete the order.",
		}
		dispatchToRecipient(deps, event.DriverID, "DRIVER", EventPaymentSettled, driverNotif)
	}
}

func handlePaymentFailed(deps NotificationDeps, data []byte) {
	var event PaymentFailedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYMENT_FAILED", "err", err)
		return
	}

	notif := notifications.FormatPaymentFailed(event.OrderID, event.Gateway, event.Reason)
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventPaymentFailed, notif)
}

func handleDriverAvailabilityChanged(deps NotificationDeps, data []byte) {
	var event DriverAvailabilityChangedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "DRIVER_AVAILABILITY_CHANGED", "err", err)
		return
	}

	// Look up driver name for the notification
	driverName := event.DriverID[:8] // fallback
	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Drivers", spanner.Key{event.DriverID}, []string{"FullName"})
	if err == nil {
		var name spanner.NullString
		if err := row.Columns(&name); err == nil && name.Valid {
			driverName = name.StringVal
		}
	}

	var notif notifications.FormattedNotification
	if event.Available {
		notif = notifications.FormatDriverOnline(driverName)
	} else {
		notif = notifications.FormatDriverOffline(driverName, event.Reason)
	}
	// Notify the supplier (admin portal)
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventDriverAvailabilityChanged, notif)
}

func handleOrderReassigned(deps NotificationDeps, data []byte) {
	var event OrderReassignedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_REASSIGNED", "err", err)
		return
	}

	orderLabel := "orders"
	if len(event.OrderIDs) == 1 {
		orderLabel = event.OrderIDs[0]
	} else if len(event.OrderIDs) > 1 {
		orderLabel = fmt.Sprintf("%d orders", len(event.OrderIDs))
	}

	// Notify old driver: order(s) removed from their route
	if event.OldDriverID != "" {
		notif := notifications.FormatOrderReassignedRemoved(orderLabel)
		dispatchToRecipient(deps, event.OldDriverID, "DRIVER", EventOrderReassigned, notif)
	}

	// Notify new driver: new order(s) added to their route
	if event.NewDriverID != "" {
		notif := notifications.FormatOrderReassignedAdded(orderLabel)
		dispatchToRecipient(deps, event.NewDriverID, "DRIVER", EventOrderReassigned, notif)
	}
}

func handleOrderModified(deps NotificationDeps, data []byte) {
	var event OrderModifiedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_MODIFIED", "err", err)
		return
	}

	notif := notifications.FormatOrderAmended(event.OrderID, event.Refunded)

	// Notify supplier of the amendment
	if event.SupplierID != "" {
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventOrderModified, notif)
	}

	// Notify retailer of the adjusted invoice
	if event.RetailerID != "" {
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOrderModified, notif)
	}
}

func handleWarehouseStatusChanged(deps NotificationDeps, data []byte) {
	var event WarehouseStatusChangedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "WAREHOUSE_STATUS_CHANGED", "err", err)
		return
	}

	var notif notifications.FormattedNotification
	if event.Field == "is_active" && !event.NewValue {
		notif = notifications.FormattedNotification{
			Title: "Warehouse Disabled",
			Body:  "Warehouse " + event.WarehouseId[:8] + " has been disabled. Orders may be rerouted.",
		}
	} else if event.Field == "is_on_shift" && !event.NewValue {
		notif = notifications.FormattedNotification{
			Title: "Warehouse Off Shift",
			Body:  "Warehouse " + event.WarehouseId[:8] + " is now off shift.",
		}
	} else if event.Field == "is_active" && event.NewValue {
		notif = notifications.FormattedNotification{
			Title: "Warehouse Enabled",
			Body:  "Warehouse " + event.WarehouseId[:8] + " is back online.",
		}
	} else {
		notif = notifications.FormattedNotification{
			Title: "Warehouse On Shift",
			Body:  "Warehouse " + event.WarehouseId[:8] + " is now on shift.",
		}
	}

	// Notify supplier (admin portal)
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventWarehouseStatusChanged, notif)
}

func handleOutOfStock(deps NotificationDeps, data []byte) {
	var event OutOfStockEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "OUT_OF_STOCK", "err", err)
		return
	}

	notif := notifications.FormattedNotification{
		Title: "Items Out of Stock",
		Body:  "Your checkout was blocked because all requested items are out of stock at warehouse " + event.WarehouseId[:8] + ".",
	}
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOutOfStock, notif)

	// Also alert the supplier
	supplierNotif := notifications.FormattedNotification{
		Title: "Out of Stock Alert",
		Body:  "Retailer checkout blocked — all items OOS at warehouse " + event.WarehouseId[:8] + ".",
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventOutOfStock, supplierNotif)
}

func handleRetailerPriceOverride(deps NotificationDeps, data []byte) {
	var event RetailerPriceOverrideEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "RETAILER_PRICE_OVERRIDE", "err", err)
		return
	}

	var notif notifications.FormattedNotification
	if event.Action == "CREATED" {
		notif = notifications.FormattedNotification{
			Title: "Custom Pricing Applied",
			Body:  "A custom price has been set for you on product " + event.SkuId[:8] + ".",
		}
	} else {
		notif = notifications.FormattedNotification{
			Title: "Custom Pricing Removed",
			Body:  "Custom pricing for product " + event.SkuId[:8] + " has been removed. Standard pricing applies.",
		}
	}
	dispatchToRecipient(deps, event.RetailerId, "RETAILER", EventRetailerPriceOverride, notif)
}

func handleCancelRequested(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		RetailerID string `json:"retailer_id"`
		SupplierID string `json:"supplier_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "CANCEL_REQUESTED", "err", err)
		return
	}
	// Look up supplier from order if not in payload
	supplierID := event.SupplierID
	if supplierID == "" {
		ctx := context.Background()
		row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"SupplierId"})
		if err == nil {
			var sid spanner.NullString
			_ = row.Columns(&sid)
			supplierID = sid.StringVal
		}
	}
	if supplierID != "" {
		notif := notifications.FormattedNotification{
			Title: "Cancel Request",
			Body:  "A retailer has requested cancellation for order " + event.OrderID[:min(8, len(event.OrderID))] + ". Review and approve or deny.",
		}
		dispatchToRecipient(deps, supplierID, "SUPPLIER", EventCancelRequested, notif)
	}
}

func handleCancelApproved(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		RetailerID string `json:"retailer_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "CANCEL_APPROVED", "err", err)
		return
	}
	retailerID := event.RetailerID
	if retailerID == "" {
		ctx := context.Background()
		row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId"})
		if err == nil {
			var rid spanner.NullString
			_ = row.Columns(&rid)
			retailerID = rid.StringVal
		}
	}
	if retailerID != "" {
		notif := notifications.FormattedNotification{
			Title: "Order Cancelled",
			Body:  "Your cancellation request for order " + event.OrderID[:min(8, len(event.OrderID))] + " has been approved.",
		}
		dispatchToRecipient(deps, retailerID, "RETAILER", EventCancelApproved, notif)
	}
}

func handleOrderCompleted(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		RetailerID string `json:"retailer_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_COMPLETED", "err", err)
		return
	}
	if event.RetailerID != "" {
		notif := notifications.FormattedNotification{
			Title: "Delivery Complete",
			Body:  "Your order " + event.OrderID[:min(8, len(event.OrderID))] + " has been delivered successfully.",
		}
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOrderCompleted, notif)
	}
}

func handlePaymentBypassIssued(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID  string `json:"order_id"`
		IssuedBy string `json:"issued_by"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYMENT_BYPASS_ISSUED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "PAYMENT_BYPASS_ISSUED", "order_id", event.OrderID, "err", err)
		return
	}
	var retailerID spanner.NullString
	if err := row.Columns(&retailerID); err != nil || !retailerID.Valid {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	body := fmt.Sprintf("Supplier issued a payment bypass token for order %s.", orderRef)
	if event.Reason != "" {
		body = fmt.Sprintf("Supplier issued a payment bypass token for order %s (%s).", orderRef, event.Reason)
	}
	dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventPaymentBypassIssued,
		notifications.FormattedNotification{
			Title: "Payment Bypass Issued",
			Body:  body,
		})
}

func handlePaymentBypassCompleted(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID  string `json:"order_id"`
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYMENT_BYPASS_COMPLETED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId", "SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "PAYMENT_BYPASS_COMPLETED", "order_id", event.OrderID, "err", err)
		return
	}
	var retailerID, supplierID spanner.NullString
	if err := row.Columns(&retailerID, &supplierID); err != nil {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	if retailerID.Valid {
		dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventPaymentBypassCompleted,
			notifications.FormattedNotification{
				Title: "Payment Completed",
				Body:  fmt.Sprintf("Payment for order %s was completed using bypass verification.", orderRef),
			})
	}
	if supplierID.Valid {
		dispatchToRecipient(deps, supplierID.StringVal, "SUPPLIER", EventPaymentBypassCompleted,
			notifications.FormattedNotification{
				Title: "Bypass Payment Completed",
				Body:  fmt.Sprintf("Order %s was completed with payment bypass confirmation.", orderRef),
			})
	}
}

func handleSMSQuickComplete(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID  string `json:"order_id"`
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SMS_QUICK_COMPLETE", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId", "SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "SMS_QUICK_COMPLETE", "order_id", event.OrderID, "err", err)
		return
	}
	var retailerID, supplierID spanner.NullString
	if err := row.Columns(&retailerID, &supplierID); err != nil {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	if retailerID.Valid {
		dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventSmsQuickComplete,
			notifications.FormattedNotification{
				Title: "Delivery Completed",
				Body:  fmt.Sprintf("Order %s was completed via SMS fallback confirmation.", orderRef),
			})
	}
	if supplierID.Valid {
		dispatchToRecipient(deps, supplierID.StringVal, "SUPPLIER", EventSmsQuickComplete,
			notifications.FormattedNotification{
				Title: "SMS Completion",
				Body:  fmt.Sprintf("Driver completed order %s using SMS quick-complete fallback.", orderRef),
			})
	}
}

func handleEarlyCompleteRequested(deps NotificationDeps, data []byte) {
	var event EarlyCompleteRequestedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "EARLY_COMPLETE_REQUESTED", "err", err)
		return
	}
	if event.SupplierID != "" {
		notif := notifications.FormattedNotification{
			Title: "Early Route Complete Request",
			Body:  "Driver requests early route completion. Reason: " + event.Reason + ". " + fmt.Sprintf("%d", len(event.OrderIDs)) + " orders remaining.",
		}
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventEarlyCompleteRequested, notif)
	}
}

func handleEarlyCompleteApproved(deps NotificationDeps, data []byte) {
	var event EarlyCompleteRequestedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "EARLY_COMPLETE_APPROVED", "err", err)
		return
	}
	if event.DriverID == "" {
		return
	}
	dispatchToRecipient(deps, event.DriverID, "DRIVER", EventEarlyCompleteApproved,
		notifications.FormattedNotification{
			Title: "Early Completion Approved",
			Body:  fmt.Sprintf("Supplier approved early completion for %d remaining orders. Return to your home node.", len(event.OrderIDs)),
		})
}

func handleNegotiationProposed(deps NotificationDeps, data []byte) {
	var event NegotiationProposedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "NEGOTIATION_PROPOSED", "err", err)
		return
	}
	if event.SupplierID != "" {
		notif := notifications.FormattedNotification{
			Title: "Quantity Negotiation",
			Body:  "Driver proposed quantity changes for order " + event.OrderID[:min(8, len(event.OrderID))] + ". Review and approve or reject.",
		}
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventNegotiationProposed, notif)
	}
	if event.RetailerID != "" {
		notif := notifications.FormattedNotification{
			Title: "Delivery Adjustment Proposed",
			Body:  "A quantity change has been proposed for your order " + event.OrderID[:min(8, len(event.OrderID))] + ".",
		}
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventNegotiationProposed, notif)
	}
}

func handleNegotiationResolved(deps NotificationDeps, data []byte) {
	var event NegotiationResolvedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "NEGOTIATION_RESOLVED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "NEGOTIATION_RESOLVED", "order_id", event.OrderID, "err", err)
		return
	}
	var retailerID spanner.NullString
	if err := row.Columns(&retailerID); err != nil || !retailerID.Valid {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	action := "updated"
	if event.Action == "APPROVED" {
		action = "approved"
	} else if event.Action == "REJECTED" {
		action = "rejected"
	}

	dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventNegotiationResolved,
		notifications.FormattedNotification{
			Title: "Negotiation Resolved",
			Body:  fmt.Sprintf("Supplier %s your quantity negotiation for order %s.", action, orderRef),
		})
}

func handleCreditDeliveryMarked(deps NotificationDeps, data []byte) {
	var event CreditDeliveryEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "CREDIT_DELIVERY_MARKED", "err", err)
		return
	}
	if event.SupplierID != "" {
		notif := notifications.FormattedNotification{
			Title: "Credit Delivery",
			Body:  "Order " + event.OrderID[:min(8, len(event.OrderID))] + " delivered on credit. Awaiting supplier decision.",
		}
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventCreditDeliveryMarked, notif)
	}
}

func handleCreditDeliveryResolved(deps NotificationDeps, data []byte) {
	var event CreditDeliveryEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "CREDIT_DELIVERY_RESOLVED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"DriverId", "RetailerId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "CREDIT_DELIVERY_RESOLVED", "order_id", event.OrderID, "err", err)
		return
	}
	var driverID, retailerID spanner.NullString
	if err := row.Columns(&driverID, &retailerID); err != nil {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	action := "APPROVED"
	if event.Action == "DENY" {
		action = "DENIED"
	}
	body := fmt.Sprintf("Credit delivery for order %s was %s by supplier.", orderRef, action)

	if driverID.Valid {
		dispatchToRecipient(deps, driverID.StringVal, "DRIVER", EventCreditDeliveryResolved,
			notifications.FormattedNotification{Title: "Credit Delivery Resolved", Body: body})
	}
	if retailerID.Valid {
		dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventCreditDeliveryResolved,
			notifications.FormattedNotification{Title: "Credit Delivery Update", Body: body})
	}
}

func handleMissingItemsReported(deps NotificationDeps, data []byte) {
	var event MissingItemsEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MISSING_ITEMS_REPORTED", "err", err)
		return
	}
	if event.SupplierID != "" {
		notif := notifications.FormattedNotification{
			Title: "Missing Items Report",
			Body:  fmt.Sprintf("Driver reports %d items missing from order %s after seal.", event.ItemCount, event.OrderID[:min(8, len(event.OrderID))]),
		}
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventMissingItemsReported, notif)
	}
}

func handleSplitPaymentCreated(deps NotificationDeps, data []byte) {
	var event SplitPaymentEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SPLIT_PAYMENT_CREATED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId", "SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "SPLIT_PAYMENT_CREATED", "order_id", event.OrderID, "err", err)
		return
	}
	var retailerID, supplierID spanner.NullString
	if err := row.Columns(&retailerID, &supplierID); err != nil {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	body := fmt.Sprintf("Split payment created for order %s: first %d, second %d.", orderRef, event.FirstAmount, event.SecondAmount)
	if retailerID.Valid {
		dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventSplitPaymentCreated,
			notifications.FormattedNotification{Title: "Split Payment Created", Body: body})
	}
	if supplierID.Valid {
		dispatchToRecipient(deps, supplierID.StringVal, "SUPPLIER", EventSplitPaymentCreated,
			notifications.FormattedNotification{Title: "Split Payment Created", Body: body})
	}
}

func handleAiOrderConfirmed(deps NotificationDeps, data []byte) {
	var event AiOrderEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "AI_ORDER_CONFIRMED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "AI_ORDER_CONFIRMED", "order_id", event.OrderID, "err", err)
		return
	}
	var supplierID spanner.NullString
	if err := row.Columns(&supplierID); err != nil || !supplierID.Valid {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, supplierID.StringVal, "SUPPLIER", EventAiOrderConfirmed,
		notifications.FormattedNotification{
			Title: "AI Order Confirmed",
			Body:  fmt.Sprintf("Retailer confirmed AI-suggested order %s.", orderRef),
		})
}

func handleAiOrderRejected(deps NotificationDeps, data []byte) {
	var event AiOrderEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "AI_ORDER_REJECTED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "AI_ORDER_REJECTED", "order_id", event.OrderID, "err", err)
		return
	}
	var supplierID spanner.NullString
	if err := row.Columns(&supplierID); err != nil || !supplierID.Valid {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	body := fmt.Sprintf("Retailer rejected AI-suggested order %s.", orderRef)
	if event.Reason != "" {
		body = fmt.Sprintf("Retailer rejected AI-suggested order %s (%s).", orderRef, event.Reason)
	}
	dispatchToRecipient(deps, supplierID.StringVal, "SUPPLIER", EventAiOrderRejected,
		notifications.FormattedNotification{Title: "AI Order Rejected", Body: body})
}

func handleShopClosed(deps NotificationDeps, data []byte) {
	var event ShopClosedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SHOP_CLOSED", "err", err)
		return
	}
	if event.SupplierID == "" || event.OrderID == "" {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventShopClosed,
		notifications.FormattedNotification{
			Title: "Shop Closed Reported",
			Body:  fmt.Sprintf("Driver reported shop closed for order %s. Attempt %s requires follow-up.", orderRef, event.AttemptID),
		})
}

func handlePowerOutageReported(deps NotificationDeps, data []byte) {
	var event ShopClosedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "POWER_OUTAGE_REPORTED", "err", err)
		return
	}
	if event.SupplierID == "" || event.OrderID == "" {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventPowerOutageReported,
		notifications.FormattedNotification{
			Title: "Power Outage Reported",
			Body:  fmt.Sprintf("Driver reported probable power outage at retailer for order %s.", orderRef),
		})
}

func handleShopClosedEscalated(deps NotificationDeps, data []byte) {
	var event ShopClosedEscalatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SHOP_CLOSED_ESCALATED", "err", err)
		return
	}
	if event.EscalatedTo == "" {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.EscalatedTo, "SUPPLIER", EventShopClosedEscalated,
		notifications.FormattedNotification{
			Title: "Shop Closed Escalation",
			Body:  fmt.Sprintf("Order %s was escalated for immediate supplier action.", orderRef),
		})
}

func handleOffloadConfirmed(deps NotificationDeps, data []byte) {
	var event OffloadConfirmedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "OFFLOAD_CONFIRMED", "err", err)
		return
	}
	if event.RetailerID == "" || event.OrderID == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOffloadConfirmed,
		notifications.FormattedNotification{
			Title: "Offload Confirmed",
			Body:  fmt.Sprintf("Offload confirmed for order %s. Payment flow is now active.", orderRef),
		})
}

func handleSupplyRequestSubmitted(deps NotificationDeps, data []byte) {
	var event SupplyRequestEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SUPPLY_REQUEST_SUBMITTED", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventSupplyRequestSubmitted,
		notifications.FormattedNotification{
			Title: "Supply Request Submitted",
			Body:  fmt.Sprintf("Supply request %s moved to %s priority %s.", event.RequestID, event.State, event.Priority),
		})
}

func handleManifestDraftCreated(deps NotificationDeps, data []byte) {
	var event ManifestLifecycleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_DRAFT_CREATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestDraftCreated,
		notifications.FormattedNotification{
			Title: "Manifest Draft Created",
			Body:  fmt.Sprintf("Manifest %s draft created with %d planned stops.", event.ManifestID, event.StopCount),
		})
}

func handleManifestLoadingStarted(deps NotificationDeps, data []byte) {
	var event ManifestLifecycleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_LOADING_STARTED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestLoadingStarted,
		notifications.FormattedNotification{
			Title: "Manifest Loading Started",
			Body:  fmt.Sprintf("Loading started for manifest %s.", event.ManifestID),
		})
}

func handleManifestSealed(deps NotificationDeps, data []byte) {
	var event ManifestLifecycleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_SEALED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestSealed,
		notifications.FormattedNotification{
			Title: "Manifest Sealed",
			Body:  fmt.Sprintf("Manifest %s sealed at %.1f/%.1f VU.", event.ManifestID, event.VolumeVU, event.MaxVolumeVU),
		})
}

func handleManifestOrderException(deps NotificationDeps, data []byte) {
	var event ManifestOrderExceptionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_ORDER_EXCEPTION", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestOrderException,
		notifications.FormattedNotification{
			Title: "Manifest Exception",
			Body:  fmt.Sprintf("Exception %s on order %s in manifest %s.", event.Reason, orderRef, event.ManifestID),
		})
}

func handleManifestOrderInjected(deps NotificationDeps, data []byte) {
	var event ManifestOrderInjectedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_ORDER_INJECTED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestOrderInjected,
		notifications.FormattedNotification{
			Title: "Order Injected Into Manifest",
			Body:  fmt.Sprintf("Order %s was injected into manifest %s.", orderRef, event.ManifestID),
		})
}

func handleManifestForceSeal(deps NotificationDeps, data []byte) {
	var event ManifestForceSealEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_FORCE_SEALED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestForceSeal,
		notifications.FormattedNotification{
			Title: "Manifest Force-Sealed",
			Body:  fmt.Sprintf("Manifest %s force-sealed by %s (%s).", event.ManifestID, event.SealedBy, event.Reason),
		})
}

func handleManifestDLQEscalation(deps NotificationDeps, data []byte) {
	var event ManifestOrderExceptionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_DLQ_ESCALATION", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestDLQEscalation,
		notifications.FormattedNotification{
			Title: "Manifest DLQ Escalation",
			Body:  fmt.Sprintf("Order %s on manifest %s exceeded retry threshold and requires intervention.", orderRef, event.ManifestID),
		})
}

func handleRouteCreated(deps NotificationDeps, data []byte) {
	var event RouteCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ROUTE_CREATED", "err", err)
		return
	}
	if event.SupplierID != "" {
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventRouteCreated,
			notifications.FormattedNotification{
				Title: "Route Created",
				Body:  fmt.Sprintf("Route %s created with %d stops.", event.RouteID, event.StopCount),
			})
	}
	if event.DriverID != "" {
		dispatchToRecipient(deps, event.DriverID, "DRIVER", EventRouteCreated,
			notifications.FormattedNotification{
				Title: "New Route Assigned",
				Body:  fmt.Sprintf("Route %s is ready with %d planned stops.", event.RouteID, event.StopCount),
			})
	}
}

func handleFactoryManifestCreated(deps NotificationDeps, data []byte) {
	var event RouteCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FACTORY_MANIFEST_CREATED", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventFactoryManifestCreated,
		notifications.FormattedNotification{
			Title: "Factory Manifest Created",
			Body:  fmt.Sprintf("Factory manifest %s created with %d convoy stops.", event.RouteID, event.StopCount),
		})
}

func handleOrderAssigned(deps NotificationDeps, data []byte) {
	var event OrderAssignedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_ASSIGNED", "err", err)
		return
	}
	if event.DriverID == "" || event.OrderID == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.DriverID, "DRIVER", EventOrderAssigned,
		notifications.FormattedNotification{
			Title: "Order Assigned",
			Body:  fmt.Sprintf("Order %s assigned to route %s.", orderRef, event.RouteID),
		})
}

func handleRouteFinalized(deps NotificationDeps, data []byte) {
	var event RouteFinalizedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ROUTE_FINALIZED", "err", err)
		return
	}
	if event.DriverID == "" {
		return
	}
	dispatchToRecipient(deps, event.DriverID, "DRIVER", EventRouteFinalized,
		notifications.FormattedNotification{
			Title: "Route Finalized",
			Body:  fmt.Sprintf("Manifest %s route finalized with %d stops.", event.ManifestID, event.StopCount),
		})
}

func handleWarehouseCreated(deps NotificationDeps, data []byte) {
	var event WarehouseCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "WAREHOUSE_CREATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventWarehouseCreated,
		notifications.FormattedNotification{
			Title: "Warehouse Created",
			Body:  fmt.Sprintf("Warehouse %s is now active for supplier operations.", event.Name),
		})
}

func handleFactoryCreated(deps NotificationDeps, data []byte) {
	var event FactoryCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FACTORY_CREATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventFactoryCreated,
		notifications.FormattedNotification{
			Title: "Factory Created",
			Body:  fmt.Sprintf("Factory %s is now active (%d warehouses linked).", event.Name, event.WarehousesLinked),
		})
}

func handleRetailerRegistered(deps NotificationDeps, data []byte) {
	var event RetailerRegisteredEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "RETAILER_REGISTERED", "err", err)
		return
	}
	if event.RetailerId == "" {
		return
	}
	// Retailer self-registration is not scoped to a single supplier — broadcast
	// to the retailer themselves as a welcome receipt. Supplier-side discovery
	// of new retailers happens via the catalog indexer (separate consumer).
	dispatchToRecipient(deps, event.RetailerId, "RETAILER", EventRetailerRegistered,
		notifications.FormattedNotification{
			Title: "Welcome to The Lab",
			Body:  fmt.Sprintf("Account %s registered. You can now place orders.", event.ShopName),
		})
}

func handleWarehouseSpatialUpdated(deps NotificationDeps, data []byte) {
	var event WarehouseSpatialUpdatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "WAREHOUSE_SPATIAL_UPDATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventWarehouseSpatialUpdated,
		notifications.FormattedNotification{
			Title: "Warehouse Coverage Updated",
			Body:  fmt.Sprintf("Warehouse %s coverage updated: H3 %d → %d.", event.WarehouseId, event.OldH3Count, event.NewH3Count),
		})
}

func handleFactorySLABreach(deps NotificationDeps, data []byte) {
	var event FactorySLABreachEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FACTORY_SLA_BREACH", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventFactorySLABreach,
		notifications.FormattedNotification{
			Title: "Factory SLA Breach",
			Body:  fmt.Sprintf("Transfer %s breached SLA at %s level.", event.TransferId, event.EscalationLevel),
		})
}

func handleInboundFreightUnannounced(deps NotificationDeps, data []byte) {
	var event InboundFreightUnannouncedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "INBOUND_FREIGHT_UNANNOUNCED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventInboundFreightUnannounced,
		notifications.FormattedNotification{
			Title: "Unannounced Freight Received",
			Body:  fmt.Sprintf("Warehouse force-received transfer %s with %d items.", event.TransferId, event.ItemsCount),
		})
}

func handleSupplyLaneTransitUpdated(deps NotificationDeps, data []byte) {
	var event SupplyLaneTransitUpdatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SUPPLY_LANE_TRANSIT_UPDATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventSupplyLaneTransitUpdated,
		notifications.FormattedNotification{
			Title: "Supply Lane Updated",
			Body:  fmt.Sprintf("Transit estimate updated for lane %s to %.1fh.", event.LaneId, event.NewDampenedHours),
		})
}

func handleNetworkModeChanged(deps NotificationDeps, data []byte) {
	var event NetworkModeChangedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "NETWORK_MODE_CHANGED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventNetworkModeChanged,
		notifications.FormattedNotification{
			Title: "Network Mode Changed",
			Body:  fmt.Sprintf("Optimization mode changed from %s to %s.", event.OldMode, event.NewMode),
		})
}

func handlePullMatrixCompleted(deps NotificationDeps, data []byte) {
	var event PullMatrixCompletedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PULL_MATRIX_COMPLETED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventPullMatrixCompleted,
		notifications.FormattedNotification{
			Title: "Pull Matrix Completed",
			Body:  fmt.Sprintf("Run %s generated %d transfers across %d SKUs.", event.RunId, event.TransfersGenerated, event.SKUsProcessed),
		})
}

func handleReplenishmentTransferCreated(deps NotificationDeps, data []byte) {
	var event struct {
		TransferID  string `json:"transfer_id"`
		SupplierID  string `json:"supplier_id"`
		WarehouseID string `json:"warehouse_id"`
		Source      string `json:"source"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "REPLENISHMENT_TRANSFER_CREATED", "err", err)
		return
	}
	if event.SupplierID == "" || event.TransferID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventReplenishmentTransferCreated,
		notifications.FormattedNotification{
			Title: "Replenishment Transfer Created",
			Body:  fmt.Sprintf("Transfer %s created for warehouse %s (%s).", event.TransferID, event.WarehouseID, event.Source),
		})
}

func handleInsightApprovedTransferCreated(deps NotificationDeps, data []byte) {
	var event struct {
		InsightID   string `json:"insight_id"`
		TransferID  string `json:"transfer_id"`
		WarehouseID string `json:"warehouse_id"`
		SupplierID  string `json:"supplier_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "INSIGHT_APPROVED_TRANSFER_CREATED", "err", err)
		return
	}
	if event.TransferID == "" {
		return
	}

	supplierID := event.SupplierID
	if supplierID == "" {
		ctx := context.Background()
		row, err := deps.SpannerClient.Single().ReadRow(ctx, "InternalTransferOrders", spanner.Key{event.TransferID}, []string{"SupplierId"})
		if err == nil {
			var sid spanner.NullString
			if row.Columns(&sid) == nil && sid.Valid {
				supplierID = sid.StringVal
			}
		}
	}
	if supplierID == "" {
		return
	}

	dispatchToRecipient(deps, supplierID, "SUPPLIER", EventInsightApprovedTransferCreated,
		notifications.FormattedNotification{
			Title: "Insight Transfer Created",
			Body:  fmt.Sprintf("Insight %s created transfer %s for warehouse %s.", event.InsightID, event.TransferID, event.WarehouseID),
		})
}

func handleCashCollectionRequired(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		RetailerID string `json:"retailer_id"`
		Amount     int64  `json:"amount"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "CASH_COLLECTION_REQUIRED", "err", err)
		return
	}
	if event.RetailerID == "" || event.OrderID == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventCashCollectionRequired,
		notifications.FormattedNotification{
			Title: "Cash Collection Required",
			Body:  fmt.Sprintf("Order %s is awaiting cash collection for %d.", orderRef, event.Amount),
		})
}

func handleFulfillmentPaymentCompleted(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		SupplierID string `json:"supplier_id"`
		RetailerID string `json:"retailer_id"`
		DriverID   string `json:"driver_id"`
		Amount     int64  `json:"amount"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FULFILLMENT_PAYMENT_COMPLETED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	if event.RetailerID != "" {
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventFulfillmentPaymentCompleted,
			notifications.FormattedNotification{
				Title: "Payment Completed",
				Body:  fmt.Sprintf("Payment for order %s was completed successfully.", orderRef),
			})
	}
	if event.SupplierID != "" {
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventFulfillmentPaymentCompleted,
			notifications.FormattedNotification{
				Title: "Fulfillment Payment Completed",
				Body:  fmt.Sprintf("Order %s payment completed for %d.", orderRef, event.Amount),
			})
	}
	if event.DriverID != "" {
		dispatchToRecipient(deps, event.DriverID, "DRIVER", EventFulfillmentPaymentCompleted,
			notifications.FormattedNotification{
				Title: "Retailer Payment Confirmed",
				Body:  fmt.Sprintf("Order %s payment is confirmed. Continue route execution.", orderRef),
			})
	}
}

func handleFulfillmentPaid(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		SupplierID string `json:"supplier_id"`
		RetailerID string `json:"retailer_id"`
		Amount     int64  `json:"amount"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FULFILLMENT_PAID", "err", err)
		return
	}
	if event.SupplierID == "" || event.OrderID == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventFulfillmentPaid,
		notifications.FormattedNotification{
			Title: "Fulfillment Paid",
			Body:  fmt.Sprintf("Order %s marked paid: %d.", orderRef, event.Amount),
		})
}

func handleOrderCreated(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID    string `json:"order_id"`
		SupplierID string `json:"supplier_id"`
		RetailerID string `json:"retailer_id"`
		Total      int64  `json:"total"`
		Currency   string `json:"currency"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_CREATED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}
	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	if event.SupplierID != "" {
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventOrderCreated,
			notifications.FormattedNotification{
				Title: "New Order Created",
				Body:  fmt.Sprintf("Order %s created for %d %s.", orderRef, event.Total, event.Currency),
			})
	}
	if event.RetailerID != "" {
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOrderCreated,
			notifications.FormattedNotification{
				Title: "Order Placed",
				Body:  fmt.Sprintf("Order %s was placed successfully.", orderRef),
			})
	}
}

func handleUnifiedCheckoutCompleted(deps NotificationDeps, data []byte) {
	var event struct {
		InvoiceID  string `json:"invoice_id"`
		RetailerID string `json:"retailer_id"`
		Total      int64  `json:"total"`
		Currency   string `json:"currency"`
		OrderCount int    `json:"order_count"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "UNIFIED_CHECKOUT_COMPLETED", "err", err)
		return
	}
	if event.RetailerID == "" {
		return
	}
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventUnifiedCheckoutCompleted,
		notifications.FormattedNotification{
			Title: "Checkout Completed",
			Body:  fmt.Sprintf("Checkout completed: %d orders, total %d %s.", event.OrderCount, event.Total, event.Currency),
		})
}

func handleStockBackordered(deps NotificationDeps, data []byte) {
	var event struct {
		BackOrderID string `json:"backorder_id"`
		SupplierID  string `json:"supplier_id"`
		RetailerID  string `json:"retailer_id"`
		Total       int64  `json:"total"`
		Currency    string `json:"currency"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "STOCK_BACKORDERED", "err", err)
		return
	}
	if event.BackOrderID == "" {
		return
	}
	orderRef := event.BackOrderID[:min(8, len(event.BackOrderID))]
	if event.RetailerID != "" {
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventStockBackordered,
			notifications.FormattedNotification{
				Title: "Backorder Created",
				Body:  fmt.Sprintf("Backorder %s created for %d %s.", orderRef, event.Total, event.Currency),
			})
	}
	if event.SupplierID != "" {
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventStockBackordered,
			notifications.FormattedNotification{
				Title: "Backorder Added",
				Body:  fmt.Sprintf("Backorder %s was created from shortfall handling.", orderRef),
			})
	}
}

func handleOrderCancelled(deps NotificationDeps, data []byte) {
	var event struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_CANCELLED", "err", err)
		return
	}
	if event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{event.OrderID}, []string{"RetailerId", "SupplierId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "ORDER_CANCELLED", "order_id", event.OrderID, "err", err)
		return
	}
	var retailerID, supplierID spanner.NullString
	if err := row.Columns(&retailerID, &supplierID); err != nil {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	if retailerID.Valid {
		dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventOrderCancelled,
			notifications.FormattedNotification{
				Title: "Order Cancelled",
				Body:  fmt.Sprintf("Order %s has been cancelled.", orderRef),
			})
	}
	if supplierID.Valid {
		dispatchToRecipient(deps, supplierID.StringVal, "SUPPLIER", EventOrderCancelled,
			notifications.FormattedNotification{
				Title: "Order Cancelled",
				Body:  fmt.Sprintf("Order %s was cancelled and removed from active flow.", orderRef),
			})
	}
}

func handleShopClosedResponse(deps NotificationDeps, data []byte) {
	var event ShopClosedResponseEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SHOP_CLOSED_RESPONSE", "err", err)
		return
	}
	if event.AttemptID == "" || event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "ShopClosedAttempts", spanner.Key{event.AttemptID}, []string{"DriverId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "SHOP_CLOSED_RESPONSE", "attempt_id", event.AttemptID, "err", err)
		return
	}
	var driverID spanner.NullString
	if err := row.Columns(&driverID); err != nil || !driverID.Valid {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	dispatchToRecipient(deps, driverID.StringVal, "DRIVER", EventShopClosedResponse,
		notifications.FormattedNotification{
			Title: "Retailer Responded",
			Body:  fmt.Sprintf("Retailer response for order %s: %s.", orderRef, event.Response),
		})
}

func handleShopClosedResolved(deps NotificationDeps, data []byte) {
	var event ShopClosedResolvedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SHOP_CLOSED_RESOLVED", "err", err)
		return
	}
	if event.AttemptID == "" || event.OrderID == "" {
		return
	}

	ctx := context.Background()
	row, err := deps.SpannerClient.Single().ReadRow(ctx, "ShopClosedAttempts", spanner.Key{event.AttemptID}, []string{"DriverId", "RetailerId"})
	if err != nil {
		slog.Error("notification_dispatcher.lookup", "event", "SHOP_CLOSED_RESOLVED", "attempt_id", event.AttemptID, "err", err)
		return
	}
	var driverID, retailerID spanner.NullString
	if err := row.Columns(&driverID, &retailerID); err != nil {
		return
	}

	orderRef := event.OrderID[:min(8, len(event.OrderID))]
	body := fmt.Sprintf("Shop-closed case for order %s resolved as %s.", orderRef, event.Resolution)
	if driverID.Valid {
		dispatchToRecipient(deps, driverID.StringVal, "DRIVER", EventShopClosedResolved,
			notifications.FormattedNotification{Title: "Shop-Closed Resolved", Body: body})
	}
	if retailerID.Valid {
		dispatchToRecipient(deps, retailerID.StringVal, "RETAILER", EventShopClosedResolved,
			notifications.FormattedNotification{Title: "Shop-Closed Resolved", Body: body})
	}
}

func handleTransferStateChanged(deps NotificationDeps, data []byte) {
	var event struct {
		TransferID string `json:"transfer_id"`
		SupplierID string `json:"supplier_id"`
		FromState  string `json:"from_state"`
		ToState    string `json:"to_state"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "TRANSFER_STATE_CHANGED", "err", err)
		return
	}
	if event.SupplierID == "" || event.TransferID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventTransferStateChanged,
		notifications.FormattedNotification{
			Title: "Transfer State Updated",
			Body:  fmt.Sprintf("Transfer %s moved from %s to %s.", event.TransferID, event.FromState, event.ToState),
		})
}

func handleTransferApproved(deps NotificationDeps, data []byte) {
	var event struct {
		TransferID string  `json:"transfer_id"`
		SupplierID string  `json:"supplier_id"`
		VolumeVU   float64 `json:"volume_vu"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "TRANSFER_APPROVED", "err", err)
		return
	}
	if event.SupplierID == "" || event.TransferID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventTransferApproved,
		notifications.FormattedNotification{
			Title: "Transfer Approved",
			Body:  fmt.Sprintf("Transfer %s approved for %.1f VU.", event.TransferID, event.VolumeVU),
		})
}

func handleTransferReceived(deps NotificationDeps, data []byte) {
	var event struct {
		TransferID string `json:"transfer_id"`
		SupplierID string `json:"supplier_id"`
		ItemsCount int    `json:"items_count"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "TRANSFER_RECEIVED", "err", err)
		return
	}
	if event.SupplierID == "" || event.TransferID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventTransferReceived,
		notifications.FormattedNotification{
			Title: "Transfer Received",
			Body:  fmt.Sprintf("Transfer %s received with %d items reconciled.", event.TransferID, event.ItemsCount),
		})
}

func handleTransferUnassigned(deps NotificationDeps, data []byte) {
	var event TransferUnassignedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "TRANSFER_UNASSIGNED", "err", err)
		return
	}
	if event.SupplierID == "" || event.TransferID == "" {
		return
	}
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventTransferUnassigned,
		notifications.FormattedNotification{
			Title: "Transfer Unassigned",
			Body:  fmt.Sprintf("Transfer %s was unassigned from manifest %s.", event.TransferID, event.ManifestID),
		})
}

// ─── Dispatch Protocol ─────────────────────────────────────────────────────────
// 1. Write to Spanner Notifications table (persistent inbox)
// 2. Push via WebSocket (real-time toast for connected clients)
// 3. FCM fallback (if WebSocket miss and FCM token available - retailers only)
// 4. Telegram message (if TelegramChatId configured)

func dispatchToRecipient(deps NotificationDeps, recipientID, role, eventType string, notif notifications.FormattedNotification) {
	if recipientID == "" {
		return
	}

	ctx := context.Background()
	payloadJSON, _ := json.Marshal(map[string]string{"event_type": eventType})

	// 1. Persistent inbox
	if err := notifications.InsertNotification(ctx, deps.SpannerClient,
		recipientID, role, eventType, notif.Title, notif.Body, string(payloadJSON), "PUSH",
	); err != nil {
		slog.Error("notification_dispatcher.inbox_insert", "role", role, "recipient_id", recipientID, "err", err)
	}

	// 2. WebSocket push
	wsDelivered := false
	wsPayload := map[string]interface{}{
		"type":    eventType,
		"title":   notif.Title,
		"body":    notif.Body,
		"channel": "PUSH",
	}

	switch role {
	case "RETAILER":
		if deps.RetailerHub != nil {
			wsDelivered = deps.RetailerHub.PushToRetailer(recipientID, wsPayload)
		}
	case "DRIVER":
		if deps.DriverHub != nil {
			wsDelivered = deps.DriverHub.PushToDriver(recipientID, wsPayload)
		}
	case "SUPPLIER":
		if deps.PayloaderHub != nil {
			wsDelivered = deps.PayloaderHub.PushToPayloader(recipientID, wsPayload)
		}
	case "PAYLOADER":
		if deps.PayloaderHub != nil {
			wsDelivered = deps.PayloaderHub.PushToPayloader(recipientID, wsPayload)
		}
	}

	// 3. FCM fallback (retailer only, when WS missed)
	if !wsDelivered && role == "RETAILER" && deps.FCM != nil {
		token, _ := lookupRetailerDeviceToken(context.Background(), deps.SpannerClient, recipientID)
		if token != "" {
			if err := deps.FCM.SendDataMessage(token, map[string]string{
				"type":  eventType,
				"title": notif.Title,
				"body":  notif.Body,
			}); err != nil {
				slog.Error("notification_dispatcher.fcm_failed", "recipient_id", recipientID, "err", err)
			}
		}
	}

	// 4. Telegram (if configured)
	if deps.Telegram != nil {
		chatID := notifications.LookupTelegramChatID(deps.SpannerClient, recipientID, role)
		if chatID != "" {
			text := notifications.FormatTelegram(notif)
			if err := deps.Telegram.SendMessage(chatID, text); err != nil {
				slog.Error("notification_dispatcher.telegram_failed", "role", role, "recipient_id", recipientID, "err", err)
			}
		}
	}

	slog.Info("notification_dispatcher.delivered", "event", eventType, "role", role, "recipient_id", recipientID[:8], "ws", wsDelivered)
}

// ─── Preorder Lifecycle Handlers ───────────────────────────────────────────────

func handlePreOrderAutoAccepted(deps NotificationDeps, data []byte) {
	var event PreOrderAutoAcceptedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PRE_ORDER_AUTO_ACCEPTED", "err", err)
		return
	}
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventPreOrderAutoAccepted,
		notifications.FormattedNotification{
			Title: "Preorder Auto-Accepted",
			Body:  fmt.Sprintf("Your scheduled order %s has been accepted and is now being prepared for delivery on %s.", event.OrderID[:8], event.DeliveryDate),
		})
}

func handlePreOrderConfirmed(deps NotificationDeps, data []byte) {
	var event PreOrderConfirmedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PRE_ORDER_CONFIRMED", "err", err)
		return
	}
	dispatchToRecipient(deps, event.ConfirmedBy, "RETAILER", EventPreOrderConfirmed,
		notifications.FormattedNotification{
			Title: "Preorder Confirmed",
			Body:  fmt.Sprintf("You confirmed your scheduled order %s. It will be auto-accepted when the delivery date approaches.", event.OrderID[:8]),
		})
}

func handlePreOrderEdited(deps NotificationDeps, data []byte) {
	var event PreOrderEditedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PRE_ORDER_EDITED", "err", err)
		return
	}
	body := fmt.Sprintf("Your scheduled order %s has been updated.", event.OrderID[:8])
	if event.NewDate != "" {
		body = fmt.Sprintf("Your scheduled order %s has been rescheduled to %s.", event.OrderID[:8], event.NewDate)
	}
	dispatchToRecipient(deps, event.EditedBy, "RETAILER", EventPreOrderEdited,
		notifications.FormattedNotification{
			Title: "Preorder Updated",
			Body:  body,
		})
}

// ─── Fleet / Dispatch-Lock / Resource Lifecycle Handlers ───────────────────────

func handleFleetDispatched(deps NotificationDeps, data []byte) {
	var event FleetDispatchedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FLEET_DISPATCHED", "err", err)
		return
	}
	notif := notifications.FormatFleetDispatched(event.RouteID, len(event.OrderIDs))
	if event.DriverID != "" {
		dispatchToRecipient(deps, event.DriverID, "DRIVER", EventFleetDispatched,
			notifications.FormatDriverDispatched(event.RouteID, len(event.OrderIDs)))
	}
	if event.SupplierID != "" {
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventFleetDispatched, notif)
	}
}

func handleDispatchLockAcquired(deps NotificationDeps, data []byte) {
	var event DispatchLockEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "DISPATCH_LOCK_ACQUIRED", "err", err)
		return
	}
	if event.SupplierID != "" {
		notif := notifications.FormatDispatchLockAcquired(event.LockType, event.LockedBy)
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventDispatchLockAcquired, notif)
	}
}

func handleDispatchLockReleased(deps NotificationDeps, data []byte) {
	var event DispatchLockEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "DISPATCH_LOCK_RELEASED", "err", err)
		return
	}
	if event.SupplierID != "" {
		notif := notifications.FormatDispatchLockReleased(event.LockType, event.LockedBy)
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventDispatchLockReleased, notif)
	}
}

func handleFreezeLockAcquired(deps NotificationDeps, data []byte) {
	var event DispatchLockEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FREEZE_LOCK_ACQUIRED", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	scope := event.WarehouseID
	if scope == "" {
		scope = event.FactoryID
	}
	if scope == "" {
		scope = event.SupplierID
	}
	notif := notifications.FormatFreezeLockAcquired(scope)
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventFreezeLockAcquired, notif)
}

func handleFreezeLockReleased(deps NotificationDeps, data []byte) {
	var event DispatchLockEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FREEZE_LOCK_RELEASED", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	scope := event.WarehouseID
	if scope == "" {
		scope = event.FactoryID
	}
	if scope == "" {
		scope = event.SupplierID
	}
	notif := notifications.FormatFreezeLockReleased(scope)
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventFreezeLockReleased, notif)
}

func handleDriverCreated(deps NotificationDeps, data []byte) {
	var event DriverCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "DRIVER_CREATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	notif := notifications.FormatDriverCreated(event.Name, event.Phone)
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventDriverCreated, notif)
}

func handleVehicleCreated(deps NotificationDeps, data []byte) {
	var event VehicleCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "VEHICLE_CREATED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	notif := notifications.FormatVehicleCreated(event.Label, event.LicensePlate)
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventVehicleCreated, notif)
}

func handleManifestRebalanced(deps NotificationDeps, data []byte) {
	var event ManifestRebalancedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_REBALANCED", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	notif := notifications.FormatManifestRebalanced(event.TargetManifestID, len(event.TransferIDs), 0)
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventManifestRebalanced, notif)
}

func handleManifestCancelled(deps NotificationDeps, data []byte) {
	var event ManifestCancelledEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_CANCELLED", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	notif := notifications.FormatManifestCancelled(event.ManifestID, event.Reason, len(event.ReleasedIDs))
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventManifestCancelled, notif)
}

// handleManifestDispatched and handleManifestCompleted are the LEO Phase V
// terminal-side consumers. Both events are emitted via the transactional outbox
// from the driver-depart and order-completion paths respectively, and both
// notify the SUPPLIER channel only — drivers already get explicit ORDER_*
// notifications, and retailers get DRIVER_ARRIVED / ORDER_COMPLETED.
func handleManifestDispatched(deps NotificationDeps, data []byte) {
	var event ManifestLifecycleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_DISPATCHED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	notif := notifications.FormatManifestDispatched(event.ManifestID, event.StopCount)
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestDispatched, notif)
}

func handleManifestCompleted(deps NotificationDeps, data []byte) {
	var event ManifestLifecycleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_COMPLETED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	notif := notifications.FormatManifestCompleted(event.ManifestID, event.StopCount)
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestCompleted, notif)
}

func handleManifestSettled(deps NotificationDeps, data []byte) {
	var event ManifestSettlementEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_SETTLED", "err", err)
		return
	}
	if event.SupplierId == "" {
		return
	}
	notif := notifications.FormatManifestSettled(event.ManifestID, event.SupplierPayout, event.Currency)
	dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventManifestSettled, notif)
}

func handleForceSealAlert(deps NotificationDeps, data []byte) {
	var event ForceSealAlertEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "FORCE_SEAL_ALERT", "err", err)
		return
	}
	if event.SupplierID == "" {
		return
	}
	notif := notifications.FormatForceSealAlert(event.ManifestID, event.Count24h, event.Quota)
	dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventForceSealAlert, notif)
}

func handleOrderDelayed(deps NotificationDeps, data []byte) {
	var event OrderDelayedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_DELAYED", "err", err)
		return
	}
	if event.RetailerID != "" {
		notif := notifications.FormatOrderDelayed(event.OrderID, event.Reason)
		dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventOrderDelayed, notif)
	}
	if event.SupplierID != "" {
		notif := notifications.FormatOrderDelayed(event.OrderID, event.Reason)
		dispatchToRecipient(deps, event.SupplierID, "SUPPLIER", EventOrderDelayed, notif)
	}
}

func handlePayloadSync(deps NotificationDeps, data []byte) {
	var event PayloadSyncEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYLOAD_SYNC", "err", err)
		return
	}
	if event.SupplierID == "" || deps.PayloaderHub == nil {
		return
	}
	deps.PayloaderHub.PushToPayloader(event.SupplierID, data)
}

// Stakeholder is a (recipientID, role) pair for multi-party notification fan-out.
type Stakeholder struct {
	RecipientID string
	Role        string
}

// BroadcastToStakeholders dispatches the same notification to every non-empty
// stakeholder. Extracted from the 3-way pattern in handleOrderCancelledByOrigin
// to avoid repeating the nil-guard + dispatchToRecipient call per recipient.
func BroadcastToStakeholders(deps NotificationDeps, eventType string, notif notifications.FormattedNotification, parties []Stakeholder) {
	for _, p := range parties {
		if p.RecipientID != "" {
			dispatchToRecipient(deps, p.RecipientID, p.Role, eventType, notif)
		}
	}
}

// handleOrderCancelledByOrigin — Hard Kill: 3-way notification (warehouse + supplier + retailer).
func handleOrderCancelledByOrigin(deps NotificationDeps, data []byte) {
	var event OrderCancelledByOriginEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "ORDER_CANCELLED_BY_ORIGIN", "err", err)
		return
	}
	notif := notifications.FormatOrderCancelledByOrigin(event.OrderID, event.Reason)
	BroadcastToStakeholders(deps, EventOrderCancelledByOrigin, notif, []Stakeholder{
		{RecipientID: event.SupplierId, Role: "SUPPLIER"},
		{RecipientID: event.WarehouseId, Role: "WAREHOUSE_ADMIN"},
		{RecipientID: event.RetailerId, Role: "RETAILER"},
	})
}

// handlePayloadOverflow — Soft Stop: supplier-only notification.
func handlePayloadOverflow(deps NotificationDeps, data []byte) {
	var event PayloadOverflowEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "PAYLOAD_OVERFLOW", "err", err)
		return
	}
	notif := notifications.FormatPayloadOverflow(event.OrderID, event.ManifestID)
	if event.SupplierId != "" {
		dispatchToRecipient(deps, event.SupplierId, "SUPPLIER", EventPayloadOverflow, notif)
	}
}
