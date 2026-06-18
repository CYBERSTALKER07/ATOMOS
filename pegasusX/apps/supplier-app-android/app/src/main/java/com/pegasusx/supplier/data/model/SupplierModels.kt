package com.pegasusx.supplier.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class LoginRequest(
    val phone: String,
    val password: String,
)

@Serializable
data class LoginResponse(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("is_configured") val isConfigured: Boolean,
    @SerialName("next_step") val nextStep: String = "",
    val token: String? = null,
    @SerialName("refresh_token") val refreshToken: String? = null,
)

@Serializable
data class RefreshTokenRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

@Serializable
data class SupplierDashboard(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("is_configured") val isConfigured: Boolean,
    @SerialName("inventory_skus") val inventorySKUs: Int = 0,
    @SerialName("pending_orders") val pendingOrders: Int = 0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierProfile(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("legal_name") val legalName: String = "",
    @SerialName("contact_name") val contactName: String = "",
    val email: String = "",
    val phone: String = "",
    val country: String = "",
    val currency: String = "",
    val categories: List<String> = emptyList(),
    @SerialName("is_registered") val isRegistered: Boolean = false,
    @SerialName("is_configured") val isConfigured: Boolean = false,
    @SerialName("selected_gateways") val selectedGateways: List<String> = emptyList(),
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierOrdersResponse(
    val orders: List<SupplierOrder> = emptyList(),
    val total: Int? = null,
    val limit: Int? = null,
    val offset: Int? = null,
)

@Serializable
data class SupplierOrder(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String,
    val status: String = "",
    val decision: String? = null,
    val note: String? = null,
    @SerialName("total_minor") val totalMinor: Long = 0,
    val currency: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class FleetDriversResponse(
    @SerialName("supplier_id") val supplierId: String = "",
    val items: List<FleetDriver> = emptyList(),
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class FleetDriver(
    @SerialName("driver_id") val driverId: String,
    val name: String = "",
    val phone: String = "",
    @SerialName("home_node_type") val homeNodeType: String = "",
    @SerialName("home_node_id") val homeNodeId: String = "",
    @SerialName("is_active") val isActive: Boolean = false,
)

@Serializable
data class FleetVehiclesResponse(
    @SerialName("supplier_id") val supplierId: String = "",
    val items: List<FleetVehicle> = emptyList(),
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class FleetVehicle(
    @SerialName("vehicle_id") val vehicleId: String,
    val label: String? = null,
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("home_node_type") val homeNodeType: String = "",
    @SerialName("home_node_id") val homeNodeId: String = "",
    @SerialName("is_active") val isActive: Boolean = false,
)

@Serializable
data class InventoryListResponse(
    val items: List<InventoryItem> = emptyList(),
)

@Serializable
data class InventoryItem(
    val sku: String,
    @SerialName("product_name") val productName: String = "",
    val quantity: Long = 0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class CatalogCategory(
    @SerialName("category_id") val categoryId: String,
    val name: String = "",
)

@Serializable
data class CatalogUploadTicket(
    @SerialName("upload_url") val uploadUrl: String,
    @SerialName("image_url") val imageUrl: String,
)

@Serializable
data class CatalogProductCreateRequest(
    @SerialName("category_id") val categoryId: String,
    val name: String,
    val description: String = "",
    @SerialName("price_minor") val priceMinor: Long,
    val currency: String,
    @SerialName("unit_volume_vu") val unitVolumeVu: Double,
    @SerialName("stock_quantity") val stockQuantity: Long = 0,
    val unit: String = "UNIT",
    @SerialName("image_url") val imageUrl: String? = null,
    val barcode: String? = null,
)

@Serializable
data class CatalogProduct(
    @SerialName("product_id") val productId: String,
    val name: String = "",
    @SerialName("category_id") val categoryId: String = "",
    @SerialName("price_minor") val priceMinor: Long = 0,
    val currency: String = "",
    val unit: String = "UNIT",
    @SerialName("unit_volume_vu") val unitVolumeVu: Double = 1.0,
    @SerialName("image_url") val imageUrl: String? = null,
    val barcode: String? = null,
    @SerialName("is_active") val isActive: Boolean = true,
    val version: Long = 0,
)

@Serializable
data class CatalogProductUpdateRequest(
    val name: String,
    @SerialName("price_minor") val priceMinor: Long,
    val currency: String,
    val unit: String,
    @SerialName("unit_volume_vu") val unitVolumeVu: Double,
    @SerialName("image_url") val imageUrl: String? = null,
    val barcode: String? = null,
    @SerialName("is_active") val isActive: Boolean,
    val version: Long,
)

@Serializable
data class SupplierEarnings(
    val currency: String = "",
    @SerialName("today_minor") val todayMinor: Long = 0,
    @SerialName("week_minor") val weekMinor: Long = 0,
    @SerialName("month_minor") val monthMinor: Long = 0,
    val authoritative: Boolean = false,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class SupplierPromotion(
    @SerialName("promotion_id") val promotionId: String,
    @SerialName("supplier_id") val supplierId: String = "",
    val name: String = "",
    val description: String? = null,
    @SerialName("discount_bps") val discountBps: Long = 0,
    @SerialName("scope_type") val scopeType: String = "ALL_PRODUCTS",
    @SerialName("scope_product_id") val scopeProductId: String? = null,
    @SerialName("retailer_scope") val retailerScope: String = "ALL",
    @SerialName("is_active") val isActive: Boolean = true,
    val priority: Long = 0,
)

@Serializable
data class SupplierPromotionsResponse(
    val promotions: List<SupplierPromotion> = emptyList(),
)

@Serializable
data class SupplierPromotionUpsertRequest(
    val name: String,
    val description: String = "",
    @SerialName("discount_bps") val discountBps: Long,
    @SerialName("scope_type") val scopeType: String = "ALL_PRODUCTS",
    @SerialName("scope_product_id") val scopeProductId: String? = null,
    @SerialName("retailer_scope") val retailerScope: String = "ALL",
    @SerialName("retailer_ids") val retailerIds: List<String> = emptyList(),
    @SerialName("min_line_quantity") val minLineQuantity: Long = 0,
    @SerialName("min_order_amount_minor") val minOrderAmountMinor: Long = 0,
    val priority: Long = 0,
)

@Serializable
data class BillingSetupRequest(
    @SerialName("bank_name") val bankName: String,
    @SerialName("account_holder") val accountHolder: String,
    @SerialName("account_number") val accountNumber: String,
    @SerialName("swift_bic") val swiftBic: String,
    val iban: String? = null,
    @SerialName("selected_gateways") val selectedGateways: List<String>,
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

@Serializable
data class BillingSetupResponse(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("is_configured") val isConfigured: Boolean,
    @SerialName("selected_gateways") val selectedGateways: List<String> = emptyList(),
)

@Serializable
data class SupplierReturnRow(
    @SerialName("return_id") val returnId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("sku_id") val skuId: String,
    @SerialName("product_name") val productName: String,
    val quantity: Long = 0,
    @SerialName("unit_price") val unitPrice: Long = 0,
    val status: String = "",
    @SerialName("physical_status") val physicalStatus: String = "",
    @SerialName("received_qty") val receivedQty: Long = 0,
    val reason: String = "",
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class SupplierReturnsResponse(
    val data: List<SupplierReturnRow> = emptyList(),
)

@Serializable
data class ResolveReturnRequest(
    @SerialName("return_id") val returnId: String,
    @SerialName("line_item_id") val lineItemId: String,
    val resolution: String,
    val notes: String = "",
)
