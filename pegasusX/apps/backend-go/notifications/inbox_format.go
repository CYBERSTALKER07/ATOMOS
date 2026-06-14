package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// FormatFromEvent maps a Kafka notification envelope to inbox copy.
func FormatFromEvent(eventType string, payload []byte) FormattedNotification {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return FormattedNotification{Title: "Update", Body: "You have a new notification", Priority: "normal"}
	}

	switch eventType {
	case events.EventOrderCreated:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatOrderCreated(e.OrderID, "", e.TotalMinor, e.Currency)
		}
	case events.EventOrderStatusChanged, events.EventOrderFinalized:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			status := strings.TrimSpace(e.Status)
			if status == "" {
				status = eventType
			}
			return FormatOrderStatusChanged(e.OrderID, status)
		}
	case events.EventOrderAssigned, events.EventOrderReassigned:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Route Assignment",
				Body:     fmt.Sprintf("Order %s assigned to execution", e.OrderID),
				DeepLink: "/orders/" + e.OrderID,
				Priority: "high",
			}
		}
	case events.EventManifestDispatched:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDispatched(e.ManifestID, e.DriverID)
		}
	case events.EventManifestCompleted:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestCompleted(e.ManifestID, 0)
		}
	case events.EventPaymentCleared, events.EventPaymentRequired, "PAYMENT_SETTLED":
		var e events.FinanceEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatPaymentReceived(e.OrderID, 0, "")
		}
	}

	var generic struct {
		OrderID     string `json:"order_id"`
		ManifestID  string `json:"manifest_id"`
		Status      string `json:"status"`
		WarehouseID string `json:"warehouse_id"`
	}
	_ = json.Unmarshal(payload, &generic)

	title := humanizeEventType(eventType)
	body := title
	deepLink := ""
	if generic.OrderID != "" {
		body = fmt.Sprintf("%s — order %s", title, generic.OrderID)
		deepLink = "/orders/" + generic.OrderID
	} else if generic.ManifestID != "" {
		body = fmt.Sprintf("%s — manifest %s", title, generic.ManifestID)
		deepLink = "/manifests/" + generic.ManifestID
	} else if generic.Status != "" {
		body = fmt.Sprintf("%s (%s)", title, generic.Status)
	}

	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: deepLink,
		Priority: "normal",
	}
}

// ShouldPersistInboxEvent returns false for high-frequency telemetry envelopes.
func ShouldPersistInboxEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case events.EventDriverLocationUpdated, "TRACKING_POSITION", "TELEMETRY_PING", "FLEET_LOCATION":
		return false
	default:
		return !strings.HasPrefix(eventType, "SYSTEM")
	}
}

func humanizeEventType(eventType string) string {
	parts := strings.Split(strings.TrimSpace(eventType), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
