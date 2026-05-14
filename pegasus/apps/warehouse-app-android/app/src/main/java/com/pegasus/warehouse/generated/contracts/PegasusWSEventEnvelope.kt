// To parse the JSON, install kotlin's serialization plugin and do:
//
// val json                   = Json { allowStructuredMapKeys = true }
// val pegasusWSEventEnvelope = json.parse(PegasusWSEventEnvelope.serializer(), jsonString)

package com.pegasus.warehouse.generated.contracts

import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlinx.serialization.descriptors.*
import kotlinx.serialization.encoding.*

@Serializable
data class PegasusWSEventEnvelope (
    val type: Type,

    @SerialName("command_id")
    val commandID: String? = null,

    @SerialName("command_state")
    val commandState: String? = null,

    @SerialName("event_type")
    val eventType: String? = null,

    @SerialName("target_id")
    val targetID: String? = null,

    @SerialName("target_role")
    val targetRole: String? = null,

    val timestamp: String? = null,

    @SerialName("trace_id")
    val traceID: String? = null,

    @SerialName("ack_by_role")
    val ackByRole: String? = null,

    @SerialName("ack_by_user_id")
    val ackByUserID: String? = null,

    @SerialName("disputed_by")
    val disputedBy: String? = null,

    @SerialName("driver_id")
    val driverID: String? = null,

    @SerialName("order_id")
    val orderID: String? = null,

    val reason: String? = null,

    @SerialName("retailer_id")
    val retailerID: String? = null,

    @SerialName("session_id")
    val sessionID: String? = null,

    @SerialName("supplier_id")
    val supplierID: String? = null,

    @SerialName("adjusted_amount")
    val adjustedAmount: Long? = null,

    val currency: String? = null,

    @SerialName("fee_amount")
    val feeAmount: Long? = null,

    @SerialName("fee_basis_points")
    val feeBasisPoints: Long? = null,

    @SerialName("original_amount")
    val originalAmount: Long? = null,

    val state: String? = null,

    @SerialName("sku_count")
    val skuCount: Long? = null,

    @SerialName("warehouse_id")
    val warehouseID: String? = null,

    val available: Boolean? = null,
    val note: String? = null,

    @SerialName("truck_id")
    val truckID: String? = null,

    @SerialName("created_by")
    val createdBy: String? = null,

    @SerialName("driver_type")
    val driverType: String? = null,

    @SerialName("home_node_id")
    val homeNodeID: String? = null,

    @SerialName("home_node_type")
    val homeNodeType: String? = null,

    val name: String? = null,
    val phone: String? = null,

    @SerialName("order_ids")
    val orderIDS: List<String>? = null,

    @SerialName("route_id")
    val routeID: String? = null,

    @SerialName("factory_id")
    val factoryID: String? = null,

    @SerialName("h3_index")
    val h3Index: String? = null,

    val lat: Double? = null,

    @SerialName("lead_time_days")
    val leadTimeDays: Long? = null,

    val lng: Double? = null,

    @SerialName("product_types")
    val productTypes: List<String>? = null,

    @SerialName("production_capacity_vu")
    val productionCapacityVu: Double? = null,

    @SerialName("region_code")
    val regionCode: String? = null,

    @SerialName("warehouses_linked")
    val warehousesLinked: Long? = null,

    @SerialName("escalation_level")
    val escalationLevel: String? = null,

    @SerialName("replacement_transfer_id")
    val replacementTransferID: String? = null,

    @SerialName("sla_breach_minutes")
    val slaBreachMinutes: Long? = null,

    @SerialName("transfer_id")
    val transferID: String? = null,

    @SerialName("global_order_count")
    val globalOrderCount: Long? = null,

    @SerialName("milestone_index")
    val milestoneIndex: Long? = null,

    @SerialName("milestone_order_count")
    val milestoneOrderCount: Long? = null,

    @SerialName("new_fee_basis_points")
    val newFeeBasisPoints: Long? = null,

    @SerialName("previous_fee_basis_points")
    val previousFeeBasisPoints: Long? = null,

    @SerialName("trigger_order_id")
    val triggerOrderID: String? = null,

    @SerialName("geo_zone")
    val geoZone: String? = null,

    @SerialName("manifest_id")
    val manifestID: String? = null,

    @SerialName("count_24h")
    val count24H: Long? = null,

    val quota: Long? = null,

    @SerialName("sealed_by")
    val sealedBy: String? = null,

    @SerialName("items_count")
    val itemsCount: Long? = null,

    @SerialName("received_by")
    val receivedBy: String? = null,

    @SerialName("confirmed_by")
    val confirmedBy: String? = null,

    @SerialName("volume_vu")
    val volumeVu: Double? = null,

    val status: String? = null,

    @SerialName("suggested_mappings")
    val suggestedMappings: Long? = null,

    @SerialName("gcs_path")
    val gcsPath: String? = null,

    @SerialName("duration_ms")
    val durationMS: Long? = null,

    @SerialName("horizon_days")
    val horizonDays: Long? = null,

    @SerialName("run_id")
    val runID: String? = null,

    val source: String? = null,

    @SerialName("cancelled_by")
    val cancelledBy: String? = null,

    @SerialName("released_ids")
    val releasedIDS: List<String>? = null,

    @SerialName("released_kind")
    val releasedKind: String? = null,

    @SerialName("attempt_count")
    val attemptCount: Long? = null,

    val escalated: Boolean? = null,

    @SerialName("exception_id")
    val exceptionID: String? = null,

    val metadata: String? = null,

    @SerialName("injected_by")
    val injectedBy: String? = null,

    @SerialName("new_total_volume_vu")
    val newTotalVolumeVu: Double? = null,

    @SerialName("new_driver_id")
    val newDriverID: String? = null,

    @SerialName("old_driver_id")
    val oldDriverID: String? = null,

    @SerialName("reassigned_by")
    val reassignedBy: String? = null,

    @SerialName("source_manifest_id")
    val sourceManifestID: String? = null,

    @SerialName("target_manifest_id")
    val targetManifestID: String? = null,

    @SerialName("rebalanced_by")
    val rebalancedBy: String? = null,

    @SerialName("transfer_ids")
    val transferIDS: List<String>? = null,

    @SerialName("proposal_id")
    val proposalID: String? = null,

    val action: String? = null,

    @SerialName("changed_by")
    val changedBy: String? = null,

    @SerialName("new_mode")
    val newMode: String? = null,

    @SerialName("old_mode")
    val oldMode: String? = null,

    val amount: Long? = null,

    @SerialName("payment_method")
    val paymentMethod: String? = null,

    @SerialName("fiscal_sign")
    val fiscalSign: String? = null,

    @SerialName("invoice_id")
    val invoiceID: String? = null,

    val items: JsonArray? = null,

    @SerialName("receipt_type")
    val receiptType: String? = null,

    @SerialName("terminal_id")
    val terminalID: String? = null,

    val tin: String? = null,
    val total: Long? = null,

    @SerialName("warehouse_name")
    val warehouseName: String? = null,

    @SerialName("amendment_id")
    val amendmentID: String? = null,

    @SerialName("new_amount")
    val newAmount: Long? = null,

    val refunded: Long? = null,

    @SerialName("new_route_id")
    val newRouteID: String? = null,

    @SerialName("old_route_id")
    val oldRouteID: String? = null,

    @SerialName("distance_km")
    val distanceKM: Double? = null,

    @SerialName("new_load_percent")
    val newLoadPercent: Double? = null,

    @SerialName("new_warehouse_id")
    val newWarehouseID: String? = null,

    @SerialName("original_load_percent")
    val originalLoadPercent: Double? = null,

    @SerialName("original_warehouse_id")
    val originalWarehouseID: String? = null,

    @SerialName("retailer_lat")
    val retailerLat: Double? = null,

    @SerialName("retailer_lng")
    val retailerLng: Double? = null,

    @SerialName("new_state")
    val newState: String? = null,

    @SerialName("old_state")
    val oldState: String? = null,

    @SerialName("shortfall_map")
    val shortfallMap: Map<String, Long>? = null,

    @SerialName("delivery_token")
    val deliveryToken: String? = null,

    val gateway: String? = null,

    @SerialName("delivery_date")
    val deliveryDate: String? = null,

    @SerialName("edited_by")
    val editedBy: String? = null,

    @SerialName("new_date")
    val newDate: String? = null,

    @SerialName("skus_processed")
    val skusProcessed: Long? = null,

    @SerialName("transfers_generated")
    val transfersGenerated: Long? = null,

    @SerialName("override_id")
    val overrideID: String? = null,

    val price: Long? = null,

    @SerialName("set_by")
    val setBy: String? = null,

    @SerialName("set_by_role")
    val setByRole: String? = null,

    @SerialName("sku_id")
    val skuID: String? = null,

    @SerialName("h3_cell")
    val h3Cell: String? = null,

    @SerialName("owner_name")
    val ownerName: String? = null,

    @SerialName("phone_number")
    val phoneNumber: String? = null,

    @SerialName("shop_name")
    val shopName: String? = null,

    @SerialName("stop_count")
    val stopCount: Long? = null,

    @SerialName("route_json")
    val routeJSON: String? = null,

    @SerialName("payment_session_id")
    val paymentSessionID: String? = null,

    @SerialName("attempt_id")
    val attemptID: String? = null,

    @SerialName("gps_lat")
    val gpsLat: Double? = null,

    @SerialName("gps_lng")
    val gpsLng: Double? = null,

    @SerialName("escalated_to")
    val escalatedTo: String? = null,

    val resolution: String? = null,

    @SerialName("resolved_by")
    val resolvedBy: String? = null,

    val response: String? = null,

    @SerialName("backorder_id")
    val backorderID: String? = null,

    @SerialName("backorder_order_id")
    val backorderOrderID: String? = null,

    @SerialName("current_stock")
    val currentStock: Long? = null,

    @SerialName("product_id")
    val productID: String? = null,

    @SerialName("safety_level")
    val safetyLevel: Long? = null,

    @SerialName("lane_id")
    val laneID: String? = null,

    @SerialName("new_dampened_hours")
    val newDampenedHours: Double? = null,

    @SerialName("old_dampened_hours")
    val oldDampenedHours: Double? = null,

    @SerialName("raw_transit_hours")
    val rawTransitHours: Double? = null,

    @SerialName("unassigned_by")
    val unassignedBy: String? = null,

    @SerialName("order_count")
    val orderCount: Long? = null,

    val label: String? = null,

    @SerialName("license_plate")
    val licensePlate: String? = null,

    @SerialName("max_volume_vu")
    val maxVolumeVu: Double? = null,

    @SerialName("vehicle_class")
    val vehicleClass: String? = null,

    @SerialName("vehicle_id")
    val vehicleID: String? = null,

    @SerialName("coverage_radius_km")
    val coverageRadiusKM: Double? = null,

    @SerialName("h3_count")
    val h3Count: Long? = null,

    @SerialName("new_h3_count")
    val newH3Count: Long? = null,

    @SerialName("old_h3_count")
    val oldH3Count: Long? = null,

    val field: String? = null,

    @SerialName("new_value")
    val newValue: Boolean? = null,

    @SerialName("old_value")
    val oldValue: Boolean? = null
)

