package com.pegasusx.retailer.ui.screens.cart

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.CartItem
import com.pegasusx.retailer.data.model.CheckoutLineItem
import com.pegasusx.retailer.data.model.CheckoutQuoteLine
import com.pegasusx.retailer.data.model.CheckoutQuoteRequest
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.StockWarning
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.data.model.Variant
import com.pegasusx.retailer.ui.components.CheckoutPhase
import com.pegasusx.retailer.ui.components.CheckoutPaymentOption
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.longOrNull
import kotlinx.serialization.json.JsonObject
import retrofit2.HttpException
import javax.inject.Inject

enum class CartLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class CartUiState(
    val items: List<CartItem> = emptyList(),
    val showCheckout: Boolean = false,
    val checkoutPhase: CheckoutPhase = CheckoutPhase.REVIEW,
    val checkoutError: String? = null,
    val selectedPaymentGateway: String = "GLOBAL_PAY",
    val lastOrderId: String? = null,
    val removedItemMessage: String? = null,
    val supplierIsActive: Boolean = true,
    val oosItems: List<String> = emptyList(),
    val stockWarnings: List<StockWarning> = emptyList(),
    val previewMaxQuantities: Map<String, Int> = emptyMap(),
    val previewShowStockCounts: Boolean = false,
    val previewShortfall: Map<String, Long> = emptyMap(),
    val previewLoading: Boolean = false,
    val deliveryMode: String = "STANDARD",
    val deliveryDate: String? = null,
    val expressPriority: Boolean = false,
    val preorderMinLeadDays: Long = 3,
    val preorderMaxLeadDays: Long = 0,
    val deliveryFeeMinor: Long = 0,
    val deliveryDistanceKm: Double = 0.0,
    val orderLineMinQuantity: Long? = null,
    val orderLineMaxQuantity: Long? = null,
    val pendingBackorderConfirm: Boolean = false,
    val paymentOptions: List<CheckoutPaymentOption> = emptyList(),
    val isRefreshing: Boolean = false,
    val syncError: String? = null,
    val loadIssue: CartLoadIssue? = null,
    val paymentUrlToOpen: String? = null,
    val quotedDiscountMinor: Long? = null,
    val quotedSubtotalMinor: Long? = null,
) {
    val isEmpty: Boolean get() = items.isEmpty()
    val totalItems: Int get() = items.sumOf { it.quantity }
    val subtotal: Double get() = quotedSubtotalMinor?.toDouble() ?: items.sumOf { it.totalPrice }
    val shipping: Double get() = if (subtotal > 50_000) 0.0 else 15_000.0
    val discount: Double get() = quotedDiscountMinor?.toDouble() ?: 0.0
    val total: Double get() = subtotal + shipping - discount
    val displaySubtotal: String get() = "%,.0f".format(subtotal)
    val displayShipping: String get() = if (shipping == 0.0) "Free" else "%,.0f".format(shipping)
    val displayDiscount: String get() = if (discount == 0.0) "0" else "-%,.0f".format(discount)
    val displayTotal: String get() = "%,.0f".format(total)
    val firstProductName: String get() = items.firstOrNull()?.product?.name ?: "Order"
    val selectedPaymentLabel: String get() = checkoutPaymentLabel(selectedPaymentGateway, paymentOptions)
    val displayDeliveryFee: String
        get() = if (deliveryFeeMinor > 0) "%,.0f".format(deliveryFeeMinor.toDouble()) else "Free"
    val syncMessage: String?
        get() = when (loadIssue) {
            CartLoadIssue.RESTRICTED -> "Cart sync access is restricted for this account"
            CartLoadIssue.OFFLINE -> "Offline mode active. Showing latest cart"
            CartLoadIssue.ERROR -> "Cart sync degraded. Retry is available"
            null -> null
        }
}

private fun checkoutPaymentLabel(gateway: String, options: List<CheckoutPaymentOption>): String {
    return options.find { it.gateway == gateway }?.label ?: when (gateway.uppercase()) {
        "GLOBAL_PAY" -> "GlobalPay"
        "ADYEN" -> "Adyen"
        "CASH" -> "Cash on Delivery"
        else -> gateway
    }
}

private fun com.pegasusx.retailer.data.model.CheckoutPreviewResponse.orderableCaps(): Map<String, Int> {
    val source = orderableQuantities.ifEmpty { maxQuantities }
    return source.mapValues { it.value.toInt() }
}

