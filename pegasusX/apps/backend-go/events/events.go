// Package events declares canonical Kafka topic names and event-type string
// constants. Mirrors contracts/events.schema.json and packages/types EventType.
package events

import (
	"os"
	"strings"
)

const (
	// DefaultTopicMain is the fallback multi-event topic for state transitions.
	DefaultTopicMain = "pegasusx-main"
	// TopicCacheInvalidate is the Redis Pub/Sub channel for cross-pod cache
	// invalidation. Held here so cache + outbox share a single constant.
	TopicCacheInvalidate = "cache:invalidate"
)

var (
	// TopicMain resolves once at process start so isolated sandboxes can publish
	// to client-specific Kafka topics without patching every outbox call site.
	TopicMain = topicFromEnv("KAFKA_TOPIC_MAIN", DefaultTopicMain)
	// TopicFreezeLocks carries AI-worker freeze-lock signals for manual dispatch.
	TopicFreezeLocks = topicFromEnv("KAFKA_TOPIC_FREEZE_LOCKS", "pegasusx-freeze-locks")
)

// EventType constants. Add new types here, in events.schema.json, and in
// packages/types/index.ts EventType union — in the same commit.
const (
	EventSupplierCreated              = "SUPPLIER_CREATED"
	EventSupplierUpdated              = "SUPPLIER_UPDATED"
	EventSupplierProfileUpdated       = "SUPPLIER_PROFILE_UPDATED"
	EventSupplierBillingUpdated       = "SUPPLIER_BILLING_UPDATED"
	EventSupplierBillingConfigured    = "SUPPLIER_BILLING_CONFIGURED"
	EventSupplierMemberAdded          = "SUPPLIER_MEMBER_ADDED"
	EventRetailerRegistered           = "RETAILER_REGISTERED"
	EventDriverCreated                = "DRIVER_CREATED"
	EventVehicleCreated               = "VEHICLE_CREATED"
	EventWarehouseCreated             = "WAREHOUSE_CREATED"
	EventWarehouseSupplyRequestOpened = "WAREHOUSE_SUPPLY_REQUEST_OPENED"
	EventSupplyRequestAccepted        = "SUPPLY_REQUEST_ACCEPTED"
	EventSupplyRequestUpdate          = "SUPPLY_REQUEST_UPDATE"
	EventFactorySupplyRequestUpdate   = "FACTORY_SUPPLY_REQUEST_UPDATE"
	EventWarehouseDispatchLockChanged = "WAREHOUSE_DISPATCH_LOCK_CHANGED"
	EventFreezeLockAcquired           = "FREEZE_LOCK_ACQUIRED"
	EventFreezeLockReleased           = "FREEZE_LOCK_RELEASED"

	EventWarehouseTransferCreated  = "WAREHOUSE_TRANSFER_CREATED"
	EventWarehouseTransferReceived = "WAREHOUSE_TRANSFER_RECEIVED"

	EventFactoryCreated            = "FACTORY_CREATED"
	EventOrderCreated              = "ORDER_CREATED"
	EventOrderStatusChanged        = "ORDER_STATUS_CHANGED"
	EventOrderValidationFailed     = "ORDER_VALIDATION_FAILED"
	EventOrderAssigned             = "ORDER_ASSIGNED"
	EventOrderReassigned           = "ORDER_REASSIGNED"
	EventOrderFinalized            = "ORDER_FINALIZED"
	EventAIRecommendationCreated   = "AI_RECOMMENDATION_CREATED"
	EventAIRecommendationDecided   = "AI_RECOMMENDATION_DECIDED"
	EventRouteCreated              = "ROUTE_CREATED"
	EventRouteReordered            = "ROUTE_REORDERED"
	EventMissingItemsReported      = "MISSING_ITEMS_REPORTED"
	EventSplitPaymentCreated       = "SPLIT_PAYMENT_CREATED"
	EventManifestDraftCreated      = "MANIFEST_DRAFT_CREATED"
	EventManifestLoadingStarted    = "MANIFEST_LOADING_STARTED"
	EventManifestOrderInjected     = "MANIFEST_ORDER_INJECTED"
	EventManifestOrderException    = "MANIFEST_ORDER_EXCEPTION"
	EventManifestDLQEscalation     = "MANIFEST_DLQ_ESCALATION"
	EventManifestRebalanced        = "MANIFEST_REBALANCED"
	EventManifestCancelled         = "MANIFEST_CANCELLED"
	EventManifestSealed            = "MANIFEST_SEALED"
	EventManifestDispatched        = "MANIFEST_DISPATCHED"
	EventManifestCompleted         = "MANIFEST_COMPLETED"
	EventPaymentCleared            = "PAYMENT_CLEARED"
	EventPaymentRequired           = "PAYMENT_REQUIRED"
	EventSettlementRequired        = "SETTLEMENT_REQUIRED"
	EventDeliverySessionUpdated    = "DELIVERY_SESSION_UPDATED"
	EventDeliveryDisputed          = "DELIVERY_DISPUTED"
	EventDriverAvailabilityChanged = "DRIVER_AVAILABILITY_CHANGED"
	EventDriverLocationUpdated     = "DRIVER_LOCATION_UPDATED"
	EventShopClosed                = "SHOP_CLOSED"
	EventShopClosedResponse        = "SHOP_CLOSED_RESPONSE"
	EventShopClosedEscalated       = "SHOP_CLOSED_ESCALATED"
	EventShopClosedResolved        = "SHOP_CLOSED_RESOLVED"
	EventNegotiationProposed       = "NEGOTIATION_PROPOSED"
	EventNegotiationResolved       = "NEGOTIATION_RESOLVED"
	EventCartSyncUpdated           = "CART_SYNC_UPDATED"
	EventInventorySyncComplete     = "INVENTORY_SYNC_COMPLETE"
	EventPromotionChanged          = "PROMOTION_CHANGED"
	EventRetailerPriceOverride     = "RETAILER_PRICE_OVERRIDE"
	EventCommandDispatched         = "COMMAND_DISPATCHED"
	EventCommandReceived           = "COMMAND_RECEIVED"
	EventCommandSettled            = "COMMAND_SETTLED"
	EventSystemAppOutdated         = "SYSTEM_APP_OUTDATED"
)

// AggregateTypes used in OutboxEvents.AggregateType.
const (
	AggregateSupplier         = "Supplier"
	AggregateRetailer         = "Retailer"
	AggregateDriver           = "Driver"
	AggregateVehicle          = "Vehicle"
	AggregateWarehouse        = "Warehouse"
	AggregateFactory          = "Factory"
	AggregateOrder            = "Order"
	AggregateAIRecommendation = "AIRecommendation"
	AggregateRoute            = "Route"
	AggregateManifest         = "Manifest"
	AggregateSession          = "DeliverySession"
	AggregatePromotion            = "Promotion"
	AggregateRetailerPriceOverride = "RetailerPriceOverride"
)

func topicFromEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
