package kafka

import (
	"backend-go/notifications"
	"encoding/json"
	"testing"
	"time"
)

func TestNewPayloadSyncFrame(t *testing.T) {
	ts := time.Date(2026, time.May, 3, 11, 0, 0, 0, time.UTC)
	frame := newPayloadSyncFrame(PayloadSyncEvent{
		SupplierID:  "supplier-1",
		WarehouseID: "warehouse-9",
		ManifestID:  "manifest-7",
		Reason:      "REBALANCED",
		Timestamp:   ts,
	})

	if frame.Type != EventPayloadSync {
		t.Fatalf("type = %q, want %q", frame.Type, EventPayloadSync)
	}
	if frame.Channel != "SYNC" {
		t.Fatalf("channel = %q, want %q", frame.Channel, "SYNC")
	}
	if frame.ManifestID != "manifest-7" {
		t.Fatalf("manifest_id = %q, want manifest-7", frame.ManifestID)
	}
	if frame.WarehouseID != "warehouse-9" {
		t.Fatalf("warehouse_id = %q, want warehouse-9", frame.WarehouseID)
	}
	if frame.Reason != "REBALANCED" {
		t.Fatalf("reason = %q, want REBALANCED", frame.Reason)
	}
	if !frame.Timestamp.Equal(ts) {
		t.Fatalf("timestamp = %s, want %s", frame.Timestamp, ts)
	}

	data, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}

	if raw["type"] != EventPayloadSync {
		t.Fatalf("json type = %#v, want %q", raw["type"], EventPayloadSync)
	}
	if raw["channel"] != "SYNC" {
		t.Fatalf("json channel = %#v, want %q", raw["channel"], "SYNC")
	}
	if raw["manifest_id"] != "manifest-7" {
		t.Fatalf("json manifest_id = %#v, want manifest-7", raw["manifest_id"])
	}
	if _, ok := raw["supplier_id"]; ok {
		t.Fatal("json unexpectedly contains supplier_id")
	}
}

func TestNewNotificationWSFrame(t *testing.T) {
	ts := time.Date(2026, time.May, 3, 12, 0, 0, 0, time.UTC)
	frame := newNotificationWSFrame(
		"notif-1",
		EventOrderStatusChanged,
		notifications.NewFormattedNotification(
			"Order Updated",
			"Status changed.",
			"notification.order_status_changed.title",
			"notification.order_status_changed.body",
			map[string]string{"order_id": "ord-1", "old_state": "PENDING", "new_state": "IN_TRANSIT"},
		),
		`{"event_type":"ORDER_STATUS_CHANGED"}`,
		ts,
	)

	if frame.ID != "notif-1" {
		t.Fatalf("id = %q, want notif-1", frame.ID)
	}
	if frame.Channel != "PUSH" {
		t.Fatalf("channel = %q, want PUSH", frame.Channel)
	}
	if frame.CreatedAt != ts.Format(time.RFC3339) {
		t.Fatalf("created_at = %q, want %q", frame.CreatedAt, ts.Format(time.RFC3339))
	}
	if frame.Payload != `{"event_type":"ORDER_STATUS_CHANGED"}` {
		t.Fatalf("payload = %q, want event payload", frame.Payload)
	}
	if frame.MessageArgs["order_id"] != "ord-1" {
		t.Fatalf("message_args order_id = %q, want ord-1", frame.MessageArgs["order_id"])
	}
	if frame.OrderID != "ord-1" {
		t.Fatalf("order_id = %q, want ord-1", frame.OrderID)
	}
	if frame.State != "IN_TRANSIT" {
		t.Fatalf("state = %q, want IN_TRANSIT", frame.State)
	}
	if frame.OldState != "PENDING" {
		t.Fatalf("old_state = %q, want PENDING", frame.OldState)
	}

	data, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}

	if raw["id"] != "notif-1" {
		t.Fatalf("json id = %#v, want notif-1", raw["id"])
	}
	if raw["channel"] != "PUSH" {
		t.Fatalf("json channel = %#v, want PUSH", raw["channel"])
	}
	if raw["payload"] != `{"event_type":"ORDER_STATUS_CHANGED"}` {
		t.Fatalf("json payload = %#v, want event payload", raw["payload"])
	}
	if raw["order_id"] != "ord-1" {
		t.Fatalf("json order_id = %#v, want ord-1", raw["order_id"])
	}
	if raw["state"] != "IN_TRANSIT" {
		t.Fatalf("json state = %#v, want IN_TRANSIT", raw["state"])
	}
}
