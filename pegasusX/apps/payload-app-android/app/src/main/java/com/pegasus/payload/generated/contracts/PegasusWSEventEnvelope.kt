// To parse the JSON, install Klaxon and do:
//
//   val pegasusWSEventEnvelope = PegasusWSEventEnvelope.fromJson(jsonString)

package com.pegasus.payload.generated.contracts

import com.beust.klaxon.*

private fun <T> Klaxon.convert(k: kotlin.reflect.KClass<*>, fromJson: (JsonValue) -> T, toJson: (T) -> String, isUnion: Boolean = false) =
    this.converter(object: Converter {
        @Suppress("UNCHECKED_CAST")
        override fun toJson(value: Any)        = toJson(value as T)
        override fun fromJson(jv: JsonValue)   = fromJson(jv) as Any
        override fun canConvert(cls: Class<*>) = cls == k.java || (isUnion && cls.superclass == k.java)
    })

private val klaxon = Klaxon()
    .convert(Type::class, { Type.fromValue(it.string!!) }, { "\"${it.value}\"" })

data class PegasusWSEventEnvelope (
    val type: Type
) {
    public fun toJson() = klaxon.toJsonString(this)

    companion object {
        public fun fromJson(json: String) = klaxon.parse<PegasusWSEventEnvelope>(json)
    }
}

enum class Type(val value: String) {
    AIRecommendationCreated("AI_RECOMMENDATION_CREATED"),
    AIRecommendationDecided("AI_RECOMMENDATION_DECIDED"),
    CartSyncUpdated("CART_SYNC_UPDATED"),
    CommandDispatched("COMMAND_DISPATCHED"),
    CommandReceived("COMMAND_RECEIVED"),
    CommandSettled("COMMAND_SETTLED"),
    DeliveryDisputed("DELIVERY_DISPUTED"),
    DeliverySessionUpdated("DELIVERY_SESSION_UPDATED"),
    DriverAvailabilityChanged("DRIVER_AVAILABILITY_CHANGED"),
    DriverCreated("DRIVER_CREATED"),
    DriverLocationUpdated("DRIVER_LOCATION_UPDATED"),
    FactoryCreated("FACTORY_CREATED"),
    FactorySupplyRequestUpdate("FACTORY_SUPPLY_REQUEST_UPDATE"),
    FreezeLockAcquired("FREEZE_LOCK_ACQUIRED"),
    FreezeLockReleased("FREEZE_LOCK_RELEASED"),
    InventoryImportStatusUpdate("INVENTORY_IMPORT_STATUS_UPDATE"),
    InventoryImportUploaded("INVENTORY_IMPORT_UPLOADED"),
    InventorySyncComplete("INVENTORY_SYNC_COMPLETE"),
    ManifestCancelled("MANIFEST_CANCELLED"),
    ManifestCompleted("MANIFEST_COMPLETED"),
    ManifestDispatched("MANIFEST_DISPATCHED"),
    ManifestDlqEscalation("MANIFEST_DLQ_ESCALATION"),
    ManifestDraftCreated("MANIFEST_DRAFT_CREATED"),
    ManifestLoadingStarted("MANIFEST_LOADING_STARTED"),
    ManifestOrderException("MANIFEST_ORDER_EXCEPTION"),
    ManifestOrderInjected("MANIFEST_ORDER_INJECTED"),
    ManifestRebalanced("MANIFEST_REBALANCED"),
    ManifestSealed("MANIFEST_SEALED"),
    MissingItemsReported("MISSING_ITEMS_REPORTED"),
    NegotiationProposed("NEGOTIATION_PROPOSED"),
    NegotiationResolved("NEGOTIATION_RESOLVED"),
    OrderAssigned("ORDER_ASSIGNED"),
    OrderCreated("ORDER_CREATED"),
    OrderFinalized("ORDER_FINALIZED"),
    OrderReassigned("ORDER_REASSIGNED"),
    OrderStatusChanged("ORDER_STATUS_CHANGED"),
    OrderValidationFailed("ORDER_VALIDATION_FAILED"),
    PaymentCleared("PAYMENT_CLEARED"),
    PaymentRequired("PAYMENT_REQUIRED"),
    PromotionChanged("PROMOTION_CHANGED"),
    RetailerPriceOverride("RETAILER_PRICE_OVERRIDE"),
    RetailerRegistered("RETAILER_REGISTERED"),
    RouteCreated("ROUTE_CREATED"),
    RouteReordered("ROUTE_REORDERED"),
    SettlementRequired("SETTLEMENT_REQUIRED"),
    ShopClosed("SHOP_CLOSED"),
    ShopClosedEscalated("SHOP_CLOSED_ESCALATED"),
    ShopClosedResolved("SHOP_CLOSED_RESOLVED"),
    ShopClosedResponse("SHOP_CLOSED_RESPONSE"),
    SplitPaymentCreated("SPLIT_PAYMENT_CREATED"),
    SupplierBillingConfigured("SUPPLIER_BILLING_CONFIGURED"),
    SupplierBillingUpdated("SUPPLIER_BILLING_UPDATED"),
    SupplierCreated("SUPPLIER_CREATED"),
    SupplierMemberAdded("SUPPLIER_MEMBER_ADDED"),
    SupplierProfileUpdated("SUPPLIER_PROFILE_UPDATED"),
    SupplierUpdated("SUPPLIER_UPDATED"),
    SupplyRequestAccepted("SUPPLY_REQUEST_ACCEPTED"),
    SupplyRequestUpdate("SUPPLY_REQUEST_UPDATE"),
    SystemAppOutdated("SYSTEM_APP_OUTDATED"),
    VehicleCreated("VEHICLE_CREATED"),
    WarehouseCreated("WAREHOUSE_CREATED"),
    WarehouseDispatchLockChanged("WAREHOUSE_DISPATCH_LOCK_CHANGED"),
    WarehouseSupplyRequestOpened("WAREHOUSE_SUPPLY_REQUEST_OPENED"),
    WarehouseTransferCreated("WAREHOUSE_TRANSFER_CREATED"),
    WarehouseTransferReceived("WAREHOUSE_TRANSFER_RECEIVED");

