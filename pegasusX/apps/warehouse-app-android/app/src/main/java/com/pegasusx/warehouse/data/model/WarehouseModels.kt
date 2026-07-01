package com.pegasusx.warehouse.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

// ── Auth ──
@Serializable
data class LoginRequest(
    val phone: String = "",
    val pin: String = "",
    @SerialName("id_token") val idToken: String = "",
)

@Serializable
data class AuthResponse(
    val token: String,
    @SerialName("refresh_token") val refreshToken: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    val role: String = "",
    val name: String = "",
    @SerialName("is_configured") val isConfigured: Boolean = false,
)

@Serializable
data class RefreshTokenRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

// ── Dashboard ──
@Serializable
data class FleetStatusEntry(
    val status: String = "",
    val count: Long = 0,
)

@Serializable
data class DashboardData(
    @SerialName("active_orders") val activeOrders: Long = 0,
    @SerialName("completed_today") val completedToday: Long = 0,
    @SerialName("pending_dispatch") val pendingDispatch: Long = 0,
    @SerialName("drivers_on_route") val driversOnRoute: Long = 0,
    @SerialName("drivers_idle") val driversIdle: Long = 0,
    @SerialName("total_drivers") val totalDrivers: Long = 0,
    @SerialName("total_vehicles") val totalVehicles: Long = 0,
    @SerialName("today_revenue") val todayRevenue: Long = 0,
    @SerialName("low_stock_count") val lowStockCount: Long = 0,
    @SerialName("total_staff") val totalStaff: Long = 0,
    @SerialName("fleet_status") val fleetStatus: List<FleetStatusEntry> = emptyList(),
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
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("vehicle_class") val vehicleClass: String = "",
    @SerialName("vehicle_is_active") val vehicleIsActive: Boolean = false,
    @SerialName("vehicle_unavailable_reason") val vehicleUnavailableReason: String? = null,
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

@Serializable
data class AssignVehicleRequest(
    @SerialName("vehicle_id") val vehicleId: String? = null,
)

@Serializable
data class AssignVehicleResponse(
    val status: String = "",
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("previously_assigned_driver") val previouslyAssignedDriver: String? = null,
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
    @SerialName("is_active") val isActive: Boolean = true,
    @SerialName("unavailable_reason") val unavailableReason: String? = null,
    @SerialName("unavailable_note") val unavailableNote: String? = null,
    @SerialName("assigned_driver_id") val assignedDriverId: String? = null,
    @SerialName("assigned_driver_name") val assignedDriverName: String = "",
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

@Serializable
data class UpdateVehicleRequest(
    @SerialName("is_active") val isActive: Boolean? = null,
    @SerialName("unavailable_reason") val unavailableReason: String? = null,
    @SerialName("unavailable_note") val unavailableNote: String? = null,
)

@Serializable
data class VehicleDetailResponse(
    val vehicle: Vehicle = Vehicle(vehicleId = ""),
)

@Serializable
data class VehicleMutationResponse(
    val status: String = "",
    @SerialName("vehicle_id") val vehicleId: String = "",
    @SerialName("unavailable_reason") val unavailableReason: String? = null,
)

// ── Inventory ──
@Serializable
data class InventoryItem(
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String = "",
    val quantity: Int = 0,
    @SerialName("reorder_threshold") val reorderThreshold: Int = 0,
    val sku: String = "",
    @SerialName("out_of_stock_policy") val outOfStockPolicy: String? = null,
    @SerialName("effective_policy") val effectivePolicy: String? = null,
)

@Serializable
data class InventoryPolicyPatchRequest(
    @SerialName("out_of_stock_policy") val outOfStockPolicy: String,
)

@Serializable
data class DeliveryFeeTier(
    @SerialName("max_km") val maxKm: Double? = null,
    @SerialName("fee_minor") val feeMinor: Long = 0,
)

@Serializable
data class DeliveryFeeRules(
    val currency: String = "UZS",
    @SerialName("base_fee_minor") val baseFeeMinor: Long = 0,
    val tiers: List<DeliveryFeeTier> = emptyList(),
)

@Serializable
data class WarehouseOpsSettingsResponse(
    @SerialName("warehouse_id") val warehouseId: String = "",
    val name: String = "",
    @SerialName("default_out_of_stock_policy") val defaultOutOfStockPolicy: String = "REJECT",
    @SerialName("show_stock_counts_to_retailers") val showStockCountsToRetailers: Boolean = false,
    @SerialName("operating_schedule") val operatingSchedule: kotlinx.serialization.json.JsonElement? = null,
    @SerialName("ops_always_available") val opsAlwaysAvailable: Boolean = true,
    @SerialName("express_enabled") val expressEnabled: Boolean = false,
    @SerialName("express_stock_floor") val expressStockFloor: Long = 0,
    @SerialName("preorder_min_lead_days") val preorderMinLeadDays: Long = 3,
    @SerialName("preorder_max_lead_days") val preorderMaxLeadDays: Long = 90,
    @SerialName("order_line_min_quantity") val orderLineMinQuantity: Long? = null,
    @SerialName("order_line_max_quantity") val orderLineMaxQuantity: Long? = null,
    @SerialName("delivery_fee_rules") val deliveryFeeRules: DeliveryFeeRules? = null,
)

@Serializable
data class WarehouseOpsSettingsPatchRequest(
    @SerialName("default_out_of_stock_policy") val defaultOutOfStockPolicy: String,
    @SerialName("show_stock_counts_to_retailers") val showStockCountsToRetailers: Boolean? = null,
    @SerialName("operating_schedule") val operatingSchedule: kotlinx.serialization.json.JsonElement,
    @SerialName("express_enabled") val expressEnabled: Boolean? = null,
    @SerialName("express_stock_floor") val expressStockFloor: Long? = null,
    @SerialName("preorder_min_lead_days") val preorderMinLeadDays: Long? = null,
    @SerialName("preorder_max_lead_days") val preorderMaxLeadDays: Long? = null,
    @SerialName("order_line_min_quantity") val orderLineMinQuantity: Long? = null,
    @SerialName("order_line_max_quantity") val orderLineMaxQuantity: Long? = null,
    @SerialName("clear_order_line_min_quantity") val clearOrderLineMinQuantity: Boolean? = null,
    @SerialName("clear_order_line_max_quantity") val clearOrderLineMaxQuantity: Boolean? = null,
    @SerialName("delivery_fee_rules") val deliveryFeeRules: DeliveryFeeRules? = null,
    @SerialName("clear_delivery_fee_rules") val clearDeliveryFeeRules: Boolean? = null,
)

@Serializable
data class WarehousePreorderRow(
    @SerialName("order_id") val orderId: String,
    val status: String = "",
    @SerialName("order_source") val orderSource: String? = null,
    @SerialName("confirmation_status") val confirmationStatus: String? = null,
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String? = null,
    @SerialName("proposed_delivery_date") val proposedDeliveryDate: String? = null,
    @SerialName("delivery_proposal_reason") val deliveryProposalReason: String? = null,
    @SerialName("preorder_badge") val preorderBadge: String? = null,
)

@Serializable
data class WarehouseProposeDeliveryRequest(
    @SerialName("proposed_delivery_date") val proposedDeliveryDate: String,
    val reason: String = "",
)

@Serializable
data class WarehousePreorderEditRequest(
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String? = null,
    val reason: String = "",
)

@Serializable
data class WarehousePreordersResponse(
    val preorders: List<WarehousePreorderRow> = emptyList(),
    val items: List<WarehousePreorderRow> = emptyList(),
)

@Serializable
data class StockCommitmentRow(
    @SerialName("sku_id") val skuId: String,
    val name: String? = null,
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("on_hand") val onHand: Long = 0,
    @SerialName("available_qty") val availableQty: Long = 0,
    @SerialName("reserved_asap") val reservedAsap: Long = 0,
    @SerialName("reserved_scheduled") val reservedScheduled: Long = 0,
    @SerialName("deficit_qty") val deficitQty: Long = 0,
)

@Serializable
data class StockCommitmentsResponse(
    val items: List<StockCommitmentRow> = emptyList(),
    val skus: List<StockCommitmentRow> = emptyList(),
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
    @SerialName("total_orders") val totalOrders: Long = 0,
    @SerialName("completed_orders") val completedOrders: Long = 0,
    @SerialName("cancelled_orders") val cancelledOrders: Long = 0,
    @SerialName("avg_order_value") val avgOrderValue: Double = 0.0,
    @SerialName("fleet_utilization") val fleetUtilization: FleetUtilization = FleetUtilization(),
    @SerialName("fleet_utilization_pct") val fleetUtilizationPctLegacy: Double = 0.0,
    @SerialName("import_freshness") val importFreshness: ImportFreshness = ImportFreshness(),
    @SerialName("import_anomaly_queue") val importAnomalyQueue: ImportAnomalyQueue = ImportAnomalyQueue(),
    @SerialName("top_products") val topProducts: List<TopProduct> = emptyList(),
    @SerialName("daily_breakdown") val dailyBreakdown: List<DailyMetric> = emptyList(),
    val daily: List<DailyMetric> = emptyList(),
) {
    val fleetUtilizationPct: Double
        get() = if (fleetUtilization.utilizationPct > 0.0) {
            fleetUtilization.utilizationPct
        } else {
            fleetUtilizationPctLegacy
        }

    val chartDaily: List<DailyMetric>
        get() = if (dailyBreakdown.isNotEmpty()) dailyBreakdown else daily
}

@Serializable
data class FleetUtilization(
    @SerialName("total_drivers") val totalDrivers: Long = 0,
    @SerialName("active_drivers") val activeDrivers: Long = 0,
    @SerialName("utilization_pct") val utilizationPct: Double = 0.0,
    @SerialName("avg_stops_per_day") val avgStopsPerDay: Double = 0.0,
)

@Serializable
data class ImportFreshness(
    @SerialName("applied_rows_30d") val appliedRows30d: Long = 0,
    @SerialName("applied_skus_30d") val appliedSkus30d: Long = 0,
    @SerialName("quantity_delta_30d") val quantityDelta30d: Long = 0,
    @SerialName("last_session_id") val lastSessionId: String = "",
    @SerialName("last_applied_at") val lastAppliedAt: String = "",
)

@Serializable
data class ImportAnomalyQueue(
    @SerialName("open_rows_30d") val openRows30d: Long = 0,
    @SerialName("affected_sessions_30d") val affectedSessions30d: Long = 0,
    @SerialName("last_session_id") val lastSessionId: String = "",
    @SerialName("last_detected_at") val lastDetectedAt: String = "",
    @SerialName("last_detail") val lastDetail: String = "",
)

@Serializable
data class TopProduct(
    @SerialName("product_name") val productName: String = "",
    @SerialName("total_sold") val totalSold: Long = 0,
    @SerialName("total_qty") val totalQty: Long = 0,
    val revenue: Long = 0,
) {
    val displayUnits: Long
        get() = if (totalQty > 0) totalQty else totalSold
}

@Serializable
data class DailyMetric(
    val date: String = "",
    val revenue: Long = 0,
    val orders: Long = 0,
    @SerialName("completed") val completed: Long = 0,
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

@Serializable
data class InboundReturnRow(
    @SerialName("return_id") val returnId: String,
    @SerialName("order_id") val orderId: String = "",
    @SerialName("product_name") val productName: String = "",
    @SerialName("expected_qty") val expectedQty: Long = 0,
    @SerialName("received_qty") val receivedQty: Long = 0,
    val reason: String = "",
    @SerialName("physical_status") val physicalStatus: String = "",
    @SerialName("driver_name") val driverName: String = "",
    val barcode: String = "",
)

@Serializable
data class InboundReturnListResponse(
    val data: List<InboundReturnRow> = emptyList(),
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
    @SerialName("amount") val amount: Long = 0,
    @SerialName("amount_uzs") val amountUzs: Long = 0,
    val currency: String = "UZS",
    val status: String = "",
    @SerialName("fee_amount") val feeAmount: Long = 0,
    @SerialName("net_payout_amount") val netPayoutAmount: Long = 0,
    @SerialName("payout_owner_type") val payoutOwnerType: String = "",
    @SerialName("payout_owner_id") val payoutOwnerId: String = "",
    @SerialName("fee_policy_version") val feePolicyVersion: String = "",
    @SerialName("settlement_target") val settlementTarget: String = "",
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
    @SerialName("unavailable_drivers") val unavailableDrivers: List<AvailableDriver> = emptyList(),
    @SerialName("proposed_routes") val proposedRoutes: List<DispatchProposedRoute> = emptyList(),
    @SerialName("optimizer_source") val optimizerSource: String? = null,
    @SerialName("optimizer_warnings") val optimizerWarnings: List<String> = emptyList(),
    @SerialName("window_constrained_count") val windowConstrainedCount: Int = 0,
    @SerialName("fleet_effective_capacity_vu") val fleetEffectiveCapacityVu: Double = 0.0,
    @SerialName("plan_fingerprint") val planFingerprint: String? = null,
)

// mirror of backend-go/dispatch/plan.RoutesToWire preview payload
@Serializable
data class DispatchProposedStop(
    @SerialName("order_id") val orderId: String = "",
    @SerialName("retailer_id") val retailerId: String? = null,
    @SerialName("retailer_name") val retailerName: String? = null,
    val lat: Double? = null,
    val lng: Double? = null,
    @SerialName("volume_vu") val volumeVu: Double? = null,
)

@Serializable
data class DispatchProposedRoute(
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("driver_name") val driverName: String? = null,
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("order_ids") val orderIds: List<String> = emptyList(),
    val stops: List<DispatchProposedStop> = emptyList(),
    @SerialName("volume_vu") val volumeVu: Double? = null,
    @SerialName("loaded_volume") val loadedVolume: Double? = null,
    @SerialName("max_volume_vu") val maxVolumeVu: Double? = null,
    @SerialName("stop_count") val stopCount: Int? = null,
    @SerialName("route_geometry") val routeGeometry: RouteGeometryWire? = null,
)

@Serializable
data class DispatchOrder(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_name") val retailerName: String = "",
    @SerialName("total_uzs") val totalUzs: Long = 0,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("item_count") val itemCount: Int = 0,
    @SerialName("volume_vu") val volumeVu: Double = 0.0,
)

@Serializable
data class AvailableDriver(
    @SerialName("driver_id") val driverId: String,
    val name: String = "",
    val phone: String = "",
    @SerialName("vehicle_label") val vehicleLabel: String = "",
    @SerialName("truck_status") val truckStatus: String = "",
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("used_volume_vu") val usedVolumeVu: Double? = null,
    @SerialName("free_volume_vu") val freeVolumeVu: Double? = null,
    @SerialName("active_manifest_id") val activeManifestId: String? = null,
    @SerialName("unavailable_reason") val unavailableReason: String? = null,
)

@Serializable
data class DispatchCapacityWarning(
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("loaded_vu") val loadedVu: Double = 0.0,
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("effective_max_vu") val effectiveMaxVu: Double = 0.0,
    @SerialName("excess_vu") val excessVu: Double = 0.0,
    @SerialName("suggested_unselect_order_ids") val suggestedUnselectOrderIds: List<String> = emptyList(),
    @SerialName("suggested_defer_order_ids") val suggestedDeferOrderIds: List<String> = emptyList(),
)

@Serializable
data class DispatchExecuteResponse(
    val status: String = "",
    @SerialName("orders_assigned") val ordersAssigned: Int = 0,
    val warnings: List<String> = emptyList(),
    @SerialName("capacity_warnings") val capacityWarnings: List<DispatchCapacityWarning> = emptyList(),
    @SerialName("orphan_order_ids") val orphanOrderIds: List<String> = emptyList(),
)

// ── Warehouse Realtime ──
@Serializable
data class SupplyRequestListResponse(
    val requests: List<WarehouseSupplyRequest> = emptyList(),
    @SerialName("supply_requests") val supplyRequests: List<WarehouseSupplyRequest> = emptyList(),
) {
    fun resolved(): List<WarehouseSupplyRequest> =
        requests.ifEmpty { supplyRequests }
}

@Serializable
data class DispatchLockListResponse(
    val locks: List<WarehouseDispatchLock> = emptyList(),
)

@Serializable
data class DemandForecastDay(
    val date: String = "",
    @SerialName("projected_units") val projectedUnits: Long = 0,
    @SerialName("projected_revenue") val projectedRevenue: Long = 0,
    @SerialName("committed_units") val committedUnits: Long = 0,
    @SerialName("pending_confirmation_units") val pendingConfirmationUnits: Long = 0,
    val currency: String = "",
)

@Serializable
data class DemandForecastSources(
    @SerialName("incoming_orders") val incomingOrders: Long = 0,
    @SerialName("ai_prediction") val aiPrediction: Long = 0,
    @SerialName("pre_orders") val preOrders: Long = 0,
    @SerialName("burn_rate") val burnRate: Double = 0.0,
)

@Serializable
data class DemandForecastProduct(
    @SerialName("product_id") val productId: String = "",
    @SerialName("product_name") val productName: String = "",
    @SerialName("current_stock") val currentStock: Long = 0,
    @SerialName("recommended_qty") val recommendedQty: Long = 0,
    @SerialName("days_until_stockout") val daysUntilStockout: Double = 0.0,
    val priority: String = "",
    val unit: String = "",
    val sources: DemandForecastSources = DemandForecastSources(),
    @SerialName("demand_breakdown") val demandBreakdown: JsonObject? = null,
)

@Serializable
data class DemandForecastResponse(
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("forecast_days") val forecastDays: Int = 7,
    @SerialName("generated_at") val generatedAt: String? = null,
    val series: List<DemandForecastDay> = emptyList(),
    val products: List<DemandForecastProduct> = emptyList(),
)

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

@Serializable
data class CreateWarehouseSupplyRequestItem(
    @SerialName("product_id") val productId: String,
    @SerialName("requested_quantity") val requestedQuantity: Int,
    @SerialName("recommended_qty") val recommendedQty: Int,
    @SerialName("unit_volume_vu") val unitVolumeVu: Double,
)

@Serializable
data class CreateWarehouseSupplyRequestRequest(
    @SerialName("factory_id") val factoryId: String,
    val priority: String,
    val notes: String,
    val items: List<CreateWarehouseSupplyRequestItem> = emptyList(),
    @SerialName("use_demand_forecast") val useDemandForecast: Boolean = true,
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String? = null,
)

@Serializable
data class CreateWarehouseSupplyRequestResponse(
    @SerialName("request_id") val requestId: String,
    val state: String = "",
    val priority: String = "",
    @SerialName("total_volume_vu") val totalVolumeVu: Double = 0.0,
    @SerialName("items_count") val itemsCount: Int = 0,
)

@Serializable
data class WarehouseSupplyRequestTransitionRequest(
    val action: String,
    @SerialName("transfer_order_id") val transferOrderId: String? = null,
)

@Serializable
data class WarehouseSupplyRequestTransitionResponse(
    @SerialName("request_id") val requestId: String,
    val state: String = "",
)

@Serializable
data class CreateWarehouseDispatchLockRequest(
    @SerialName("lock_type") val lockType: String,
)

@Serializable
data class CreateWarehouseDispatchLockResponse(
    @SerialName("lock_id") val lockId: String,
    @SerialName("lock_type") val lockType: String = "",
    val status: String = "",
)

@Serializable
data class ReleaseWarehouseDispatchLockResponse(
    @SerialName("lock_id") val lockId: String,
    val status: String = "",
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
