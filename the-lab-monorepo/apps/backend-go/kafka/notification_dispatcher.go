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
			case EventOrderCompleted:
				handleOrderCompleted(deps, m.Value)
			case EventEarlyCompleteRequested:
				handleEarlyCompleteRequested(deps, m.Value)
			case EventNegotiationProposed:
				handleNegotiationProposed(deps, m.Value)
			case EventCreditDeliveryMarked:
				handleCreditDeliveryMarked(deps, m.Value)
			case EventMissingItemsReported:
				handleMissingItemsReported(deps, m.Value)
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
			case EventManifestOrderReassigned:
				handleManifestOrderReassigned(deps, m.Value)
			case EventPayloadSync:
				handlePayloadSync(deps, m.Value)
			case EventOrderCancelledByOrigin:
				handleOrderCancelledByOrigin(deps, m.Value)
			case EventPayloadOverflow:
				handlePayloadOverflow(deps, m.Value)
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

func handleManifestOrderReassigned(deps NotificationDeps, data []byte) {
	var event ManifestOrderReassignedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "MANIFEST_ORDER_REASSIGNED", "err", err)
		return
	}
	retailerMap := notifications.LookupRetailerIDsForOrders(deps.SpannerClient, []string{event.OrderID})
	if rid, ok := retailerMap[event.OrderID]; ok && rid != "" {
		notif := notifications.FormatManifestOrderReassigned(event.OrderID)
		dispatchToRecipient(deps, rid, "RETAILER", EventManifestOrderReassigned, notif)
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
