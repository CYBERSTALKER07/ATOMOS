package com.thelab.factory.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// ── Auth ──
@Serializable
data class LoginRequest(
    val phone: String,
    val password: String,
)

@Serializable
data class AuthResponse(
    val token: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("factory_id") val factoryId: String,
    @SerialName("factory_name") val factoryName: String,
)

// ── Dashboard ──
@Serializable
data class DashboardStats(
    @SerialName("pending_transfers") val pendingTransfers: Int = 0,
    @SerialName("loading_transfers") val loadingTransfers: Int = 0,
    @SerialName("active_manifests") val activeManifests: Int = 0,
    @SerialName("dispatched_today") val dispatchedToday: Int = 0,
    @SerialName("vehicles_total") val vehiclesTotal: Int = 0,
    @SerialName("vehicles_available") val vehiclesAvailable: Int = 0,
    @SerialName("staff_on_shift") val staffOnShift: Int = 0,
    @SerialName("critical_insights") val criticalInsights: Int = 0,
)

// ── Transfers ──
@Serializable
data class Transfer(
    val id: String,
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("warehouse_name") val warehouseName: String = "",
    val state: String = "",
    val priority: String = "",
    @SerialName("total_items") val totalItems: Int = 0,
    @SerialName("total_volume_l") val totalVolumeL: Double = 0.0,
    val notes: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    val items: List<TransferItem> = emptyList(),
)

@Serializable
data class TransferItem(
    val id: String,
    @SerialName("product_id") val productId: String = "",
    @SerialName("product_name") val productName: String = "",
    val quantity: Int = 0,
    @SerialName("quantity_available") val quantityAvailable: Int = 0,
    @SerialName("unit_volume_l") val unitVolumeL: Double = 0.0,
)

@Serializable
data class TransferListResponse(
    val transfers: List<Transfer> = emptyList(),
    val total: Int = 0,
)

@Serializable
data class TransitionRequest(
    @SerialName("target_state") val targetState: String,
)

// ── Manifests ──
@Serializable
data class Manifest(
    val id: String,
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("truck_plate") val truckPlate: String = "",
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("transfer_ids") val transferIds: List<String> = emptyList(),
    val state: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

// ── Fleet ──
@Serializable
data class Vehicle(
    val id: String,
    @SerialName("plate_number") val plateNumber: String = "",
    @SerialName("driver_name") val driverName: String = "",
    val status: String = "",
    @SerialName("capacity_kg") val capacityKg: Double = 0.0,
    @SerialName("capacity_l") val capacityL: Double = 0.0,
    @SerialName("current_route") val currentRoute: String = "",
)

@Serializable
data class VehicleListResponse(
    val vehicles: List<Vehicle> = emptyList(),
)

// ── Staff ──
@Serializable
data class StaffMember(
    val id: String,
    val name: String = "",
    val phone: String = "",
    val role: String = "",
    val status: String = "",
    @SerialName("joined_at") val joinedAt: String = "",
)

@Serializable
data class StaffListResponse(
    val staff: List<StaffMember> = emptyList(),
)

// ── Insights ──
@Serializable
data class Insight(
    val id: String,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("warehouse_name") val warehouseName: String = "",
    @SerialName("product_id") val productId: String = "",
    @SerialName("product_name") val productName: String = "",
    val urgency: String = "",
    @SerialName("current_stock") val currentStock: Int = 0,
    @SerialName("avg_daily_velocity") val avgDailyVelocity: Double = 0.0,
    @SerialName("days_until_stockout") val daysUntilStockout: Int = 0,
    @SerialName("reorder_quantity") val reorderQuantity: Int = 0,
    val status: String = "",
)

@Serializable
data class InsightListResponse(
    val insights: List<Insight> = emptyList(),
)

// ── Dispatch ──
@Serializable
data class DispatchRequest(
    @SerialName("transfer_ids") val transferIds: List<String>,
)

@Serializable
data class DispatchResponse(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("truck_plate") val truckPlate: String = "",
    @SerialName("stop_count") val stopCount: Int = 0,
)
