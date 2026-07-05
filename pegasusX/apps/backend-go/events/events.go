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
	// TopicInventoryImportEvents carries supplier bulk-import session lifecycle events.
	TopicInventoryImportEvents = topicFromEnv("KAFKA_TOPIC_INVENTORY_IMPORT", "pegasusx-inventory-import")
)

// EventType constants. Add new types here, in events.schema.json, and in
// packages/types/index.ts EventType union — in the same commit.
const (
	// @Sync(SupplierEvent)
	EventSupplierCreated           = "SUPPLIER_CREATED"
	EventSupplierUpdated           = "SUPPLIER_UPDATED"
	EventSupplierProfileUpdated    = "SUPPLIER_PROFILE_UPDATED"
	EventSupplierBillingUpdated    = "SUPPLIER_BILLING_UPDATED"
	EventSupplierBillingConfigured = "SUPPLIER_BILLING_CONFIGURED"
	EventSupplierMemberAdded       = "SUPPLIER_MEMBER_ADDED"

	// @Sync(RetailerEvent)
	EventRetailerRegistered = "RETAILER_REGISTERED"

	// @Sync(DriverEvent)
	EventDriverCreated             = "DRIVER_CREATED"
	EventDriverAvailabilityChanged = "DRIVER_AVAILABILITY_CHANGED"
	EventDriverLocationUpdated     = "DRIVER_LOCATION_UPDATED"

	// @Sync(VehicleEvent)
	EventVehicleCreated             = "VEHICLE_CREATED"
	EventVehicleAvailabilityChanged = "VEHICLE_AVAILABILITY_CHANGED"

	// @Sync(WarehouseEvent)
	EventWarehouseCreated             = "WAREHOUSE_CREATED"
	EventWarehouseLocationUpdated     = "WAREHOUSE_LOCATION_UPDATED"
	EventWarehouseDispatchLockChanged = "WAREHOUSE_DISPATCH_LOCK_CHANGED"

	// @Sync(SupplyRequestEvent)
	EventWarehouseSupplyRequestOpened = "WAREHOUSE_SUPPLY_REQUEST_OPENED"
	EventSupplyRequestAccepted        = "SUPPLY_REQUEST_ACCEPTED"
	EventSupplyRequestUpdate          = "SUPPLY_REQUEST_UPDATE"
	EventFactorySupplyRequestUpdate   = "FACTORY_SUPPLY_REQUEST_UPDATE"

	// @Sync(SystemEvent)
	EventFreezeLockAcquired = "FREEZE_LOCK_ACQUIRED"
	EventFreezeLockReleased = "FREEZE_LOCK_RELEASED"
	EventSystemAppOutdated  = "SYSTEM_APP_OUTDATED"

	// @Sync(WarehouseTransferEvent)
	EventWarehouseTransferCreated  = "WAREHOUSE_TRANSFER_CREATED"
	EventWarehouseTransferReceived = "WAREHOUSE_TRANSFER_RECEIVED"
	EventSupplyTransferApproaching = "SUPPLY_TRANSFER_APPROACHING"

	// @Sync(FactoryEvent)
	EventFactoryCreated         = "FACTORY_CREATED"
	EventFactoryLocationUpdated = "FACTORY_LOCATION_UPDATED"

	// @Sync(OrderEvent)
	EventOrderCreated          = "ORDER_CREATED"
	EventOrderStatusChanged    = "ORDER_STATUS_CHANGED"
	EventOrderValidationFailed = "ORDER_VALIDATION_FAILED"
	EventOrderAssigned         = "ORDER_ASSIGNED"
	EventOrderReassigned       = "ORDER_REASSIGNED"
	EventOrderFinalized        = "ORDER_FINALIZED"
	EventMissingItemsReported  = "MISSING_ITEMS_REPORTED"
	EventOrderAmended          = "ORDER_AMENDED"

	// @Sync(AIRecommendationEvent)
	EventAIRecommendationCreated = "AI_RECOMMENDATION_CREATED"
	EventAIRecommendationDecided = "AI_RECOMMENDATION_DECIDED"

	// @Sync(RouteEvent)
	EventRouteCreated   = "ROUTE_CREATED"
	EventRouteReordered = "ROUTE_REORDERED"

	// @Sync(FinanceEvent)
	EventSplitPaymentCreated = "SPLIT_PAYMENT_CREATED"
	EventPaymentCleared      = "PAYMENT_CLEARED"
	EventPaymentFailed       = "PAYMENT_FAILED"
	EventPaymentRequired     = "PAYMENT_REQUIRED"
	EventSettlementRequired  = "SETTLEMENT_REQUIRED"

	// @Sync(ManifestEvent)
	EventManifestDraftCreated   = "MANIFEST_DRAFT_CREATED"
	EventManifestLoadingStarted = "MANIFEST_LOADING_STARTED"
	EventManifestOrderInjected  = "MANIFEST_ORDER_INJECTED"
	EventManifestOrderException = "MANIFEST_ORDER_EXCEPTION"
	EventManifestDLQEscalation  = "MANIFEST_DLQ_ESCALATION"
	EventManifestRebalanced     = "MANIFEST_REBALANCED"
	EventManifestCancelled      = "MANIFEST_CANCELLED"
	EventManifestSealed         = "MANIFEST_SEALED"
	EventManifestDispatched     = "MANIFEST_DISPATCHED"
	EventManifestCompleted      = "MANIFEST_COMPLETED"

	// @Sync(DeliverySessionEvent)
	EventDeliverySessionUpdated = "DELIVERY_SESSION_UPDATED"
	EventDeliveryDisputed       = "DELIVERY_DISPUTED"

	// @Sync(ShopClosedEvent)
	EventShopClosed          = "SHOP_CLOSED"
	EventShopClosedResponse  = "SHOP_CLOSED_RESPONSE"
	EventShopClosedEscalated = "SHOP_CLOSED_ESCALATED"
	EventShopClosedResolved  = "SHOP_CLOSED_RESOLVED"

	// @Sync(ShopClosedBypassOffloadEvent)
	EventShopClosedBypassOffload = "SHOP_CLOSED_BYPASS_OFFLOAD"

	// @Sync(CreditDeliveryEvent)
	EventCreditDeliveryMarked   = "CREDIT_DELIVERY_MARKED"
	EventCreditDeliveryResolved = "CREDIT_DELIVERY_RESOLVED"

	// @Sync(NegotiationEvent)
	EventNegotiationProposed = "NEGOTIATION_PROPOSED"
	EventNegotiationResolved = "NEGOTIATION_RESOLVED"

	// @Sync(SyncEvent)
	EventCartSyncUpdated       = "CART_SYNC_UPDATED"
	EventInventorySyncComplete = "INVENTORY_SYNC_COMPLETE"

	// @Sync(InventoryImportEvent)
	EventInventoryImportUploaded     = "INVENTORY_IMPORT_UPLOADED"
	EventInventoryImportStatusUpdate = "INVENTORY_IMPORT_STATUS_UPDATE"

	// @Sync(PromotionEvent)
	EventPromotionChanged = "PROMOTION_CHANGED"

	// @Sync(RetailerPriceOverrideEvent)
	EventRetailerPriceOverride = "RETAILER_PRICE_OVERRIDE"

	// @Sync(CommandEvent)
	EventCommandDispatched = "COMMAND_DISPATCHED"
	EventCommandReceived   = "COMMAND_RECEIVED"
	EventCommandSettled    = "COMMAND_SETTLED"

	// @Sync(ReturnEvent)
	EventSupplierReturnCreated     = "SUPPLIER_RETURN_CREATED"
	EventSupplierReturnResolved    = "SUPPLIER_RETURN_RESOLVED"
	EventDriverReturnApproaching   = "DRIVER_RETURN_APPROACHING"
	EventReturnReceivedAtWarehouse = "RETURN_RECEIVED_AT_WAREHOUSE"

	// @Sync(ConditionEvent)
	EventOrderConditionReported = "ORDER_CONDITION_REPORTED"

	// @Sync(CreditProfileEvent)
	EventRetailerCreditProfileChanged = "RETAILER_CREDIT_PROFILE_CHANGED"
	// @Sync(CreditLimitEvent)
	EventRetailerCreditLimitBreached = "RETAILER_CREDIT_LIMIT_BREACHED"

	// @Sync(ProductEvent)
	EventProductHandlingUpdated = "PRODUCT_HANDLING_UPDATED"

	// @Sync(PreOrderEvent)
	EventPreOrderNotified     = "PRE_ORDER_NOTIFIED"
	EventPreOrderNudge        = "PRE_ORDER_NUDGE"
	EventPreOrderConfirmation = "PRE_ORDER_CONFIRMATION"
	EventPreOrderConfirmed    = "PRE_ORDER_CONFIRMED"
	EventPreOrderEdited       = "PRE_ORDER_EDITED"
	EventPreOrderCancelled    = "PRE_ORDER_CANCELLED"
	EventPreOrderAutoAccepted = "PRE_ORDER_AUTO_ACCEPTED"
	EventPreOrderDateProposed = "PRE_ORDER_DATE_PROPOSED"
	EventPreOrderDateAccepted = "PRE_ORDER_DATE_ACCEPTED"
	EventPreOrderDateRejected = "PRE_ORDER_DATE_REJECTED"

	// @Sync(PlanningEvent)
	EventReplenishmentAutoApproved    = "REPLENISHMENT_AUTO_APPROVED"
	EventReplenishmentInsightCreated  = "REPLENISHMENT_INSIGHT_CREATED"
	EventDispatchZoneOverride         = "DISPATCH_ZONE_OVERRIDE"
	EventPlanningMEIORecommendation   = "planning.meio.recommendation.v1"
	EventDemandBaselineUpdated        = "DEMAND_BASELINE_UPDATED"
	EventPlanningAgentBroadcast       = "PLANNING_AGENT_BROADCAST"
	EventPlanningForecastUpdated      = "PLANNING_FORECAST_UPDATED"
	EventPlanningPromoSimulationReady = "PLANNING_PROMO_SIMULATION_READY"
	EventPlanningConfidenceDowngraded = "PLANNING_CONFIDENCE_DOWNGRADED"
	EventPlanningSignalIngest         = "planning.signal.ingest.v1"
)

const (
	TopicPlanningSignalIngest    = "planning.signal.ingest.v1"
	TopicPlanningForecastRequest = "planning.forecast.request.v1"
	TopicPlanningForecastResult  = "planning.forecast.result.v1"
)

// AggregateTypes used in OutboxEvents.AggregateType.
const (
	AggregateSupplier              = "Supplier"
	AggregateRetailer              = "Retailer"
	AggregateDriver                = "Driver"
	AggregateVehicle               = "Vehicle"
	AggregateWarehouse             = "Warehouse"
	AggregateFactory               = "Factory"
	AggregateOrder                 = "Order"
	AggregateAIRecommendation      = "AIRecommendation"
	AggregateRoute                 = "Route"
	AggregateManifest              = "Manifest"
	AggregateSession               = "DeliverySession"
	AggregatePromotion             = "Promotion"
	AggregateRetailerPriceOverride = "RetailerPriceOverride"
	AggregatePlanning              = "Planning"
	AggregateProduct               = "Product"
	AggregateConditionReport       = "ConditionReport"
	AggregateCreditProfile         = "CreditProfile"
)

func topicFromEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
