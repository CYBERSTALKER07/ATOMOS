package com.pegasusx.retailer.ui.screens.cart

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.CardCheckoutRequest
import com.pegasusx.retailer.data.model.CartItem
import com.pegasusx.retailer.data.model.CashCheckoutRequest
import com.pegasusx.retailer.data.model.CheckoutLineItem
import com.pegasusx.retailer.data.model.CheckoutQuoteLine
import com.pegasusx.retailer.data.model.CheckoutQuoteRequest
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.StockWarning
import com.pegasusx.retailer.data.model.SupplierOrderResult
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
                val response = api.unifiedCheckout(request, order.idempotencyKey)
                completeSupplierOrderPayments(
                    gateway = request.paymentGateway,
                    invoiceId = response.invoiceId,
                    supplierOrders = response.supplierOrders,
                )
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
                state.copy(items = state.items.map { if (it.id == itemId) it.copy(quantity = it.quantity + 1) else it })
            } else {
                state.copy(items = state.items + CartItem(id = itemId, product = product, variant = variant, quantity = 1))
            }
        }
        scheduleCartSyncPush()
        scheduleQuoteRefresh()
    }

    fun updateQuantity(itemId: String, quantity: Int) {
        if (quantity <= 0) {
            removeItem(itemId)
            return
        }
        _uiState.update { state ->
            state.copy(items = state.items.map { if (it.id == itemId) it.copy(quantity = quantity) else it })
        }
        scheduleCartSyncPush()
        scheduleQuoteRefresh()
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

    fun processPayment() {
        viewModelScope.launch {
            _uiState.update { it.copy(checkoutPhase = CheckoutPhase.PROCESSING, checkoutError = null, stockWarnings = emptyList()) }
            try {
                val state = _uiState.value
                val retailerId = tokenManager.getUserId() ?: ""
                val lineItems = state.items.map { cartItem ->
                    CheckoutLineItem(
                        skuId = cartItem.variant.id,
                        quantity = cartItem.quantity,
                        unitPrice = cartItem.variant.price.toLong(),
                    )
                }
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

                val request = UnifiedCheckoutRequest(
                    retailerId = retailerId,
                    paymentGateway = finalGateway,
                    items = lineItems,
                )
                val response = api.unifiedCheckout(request, checkoutIdempotencyKey(request))
                val paymentUrl = completeSupplierOrderPayments(
                    gateway = finalGateway,
                    invoiceId = response.invoiceId,
                    supplierOrders = response.supplierOrders,
                )
                val firstOrderId = response.supplierOrders.firstOrNull()?.orderId
                _uiState.update {
                    it.copy(
                        stockWarnings = response.stockWarnings,
                        lastOrderId = firstOrderId ?: response.invoiceId,
                        paymentUrlToOpen = paymentUrl,
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
                            "ALL_ITEMS_OUT_OF_STOCK" -> "All items are out of stock. Please update your cart."
                            "PARTIAL_OUT_OF_STOCK_REJECTED" -> "Some items are out of stock and cannot be backordered. Please update your cart."
                            else -> body ?: "Some items are out of stock"
                        }
                    } catch (_: Exception) {
                        msg = body ?: "Item out of stock — pull to refresh"
                    }
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
                _uiState.update {
                    it.copy(
                        checkoutPhase = CheckoutPhase.REVIEW,
                        checkoutError = resolveCheckoutErrorMessage(e, "Checkout failed"),
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

    private suspend fun completeSupplierOrderPayments(
        gateway: String,
        invoiceId: String,
        supplierOrders: List<SupplierOrderResult>,
    ): String? {
        if (supplierOrders.isEmpty()) return null
        val normalizedGateway = gateway.trim().uppercase()
        var paymentUrl: String? = null
        for (supplierOrder in supplierOrders) {
            if (normalizedGateway == "CASH") {
                api.cashCheckout(
                    CashCheckoutRequest(orderId = supplierOrder.orderId),
                    "retailer-cash-checkout:${supplierOrder.orderId}",
                )
            } else {
                val card = api.cardCheckout(
                    CardCheckoutRequest(orderId = supplierOrder.orderId, gateway = normalizedGateway),
                    "retailer-card-checkout:${supplierOrder.orderId}:$normalizedGateway",
                )
                if (card.paymentUrl.isNotBlank()) {
                    paymentUrl = card.paymentUrl
                }
            }
        }
        return paymentUrl
    }

    private fun checkoutIdempotencyKey(request: UnifiedCheckoutRequest): String {
        val itemKey = request.items
            .sortedBy { it.skuId }
            .joinToString("|") { "${it.skuId}:${it.quantity}:${it.unitPrice}" }
        return "retailer-checkout:${request.paymentGateway}:$itemKey"
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
