package com.pegasusx.retailer.data.model

import java.text.NumberFormat
import java.util.Locale
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
    @SerialName("AWAITING_PAYMENT") AWAITING_PAYMENT,
    @SerialName("PENDING_CASH_COLLECTION") PENDING_CASH_COLLECTION,
    @SerialName("FISCALIZING") FISCALIZING,
    @SerialName("FISCAL_FAILED") FISCAL_FAILED,
    @SerialName("CANCEL_REQUESTED") CANCEL_REQUESTED,
    @SerialName("NO_CAPACITY") NO_CAPACITY,
    @SerialName("COMPLETED") COMPLETED,
    @SerialName("CANCELLED") CANCELLED,
    @SerialName("QUARANTINE") QUARANTINE,
    @SerialName("DELIVERED_ON_CREDIT") DELIVERED_ON_CREDIT,
    @SerialName("RECONCILIATION_REQUIRED") RECONCILIATION_REQUIRED,
    @SerialName("DELAYED") DELAYED;

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
            AWAITING_PAYMENT -> "Payment Required"
            PENDING_CASH_COLLECTION -> "Cash Collection"
            FISCALIZING -> "Pending Fiscal"
            FISCAL_FAILED -> "Fiscal Failed"
            CANCEL_REQUESTED -> "Cancel Requested"
            NO_CAPACITY -> "No Capacity"
            COMPLETED -> "Delivered"
            CANCELLED -> "Cancelled"
            QUARANTINE -> "On Hold"
            DELIVERED_ON_CREDIT -> "Delivered (Credit)"
            RECONCILIATION_REQUIRED -> "Reconciliation Required"
            DELAYED -> "Delayed"
        }

    val isActive: Boolean
        get() = this in listOf(
            AUTO_ACCEPTED, LOADED, DISPATCHED, IN_TRANSIT, ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED,
            AWAITING_PAYMENT, PENDING_CASH_COLLECTION, FISCALIZING, FISCAL_FAILED,
        )

    /** 6-step logistics pipeline for the timeline. */
    val progressFraction: Float
        get() = when (this) {
            PENDING, PENDING_REVIEW, SCHEDULED -> 0.17f
            AUTO_ACCEPTED -> 0.25f
            LOADED -> 0.33f
            DISPATCHED -> 0.50f
            IN_TRANSIT -> 0.67f
            ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED -> 0.83f
            AWAITING_PAYMENT -> 0.83f
            PENDING_CASH_COLLECTION -> 0.83f
            FISCALIZING, FISCAL_FAILED -> 0.92f
            COMPLETED, DELIVERED_ON_CREDIT -> 1.0f
            CANCELLED, CANCEL_REQUESTED, NO_CAPACITY, QUARANTINE, RECONCILIATION_REQUIRED, DELAYED -> 0f
        }

    val ringLabel: String
        get() = when (this) {
            PENDING, PENDING_REVIEW, SCHEDULED -> "1/6"
            AUTO_ACCEPTED -> "1/6"
            LOADED -> "2/6"
            DISPATCHED -> "3/6"
            IN_TRANSIT -> "4/6"
            ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED -> "5/6"
            AWAITING_PAYMENT -> "Pay"
            PENDING_CASH_COLLECTION -> "Cash"
            FISCALIZING -> "Fiscal"
            FISCAL_FAILED -> "Fiscal!"
            COMPLETED, DELIVERED_ON_CREDIT -> "Done"
            CANCELLED, CANCEL_REQUESTED, NO_CAPACITY, QUARANTINE, RECONCILIATION_REQUIRED, DELAYED -> "X"
        }

    val canCancel: Boolean
        get() = this == PENDING || this == PENDING_REVIEW || this == SCHEDULED || this == AUTO_ACCEPTED

    /** QR code is available after payload seal (DISPATCHED) when JIT token is generated. */
    val hasDeliveryToken: Boolean
        get() = this in listOf(DISPATCHED, IN_TRANSIT, ARRIVING, ARRIVED)

    /** Ordered list of steps for the timeline UI. */
    val timelineStepIndex: Int
        get() = when (this) {
            PENDING, PENDING_REVIEW, SCHEDULED, AUTO_ACCEPTED -> 0
            LOADED -> 1
            DISPATCHED -> 2
            IN_TRANSIT -> 3
            ARRIVING, ARRIVED, ARRIVED_SHOP_CLOSED, AWAITING_PAYMENT, PENDING_CASH_COLLECTION, FISCALIZING, FISCAL_FAILED -> 4
            COMPLETED, DELIVERED_ON_CREDIT -> 5
            CANCELLED, CANCEL_REQUESTED, NO_CAPACITY, QUARANTINE, RECONCILIATION_REQUIRED, DELAYED -> -1
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
data class ProductOffer(
    @SerialName("product_id") val productId: String = "",
    @SerialName("list_price_minor") val listPriceMinor: Long = 0,
    @SerialName("sale_price_minor") val salePriceMinor: Long? = null,
    @SerialName("discount_bps") val discountBps: Long? = null,
    @SerialName("promotion_id") val promotionId: String? = null,
    @SerialName("promotion_name") val promotionName: String? = null,
    @SerialName("promotion_label") val promotionLabel: String? = null,
    @SerialName("promotion_ends_at") val promotionEndsAt: String? = null,
)

@Serializable
data class Product(
    @JsonNames("product_id", "id") val id: String,
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
    @JsonNames("is_out_of_stock", "isOutOfStock") val isOutOfStockFlag: Boolean = false,
    @JsonNames("accepts_backorder", "acceptsBackorder") val acceptsBackorder: Boolean = false,
    @JsonNames("show_stock_counts", "showStockCounts") val showStockCounts: Boolean = false,
    @JsonNames("max_quantity", "maxQuantity") val maxQuantity: Int? = null,
    @JsonNames("price_minor") val priceMinor: Long? = null,
    @SerialName("offer") val offer: ProductOffer? = null,
) {
    val isOutOfStock: Boolean
        get() = (availableStock != null && availableStock <= 0) || isOutOfStockFlag
    val isLowStock: Boolean get() = availableStock != null && availableStock in 1..5
    val blocksAddToCart: Boolean get() = isOutOfStock && !acceptsBackorder
    val cartMaxQuantity: Int?
        get() = when {
            maxQuantity != null && maxQuantity > 0 -> maxQuantity
            showStockCounts && availableStock != null && availableStock > 0 && !acceptsBackorder -> availableStock
            availableStock != null && availableStock > 0 && !acceptsBackorder -> availableStock
            else -> null
        }
    val defaultVariant: Variant? get() = variants.firstOrNull()
    val hasSaleOffer: Boolean
        get() = offer?.salePriceMinor?.let { it > 0 } == true

    val displayListPrice: String?
        get() {
            val listMinor = offer?.listPriceMinor?.takeIf { it > 0 }
                ?: priceMinor?.takeIf { it > 0 }
                ?: defaultVariant?.price?.toLong()?.takeIf { it > 0 }
                ?: price?.toLong()?.takeIf { it > 0 }
            return listMinor?.let { "%,d".format(it) }
        }

    val displayPrice: String
        get() {
            offer?.salePriceMinor?.takeIf { it > 0 }?.let { return "%,d".format(it) }
            defaultVariant?.let { return "%,.0f".format(it.price) }
            price?.let { return "%,d".format(it) }
            priceMinor?.let { return "%,d".format(it) }
            return "—"
        }

    val promotionLabel: String?
        get() = offer?.promotionLabel

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
    @SerialName("line_item_id") @JsonNames("id") val id: String,
    @SerialName("sku_id") @JsonNames("product_id") val productId: String,
    @SerialName("sku_name") @JsonNames("product_name") val productName: String,
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
    @SerialName("amount") val totalAmount: Long = 0,
    @SerialName("currency") val currency: String = "",
    @SerialName("payment_gateway") val paymentGateway: String = "",
    @SerialName("payment_status") val paymentStatus: String? = null,
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("auto_confirm_at") val autoConfirmAt: String? = null,
    @SerialName("deliver_before") val deliverBefore: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
    @SerialName("estimated_delivery") val estimatedDelivery: String? = null,
    @SerialName("delivery_token") val qrCode: String? = null,
    @SerialName("order_source") val orderSource: String = "MANUAL",
    @SerialName("confirmation_status") val confirmationStatus: String? = null,
    @SerialName("delivery_priority") val deliveryPriority: String? = null,
    @SerialName("preorder_badge") val preorderBadge: String? = null,
    @SerialName("proposed_delivery_date") val proposedDeliveryDate: String? = null,
    @SerialName("delivery_proposal_reason") val deliveryProposalReason: String? = null,
    @SerialName("version") val version: Long = 0,
) {
    val displayTotal: String get() = formatRetailerAmount(totalAmount, currency)
    val itemCount: Int get() = items.sumOf { it.quantity }
    val isAiGenerated: Boolean get() = orderSource == "AI_PREDICTED"
    val needsAiConfirmation: Boolean get() = status == OrderStatus.PENDING_REVIEW
    val needsPreorderAction: Boolean get() =
        orderSource == "MANUAL_PREORDER" && status == OrderStatus.SCHEDULED &&
            (confirmationStatus.isNullOrBlank() || confirmationStatus == "DRAFT")
    val needsDeliveryProposalReview: Boolean get() =
        confirmationStatus == "PENDING_WAREHOUSE" || preorderBadge == "REVIEW_DELIVERY"
}

fun formatRetailerAmount(amount: Long, currency: String): String {
    val formatter = NumberFormat.getIntegerInstance(Locale.US)
    val formatted = formatter.format(amount).replace(',', ' ')
    val normalizedCurrency = currency.ifBlank { com.pegasus.design.packCurrency(com.pegasus.design.MarketPackStore.pack) }
    return "$formatted $normalizedCurrency"
}

@Serializable
data class ProcurementOrderResponse(
    @SerialName("status") val status: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("state") val state: OrderStatus = OrderStatus.PENDING,
    @SerialName("amount") val amount: Long = 0,
    @SerialName("total") val total: Long = 0,
    @SerialName("currency") val currency: String = "UZS",
    @SerialName("order_source") val orderSource: String = "PROCUREMENT",
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class ProcurementOrderRequest(
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("items") val items: List<ProcurementOrderItem>,
)

@Serializable
data class ProcurementOrderItem(
    @SerialName("product_id") val productId: String,
    @SerialName("quantity") val quantity: Int,
)

// ── AI Demand Forecast (legacy SKU-forecast shape; NOT returned by GET /v1/retailer/ai/predictions) ──

@Serializable
data class DemandForecast(
    @SerialName("id") val id: String,
    @SerialName("product_id") val productId: String = "",
    @SerialName("product_name") val productName: String = "Predicted Order",
    @SerialName("predicted_quantity") val predictedQuantity: Int = 1,
    @SerialName("confidence") val confidence: Double = 0.85,
    @SerialName("reasoning") val reasoning: String = "",
    @SerialName("suggested_order_date") val suggestedOrderDate: String = "",
    @SerialName("blocked") val blocked: Boolean = false,
    @SerialName("blocked_reason") val blockedReason: String? = null,
    @SerialName("label") val label: String? = null,
) {
    val confidencePercent: String get() = "${(confidence * 100).toInt()}%"
    val isBlocked: Boolean
        get() = blocked || label == "insufficient_history" || !blockedReason.isNullOrBlank()
}

/** Live GET /v1/retailer/ai/predictions item — pending AI preorder, not a SKU forecast. */
@Serializable
data class RetailerAILineItem(
    @SerialName("sku") val sku: String = "",
    @SerialName("name") val name: String = "",
    @SerialName("quantity") val quantity: Long = 0,
    @SerialName("unit_price_minor") val unitPriceMinor: Long = 0,
)

@Serializable
data class RetailerAIPrediction(
    @SerialName("order_id") val orderId: String,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("order_source") val orderSource: String = "",
    @SerialName("confirmation_status") val confirmationStatus: String = "",
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String = "",
    @SerialName("auto_confirm_at") val autoConfirmAt: String = "",
    @SerialName("total_minor") val totalMinor: Long = 0,
    @SerialName("currency") val currency: String = "",
    @SerialName("derived_from_order_id") val derivedFromOrderId: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    @SerialName("line_items") val lineItems: List<RetailerAILineItem> = emptyList(),
) {
    val title: String
        get() {
            val first = lineItems.firstOrNull()
            val label = first?.name?.takeIf { it.isNotBlank() } ?: first?.sku?.takeIf { it.isNotBlank() }
            return label ?: orderId
        }
    val quantity: Long get() = lineItems.sumOf { it.quantity }
    val statusLabel: String
        get() = confirmationStatus.ifBlank { "PENDING" }.replace('_', ' ')
    val deliveryLabel: String
        get() = requestedDeliveryDate.take(10).ifBlank { orderId }
    val formattedTotal: String
        get() {
            val units = totalMinor / 100.0
            return if (currency.isNotBlank()) {
                "${units.toLong()} $currency"
            } else {
                units.toLong().toString()
            }
        }
}

@Serializable
data class RetailerAIPredictionsResponse(
    @SerialName("items") val items: List<RetailerAIPrediction> = emptyList(),
)

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
    @SerialName("phone_number") val phoneNumber: String = "",
    @SerialName("password") val password: String = "",
    @SerialName("id_token") val idToken: String = "",
)

@Serializable
data class ResolvedLocationResponse(
    val address: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("place_id") val placeId: String? = null,
)

@Serializable
data class RegisterRequest(
    @SerialName("phone_number") val phoneNumber: String,
    @SerialName("password") val password: String,
    @SerialName("store_name") val storeName: String,
    @SerialName("owner_name") val ownerName: String,
    @SerialName("address_text") val addressText: String,
    @SerialName("delivery_address") val deliveryAddress: String? = null,
    @SerialName("place_id") val placeId: String? = null,
    @SerialName("latitude") val latitude: Double,
    @SerialName("longitude") val longitude: Double,
    @SerialName("tax_id") val taxId: String? = null,
    @SerialName("receiving_window_open") val receivingWindowOpen: String? = null,
    @SerialName("receiving_window_close") val receivingWindowClose: String? = null,
    @SerialName("access_type") val accessType: String? = null,
    @SerialName("storage_ceiling_height_cm") val storageCeilingHeightCM: Double? = null,
)

@Serializable
data class RetailerMembershipDTO(
    @SerialName("user_id") val userId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("retailer_role") val retailerRole: String = "",
    @SerialName("name") val name: String = "",
    @SerialName("phone") val phone: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class AuthResponse(
    @SerialName("token") val token: String = "",
    @SerialName("token_type") val tokenType: String = "full",
    @SerialName("user") val user: User? = null,
    @SerialName("firebase_token") val firebaseToken: String = "",
    @SerialName("is_configured") val isConfigured: Boolean? = null,
    @SerialName("memberships") val memberships: List<RetailerMembershipDTO> = emptyList(),
    @SerialName("expires_in_sec") val expiresInSec: Int = 420,
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("refresh_token") val refreshToken: String = "",
) {
    val isPendingOrgSelect: Boolean
        get() = tokenType.equals("pending_org_select", ignoreCase = true)
}

@Serializable
data class SelectOrgRequest(
    @SerialName("retailer_id") val retailerId: String,
)

@Serializable
data class MembershipsResponse(
    @SerialName("memberships") val memberships: List<RetailerMembershipDTO> = emptyList(),
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
    @SerialName("global_auto_order_enabled") val globalAutoOrderEnabled: Boolean? = null,
    @SerialName("global_enabled") val globalEnabled: Boolean? = null,
    @SerialName("execution_mode") val executionMode: String? = null,
    @SerialName("use_history") val useHistory: Boolean? = null,
)

@Serializable
data class AutoOrderShadowStats(
    @SerialName("proposal_count") val proposalCount: Long = 0,
    @SerialName("matched_orders") val matchedOrders: Long = 0,
    @SerialName("wape") val wape: Double = 0.0,
    @SerialName("unmodified_accept_rate") val unmodifiedAcceptRate: Double = 0.0,
    @SerialName("window_days") val windowDays: Int = 30,
)

@Serializable
data class AutoOrderSoakDecision(
    @SerialName("allowed") val allowed: Boolean = false,
    @SerialName("reasons") val reasons: List<String> = emptyList(),
    @SerialName("stats") val stats: AutoOrderShadowStats? = null,
    @SerialName("bypass_source") val bypassSource: String? = null,
)

@Serializable
data class AutoOrderSoakThresholds(
    @SerialName("min_proposals") val minProposals: Long = 20,
    @SerialName("max_wape") val maxWape: Double = 0.30,
    @SerialName("min_unmodified") val minUnmodified: Double = 0.80,
    @SerialName("gate_disabled") val gateDisabled: Boolean = false,
    @SerialName("bypass_source") val bypassSource: String? = null,
)

@Serializable
data class AutoOrderSoakGate(
    @SerialName("decision") val decision: AutoOrderSoakDecision? = null,
    @SerialName("thresholds") val thresholds: AutoOrderSoakThresholds? = null,
    @SerialName("place_flag_enabled") val placeFlagEnabled: Boolean? = null,
)

@Serializable
data class AutoOrderShadowProposal(
    @SerialName("proposal_id") val proposalId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("sku") val sku: String = "",
    @SerialName("supplier_id") val supplierId: String? = null,
    @SerialName("proposed_qty") val proposedQty: Long = 0,
    @SerialName("ip") val ip: Double = 0.0,
    @SerialName("reorder_point") val reorderPoint: Double = 0.0,
    @SerialName("order_up_to") val orderUpTo: Double = 0.0,
    @SerialName("confidence") val confidence: Double? = null,
    @SerialName("reason") val reason: String? = null,
    @SerialName("bucket_date") val bucketDate: String = "",
    @SerialName("status") val status: String = "",
)

@Serializable
data class AutoOrderShadowProposalsResponse(
    @SerialName("items") val items: List<AutoOrderShadowProposal> = emptyList(),
)

// ── Auto-Order Full Settings Response ──

@Serializable
data class AutoOrderSettings(
    @SerialName("global_enabled") val globalEnabled: Boolean = false,
    /** off | shadow | draft | place — place creates real supplier orders when server flag is on */
    @SerialName("execution_mode") val executionMode: String? = null,
    @SerialName("analytics_start_date") val analyticsStartDate: String? = null,
    @SerialName("has_any_history") val hasAnyHistory: Boolean = false,
    @SerialName("supplier_overrides") val supplierOverrides: List<SupplierOverride> = emptyList(),
    @SerialName("category_overrides") val categoryOverrides: List<CategoryOverride> = emptyList(),
    @SerialName("product_overrides") val productOverrides: List<ProductOverride> = emptyList(),
    @SerialName("variant_overrides") val variantOverrides: List<VariantOverride> = emptyList(),
    @SerialName("shadow_stats") val shadowStats: AutoOrderShadowStats? = null,
)

@Serializable
data class AutoOrderSkip(
    @SerialName("sku") val sku: String? = null,
    @SerialName("reason") val reason: String = "",
)

@Serializable
data class AutoOrderPlacedOrder(
    @SerialName("order_id") val orderId: String = "",
    @SerialName("supplier_id") val supplierId: String? = null,
    @SerialName("line_count") val lineCount: Int = 0,
    @SerialName("total_minor") val totalMinor: Long = 0,
    @SerialName("skus") val skus: List<String> = emptyList(),
)

@Serializable
data class AutoOrderRun(
    @SerialName("run_id") val runId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("started_at") val startedAt: String = "",
    @SerialName("finished_at") val finishedAt: String? = null,
    @SerialName("mode") val mode: String = "draft",
    @SerialName("draft_lines") val draftLines: Int = 0,
    @SerialName("placed_lines") val placedLines: Int = 0,
    @SerialName("placed_orders") val placedOrders: List<AutoOrderPlacedOrder> = emptyList(),
    @SerialName("skipped") val skipped: List<AutoOrderSkip> = emptyList(),
    @SerialName("status") val status: String = "",
    @SerialName("message") val message: String? = null,
    @SerialName("suggestions_seen") val suggestionsSeen: Int = 0,
    @SerialName("schedule_bucket") val scheduleBucket: String? = null,
    @SerialName("candidate_source") val candidateSource: String? = null,
)

@Serializable
data class AutoOrderRunsResponse(
    @SerialName("items") val items: List<AutoOrderRun> = emptyList(),
)

/** L3.4 retailer reorder suggestion with demand source chips. */
@Serializable
data class RetailerReorderSuggestion(
    @SerialName("sku") val sku: String = "",
    @SerialName("suggested_qty") val suggestedQty: Long = 0,
    @SerialName("adjusted_demand_per_day") val adjustedDemandPerDay: Double = 0.0,
    @SerialName("current_stock") val currentStock: Long = 0,
    @SerialName("in_flight_qty") val inFlightQty: Long = 0,
    @SerialName("safety_stock") val safetyStock: Double = 0.0,
    @SerialName("status") val status: String? = null,
    @SerialName("sources") val sources: List<String> = emptyList(),
    @SerialName("sell_through_velocity") val sellThroughVelocity: Double = 0.0,
    @SerialName("base_demand_per_day") val baseDemandPerDay: Double = 0.0,
)

@Serializable
data class RetailerReorderSuggestionsResponse(
    @SerialName("items") val items: List<RetailerReorderSuggestion> = emptyList(),
)

@Serializable
data class FamilyMigrateItem(
    @SerialName("member_id") val memberId: String = "",
    @SerialName("user_id") val userId: String = "",
    @SerialName("phone") val phone: String = "",
    @SerialName("name") val name: String = "",
    @SerialName("retailer_role") val retailerRole: String = "",
    @SerialName("temp_password") val tempPassword: String? = null,
)

@Serializable
data class FamilyMigrateSkipped(
    @SerialName("member_id") val memberId: String = "",
    @SerialName("phone") val phone: String? = null,
    @SerialName("reason") val reason: String = "",
)

@Serializable
data class FamilyMigrateResult(
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("migrated") val migrated: List<FamilyMigrateItem> = emptyList(),
    @SerialName("skipped") val skipped: List<FamilyMigrateSkipped> = emptyList(),
    @SerialName("family_remaining") val familyRemaining: Int = 0,
    @SerialName("family_writes") val familyWrites: String = "",
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
data class CheckoutQuoteLine(
    @SerialName("product_id") val productId: String,
    @SerialName("quantity") val quantity: Long,
    @SerialName("unit_price_minor") val unitPriceMinor: Long,
    @SerialName("currency") val currency: String = "",
)

@Serializable
data class CheckoutQuoteRequest(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("lines") val lines: List<CheckoutQuoteLine>,
)

@Serializable
data class CheckoutQuoteResponse(
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("subtotal_minor") val subtotalMinor: Long = 0,
    @SerialName("discount_minor") val discountMinor: Long = 0,
    @SerialName("total_minor") val totalMinor: Long = 0,
    @SerialName("currency") val currency: String = "UZS",
)

@Serializable
data class StockWarning(
    @SerialName("sku") val sku: String,
    @SerialName("requested") val requested: Long = 0,
    @SerialName("available") val available: Long = 0,
    @SerialName("backorder_qty") val backorderQty: Long = 0,
    @SerialName("accepts_backorder") val acceptsBackorder: Boolean = false,
)

@Serializable
data class CheckoutPreviewResponse(
    val ok: Boolean = false,
    val blocked: Boolean = false,
    val code: String? = null,
    val message: String? = null,
    @SerialName("rejected_skus") val rejectedSkus: List<String> = emptyList(),
    @SerialName("oos_items") val oosItems: List<String> = emptyList(),
    val shortfall: Map<String, Long> = emptyMap(),
    @SerialName("stock_warnings") val stockWarnings: List<StockWarning> = emptyList(),
    @SerialName("max_quantities") val maxQuantities: Map<String, Long> = emptyMap(),
    @SerialName("orderable_quantities") val orderableQuantities: Map<String, Long> = emptyMap(),
    @SerialName("line_errors") val lineErrors: Map<String, String> = emptyMap(),
    @SerialName("backordered_item_count") val backorderedItemCount: Int = 0,
    @SerialName("show_stock_counts") val showStockCounts: Boolean = false,
    @SerialName("preorder_min_lead_days") val preorderMinLeadDays: Long? = null,
    @SerialName("preorder_max_lead_days") val preorderMaxLeadDays: Long? = null,
    @SerialName("order_line_min_quantity") val orderLineMinQuantity: Long? = null,
    @SerialName("order_line_max_quantity") val orderLineMaxQuantity: Long? = null,
    @SerialName("delivery_fee_minor") val deliveryFeeMinor: Long = 0,
    @SerialName("delivery_distance_km") val deliveryDistanceKm: Double? = null,
    @SerialName("default_out_of_stock_policy") val defaultOutOfStockPolicy: String? = null,
    @SerialName("checkout_policy_token") val checkoutPolicyToken: String? = null,
    @SerialName("checkout_policy_expires_at") val checkoutPolicyExpiresAt: String? = null,
    @SerialName("order_acceptance_open") val orderAcceptanceOpen: Boolean? = null,
    @SerialName("order_acceptance_window_label") val orderAcceptanceWindowLabel: String? = null,
    @SerialName("next_order_acceptance_at") val nextOrderAcceptanceAt: String? = null,
)

@Serializable
data class UnifiedCheckoutResponse(
    @SerialName("status") val status: String,
    @SerialName("invoice_id") val invoiceId: String,
    @SerialName("total") val total: Long = 0,
    @SerialName("supplier_orders") val supplierOrders: List<SupplierOrderResult> = emptyList(),
    @SerialName("backordered_item_count") val backorderedItemCount: Int = 0,
    @SerialName("backorder_order_id") val backorderOrderId: String? = null,
    @SerialName("stock_warnings") val stockWarnings: List<StockWarning> = emptyList(),
)

@Serializable
data class CashCheckoutRequest(
    @SerialName("order_id") val orderId: String,
)

@Serializable
data class CashCheckoutResponse(
    @SerialName("order_id") val orderId: String,
    val state: String,
    val amount: Long,
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("retailer_id") val retailerId: String,
    val message: String,
)

@Serializable
data class ConfirmCashRequest(
    @SerialName("order_id") val orderId: String,
)

@Serializable
data class ConfirmCashResponse(
    val success: Boolean,
    @SerialName("order_id") val orderId: String,
    val state: String,
    val message: String,
)

@Serializable
data class CardCheckoutRequest(
    @SerialName("order_id") val orderId: String,
    val gateway: String,
)

@Serializable
data class CardCheckoutResponse(
    @SerialName("order_id") val orderId: String,
    val state: String,
    val amount: Long,
    val gateway: String,
    @SerialName("resolved_gateway") val resolvedGateway: String? = null,
    @SerialName("policy_source") val policySource: String? = null,
    @SerialName("allowed_gateways") val allowedGateways: List<String>? = null,
    @SerialName("policy_reason") val policyReason: String? = null,
    @SerialName("payment_url") val paymentUrl: String,
    @SerialName("invoice_id") val invoiceId: String,
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("attempt_id") val attemptId: String? = null,
    @SerialName("attempt_no") val attemptNo: Int? = null,
    @SerialName("retailer_id") val retailerId: String,
    val message: String,
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
    @SerialName("delivery_mode") val deliveryMode: String? = null,
    @SerialName("requested_delivery_date") val requestedDeliveryDate: String? = null,
    @SerialName("delivery_priority") val deliveryPriority: String? = null,
    @SerialName("checkout_policy_token") val checkoutPolicyToken: String? = null,
    @SerialName("currency") val currency: String? = null,
)

@Serializable
data class OrderCurrencyOptions(
    @SerialName("enabled") val enabled: Boolean = false,
    @SerialName("operating_currency") val operatingCurrency: String = "UZS",
    @SerialName("allowlist") val allowlist: List<String> = emptyList(),
)

@Serializable
data class PSPListing(
    val code: String = "",
    val status: String = "",
    val selectable: Boolean = true,
    @SerialName("national_cards") val nationalCards: Boolean = false,
)

@Serializable
data class RetailerPaymentCatalogResponse(
    @SerialName("currency_code") val currencyCode: String = "",
    @SerialName("market_code") val marketCode: String = "",
    val catalog: List<PSPListing> = emptyList(),
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
data class RouteGeometryWire(
    @SerialName("route_id") val routeId: String = "",
    @SerialName("encoded_polyline") val encodedPolyline: String = "",
    val coordinates: List<RouteLatLng> = emptyList(),
    val source: String = "",
    @SerialName("stop_count") val stopCount: Int = 0,
)

@Serializable
data class RouteLatLng(
    val lat: Double,
    val lng: Double,
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
    @SerialName("live_location_available") val liveLocationAvailable: Boolean = false,
    @SerialName("delivery_token") val deliveryToken: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("fiscal_status") val fiscalStatus: String = "",
    @SerialName("fiscal_qr") val fiscalQr: String = "",
    @SerialName("latest_fiscal_receipt_id") val latestFiscalReceiptId: String = "",
    val items: List<TrackingOrderItem> = emptyList(),
    @SerialName("route_geometry") val routeGeometry: RouteGeometryWire? = null,
) {
    /** Retailer receipt label for tracking / recent receipts. */
    val fiscalReceiptLabel: String
        get() {
            val st = state.uppercase()
            val fs = fiscalStatus.uppercase()
            return when {
                fs == "SUCCESS" || st == "COMPLETED" ->
                    if (latestFiscalReceiptId.isNotBlank()) "Fiscalized · $latestFiscalReceiptId"
                    else "Fiscalized"
                fs == "PENDING" || st == "FISCALIZING" -> "Pending fiscal"
                fs == "FAILED" || st == "FISCAL_FAILED" -> "Fiscal failed"
                fs == "FORCE_SKIPPED" -> "Fiscal exception"
                else -> st.ifBlank { "—" }
            }
        }
}

@Serializable
data class TrackingResponse(
    val status: String? = null,
    val orders: List<TrackingOrder> = emptyList(),
    @SerialName("recent_receipts") val recentReceipts: List<TrackingOrder> = emptyList(),
)

@Serializable
data class ActiveFulfillmentItem(
    @SerialName("order_id") val orderId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("supplier_name") val supplierName: String,
    val state: String,
    @SerialName("adjusted_amount") val adjustedAmount: Long,
    @SerialName("item_count") val itemCount: Int,
    @SerialName("live_location_available") val liveLocationAvailable: Boolean = false,
)

@Serializable
data class ActiveFulfillmentsResponse(
    val fulfillments: List<ActiveFulfillmentItem> = emptyList(),
    val count: Int = 0,
)

@Serializable
data class PendingPaymentSession(
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("gateway") val gateway: String? = null,
    @SerialName("locked_amount") val lockedAmount: Long,
    @SerialName("currency") val currency: String,
    @SerialName("status") val status: String,
    @SerialName("current_attempt_no") val currentAttemptNo: Int,
    @SerialName("invoice_id") val invoiceId: String? = null,
    @SerialName("redirect_url") val redirectUrl: String? = null,
    @SerialName("expires_at") val expiresAt: String? = null,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class PendingPaymentsResponse(
    @SerialName("pending_payments") val pendingPayments: List<PendingPaymentSession> = emptyList(),
    @SerialName("count") val count: Int = 0,
)

@Serializable
data class OrderTimelineEntry(
    @SerialName("transition_id") val transitionId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("previous_status") val previousStatus: String? = null,
    @SerialName("new_status") val newStatus: String,
    val reason: String? = null,
    @SerialName("actor_role") val actorRole: String? = null,
    @SerialName("actor_id") val actorId: String? = null,
    @SerialName("event_kind") val eventKind: String? = null,
    @SerialName("created_at") val createdAt: String,
)

@Serializable
data class OrderTimelineResponse(
    @SerialName("order_id") val orderId: String,
    val items: List<OrderTimelineEntry> = emptyList(),
)

@Serializable
data class CreditProfile(
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("credit_limit_minor") val creditLimitMinor: Long = 0,
    @SerialName("current_balance_minor") val currentBalanceMinor: Long = 0,
    @SerialName("available_credit_minor") val availableCreditMinor: Long = 0,
    @SerialName("risk_score") val riskScore: Long = 0,
    @SerialName("risk_tier") val riskTier: String = "",
    @SerialName("delinquency_count") val delinquencyCount: Long = 0,
    val status: String = "",
    val version: Long = 0,
)

@Serializable
data class RetailerSupplierOrderFacet(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("orders_by_status") val ordersByStatus: Map<String, Int> = emptyMap(),
)

@Serializable
data class RetailerPulseLoyalty(
    val enrolled: Boolean = false,
)

@Serializable
data class ControlTowerPulse(
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("generated_at") val generatedAt: String = "",
    @SerialName("open_orders") val openOrders: Int = 0,
    @SerialName("active_fulfillments") val activeFulfillments: Int = 0,
    @SerialName("dock_pending") val dockPending: Int = 0,
    @SerialName("pos_open_sessions") val posOpenSessions: Int = 0,
    @SerialName("open_shifts") val openShifts: Int = 0,
    @SerialName("open_assist_tickets") val openAssistTickets: Int = 0,
    @SerialName("low_stock_sku_bins") val lowStockSkuBins: Int = 0,
    @SerialName("shift_variances_7d") val shiftVariances7d: Int = 0,
    @SerialName("sales_minor_7d") val salesMinor7d: Long = 0,
    val capabilities: List<String> = emptyList(),
    val empty: Boolean = true,
    val source: String = "empty",
    @SerialName("orders_by_status") val ordersByStatus: Map<String, Int> = emptyMap(),
    @SerialName("orders_by_supplier") val ordersBySupplier: List<RetailerSupplierOrderFacet> = emptyList(),
    val loyalty: RetailerPulseLoyalty = RetailerPulseLoyalty(),
)

@Serializable
data class LoyaltyTierView(
    val enrolled: Boolean = false,
    val tier: String = "",
    @SerialName("lifetime_points") val lifetimePoints: Long = 0,
    @SerialName("available_points") val availablePoints: Long = 0,
    @SerialName("next_tier") val nextTier: String = "",
    @SerialName("points_to_next") val pointsToNext: Long = 0,
    @SerialName("earn_bps") val earnBps: Long = 0,
)

@Serializable
data class LoyaltyLedgerEntry(
    @SerialName("ledger_id") val ledgerId: String = "",
    @SerialName("order_id") val orderId: String = "",
    val points: Long = 0,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class LoyaltyLedgerResponse(
    val entries: List<LoyaltyLedgerEntry> = emptyList(),
)