private fun standardPaymentOptions(): List<CheckoutPaymentOption> = listOf(
    CheckoutPaymentOption(gateway = "GLOBAL_PAY", label = "GlobalPay (New)"),
    CheckoutPaymentOption(gateway = "ADYEN", label = "Adyen"),
    CheckoutPaymentOption(gateway = "CASH", label = "Cash on Delivery"),
)

@HiltViewModel
class CartViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
    private val pendingOrderDao: PendingOrderDao,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CartUiState())
    val uiState: StateFlow<CartUiState> = _uiState.asStateFlow()

    private var paymentListenerJob: Job? = null
    private var cartSyncDebounceJob: Job? = null
    private var cartSyncEventsJob: Job? = null
    private var quoteDebounceJob: Job? = null
    private var previewDebounceJob: Job? = null
    private var lastCartSignature: String = ""

init { 
        flushPendingOrders()
        fetchPaymentOptions()
        refreshCartFromServer()
        observeCartSyncUpdates()
    }

    fun retrySync() {
        refreshCartFromServer()
        fetchPaymentOptions()
    }

    private fun cartSignature(items: List<CartItem>): String {
        return items
            .sortedBy { if (it.variant.id.isNotBlank()) it.variant.id else it.product.id }
            .joinToString("|") { item ->
                val skuId = if (item.variant.id.isNotBlank()) item.variant.id else item.product.id
                val supplierId = item.product.supplierId.orEmpty()
                "$skuId:$supplierId:${item.quantity}:${item.variant.price.toInt()}"
            }
    }

    private fun JsonObject.stringField(name: String): String? {
        return this[name]?.jsonPrimitive?.contentOrNull
    }

    private fun JsonObject.longField(name: String): Long {
        val primitive = this[name]?.jsonPrimitive ?: return 0L
        return primitive.longOrNull ?: primitive.contentOrNull?.toLongOrNull() ?: 0L
    }

    private fun mapServerCart(items: List<JsonObject>): List<CartItem> {
        val existingBySku = mutableMapOf<String, CartItem>()
        _uiState.value.items.forEach { item ->
            existingBySku[item.product.id] = item
            existingBySku[item.variant.id] = item
        }

        return items.mapNotNull { raw ->
            val skuId = raw.stringField("sku_id") ?: return@mapNotNull null
            val supplierId = raw.stringField("supplier_id") ?: ""
            val quantity = raw.longField("quantity").toInt()
            if (quantity <= 0) {
                return@mapNotNull null
            }
            val unitPrice = raw.longField("unit_price").toDouble()

            val existing = existingBySku[skuId]
            if (existing != null) {
                val variant = existing.variant.copy(price = unitPrice)
                val product = existing.product.copy(
                    variants = listOf(variant),
                    supplierId = existing.product.supplierId ?: supplierId,
                    price = unitPrice.toInt(),
                )
                return@mapNotNull CartItem(
                    id = "${product.id}_${variant.id}",
                    product = product,
                    variant = variant,
                    quantity = quantity,
                )
            }

            val fallbackVariant = Variant(
                id = skuId,
                size = "Standard",
                pack = "Per unit",
                packCount = 1,
                weightPerUnit = "1 unit",
                price = unitPrice,
            )
            val fallbackProduct = Product(
                id = skuId,
                name = "Item",
                variants = listOf(fallbackVariant),
                supplierId = supplierId,
                price = unitPrice.toInt(),
            )
            CartItem(
                id = "${fallbackProduct.id}_${fallbackVariant.id}",
                product = fallbackProduct,
                variant = fallbackVariant,
                quantity = quantity,
            )
        }
    }

    private fun refreshCartFromServer() = viewModelScope.launch {
        _uiState.update { it.copy(isRefreshing = true, syncError = null) }
        try {
            val payload = api.getCartSync().jsonObject
            val rawItems = payload["items"]?.jsonArray?.map { it.jsonObject } ?: emptyList()
            val merged = mapServerCart(rawItems)
            val signature = cartSignature(merged)
            if (signature == lastCartSignature) {
                _uiState.update {
                    it.copy(
                        isRefreshing = false,
                        syncError = null,
                        loadIssue = null,
                    )
                }
                return@launch
            }
            _uiState.update {
                it.copy(
                    items = merged,
                    isRefreshing = false,
                    syncError = null,
                    loadIssue = null,
                )
            }
            lastCartSignature = signature
            scheduleQuoteRefresh()
        } catch (e: Exception) {
            val issue = resolveLoadIssue(e)
            _uiState.update {
                it.copy(
                    isRefreshing = false,
                    syncError = resolveErrorMessage(e, issue),
                    loadIssue = issue,
                )
            }
        }
    }

    private fun observeCartSyncUpdates() {
        retailerWebSocket.connect()
        cartSyncEventsJob?.cancel()
        cartSyncEventsJob = viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "CART_SYNC_UPDATED" || it.type == "PROMOTION_CHANGED" }
                .collect {
                    if (it.type == "CART_SYNC_UPDATED") {
                        refreshCartFromServer()
                    } else {
                        scheduleQuoteRefresh()
                    }
                }
        }
    }

    private fun scheduleCartSyncPush() {
        val currentItems = _uiState.value.items
        val signature = cartSignature(currentItems)
        if (signature == lastCartSignature) {
            return
        }

        cartSyncDebounceJob?.cancel()
        cartSyncDebounceJob = viewModelScope.launch {
            delay(250)
            val snapshot = _uiState.value.items
            val snapshotSignature = cartSignature(snapshot)
            if (snapshotSignature == lastCartSignature) {
                return@launch
            }

            val syncItems = snapshot.mapNotNull { item ->
                val skuId = if (item.variant.id.isNotBlank()) item.variant.id else item.product.id
                val supplierId = item.product.supplierId.orEmpty()
                if (skuId.isBlank() || supplierId.isBlank() || item.quantity <= 0) {
                    null
                } else {
                    mapOf(
                        "sku_id" to skuId,
                        "supplier_id" to supplierId,
                        "quantity" to item.quantity,
                        "unit_price" to item.variant.price.toLong(),
                        "currency" to "UZS",
                    )
                }
            }

            try {
                api.postCartSync(body = mapOf("items" to syncItems))
                lastCartSignature = snapshotSignature
                _uiState.update { it.copy(syncError = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        syncError = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun fetchPaymentOptions() = viewModelScope.launch {
        try {
            val cardsElement = api.getCards()
            val cardsArray = when {
                cardsElement is JsonObject -> cardsElement["cards"]?.let { element ->
                    runCatching { element.jsonArray }.getOrNull()
                } ?: emptyList()
                else -> runCatching { cardsElement.jsonArray }.getOrElse { emptyList() }
            }
            val dynamicOptions = cardsArray.mapNotNull { cardElement ->
                val cardObject = runCatching { cardElement.jsonObject }.getOrNull() ?: return@mapNotNull null
                val tokenId = cardObject["token_id"]?.jsonPrimitive?.contentOrNull
                    ?: cardObject["id"]?.jsonPrimitive?.contentOrNull
                if (tokenId.isNullOrBlank()) return@mapNotNull null

                val panMask = cardObject["pan_mask"]?.jsonPrimitive?.contentOrNull
                    ?: cardObject["pan"]?.jsonPrimitive?.contentOrNull
                    ?: "Card"
                val lastFour = panMask.filter(Char::isDigit).takeLast(4)
                val label = if (lastFour.length == 4) "•••• $lastFour" else panMask

                CheckoutPaymentOption(gateway = tokenId, label = label)
            }

            _uiState.update { it.copy(paymentOptions = dynamicOptions + standardPaymentOptions()) }
            _uiState.update { it.copy(syncError = null, loadIssue = null) }
        } catch (e: Exception) {
            val issue = resolveLoadIssue(e)
            _uiState.update {
                it.copy(
                    paymentOptions = standardPaymentOptions(),
                    syncError = resolveErrorMessage(e, issue),
                    loadIssue = issue,
                )
            }
        }
    }

    private fun flushPendingOrders() = viewModelScope.launch {
        val pending = pendingOrderDao.getAll()
        for (order in pending) {
            try {
                val request = Json.decodeFromString<UnifiedCheckoutRequest>(order.payloadJson)
                api.unifiedCheckout(request, order.idempotencyKey)
                pendingOrderDao.deleteById(order.id)
                _uiState.update { it.copy(syncError = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                val reason = resolveErrorMessage(e, issue)
                pendingOrderDao.incrementRetry(order.id, reason)
                _uiState.update {
                    it.copy(
                        syncError = reason,
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun addToCart(product: Product, variant: Variant) {
        if (product.blocksAddToCart) return
        val itemId = "${product.id}_${variant.id}"
        _uiState.update { state ->
            val existing = state.items.find { it.id == itemId }
            if (existing != null) {
                val next = existing.quantity + 1
                val capped = product.cartMaxQuantity?.let { minOf(next, it) } ?: next
                state.copy(items = state.items.map { if (it.id == itemId) it.copy(quantity = capped) else it })
            } else {
                val qty = product.cartMaxQuantity?.let { minOf(1, it) } ?: 1
                state.copy(items = state.items + CartItem(id = itemId, product = product, variant = variant, quantity = qty))
            }
        }
        scheduleCartSyncPush()
        scheduleQuoteRefresh()
        schedulePreviewRefresh()
    }

    fun updateQuantity(itemId: String, quantity: Int) {
        if (quantity <= 0) {
            removeItem(itemId)
            return
        }
        _uiState.update { state ->
            state.copy(items = state.items.map { item ->
                if (item.id != itemId) item
                else {
                    val cap = effectiveMaxQuantity(item, state)
                    val next = if (cap != null) minOf(quantity, cap) else quantity
                    item.copy(quantity = next)
                }
            })
        }
        scheduleCartSyncPush()
        scheduleQuoteRefresh()
        schedulePreviewRefresh()
    }

    private fun skuIdFor(item: CartItem): String {
        return if (item.variant.id.isNotBlank()) item.variant.id else item.product.id
    }

    private fun effectiveMaxQuantity(item: CartItem, state: CartUiState): Int? {
        val sku = skuIdFor(item)
        if (!item.product.acceptsBackorder) {
            state.previewMaxQuantities[sku]?.let { return it }
        }
        return item.product.cartMaxQuantity?.let { cap ->
            if (item.product.acceptsBackorder) {
                state.orderLineMaxQuantity?.toInt()?.let { minOf(cap, it) } ?: cap
            } else {
                cap
            }
        } ?: state.orderLineMaxQuantity?.toInt()
    }

    private fun applyPreviewCaps(preview: com.pegasusx.retailer.data.model.CheckoutPreviewResponse) {
        val maxMap = preview.orderableCaps()
        _uiState.update { state ->
            state.copy(
                items = state.items.map { item ->
                    if (item.product.acceptsBackorder) return@map item
                    val sku = skuIdFor(item)
                    val cap = maxMap[sku] ?: return@map item
                    if (item.quantity > cap) item.copy(quantity = cap) else item
                },
            )
        }
    }

    fun removeItem(itemId: String) {
        val removedName = _uiState.value.items.find { it.id == itemId }?.product?.name ?: "Item"
        _uiState.update { state ->
            state.copy(
                items = state.items.filter { it.id != itemId },
                removedItemMessage = "$removedName removed from cart"
            )
        }
        scheduleCartSyncPush()
        scheduleQuoteRefresh()
        schedulePreviewRefresh()
    }

    fun clearRemovedItemMessage() {
        _uiState.update { it.copy(removedItemMessage = null) }
    }

    fun clearCart() {
        _uiState.update {
            it.copy(
                items = emptyList(),
                quotedDiscountMinor = null,
                quotedSubtotalMinor = null,
            )
        }
        scheduleCartSyncPush()
    }

    private fun scheduleQuoteRefresh() {
        quoteDebounceJob?.cancel()
        quoteDebounceJob = viewModelScope.launch {
            delay(300)
            refreshCheckoutQuote()
        }
    }

    private fun schedulePreviewRefresh() {
        previewDebounceJob?.cancel()
        previewDebounceJob = viewModelScope.launch {
            delay(300)
            refreshCheckoutPreview()
        }
    }

    private suspend fun refreshCheckoutQuote() {
        val snapshot = _uiState.value.items
        if (snapshot.isEmpty()) {
            _uiState.update { it.copy(quotedDiscountMinor = null, quotedSubtotalMinor = null) }
            return
        }
        val grouped = snapshot.groupBy { it.product.supplierId.orEmpty() }.filterKeys { it.isNotBlank() }
        if (grouped.isEmpty()) {
            _uiState.update { it.copy(quotedDiscountMinor = 0L, quotedSubtotalMinor = null) }
            return
        }
        try {
            var subtotalMinor = 0L
            var discountMinor = 0L
            for ((supplierId, lines) in grouped) {
                val quote = api.checkoutQuote(
                    CheckoutQuoteRequest(
                        supplierId = supplierId,
                        lines = lines.map { item ->
                            val skuId = if (item.variant.id.isNotBlank()) item.variant.id else item.product.id
                            CheckoutQuoteLine(
                                productId = skuId,
                                quantity = item.quantity.toLong(),
                                unitPriceMinor = item.variant.price.toLong(),
                            )
                        },
                    ),
                )
                subtotalMinor += quote.subtotalMinor
                discountMinor += quote.discountMinor
            }
            _uiState.update {
                it.copy(
                    quotedSubtotalMinor = subtotalMinor,
                    quotedDiscountMinor = discountMinor,
                )
            }
        } catch (_: Exception) {
            _uiState.update { it.copy(quotedDiscountMinor = null, quotedSubtotalMinor = null) }
        }
    }

    fun showCheckout() {
        _uiState.update { it.copy(showCheckout = true, checkoutPhase = CheckoutPhase.REVIEW) }
        scheduleQuoteRefresh()
        refreshCheckoutPreview()
    }

    fun setDeliveryMode(mode: String) {
        _uiState.update { it.copy(deliveryMode = mode.uppercase()) }
    }

    fun setDeliveryDate(date: String?) {
        _uiState.update { it.copy(deliveryDate = date?.takeIf { it.isNotBlank() }) }
    }

    fun setExpressPriority(enabled: Boolean) {
        _uiState.update { it.copy(expressPriority = enabled) }
    }

    fun dismissBackorderConfirm() {
        _uiState.update { it.copy(pendingBackorderConfirm = false) }
    }

    fun confirmBackorderCheckout() {
        _uiState.update { it.copy(pendingBackorderConfirm = false) }
        processPayment(skipBackorderConfirm = true)
    }

    private fun refreshCheckoutPreview() = viewModelScope.launch {
        val state = _uiState.value
        if (state.items.isEmpty()) {
            _uiState.update {
                it.copy(
                    previewMaxQuantities = emptyMap(),
                    previewShortfall = emptyMap(),
                    oosItems = emptyList(),
                    stockWarnings = emptyList(),
                    previewLoading = false,
                )
            }
            return@launch
        }
        _uiState.update { it.copy(previewLoading = true) }
        try {
            val retailerId = tokenManager.getUserId().orEmpty()
            val request = buildCheckoutRequest(state, retailerId, state.selectedPaymentGateway)
            val preview = api.checkoutPreview(request)
            applyPreviewCaps(preview)
            _uiState.update {
                it.copy(
                    previewMaxQuantities = preview.orderableCaps(),
                    previewShowStockCounts = preview.showStockCounts,
                    previewShortfall = preview.shortfall,
                    oosItems = preview.oosItems.ifEmpty { preview.rejectedSkus },
                    stockWarnings = preview.stockWarnings,
                    preorderMinLeadDays = preview.preorderMinLeadDays.takeIf { days -> days > 0 } ?: 3,
                    preorderMaxLeadDays = preview.preorderMaxLeadDays,
                    deliveryFeeMinor = preview.deliveryFeeMinor,
                    deliveryDistanceKm = preview.deliveryDistanceKm,
                    orderLineMinQuantity = preview.orderLineMinQuantity,
                    orderLineMaxQuantity = preview.orderLineMaxQuantity,
                    previewLoading = false,
                )
            }
        } catch (e: Exception) {
            _uiState.update {
                it.copy(
                    previewLoading = false,
                    checkoutError = resolveCheckoutErrorMessage(e, "Could not refresh stock preview"),
                )
            }
        }
    }

    private fun buildCheckoutRequest(
        state: CartUiState,
        retailerId: String,
        gateway: String,
    ): UnifiedCheckoutRequest {
        val lineItems = state.items.map { cartItem ->
            CheckoutLineItem(
                skuId = cartItem.variant.id.ifBlank { cartItem.product.id },
                quantity = cartItem.quantity,
                unitPrice = cartItem.variant.price.toLong(),
            )
        }
        val requestedDeliveryDate = state.deliveryDate?.let { date ->
            java.time.LocalDate.parse(date)
                .atTime(12, 0)
                .atOffset(java.time.ZoneOffset.ofHours(5))
                .format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
        }
        return UnifiedCheckoutRequest(
            retailerId = retailerId,
            paymentGateway = gateway,
            items = lineItems,
            deliveryMode = state.deliveryMode,
            requestedDeliveryDate = if (state.deliveryMode == "SCHEDULED") requestedDeliveryDate else null,
            deliveryPriority = if (state.expressPriority) "EXPRESS" else "STANDARD",
        )
    }

    fun setSupplierIsActive(value: Boolean) {
        _uiState.update { it.copy(supplierIsActive = value) }
    }

    fun dismissCheckout() {
        _uiState.update { it.copy(showCheckout = false, checkoutPhase = CheckoutPhase.REVIEW) }
    }

    fun setSelectedPaymentGateway(gateway: String) {
        // Uppercase known gateways, otherwise keep token IDs untouched.
        val gw = if (gateway.equals("GLOBAL_PAY", ignoreCase=true) || gateway.equals("ADYEN", ignoreCase=true) || gateway.equals("CASH", ignoreCase=true)) gateway.trim().uppercase() else gateway.trim()
        _uiState.update { it.copy(selectedPaymentGateway = gw) }
    }

    fun processPayment(skipBackorderConfirm: Boolean = false) {
        viewModelScope.launch {
            _uiState.update { it.copy(checkoutPhase = CheckoutPhase.PROCESSING, checkoutError = null) }
            var checkoutRequest: UnifiedCheckoutRequest? = null
            try {
                val state = _uiState.value
                val retailerId = tokenManager.getUserId() ?: ""
                var finalGateway = state.selectedPaymentGateway
                if (finalGateway != "GLOBAL_PAY" && finalGateway != "ADYEN" && finalGateway != "CASH") {
                    try {
                        api.setDefaultCard(mapOf("token_id" to finalGateway))
                        finalGateway = "GLOBAL_PAY"
                    } catch (e: Exception) {
                        _uiState.update {
                            it.copy(
                                checkoutPhase = CheckoutPhase.REVIEW,
                                checkoutError = resolveCheckoutErrorMessage(e, "Failed to select payment method"),
                            )
                        }
                        return@launch
                    }
                }

                val preview = api.checkoutPreview(buildCheckoutRequest(state, retailerId, finalGateway))
                applyPreviewCaps(preview)
                val refreshedState = _uiState.value
                if (preview.blocked) {
                    _uiState.update {
                        it.copy(
                            checkoutPhase = CheckoutPhase.REVIEW,
                            checkoutError = preview.message ?: "Checkout blocked by stock policy",
                            oosItems = preview.oosItems.ifEmpty { preview.rejectedSkus },
                            stockWarnings = preview.stockWarnings,
                            previewMaxQuantities = preview.orderableCaps(),
                            previewShowStockCounts = preview.showStockCounts,
                            previewShortfall = preview.shortfall,
                        )
                    }
                    return@launch
                }
                if (preview.stockWarnings.isNotEmpty() && !skipBackorderConfirm) {
                    _uiState.update {
                        it.copy(
                            checkoutPhase = CheckoutPhase.REVIEW,
                            stockWarnings = preview.stockWarnings,
                            pendingBackorderConfirm = true,
                        )
                    }
                    return@launch
                }

                val request = buildCheckoutRequest(refreshedState, retailerId, finalGateway)
                checkoutRequest = request
                val response = api.unifiedCheckout(request, checkoutIdempotencyKey(request))
                val firstOrderId = response.supplierOrders.firstOrNull()?.orderId
                _uiState.update {
                    it.copy(
                        stockWarnings = response.stockWarnings,
                        lastOrderId = firstOrderId ?: response.invoiceId,
                        paymentUrlToOpen = null,
                        checkoutPhase = CheckoutPhase.COMPLETE,
                    )
                }
                delay(1800)
                _uiState.update {
                    it.copy(
                        showCheckout = false,
                        checkoutPhase = CheckoutPhase.REVIEW,
                        items = emptyList(),
                        lastOrderId = null,
                    )
                }
            } catch (e: HttpException) {
                val body = e.response()?.errorBody()?.string()
                var msg: String
                var flaggedOos = emptyList<String>()
                if (e.code() == 409) {
                    // Attempt to parse structured OOS response: {"code":"ALL_ITEMS_OUT_OF_STOCK","oos_items":["sku1","sku2"]}
                    try {
                        val json = body?.let { Json.decodeFromString<Map<String, kotlinx.serialization.json.JsonElement>>(it) }
                        val code = json?.get("code")?.toString()?.trim('"') ?: ""
                        val oosArr = json?.get("oos_items")
                        if (oosArr is kotlinx.serialization.json.JsonArray) {
                            flaggedOos = oosArr.mapNotNull { it.toString().trim('"').takeIf { s -> s.isNotEmpty() } }
                        }
                        msg = when (code) {
                            "inventory_exhausted" -> "Stock changed while checking out. Review your cart and try again."
                            "ALL_ITEMS_OUT_OF_STOCK" -> "All items are out of stock. Please update your cart."
                            "PARTIAL_OUT_OF_STOCK_REJECTED" -> "Some items are out of stock and cannot be backordered. Please update your cart."
                            else -> body ?: "Some items are out of stock"
                        }
                    } catch (_: Exception) {
                        msg = body ?: "Item out of stock — pull to refresh"
                    }
                    refreshCheckoutPreview()
                } else {
                    msg = body ?: "Checkout failed (${e.code()})"
                }
                _uiState.update {
                    it.copy(
                        checkoutPhase = CheckoutPhase.REVIEW,
                        checkoutError = msg,
                        oosItems = flaggedOos,
                    )
                }
            } catch (e: Exception) {
                val request = checkoutRequest
                if (e is IOException && request != null) {
                    queuePendingCheckout(request, checkoutIdempotencyKey(request))
                }
                _uiState.update {
                    it.copy(
                        checkoutPhase = CheckoutPhase.REVIEW,
                        checkoutError = if (e is IOException && request != null) {
                            "Network issue during checkout. Saved for automatic retry when you reconnect."
                        } else {
                            resolveCheckoutErrorMessage(e, "Checkout failed")
                        },
                    )
                }
            }
        }
    }

    fun clearCheckoutError() {
        _uiState.update { it.copy(checkoutError = null) }
    }

    fun clearPaymentUrl() {
        _uiState.update { it.copy(paymentUrlToOpen = null) }
    }

    fun effectiveMaxQuantityFor(item: CartItem): Int? {
        return effectiveMaxQuantity(item, _uiState.value)
    }

    fun stockLeftHintFor(item: CartItem): String? {
        val state = _uiState.value
        if (!state.previewShowStockCounts || item.product.acceptsBackorder) return null
        val cap = state.previewMaxQuantities[skuIdFor(item)] ?: return null
        return if (cap > 0) "Only $cap left" else null
    }

    private fun checkoutIdempotencyKey(request: UnifiedCheckoutRequest): String {
        val itemKey = request.items
            .sortedBy { it.skuId }
            .joinToString("|") { "${it.skuId}:${it.quantity}:${it.unitPrice}" }
        return "retailer-checkout:${request.paymentGateway}:$itemKey"
    }

    private suspend fun queuePendingCheckout(request: UnifiedCheckoutRequest, idempotencyKey: String) {
        pendingOrderDao.insert(
            PendingOrderEntity(
                payloadJson = Json.encodeToString(UnifiedCheckoutRequest.serializer(), request),
                idempotencyKey = idempotencyKey,
            ),
        )
    }

    private fun resolveLoadIssue(error: Exception): CartLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> CartLoadIssue.RESTRICTED
            error is IOException -> CartLoadIssue.OFFLINE
            else -> CartLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: CartLoadIssue): String {
        return when (issue) {
            CartLoadIssue.RESTRICTED -> "Cart sync access is restricted for this account"
            CartLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            CartLoadIssue.ERROR -> error.message ?: "Cart sync failed"
        }
    }

    private fun resolveCheckoutErrorMessage(error: Exception, fallback: String): String {
        return when (resolveLoadIssue(error)) {
            CartLoadIssue.RESTRICTED -> "Checkout access is restricted for this account"
            CartLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry checkout"
            CartLoadIssue.ERROR -> error.message ?: fallback
        }
    }

    override fun onCleared() {
        super.onCleared()
        paymentListenerJob?.cancel()
        cartSyncDebounceJob?.cancel()
        cartSyncEventsJob?.cancel()
        quoteDebounceJob?.cancel()
    }
}
