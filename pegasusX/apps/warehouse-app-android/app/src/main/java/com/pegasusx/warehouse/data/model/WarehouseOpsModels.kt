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
