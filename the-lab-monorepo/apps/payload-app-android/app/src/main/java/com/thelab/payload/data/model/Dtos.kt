package com.thelab.payload.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// ─── Auth ────────────────────────────────────────────────────────────────────

@Serializable
data class LoginRequest(val phone: String, val pin: String)

@Serializable
data class LoginResponse(
    val token: String,
    @SerialName("worker_id") val workerId: String,
    @SerialName("supplier_id") val supplierId: String,
    val role: String,
    val name: String,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("warehouse_name") val warehouseName: String = "",
    @SerialName("warehouse_lat") val warehouseLat: Double = 0.0,
    @SerialName("warehouse_lng") val warehouseLng: Double = 0.0,
    @SerialName("firebase_token") val firebaseToken: String? = null,
)

// ─── Trucks ──────────────────────────────────────────────────────────────────
// Wire format: bare JSON array of {id, label, license_plate, vehicle_class}
// from supplier/staff.go::HandlePayloaderTrucks.

@Serializable
data class Truck(
    val id: String,
    val label: String = "",
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("vehicle_class") val vehicleClass: String = "",
)

// ─── Manifest / Live order ───────────────────────────────────────────────────

@Serializable
data class LiveOrderItem(
    @SerialName("line_item_id") val lineItemId: String,
    @SerialName("sku_id") val skuId: String,
    @SerialName("sku_name") val skuName: String,
    val quantity: Int,
    @SerialName("unit_price") val unitPrice: Long = 0,
    val status: String = "",
)

@Serializable
data class LiveOrder(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String = "",
    val amount: Long = 0,
    @SerialName("payment_gateway") val paymentGateway: String = "",
    val state: String = "",
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("warehouse_id") val warehouseId: String? = null,
    val items: List<LiveOrderItem> = emptyList(),
)

@Serializable
data class Manifest(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("truck_id") val truckId: String = "",
    @SerialName("driver_id") val driverId: String = "",
    val state: String = "DRAFT", // DRAFT | LOADING | SEALED | DISPATCHED | COMPLETED
    @SerialName("total_volume_vu") val totalVolumeVu: Double = 0.0,
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("stop_count") val stopCount: Int = 0,
    @SerialName("region_code") val regionCode: String = "",
    @SerialName("sealed_at") val sealedAt: String = "",
    @SerialName("dispatched_at") val dispatchedAt: String = "",
    @SerialName("created_at") val createdAt: String = "",
    // Hydrated by the detail endpoint only — Phase 4 wires this.
    val orders: List<LiveOrder> = emptyList(),
    @SerialName("overflow_count") val overflowCount: Int = 0,
)

@Serializable
data class ManifestsResponse(val manifests: List<Manifest> = emptyList())

// ─── Seal / dispatch ─────────────────────────────────────────────────────────
// Backend: order/service.go::PayloadSealRequest → {order_id, terminal_id, manifest_cleared}.
// terminal_id is the active vehicle/truck id (Expo passes activeTruck).

@Serializable
data class SealOrderRequest(
    @SerialName("order_id") val orderId: String,
    @SerialName("terminal_id") val terminalId: String,
    @SerialName("manifest_cleared") val manifestCleared: Boolean = true,
)

@Serializable
data class SealOrderResponse(
    val status: String = "",
    @SerialName("dispatch_code") val dispatchCode: String = "",
    @SerialName("order_id") val orderId: String = "",
)

@Serializable
data class SealManifestResponse(
    val status: String = "",
    @SerialName("stop_count") val stopCount: Int = 0,
    @SerialName("volume_vu") val volumeVu: Double = 0.0,
    @SerialName("max_vu") val maxVu: Double = 0.0,
)

// ─── Exception ───────────────────────────────────────────────────────────────

@Serializable
data class ManifestExceptionRequest(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("order_id") val orderId: String,
    val reason: String, // OVERFLOW | DAMAGED | MANUAL
    val metadata: String = "",
)

@Serializable
data class ManifestExceptionResponse(
    val status: String = "",
    val escalated: Boolean = false,
    @SerialName("overflow_count") val overflowCount: Int = 0,
)

// ─── Inject ──────────────────────────────────────────────────────────────────

@Serializable
data class InjectOrderRequest(
    @SerialName("order_id") val orderId: String,
)

// ─── Re-dispatch ─────────────────────────────────────────────────────────────

