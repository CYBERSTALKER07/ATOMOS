// To parse the JSON, install kotlin's serialization plugin and do:
//
// val json                   = Json { allowStructuredMapKeys = true }
// val pegasusWSEventEnvelope = json.parse(PegasusWSEventEnvelope.serializer(), jsonString)

package com.pegasus.payload.generated.contracts

import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlinx.serialization.descriptors.*
import kotlinx.serialization.encoding.*

@Serializable
data class PegasusWSEventEnvelope (
    val type: Type
)

@Serializable
enum class Type(val value: String) {
    @SerialName("AI_RECOMMENDATION_CREATED") AIRecommendationCreated("AI_RECOMMENDATION_CREATED"),
    @SerialName("AI_RECOMMENDATION_DECIDED") AIRecommendationDecided("AI_RECOMMENDATION_DECIDED"),
    @SerialName("CART_SYNC_UPDATED") CartSyncUpdated("CART_SYNC_UPDATED"),
    @SerialName("COMMAND_DISPATCHED") CommandDispatched("COMMAND_DISPATCHED"),
    @SerialName("COMMAND_RECEIVED") CommandReceived("COMMAND_RECEIVED"),
    @SerialName("COMMAND_SETTLED") CommandSettled("COMMAND_SETTLED"),
    @SerialName("DELIVERY_DISPUTED") DeliveryDisputed("DELIVERY_DISPUTED"),
    @SerialName("DELIVERY_SESSION_UPDATED") DeliverySessionUpdated("DELIVERY_SESSION_UPDATED"),
    @SerialName("DRIVER_AVAILABILITY_CHANGED") DriverAvailabilityChanged("DRIVER_AVAILABILITY_CHANGED"),
    @SerialName("DRIVER_CREATED") DriverCreated("DRIVER_CREATED"),
    @SerialName("DRIVER_LOCATION_UPDATED") DriverLocationUpdated("DRIVER_LOCATION_UPDATED"),
    @SerialName("DRIVER_RETURN_APPROACHING") DriverReturnApproaching("DRIVER_RETURN_APPROACHING"),
    @SerialName("FACTORY_CREATED") FactoryCreated("FACTORY_CREATED"),
    @SerialName("FACTORY_LOCATION_UPDATED") FactoryLocationUpdated("FACTORY_LOCATION_UPDATED"),
    @SerialName("FACTORY_SUPPLY_REQUEST_UPDATE") FactorySupplyRequestUpdate("FACTORY_SUPPLY_REQUEST_UPDATE"),
    @SerialName("FREEZE_LOCK_ACQUIRED") FreezeLockAcquired("FREEZE_LOCK_ACQUIRED"),
    @SerialName("FREEZE_LOCK_RELEASED") FreezeLockReleased("FREEZE_LOCK_RELEASED"),
    @SerialName("INVENTORY_IMPORT_STATUS_UPDATE") InventoryImportStatusUpdate("INVENTORY_IMPORT_STATUS_UPDATE"),
    @SerialName("INVENTORY_IMPORT_UPLOADED") InventoryImportUploaded("INVENTORY_IMPORT_UPLOADED"),
    @SerialName("INVENTORY_SYNC_COMPLETE") InventorySyncComplete("INVENTORY_SYNC_COMPLETE"),
    @SerialName("MANIFEST_CANCELLED") ManifestCancelled("MANIFEST_CANCELLED"),
    @SerialName("MANIFEST_COMPLETED") ManifestCompleted("MANIFEST_COMPLETED"),
    @SerialName("MANIFEST_DISPATCHED") ManifestDispatched("MANIFEST_DISPATCHED"),
    @SerialName("MANIFEST_DLQ_ESCALATION") ManifestDlqEscalation("MANIFEST_DLQ_ESCALATION"),
    @SerialName("MANIFEST_DRAFT_CREATED") ManifestDraftCreated("MANIFEST_DRAFT_CREATED"),
    @SerialName("MANIFEST_LOADING_STARTED") ManifestLoadingStarted("MANIFEST_LOADING_STARTED"),
    @SerialName("MANIFEST_ORDER_EXCEPTION") ManifestOrderException("MANIFEST_ORDER_EXCEPTION"),
    @SerialName("MANIFEST_ORDER_INJECTED") ManifestOrderInjected("MANIFEST_ORDER_INJECTED"),
    @SerialName("MANIFEST_REBALANCED") ManifestRebalanced("MANIFEST_REBALANCED"),
    @SerialName("MANIFEST_SEALED") ManifestSealed("MANIFEST_SEALED"),
    @SerialName("MISSING_ITEMS_REPORTED") MissingItemsReported("MISSING_ITEMS_REPORTED"),
    @SerialName("NEGOTIATION_PROPOSED") NegotiationProposed("NEGOTIATION_PROPOSED"),
    @SerialName("NEGOTIATION_RESOLVED") NegotiationResolved("NEGOTIATION_RESOLVED"),
    @SerialName("ORDER_AMENDED") OrderAmended("ORDER_AMENDED"),
    @SerialName("ORDER_ASSIGNED") OrderAssigned("ORDER_ASSIGNED"),
    @SerialName("ORDER_CREATED") OrderCreated("ORDER_CREATED"),
    @SerialName("ORDER_FINALIZED") OrderFinalized("ORDER_FINALIZED"),
    @SerialName("ORDER_REASSIGNED") OrderReassigned("ORDER_REASSIGNED"),
    @SerialName("ORDER_STATUS_CHANGED") OrderStatusChanged("ORDER_STATUS_CHANGED"),
    @SerialName("ORDER_VALIDATION_FAILED") OrderValidationFailed("ORDER_VALIDATION_FAILED"),
    @SerialName("PAYMENT_CLEARED") PaymentCleared("PAYMENT_CLEARED"),
    @SerialName("PAYMENT_REQUIRED") PaymentRequired("PAYMENT_REQUIRED"),
    @SerialName("PRE_ORDER_AUTO_ACCEPTED") PreOrderAutoAccepted("PRE_ORDER_AUTO_ACCEPTED"),
    @SerialName("PRE_ORDER_CANCELLED") PreOrderCancelled("PRE_ORDER_CANCELLED"),
    @SerialName("PRE_ORDER_CONFIRMATION") PreOrderConfirmation("PRE_ORDER_CONFIRMATION"),
    @SerialName("PRE_ORDER_CONFIRMED") PreOrderConfirmed("PRE_ORDER_CONFIRMED"),
    @SerialName("PRE_ORDER_DATE_ACCEPTED") PreOrderDateAccepted("PRE_ORDER_DATE_ACCEPTED"),
    @SerialName("PRE_ORDER_DATE_PROPOSED") PreOrderDateProposed("PRE_ORDER_DATE_PROPOSED"),
    @SerialName("PRE_ORDER_DATE_REJECTED") PreOrderDateRejected("PRE_ORDER_DATE_REJECTED"),
    @SerialName("PRE_ORDER_EDITED") PreOrderEdited("PRE_ORDER_EDITED"),
    @SerialName("PRE_ORDER_NOTIFIED") PreOrderNotified("PRE_ORDER_NOTIFIED"),
    @SerialName("PRE_ORDER_NUDGE") PreOrderNudge("PRE_ORDER_NUDGE"),
    @SerialName("PROMOTION_CHANGED") PromotionChanged("PROMOTION_CHANGED"),
    @SerialName("RETAILER_PRICE_OVERRIDE") RetailerPriceOverride("RETAILER_PRICE_OVERRIDE"),
    @SerialName("RETAILER_REGISTERED") RetailerRegistered("RETAILER_REGISTERED"),
    @SerialName("RETURN_RECEIVED_AT_WAREHOUSE") ReturnReceivedAtWarehouse("RETURN_RECEIVED_AT_WAREHOUSE"),
    @SerialName("ROUTE_CREATED") RouteCreated("ROUTE_CREATED"),
    @SerialName("ROUTE_REORDERED") RouteReordered("ROUTE_REORDERED"),
    @SerialName("SETTLEMENT_REQUIRED") SettlementRequired("SETTLEMENT_REQUIRED"),
    @SerialName("SHOP_CLOSED") ShopClosed("SHOP_CLOSED"),
    @SerialName("SHOP_CLOSED_ESCALATED") ShopClosedEscalated("SHOP_CLOSED_ESCALATED"),
    @SerialName("SHOP_CLOSED_RESOLVED") ShopClosedResolved("SHOP_CLOSED_RESOLVED"),
    @SerialName("SHOP_CLOSED_RESPONSE") ShopClosedResponse("SHOP_CLOSED_RESPONSE"),
    @SerialName("SPLIT_PAYMENT_CREATED") SplitPaymentCreated("SPLIT_PAYMENT_CREATED"),
    @SerialName("SUPPLIER_BILLING_CONFIGURED") SupplierBillingConfigured("SUPPLIER_BILLING_CONFIGURED"),
    @SerialName("SUPPLIER_BILLING_UPDATED") SupplierBillingUpdated("SUPPLIER_BILLING_UPDATED"),
    @SerialName("SUPPLIER_CREATED") SupplierCreated("SUPPLIER_CREATED"),
    @SerialName("SUPPLIER_MEMBER_ADDED") SupplierMemberAdded("SUPPLIER_MEMBER_ADDED"),
    @SerialName("SUPPLIER_PROFILE_UPDATED") SupplierProfileUpdated("SUPPLIER_PROFILE_UPDATED"),
    @SerialName("SUPPLIER_RETURN_CREATED") SupplierReturnCreated("SUPPLIER_RETURN_CREATED"),
    @SerialName("SUPPLIER_RETURN_RESOLVED") SupplierReturnResolved("SUPPLIER_RETURN_RESOLVED"),
    @SerialName("SUPPLIER_UPDATED") SupplierUpdated("SUPPLIER_UPDATED"),
    @SerialName("SUPPLY_REQUEST_ACCEPTED") SupplyRequestAccepted("SUPPLY_REQUEST_ACCEPTED"),
    @SerialName("SUPPLY_REQUEST_UPDATE") SupplyRequestUpdate("SUPPLY_REQUEST_UPDATE"),
    @SerialName("SUPPLY_TRANSFER_APPROACHING") SupplyTransferApproaching("SUPPLY_TRANSFER_APPROACHING"),
    @SerialName("SYSTEM_APP_OUTDATED") SystemAppOutdated("SYSTEM_APP_OUTDATED"),
    @SerialName("VEHICLE_AVAILABILITY_CHANGED") VehicleAvailabilityChanged("VEHICLE_AVAILABILITY_CHANGED"),
    @SerialName("VEHICLE_CREATED") VehicleCreated("VEHICLE_CREATED"),
    @SerialName("WAREHOUSE_CREATED") WarehouseCreated("WAREHOUSE_CREATED"),
    @SerialName("WAREHOUSE_DISPATCH_LOCK_CHANGED") WarehouseDispatchLockChanged("WAREHOUSE_DISPATCH_LOCK_CHANGED"),
    @SerialName("WAREHOUSE_LOCATION_UPDATED") WarehouseLocationUpdated("WAREHOUSE_LOCATION_UPDATED"),
    @SerialName("WAREHOUSE_SUPPLY_REQUEST_OPENED") WarehouseSupplyRequestOpened("WAREHOUSE_SUPPLY_REQUEST_OPENED"),
    @SerialName("WAREHOUSE_TRANSFER_CREATED") WarehouseTransferCreated("WAREHOUSE_TRANSFER_CREATED"),
    @SerialName("WAREHOUSE_TRANSFER_RECEIVED") WarehouseTransferReceived("WAREHOUSE_TRANSFER_RECEIVED");
}
