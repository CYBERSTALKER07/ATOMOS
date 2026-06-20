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
	case events.EventManifestDraftCreated:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDraftCreated(e.ManifestID)
		}
	case events.EventManifestLoadingStarted:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestLoadingStarted(e.ManifestID, e.DriverID)
		}
	case events.EventManifestSealed:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestSealed(e.ManifestID)
		}
	case events.EventManifestOrderInjected:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestOrderInjected(e.ManifestID, e.OrderID)
		}
	case events.EventManifestOrderException:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestOrderException(e.ManifestID, e.OrderID, e.Reason)
		}
	case events.EventManifestDLQEscalation:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDLQEscalation(e.ManifestID, e.Reason)
		}
	case events.EventManifestRebalanced:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestRebalanced(e.ManifestID, e.FromManifestID, e.ToManifestID)
		}
	case events.EventManifestCancelled:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestCancelled(e.ManifestID, e.Reason)
		}
	case events.EventManifestDispatched:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDispatched(e.ManifestID, e.DriverID)
		}
	case events.EventManifestCompleted:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestCompleted(e.ManifestID, e.OrderCount)
		}
	case events.EventPaymentCleared, events.EventPaymentRequired, "PAYMENT_SETTLED":
		var e events.FinanceEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatPaymentReceived(e.OrderID, 0, "")
		}
	case events.EventShopClosed:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosed(e.OrderID, e.DriverID)
		}
	case events.EventShopClosedEscalated:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedEscalated(e.OrderID)
		}
	case events.EventShopClosedResolved:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedResolved(e.OrderID, e.Resolution)
		}
	case events.EventShopClosedResponse:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedResponse(e.OrderID, e.Response)
		}
	case events.EventDriverCreated:
		var e events.DriverEvent
		if json.Unmarshal(payload, &e) == nil && e.DriverID != "" {
			return FormatDriverCreated(e.DriverID, e.HomeNodeID)
		}
	case events.EventVehicleCreated:
		var e events.VehicleEvent
		if json.Unmarshal(payload, &e) == nil && e.VehicleID != "" {
			return FormatVehicleCreated(e.VehicleID, e.HomeNodeID)
		}
	case events.EventDriverAvailabilityChanged:
		var e events.DriverEvent
		if json.Unmarshal(payload, &e) == nil && e.DriverID != "" {
			return FormatDriverAvailabilityChanged(e.DriverID, e.OnShift, e.Reason, e.Note)
		}
	case events.EventVehicleAvailabilityChanged:
		var e events.VehicleEvent
		if json.Unmarshal(payload, &e) == nil && e.VehicleID != "" {
			return FormatVehicleAvailabilityChanged(e.VehicleID, e.IsActive, e.UnavailableReason, e.UnavailableNote)
		}
	case events.EventRetailerPriceOverride:
		var e events.RetailerPriceOverrideEvent
		if json.Unmarshal(payload, &e) == nil && e.RetailerID != "" {
			return FormatRetailerPriceOverride(e.ProductID, e.PriceMinor, "UZS", strings.EqualFold(e.Action, "CREATED"))
		}
	case events.EventPreOrderDateProposed:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Review delivery date",
				Body:     fmt.Sprintf("Warehouse proposed a new delivery date for order %s. Please review.", e.OrderID),
				DeepLink: "/orders/" + e.OrderID + "?review=1",
				Priority: "high",
			}
		}
	case events.EventPreOrderDateAccepted:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Delivery date accepted",
				Body:     fmt.Sprintf("Retailer accepted the proposed delivery date for order %s.", e.OrderID),
				DeepLink: "/orders/" + e.OrderID,
				Priority: "normal",
			}
		}
	case events.EventPreOrderDateRejected:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Delivery proposal declined",
				Body:     fmt.Sprintf("Retailer declined the proposed delivery date. Order %s was cancelled.", e.OrderID),
				DeepLink: "/orders/" + e.OrderID,
				Priority: "high",
			}
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
