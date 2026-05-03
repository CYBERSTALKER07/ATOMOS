package com.pegasus.warehouse.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// ── Auth ──
@Serializable
data class LoginRequest(
    val phone: String,
    val pin: String,
)

@Serializable
data class AuthResponse(
    val token: String,
    @SerialName("refresh_token") val refreshToken: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    val role: String = "",
    val name: String = "",
)

// ── Dashboard ──
@Serializable
data class DashboardData(
    @SerialName("active_orders") val activeOrders: Int = 0,
    @SerialName("completed_today") val completedToday: Int = 0,
    @SerialName("pending_dispatch") val pendingDispatch: Int = 0,
    @SerialName("today_revenue") val todayRevenue: Long = 0,
    @SerialName("drivers_on_route") val driversOnRoute: Int = 0,
    @SerialName("idle_drivers") val idleDrivers: Int = 0,
    @SerialName("vehicles") val vehicles: Int = 0,
    @SerialName("low_stock_items") val lowStockItems: Int = 0,
    @SerialName("total_staff") val totalStaff: Int = 0,
    @SerialName("fleet_status") val fleetStatus: Map<String, Int> = emptyMap(),
)

// ── Orders ──
@Serializable
data class Order(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_name") val retailerName: String = "",
    val state: String = "",
    @SerialName("total_uzs") val totalUzs: Long = 0,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("line_items") val lineItems: List<LineItem> = emptyList(),
)

@Serializable
data class LineItem(
    @SerialName("product_name") val productName: String = "",
    val quantity: Int = 0,
    @SerialName("unit_price") val unitPrice: Long = 0,
)

@Serializable
data class OrderListResponse(
    val orders: List<Order> = emptyList(),
)

