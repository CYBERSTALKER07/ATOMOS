package kafka

import (
	"context"
	"encoding/json"

	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

func enrichFleetAvailabilityWS(rawPayload []byte, formatted notifications.FormattedNotification) []byte {
	var envelope map[string]any
	if err := json.Unmarshal(rawPayload, &envelope); err != nil || envelope == nil {
		envelope = map[string]any{}
	}
	envelope["title"] = formatted.Title
	envelope["body"] = formatted.Body
	if formatted.DeepLink != "" {
		envelope["deep_link"] = formatted.DeepLink
	}
	if formatted.Priority != "" {
		envelope["priority"] = formatted.Priority
	}
	enriched, err := json.Marshal(envelope)
	if err != nil {
		return rawPayload
	}
	return enriched
}

func (d *NotificationDispatcher) notifyWarehouseFleetAvailability(ctx context.Context, supplierID, warehouseID, eventType string, rawPayload []byte) {
	if warehouseID == "" {
		return
	}
	formatted := notifications.FormatFromEvent(eventType, rawPayload)
	payload := enrichFleetAvailabilityWS(rawPayload, formatted)
	d.broadcastWarehouse(ctx, warehouseID, payload)
	if supplierID == "" {
		return
	}
	d.persistInbox(ctx, supplierID, "WAREHOUSE_ADMIN", payload)
	if d.deps.Push != nil {
		d.deps.Push.NotifyActor(ctx, supplierID, "WAREHOUSE_ADMIN", map[string]string{
			"type":  eventType,
			"title": formatted.Title,
			"body":  formatted.Body,
		})
		d.deps.Push.NotifyActor(ctx, supplierID, "WAREHOUSE", map[string]string{
			"type":  eventType,
			"title": formatted.Title,
			"body":  formatted.Body,
		})
	}
}
