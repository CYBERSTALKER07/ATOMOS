package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

type partyEnvelope struct {
	Type         string `json:"type"`
	OrderID      string `json:"order_id"`
	SupplierID   string `json:"supplier_id"`
	RetailerID   string `json:"retailer_id"`
	DriverID     string `json:"driver_id"`
	WarehouseID  string `json:"warehouse_id"`
	FactoryID    string `json:"factory_id"`
	ManifestID   string `json:"manifest_id"`
	TransferID   string `json:"transfer_id"`
	RouteID      string `json:"route_id"`
	HomeNodeType string `json:"home_node_type"`
	HomeNodeID   string `json:"home_node_id"`
	Role         string `json:"role"`
	ActorID      string `json:"actor_id"`
}

func decodePartyEnvelope(payload []byte) (partyEnvelope, error) {
	var e partyEnvelope
	if err := json.Unmarshal(payload, &e); err != nil {
		return partyEnvelope{}, fmt.Errorf("decode party envelope: %w", err)
	}
	var legacy struct {
		SupplierId  string `json:"SupplierId"`
		RetailerId  string `json:"RetailerId"`
		WarehouseId string `json:"WarehouseId"`
		FactoryId   string `json:"FactoryId"`
	}
	_ = json.Unmarshal(payload, &legacy)
	if e.SupplierID == "" {
		e.SupplierID = legacy.SupplierId
	}
	if e.RetailerID == "" {
		e.RetailerID = legacy.RetailerId
	}
	if e.WarehouseID == "" {
		e.WarehouseID = legacy.WarehouseId
	}
	if e.FactoryID == "" {
		e.FactoryID = legacy.FactoryId
	}
	return e, nil
}

func (e partyEnvelope) supplierID() string {
	return strings.TrimSpace(e.SupplierID)
}

func (e partyEnvelope) retailerID() string {
	return strings.TrimSpace(e.RetailerID)
}

func (e partyEnvelope) warehouseID() string {
	return strings.TrimSpace(e.WarehouseID)
}

func (e partyEnvelope) factoryID() string {
	return strings.TrimSpace(e.FactoryID)
}

