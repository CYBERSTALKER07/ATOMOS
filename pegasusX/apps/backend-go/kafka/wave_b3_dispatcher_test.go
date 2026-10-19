package kafka

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func TestHandleParentOrderEvent_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		RetailerHub: ws.NewHub("retailer", nil, nil),
	})
	payload, _ := json.Marshal(events.ParentOrderEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventParentOrderCreated},
		ParentOrderID: "par-1",
		RetailerID:    "ret-org-1",
		Status:        "PENDING",
		ChildCount:    2,
	})
	if err := d.handleParentOrderEvent(context.Background(), payload, "tr-b3"); err != nil {
		t.Fatalf("handleParentOrderEvent: %v", err)
	}
}

func TestHandleRetailerOpsEvent_POSSale_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		RetailerHub: ws.NewHub("retailer", nil, nil),
	})
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventPosSaleCompleted,
		"retailer_id": "ret-org-1",
		"sale_id":     "sale-1",
	})
	if err := d.handleRetailerOpsEvent(context.Background(), payload, "tr-b3-pos"); err != nil {
		t.Fatalf("handleRetailerOpsEvent: %v", err)
	}
}

func TestHandleSyncEvent_Cart_NoError(t *testing.T) {
	d := NewNotificationDispatcher(DispatcherDeps{
		RetailerHub: ws.NewHub("retailer", nil, nil),
	})
	payload, _ := json.Marshal(map[string]any{
		"type":        events.EventCartSyncUpdated,
		"retailer_id": "ret-org-1",
		"item_count":  3,
	})
	if err := d.handleSyncEvent(context.Background(), payload, "tr-b3-cart"); err != nil {
		t.Fatalf("handleSyncEvent: %v", err)
	}
}
