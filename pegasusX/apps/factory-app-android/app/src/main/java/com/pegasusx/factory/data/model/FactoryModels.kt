package com.pegasusx.factory.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// ── Auth ──
@Serializable
data class LoginRequest(
    val phone: String = "",
    val password: String = "",
    val pin: String = "",
    @SerialName("id_token") val idToken: String = "",
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

// mirror of backend-go/factory HandleAnalyticsOverview + packages/types FactoryAnalyticsOverviewResponse
@Serializable
data class FactoryAnalyticsDayBucket(
    val date: String = "",
    val transfers: Long = 0,
)

@Serializable
data class FactoryAnalyticsOverview(
    @SerialName("daily_activity") val dailyActivity: List<FactoryAnalyticsDayBucket> = emptyList(),
    @SerialName("transfers_total") val transfersTotal: Long = 0,
    @SerialName("manifests_active") val manifestsActive: Long = 0,
    @SerialName("exception_queue") val exceptionQueue: Long = 0,
    @SerialName("avg_lead_time_mins") val avgLeadTimeMins: Double = 0.0,
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

// ── Supply Requests ──
@Serializable
data class SupplyRequestListResponse(
    val requests: List<SupplyRequest> = emptyList(),
)

@Serializable
data class SupplyRequest(
    @SerialName("request_id") val id: String,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    val state: String = "",
    val priority: String = "",
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String? = null,
    @SerialName("total_volume_vu") val totalVolumeVU: Double = 0.0,
    val notes: String = "",
    @SerialName("transfer_order_id") val transferOrderId: String = "",
    @SerialName("created_by") val createdBy: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class SupplyRequestTransitionRequest(
    val action: String,
    @SerialName("transfer_order_id") val transferOrderId: String? = null,
)

@Serializable
data class SupplyRequestTransitionResponse(
    @SerialName("request_id") val requestId: String,
    val state: String,
)

// ── Manifests ──
@Serializable
data class Manifest(
    @SerialName("manifest_id") val id: String,
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("vehicle_id") val vehicleId: String = "",
    @SerialName("vehicle_label") val vehicleLabel: String = "",
    @SerialName("truck_id") val truckId: String = "",
    @SerialName("truck_plate") val truckPlate: String = "",
    val state: String = "",
    val status: String = "",
    @SerialName("total_volume_vu") val totalVolumeVU: Double = 0.0,
    @SerialName("max_volume_vu") val maxVolumeVU: Double = 0.0,
    @SerialName("max_capacity_vu") val maxCapacityVU: Double = 0.0,
    @SerialName("stop_count") val stopCount: Int = 0,
    @SerialName("region_code") val regionCode: String = "",
    @SerialName("created_at") val createdAt: String = "",
    val transfers: List<ManifestTransfer> = emptyList(),
)

@Serializable
data class ManifestTransfer(
    @SerialName("transfer_id") val transferId: String,
    @SerialName("product_name") val productName: String = "",
    val quantity: Int = 0,
    @SerialName("volume_vu") val volumeVU: Double = 0.0,
    val state: String = "",
)

@Serializable
data class ManifestListResponse(
    val manifests: List<Manifest> = emptyList(),
    val total: Int = 0,
)

@Serializable
data class ManifestDetailCore(
    @SerialName("manifest_id") val id: String,
    val state: String = "",
    @SerialName("transfer_count") val transferCount: Int = 0,
    @SerialName("total_volume_vu") val totalVolumeVU: Long = 0,
    @SerialName("max_volume_vu") val maxVolumeVU: Long = 0,
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("vehicle_id") val vehicleId: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class ManifestTransitionRow(
    val action: String = "",
    @SerialName("from_state") val fromState: String = "",
    @SerialName("to_state") val toState: String = "",
    val at: String = "",
    val reason: String = "",
)

@Serializable
data class ManifestDetailResponse(
    val manifest: ManifestDetailCore,
    val transfers: List<ManifestTransfer> = emptyList(),
    val transitions: List<ManifestTransitionRow> = emptyList(),
    val exceptions: List<ManifestException> = emptyList(),
    @SerialName("route_id") val routeId: String = "",
    @SerialName("stop_count") val stopCount: Int = 0,
    @SerialName("order_count") val orderCount: Int = 0,
)

@Serializable
data class ManifestTransitionRequest(
    val reason: String = "",
)

@Serializable
data class ManifestTransitionResponse(
    val status: String = "",
    @SerialName("manifest_id") val manifestId: String = "",
    val state: String = "",
)

@Serializable
data class ManifestRebalanceRequest(
    @SerialName("source_manifest_id") val sourceManifestId: String,
    @SerialName("target_manifest_id") val targetManifestId: String,
    @SerialName("transfer_ids") val transferIds: List<String>,
)

@Serializable
data class ManifestRebalanceResponse(
    @SerialName("source_manifest_id") val sourceManifestId: String,
    @SerialName("target_manifest_id") val targetManifestId: String,
    @SerialName("transfers_moved") val transfersMoved: Int = 0,
    @SerialName("volume_moved_vu") val volumeMovedVU: Double = 0.0,
    val reason: String = "",
)

@Serializable
data class ManifestCancelTransferRequest(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("transfer_id") val transferId: String,
)

@Serializable
data class ManifestCancelTransferResponse(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("transfer_id") val transferId: String,
    val status: String = "",
)

@Serializable
data class ManifestCancelRequest(
    @SerialName("manifest_id") val manifestId: String,
)

@Serializable
data class ManifestCancelResponse(
    @SerialName("manifest_id") val manifestId: String,
    val status: String = "",
    @SerialName("transfers_released") val transfersReleased: Int = 0,
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

// ── Manifest exceptions ──
@Serializable
data class ManifestException(
    @SerialName("exception_id") val exceptionId: String,
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("transfer_id") val transferId: String,
    val reason: String = "",
    val metadata: String = "",
    @SerialName("attempt_count") val attemptCount: Int = 0,
    val escalated: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("correlation_id") val correlationId: String = "",
)

@Serializable
data class ManifestExceptionListResponse(
    val exceptions: List<ManifestException> = emptyList(),
)

// ── Transfer create / fleet pickers ──
@Serializable
data class CreateTransferRequest(
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("total_vu") val totalVu: Long = 25,
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("vehicle_id") val vehicleId: String? = null,
)

@Serializable
data class CreateTransferResponse(
    @SerialName("transfer_id") val transferId: String,
    val state: String = "",
    @SerialName("total_vu") val totalVu: Long = 0,
)

@Serializable
data class FleetDriverRow(
    @SerialName("driver_id") val driverId: String,
    val name: String = "",
    @SerialName("on_shift") val onShift: Boolean = false,
)

@Serializable
data class FleetDriverListResponse(
    val drivers: List<FleetDriverRow> = emptyList(),
)

@Serializable
data class FleetVehicleRow(
    @SerialName("vehicle_id") val vehicleId: String,
    @SerialName("plate_no") val plateNo: String = "",
    val state: String = "",
)

@Serializable
data class FleetVehicleListResponse(
    val vehicles: List<FleetVehicleRow> = emptyList(),
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

// ── Notifications + client policy ──
@Serializable
data class NotificationItem(
    val id: String = "",
    val type: String = "",
    val title: String = "",
    val body: String = "",
    val payload: String = "",
    val channel: String = "",
    @SerialName("read_at") val readAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class NotificationsResponse(
    val notifications: List<NotificationItem> = emptyList(),
    @SerialName("unread_count") val unreadCount: Int = 0,
    @SerialName("has_more") val hasMore: Boolean = false,
)

@Serializable
data class MarkNotificationsReadRequest(
    @SerialName("notification_ids") val notificationIds: List<String>? = null,
    @SerialName("mark_all") val markAll: Boolean = false,
)

@Serializable
data class ClientPolicyResponse(
    val role: String = "",
    val platform: String = "",
    val channel: String = "",
    @SerialName("client_version") val clientVersion: String = "",
    @SerialName("minimum_version") val minimumVersion: String = "",
    @SerialName("recommended_version") val recommendedVersion: String = "",
    @SerialName("force_update") val forceUpdate: Boolean = false,
    @SerialName("update_url") val updateUrl: String? = null,
    @SerialName("update_deferred") val updateDeferred: Boolean = false,
    @SerialName("defer_reason") val deferReason: String? = null,
    val outdated: Boolean = false,
)
