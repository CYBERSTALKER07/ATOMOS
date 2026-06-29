package kafka

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

type inboxMemoryRepo struct {
	mu   sync.Mutex
	rows []notifications.Notification
}

func (r *inboxMemoryRepo) Create(_ context.Context, n notifications.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, n)
	return nil
}

func (r *inboxMemoryRepo) ListByRecipient(context.Context, string, int, int) ([]notifications.Notification, error) {
	return nil, nil
}

func (r *inboxMemoryRepo) MarkRead(context.Context, []string) error { return nil }

func (r *inboxMemoryRepo) MarkAllRead(context.Context, string) error { return nil }

func (r *inboxMemoryRepo) UnreadCount(context.Context, string) (int64, error) { return 0, nil }

func (r *inboxMemoryRepo) created() []notifications.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]notifications.Notification, len(r.rows))
	copy(out, r.rows)
	return out
}

func TestNotificationDispatcher_PersistsRetailerInbox(t *testing.T) {
	t.Parallel()

	repo := &inboxMemoryRepo{}
	inbox := notifications.NewService(repo, nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		RetailerHub: retailerHub,
		Inbox:       inbox,
	})

	payload, err := json.Marshal(map[string]any{
		"type":        events.EventOrderStatusChanged,
		"order_id":    "ord-1",
		"retailer_id": "ret-1",
		"status":      "IN_TRANSIT",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := dispatcher.handleOrderEvent(context.Background(), payload, "trace-1"); err != nil {
		t.Fatalf("handleOrderEvent: %v", err)
	}

	rows := repo.created()
	if len(rows) != 1 {
		t.Fatalf("created=%d want=1", len(rows))
	}
	if rows[0].RecipientID != "ret-1" || rows[0].RecipientRole != "RETAILER" {
		t.Fatalf("recipient=%+v", rows[0])
	}
	if rows[0].EventType != events.EventOrderStatusChanged {
		t.Fatalf("event_type=%q", rows[0].EventType)
	}
}

func TestNotificationDispatcher_SkipsTelemetryInbox(t *testing.T) {
	t.Parallel()

	repo := &inboxMemoryRepo{}
	inbox := notifications.NewService(repo, nil, nil)
	driverHub := ws.NewHub("driver", nil, nil)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		DriverHub: driverHub,
		Inbox:     inbox,
	})

	payload, _ := json.Marshal(map[string]any{
		"type":      events.EventDriverLocationUpdated,
		"driver_id": "drv-1",
	})
	if err := dispatcher.handleTelemetryLocation(context.Background(), payload, "trace-2"); err != nil {
		t.Fatalf("handleTelemetryLocation: %v", err)
	}
	if len(repo.created()) != 0 {
		t.Fatalf("telemetry should not persist inbox rows")
	}
}

func TestNotificationDispatcher_PersistsRetailerPriceOverrideInbox(t *testing.T) {
	t.Parallel()

	repo := &inboxMemoryRepo{}
	inbox := notifications.NewService(repo, nil, nil)
	retailerHub := ws.NewHub("retailer", nil, nil)

	dispatcher := NewNotificationDispatcher(DispatcherDeps{
		RetailerHub:     retailerHub,
		Inbox:           inbox,
		ConsumerGroupID: "void-notification-dispatcher",
	})

	payload, err := json.Marshal(map[string]any{
		"type":        events.EventRetailerPriceOverride,
		"override_id": "ovr-1",
		"retailer_id": "ret-1",
		"product_id":  "sku-1",
		"price_minor": int64(42000),
		"action":      "CREATED",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := dispatcher.handleRetailerPriceOverride(context.Background(), payload, "trace-3"); err != nil {
		t.Fatalf("handleRetailerPriceOverride: %v", err)
	}

	rows := repo.created()
	if len(rows) != 1 {
		t.Fatalf("created=%d want=1", len(rows))
	}
	if rows[0].RecipientID != "ret-1" || rows[0].EventType != events.EventRetailerPriceOverride {
		t.Fatalf("recipient=%+v", rows[0])
	}
}
