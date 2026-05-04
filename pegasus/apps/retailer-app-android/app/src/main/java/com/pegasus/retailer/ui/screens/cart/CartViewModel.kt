package com.pegasus.retailer.ui.screens.cart

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.api.RetailerWebSocket
import com.pegasus.retailer.data.local.PendingOrderDao
import com.pegasus.retailer.data.local.PendingOrderEntity
import com.pegasus.retailer.data.local.TokenManager
import com.pegasus.retailer.data.model.CartItem
import com.pegasus.retailer.data.model.CheckoutLineItem
import com.pegasus.retailer.data.model.Product
import com.pegasus.retailer.data.model.UnifiedCheckoutRequest
import com.pegasus.retailer.data.model.Variant
import com.pegasus.retailer.ui.components.CheckoutPhase
import retrofit2.HttpException
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import javax.inject.Inject

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
) {
    val isEmpty: Boolean get() = items.isEmpty()
    val totalItems: Int get() = items.sumOf { it.quantity }
    val subtotal: Double get() = items.sumOf { it.totalPrice }
    val shipping: Double get() = if (subtotal > 50_000) 0.0 else 15_000.0
    val discount: Double get() = if (subtotal > 500_000) subtotal * 0.05 else 0.0
    val total: Double get() = subtotal + shipping - discount
    val displaySubtotal: String get() = "%,.0f".format(subtotal)
    val displayShipping: String get() = if (shipping == 0.0) "Free" else "%,.0f".format(shipping)
    val displayDiscount: String get() = if (discount == 0.0) "0" else "-%,.0f".format(discount)
    val displayTotal: String get() = "%,.0f".format(total)
    val firstProductName: String get() = items.firstOrNull()?.product?.name ?: "Order"
    val selectedPaymentLabel: String get() = checkoutPaymentLabel(selectedPaymentGateway)
}

private fun checkoutPaymentLabel(gateway: String): String {
    return when (gateway.trim().uppercase()) {
        
        "GLOBAL_PAY" -> "GlobalPay"
        "GLOBAL_PAY" -> "GlobalPay"
        "CASH" -> "Cash on Delivery"
        else -> "Cash"
    }
}

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

    init { flushPendingOrders() }

    private fun flushPendingOrders() = viewModelScope.launch {
        val pending = pendingOrderDao.getAll()
        for (order in pending) {
            try {
                val request = Json.decodeFromString<UnifiedCheckoutRequest>(order.payloadJson)
                api.unifiedCheckout(request, order.idempotencyKey)
                pendingOrderDao.deleteById(order.id)
            } catch (e: Exception) {
                pendingOrderDao.incrementRetry(order.id, e.message ?: e::class.java.simpleName)
            }
        }
    }

    fun addToCart(product: Product, variant: Variant) {
        val itemId = "${product.id}_${variant.id}"
        _uiState.update { state ->
            val existing = state.items.find { it.id == itemId }
            if (existing != null) {
                state.copy(items = state.items.map { if (it.id == itemId) it.copy(quantity = it.quantity + 1) else it })
            } else {
                state.copy(items = state.items + CartItem(id = itemId, product = product, variant = variant, quantity = 1))
            }
        }
    }

    fun updateQuantity(itemId: String, quantity: Int) {
        if (quantity <= 0) {
            removeItem(itemId)
            return
        }
        _uiState.update { state ->
            state.copy(items = state.items.map { if (it.id == itemId) it.copy(quantity = quantity) else it })
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
    }

    fun clearRemovedItemMessage() {
        _uiState.update { it.copy(removedItemMessage = null) }
    }

    fun clearCart() {
        _uiState.update { it.copy(items = emptyList()) }
    }

    fun showCheckout() {
        _uiState.update { it.copy(showCheckout = true, checkoutPhase = CheckoutPhase.REVIEW) }
    }

    fun setSupplierIsActive(value: Boolean) {
        _uiState.update { it.copy(supplierIsActive = value) }
    }

    fun dismissCheckout() {
        _uiState.update { it.copy(showCheckout = false, checkoutPhase = CheckoutPhase.REVIEW) }
    }

    fun setSelectedPaymentGateway(gateway: String) {
        _uiState.update { it.copy(selectedPaymentGateway = gateway.trim().uppercase()) }
    }

    fun processPayment() {
        viewModelScope.launch {
            _uiState.update { it.copy(checkoutPhase = CheckoutPhase.PROCESSING, checkoutError = null) }
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
                val request = UnifiedCheckoutRequest(
                    retailerId = retailerId,
                    paymentGateway = state.selectedPaymentGateway,
                    items = lineItems,
                )
                val response = api.unifiedCheckout(request, checkoutIdempotencyKey(request))
                val invoiceId = response.invoiceId
                val firstOrderId = response.supplierOrders.firstOrNull()?.orderId
                _uiState.update { it.copy(lastOrderId = firstOrderId ?: invoiceId) }

                // Transition to COMPLETE immediately — the backend already committed the order.
                // WebSocket GLOBAL_PAYNT_SETTLED can be layered in later for real-time settlement tracking.
                _uiState.update { it.copy(checkoutPhase = CheckoutPhase.COMPLETE) }
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
                        msg = if (code == "ALL_ITEMS_OUT_OF_STOCK") {
                            "All items are out of stock. Please update your cart."
                        } else {
                            body ?: "Some items are out of stock"
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
                        checkoutError = e.message ?: "Checkout failed",
                    )
                }
            }
        }
    }

    fun clearCheckoutError() {
        _uiState.update { it.copy(checkoutError = null) }
    }

    private fun checkoutIdempotencyKey(request: UnifiedCheckoutRequest): String {
        val itemKey = request.items
            .sortedBy { it.skuId }
            .joinToString("|") { "${it.skuId}:${it.quantity}:${it.unitPrice}" }
        return "retailer-checkout:${request.paymentGateway}:$itemKey"
    }

    override fun onCleared() {
        super.onCleared()
        paymentListenerJob?.cancel()
    }
}
