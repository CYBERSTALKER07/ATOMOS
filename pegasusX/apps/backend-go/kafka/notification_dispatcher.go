package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"github.com/segmentio/kafka-go"
)

// PromotionAudienceResolver resolves retailer IDs for ALL-scope promotion fanout.
type PromotionAudienceResolver interface {
	EngagedRetailerIDs(ctx context.Context, supplierID string) ([]string, error)
}

// DispatcherDeps provides the dependencies to the NotificationDispatcher.
type DispatcherDeps struct {
	SupplierHub       *ws.Hub
	WarehouseHub      *ws.Hub
	DriverHub         *ws.Hub
	RetailerHub       *ws.Hub
	FactoryHub        *ws.Hub
	PayloadHub        *ws.Hub
	Push              *notifications.PushBridge
	PromotionAudience PromotionAudienceResolver
}

// NotificationDispatcher consumes generic events from Kafka and routes
// them to downstream systems like WebSocket Hubs or Push Notifications.
type NotificationDispatcher struct {
	deps  DispatcherDeps
	dedup *fanoutDedup
}

// NewNotificationDispatcher creates a new dispatcher instance.
func NewNotificationDispatcher(deps DispatcherDeps) *NotificationDispatcher {
	return &NotificationDispatcher{
		deps:  deps,
		dedup: newFanoutDedup(defaultFanoutDedupTTL),
	}
}

// HandleEvent is the partition-parallel consumer entrypoint.
func (d *NotificationDispatcher) HandleEvent(ctx context.Context, msg kafka.Message) error {
	ctx = WithTraceFromMessage(ctx, msg)

	envelope, err := ParseEnvelope(msg.Value)
	if err != nil {
		return err
	}
	if envelope.Type == "" {
		consumerPoisonSkipped.WithLabelValues("notification_dispatcher").Inc()
		return nil
	}

	traceID := outbox.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = envelope.TraceID
	}
	slog.InfoContext(ctx, "notification dispatcher received event",
		"type", envelope.Type,
		"trace_id", traceID,
		"partition", msg.Partition,
		"offset", msg.Offset,
	)

	switch envelope.Type {
	case events.EventDriverCreated:
		return d.handleDriverCreated(ctx, msg.Value, traceID)
	case events.EventVehicleCreated:
		return d.handleVehicleCreated(ctx, msg.Value, traceID)
	case events.EventWarehouseCreated:
		return d.handleWarehouseCreated(ctx, msg.Value, traceID)
	case events.EventWarehouseSupplyRequestOpened, events.EventWarehouseDispatchLockChanged:
		return d.handleWarehouseOperationalEvent(ctx, msg.Value, traceID)
	case events.EventPaymentRequired, events.EventPaymentCleared, events.EventSettlementRequired, events.EventDeliveryDisputed:
		return d.handleSupplierFinanceEvent(ctx, msg.Value, traceID)
	case events.EventOrderCreated, events.EventOrderStatusChanged, events.EventOrderFinalized:
		return d.handleOrderEvent(ctx, msg.Value, traceID)
	case events.EventOrderAssigned, events.EventOrderReassigned:
		return d.handleOrderAssignmentEvent(ctx, msg.Value, traceID)
	case events.EventRetailerRegistered:
		return d.handleRetailerRegistered(ctx, msg.Value, traceID)
	case events.EventSupplierUpdated, events.EventSupplierBillingConfigured:
		return d.handleSupplierUpdated(ctx, msg.Value, traceID)
	case events.EventShopClosed, events.EventShopClosedResponse, events.EventShopClosedEscalated, events.EventShopClosedResolved:
		return d.handleShopClosedEvent(ctx, msg.Value, traceID)
	case events.EventNegotiationProposed, events.EventNegotiationResolved:
		return d.handleNegotiationEvent(ctx, msg.Value, traceID)
	case events.EventRouteReordered, events.EventRouteCreated:
		return d.handleRouteEvent(ctx, msg.Value, traceID)
	case events.EventMissingItemsReported, events.EventSplitPaymentCreated:
		return d.handleDriverEdgeEvent(ctx, msg.Value, traceID)
	case events.EventDriverAvailabilityChanged:
		return d.handleDriverAvailabilityChanged(ctx, msg.Value, traceID)
	case events.EventAIRecommendationCreated, events.EventAIRecommendationDecided:
		return d.handleAIRecommendationEvent(ctx, msg.Value, traceID)
	case events.EventDeliverySessionUpdated:
		return d.handleDeliverySessionEvent(ctx, msg.Value, traceID)
	case events.EventFactoryCreated:
		return d.handleFactoryCreated(ctx, msg.Value, traceID)
	case events.EventSupplierCreated:
		return d.handleSupplierCreated(ctx, msg.Value, traceID)
	case events.EventCartSyncUpdated, events.EventInventorySyncComplete:
		return d.handleSyncEvent(ctx, msg.Value, traceID)
	case events.EventPromotionChanged:
		return d.handlePromotionChanged(ctx, msg.Value, traceID)
	case events.EventCommandDispatched, events.EventCommandReceived, events.EventCommandSettled:
		return d.handleCommandEvent(ctx, msg.Value, traceID)
	case events.EventSystemAppOutdated:
		return d.handlePlatformOutdated(ctx, msg.Value, traceID)
	case events.EventOrderValidationFailed:
		return d.handleOrderEvent(ctx, msg.Value, traceID)
	case events.EventDriverLocationUpdated:
		return d.handleTelemetryLocation(ctx, msg.Value, traceID)
	default:
		if strings.HasPrefix(envelope.Type, "MANIFEST_") {
			return d.handleManifestEvent(ctx, msg.Value, traceID)
		}
		return d.dispatchParityEvent(ctx, envelope.Type, msg.Value, traceID)
	}
}

