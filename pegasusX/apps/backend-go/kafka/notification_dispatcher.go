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

// DispatcherDeps provides the dependencies to the NotificationDispatcher.
type DispatcherDeps struct {
	SupplierHub      *ws.Hub
	WarehouseHub     *ws.Hub
	DriverHub        *ws.Hub
	RetailerHub      *ws.Hub
	FactoryHub       *ws.Hub
	PayloadHub       *ws.Hub
	Push             *notifications.PushBridge
	Inbox            *notifications.Service
	EventDedup       EventDedupStore
	ConsumerGroupID  string
}

// NotificationDispatcher consumes generic events from Kafka and routes
// them to downstream systems like WebSocket Hubs or Push Notifications.
type NotificationDispatcher struct {
	deps              DispatcherDeps
	dedup             *fanoutDedup
	eventDedup        EventDedupStore
	consumerGroupID   string
}

// NewNotificationDispatcher creates a new dispatcher instance.
func NewNotificationDispatcher(deps DispatcherDeps) *NotificationDispatcher {
	eventDedup := deps.EventDedup
	if eventDedup == nil {
		eventDedup = NewInMemoryEventDedup(defaultFanoutDedupTTL)
	}
	return &NotificationDispatcher{
		deps:            deps,
		dedup:           newFanoutDedup(defaultFanoutDedupTTL),
		eventDedup:      eventDedup,
		consumerGroupID: strings.TrimSpace(deps.ConsumerGroupID),
	}
}

// HandleEvent is the partition-parallel consumer entrypoint.
func (d *NotificationDispatcher) HandleEvent(ctx context.Context, msg kafka.Message) error {
	ctx = WithTraceFromMessage(ctx, msg)

	dedupKey := DedupKeyForConsumerGroup(d.consumerGroupID, msg.Topic, msg.Partition, msg.Offset)
	if d.eventDedup != nil {
		ok, err := d.eventDedup.ShouldProcess(ctx, dedupKey)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

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
	case events.EventVehicleAvailabilityChanged:
		return d.handleVehicleAvailabilityChanged(ctx, msg.Value, traceID)
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
	case events.EventSupplierUpdated, events.EventSupplierBillingConfigured,
		events.EventSupplierProfileUpdated, events.EventSupplierBillingUpdated, events.EventSupplierMemberAdded:
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
	case events.EventRetailerPriceOverride:
		return d.handleRetailerPriceOverride(ctx, msg.Value, traceID)
	case events.EventCommandDispatched, events.EventCommandReceived, events.EventCommandSettled:
		return d.handleCommandEvent(ctx, msg.Value, traceID)
	case events.EventSystemAppOutdated:
		return d.handlePlatformOutdated(ctx, msg.Value, traceID)
	case events.EventOrderValidationFailed:
		return d.handleOrderEvent(ctx, msg.Value, traceID)
	case events.EventDriverLocationUpdated:
		return d.handleTelemetryLocation(ctx, msg.Value, traceID)
	case events.EventOrderAmended:
		return d.handleOrderEvent(ctx, msg.Value, traceID)
	case events.EventSupplierReturnCreated, events.EventSupplierReturnResolved,
		events.EventDriverReturnApproaching, events.EventReturnReceivedAtWarehouse:
		return d.handleReturnGateEvent(ctx, msg.Value, traceID)
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

func (d *NotificationDispatcher) handleVehicleAvailabilityChanged(ctx context.Context, payload []byte, traceID string) error {
	var e events.VehicleEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode vehicle availability event: %w", err)
	}
	if d.dropFanout(e.Type, traceID, e.VehicleID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	if strings.EqualFold(e.HomeNodeType, "WAREHOUSE") && e.HomeNodeID != "" {
		d.notifyWarehouseFleetAvailability(ctx, e.SupplierID, e.HomeNodeID, e.Type, payload)
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
	payload = enrichOrderWSPayload(payload)
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, e.WarehouseID, payload)
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
	payload = enrichOrderWSPayload(payload)
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, driverID, e.WarehouseID, payload)
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
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, e.WarehouseID, payload)
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
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, e.WarehouseID, payload)
	slog.DebugContext(ctx, "fanned out shop closed event", "event_type", e.Type, "order_id", e.OrderID)
	return nil
}

