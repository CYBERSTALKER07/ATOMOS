package notifications

import "fmt"

// FormatShopClosed notifies exactly one retailer that a driver reported their shop closed.
func FormatShopClosed(orderID, attemptID string) FormattedNotification {
	return newFormattedNotification(
		"Driver Waiting - Shop Appears Closed",
		fmt.Sprintf("Driver reported shop closed for order %s. Attempt %s requires follow-up.", orderID, attemptID),
		"notification.shop_closed.title",
		"notification.shop_closed.body",
		map[string]string{"order_id": orderID, "attempt_id": attemptID},
	)
}

// FormatShopClosedResponse notifies the driver that the retailer responded to the closed alert.
func FormatShopClosedResponse(orderID, response string) FormattedNotification {
	var body string
	switch response {
	case "OPEN_NOW":
		body = fmt.Sprintf("Retailer response for order %s: OPEN_NOW. They are open, check again.", orderID)
	case "5_MIN":
		body = fmt.Sprintf("Retailer response for order %s: 5_MIN. They will be there shortly.", orderID)
	case "CALL_ME":
		body = fmt.Sprintf("Retailer response for order %s: CALL_ME. Please call them.", orderID)
	case "CLOSED_TODAY":
		body = fmt.Sprintf("Retailer response for order %s: CLOSED_TODAY. Skip stop.", orderID)
	default:
		body = fmt.Sprintf("Retailer response for order %s: %s.", orderID, response)
	}

	return FormattedNotification{
		Title:       "Retailer Responded",
		Body:        body,
		TitleKey:    "notification.shop_closed_response.title",
		BodyKey:     "notification.shop_closed_response.body",
		MessageArgs: map[string]string{"order_id": orderID, "response": response},
	}
}

// FormatShopClosedEscalated notifies the supplier admin that a shop closed case requires intervention.
func FormatShopClosedEscalated(orderID string) FormattedNotification {
	return newFormattedNotification(
		"Shop Closed Escalation",
		fmt.Sprintf("Order %s was escalated for immediate supplier action.", orderID),
		"notification.shop_closed_escalated.title",
		"notification.shop_closed_escalated.body",
		map[string]string{"order_id": orderID},
	)
}

// FormatShopClosedResolved notifies the driver and retailer of the resolution outcome.
func FormatShopClosedResolved(orderID, action string) FormattedNotification {
	var body string
	if action == "RETURN_TO_HUB" {
		body = fmt.Sprintf("Order %s shop-closed resolved by Admin: RETURN_TO_HUB.", orderID)
	} else if action == "REATTEMPT" {
		body = fmt.Sprintf("Order %s shop-closed resolved by Admin: REATTEMPT.", orderID)
	} else {
		body = fmt.Sprintf("Order %s shop-closed resolved by Admin: %s.", orderID, action)
	}

	return FormattedNotification{
		Title:       "Shop-Closed Resolved",
		Body:        body,
		TitleKey:    "notification.shop_closed_resolved.title",
		BodyKey:     "notification.shop_closed_resolved.body",
		MessageArgs: map[string]string{"order_id": orderID, "action": action},
	}
}