func (d *NotificationDispatcher) handleDriverCreated(ctx context.Context, payload []byte, traceID string) error {
	var e events.DriverEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode driver created event: %w", err)
	}
	if d.dropFanout(events.EventDriverCreated, traceID, e.DriverID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	if e.HomeNodeType == "WAREHOUSE" && e.HomeNodeID != "" {
		d.broadcastWarehouse(ctx, e.HomeNodeID, payload)
	}
	d.broadcastDriver(ctx, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handleVehicleCreated(ctx context.Context, payload []byte, traceID string) error {
	var e events.VehicleEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode vehicle created event: %w", err)
	}
	if d.dropFanout(events.EventVehicleCreated, traceID, e.VehicleID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	if e.HomeNodeType == "WAREHOUSE" && e.HomeNodeID != "" {
		d.broadcastWarehouse(ctx, e.HomeNodeID, payload)
	}
	return nil
}

func (d *NotificationDispatcher) handleWarehouseCreated(ctx context.Context, payload []byte, traceID string) error {
	var e events.WarehouseEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode warehouse event: %w", err)
	}
	if d.dropFanout(events.EventWarehouseCreated, traceID, e.WarehouseID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastWarehouse(ctx, e.WarehouseID, payload)
	return nil
}

func (d *NotificationDispatcher) handleOrderEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.OrderEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode order event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, payload)
	slog.DebugContext(ctx, "fanned out order event", "event_type", e.Type, "order_id", e.OrderID)
	return nil
}

func (d *NotificationDispatcher) handleOrderAssignmentEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.OrderEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode order assignment event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	driverID := strings.TrimSpace(e.DriverID)
	if driverID == "" {
		driverID = strings.TrimSpace(e.ToDriverID)
	}
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, driverID, payload)
	d.broadcastWarehouse(ctx, e.WarehouseID, payload)
	if from := strings.TrimSpace(e.FromDriverID); from != "" && from != driverID {
		d.broadcastDriver(ctx, from, payload)
	}
	slog.DebugContext(ctx, "fanned out order assignment", "event_type", e.Type, "order_id", e.OrderID, "driver_id", driverID)
	return nil
}

