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
data class SupplierEarnings(
    val currency: String = "",
    @SerialName("today_minor") val todayMinor: Long = 0,
    @SerialName("week_minor") val weekMinor: Long = 0,
    @SerialName("month_minor") val monthMinor: Long = 0,
    val authoritative: Boolean = false,
    @SerialName("updated_at") val updatedAt: String? = null,
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

@Serializable
data class BillingSetupResponse(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("is_configured") val isConfigured: Boolean,
    @SerialName("selected_gateways") val selectedGateways: List<String> = emptyList(),
)