@Serializable
data class RecommendReassignRequest(
    @SerialName("order_id") val orderId: String,
)

@Serializable
data class TruckRecommendation(
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("vehicle_id") val vehicleId: String = "",
    @SerialName("vehicle_class") val vehicleClass: String = "",
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("used_volume_vu") val usedVolumeVu: Double = 0.0,
    @SerialName("free_volume_vu") val freeVolumeVu: Double = 0.0,
    @SerialName("distance_km") val distanceKm: Double = 0.0,
    @SerialName("order_count") val orderCount: Int = 0,
    @SerialName("truck_status") val truckStatus: String = "",
    val score: Double = 0.0,
    val recommendation: String = "",
)

@Serializable
data class RecommendReassignResponse(
    @SerialName("order_id") val orderId: String = "",
    @SerialName("retailer_name") val retailerName: String = "",
    @SerialName("order_volume_vu") val orderVolumeVu: Double = 0.0,
    @SerialName("current_driver") val currentDriver: String = "",
    val recommendations: List<TruckRecommendation> = emptyList(),
)

@Serializable
data class FleetReassignRequest(
    @SerialName("order_ids") val orderIds: List<String>,
    /** In this codebase RouteId == DriverId; payload terminal sends the recommended driver_id. */
    @SerialName("new_route_id") val newRouteId: String,
)

@Serializable
data class FleetReassignResponse(
    val conflicts: List<ReassignConflict> = emptyList(),
    val total: Int = 0,
    val reassigned: Int = 0,
    @SerialName("new_route_id") val newRouteId: String = "",
)

@Serializable
data class ReassignConflict(
    @SerialName("order_id") val orderId: String,
    val reason: String = "",
)

// ─── Missing items (Edge 33) ─────────────────────────────────────────────────

@Serializable
data class MissingItemEntry(
    @SerialName("line_item_id") val lineItemId: String,
    val quantity: Int,
)

@Serializable
data class MissingItemsRequest(
    @SerialName("order_id") val orderId: String,
    @SerialName("missing_items") val missingItems: List<MissingItemEntry>,
)

// ─── FCM ─────────────────────────────────────────────────────────────────────

@Serializable
data class DeviceTokenRequest(
    val token: String,
    val platform: String = "ANDROID",
)

// ─── Notifications ───────────────────────────────────────────────────────────
// Wire shape from notifications/inbox.go::NotificationItem.

@Serializable
data class NotificationItem(
    @SerialName("notification_id") val notificationId: String,
    val type: String = "",
    val title: String = "",
    val body: String = "",
    val payload: String? = null,
    val channel: String = "",
    @SerialName("read_at") val readAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
) {
    val isUnread: Boolean get() = readAt.isNullOrEmpty()
}

@Serializable
data class NotificationsResponse(
    val notifications: List<NotificationItem> = emptyList(),
    @SerialName("unread_count") val unreadCount: Long = 0,
    val total: Int = 0,
    val limit: Long = 0,
)

@Serializable
data class MarkReadRequest(
    @SerialName("notification_ids") val notificationIds: List<String>? = null,
    val all: Boolean? = null,
)

// ─── WebSocket frames ────────────────────────────────────────────────────────
// notification_dispatcher.go pushes flat: {type:<eventType>, title, body, channel}.
// Anything carrying `title` or `body` is rendered as an in-app notification —
// the literal `type` field carries the event name (ORDER_REASSIGNED, etc.).

@Serializable
data class WsMessage(
    val type: String = "",
    val title: String? = null,
    val body: String? = null,
    val channel: String? = null,
)

// ─── Offline action queue ────────────────────────────────────────────────────
// Persisted as JSON in SecureStore. Matches Expo's QueuedAction shape.

@Serializable
data class QueuedAction(
    val id: String,
    val endpoint: String,
    val method: String,
    val body: String,
    @SerialName("created_at") val createdAt: Long,
)

// ─── Generic ─────────────────────────────────────────────────────────────────

@Serializable
data class StatusResponse(val status: String = "")

/** RFC 7807 ProblemDetail — backend canonical error envelope. */
@Serializable
data class ProblemDetail(
    val type: String? = null,
    val title: String? = null,
    val status: Int = 0,
    val detail: String? = null,
    @SerialName("trace_id") val traceId: String? = null,
    val code: String? = null,
    val retryable: Boolean? = null,
)