@Serializable
enum class Type(val value: String) {
    @SerialName("AI_ORDER_CONFIRMED") AIOrderConfirmed("AI_ORDER_CONFIRMED"),
    @SerialName("AI_ORDER_REJECTED") AIOrderRejected("AI_ORDER_REJECTED"),
    @SerialName("AI_PLAN_DATE_SHIFT") AIPlanDateShift("AI_PLAN_DATE_SHIFT"),
    @SerialName("AI_PLAN_SKU_MODIFIED") AIPlanSkuModified("AI_PLAN_SKU_MODIFIED"),
    @SerialName("AI_PREDICTION") AIPrediction("AI_PREDICTION"),
    @SerialName("AI_PREDICTION_CORRECTED") AIPredictionCorrected("AI_PREDICTION_CORRECTED"),
    @SerialName("BYPASS_TOKEN_ISSUED") BypassTokenIssued("BYPASS_TOKEN_ISSUED"),
    @SerialName("CANCEL_APPROVED") CancelApproved("CANCEL_APPROVED"),
    @SerialName("CANCEL_REQUESTED") CancelRequested("CANCEL_REQUESTED"),
    @SerialName("CART_SYNC_UPDATED") CartSyncUpdated("CART_SYNC_UPDATED"),
    @SerialName("CASH_COLLECTION_REQUIRED") CashCollectionRequired("CASH_COLLECTION_REQUIRED"),
    @SerialName("COMMAND_DISPATCHED") CommandDispatched("COMMAND_DISPATCHED"),
    @SerialName("COMMAND_RECEIVED") CommandReceived("COMMAND_RECEIVED"),
    @SerialName("COMMAND_SETTLED") CommandSettled("COMMAND_SETTLED"),
    @SerialName("CREDIT_DELIVERY_MARKED") CreditDeliveryMarked("CREDIT_DELIVERY_MARKED"),
    @SerialName("CREDIT_DELIVERY_RESOLVED") CreditDeliveryResolved("CREDIT_DELIVERY_RESOLVED"),
    @SerialName("DELIVERY_DISPUTED") DeliveryDisputed("DELIVERY_DISPUTED"),
    @SerialName("DELIVERY_SESSION_UPDATED") DeliverySessionUpdated("DELIVERY_SESSION_UPDATED"),
    @SerialName("DEMAND_FORECAST_READY") DemandForecastReady("DEMAND_FORECAST_READY"),
    @SerialName("DISPATCH_LOCK_ACQUIRED") DispatchLockAcquired("DISPATCH_LOCK_ACQUIRED"),
    @SerialName("DISPATCH_LOCK_CHANGE") DispatchLockChange("DISPATCH_LOCK_CHANGE"),
    @SerialName("DISPATCH_LOCK_RELEASED") DispatchLockReleased("DISPATCH_LOCK_RELEASED"),
    @SerialName("DRIVER_APPROACHING") DriverApproaching("DRIVER_APPROACHING"),
    @SerialName("DRIVER_ARRIVED") DriverArrived("DRIVER_ARRIVED"),
    @SerialName("DRIVER_AVAILABILITY_CHANGED") DriverAvailabilityChanged("DRIVER_AVAILABILITY_CHANGED"),
    @SerialName("DRIVER_CREATED") DriverCreated("DRIVER_CREATED"),
    @SerialName("EARLY_COMPLETE_APPROVED") EarlyCompleteApproved("EARLY_COMPLETE_APPROVED"),
    @SerialName("EARLY_COMPLETE_REQUESTED") EarlyCompleteRequested("EARLY_COMPLETE_REQUESTED"),
    @SerialName("ETA_UPDATED") EtaUpdated("ETA_UPDATED"),
    @SerialName("FACTORY_CREATED") FactoryCreated("FACTORY_CREATED"),
    @SerialName("FACTORY_MANIFEST_CREATED") FactoryManifestCreated("FACTORY_MANIFEST_CREATED"),
    @SerialName("FACTORY_MANIFEST_UPDATE") FactoryManifestUpdate("FACTORY_MANIFEST_UPDATE"),
    @SerialName("FACTORY_OUTBOX_FAILED") FactoryOutboxFailed("FACTORY_OUTBOX_FAILED"),
    @SerialName("FACTORY_SLA_BREACH") FactorySlaBreach("FACTORY_SLA_BREACH"),
    @SerialName("FACTORY_SUPPLY_REQUEST_UPDATE") FactorySupplyRequestUpdate("FACTORY_SUPPLY_REQUEST_UPDATE"),
    @SerialName("FACTORY_TRANSFER_UPDATE") FactoryTransferUpdate("FACTORY_TRANSFER_UPDATE"),
    @SerialName("FEE_RATE_ADJUSTED") FeeRateAdjusted("FEE_RATE_ADJUSTED"),
    @SerialName("FLEET_DISPATCHED") FleetDispatched("FLEET_DISPATCHED"),
    @SerialName("FORCE_SEAL_ALERT") ForceSealAlert("FORCE_SEAL_ALERT"),
    @SerialName("FREEZE_LOCK_ACQUIRED") FreezeLockAcquired("FREEZE_LOCK_ACQUIRED"),
    @SerialName("FREEZE_LOCK_RELEASED") FreezeLockReleased("FREEZE_LOCK_RELEASED"),
    @SerialName("FULFILLMENT_PAID") FulfillmentPaid("FULFILLMENT_PAID"),
    @SerialName("FULFILLMENT_PAYMENT_COMPLETED") FulfillmentPaymentCompleted("FULFILLMENT_PAYMENT_COMPLETED"),
    @SerialName("INBOUND_FREIGHT_UNANNOUNCED") InboundFreightUnannounced("INBOUND_FREIGHT_UNANNOUNCED"),
    @SerialName("INSIGHT_APPROVED_TRANSFER_CREATED") InsightApprovedTransferCreated("INSIGHT_APPROVED_TRANSFER_CREATED"),
    @SerialName("INTERNAL_LOAD_CONFIRMED") InternalLoadConfirmed("INTERNAL_LOAD_CONFIRMED"),
    @SerialName("INVENTORY_IMPORT_STATUS_UPDATE") InventoryImportStatusUpdate("INVENTORY_IMPORT_STATUS_UPDATE"),
    @SerialName("INVENTORY_IMPORT_UPLOADED") InventoryImportUploaded("INVENTORY_IMPORT_UPLOADED"),
    @SerialName("INVENTORY_SYNC_COMPLETE") InventorySyncComplete("INVENTORY_SYNC_COMPLETE"),
    @SerialName("LOOK_AHEAD_COMPLETED") LookAheadCompleted("LOOK_AHEAD_COMPLETED"),
    @SerialName("MANIFEST_CANCELLED") ManifestCancelled("MANIFEST_CANCELLED"),
    @SerialName("MANIFEST_COMPLETED") ManifestCompleted("MANIFEST_COMPLETED"),
    @SerialName("MANIFEST_DISPATCHED") ManifestDispatched("MANIFEST_DISPATCHED"),
    @SerialName("MANIFEST_DLQ_ESCALATION") ManifestDlqEscalation("MANIFEST_DLQ_ESCALATION"),
    @SerialName("MANIFEST_DRAFT_CREATED") ManifestDraftCreated("MANIFEST_DRAFT_CREATED"),
    @SerialName("MANIFEST_FORCE_SEALED") ManifestForceSealed("MANIFEST_FORCE_SEALED"),
    @SerialName("MANIFEST_LOADING_STARTED") ManifestLoadingStarted("MANIFEST_LOADING_STARTED"),
    @SerialName("MANIFEST_ORDER_EXCEPTION") ManifestOrderException("MANIFEST_ORDER_EXCEPTION"),
    @SerialName("MANIFEST_ORDER_INJECTED") ManifestOrderInjected("MANIFEST_ORDER_INJECTED"),
    @SerialName("MANIFEST_ORDER_REASSIGNED") ManifestOrderReassigned("MANIFEST_ORDER_REASSIGNED"),
    @SerialName("MANIFEST_REBALANCED") ManifestRebalanced("MANIFEST_REBALANCED"),
    @SerialName("MANIFEST_SEALED") ManifestSealed("MANIFEST_SEALED"),
    @SerialName("MANIFEST_SETTLED") ManifestSettled("MANIFEST_SETTLED"),
    @SerialName("MISSING_ITEMS_REPORTED") MissingItemsReported("MISSING_ITEMS_REPORTED"),
    @SerialName("NEGOTIATION_PROPOSED") NegotiationProposed("NEGOTIATION_PROPOSED"),
    @SerialName("NEGOTIATION_RESOLVED") NegotiationResolved("NEGOTIATION_RESOLVED"),
    @SerialName("NETWORK_MODE_CHANGED") NetworkModeChanged("NETWORK_MODE_CHANGED"),
    @SerialName("OFFLOAD_CONFIRMED") OffloadConfirmed("OFFLOAD_CONFIRMED"),
    @SerialName("ORDER_AMENDED") OrderAmended("ORDER_AMENDED"),
    @SerialName("ORDER_ASSIGNED") OrderAssigned("ORDER_ASSIGNED"),
    @SerialName("ORDER_CANCEL_LOCKED") OrderCancelLocked("ORDER_CANCEL_LOCKED"),
    @SerialName("ORDER_CANCELLED") OrderCancelled("ORDER_CANCELLED"),
    @SerialName("ORDER_CANCELLED_BY_ORIGIN") OrderCancelledByOrigin("ORDER_CANCELLED_BY_ORIGIN"),
    @SerialName("ORDER_COMPLETED") OrderCompleted("ORDER_COMPLETED"),
    @SerialName("ORDER_CREATED") OrderCreated("ORDER_CREATED"),
    @SerialName("ORDER_DELAYED") OrderDelayed("ORDER_DELAYED"),
    @SerialName("ORDER_DISPATCHED") OrderDispatched("ORDER_DISPATCHED"),
    @SerialName("ORDER_FINALIZED") OrderFinalized("ORDER_FINALIZED"),
    @SerialName("ORDER_MODIFIED") OrderModified("ORDER_MODIFIED"),
    @SerialName("ORDER_REASSIGNED") OrderReassigned("ORDER_REASSIGNED"),
    @SerialName("ORDER_REJECTED_BY_SUPPLIER") OrderRejectedBySupplier("ORDER_REJECTED_BY_SUPPLIER"),
    @SerialName("ORDER_REROUTED") OrderRerouted("ORDER_REROUTED"),
    @SerialName("ORDER_STATE_CHANGED") OrderStateChanged("ORDER_STATE_CHANGED"),
    @SerialName("ORDER_STATUS_CHANGED") OrderStatusChanged("ORDER_STATUS_CHANGED"),
    @SerialName("ORDER_SYNC") OrderSync("ORDER_SYNC"),
    @SerialName("ORDER_VALIDATION_FAILED") OrderValidationFailed("ORDER_VALIDATION_FAILED"),
    @SerialName("OUT_OF_STOCK") OutOfStock("OUT_OF_STOCK"),
    @SerialName("OUTBOX_FAILED") OutboxFailed("OUTBOX_FAILED"),
    @SerialName("PAYLOAD_OVERFLOW") PayloadOverflow("PAYLOAD_OVERFLOW"),
    @SerialName("PAYLOAD_READY_TO_SEAL") PayloadReadyToSeal("PAYLOAD_READY_TO_SEAL"),
    @SerialName("PAYLOAD_SEALED") PayloadSealed("PAYLOAD_SEALED"),
    @SerialName("PAYLOAD_SYNC") PayloadSync("PAYLOAD_SYNC"),
    @SerialName("PAYMENT_BYPASS_COMPLETED") PaymentBypassCompleted("PAYMENT_BYPASS_COMPLETED"),
    @SerialName("PAYMENT_BYPASS_ISSUED") PaymentBypassIssued("PAYMENT_BYPASS_ISSUED"),
    @SerialName("PAYMENT_CLEARED") PaymentCleared("PAYMENT_CLEARED"),
    @SerialName("PAYMENT_EXPIRED") PaymentExpired("PAYMENT_EXPIRED"),
    @SerialName("PAYMENT_FAILED") PaymentFailed("PAYMENT_FAILED"),
    @SerialName("PAYMENT_INTENT_CREATED") PaymentIntentCreated("PAYMENT_INTENT_CREATED"),
    @SerialName("PAYMENT_REFUNDED") PaymentRefunded("PAYMENT_REFUNDED"),
    @SerialName("PAYMENT_REQUIRED") PaymentRequired("PAYMENT_REQUIRED"),
    @SerialName("PAYMENT_SETTLED") PaymentSettled("PAYMENT_SETTLED"),
    @SerialName("POWER_OUTAGE_REPORTED") PowerOutageReported("POWER_OUTAGE_REPORTED"),
    @SerialName("PRE_ORDER_AUTO_ACCEPTED") PreOrderAutoAccepted("PRE_ORDER_AUTO_ACCEPTED"),
    @SerialName("PRE_ORDER_CANCELLED") PreOrderCancelled("PRE_ORDER_CANCELLED"),
    @SerialName("PRE_ORDER_CONFIRMATION") PreOrderConfirmation("PRE_ORDER_CONFIRMATION"),
    @SerialName("PRE_ORDER_CONFIRMED") PreOrderConfirmed("PRE_ORDER_CONFIRMED"),
    @SerialName("PRE_ORDER_EDITED") PreOrderEdited("PRE_ORDER_EDITED"),
    @SerialName("PRE_ORDER_NOTIFIED") PreOrderNotified("PRE_ORDER_NOTIFIED"),
    @SerialName("PRE_ORDER_NUDGE") PreOrderNudge("PRE_ORDER_NUDGE"),
    @SerialName("PULL_MATRIX_COMPLETED") PullMatrixCompleted("PULL_MATRIX_COMPLETED"),
    @SerialName("REPLENISHMENT_LOCK_ACQUIRED") ReplenishmentLockAcquired("REPLENISHMENT_LOCK_ACQUIRED"),
    @SerialName("REPLENISHMENT_LOCK_RELEASED") ReplenishmentLockReleased("REPLENISHMENT_LOCK_RELEASED"),
    @SerialName("REPLENISHMENT_TRANSFER_CREATED") ReplenishmentTransferCreated("REPLENISHMENT_TRANSFER_CREATED"),
    @SerialName("RETAILER_PRICE_OVERRIDE") RetailerPriceOverride("RETAILER_PRICE_OVERRIDE"),
    @SerialName("RETAILER_REGISTERED") RetailerRegistered("RETAILER_REGISTERED"),
    @SerialName("RETURN_RESOLVED") ReturnResolved("RETURN_RESOLVED"),
    @SerialName("ROUTE_CREATED") RouteCreated("ROUTE_CREATED"),
    @SerialName("ROUTE_FINALIZED") RouteFinalized("ROUTE_FINALIZED"),
    @SerialName("SMS_QUICK_COMPLETE") SMSQuickComplete("SMS_QUICK_COMPLETE"),
    @SerialName("SETTLEMENT_REQUIRED") SettlementRequired("SETTLEMENT_REQUIRED"),
    @SerialName("SHOP_CLOSED") ShopClosed("SHOP_CLOSED"),
    @SerialName("SHOP_CLOSED_ALERT") ShopClosedAlert("SHOP_CLOSED_ALERT"),
    @SerialName("SHOP_CLOSED_ESCALATED") ShopClosedEscalated("SHOP_CLOSED_ESCALATED"),
    @SerialName("SHOP_CLOSED_RESOLVED") ShopClosedResolved("SHOP_CLOSED_RESOLVED"),
    @SerialName("SHOP_CLOSED_RESPONSE") ShopClosedResponse("SHOP_CLOSED_RESPONSE"),
    @SerialName("SPLIT_PAYMENT_CREATED") SplitPaymentCreated("SPLIT_PAYMENT_CREATED"),
    @SerialName("STOCK_BACKORDERED") StockBackordered("STOCK_BACKORDERED"),
    @SerialName("STOCK_THRESHOLD_BREACH") StockThresholdBreach("STOCK_THRESHOLD_BREACH"),
    @SerialName("SUPPLY_LANE_TRANSIT_UPDATED") SupplyLaneTransitUpdated("SUPPLY_LANE_TRANSIT_UPDATED"),
    @SerialName("SUPPLY_REQUEST_ACKNOWLEDGED") SupplyRequestAcknowledged("SUPPLY_REQUEST_ACKNOWLEDGED"),
    @SerialName("SUPPLY_REQUEST_CANCELLED") SupplyRequestCancelled("SUPPLY_REQUEST_CANCELLED"),
    @SerialName("SUPPLY_REQUEST_FULFILLED") SupplyRequestFulfilled("SUPPLY_REQUEST_FULFILLED"),
    @SerialName("SUPPLY_REQUEST_READY") SupplyRequestReady("SUPPLY_REQUEST_READY"),
    @SerialName("SUPPLY_REQUEST_SUBMITTED") SupplyRequestSubmitted("SUPPLY_REQUEST_SUBMITTED"),
    @SerialName("SUPPLY_REQUEST_UPDATE") SupplyRequestUpdate("SUPPLY_REQUEST_UPDATE"),
    @SerialName("SYSTEM_APP_OUTDATED") SystemAppOutdated("SYSTEM_APP_OUTDATED"),
    @SerialName("SYSTEM_BROADCAST") SystemBroadcast("SYSTEM_BROADCAST"),
    @SerialName("TOKEN_REFRESH_NEEDED") TokenRefreshNeeded("TOKEN_REFRESH_NEEDED"),
    @SerialName("TRANSFER_APPROVED") TransferApproved("TRANSFER_APPROVED"),
    @SerialName("TRANSFER_RECEIVED") TransferReceived("TRANSFER_RECEIVED"),
    @SerialName("TRANSFER_STATE_CHANGED") TransferStateChanged("TRANSFER_STATE_CHANGED"),
    @SerialName("TRANSFER_UNASSIGNED") TransferUnassigned("TRANSFER_UNASSIGNED"),
    @SerialName("UNIFIED_CHECKOUT_COMPLETED") UnifiedCheckoutCompleted("UNIFIED_CHECKOUT_COMPLETED"),
    @SerialName("VEHICLE_CREATED") VehicleCreated("VEHICLE_CREATED"),
    @SerialName("WAREHOUSE_CREATED") WarehouseCreated("WAREHOUSE_CREATED"),
    @SerialName("WAREHOUSE_SPATIAL_UPDATED") WarehouseSpatialUpdated("WAREHOUSE_SPATIAL_UPDATED"),
    @SerialName("WAREHOUSE_STATUS_CHANGED") WarehouseStatusChanged("WAREHOUSE_STATUS_CHANGED");
}