// ── Drivers ──
@Serializable
data class Driver(
    @SerialName("driver_id") val driverId: String,
    val name: String = "",
    val phone: String = "",
    @SerialName("truck_status") val truckStatus: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class CreateDriverRequest(
    val name: String,
    val phone: String,
)

@Serializable
data class CreateDriverResponse(
    @SerialName("driver_id") val driverId: String = "",
    val pin: String = "",
)

@Serializable
data class DriverListResponse(
    val drivers: List<Driver> = emptyList(),
)

// ── Vehicles ──
@Serializable
data class Vehicle(
    @SerialName("vehicle_id") val vehicleId: String,
    val label: String = "",
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("vehicle_class") val vehicleClass: String = "",
    @SerialName("capacity_vu") val capacityVu: Int = 0,
    val status: String = "",
)

@Serializable
data class CreateVehicleRequest(
    val label: String,
    @SerialName("license_plate") val licensePlate: String,
    @SerialName("vehicle_class") val vehicleClass: String,
)

@Serializable
data class VehicleListResponse(
    val vehicles: List<Vehicle> = emptyList(),
)

// ── Inventory ──
@Serializable
data class InventoryItem(
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String = "",
    val quantity: Int = 0,
    @SerialName("reorder_threshold") val reorderThreshold: Int = 0,
    val sku: String = "",
)

@Serializable
data class InventoryListResponse(
    val items: List<InventoryItem> = emptyList(),
)

@Serializable
data class InventoryAdjustRequest(
    @SerialName("product_id") val productId: String,
    val quantity: Int,
)

// ── Products ──
@Serializable
data class Product(
    @SerialName("product_id") val productId: String,
    val name: String = "",
    @SerialName("sku_id") val skuId: String = "",
    val category: String = "",
    @SerialName("price_uzs") val priceUzs: Long = 0,
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class ProductListResponse(
    val products: List<Product> = emptyList(),
)

// ── Manifests ──
@Serializable
data class Manifest(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("vehicle_label") val vehicleLabel: String = "",
    @SerialName("license_plate") val licensePlate: String = "",
    @SerialName("stop_count") val stopCount: Int = 0,
    val status: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class ManifestListResponse(
    val manifests: List<Manifest> = emptyList(),
)

// ── Analytics ──
@Serializable
data class AnalyticsData(
    val period: String = "",
    @SerialName("total_revenue") val totalRevenue: Long = 0,
    @SerialName("total_orders") val totalOrders: Int = 0,
    @SerialName("avg_order_value") val avgOrderValue: Long = 0,
    @SerialName("fleet_utilization_pct") val fleetUtilizationPct: Double = 0.0,
    @SerialName("top_products") val topProducts: List<TopProduct> = emptyList(),
    val daily: List<DailyMetric> = emptyList(),
)

@Serializable
data class TopProduct(
    @SerialName("product_name") val productName: String = "",
    @SerialName("total_sold") val totalSold: Int = 0,
    val revenue: Long = 0,
)

@Serializable
data class DailyMetric(
    val date: String = "",
    val revenue: Long = 0,
    val orders: Int = 0,
)

// ── CRM ──
@Serializable
data class Retailer(
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("business_name") val businessName: String = "",
    @SerialName("total_orders") val totalOrders: Int = 0,
    @SerialName("total_revenue") val totalRevenue: Long = 0,
    @SerialName("last_order_date") val lastOrderDate: String = "",
)

@Serializable
data class RetailerListResponse(
    val retailers: List<Retailer> = emptyList(),
)

// ── Returns ──
@Serializable
data class ReturnItem(
    @SerialName("line_item_id") val lineItemId: String,
    @SerialName("order_id") val orderId: String = "",
    @SerialName("product_name") val productName: String = "",
    val quantity: Int = 0,
    val status: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class ReturnListResponse(
    val items: List<ReturnItem> = emptyList(),
)

// ── Treasury ──
@Serializable
data class TreasuryOverview(
    @SerialName("total_invoiced") val totalInvoiced: Long = 0,
    @SerialName("total_paid") val totalPaid: Long = 0,
    @SerialName("total_outstanding") val totalOutstanding: Long = 0,
)

@Serializable
data class Invoice(
    @SerialName("invoice_id") val invoiceId: String,
    @SerialName("retailer_name") val retailerName: String = "",
    @SerialName("amount_uzs") val amountUzs: Long = 0,
    val status: String = "",
    @SerialName("due_date") val dueDate: String = "",
)

@Serializable
data class InvoiceListResponse(
    val invoices: List<Invoice> = emptyList(),
)

// ── Dispatch Preview ──
@Serializable
data class DispatchPreview(
    @SerialName("undispatched_orders") val undispatchedOrders: List<DispatchOrder> = emptyList(),
    @SerialName("available_drivers") val availableDrivers: List<AvailableDriver> = emptyList(),
)

@Serializable
data class DispatchOrder(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_name") val retailerName: String = "",
    @SerialName("total_uzs") val totalUzs: Long = 0,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("item_count") val itemCount: Int = 0,
)

@Serializable
data class AvailableDriver(
    @SerialName("driver_id") val driverId: String,
    val name: String = "",
    val phone: String = "",
    @SerialName("vehicle_label") val vehicleLabel: String = "",
    @SerialName("truck_status") val truckStatus: String = "",
)

// ── Warehouse Realtime ──
@Serializable
data class WarehouseSupplyRequest(
    @SerialName("request_id") val requestId: String,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    val state: String = "",
    val priority: String = "",
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String? = null,
    @SerialName("total_volume_vu") val totalVolumeVu: Double = 0.0,
    val notes: String = "",
    @SerialName("transfer_order_id") val transferOrderId: String? = null,
    @SerialName("created_by") val createdBy: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class WarehouseDispatchLock(
    @SerialName("lock_id") val lockId: String,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("factory_id") val factoryId: String = "",
    @SerialName("lock_type") val lockType: String = "",
    @SerialName("locked_at") val lockedAt: String = "",
    @SerialName("unlocked_at") val unlockedAt: String? = null,
    @SerialName("locked_by") val lockedBy: String = "",
)

@Serializable
data class WarehouseLiveEvent(
    val type: String,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("request_id") val requestId: String? = null,
    val state: String? = null,
    @SerialName("lock_id") val lockId: String? = null,
    val action: String? = null,
    val timestamp: String? = null,
)

// ── Staff ──
@Serializable
data class StaffMember(
    @SerialName("worker_id") val workerId: String,
    val name: String = "",
    val phone: String = "",
    val role: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class StaffListResponse(
    val staff: List<StaffMember> = emptyList(),
)

@Serializable
data class CreateStaffRequest(
    val name: String,
    val phone: String,
    val role: String = "WAREHOUSE_STAFF",
)

@Serializable
data class CreateStaffResponse(
    @SerialName("worker_id") val workerId: String = "",
    val pin: String = "",
)

// ── Payment Config ──
@Serializable
data class PaymentGateway(
    @SerialName("gateway_name") val gatewayName: String,
    val provider: String = "",
    @SerialName("is_active") val isActive: Boolean = false,
    val mode: String = "",
)

@Serializable
data class PaymentConfigResponse(
    val gateways: List<PaymentGateway> = emptyList(),
)
