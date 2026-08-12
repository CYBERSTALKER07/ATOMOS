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
	// TopicDemand carries STORE_POS flywheel DEMAND_SIGNAL events for supplier subscribers.
	// Dual-write from TopicMain when KAFKA_TOPIC_DUAL_WRITE=true (see DomainTopicForEventType).
	TopicDemand = topicFromEnv("KAFKA_TOPIC_DEMAND", "pegasusx-demand")
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
	// Retail OS Phase 0
	EventRetailerStaffCreated      = "RETAILER_STAFF_CREATED"
	EventRetailerCapabilityChanged = "RETAILER_CAPABILITY_PACK_CHANGED"
	EventRetailerAutoOrderUpdated  = "RETAILER_AUTO_ORDER_UPDATED"
	// Retail OS Phase 2
	EventRetailerLocationCreated = "RETAILER_LOCATION_CREATED"
	EventRetailerLocationUpdated = "RETAILER_LOCATION_UPDATED"
	// Retail OS Phase 3
	EventStoreStockReceived    = "STORE_STOCK_RECEIVED"
	EventStoreStockAdjusted    = "STORE_STOCK_ADJUSTED"
	EventStoreStockTransferred = "STORE_STOCK_TRANSFERRED"
	EventStoreStockCounted     = "STORE_STOCK_COUNTED"
	EventStoreStockClaimHold   = "STORE_STOCK_CLAIM_HOLD"
	// Retail OS Phase 4
	EventPosSessionOpened = "POS_SESSION_OPENED"
	EventPosSessionClosed = "POS_SESSION_CLOSED"
	EventPosSaleCompleted = "POS_SALE_COMPLETED"
	EventPosSaleVoided    = "POS_SALE_VOIDED"
	// L3 sell-through flywheel
	EventRetailerSellThroughUpdated = "RETAILER_SELL_THROUGH_UPDATED"
	// B4 flywheel broadcast to suppliers (sku/qty/day only — no POS internals)
	EventDemandSignal = "DEMAND_SIGNAL"
	// Retail OS Phase 5
	EventRetailerClockIn       = "RETAILER_CLOCK_IN"
	EventRetailerClockOut      = "RETAILER_CLOCK_OUT"
	EventRetailerShiftOpened   = "RETAILER_SHIFT_OPENED"
	EventRetailerShiftClosed   = "RETAILER_SHIFT_CLOSED"
	EventRetailerShiftVariance = "RETAILER_SHIFT_CASH_VARIANCE"
	// Retail OS Phase 6
	EventRetailerSectionCreated        = "RETAILER_SECTION_CREATED"
	EventRetailerSectionUpdated        = "RETAILER_SECTION_UPDATED"
	EventRetailerSectionSkuMapped      = "RETAILER_SECTION_SKU_MAPPED"
	EventRetailerStaffSectionAssigned  = "RETAILER_STAFF_SECTION_ASSIGNED"
	EventRetailerAssistTicketOpened    = "RETAILER_ASSIST_TICKET_OPENED"
	EventRetailerAssistTicketClaimed   = "RETAILER_ASSIST_TICKET_CLAIMED"
	EventRetailerAssistTicketCompleted = "RETAILER_ASSIST_TICKET_COMPLETED"
	EventRetailerAssistTicketCancelled = "RETAILER_ASSIST_TICKET_CANCELLED"
	// Wave C4.1: SLA breach (OPEN past SlaDueAt)
	EventRetailerAssistSLABreached = "RETAILER_ASSIST_SLA_BREACHED"

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
	EventOrderCreated               = "ORDER_CREATED"
	EventOrderStatusChanged         = "ORDER_STATUS_CHANGED"
	EventOrderValidationFailed      = "ORDER_VALIDATION_FAILED"
	EventOrderAssigned              = "ORDER_ASSIGNED"
	EventOrderReassigned            = "ORDER_REASSIGNED"
	EventOrderFinalized             = "ORDER_FINALIZED"
	EventMissingItemsReported       = "MISSING_ITEMS_REPORTED"
	EventOrderAmended               = "ORDER_AMENDED"
	EventOrderAllocated             = "ORDER_ALLOCATED"
	EventAllocationPolicyApplied    = "ALLOCATION_POLICY_APPLIED"
	EventAllocationFairShareApplied = "ALLOCATION_FAIR_SHARE_APPLIED"
	EventRetailerSegmentUpdated     = "RETAILER_SEGMENT_UPDATED"
	EventSkuClassUpdated            = "SKU_CLASS_UPDATED"
	EventServicePolicyUpdated       = "SERVICE_POLICY_UPDATED"

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
	// @Sync(FinanceEvent) refund lifecycle (provider-confirmed reversal legs)
	EventRefundRequested = "REFUND_REQUESTED"
	EventRefundSucceeded = "REFUND_SUCCEEDED"
	EventRefundFailed    = "REFUND_FAILED"
	// @Sync(FiscalReceiptEvent) fiscal corrective chain (credit note EHF)
	EventFiscalCorrectiveRequested = "FISCAL_CORRECTIVE_REQUESTED"

	// @Sync(FiscalReceiptEvent) ADR-009 fiscal hard-gate
	EventFiscalReceiptRequested = "FISCAL_RECEIPT_REQUESTED"
	EventFiscalReceiptSucceeded = "FISCAL_RECEIPT_SUCCEEDED"
	EventFiscalReceiptFailed    = "FISCAL_RECEIPT_FAILED"
	// @Sync(BuyerAcceptanceEvent) Soliq EHF buyer clearance (parallel to ADR-009 COMPLETED)
	EventBuyerAcceptancePending  = "BUYER_ACCEPTANCE_PENDING"
	EventBuyerAcceptanceAccepted = "BUYER_ACCEPTANCE_ACCEPTED"
	EventBuyerAcceptanceRejected = "BUYER_ACCEPTANCE_REJECTED"
	EventBuyerAcceptanceExpired  = "BUYER_ACCEPTANCE_EXPIRED"
	// @Sync(OrderForceCompletedEvent)
	EventOrderForceCompleted = "ORDER_FORCE_COMPLETED"
	// @Sync(CashVarianceEvent) cash collect shortfall / overage (integer Tiyin)
	EventCashShortfall = "CASH_SHORTFALL"
	EventCashOverage   = "CASH_OVERAGE"

	// @Sync(LogisticsException) claims / OS&D / reverse logistics
	EventClaimFiled                 = "CLAIM_FILED"
	EventClaimResolved              = "CLAIM_RESOLVED"
	EventLogisticsExceptionReported = "LOGISTICS_EXCEPTION_REPORTED"
	EventReverseLogisticsRequired   = "REVERSE_LOGISTICS_REQUIRED"
	EventLogisticsTelemetry         = "LOGISTICS_TELEMETRY"

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

	// @Sync(SplitShipmentEvent)
	// EventSplitShipmentCreated fires when a warehouse admin approves splitting
	// a retailer's oversized order across multiple trucks. All drivers in the
	// group receive the same route; payment is coordinated — only one collection
	// event is accepted regardless of which driver reaches the retailer first.
	EventSplitShipmentCreated = "SPLIT_SHIPMENT_CREATED"

	// EventOrderCapacityOverflow is broadcast to the warehouse portal WebSocket
	// when binpack detects that a retailer's consolidated order exceeds every
	// truck's capacity. The payload contains the retailer ID, order IDs, and
	// excess volume so the admin can take corrective action.
	EventOrderCapacityOverflow = "ORDER_CAPACITY_OVERFLOW"

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

	// Enhanced shop-closed / proximity / partial offload (Phase-1 last-mile).
	EventShopClosedTimeout = "SHOP_CLOSED_TIMEOUT"
	EventProximityUnlocked = "PROXIMITY_UNLOCKED"
	EventPartialOffload    = "PARTIAL_OFFLOAD"
	EventCreditLeave       = "CREDIT_LEAVE"

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

	// @Sync(AREvent) accounts-receivable open items
	EventARInvoiceOpened  = "AR_INVOICE_OPENED"
	EventARInvoicePayment = "AR_INVOICE_PAYMENT"
	EventARInvoiceDunned  = "AR_INVOICE_DUNNED"
	EventARInvoiceSettled = "AR_INVOICE_SETTLED"
	// @Sync(PayoutEvent) supplier payout lifecycle
	EventPayoutBatchGenerated  = "PAYOUT_BATCH_GENERATED"
	EventPayoutBatchExported   = "PAYOUT_BATCH_EXPORTED"
	EventPayoutBatchDispatched = "PAYOUT_BATCH_DISPATCHED"
	EventPayoutBatchPaid       = "PAYOUT_BATCH_PAID"

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
	AggregateClaim                 = "Claim"
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
	AggregateDemandSignal          = "DemandSignal"
	AggregateARInvoice             = "ARInvoice"
	AggregatePayoutBatch           = "PayoutBatch"
)

func topicFromEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
