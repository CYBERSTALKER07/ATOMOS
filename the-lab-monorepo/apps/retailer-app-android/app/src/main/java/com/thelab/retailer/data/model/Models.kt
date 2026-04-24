package com.thelab.retailer.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonNames

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

// ── Order Status (matches backend State field) ──

@Serializable
enum class OrderStatus {
    @SerialName("PENDING") PENDING,
    @SerialName("PENDING_REVIEW") PENDING_REVIEW,
    @SerialName("SCHEDULED") SCHEDULED,
    @SerialName("AUTO_ACCEPTED") AUTO_ACCEPTED,
    @SerialName("LOADED") LOADED,
    @SerialName("DISPATCHED") DISPATCHED,
    @SerialName("IN_TRANSIT") IN_TRANSIT,
    @SerialName("ARRIVING") ARRIVING,
    @SerialName("ARRIVED") ARRIVED,
    @SerialName("ARRIVED_SHOP_CLOSED") ARRIVED_SHOP_CLOSED,
    @SerialName("AWAITING_GLOBAL_PAYNT") AWAITING_GLOBAL_PAYNT,
    @SerialName("PENDING_CASH_COLLECTION") PENDING_CASH_COLLECTION,
    @SerialName("CANCEL_REQUESTED") CANCEL_REQUESTED,
    @SerialName("NO_CAPACITY") NO_CAPACITY,
    @SerialName("COMPLETED") COMPLETED,
    @SerialName("CANCELLED") CANCELLED,
    @SerialName("QUARANTINE") QUARANTINE,
    @SerialName("DELIVERED_ON_CREDIT") DELIVERED_ON_CREDIT;

    /** Retailer-friendly label (not the raw backend state name). */
    val displayName: String
        get() = when (this) {
            PENDING -> "Order Placed"
            PENDING_REVIEW -> "Under Review"
            SCHEDULED -> "Scheduled"
            AUTO_ACCEPTED -> "Auto-Accepted"
            LOADED -> "Approved"
            DISPATCHED -> "Dispatched"
            IN_TRANSIT -> "Active"
            ARRIVING -> "Driver Nearby"
            ARRIVED -> "Driver Arrived"
            ARRIVED_SHOP_CLOSED -> "Shop Closed"
            AWAITING_GLOBAL_PAYNT -> "Payment Required"
            PENDING_CASH_COLLECTION -> "Cash Collection"
            CANCEL_REQUESTED -> "Cancel Requested"
            NO_CAPACITY -> "No Capacity"
            COMPLETED -> "Delivered"
            CANCELLED -> "Cancelled"
            QUARANTINE -> "On Hold"
            DELIVERED_ON_CREDIT -> "Delivered (Credit)"
        }

    val isActive: Boolean
        get() = this in listOf(AUTO_ACCEPTED, LOADED, DISPATCHED, IN_TRANSIT, ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED, AWAITING_GLOBAL_PAYNT, PENDING_CASH_COLLECTION)

    /** 6-step logistics pipeline for the timeline. */
    val progressFraction: Float
        get() = when (this) {
            PENDING, PENDING_REVIEW, SCHEDULED -> 0.17f
            AUTO_ACCEPTED -> 0.25f
            LOADED -> 0.33f
            DISPATCHED -> 0.50f
            IN_TRANSIT -> 0.67f
            ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED -> 0.83f
            AWAITING_GLOBAL_PAYNT -> 0.83f
            PENDING_CASH_COLLECTION -> 0.83f
            COMPLETED, DELIVERED_ON_CREDIT -> 1.0f
            CANCELLED, CANCEL_REQUESTED, NO_CAPACITY, QUARANTINE -> 0f
        }

    val ringLabel: String
        get() = when (this) {
            PENDING, PENDING_REVIEW, SCHEDULED -> "1/6"
            AUTO_ACCEPTED -> "1/6"
            LOADED -> "2/6"
            DISPATCHED -> "3/6"
            IN_TRANSIT -> "4/6"
            ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED -> "5/6"
            AWAITING_GLOBAL_PAYNT -> "Pay"
            PENDING_CASH_COLLECTION -> "Cash"
            COMPLETED, DELIVERED_ON_CREDIT -> "Done"
            CANCELLED, CANCEL_REQUESTED, NO_CAPACITY, QUARANTINE -> "X"
        }

