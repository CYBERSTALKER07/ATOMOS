package notifications

import (
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func TestFormatFromEvent_OrderCreated(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"type":     events.EventOrderCreated,
		"order_id": "ord-99",
	})
	got := FormatFromEvent(events.EventOrderCreated, payload)
	if got.Title != "New Order Received" {
		t.Fatalf("title=%q", got.Title)
	}
	if got.DeepLink != "/orders/ord-99" {
		t.Fatalf("deep_link=%q", got.DeepLink)
	}
}

func TestShouldPersistInboxEvent_TelemetryFalse(t *testing.T) {
	if ShouldPersistInboxEvent(events.EventDriverLocationUpdated) {
		t.Fatal("telemetry should not persist")
	}
}
