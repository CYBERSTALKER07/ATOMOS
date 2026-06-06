package com.pegasusx.supplier.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class SupplierExceptionRow(
    @SerialName("order_id") val orderId: String,
    val kind: String = "",
    val status: String = "",
    @SerialName("retailer_id") val retailerId: String? = null,
    val note: String? = null,
    @SerialName("manifest_id") val manifestId: String? = null,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierExceptionsResponse(
    val exceptions: List<SupplierExceptionRow> = emptyList(),
)

@Serializable
data class SupplierManifestRow(
    @SerialName("manifest_id") val manifestId: String,
    val status: String = "",
    val state: String = "",
    @SerialName("orders_count") val ordersCount: Int = 0,
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("vehicle_plate") val vehiclePlate: String? = null,
    @SerialName("truck_id") val truckId: String? = null,
    @SerialName("total_vu") val totalVu: Long = 0,
    @SerialName("total_volume_vu") val totalVolumeVu: Double = 0.0,
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("stop_count") val stopCount: Int = 0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierManifestsResponse(
    val manifests: List<SupplierManifestRow> = emptyList(),
)

@Serializable
data class SupplierDispatchPreview(
    @SerialName("undispatched_orders") val undispatchedOrders: List<JsonElement> = emptyList(),
    @SerialName("available_drivers") val availableDrivers: List<JsonElement> = emptyList(),
    @SerialName("unavailable_drivers") val unavailableDrivers: List<JsonElement> = emptyList(),
    @SerialName("pending_count") val pendingCount: Int = 0,
    @SerialName("available_driver_count") val availableDriverCount: Int = 0,
)

@Serializable
data class SupplierPricingRule(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("base_markup_bps") val baseMarkupBps: Long = 0,
    @SerialName("retailer_discount_bps") val retailerDiscountBps: Long = 0,
    @SerialName("min_margin_bps") val minMarginBps: Long = 0,
    val currency: String = "",
    @SerialName("rule_version") val ruleVersion: Long = 0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierTopologyWarehouse(
    @SerialName("warehouse_id") val warehouseId: String,
    val name: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
)

@Serializable
data class SupplierTopologyFactory(
    @SerialName("factory_id") val factoryId: String,
    val name: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
)

@Serializable
data class SupplierTopologyResponse(
    @SerialName("supplier_id") val supplierId: String = "",
    val warehouses: List<SupplierTopologyWarehouse> = emptyList(),
    val factories: List<SupplierTopologyFactory> = emptyList(),
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierSupplyLaneRow(
    @SerialName("lane_id") val laneId: String,
    val name: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("h3_cells") val h3Cells: Int = 0,
    val drivers: Int = 0,
    @SerialName("orders_today") val ordersToday: Int = 0,
    val capacity: Int = 0,
    @SerialName("utilization_pct") val utilizationPct: Double = 0.0,
)

@Serializable
data class SupplierSupplyLanesResponse(
    val lanes: List<SupplierSupplyLaneRow> = emptyList(),
)

@Serializable
data class SupplierWsSessionResponse(
    val token: String = "",
    @SerialName("expires_at") val expiresAt: String = "",
)

@Serializable
data class SupplierFleetOrderRow(
    val id: String = "",
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String? = null,
    @SerialName("driver_id") val driverId: String? = null,
    val status: String = "",
    val state: String? = null,
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("total_minor") val totalMinor: Long? = null,
    val currency: String? = null,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class ShopClosedAttemptRow(
    @SerialName("attempt_id") val attemptId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    val resolution: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class ShopClosedActiveResponse(
    val data: List<ShopClosedAttemptRow> = emptyList(),
)

@Serializable
data class NegotiationProposalItem(
    @SerialName("sku_id") val skuId: String,
    @SerialName("original_qty") val originalQty: Int = 0,
    @SerialName("proposed_qty") val proposedQty: Int = 0,
)

@Serializable
data class NegotiationProposalRow(
    @SerialName("proposal_id") val proposalId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("driver_id") val driverId: String = "",
    val items: List<NegotiationProposalItem> = emptyList(),
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class NegotiationPendingResponse(
    val data: List<NegotiationProposalRow> = emptyList(),
)

@Serializable
data class ShopClosedResolveRequest(
    @SerialName("attempt_id") val attemptId: String,
    val action: String,
)

@Serializable
data class NegotiationResolveRequest(
    @SerialName("proposal_id") val proposalId: String,
    val action: String,
    val resolution: String? = null,
)

@Serializable
data class NegotiationResolveResponse(
    val status: String = "",
    @SerialName("proposal_id") val proposalId: String = "",
    @SerialName("order_id") val orderId: String = "",
)

@Serializable
data class SupplierReplenishmentTriggerResponse(
    val status: String = "",
    @SerialName("request_id") val requestId: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
)

@Serializable
data class PaymentLedgerEntry(
    @SerialName("ledger_entry_id") val ledgerEntryId: String,
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("supplier_id") val supplierId: String? = null,
    @SerialName("retailer_id") val retailerId: String? = null,
    val gateway: String = "",
    @SerialName("entry_type") val entryType: String = "",
    @SerialName("amount_minor") val amountMinor: Long = 0,
    val currency: String = "",
    @SerialName("occurred_at") val occurredAt: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class PaymentLedgerResponse(
    val items: List<PaymentLedgerEntry> = emptyList(),
    val count: Int = 0,
    val limit: Int = 0,
    @SerialName("supplier_id") val supplierId: String = "",
)

@Serializable
data class FleetDriverCreateRequest(
    val name: String,
    val phone: String,
    val pin: String,
    @SerialName("home_node_type") val homeNodeType: String,
    @SerialName("home_node_id") val homeNodeId: String,
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("is_active") val isActive: Boolean? = null,
)

@Serializable
data class FleetVehicleCreateRequest(
    val label: String? = null,
    @SerialName("license_plate") val licensePlate: String,
    @SerialName("home_node_type") val homeNodeType: String,
    @SerialName("home_node_id") val homeNodeId: String,
    @SerialName("is_active") val isActive: Boolean? = null,
)

@Serializable
data class SupplierActivityEvent(
    val id: String = "",
    val type: String = "",
    val timestamp: String = "",
    val description: String = "",
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("manifest_id") val manifestId: String? = null,
)

@Serializable
data class SupplierActivityResponse(
    val events: List<SupplierActivityEvent> = emptyList(),
)
