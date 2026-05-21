package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/segmentio/kafka-go"
)

// DispatcherDeps provides the dependencies to the NotificationDispatcher.
type DispatcherDeps struct {
	SupplierHub  *ws.Hub
	WarehouseHub *ws.Hub
	DriverHub    *ws.Hub
	RetailerHub  *ws.Hub
	// Other transports like FCM/APNs could live here
}

// NotificationDispatcher consumes generic events from Kafka and routes
// them to downstream systems like WebSocket Hubs or Push Notifications.
type NotificationDispatcher struct {
	deps DispatcherDeps
}

// NewNotificationDispatcher creates a new dispatcher instance.
func NewNotificationDispatcher(deps DispatcherDeps) *NotificationDispatcher {
	return &NotificationDispatcher{
		deps: deps,
	}
}

// HandleEvent matches the signature of EventHandler and serves as the
// entrypoint for the main consumer loop.
func (d *NotificationDispatcher) HandleEvent(ctx context.Context, msg kafka.Message) error {
	// 1. Unmarshal Base Envelope to figure out the type
	var envelope struct {
		TraceID   string `json:"trace_id"`
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		slog.Error("failed to unmarshal kafka message envelope", "err", err, "partition", msg.Partition, "offset", msg.Offset)
		// Return nil so we don't retry poison pills forever
		return nil
	}

	slog.Info("notification dispatcher received event", "type", envelope.Type, "trace_id", envelope.TraceID)

	// 2. Dispatch based on event type
	// Note: in a real application, we would check the version and idempotency here if we mutated local state,
	// but notifications just fan out.
	switch envelope.Type {
	case events.EventDriverCreated, events.EventVehicleCreated:
		return d.handleFleetCreated(ctx, msg.Value)
	case events.EventWarehouseCreated:
		return d.handleWarehouseCreated(ctx, msg.Value)
	case events.EventWarehouseSupplyRequestOpened, events.EventWarehouseDispatchLockChanged:
		return d.handleWarehouseOperationalEvent(ctx, msg.Value)
	case events.EventOrderCreated, events.EventOrderStatusChanged, events.EventOrderFinalized:
		return d.handleOrderEvent(ctx, msg.Value)
	default:
		// Unsupported events are ignored.
		return nil
	}
}

// Example Handlers for notification routing

func (d *NotificationDispatcher) handleFleetCreated(ctx context.Context, payload []byte) error {
	// Parse specifics
	var e struct {
		TraceID      string `json:"trace_id"`
		Type         string `json:"type"`
		SupplierID   string `json:"supplier_id"`
		HomeNodeID   string `json:"home_node_id"`
		HomeNodeType string `json:"home_node_type"`
	}
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode fleet event: %w", err)
	}

	// Fan out to Supplier Hub
	if d.deps.SupplierHub != nil && e.SupplierID != "" {
		room := fmt.Sprintf("supplier:%s", e.SupplierID)
		d.deps.SupplierHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out fleet created event to supplier", "supplier", e.SupplierID)
	}

	// Fan out to Warehouse Hub if it's warehouse-scoped
	if d.deps.WarehouseHub != nil && e.HomeNodeType == "WAREHOUSE" && e.HomeNodeID != "" {
		room := fmt.Sprintf("warehouse:%s", e.HomeNodeID)
		d.deps.WarehouseHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out fleet created event to warehouse", "warehouse", e.HomeNodeID)
	}

	return nil
}

func (d *NotificationDispatcher) handleWarehouseCreated(ctx context.Context, payload []byte) error {
	var e struct {
		TraceID     string `json:"trace_id"`
		Type        string `json:"type"`
		SupplierID  string `json:"supplier_id"`
		WarehouseID string `json:"warehouse_id"`
	}
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode warehouse event: %w", err)
	}

	if d.deps.SupplierHub != nil && e.SupplierID != "" {
		room := fmt.Sprintf("supplier:%s", e.SupplierID)
		d.deps.SupplierHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out warehouse created event to supplier", "supplier", e.SupplierID)
	}
	return nil
}

func (d *NotificationDispatcher) handleOrderEvent(ctx context.Context, payload []byte) error {
	var e struct {
		TraceID    string `json:"trace_id"`
		Type       string `json:"type"`
		OrderID    string `json:"order_id"`
		SupplierID string `json:"supplier_id"`
		RetailerID string `json:"retailer_id"`
	}
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode order event: %w", err)
	}

	if d.deps.SupplierHub != nil && e.SupplierID != "" {
		room := fmt.Sprintf("supplier:%s", e.SupplierID)
		d.deps.SupplierHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out order event to supplier", "event_type", e.Type, "supplier", e.SupplierID, "order_id", e.OrderID)
	}

	if d.deps.RetailerHub != nil && e.RetailerID != "" {
		room := fmt.Sprintf("retailer:%s", e.RetailerID)
		d.deps.RetailerHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out order event to retailer", "event_type", e.Type, "retailer", e.RetailerID, "order_id", e.OrderID)
	}

	return nil
}

func (d *NotificationDispatcher) handleWarehouseOperationalEvent(ctx context.Context, payload []byte) error {
	var e struct {
		TraceID     string `json:"trace_id"`
		Type        string `json:"type"`
		SupplierID  string `json:"supplier_id"`
		WarehouseID string `json:"warehouse_id"`
		LockID      string `json:"lock_id"`
		RequestID   string `json:"request_id"`
	}
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode warehouse operational event: %w", err)
	}

	if d.deps.SupplierHub != nil && e.SupplierID != "" {
		room := fmt.Sprintf("supplier:%s", e.SupplierID)
		d.deps.SupplierHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out warehouse operational event to supplier", "event_type", e.Type, "supplier", e.SupplierID, "lock_id", e.LockID, "request_id", e.RequestID)
	}

	if d.deps.WarehouseHub != nil && e.WarehouseID != "" {
		room := fmt.Sprintf("warehouse:%s", e.WarehouseID)
		d.deps.WarehouseHub.Broadcast(ctx, room, payload)
		slog.Debug("fanned out warehouse operational event to warehouse", "event_type", e.Type, "warehouse", e.WarehouseID, "lock_id", e.LockID, "request_id", e.RequestID)
	}

	return nil
}
