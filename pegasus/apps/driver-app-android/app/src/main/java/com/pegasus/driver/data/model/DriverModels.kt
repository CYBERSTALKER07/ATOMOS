package com.pegasus.driver.data.model

import androidx.room.Entity
import androidx.room.PrimaryKey
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// ── RFC 7807 Problem Detail (mirrors backend errors.ProblemDetail) ──

@Serializable
data class ProblemDetail(
    val type: String? = null,
    val title: String? = null,
    val status: Int,
    val detail: String? = null,
    @SerialName("trace_id") val traceId: String? = null,
    val instance: String? = null,
    val code: String? = null,
    @SerialName("message_key") val messageKey: String? = null,
    val retryable: Boolean? = null,
    val action: String? = null,
)

class ProblemDetailException(val problem: ProblemDetail) :
    Exception(problem.detail ?: problem.title ?: "Server error ${problem.status}")

// ── API Response Models ──

@Serializable
enum class OrderState {
    PENDING, PENDING_REVIEW, SCHEDULED, LOADED, DISPATCHED, IN_TRANSIT, ARRIVING, ARRIVED,
    ARRIVED_SHOP_CLOSED, AWAITING_PAYMENT, PENDING_CASH_COLLECTION,
    CANCEL_REQUESTED, NO_CAPACITY, COMPLETED, CANCELLED,
    QUARANTINE, DELIVERED_ON_CREDIT
}

@Serializable
data class Order(
    val id: String,
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("retailer_name") val retailerName: String,
    @SerialName("driver_id") val driverId: String? = null,
    val state: OrderState,
    @SerialName("total_amount") val totalAmount: Long,
    @SerialName("delivery_address") val deliveryAddress: String,
    val latitude: Double? = null,
    val longitude: Double? = null,
    @SerialName("qr_token") val qrToken: String? = null,
    @SerialName("payment_gateway") val paymentGateway: String? = null,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val items: List<OrderLineItem> = emptyList(),
    @SerialName("estimated_arrival_at") val estimatedArrivalAt: String? = null,
    @SerialName("eta_duration_sec") val etaDurationSec: Int? = null,
    @SerialName("eta_distance_m") val etaDistanceM: Int? = null,
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("sequence_index") val sequenceIndex: Int = 0
)

@Serializable
data class OrderLineItem(
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String,
    val quantity: Int,
    @SerialName("unit_price") val unitPrice: Long,
    @SerialName("line_total") val lineTotal: Long
)

@Serializable
data class RouteManifest(
    @SerialName("driver_id") val driverId: String,
    val date: String,
    val orders: List<Order>,
    @SerialName("total_stops") val totalStops: Int,
    @SerialName("estimated_distance_km") val estimatedDistanceKm: Double? = null
)

@Serializable
data class DeliverySubmitRequest(
    @SerialName("order_id") val orderId: String,
    @SerialName("qr_token") val qrToken: String,
    val latitude: Double,
    val longitude: Double
)

@Serializable
data class DeliverySubmitResponse(
    val success: Boolean,
    val message: String,
    @SerialName("new_state") val newState: OrderState? = null
)

@Serializable
data class AmendOrderRequest(
    @SerialName("order_id") val orderId: String,
    val items: List<AmendItemPayload>,
    @SerialName("driver_notes") val driverNotes: String? = null
)

@Serializable
data class AmendItemPayload(
    @SerialName("product_id") val productId: String,
    @SerialName("accepted_qty") val acceptedQty: Int,
    @SerialName("rejected_qty") val rejectedQty: Int,
    val reason: String // "DAMAGED", "MISSING", "WRONG_ITEM", "OTHER"
)

@Serializable
data class AmendOrderResponse(
    val success: Boolean,
    val message: String,
    @SerialName("adjusted_total") val adjustedTotal: Long? = null
)

enum class RejectionReason(val label: String) {
    DAMAGED("Damaged"),
    MISSING("Missing"),
    WRONG_ITEM("Wrong Item"),
    OTHER("Other")
}

// ── QR Validation ──

@Serializable
data class ValidateQRRequest(
    @SerialName("order_id") val orderId: String,
    @SerialName("scanned_token") val scannedToken: String
)

@Serializable
data class ValidateQRResponse(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_name") val retailerName: String,
    @SerialName("total_amount") val totalAmount: Long,
    val state: OrderState,
    val items: List<OrderLineItem> = emptyList()
)

// ── Confirm Offload ──

@Serializable
data class ConfirmOffloadRequest(
    @SerialName("order_id") val orderId: String
)

@Serializable
data class ConfirmOffloadResponse(
    @SerialName("order_id") val orderId: String,
    val state: OrderState,
    @SerialName("payment_method") val paymentMethod: String,
    @SerialName("amount") val amount: Long,
    @SerialName("invoice_id") val invoiceId: String? = null,
    @SerialName("retailer_id") val retailerId: String? = null,
    val message: String = ""
)

