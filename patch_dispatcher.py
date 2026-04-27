import sys
import re

with open("the-lab-monorepo/apps/backend-go/kafka/notification_dispatcher.go", "r") as f:
    text = f.read()

# Replace handleShopClosed
old_sc = """func handleShopClosed(deps NotificationDeps, data []byte) {
	var event ShopClosedEvent
	if err := json.Unmarshal(data, &event); err \!= nil {
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
}"""

new_sc = """func handleShopClosed(deps NotificationDeps, data []byte) {
	var event ShopClosedEvent
	if err := json.Unmarshal(data, &event); err \!= nil {
		slog.Error("notification_dispatcher.unmarshal", "event", "SHOP_CLOSED", "err", err)
		return
	}
	if event.SupplierID == "" || event.OrderID == "" {
		return
	}

	notif := notifications.FormatShopClosed(event.OrderID, event.AttemptID)
	dispatchToRecipient(deps, event.RetailerID, "RETAILER", EventShopClosed, notif)
}"""

if old_sc in text:
    text = text.replace(old_sc, new_sc)
else:
    print("handleShopClosed not found exactly")

# Note: The problem with the previous regexes or patches is whitespaces. I will run a script to replace them exactly.