func (d *NotificationDispatcher) handleRetailerRegistered(ctx context.Context, payload []byte, traceID string) error {
	var e events.RetailerEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode retailer registered event: %w", err)
	}
	if d.dropFanout(events.EventRetailerRegistered, traceID, e.RetailerID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastRetailer(ctx, e.RetailerID, payload)
	return nil
}

func (d *NotificationDispatcher) handleSupplierUpdated(ctx context.Context, payload []byte, traceID string) error {
	var e events.SupplierEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode supplier updated event: %w", err)
	}
	if d.dropFanout(events.EventSupplierUpdated, traceID, e.SupplierID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	return nil
}

func (d *NotificationDispatcher) handleWarehouseOperationalEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.WarehouseEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode warehouse operational event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.WarehouseID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastWarehouse(ctx, e.WarehouseID, payload)
	slog.DebugContext(ctx, "fanned out warehouse operational event", "event_type", e.Type, "warehouse_id", e.WarehouseID)
	return nil
}

func (d *NotificationDispatcher) handleNegotiationEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.OrderEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode negotiation event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, payload)
	slog.DebugContext(ctx, "fanned out negotiation event", "event_type", e.Type, "order_id", e.OrderID)
	return nil
}

func (d *NotificationDispatcher) handleShopClosedEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.OrderEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode shop closed event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, payload)
	slog.DebugContext(ctx, "fanned out shop closed event", "event_type", e.Type, "order_id", e.OrderID)
	return nil
}

func (d *NotificationDispatcher) handleSupplierFinanceEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.FinanceEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode supplier finance event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastRetailer(ctx, e.RetailerID, payload)
	slog.DebugContext(ctx, "fanned out finance event", "event_type", e.Type, "order_id", e.OrderID)
	return nil
}

func (d *NotificationDispatcher) handleManifestEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.ManifestEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode manifest event: %w", err)
	}
	aggregateID := strings.TrimSpace(e.ManifestID)
	if aggregateID == "" {
		aggregateID = e.SupplierID
	}
	if d.dropFanout(e.Type, traceID, aggregateID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	factoryRoomID := strings.TrimSpace(e.FactoryID)
	if factoryRoomID == "" {
		factoryRoomID = e.SupplierID
	}
	d.broadcastFactory(ctx, factoryRoomID, payload)
	d.broadcastWarehouse(ctx, e.WarehouseID, payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	d.broadcastPayload(ctx, e.SupplierID, payload)
	slog.DebugContext(ctx, "fanned out manifest event", "event_type", e.Type, "factory_id", factoryRoomID)
	return nil
}

func (d *NotificationDispatcher) handleRouteEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.RouteEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode route event: %w", err)
	}
	aggregateID := strings.TrimSpace(e.RouteID)
	if aggregateID == "" {
		aggregateID = e.DriverID
	}
	if d.dropFanout(e.Type, traceID, aggregateID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	d.broadcastWarehouse(ctx, e.WarehouseID, payload)
	return nil
}

func (d *NotificationDispatcher) handleDriverEdgeEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.OrderEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode driver edge event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handleDriverAvailabilityChanged(ctx context.Context, payload []byte, traceID string) error {
	var e events.DriverEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode driver availability event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.DriverID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	if strings.EqualFold(e.HomeNodeType, "WAREHOUSE") {
		d.broadcastWarehouse(ctx, e.HomeNodeID, payload)
	}
	if strings.EqualFold(e.HomeNodeType, "FACTORY") {
		d.broadcastFactory(ctx, e.HomeNodeID, payload)
	}
	return nil
}

func (d *NotificationDispatcher) handleAIRecommendationEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.AIRecommendationEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode ai recommendation event: %w", err)
	}
	aggregateID := strings.TrimSpace(e.RecommendationID)
	if aggregateID == "" {
		aggregateID = e.SupplierID
	}
	if d.dropFanout(e.Type, traceID, aggregateID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	return nil
}