func (d *NotificationDispatcher) handleRetailerPriceOverride(ctx context.Context, payload []byte, traceID string) error {
	var e events.RetailerPriceOverrideEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode retailer price override event: %w", err)
	}
	aggregateID := strings.TrimSpace(e.OverrideID)
	if aggregateID == "" {
		aggregateID = e.RetailerID + ":" + e.ProductID
	}
	if d.dropFanout(events.EventRetailerPriceOverride, traceID, aggregateID) {
		return nil
	}
	d.broadcastRetailer(ctx, e.RetailerID, payload)
	slog.DebugContext(ctx, "fanned out retailer price override",
		"override_id", e.OverrideID,
		"retailer_id", e.RetailerID,
		"product_id", e.ProductID,
		"action", e.Action,
	)
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
	var meta struct {
		DriverID string `json:"driver_id"`
	}
	_ = json.Unmarshal(payload, &meta)
	if strings.TrimSpace(meta.DriverID) != "" {
		d.broadcastDriver(ctx, strings.TrimSpace(meta.DriverID), payload)
	}
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
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, e.WarehouseID, payload)
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
	if strings.EqualFold(e.HomeNodeType, "WAREHOUSE") && e.HomeNodeID != "" {
		d.notifyWarehouseFleetAvailability(ctx, e.SupplierID, e.HomeNodeID, e.Type, payload)
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
	d.fanOrderParties(ctx, e.SupplierID, e.RetailerID, e.DriverID, e.WarehouseID, payload)
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

func (d *NotificationDispatcher) fanOrderParties(ctx context.Context, supplierID, retailerID, driverID, warehouseID string, payload []byte) {
	d.broadcastSupplier(ctx, supplierID, payload)
	d.broadcastRetailer(ctx, retailerID, payload)
	d.broadcastDriver(ctx, driverID, payload)
	d.broadcastWarehouse(ctx, warehouseID, payload)
}

// enrichOrderWSPayload adds legacy client aliases on flat Kafka relay payloads.
func enrichOrderWSPayload(payload []byte) []byte {
	var envelope map[string]any
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return payload
	}
	if envelope["type"] == events.EventOrderStatusChanged {
		if status, ok := envelope["status"].(string); ok && envelope["state"] == nil {
			envelope["state"] = status
		}
	}
	updated, err := json.Marshal(envelope)
	if err != nil {
		return payload
	}
	return updated
}

func (d *NotificationDispatcher) broadcastSupplier(ctx context.Context, supplierID string, payload []byte) {
	if supplierID == "" {
		return
	}
	if d.deps.SupplierHub != nil {
		d.deps.SupplierHub.Broadcast(ctx, "supplier:"+supplierID, payload)
	}
	d.persistInbox(ctx, supplierID, "ADMIN", payload)
}

func (d *NotificationDispatcher) broadcastRetailer(ctx context.Context, retailerID string, payload []byte) {
	if retailerID == "" {
		return
	}
	if d.deps.RetailerHub != nil {
		d.deps.RetailerHub.Broadcast(ctx, "retailer:"+retailerID, payload)
	}
	d.pushFCM(ctx, retailerID, "RETAILER", payload)
	d.persistInbox(ctx, retailerID, "RETAILER", payload)
}

func (d *NotificationDispatcher) broadcastRetailerPromoSupplier(ctx context.Context, supplierID string, payload []byte) {
	if supplierID == "" || d.deps.RetailerHub == nil {
		return
	}
	d.deps.RetailerHub.Broadcast(ctx, ws.SupplierPromoRoom(supplierID), payload)
}

func (d *NotificationDispatcher) broadcastDriver(ctx context.Context, driverID string, payload []byte) {
	if driverID == "" {
		return
	}
	if d.deps.DriverHub != nil {
		d.deps.DriverHub.Broadcast(ctx, "driver:"+driverID, payload)
	}
	d.pushFCM(ctx, driverID, "DRIVER", payload)
	d.persistInbox(ctx, driverID, "DRIVER", payload)
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
	if supplierID == "" {
		return
	}
	if d.deps.PayloadHub != nil {
		d.deps.PayloadHub.Broadcast(ctx, "payload:"+supplierID, payload)
	}
	d.persistInbox(ctx, supplierID, "PAYLOAD", payload)
}

func (d *NotificationDispatcher) handleReturnGateEvent(ctx context.Context, payload []byte, traceID string) error {
	env, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(env.Type, traceID, env.dedupAggregateID()) {
		return nil
	}
	if env.supplierID() != "" {
		d.broadcastSupplier(ctx, env.supplierID(), payload)
	}
	if env.warehouseID() != "" {
		d.broadcastWarehouse(ctx, env.warehouseID(), payload)
		if d.deps.PayloadHub != nil {
			d.deps.PayloadHub.Broadcast(ctx, "warehouse:"+env.warehouseID(), payload)
		}
	}
	if env.DriverID != "" && d.deps.DriverHub != nil {
		d.deps.DriverHub.Broadcast(ctx, "driver:"+strings.TrimSpace(env.DriverID), payload)
	}
	return nil
}