    val canCancel: Boolean
        get() = this == PENDING || this == PENDING_REVIEW || this == SCHEDULED || this == AUTO_ACCEPTED

    /** QR code is available after payload seal (DISPATCHED) when JIT token is generated. */
    val hasDeliveryToken: Boolean
        get() = this in listOf(DISPATCHED, IN_TRANSIT, ARRIVED)

    /** Ordered list of steps for the timeline UI. */
    val timelineStepIndex: Int
        get() = when (this) {
            PENDING, PENDING_REVIEW, SCHEDULED, AUTO_ACCEPTED -> 0
            LOADED -> 1
            DISPATCHED -> 2
            IN_TRANSIT -> 3
            ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED, AWAITING_GLOBAL_PAYNT, PENDING_CASH_COLLECTION -> 4
            COMPLETED, DELIVERED_ON_CREDIT -> 5
            CANCELLED, CANCEL_REQUESTED, NO_CAPACITY, QUARANTINE -> -1
        }

    companion object {
        /** Ordered timeline steps with retailer-friendly labels. */
        val timelineSteps = listOf(
            "Placed" to PENDING,
            "Approved" to LOADED,
            "Dispatched" to DISPATCHED,
            "Active" to IN_TRANSIT,
            "Arrived" to ARRIVED,
            "Delivered" to COMPLETED,
        )
    }
}

// ── Variant (iOS: Variant) ──

@Serializable
data class Variant(
    @SerialName("id") val id: String,
    @SerialName("size") val size: String,
    @SerialName("pack") val pack: String,
    @SerialName("pack_count") val packCount: Int,
    @SerialName("weight_per_unit") val weightPerUnit: String,
    @SerialName("price") val price: Double,
)

// ── Product (iOS: Product) ──