func (d *NotificationDispatcher) handleDeliverySessionEvent(ctx context.Context, payload []byte, traceID string) error {
	var e events.OrderEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode delivery session event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.OrderID) {
		return nil
	}
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handleFactoryCreated(ctx context.Context, payload []byte, traceID string) error {
	var e events.FactoryEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode factory created event: %w", err)
	}
	if d.dropFanout(events.EventFactoryCreated, traceID, e.FactoryID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	d.broadcastFactory(ctx, e.FactoryID, payload)
	return nil
}

func (d *NotificationDispatcher) handleSupplierCreated(ctx context.Context, payload []byte, traceID string) error {
	var e events.SupplierEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode supplier created event: %w", err)
	}
	if d.dropFanout(events.EventSupplierCreated, traceID, e.SupplierID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	return nil
}

func (d *NotificationDispatcher) dropFanout(eventType, traceID, aggregateID string) bool {
	key := fanoutDedupKey(eventType, traceID, aggregateID)
	if !d.dedup.shouldDrop(key) {
		return false
	}
	slog.Debug("notification fanout deduplicated",
		"event_type", eventType,
		"trace_id", traceID,
		"aggregate_id", aggregateID,
	)
	return true
}

func (d *NotificationDispatcher) fanOrderParties(ctx context.Context, supplierID, retailerID, driverID string, payload []byte) {
	d.broadcastSupplier(ctx, supplierID, payload)
	d.broadcastRetailer(ctx, retailerID, payload)
	d.broadcastDriver(ctx, driverID, payload)
}

func (d *NotificationDispatcher) broadcastSupplier(ctx context.Context, supplierID string, payload []byte) {
	if supplierID == "" || d.deps.SupplierHub == nil {
		return
	}
	d.deps.SupplierHub.Broadcast(ctx, "supplier:"+supplierID, payload)
}

func (d *NotificationDispatcher) broadcastRetailer(ctx context.Context, retailerID string, payload []byte) {
	if retailerID == "" || d.deps.RetailerHub == nil {
		return
	}
	d.deps.RetailerHub.Broadcast(ctx, "retailer:"+retailerID, payload)
	d.pushFCM(ctx, retailerID, "RETAILER", payload)
}

func (d *NotificationDispatcher) broadcastRetailerPromoSupplier(ctx context.Context, supplierID string, payload []byte) {
	if supplierID == "" || d.deps.RetailerHub == nil {
		return
	}
	d.deps.RetailerHub.Broadcast(ctx, ws.SupplierPromoRoom(supplierID), payload)
}

func (d *NotificationDispatcher) broadcastDriver(ctx context.Context, driverID string, payload []byte) {
	if driverID == "" || d.deps.DriverHub == nil {
		return
	}
	d.deps.DriverHub.Broadcast(ctx, "driver:"+driverID, payload)
	d.pushFCM(ctx, driverID, "DRIVER", payload)
}

func (d *NotificationDispatcher) pushFCM(ctx context.Context, actorID, actorRole string, payload []byte) {
	if d.deps.Push == nil {
		return
	}
	var envelope struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(payload, &envelope)
	data := map[string]string{"type": envelope.Type, "body": string(payload)}
	d.deps.Push.NotifyActor(ctx, actorID, actorRole, data)
}

func (d *NotificationDispatcher) broadcastWarehouse(ctx context.Context, warehouseID string, payload []byte) {
	if warehouseID == "" || d.deps.WarehouseHub == nil {
		return
	}
	d.deps.WarehouseHub.Broadcast(ctx, "warehouse:"+warehouseID, payload)
}

func (d *NotificationDispatcher) broadcastFactory(ctx context.Context, factoryID string, payload []byte) {
	if factoryID == "" || d.deps.FactoryHub == nil {
		return
	}
	d.deps.FactoryHub.Broadcast(ctx, "factory:"+factoryID, payload)
}

func (d *NotificationDispatcher) broadcastPayload(ctx context.Context, supplierID string, payload []byte) {
	if supplierID == "" || d.deps.PayloadHub == nil {
		return
	}
	d.deps.PayloadHub.Broadcast(ctx, "payload:"+supplierID, payload)
}