func (e partyEnvelope) dedupAggregateID() string {
	for _, id := range []string{e.OrderID, e.ManifestID, e.TransferID, e.RouteID, e.supplierID()} {
		if strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	return e.Type
}

// dispatchParityEvent routes Pegasus-reference event types not covered by the primary switch.
func (d *NotificationDispatcher) dispatchParityEvent(ctx context.Context, eventType string, payload []byte, traceID string) error {
	switch eventType {
	// Order lifecycle parity (pegasus reference producers).
	case "ORDER_COMPLETED", "ORDER_MODIFIED", "ORDER_CANCELLED", "ORDER_CANCELLED_BY_ORIGIN",
		"ORDER_DISPATCHED", "ORDER_DELAYED", "ORDER_REROUTED":
		return d.handleOrderEvent(ctx, payload, traceID)
	case "DRIVER_ARRIVED", "DRIVER_APPROACHING", "OFFLOAD_CONFIRMED":
		return d.handleOrderEvent(ctx, payload, traceID)
	case "CANCEL_REQUESTED", "CANCEL_APPROVED":
		return d.handleOrderEvent(ctx, payload, traceID)
	case "EARLY_COMPLETE_REQUESTED", "EARLY_COMPLETE_APPROVED":
		return d.handleNegotiationEvent(ctx, payload, traceID)
	case "CREDIT_DELIVERY_MARKED", "CREDIT_DELIVERY_RESOLVED":
		return d.handleDriverEdgeEvent(ctx, payload, traceID)
	case "AI_ORDER_CONFIRMED", "AI_ORDER_REJECTED":
		return d.handleAIRecommendationEvent(ctx, payload, traceID)
	case "POWER_OUTAGE_REPORTED":
		return d.handleShopClosedEvent(ctx, payload, traceID)
	case "SMS_QUICK_COMPLETE":
		return d.handleOrderEvent(ctx, payload, traceID)
	case "UNIFIED_CHECKOUT_COMPLETED":
		return d.handleOrderEvent(ctx, payload, traceID)
	case "SHOP_CLOSED_BYPASS_OFFLOAD", "SHOP_CLOSED_TIMEOUT", "PROXIMITY_UNLOCKED", "PARTIAL_OFFLOAD", "CREDIT_LEAVE":
		return d.handleShopClosedEvent(ctx, payload, traceID)
	case "ROUTE_FINALIZED":
		return d.handleRouteEvent(ctx, payload, traceID)
	case "FACTORY_MANIFEST_CREATED":
		return d.handleManifestEvent(ctx, payload, traceID)
	case "FLEET_DISPATCHED":
		return d.handleFleetDispatched(ctx, payload, traceID)

	// Finance / treasury parity.
	case "PAYMENT_SETTLED", "PAYMENT_FAILED", "PAYMENT_GATEWAY_DEGRADED", "PAYMENT_INTENT_CREATED",
		"PAYMENT_REFUNDED", "PAYMENT_BYPASS_ISSUED", "PAYMENT_BYPASS_COMPLETED", "PAYMENT_BYPASS_CONFIRMED":
		return d.handleExtendedFinanceEvent(ctx, payload, traceID)
	case "FULFILLMENT_PAYMENT_COMPLETED", "FULFILLMENT_PAID", "CASH_COLLECTION_REQUIRED":
		return d.handleExtendedFinanceEvent(ctx, payload, traceID)
	case "STOCK_BACKORDERED":
		return d.handleExtendedFinanceEvent(ctx, payload, traceID)
	case "SETTLEMENT_REVISED", "DELIVERY_DELTA_REFUNDED", "FEE_RATE_ADJUSTED":
		return d.handleSupplierFinanceEvent(ctx, payload, traceID)

	// Warehouse / catalog parity.
	case "WAREHOUSE_SPATIAL_UPDATED", "WAREHOUSE_STATUS_CHANGED":
		return d.handleWarehouseEntityEvent(ctx, payload, traceID)
	case "OUT_OF_STOCK":
		return d.handleCatalogEvent(ctx, payload, traceID)

	// Supplier onboarding parity aliases.
	case "SUPPLIER_REGISTERED", "SUPPLIER_CONFIGURED":
		return d.handleSupplierUpdated(ctx, payload, traceID)

	default:
		switch {
		case strings.HasPrefix(eventType, "TRANSFER_"),
			strings.HasPrefix(eventType, "WAREHOUSE_TRANSFER_"),
			strings.HasPrefix(eventType, "SUPPLY_TRANSFER_"):
			return d.handleTransferEvent(ctx, payload, traceID)
		case eventType == events.EventSplitShipmentCreated,
			eventType == events.EventOrderCapacityOverflow:
			return d.handleWarehouseOperationalEvent(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "SUPPLY_REQUEST_"),
			strings.HasPrefix(eventType, "FACTORY_SUPPLY_"):
			return d.handleSupplyRequestEvent(ctx, payload, traceID)
		case eventType == events.EventWarehouseLocationUpdated:
			return d.handleWarehouseLocationUpdated(ctx, payload, traceID)
		case eventType == events.EventFactoryLocationUpdated:
			return d.handleFactoryLocationUpdated(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "INVENTORY_IMPORT_"):
			return d.handleImportEvent(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "OPTIMIZATION_"), eventType == "DEMAND_FORECAST_READY":
			return d.handleOptimizationEvent(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "DISPATCH_LOCK_"), strings.HasPrefix(eventType, "FREEZE_LOCK_"):
			return d.handleLockEvent(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "REPLENISHMENT_"),
			eventType == "STOCK_THRESHOLD_BREACH",
			eventType == "LOOK_AHEAD_COMPLETED",
			eventType == "PULL_MATRIX_COMPLETED",
			eventType == "FACTORY_SLA_BREACH",
			eventType == "INBOUND_FREIGHT_UNANNOUNCED",
			eventType == "SUPPLY_LANE_TRANSIT_UPDATED",
			eventType == "NETWORK_MODE_CHANGED",
			eventType == "INSIGHT_APPROVED_TRANSFER_CREATED":
			return d.handleReplenishmentEvent(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "PRE_ORDER_"):
			return d.handlePreOrderEvent(ctx, payload, traceID)
		case strings.HasPrefix(eventType, "PAYLOAD_"):
			return d.handlePayloadOpsEvent(ctx, payload, traceID)
		case eventType == "MANIFEST_SETTLED", eventType == "FORCE_SEAL_ALERT", eventType == "MANIFEST_FORCE_SEALED":
			return d.handleManifestEvent(ctx, payload, traceID)
		default:
			return nil
		}
	}
}

func (d *NotificationDispatcher) handleTelemetryLocation(ctx context.Context, payload []byte, traceID string) error {
	var nested struct {
		Type string `json:"type"`
		Data struct {
			DriverID   string `json:"driver_id"`
			SupplierID string `json:"supplier_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &nested); err != nil {
		return fmt.Errorf("decode telemetry location: %w", err)
	}
	driverID := nested.Data.DriverID
	supplierID := nested.Data.SupplierID
	if driverID == "" && supplierID == "" {
		var flat partyEnvelope
		if err := json.Unmarshal(payload, &flat); err != nil {
			return fmt.Errorf("decode telemetry location flat: %w", err)
		}
		driverID = flat.DriverID
		supplierID = flat.supplierID()
	}
	eventType := events.EventDriverLocationUpdated
	if nested.Type != "" {
		eventType = nested.Type
	}
	if d.dropFanout(eventType, traceID, driverID) {
		return nil
	}
	d.broadcastSupplier(ctx, supplierID, payload)
	d.broadcastDriver(ctx, driverID, payload)
	return nil
}

func (d *NotificationDispatcher) handlePromotionChanged(ctx context.Context, payload []byte, traceID string) error {
	var e events.PromotionEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return fmt.Errorf("decode promotion changed event: %w", err)
	}
	if d.dropFanout(events.EventPromotionChanged, traceID, e.PromotionID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.SupplierID, payload)
	if strings.EqualFold(e.RetailerScope, "ALLOWLIST") {
		for _, retailerID := range e.RetailerIDs {
			d.broadcastRetailer(ctx, strings.TrimSpace(retailerID), payload)
		}
		return nil
	}
	// ALL-scope live delivery is room-based (supplier-promo:{id}) so Kafka ACK stays O(1).
	// Retailers subscribe via cart WS attach or POST /v1/retailer/promotions/watch.
	d.broadcastRetailerPromoSupplier(ctx, e.SupplierID, payload)
	return nil
}

func (d *NotificationDispatcher) handleSyncEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastRetailer(ctx, e.retailerID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleCommandEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	d.broadcastFactory(ctx, e.factoryID(), payload)
	return nil
}

func (d *NotificationDispatcher) handlePlatformOutdated(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	aggregateID := strings.TrimSpace(e.ActorID)
	if aggregateID == "" {
		aggregateID = e.supplierID()
	}
	if d.dropFanout(events.EventSystemAppOutdated, traceID, aggregateID) {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(e.Role)) {
	case "DRIVER":
		d.broadcastDriver(ctx, e.ActorID, payload)
	case "RETAILER":
		d.broadcastRetailer(ctx, e.ActorID, payload)
	case "WAREHOUSE_ADMIN", "WAREHOUSE":
		d.broadcastWarehouse(ctx, e.ActorID, payload)
	case "FACTORY_ADMIN", "FACTORY":
		d.broadcastFactory(ctx, e.ActorID, payload)
	case "PAYLOAD":
		d.broadcastPayload(ctx, e.supplierID(), payload)
	default:
		d.broadcastSupplier(ctx, e.supplierID(), payload)
	}
	return nil
}

func (d *NotificationDispatcher) handleTransferEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	// Typed WarehouseEvent / nested transfer payloads.
	if e.warehouseID() == "" && e.supplierID() == "" {
		var wh events.WarehouseEvent
		if json.Unmarshal(payload, &wh) == nil {
			if e.Type == "" {
				e.Type = wh.Type
			}
			if e.WarehouseID == "" {
				e.WarehouseID = wh.WarehouseID
			}
			if e.SupplierID == "" {
				e.SupplierID = wh.SupplierID
			}
			if e.FactoryID == "" {
				e.FactoryID = wh.FactoryID
			}
			if e.TransferID == "" {
				e.TransferID = wh.TransferID
				if e.TransferID == "" {
					e.TransferID = wh.LinkedTransferID
				}
			}
		}
	}
	aggregateID := strings.TrimSpace(e.TransferID)
	if aggregateID == "" {
		aggregateID = e.ManifestID
	}
	if aggregateID == "" {
		aggregateID = e.warehouseID()
	}
	if aggregateID == "" {
		aggregateID = e.supplierID()
	}
	if d.dropFanout(e.Type, traceID, aggregateID) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	factoryRoom := e.factoryID()
	if factoryRoom == "" {
		factoryRoom = e.supplierID()
	}
	d.broadcastFactory(ctx, factoryRoom, payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handleSupplyRequestEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	// Nested local factory broadcasts: { "type": "...", "data": { warehouse_id, ... } }
	if e.Type == "" || (e.warehouseID() == "" && e.factoryID() == "" && e.supplierID() == "") {
		var nested struct {
			Type string `json:"type"`
			Data struct {
				RequestID   string `json:"request_id"`
				WarehouseID string `json:"warehouse_id"`
				FactoryID   string `json:"factory_id"`
				SupplierID  string `json:"supplier_id"`
				State       string `json:"state"`
			} `json:"data"`
		}
		if json.Unmarshal(payload, &nested) == nil {
			if e.Type == "" {
				e.Type = nested.Type
			}
			if e.WarehouseID == "" {
				e.WarehouseID = nested.Data.WarehouseID
			}
			if e.FactoryID == "" {
				e.FactoryID = nested.Data.FactoryID
			}
			if e.SupplierID == "" {
				e.SupplierID = nested.Data.SupplierID
			}
			if e.OrderID == "" {
				e.OrderID = nested.Data.RequestID
			}
		}
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	factoryRoom := e.factoryID()
	if factoryRoom == "" {
		factoryRoom = e.supplierID()
	}
	d.broadcastFactory(ctx, factoryRoom, payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleLockEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.warehouseID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleReplenishmentEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastFactory(ctx, e.factoryID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleImportEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.supplierID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleOptimizationEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	return nil
}

func (d *NotificationDispatcher) handlePlanningEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	if e.Type == events.EventDemandBaselineUpdated {
		invalidateForecastAggCache(ctx, d.deps.Cache, e.supplierID())
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handlePreOrderEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastRetailer(ctx, e.retailerID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	if e.DriverID != "" {
		d.broadcastDriver(ctx, e.DriverID, payload)
	}
	return nil
}

func (d *NotificationDispatcher) handlePayloadOpsEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastPayload(ctx, e.supplierID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handleExtendedFinanceEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastRetailer(ctx, e.retailerID(), payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	return nil
}

func (d *NotificationDispatcher) handleWarehouseEntityEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.warehouseID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleCatalogEvent(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastRetailer(ctx, e.retailerID(), payload)
	return nil
}

func (d *NotificationDispatcher) handleFleetDispatched(ctx context.Context, payload []byte, traceID string) error {
	e, err := decodePartyEnvelope(payload)
	if err != nil {
		return err
	}
	if d.dropFanout(e.Type, traceID, e.dedupAggregateID()) {
		return nil
	}
	d.broadcastSupplier(ctx, e.supplierID(), payload)
	d.broadcastDriver(ctx, e.DriverID, payload)
	d.broadcastWarehouse(ctx, e.warehouseID(), payload)
	return nil
}