@Serializable
data class Product(
    @SerialName("id") val id: String,
    @SerialName("name") val name: String,
    @SerialName("description") val description: String = "",
    @SerialName("nutrition") val nutrition: String = "",
    @JsonNames("image_url", "imageURL") val imageUrl: String? = null,
    @SerialName("variants") val variants: List<Variant> = emptyList(),
    @JsonNames("supplier_id", "supplierId") val supplierId: String? = null,
    @JsonNames("supplier_name", "supplierName") val supplierName: String? = null,
    @SerialName("supplier_category") val supplierCategory: String? = null,
    @JsonNames("category_id", "categoryId") val categoryId: String? = null,
    @JsonNames("category_name", "categoryName") val categoryName: String? = null,
    @JsonNames("sell_by_block", "sellByBlock") val sellByBlock: Boolean = false,
    @JsonNames("units_per_block", "unitsPerBlock") val unitsPerBlock: Int? = null,
    @JsonNames("price", "price") val price: Int? = null,
    @JsonNames("available_stock", "availableStock") val availableStock: Int? = null,
) {
    val isOutOfStock: Boolean get() = availableStock != null && availableStock <= 0
    val isLowStock: Boolean get() = availableStock != null && availableStock in 1..5
    val defaultVariant: Variant? get() = variants.firstOrNull()
    val displayPrice: String
        get() = defaultVariant?.let { "%,.0f".format(it.price) }
            ?: price?.let { "%,d".format(it) }
            ?: "—"

    val merchandisingLabel: String?
        get() = categoryName ?: when {
            sellByBlock && unitsPerBlock != null -> "$unitsPerBlock units / block"
            else -> null
        }

    companion object {
        val samples = listOf(
            Product(
                id = "prod-001", name = "Organic Whole Milk", description = "Farm-fresh organic whole milk",
                variants = listOf(
                    Variant("v-001a", "1L", "Single", 1, "1000ml", 3.49),
                    Variant("v-001b", "2L", "Twin Pack", 2, "1000ml", 6.49),
                ),
            ),
            Product(
                id = "prod-002", name = "Sourdough Bread", description = "Artisan sourdough loaf, slow-fermented",
                variants = listOf(
                    Variant("v-002a", "400g", "Single", 1, "400g", 4.99),
                    Variant("v-002b", "800g", "Large", 1, "800g", 8.49),
                ),
            ),
            Product(
                id = "prod-003", name = "Free-Range Eggs", description = "12ct large free-range eggs",
                variants = listOf(
                    Variant("v-003a", "12 ct", "Single", 1, "720g", 5.99),
                    Variant("v-003b", "30 ct", "Tray", 1, "1800g", 12.99),
                ),
            ),
            Product(
                id = "prod-004", name = "Greek Yogurt", description = "Thick, strained full-fat Greek yogurt",
                variants = listOf(
                    Variant("v-004a", "500g", "Single", 1, "500g", 4.29),
                    Variant("v-004b", "1kg", "Family", 1, "1000g", 7.99),
                ),
            ),
            Product(
                id = "prod-005", name = "Sparkling Water", description = "Natural mineral sparkling water",
                variants = listOf(
                    Variant("v-005a", "330ml", "Single", 1, "330ml", 1.49),
                    Variant("v-005b", "500ml", "6-Pack", 6, "500ml", 7.99),
                    Variant("v-005c", "1.5L", "4-Pack", 4, "1500ml", 9.99),
                ),
            ),
            Product(
                id = "prod-006", name = "Aged Cheddar", description = "12-month aged sharp cheddar",
                variants = listOf(
                    Variant("v-006a", "200g", "Block", 1, "200g", 5.49),
                    Variant("v-006b", "500g", "Block", 1, "500g", 11.99),
                ),
            ),
            Product(
                id = "prod-007", name = "Fresh Chicken Breast", description = "Boneless, skinless chicken breast",
                variants = listOf(
                    Variant("v-007a", "500g", "Single", 1, "500g", 7.99),
                    Variant("v-007b", "1kg", "Value Pack", 1, "1000g", 14.49),
                ),
            ),
            Product(
                id = "prod-008", name = "Organic Bananas", description = "Fair-trade organic bananas",
                variants = listOf(
                    Variant("v-008a", "1kg", "Bunch", 1, "1000g", 2.99),
                ),
            ),
            Product(
                id = "prod-009", name = "Extra Virgin Olive Oil", description = "Cold-pressed extra virgin",
                variants = listOf(
                    Variant("v-009a", "500ml", "Bottle", 1, "500ml", 9.99),
                    Variant("v-009b", "1L", "Bottle", 1, "1000ml", 17.49),
                ),
            ),
            Product(
                id = "prod-010", name = "Dark Chocolate 85%", description = "Single-origin dark chocolate bar",
                variants = listOf(
                    Variant("v-010a", "100g", "Bar", 1, "100g", 3.99),
                    Variant("v-010b", "100g", "3-Pack", 3, "100g", 10.49),
                ),
            ),
        )
    }
}

// ── Category (iOS: ProductCategory) ──

@Serializable
data class ProductCategory(
    @SerialName("id") val id: String,
    @SerialName("name") val name: String,
    @SerialName("icon") val icon: String,
    @JsonNames("product_count", "productCount") val productCount: Int? = null,
    @JsonNames("supplier_count", "supplierCount") val supplierCount: Int? = null,
) {
    companion object {
        val samples = listOf(
            ProductCategory("cat-dairy", "Dairy & Eggs", "🥛", 12),
            ProductCategory("cat-bakery", "Bakery", "🍞", 8),
            ProductCategory("cat-produce", "Fresh Produce", "🥬", 24),
            ProductCategory("cat-meat", "Meat & Poultry", "🥩", 15),
            ProductCategory("cat-beverages", "Beverages", "🧃", 18),
            ProductCategory("cat-snacks", "Snacks & Confectionery", "🍫", 20),
            ProductCategory("cat-frozen", "Frozen Foods", "🧊", 10),
            ProductCategory("cat-condiments", "Condiments & Sauces", "🫙", 14),
        )
    }
}

// ── Supplier (iOS: Supplier) ──