// ── Complete Order ──

@Serializable
data class CompleteOrderRequest(
    @SerialName("order_id") val orderId: String,
    val latitude: Double? = null,
    val longitude: Double? = null
)

// ── Collect Cash (geofenced) ──

@Serializable
data class CollectCashRequest(
    @SerialName("order_id") val orderId: String,
    val latitude: Double,
    val longitude: Double
)

@Serializable
data class CollectCashResponse(
    @SerialName("order_id") val orderId: String,
    val state: String = "",
    @SerialName("amount") val amount: Long = 0,
    @SerialName("distance_m") val distanceM: Double = 0.0,
    val message: String = ""
)

@Serializable
data class DepartRequest(
    @SerialName("truck_id") val truckId: String
)

@Serializable
data class ReturnCompleteRequest(
    @SerialName("truck_id") val truckId: String
)

@Serializable
data class AvailabilityRequest(
    val available: Boolean,
    val reason: String? = null,
    val note: String? = null
)

@Serializable
data class ReorderStopsRequest(
    @SerialName("route_id") val routeId: String,
    @SerialName("order_sequence") val orderSequence: List<String>
)

@Serializable
data class TelemetryPayload(
    @SerialName("driver_id") val driverId: String,
    val latitude: Double,
    val longitude: Double,
    val timestamp: Long,
    val speed: Float = 0f,
    val bearing: Float = 0f
)

@Serializable
data class AuthResponse(
    val token: String,
    val role: String,
    @SerialName("user_id") val userId: String,
    val name: String = "",
    @SerialName("vehicle_type") val vehicleType: String = "",
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("vehicle_id") val vehicleId: String = "",
    @SerialName("vehicle_class") val vehicleClass: String = "",
    @SerialName("max_volume_vu") val maxVolumeVU: Double = 0.0,
    @SerialName("firebase_token") val firebaseToken: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("warehouse_name") val warehouseName: String = "",
    @SerialName("warehouse_lat") val warehouseLat: Double = 0.0,
    @SerialName("warehouse_lng") val warehouseLng: Double = 0.0,
    @SerialName("home_node_type") val homeNodeType: String = "",
    @SerialName("home_node_id") val homeNodeId: String = "",
    @SerialName("driver_mode") val driverMode: String = "",
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("factory_name") val factoryName: String = "",
    @SerialName("factory_lat") val factoryLat: Double = 0.0,
    @SerialName("factory_lng") val factoryLng: Double = 0.0,
)

@Serializable
data class DriverProfileResponse(
    @SerialName("driver_id") val driverId: String,
    val name: String,
    val phone: String,
    @SerialName("driver_type") val driverType: String = "",
    @SerialName("vehicle_type") val vehicleType: String = "",
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("vehicle_id") val vehicleId: String = "",
    @SerialName("vehicle_class") val vehicleClass: String = "",
    @SerialName("max_volume_vu") val maxVolumeVU: Double = 0.0,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("warehouse_name") val warehouseName: String = "",
    @SerialName("warehouse_lat") val warehouseLat: Double = 0.0,
    @SerialName("warehouse_lng") val warehouseLng: Double = 0.0,
    @SerialName("home_node_type") val homeNodeType: String = "",
    @SerialName("home_node_id") val homeNodeId: String = "",
    @SerialName("driver_mode") val driverMode: String = "",
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("factory_name") val factoryName: String = "",
    @SerialName("factory_lat") val factoryLat: Double = 0.0,
    @SerialName("factory_lng") val factoryLng: Double = 0.0,
)

@Serializable
data class LoginRequest(
    val phone: String,
    val pin: String
)

// ── Room Entities ──

@Entity(tableName = "orders")
data class OrderEntity(
    @PrimaryKey val id: String,
    val retailerId: String,
    val retailerName: String,
    val state: String,
    val totalAmount: Long,
    val deliveryAddress: String,
    val latitude: Double?,
    val longitude: Double?,
    val qrToken: String?,
    val createdAt: String,
    val updatedAt: String,
    val itemsJson: String // serialized OrderLineItem list
)

@Entity(tableName = "route_manifest")
data class RouteManifestEntity(
    @PrimaryKey val date: String,
    val driverId: String,
    val totalStops: Int,
    val estimatedDistanceKm: Double?,
    val fetchedAt: Long = System.currentTimeMillis()
)

@Entity(tableName = "pending_mutations")
data class PendingMutationEntity(
    @PrimaryKey val id: String,          // UUID generated client-side
    val endpoint: String,                 // e.g. "v1/order/deliver"
    val payloadJson: String,              // serialized request body
    val idempotencyKey: String,           // UUID sent as Idempotency-Key header
    val createdAt: Long = System.currentTimeMillis()
)
