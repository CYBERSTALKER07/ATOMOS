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

func TestNewNotificationWSFrame_ProjectsSettlementFields(t *testing.T) {
	ts := time.Date(2026, time.May, 3, 12, 30, 0, 0, time.UTC)
	frame := newNotificationWSFrame(
		"notif-settlement",
		EventSettlementRequired,
		notifications.NewFormattedNotification(
			"Settlement Required",
			"Settlement pending.",
			"notification.settlement_required.retailer.title",
			"notification.settlement_required.retailer.body",
			map[string]string{
				"order_id":        "ord-100",
				"session_id":      "sess-100",
				"invoice_id":      "inv-100",
				"state":           "SETTLEMENT_AWAIT",
				"currency":        "UZS",
				"amount":          "47000",
				"original_amount": "52000",
			},
		),
		`{"event_type":"SETTLEMENT_REQUIRED"}`,
		ts,
	)

	if frame.OrderID != "ord-100" {
		t.Fatalf("order_id = %q, want ord-100", frame.OrderID)
	}
	if frame.SessionID != "sess-100" {
		t.Fatalf("session_id = %q, want sess-100", frame.SessionID)
	}
	if frame.InvoiceID != "inv-100" {
		t.Fatalf("invoice_id = %q, want inv-100", frame.InvoiceID)
	}
	if frame.State != "SETTLEMENT_AWAIT" {
		t.Fatalf("state = %q, want SETTLEMENT_AWAIT", frame.State)
	}
	if frame.Currency != "UZS" {
		t.Fatalf("currency = %q, want UZS", frame.Currency)
	}
	if frame.Amount != 47000 {
		t.Fatalf("amount = %d, want 47000", frame.Amount)
	}
	if frame.OriginalAmt != 52000 {
		t.Fatalf("original_amount = %d, want 52000", frame.OriginalAmt)
	}
}

func TestNewNotificationWSFrame_ProjectsDeliverySessionAmounts(t *testing.T) {
	ts := time.Date(2026, time.May, 3, 13, 0, 0, 0, time.UTC)
	frame := newNotificationWSFrame(
		"notif-session",
		EventDeliverySessionUpdated,
		notifications.NewFormattedNotification(
			"Delivery Session Updated",
			"Reconciliation applied.",
			"notification.delivery_session_updated.retailer.title",
			"notification.delivery_session_updated.retailer.body",
			map[string]string{
				"order_id":         "ord-200",
				"session_id":       "sess-200",
				"state":            "RECONCILIATION",
				"currency":         "UZS",
				"adjusted_amount":  "39000",
				"original_amount":  "41000",
				"fee_basis_points": "350",
				"fee_amount":       "1365",
			},
		),
		`{"event_type":"DELIVERY_SESSION_UPDATED"}`,
		ts,
	)

	if frame.OrderID != "ord-200" {
		t.Fatalf("order_id = %q, want ord-200", frame.OrderID)
	}
	if frame.SessionID != "sess-200" {
		t.Fatalf("session_id = %q, want sess-200", frame.SessionID)
	}
	if frame.State != "RECONCILIATION" {
		t.Fatalf("state = %q, want RECONCILIATION", frame.State)
	}
	if frame.AdjustedAmt != 39000 {
		t.Fatalf("adjusted_amount = %d, want 39000", frame.AdjustedAmt)
	}
	if frame.OriginalAmt != 41000 {
		t.Fatalf("original_amount = %d, want 41000", frame.OriginalAmt)
	}
	if frame.FeeBps != 350 {
		t.Fatalf("fee_basis_points = %d, want 350", frame.FeeBps)
	}
	if frame.FeeAmount != 1365 {
		t.Fatalf("fee_amount = %d, want 1365", frame.FeeAmount)
	}
}