@Serializable
data class Supplier(
    @SerialName("id") val id: String,
    @SerialName("name") val name: String,
    @JsonNames("logo_url", "logoURL") val logoUrl: String? = null,
    @SerialName("category") val category: String? = null,
    @JsonNames("order_count", "orderCount") val orderCount: Int = 0,
    @JsonNames("product_count", "productCount") val productCount: Int = 0,
    @JsonNames("last_order_date", "lastOrderDate") val lastOrderDate: String? = null,
    @SerialName("phone") val phone: String? = null,
    @SerialName("email") val email: String? = null,
    @SerialName("address") val address: String? = null,
    @SerialName("primary_category_id") val primaryCategoryId: String? = null,
    @SerialName("operating_category_ids") val operatingCategoryIds: List<String> = emptyList(),
    @SerialName("operating_category_names") val operatingCategoryNames: List<String> = emptyList(),
    @JsonNames("is_active", "isActive") val isActive: Boolean = true,
    @JsonNames("manual_off_shift", "manualOffShift") val manualOffShift: Boolean = false,
) {
    val initials: String
        get() {
            val words = name.split(" ")
            return if (words.size >= 2) "${words[0].first()}${words[1].first()}"
            else name.take(2).uppercase()
        }

    val displayCategory: String?
        get() {
            if (!category.isNullOrBlank()) return category
            val categories = operatingCategoryNames.filter { it.isNotBlank() }
            val first = categories.firstOrNull() ?: return null
            return when (categories.size) {
                1 -> first
                2 -> categories.joinToString(" • ")
                else -> "$first +${categories.size - 1} more"
            }
        }

    val catalogSubtitle: String
        get() = if (productCount > 0) "$productCount products" else "$orderCount orders"
}

// ── Order Line Item ──

@Serializable
data class OrderLineItem(
    @SerialName("id") val id: String,
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String,
    @SerialName("variant_id") val variantId: String = "",
    @SerialName("variant_size") val variantSize: String = "",
    @SerialName("quantity") val quantity: Int,
    @SerialName("unit_price") val unitPrice: Double = 0.0,
    @SerialName("total_price") val totalPrice: Double = 0.0,
)

// ── Order ──

@Serializable
data class Order(
    @SerialName("order_id") val id: String,
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("supplier_name") val supplierName: String = "",
    @SerialName("state") val status: OrderStatus = OrderStatus.PENDING,
    @SerialName("items") val items: List<OrderLineItem> = emptyList(),
    @SerialName("amount") val totalAmount: Double = 0.0,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
    @SerialName("estimated_delivery") val estimatedDelivery: String? = null,
    @SerialName("delivery_token") val qrCode: String? = null,
    @SerialName("order_source") val orderSource: String = "MANUAL",
) {
    val displayTotal: String get() = String.format("$%.2f", totalAmount)
    val itemCount: Int get() = items.sumOf { it.quantity }
    val isAiGenerated: Boolean get() = orderSource == "AI_PREDICTED"
}

// ── AI Demand Forecast (iOS: DemandForecast) ──

@Serializable
data class DemandForecast(
    @SerialName("id") val id: String,
    @SerialName("product_id") val productId: String = "",
    @SerialName("product_name") val productName: String = "Predicted Order",
    @SerialName("predicted_quantity") val predictedQuantity: Int = 1,
    @SerialName("confidence") val confidence: Double = 0.85,
    @SerialName("reasoning") val reasoning: String = "",
    @SerialName("suggested_order_date") val suggestedOrderDate: String = "",
) {
    val confidencePercent: String get() = "${(confidence * 100).toInt()}%"
}

// ── Retailer Expense Analytics ──

@Serializable
data class MonthlyExpense(
    @SerialName("month") val month: String,
    @SerialName("total") val total: Long,
) {
    val shortMonth: String
        get() {
            val parts = month.split("-")
            if (parts.size < 2) return month
            val monthNames = listOf("Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec")
            val idx = parts[1].toIntOrNull()?.minus(1) ?: return month
            return monthNames.getOrElse(idx) { month }
        }
}

@Serializable
data class TopSupplierExpense(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("supplier_name") val supplierName: String,
    @SerialName("total") val total: Long,
    @SerialName("order_count") val orderCount: Int,
)

@Serializable
data class TopProductExpense(
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String,
    @SerialName("total") val total: Long,
    @SerialName("quantity") val quantity: Int,
)

