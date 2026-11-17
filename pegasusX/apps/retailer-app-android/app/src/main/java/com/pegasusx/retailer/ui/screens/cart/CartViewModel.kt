package com.pegasusx.retailer.ui.screens.cart


import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.CartRealtimeEventSource
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.RetailerUserIdProvider
import com.pegasusx.retailer.data.model.selectableRetailerCatalogCodes
import com.pegasusx.retailer.data.model.sessionPackCurrency
import com.pegasusx.retailer.data.model.CartItem
import com.pegasusx.retailer.data.model.CheckoutLineItem
import com.pegasusx.retailer.data.model.CheckoutQuoteLine
import com.pegasusx.retailer.data.model.CheckoutQuoteRequest
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.StockWarning
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import com.pegasusx.retailer.data.model.Variant
import com.pegasusx.retailer.services.PendingOrderStatusBus
import com.pegasusx.retailer.services.PendingOrderStatusNotice
import com.pegasusx.retailer.services.PendingOrderSyncEnqueuer
import com.pegasusx.retailer.services.PendingOrderSyncProcessor
import com.pegasusx.retailer.ui.components.CheckoutPhase
import com.pegasusx.retailer.ui.components.CheckoutPaymentOption
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.OwnerApprovalRequiredDetail
import com.pegasusx.retailer.util.PackMultipleSuggestion
import com.pegasusx.retailer.util.PackSnapMode
import com.pegasusx.retailer.util.PendingOrderConflictKind
import com.pegasusx.retailer.util.RetailerW2ContractHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.CancellationException
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
import kotlinx.serialization.json.booleanOrNull
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
    val previewStockPolicyReject: Boolean = true,
    val checkoutPolicyToken: String? = null,
    val previewShortfall: Map<String, Long> = emptyMap(),
    val previewLoading: Boolean = false,
    val deliveryMode: String = "STANDARD",
    val deliveryDate: String? = null,
    val expressPriority: Boolean = false,
    val currencyPickerEnabled: Boolean = false,
    val currencyAllowlist: List<String> = emptyList(),
    val operatingCurrency: String = "",
    val orderCurrency: String = "",
    val catalogCodes: List<String> = emptyList(),
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
    val autoConfirmOptIn: Boolean = false,
    val packSuggestions: List<PackMultipleSuggestion> = emptyList(),
    val pendingQueueNotice: PendingOrderStatusNotice? = null,
    val ownerApprovalRequired: OwnerApprovalRequiredDetail? = null,
) {
    val isEmpty: Boolean get() = items.isEmpty()
    val totalItems: Int get() = items.sumOf { it.quantity }
    val subtotal: Long get() = quotedSubtotalMinor ?: items.sumOf { it.totalPrice }
    val shipping: Long get() = if (subtotal > 50_000L) 0L else 15_000L
    val discount: Long get() = quotedDiscountMinor ?: 0L
    val total: Long get() = subtotal + shipping - discount
    val displaySubtotal: String get() = "%,d".format(subtotal)
    val displayShipping: String get() = if (shipping == 0L) "Free" else "%,d".format(shipping)
    val displayDiscount: String get() = if (discount == 0L) "0" else "-%,d".format(discount)
    val displayTotal: String get() = "%,d".format(total)
    val firstProductName: String get() = items.firstOrNull()?.product?.name ?: "Order"
    val selectedPaymentLabel: String
        get() = paymentOptions.find { it.gateway.equals(selectedPaymentGateway.trim(), ignoreCase = true) }?.label
            ?: when (selectedPaymentGateway.trim().uppercase()) {
                "GLOBAL_PAY" -> "GlobalPay"
                "CASH" -> "Cash on Delivery"
                else -> selectedPaymentGateway.trim()
            }
    val displayDeliveryFee: String
        get() = if (deliveryFeeMinor > 0) "%,d".format(deliveryFeeMinor) else "Free"
    val syncMessage: String?
        get() = when (loadIssue) {
            CartLoadIssue.RESTRICTED -> "Cart sync access is restricted for this account"
            CartLoadIssue.OFFLINE -> "Offline mode active. Showing latest cart"
            CartLoadIssue.ERROR -> "Cart sync degraded. Retry is available"
            null -> null
        }
}

