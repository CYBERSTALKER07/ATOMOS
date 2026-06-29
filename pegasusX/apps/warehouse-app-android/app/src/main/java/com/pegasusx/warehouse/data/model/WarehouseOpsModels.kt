package com.pegasusx.warehouse.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class EmergencyTransferRequest(
    @SerialName("total_volume_vu") val totalVolumeVu: Double,
    val notes: String? = null,
)

@Serializable
data class ForceReceiveRequest(
    @SerialName("factory_id") val factoryId: String? = null,
    @SerialName("total_volume_vu") val totalVolumeVu: Double,
    val notes: String? = null,
)

@Serializable
data class TransferMutationResponse(
    @SerialName("transfer_id") val transferId: String,
    val state: String,
    val notes: String? = null,
)

@Serializable
data class ReplenishmentInsight(
    val id: String,
    @SerialName("warehouse_id") val warehouseId: String,
    @SerialName("warehouse_name") val warehouseName: String,
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String,
    val urgency: String = "",
    @SerialName("current_stock") val currentStock: Long = 0,
    @SerialName("avg_daily_velocity") val avgDailyVelocity: Double = 0.0,
    @SerialName("days_until_stockout") val daysUntilStockout: Int = 0,
    @SerialName("reorder_quantity") val reorderQuantity: Long = 0,
    val status: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class ReplenishmentInsightsResponse(
    val insights: List<ReplenishmentInsight> = emptyList(),
    val data: List<ReplenishmentInsight> = emptyList(),
) {
    fun resolved(): List<ReplenishmentInsight> =
        insights.ifEmpty { data }
}

@Serializable
data class DispatchSettingsResponse(
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("auto_dispatch_enabled") val autoDispatchEnabled: Boolean = false,
)

@Serializable
data class DispatchSettingsPatchRequest(
    @SerialName("auto_dispatch_enabled") val autoDispatchEnabled: Boolean,
)

@Serializable
data class ReplenishmentInsightActionResponse(
    @SerialName("insight_id") val insightId: String,
    val status: String,
    @SerialName("transfer_id") val transferId: String? = null,
)

@Serializable
data class OpsFinancialsResponse(
    @SerialName("warehouse_id") val warehouseId: String,
    val period: String = "",
    val currency: String = "",
    @SerialName("total_revenue") val totalRevenue: Long = 0,
    @SerialName("completed_orders") val completedOrders: Long = 0,
    @SerialName("avg_order_value") val avgOrderValue: Long = 0,
    @SerialName("platform_fee") val platformFee: Long = 0,
    @SerialName("net_payout") val netPayout: Long = 0,
    @SerialName("cash_pending") val cashPending: Long = 0,
    @SerialName("cash_collected") val cashCollected: Long = 0,
)

@Serializable
data class WarehouseOrderMutationRequest(
    val reason: String? = null,
)

@Serializable
data class WarehouseOrderMutationResponse(
    @SerialName("order_id") val orderId: String,
    val status: String,
)

@Serializable
data class RouteCoordinateWire(
    val lat: Double,
    val lng: Double,
)

@Serializable
data class RouteGeometryWire(
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("encoded_polyline") val encodedPolyline: String? = null,
    val coordinates: List<RouteCoordinateWire> = emptyList(),
    val source: String = "",
    @SerialName("stop_count") val stopCount: Int? = null,
)

@Serializable
data class WarehouseDriverLocationWire(
    @SerialName("driver_id") val driverId: String,
    @SerialName("supplier_id") val supplierId: String? = null,
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    val latitude: Double = 0.0,
    val longitude: Double = 0.0,
    @SerialName("reported_at") val reportedAt: String = "",
    @SerialName("received_at") val receivedAt: String = "",
    @SerialName("stale_after_seconds") val staleAfterSeconds: Int = 0,
)

@Serializable
data class WarehouseFleetLiveRoute(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("route_id") val routeId: String,
    @SerialName("driver_id") val driverId: String,
    @SerialName("driver_name") val driverName: String? = null,
    @SerialName("manifest_state") val manifestState: String,
    @SerialName("route_geometry") val routeGeometry: RouteGeometryWire? = null,
    @SerialName("driver_location") val driverLocation: WarehouseDriverLocationWire? = null,
    @SerialName("live_location_available") val liveLocationAvailable: Boolean = false,
    @SerialName("location_stale") val locationStale: Boolean? = null,
)

@Serializable
data class WarehouseFleetLiveMapResponse(
    val routes: List<WarehouseFleetLiveRoute> = emptyList(),
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("fetched_at") val fetchedAt: String = "",
)

@Serializable
data class BroadcastTemplate(
    val id: String,
    val category: String = "",
    val title: String,
    val body: String,
    @SerialName("default_role") val defaultRole: String = "DRIVER",
    val scope: String = "warehouse",
    val source: String? = null,
    @SerialName("warehouse_id") val warehouseId: String? = null,
    @SerialName("placeholder_keys") val placeholderKeys: List<String>? = null,
)

@Serializable
data class BroadcastTemplatesResponse(
    val templates: List<BroadcastTemplate> = emptyList(),
)

@Serializable
data class WarehouseBroadcastRequest(
    val title: String,
    val body: String,
    val role: String? = null,
)

@Serializable
data class WarehouseBroadcastResponse(
    val status: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
)

@Serializable
data class WarehouseBroadcastTemplateCreateRequest(
    val title: String,
    val body: String,
    @SerialName("default_role") val defaultRole: String? = null,
    val category: String? = null,
)

@Serializable
data class BroadcastTemplateDeleteResponse(
    val status: String = "",
    @SerialName("template_id") val templateId: String = "",
)

@Serializable
data class RetailerOverridePreview(
    @SerialName("retailers_on_sku_count") val retailersOnSkuCount: Int = 0,
    @SerialName("active_override_count") val activeOverrideCount: Int = 0,
    @SerialName("catalog_list_price") val catalogListPrice: Long = 0,
    @SerialName("margin_delta_per_unit") val marginDeltaPerUnit: Long = 0,
    @SerialName("margin_estimate_label") val marginEstimateLabel: String = "",
    @SerialName("affected_retailer_ids") val affectedRetailerIds: List<String>? = null,
    @SerialName("read_only") val readOnly: Boolean? = null,
)

@Serializable
data class RetailerOverridePreviewRequest(
    @SerialName("retailer_id") val retailerId: String? = null,
    @SerialName("product_id") val productId: String? = null,
    @SerialName("sku_id") val skuId: String? = null,
    @SerialName("proposed_price") val proposedPrice: Long,
)