@Serializable
data class RetailerAnalytics(
    @SerialName("monthly_expenses") val monthlyExpenses: List<MonthlyExpense> = emptyList(),
    @SerialName("top_suppliers") val topSuppliers: List<TopSupplierExpense> = emptyList(),
    @SerialName("top_products") val topProducts: List<TopProductExpense> = emptyList(),
    @SerialName("total_this_month") val totalThisMonth: Long = 0,
    @SerialName("total_last_month") val totalLastMonth: Long = 0,
)

// ── Detailed Retailer Analytics (Advanced) ──

@Serializable
data class RetailerDayExpense(
    @SerialName("date") val date: String,
    @SerialName("total") val total: Long,
    @SerialName("count") val count: Long,
)

@Serializable
data class OrderStateCount(
    @SerialName("state") val state: String,
    @SerialName("count") val count: Long,
)

@Serializable
data class CategorySpend(
    @SerialName("category") val category: String,
    @SerialName("total") val total: Long,
    @SerialName("count") val count: Long,
)

@Serializable
data class DayOfWeekPattern(
    @SerialName("weekday") val weekday: String,
    @SerialName("avg") val avg: Long,
    @SerialName("count") val count: Long,
)

@Serializable
data class RetailerDetailedAnalytics(
    @SerialName("daily_spending") val dailySpending: List<RetailerDayExpense> = emptyList(),
    @SerialName("orders_by_state") val ordersByState: List<OrderStateCount> = emptyList(),
    @SerialName("category_breakdown") val categoryBreakdown: List<CategorySpend> = emptyList(),
    @SerialName("weekday_pattern") val weekdayPattern: List<DayOfWeekPattern> = emptyList(),
    @SerialName("total_spent") val totalSpent: Long = 0,
    @SerialName("total_orders") val totalOrders: Long = 0,
    @SerialName("avg_order_value") val avgOrderValue: Long = 0,
)

// ── Cart Item (local only, not serialized over API) ──

data class CartItem(
    val id: String,       // product.id + variant.id
    val product: Product,
    val variant: Variant,
    var quantity: Int,
) {
    val totalPrice: Double get() = quantity * variant.price
}

// ── User ──

@Serializable
data class User(
    @SerialName("id") val id: String,
    @SerialName("name") val name: String,
    @SerialName("company") val company: String = "",
    @SerialName("email") val email: String = "",
    @SerialName("avatar_url") val avatarUrl: String? = null,
)

// ── Auth Request / Response ──

@Serializable
data class LoginRequest(
    @SerialName("phone_number") val phoneNumber: String,
    @SerialName("password") val password: String,
)

@Serializable
data class RegisterRequest(
    @SerialName("phone_number") val phoneNumber: String,
    @SerialName("password") val password: String,
    @SerialName("store_name") val storeName: String,
    @SerialName("owner_name") val ownerName: String,
    @SerialName("address_text") val addressText: String,
    @SerialName("latitude") val latitude: Double,
    @SerialName("longitude") val longitude: Double,
    @SerialName("tax_id") val taxId: String? = null,
    @SerialName("receiving_window_open") val receivingWindowOpen: String? = null,
    @SerialName("receiving_window_close") val receivingWindowClose: String? = null,
    @SerialName("access_type") val accessType: String? = null,
    @SerialName("storage_ceiling_height_cm") val storageCeilingHeightCM: Double? = null,
)

@Serializable
data class AuthResponse(
    @SerialName("token") val token: String,
    @SerialName("user") val user: User,
    @SerialName("firebase_token") val firebaseToken: String = "",
)

// ── Empathy Settings ──

@Serializable
data class EmpathySettings(
    @SerialName("global_auto_order_enabled") val globalAutoOrderEnabled: Boolean = false,
    @SerialName("supplier_settings") val supplierSettings: Map<String, Boolean> = emptyMap(),
    @SerialName("product_settings") val productSettings: Map<String, Boolean> = emptyMap(),
)

@Serializable
data class UpdateSettingsRequest(
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("use_history") val useHistory: Boolean? = null,
)

@Serializable
data class UpdateGlobalSettingsRequest(
    @SerialName("global_auto_order_enabled") val globalAutoOrderEnabled: Boolean,
    @SerialName("use_history") val useHistory: Boolean = true,
)

// ── Auto-Order Full Settings Response ──