    companion object {
        public fun fromValue(value: String): Type = when (value) {
            "AI_RECOMMENDATION_CREATED"       -> AIRecommendationCreated
            "AI_RECOMMENDATION_DECIDED"       -> AIRecommendationDecided
            "CART_SYNC_UPDATED"               -> CartSyncUpdated
            "COMMAND_DISPATCHED"              -> CommandDispatched
            "COMMAND_RECEIVED"                -> CommandReceived
            "COMMAND_SETTLED"                 -> CommandSettled
            "DELIVERY_DISPUTED"               -> DeliveryDisputed
            "DELIVERY_SESSION_UPDATED"        -> DeliverySessionUpdated
            "DRIVER_AVAILABILITY_CHANGED"     -> DriverAvailabilityChanged
            "DRIVER_CREATED"                  -> DriverCreated
            "DRIVER_LOCATION_UPDATED"         -> DriverLocationUpdated
            "FACTORY_CREATED"                 -> FactoryCreated
            "FACTORY_SUPPLY_REQUEST_UPDATE"   -> FactorySupplyRequestUpdate
            "FREEZE_LOCK_ACQUIRED"            -> FreezeLockAcquired
            "FREEZE_LOCK_RELEASED"            -> FreezeLockReleased
            "INVENTORY_IMPORT_STATUS_UPDATE"  -> InventoryImportStatusUpdate
            "INVENTORY_IMPORT_UPLOADED"       -> InventoryImportUploaded
            "INVENTORY_SYNC_COMPLETE"         -> InventorySyncComplete
            "MANIFEST_CANCELLED"              -> ManifestCancelled
            "MANIFEST_COMPLETED"              -> ManifestCompleted
            "MANIFEST_DISPATCHED"             -> ManifestDispatched
            "MANIFEST_DLQ_ESCALATION"         -> ManifestDlqEscalation
            "MANIFEST_DRAFT_CREATED"          -> ManifestDraftCreated
            "MANIFEST_LOADING_STARTED"        -> ManifestLoadingStarted
            "MANIFEST_ORDER_EXCEPTION"        -> ManifestOrderException
            "MANIFEST_ORDER_INJECTED"         -> ManifestOrderInjected
            "MANIFEST_REBALANCED"             -> ManifestRebalanced
            "MANIFEST_SEALED"                 -> ManifestSealed
            "MISSING_ITEMS_REPORTED"          -> MissingItemsReported
            "NEGOTIATION_PROPOSED"            -> NegotiationProposed
            "NEGOTIATION_RESOLVED"            -> NegotiationResolved
            "ORDER_ASSIGNED"                  -> OrderAssigned
            "ORDER_CREATED"                   -> OrderCreated
            "ORDER_FINALIZED"                 -> OrderFinalized
            "ORDER_REASSIGNED"                -> OrderReassigned
            "ORDER_STATUS_CHANGED"            -> OrderStatusChanged
            "ORDER_VALIDATION_FAILED"         -> OrderValidationFailed
            "PAYMENT_CLEARED"                 -> PaymentCleared
            "PAYMENT_REQUIRED"                -> PaymentRequired
            "PROMOTION_CHANGED"               -> PromotionChanged
            "RETAILER_PRICE_OVERRIDE"         -> RetailerPriceOverride
            "RETAILER_REGISTERED"             -> RetailerRegistered
            "ROUTE_CREATED"                   -> RouteCreated
            "ROUTE_REORDERED"                 -> RouteReordered
            "SETTLEMENT_REQUIRED"             -> SettlementRequired
            "SHOP_CLOSED"                     -> ShopClosed
            "SHOP_CLOSED_ESCALATED"           -> ShopClosedEscalated
            "SHOP_CLOSED_RESOLVED"            -> ShopClosedResolved
            "SHOP_CLOSED_RESPONSE"            -> ShopClosedResponse
            "SPLIT_PAYMENT_CREATED"           -> SplitPaymentCreated
            "SUPPLIER_BILLING_CONFIGURED"     -> SupplierBillingConfigured
            "SUPPLIER_BILLING_UPDATED"        -> SupplierBillingUpdated
            "SUPPLIER_CREATED"                -> SupplierCreated
            "SUPPLIER_MEMBER_ADDED"           -> SupplierMemberAdded
            "SUPPLIER_PROFILE_UPDATED"        -> SupplierProfileUpdated
            "SUPPLIER_UPDATED"                -> SupplierUpdated
            "SUPPLY_REQUEST_ACCEPTED"         -> SupplyRequestAccepted
            "SUPPLY_REQUEST_UPDATE"           -> SupplyRequestUpdate
            "SYSTEM_APP_OUTDATED"             -> SystemAppOutdated
            "VEHICLE_CREATED"                 -> VehicleCreated
            "WAREHOUSE_CREATED"               -> WarehouseCreated
            "WAREHOUSE_DISPATCH_LOCK_CHANGED" -> WarehouseDispatchLockChanged
            "WAREHOUSE_SUPPLY_REQUEST_OPENED" -> WarehouseSupplyRequestOpened
            "WAREHOUSE_TRANSFER_CREATED"      -> WarehouseTransferCreated
            "WAREHOUSE_TRANSFER_RECEIVED"     -> WarehouseTransferReceived
            else                              -> throw IllegalArgumentException()
        }
    }
}