private fun com.pegasusx.retailer.data.model.CheckoutPreviewResponse.orderableCaps(): Map<String, Int> {
    val source = orderableQuantities.ifEmpty { maxQuantities }
    return source.mapValues { it.value.toInt() }
}

private fun standardPaymentOptions(catalogCodes: List<String> = emptyList()): List<CheckoutPaymentOption> {
    val codes = if (catalogCodes.isEmpty()) listOf("CASH") else catalogCodes
    return codes.mapNotNull { code ->
        when (code) {
            "CASH" -> CheckoutPaymentOption(
                gateway = "CASH",
                label = "Cash on Delivery",
                labelRes = com.pegasusx.retailer.R.string.supplier_portal_billing_setup_gateway_cash_label,
            )
            "GLOBAL_PAY" -> CheckoutPaymentOption(
                gateway = "GLOBAL_PAY",
                label = "GlobalPay",
                labelRes = com.pegasusx.retailer.R.string.mobile_retailer_ui_globalpay,
            )
            else -> null
        }
    }
}

@HiltViewModel
class CartViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: RetailerUserIdProvider,
    private val retailerWebSocket: CartRealtimeEventSource,
    private val pendingOrderDao: PendingOrderDao,
    private val syncEnqueuer: PendingOrderSyncEnqueuer = PendingOrderSyncEnqueuer {},
) : ViewModel() {

    private val _uiState = MutableStateFlow(CartUiState())
    val uiState: StateFlow<CartUiState> = _uiState.asStateFlow()

    private var paymentListenerJob: Job? = null
    private var cartSyncDebounceJob: Job? = null
    private var cartSyncEventsJob: Job? = null
    private var quoteDebounceJob: Job? = null
    private var previewDebounceJob: Job? = null
    private var lastCartSignature: String = ""
    private var currentCheckoutSessionId: String? = null

    private fun getOrCreateCheckoutSessionId(): String {
        return currentCheckoutSessionId ?: java.util.UUID.randomUUID().toString().also {
            currentCheckoutSessionId = it
        }
    }

init { 
        flushPendingOrders()
        fetchPaymentCatalog()
        fetchPaymentOptions()
        refreshCartFromServer()
        observeCartSyncUpdates()
    }

    fun retrySync() {
        flushPendingOrders()
        refreshCartFromServer()
        fetchPaymentCatalog()
        fetchPaymentOptions()
    }

    fun retryPendingOrderSync() {
        flushPendingOrders()
    }

    private fun cartSignature(items: List<CartItem>): String {
        return items
            .sortedBy { if (it.variant.id.isNotBlank()) it.variant.id else it.product.id }
            .joinToString("|") { item ->
                val skuId = if (item.variant.id.isNotBlank()) item.variant.id else item.product.id
                val supplierId = item.product.supplierId.orEmpty()
                "$skuId:$supplierId:${item.quantity}:${item.variant.price}:${item.mustHave}:${item.allowSubstitutes}"
            }
    }

    private fun JsonObject.stringField(name: String): String? {
        return this[name]?.jsonPrimitive?.contentOrNull
    }

    private fun JsonObject.longField(name: String): Long {
        val primitive = this[name]?.jsonPrimitive ?: return 0L
        return primitive.longOrNull
            ?: primitive.contentOrNull?.trim()?.let { raw ->
                raw.toLongOrNull() ?: runCatching {
                    java.math.BigDecimal(raw).setScale(0, java.math.RoundingMode.HALF_UP).toLong()
                }.getOrNull()
            }
            ?: 0L
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
            val unitPrice = raw.longField("unit_price")
            val existing = existingBySku[skuId]
            val mustHave = raw["must_have"]?.jsonPrimitive?.booleanOrNull ?: existing?.mustHave ?: false
            val allowSubstitutes = raw["allow_substitutes"]?.jsonPrimitive?.booleanOrNull
                ?: existing?.allowSubstitutes
                ?: false

            if (existing != null) {
                val variant = existing.variant.copy(price = unitPrice)
                val product = existing.product.copy(
                    variants = listOf(variant),
                    supplierId = existing.product.supplierId ?: supplierId,
                    price = unitPrice,
                )
                return@mapNotNull CartItem(
                    id = "${product.id}_${variant.id}",
                    product = product,
                    variant = variant,
                    quantity = quantity,
                    mustHave = mustHave,
                    allowSubstitutes = allowSubstitutes,
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
                price = unitPrice,
            )
            CartItem(
                id = "${fallbackProduct.id}_${fallbackVariant.id}",
                product = fallbackProduct,
                variant = fallbackVariant,
                quantity = quantity,
                mustHave = mustHave,
                allowSubstitutes = allowSubstitutes,
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
                        "unit_price" to item.variant.price,
                        "currency" to sessionPackCurrency(),
                        "must_have" to item.mustHave,
                        "allow_substitutes" to item.allowSubstitutes,
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

            _uiState.update { it.copy(paymentOptions = dynamicOptions + standardPaymentOptions(it.catalogCodes)) }
            _uiState.update { it.copy(syncError = null, loadIssue = null) }
        } catch (e: Exception) {
            val issue = resolveLoadIssue(e)
            _uiState.update {
                it.copy(
                    paymentOptions = standardPaymentOptions(it.catalogCodes),
                    syncError = resolveErrorMessage(e, issue),
                    loadIssue = issue,
                )
            }
        }
    }

    private fun fetchPaymentCatalog() = viewModelScope.launch {
        try {
            val resp = api.getPaymentCatalog()
            val codes = selectableRetailerCatalogCodes(resp.catalog)
            val currency = resp.currencyCode.ifBlank { sessionPackCurrency() }
            _uiState.update {
                val nextOptions = it.paymentOptions.filter { option ->
                    option.gateway !in setOf("GLOBAL_PAY", "CASH", "ADYEN", "STRIPE")
                } + standardPaymentOptions(codes)
                val selected = when {
                    nextOptions.any { option -> option.gateway == it.selectedPaymentGateway } -> it.selectedPaymentGateway
                    nextOptions.any { option -> option.gateway == "CASH" } -> "CASH"
                    else -> nextOptions.firstOrNull()?.gateway ?: "CASH"
                }
                it.copy(
                    catalogCodes = codes,
                    currencyPickerEnabled = false,
                    currencyAllowlist = emptyList(),
                    operatingCurrency = currency,
                    orderCurrency = currency,
                    selectedPaymentGateway = selected,
                    paymentOptions = nextOptions,
                )
            }
        } catch (_: Exception) {
            val currency = sessionPackCurrency()
            _uiState.update {
                it.copy(
                    catalogCodes = emptyList(),
                    currencyPickerEnabled = false,
                    operatingCurrency = currency,
                    orderCurrency = currency,
                    paymentOptions = it.paymentOptions.filter { option ->
                        option.gateway !in setOf("GLOBAL_PAY", "CASH", "ADYEN", "STRIPE")
                    } + standardPaymentOptions(emptyList()),
                )
            }
        }
    }

    private fun flushPendingOrders() = viewModelScope.launch {
        val summary = PendingOrderSyncProcessor(
            api = api,
            pendingOrderDao = pendingOrderDao,
            json = OrderLineContractHelper.lenientJson,
            onStatusNotice = { notice ->
                PendingOrderStatusBus.emit(notice)
                _uiState.update { it.copy(pendingQueueNotice = notice) }
            },
        ).processAll()

        if (summary.attempted > 0) {
            if (summary.shouldRetry) {
                syncEnqueuer.enqueue()
                val keptNotice = summary.notices.firstOrNull { it.keptInQueue }
                _uiState.update {
                    it.copy(
                        syncError = keptNotice?.message ?: "Queued orders pending automatic sync retry.",
                        pendingQueueNotice = keptNotice ?: it.pendingQueueNotice,
                    )
                }
            } else {
                _uiState.update { it.copy(syncError = null, loadIssue = null) }
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

    fun effectivePackSizeFor(item: CartItem): Int =
        OrderLineContractHelper.effectivePackSize(item.product, item.variant)

    fun needsPackSnap(item: CartItem): Boolean {
        val packSize = effectivePackSizeFor(item)
        return packSize > 1 && item.quantity % packSize != 0
    }

    fun snapItemToPackMultiple(itemId: String, mode: PackSnapMode = PackSnapMode.NEAREST) {
        val targetItem = _uiState.value.items.find { it.id == itemId } ?: return
        val packSize = effectivePackSizeFor(targetItem)
        val snapped = OrderLineContractHelper.snapToPackMultiple(targetItem.quantity, packSize, mode)
        updateQuantity(itemId, snapped)
        val skuId = skuIdFor(targetItem)
        _uiState.update { state ->
            val remaining = state.packSuggestions.filterNot {
                it.skuId == skuId || it.skuId == targetItem.product.id || it.skuId == targetItem.variant.id
            }
            state.copy(
                packSuggestions = remaining,
                checkoutError = if (remaining.isEmpty()) null else state.checkoutError,
            )
        }
    }

    fun applyPackSuggestion(skuId: String, suggestedQuantity: Int) {
        val matched = _uiState.value.items.firstOrNull {
            skuIdFor(it) == skuId || it.product.id == skuId || it.variant.id == skuId
        }
        if (matched != null) {
            updateQuantity(matched.id, suggestedQuantity)
        }
        _uiState.update { state ->
            val remaining = state.packSuggestions.filterNot { it.skuId == skuId }
            state.copy(
                packSuggestions = remaining,
                checkoutError = if (remaining.isEmpty()) null else state.checkoutError,
            )
        }
    }

    fun dismissPackSuggestions() {
        _uiState.update { it.copy(packSuggestions = emptyList()) }
    }

    fun toggleMustHave(itemId: String) {
        _uiState.update { state ->
            state.copy(
                items = state.items.map { item ->
                    if (item.id == itemId) item.copy(mustHave = !item.mustHave) else item
                },
            )
        }
        scheduleCartSyncPush()
    }

    fun toggleAllowSubstitutes(itemId: String) {
        _uiState.update { state ->
            state.copy(
                items = state.items.map { item ->
                    if (item.id == itemId) item.copy(allowSubstitutes = !item.allowSubstitutes) else item
                },
            )
        }
        scheduleCartSyncPush()
    }

    fun setAutoConfirmOptIn(enabled: Boolean) {
        _uiState.update { it.copy(autoConfirmOptIn = enabled) }
    }

    fun dismissPendingQueueNotice() {
        _uiState.update { it.copy(pendingQueueNotice = null) }
        PendingOrderStatusBus.clear()
    }

    private fun skuIdFor(item: CartItem): String {
        return if (item.variant.id.isNotBlank()) item.variant.id else item.product.id
    }

    private fun effectiveMaxQuantity(item: CartItem, state: CartUiState): Int? {
        val sku = skuIdFor(item)
        val rejectPolicy = state.previewStockPolicyReject && !item.product.acceptsBackorder
        if (rejectPolicy) {
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
                    if (!state.previewStockPolicyReject || item.product.acceptsBackorder) return@map item
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
                                unitPriceMinor = item.variant.price,
                                currency = sessionPackCurrency(),
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
        } catch (e: Exception) {
            if (e is kotlinx.coroutines.CancellationException) throw e
            _uiState.update { it.copy(quotedDiscountMinor = null, quotedSubtotalMinor = null) }
        }
    }

    fun showCheckout() {
        if (currentCheckoutSessionId == null) {
            currentCheckoutSessionId = java.util.UUID.randomUUID().toString()
        }
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
                    previewStockPolicyReject = preview.defaultOutOfStockPolicy?.uppercase() != "ACCEPT_BACKORDER",
                    checkoutPolicyToken = preview.checkoutPolicyToken,
                    previewShortfall = preview.shortfall,
                    oosItems = preview.oosItems.ifEmpty { preview.rejectedSkus },
                    stockWarnings = preview.stockWarnings,
                    preorderMinLeadDays = preview.preorderMinLeadDays?.takeIf { days -> days > 0 } ?: 3L,
                    preorderMaxLeadDays = preview.preorderMaxLeadDays ?: 0L,
                    deliveryFeeMinor = preview.deliveryFeeMinor,
                    deliveryDistanceKm = preview.deliveryDistanceKm ?: 0.0,
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
        paymentMethodOverride: String? = null,
    ): UnifiedCheckoutRequest {
        val lineItems = state.items.map { cartItem ->
            CheckoutLineItem(
                skuId = cartItem.variant.id.ifBlank { cartItem.product.id },
                quantity = cartItem.quantity,
                unitPrice = cartItem.variant.price,
                mustHave = cartItem.mustHave,
                allowSubstitutes = cartItem.allowSubstitutes,
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
            checkoutPolicyToken = state.checkoutPolicyToken,
            currency = null,
            autoConfirmOptIn = state.autoConfirmOptIn,
            paymentMethod = paymentMethodOverride,
        )
    }

    fun setOrderCurrency(code: String) {
        _uiState.update { it.copy(orderCurrency = code.trim().uppercase()) }
    }

    fun setSupplierIsActive(value: Boolean) {
        _uiState.update { it.copy(supplierIsActive = value) }
    }

    fun dismissCheckout() {
        currentCheckoutSessionId = null
        _uiState.update {
            it.copy(
                showCheckout = false,
                checkoutPhase = CheckoutPhase.REVIEW,
                ownerApprovalRequired = null,
            )
        }
    }

    fun dismissOwnerApprovalRequired() {
        _uiState.update { it.copy(ownerApprovalRequired = null) }
    }

    fun setSelectedPaymentGateway(gateway: String) {
        // Uppercase known gateways, otherwise keep token IDs untouched.
        val known = setOf("GLOBAL_PAY", "CASH")
        val gw = if (known.any { gateway.equals(it, ignoreCase = true) }) gateway.trim().uppercase() else gateway.trim()
        _uiState.update { it.copy(selectedPaymentGateway = gw) }
    }

    fun retryCheckoutWithFallbackMethod(method: String) {
        val normalized = method.trim().uppercase()
        if (normalized.isEmpty()) return
        val state = _uiState.value
        val targetGateway = when (normalized) {
            "CASH" -> "CASH"
            else -> state.paymentOptions.firstOrNull { !it.gateway.equals("CASH", ignoreCase = true) }?.gateway
                ?: if (state.catalogCodes.contains("GLOBAL_PAY")) "GLOBAL_PAY" else normalized
        }
        currentCheckoutSessionId = "${getOrCreateCheckoutSessionId()}:fallback:$normalized"
        _uiState.update {
            it.copy(
                selectedPaymentGateway = targetGateway,
                ownerApprovalRequired = null,
                pendingQueueNotice = null,
                checkoutError = null,
            )
        }
        viewModelScope.launch {
            runCatching {
                pendingOrderDao.getAll()
                    .filter {
                        it.status == PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL &&
                            it.endpoint == "/v1/checkout/unified"
                    }
                    .forEach { pendingOrderDao.deleteById(it.id) }
            }
            submitCheckout(skipBackorderConfirm = false, paymentMethodOverride = normalized)
        }
    }

    fun processPayment(skipBackorderConfirm: Boolean = false) {
        viewModelScope.launch {
            submitCheckout(skipBackorderConfirm = skipBackorderConfirm, paymentMethodOverride = null)
        }
    }

    private suspend fun submitCheckout(
        skipBackorderConfirm: Boolean = false,
        paymentMethodOverride: String? = null,
    ) {
        _uiState.update {
            it.copy(
                checkoutPhase = CheckoutPhase.PROCESSING,
                checkoutError = null,
                packSuggestions = emptyList(),
                ownerApprovalRequired = null,
            )
        }
        val initialState = _uiState.value
        val retailerId = tokenManager.getUserId() ?: ""
        val sessionId = getOrCreateCheckoutSessionId()
        var checkoutRequest: UnifiedCheckoutRequest? = null
        try {
            var finalGateway = initialState.selectedPaymentGateway
            if (finalGateway != "GLOBAL_PAY" && finalGateway != "CASH" && finalGateway != "CARD") {
                try {
                    api.setDefaultCard(mapOf("token_id" to finalGateway))
                    finalGateway = if (initialState.catalogCodes.contains("GLOBAL_PAY")) "GLOBAL_PAY" else "CASH"
                } catch (e: Exception) {
                    _uiState.update {
                        it.copy(
                            checkoutPhase = CheckoutPhase.REVIEW,
                            checkoutError = resolveCheckoutErrorMessage(e, "Failed to select payment method"),
                        )
                    }
                    return
                }
            }

            val draftRequest = buildCheckoutRequest(initialState, retailerId, finalGateway, paymentMethodOverride)
            checkoutRequest = draftRequest
            val preview = api.checkoutPreview(draftRequest)
            applyPreviewCaps(preview)
            val refreshedState = _uiState.value
            if (preview.code == "order_acceptance_closed") {
                _uiState.update {
                    it.copy(
                        checkoutPhase = CheckoutPhase.REVIEW,
                        checkoutError = preview.message ?: "Orders are not accepted at this time.",
                    )
                }
                return
            }
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
                return
            }
            if (preview.stockWarnings.isNotEmpty() && !skipBackorderConfirm) {
                _uiState.update {
                    it.copy(
                        checkoutPhase = CheckoutPhase.REVIEW,
                        stockWarnings = preview.stockWarnings,
                        pendingBackorderConfirm = true,
                    )
                }
                return
            }

            val request = buildCheckoutRequest(refreshedState, retailerId, finalGateway, paymentMethodOverride)
            checkoutRequest = request
            val idempotencyKey = checkoutIdempotencyKey(request, sessionId)
            val response = api.unifiedCheckout(request, idempotencyKey)
            val firstOrderId = response.supplierOrders.firstOrNull()?.orderId
            _uiState.update {
                it.copy(
                    stockWarnings = response.stockWarnings,
                    lastOrderId = firstOrderId ?: response.invoiceId,
                    paymentUrlToOpen = null,
                    checkoutPhase = CheckoutPhase.COMPLETE,
                    packSuggestions = emptyList(),
                    ownerApprovalRequired = null,
                )
            }
            currentCheckoutSessionId = null
            delay(1800)
            _uiState.update {
                it.copy(
                    showCheckout = false,
                    checkoutPhase = CheckoutPhase.REVIEW,
                    items = emptyList(),
                    lastOrderId = null,
                )
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            handleCheckoutSubmitFailure(e, checkoutRequest, sessionId)
        }
    }

    private suspend fun handleCheckoutSubmitFailure(
        error: Exception,
        checkoutRequest: UnifiedCheckoutRequest?,
        sessionId: String,
    ) {
        val packFixes = OrderLineContractHelper.parsePackMultipleSuggestions(error)
        if (packFixes.isNotEmpty()) {
            _uiState.update {
                it.copy(
                    checkoutPhase = CheckoutPhase.REVIEW,
                    packSuggestions = packFixes,
                    checkoutError = packFixes.firstOrNull()?.message
                        ?: "Adjust item quantities to match pack size",
                )
            }
            return
        }

        val oosResult = extractOos409Result(error)
        if (oosResult != null) {
            refreshCheckoutPreview()
            _uiState.update {
                it.copy(
                    checkoutPhase = CheckoutPhase.REVIEW,
                    checkoutError = oosResult.first,
                    oosItems = oosResult.second,
                )
            }
            return
        }

        val ownerApproval = RetailerW2ContractHelper.parseOwnerApprovalRequired(error)
        val assessment = OrderLineContractHelper.assessPendingOrderError(error)
        val shouldQueue = checkoutRequest != null && (
            error is IOException ||
                ownerApproval != null ||
                assessment.kind == PendingOrderConflictKind.OWNER_APPROVAL_REQUIRED ||
                assessment.kind == PendingOrderConflictKind.REQUEST_IN_PROGRESS ||
                assessment.kind == PendingOrderConflictKind.RATE_LIMITED ||
                assessment.kind == PendingOrderConflictKind.UNAUTHORIZED
            )
        if (shouldQueue && checkoutRequest != null) {
            val status = when {
                ownerApproval != null || assessment.kind == PendingOrderConflictKind.OWNER_APPROVAL_REQUIRED ->
                    PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL
                assessment.kind == PendingOrderConflictKind.REQUEST_IN_PROGRESS -> PendingOrderEntity.STATUS_IN_PROGRESS
                assessment.kind == PendingOrderConflictKind.RATE_LIMITED -> PendingOrderEntity.STATUS_RATE_LIMITED
                assessment.kind == PendingOrderConflictKind.UNAUTHORIZED -> assessment.status
                else -> PendingOrderEntity.STATUS_PENDING
            }
            val message = if (error is IOException) {
                "Network issue during checkout. Saved for automatic retry when you reconnect."
            } else {
                ownerApproval?.message ?: assessment.message
            }
            val key = checkoutIdempotencyKey(checkoutRequest, sessionId)
            queuePendingCheckout(checkoutRequest, key, status = status, lastError = message)
            _uiState.update {
                it.copy(
                    checkoutPhase = CheckoutPhase.REVIEW,
                    checkoutError = message,
                    ownerApprovalRequired = ownerApproval,
                    pendingQueueNotice = PendingOrderStatusNotice(
                        orderId = 0L,
                        endpoint = "/v1/checkout/unified",
                        status = status,
                        message = message,
                        keptInQueue = true,
                    ),
                )
            }
            return
        }

        val fallbackMsg = when (error) {
            is ProblemDetailException -> error.problem.detail ?: error.problem.title ?: error.message ?: "Checkout failed"
            is HttpException -> RetailerW2ContractHelper.peekHttpErrorBody(error)
                ?.takeIf { it.isNotBlank() } ?: "Checkout failed (${error.code()})"
            else -> resolveCheckoutErrorMessage(error, "Checkout failed")
        }
        _uiState.update {
            it.copy(
                checkoutPhase = CheckoutPhase.REVIEW,
                checkoutError = fallbackMsg,
            )
        }
    }

    private fun extractOos409Result(error: Exception): Pair<String, List<String>>? {
        val status = when (error) {
            is ProblemDetailException -> error.problem.status
            is HttpException -> error.code()
            else -> return null
        }
        if (status != 409) return null

        val rawBody = when (error) {
            is ProblemDetailException -> error.problem.rawBody
            is HttpException -> RetailerW2ContractHelper.peekHttpErrorBody(error)
            else -> null
        }
        val problem = (error as? ProblemDetailException)?.problem
        val parsed = rawBody?.let {
            runCatching { OrderLineContractHelper.lenientJson.decodeFromString<Map<String, kotlinx.serialization.json.JsonElement>>(it) }.getOrNull()
        }
        val code = problem?.code
            ?: problem?.error
            ?: parsed?.get("code")?.jsonPrimitive?.contentOrNull
            ?: parsed?.get("error")?.jsonPrimitive?.contentOrNull
            ?: ""
        if (code.equals("owner_approval_required", ignoreCase = true) ||
            code.equals("pending_owner_approval", ignoreCase = true) ||
            code.equals("request_in_progress", ignoreCase = true)
        ) {
            return null
        }
        val flaggedOos = (
            problem?.oosItems.orEmpty() +
                (parsed?.get("oos_items") as? kotlinx.serialization.json.JsonArray)
                    ?.mapNotNull { it.jsonPrimitive.contentOrNull?.takeIf(String::isNotBlank) }
                    .orEmpty()
            ).distinct()
        if (code != "inventory_exhausted" &&
            code != "ALL_ITEMS_OUT_OF_STOCK" &&
            code != "PARTIAL_OUT_OF_STOCK_REJECTED" &&
            flaggedOos.isEmpty()
        ) {
            return null
        }
        val msg = when (code) {
            "inventory_exhausted" -> "Stock changed while checking out. Review your cart and try again."
            "ALL_ITEMS_OUT_OF_STOCK" -> "All items are out of stock. Please update your cart."
            "PARTIAL_OUT_OF_STOCK_REJECTED" -> "Some items are out of stock and cannot be backordered. Please update your cart."
            else -> problem?.detail ?: rawBody ?: "Some items are out of stock"
        }
        return msg to flaggedOos
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

    private fun checkoutIdempotencyKey(request: UnifiedCheckoutRequest, sessionId: String? = null): String {
        val itemKey = request.items
            .sortedBy { it.skuId }
            .joinToString("|") { "${it.skuId}:${it.quantity}:${it.unitPrice}:${it.mustHave}:${it.allowSubstitutes}" }
        val autoFlag = if (request.autoConfirmOptIn) ":auto" else ""
        val sid = sessionId ?: getOrCreateCheckoutSessionId()
        return "retailer-checkout:${request.paymentGateway}:$sid:$itemKey$autoFlag"
    }

    private suspend fun queuePendingCheckout(
        request: UnifiedCheckoutRequest,
        idempotencyKey: String,
        status: String = PendingOrderEntity.STATUS_PENDING,
        lastError: String? = null,
    ) {
        pendingOrderDao.insert(
            PendingOrderEntity(
                endpoint = "/v1/checkout/unified",
                method = "POST",
                payloadJson = Json.encodeToString(UnifiedCheckoutRequest.serializer(), request),
                idempotencyKey = idempotencyKey,
                lastError = lastError,
                status = status,
            ),
        )
        syncEnqueuer.enqueue()
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
        previewDebounceJob?.cancel()
    }
}