@Serializable
data class AutoOrderSettings(
    @SerialName("global_enabled") val globalEnabled: Boolean = false,
    @SerialName("analytics_start_date") val analyticsStartDate: String? = null,
    @SerialName("has_any_history") val hasAnyHistory: Boolean = false,
    @SerialName("supplier_overrides") val supplierOverrides: List<SupplierOverride> = emptyList(),
    @SerialName("category_overrides") val categoryOverrides: List<CategoryOverride> = emptyList(),
    @SerialName("product_overrides") val productOverrides: List<ProductOverride> = emptyList(),
    @SerialName("variant_overrides") val variantOverrides: List<VariantOverride> = emptyList(),
)

@Serializable
data class SupplierOverride(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("has_history") val hasHistory: Boolean = false,
    @SerialName("analytics_start_date") val analyticsStartDate: String? = null,
    @SerialName("supplier_name") val supplierName: String? = null,
)

@Serializable
data class CategoryOverride(
    @SerialName("category_id") val categoryId: String,
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("has_history") val hasHistory: Boolean = false,
    @SerialName("analytics_start_date") val analyticsStartDate: String? = null,
)

@Serializable
data class ProductOverride(
    @SerialName("product_id") val productId: String,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("has_history") val hasHistory: Boolean = false,
    @SerialName("analytics_start_date") val analyticsStartDate: String? = null,
    @SerialName("product_name") val productName: String? = null,
)

@Serializable
data class VariantOverride(
    @SerialName("sku_id") val skuId: String,
    @SerialName("product_id") val productId: String = "",
    @SerialName("enabled") val enabled: Boolean,
    @SerialName("has_history") val hasHistory: Boolean = false,
    @SerialName("analytics_start_date") val analyticsStartDate: String? = null,
    @SerialName("sku_label") val skuLabel: String? = null,
)

@Serializable
data class ApiResponse(
    @SerialName("status") val status: String,
    @SerialName("message") val message: String? = null,
)

@Serializable
data class SupplierOrderResult(
    @SerialName("order_id") val orderId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("supplier_name") val supplierName: String = "",
    @SerialName("total") val total: Long = 0,
    @SerialName("item_count") val itemCount: Int = 0,
)

@Serializable
data class UnifiedCheckoutResponse(
    @SerialName("status") val status: String,
    @SerialName("invoice_id") val invoiceId: String,
    @SerialName("total") val total: Long = 0,
    @SerialName("supplier_orders") val supplierOrders: List<SupplierOrderResult> = emptyList(),
)

@Serializable
data class CheckoutLineItem(
    @SerialName("sku_id") val skuId: String,
    @SerialName("quantity") val quantity: Int,
    @SerialName("unit_price") val unitPrice: Long,
)

@Serializable
data class UnifiedCheckoutRequest(
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("payment_gateway") val paymentGateway: String,
    @SerialName("latitude") val latitude: Double = 0.0,
    @SerialName("longitude") val longitude: Double = 0.0,
    @SerialName("items") val items: List<CheckoutLineItem>,
)

// ── Delivery Tracking (real-time driver positions) ──

@Serializable
data class TrackingOrderItem(
    @SerialName("product_id") val productId: String,
    @SerialName("product_name") val productName: String,
    val quantity: Long,
    @SerialName("unit_price") val unitPrice: Long,
    @SerialName("line_total") val lineTotal: Long,
)

@Serializable
data class TrackingOrder(
    @SerialName("order_id") val orderId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("supplier_name") val supplierName: String,
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("warehouse_name") val warehouseName: String = "",
    @SerialName("driver_id") val driverId: String = "",
    val state: String,
    @SerialName("total_amount") val totalAmount: Long,
    @SerialName("order_source") val orderSource: String = "",
    @SerialName("driver_latitude") val driverLatitude: Double? = null,
    @SerialName("driver_longitude") val driverLongitude: Double? = null,
    @SerialName("is_approaching") val isApproaching: Boolean = false,
    @SerialName("delivery_token") val deliveryToken: String = "",
    @SerialName("created_at") val createdAt: String = "",
    val items: List<TrackingOrderItem> = emptyList(),
)

@Serializable
data class TrackingResponse(
    val orders: List<TrackingOrder> = emptyList(),
)
