package com.pegasusx.retailer.ui.screens.orders

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderStatus
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class OrdersLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class OrdersUiState(
    val isLoading: Boolean = false,
    val allOrders: List<Order> = emptyList(),
    val predictions: List<DemandForecast> = emptyList(),
    val error: String? = null,
    val loadIssue: OrdersLoadIssue? = null,
    val orderActionPending: Boolean = false,
) {
    val activeOrders: List<Order> get() = allOrders.filter {
        it.status == OrderStatus.LOADED || it.status == OrderStatus.DISPATCHED || it.status == OrderStatus.IN_TRANSIT || it.status == OrderStatus.ARRIVED
    }
    val pendingOrders: List<Order> get() = allOrders.filter {
        it.status == OrderStatus.PENDING ||
            it.status == OrderStatus.PENDING_REVIEW ||
            it.status == OrderStatus.SCHEDULED
    }
    val aiPendingOrders: List<Order> get() = allOrders.filter { it.needsAiConfirmation }
    val scheduledPreorders: List<Order> get() = allOrders.filter { it.needsPreorderAction }

    val syncMessage: String?
        get() = when (loadIssue) {
            OrdersLoadIssue.RESTRICTED -> "Orders access is restricted for this account"
            OrdersLoadIssue.OFFLINE -> "Offline mode active. Showing latest cached orders"
            OrdersLoadIssue.ERROR -> "Orders sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class OrdersViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
) : ViewModel() {

    private val _uiState = MutableStateFlow(OrdersUiState())
    val uiState: StateFlow<OrdersUiState> = _uiState.asStateFlow()

    private val retailerId: String get() = tokenManager.getUserId() ?: ""
    private val cancellingIds = mutableSetOf<String>()

    init {
        refresh()
        connectWebSocket()
        viewModelScope.launch {
            retailerWebSocket.reconnects.collect {
                if (_uiState.value.orderActionPending) {
                    _uiState.update {
                        it.copy(
                            orderActionPending = false,
                            error = ORDER_RECONNECT_RECOVERY_HINT,
                            loadIssue = null,
                        )
                    }
                    refresh()
                }
            }
        }
    }

    private fun connectWebSocket() {
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events.collect { msg ->
                when (msg.type) {
                    "ORDER_STATUS_CHANGED", "ORDER_DISPATCHED", "ORDER_DELIVERED",
                    "ORDER_ARRIVING", "DELIVERY_TOKEN",
                    "PAYMENT_REQUIRED", "PAYMENT_SETTLED", "PAYMENT_FAILED",
                    "PAYMENT_EXPIRED",
                    "SETTLEMENT_REQUIRED", "DELIVERY_SESSION_UPDATED",
                    "GLOBAL_PAYNT_REQUIRED", "GLOBAL_PAYNT_SETTLED", "GLOBAL_PAYNT_FAILED",
                    "GLOBAL_PAYNT_EXPIRED", "ORDER_AMENDED", "ORDER_COMPLETED", "ORDER_REASSIGNED",
                    "PRE_ORDER_AUTO_ACCEPTED", "PRE_ORDER_CONFIRMED", "PRE_ORDER_EDITED",
                    "PRE_ORDER_DATE_PROPOSED", "PRE_ORDER_DATE_ACCEPTED", "PRE_ORDER_DATE_REJECTED",
                    "PRE_ORDER_CANCELLED", "PRE_ORDER_NUDGE", "PRE_ORDER_CONFIRMATION" -> refresh()
                }
            }
        }
    }

    override fun onCleared() {
        super.onCleared()
        retailerWebSocket.disconnect()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            var nextOrders = _uiState.value.allOrders
            var nextPredictions = _uiState.value.predictions
            var nextIssue: OrdersLoadIssue? = null
            var nextErrorMessage: String? = null

            try {
                nextOrders = api.getOrders(retailerId)
            } catch (e: Exception) {
                nextIssue = resolveLoadIssue(e)
                nextErrorMessage = resolveErrorMessage(e, nextIssue)
            }

            try {
                nextPredictions = api.getPredictions(retailerId)
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextErrorMessage = resolveErrorMessage(e, nextIssue)
                }
            }

            val hasData = nextOrders.isNotEmpty() || nextPredictions.isNotEmpty()

            _uiState.update {
                it.copy(
                    isLoading = false,
                    allOrders = nextOrders,
                    predictions = nextPredictions,
                    loadIssue = nextIssue,
                    error = if (nextIssue != null && !hasData) nextErrorMessage else null,
                )
            }
        }
    }

    fun cancelOrder(orderId: String, status: OrderStatus? = null) {
        if (!cancellingIds.add(orderId)) return
        viewModelScope.launch {
            try {
                val body = mapOf("order_id" to orderId, "retailer_id" to retailerId)
                val idempotencyKey = RetailerIdempotencyKeys.cancel(orderId)
                try {
                    api.cancelOrder(body = body, idempotencyKey = idempotencyKey)
                } catch (first: Exception) {
                    val shouldRequestCancel = status?.let {
                        it == OrderStatus.DISPATCHED ||
                            it == OrderStatus.IN_TRANSIT ||
                            it == OrderStatus.ARRIVED
                    } == true
                    if (!shouldRequestCancel) throw first
                    api.requestCancelOrder(
                        body = body + mapOf("reason" to "Retailer requested cancellation"),
                        idempotencyKey = "retailer-request-cancel:$orderId",
                    )
                }
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                cancellingIds.remove(orderId)
            }
        }
    }

    fun requestPreorder(forecast: DemandForecast) {
        // AI pre-orders removed from PegasusX retailer apps.
        _uiState.update { it.copy(error = "AI pre-orders are not available in this app") }
    }

    fun correctPrediction(predictionId: String, amount: Long) {
        viewModelScope.launch {
            try {
                api.correctPrediction(
                    predictionId = predictionId,
                    body = mapOf("amount" to amount),
                    idempotencyKey = "retailer-prediction-correct:$predictionId:amount:$amount",
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            }
        }
    }

    fun rejectPrediction(predictionId: String) {
        viewModelScope.launch {
            try {
                api.correctPrediction(
                    predictionId = predictionId,
                    body = mapOf("status" to "REJECTED"),
                    idempotencyKey = "retailer-prediction-correct:$predictionId:rejected",
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            }
        }
    }

    fun confirmAiOrder(orderId: String) {
        _uiState.update { it.copy(error = "AI orders are not available") }
    }

    fun rejectAiOrder(orderId: String, reason: String = "Retailer rejected") {
        _uiState.update { it.copy(error = "AI orders are not available") }
    }

    fun confirmPreorder(orderId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.confirmPreorder(
                    body = mapOf("order_id" to orderId),
                    idempotencyKey = RetailerIdempotencyKeys.confirmPreorder(orderId),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun acceptDeliveryProposal(orderId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.acceptDeliveryProposal(
                    body = mapOf("order_id" to orderId),
                    idempotencyKey = RetailerIdempotencyKeys.acceptDeliveryProposal(orderId),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun rejectDeliveryProposal(orderId: String, reason: String = "Retailer rejected proposed date") {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.rejectDeliveryProposal(
                    body = mapOf("order_id" to orderId, "reason" to reason),
                    idempotencyKey = RetailerIdempotencyKeys.rejectDeliveryProposal(orderId, reason),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    fun editPreorder(order: Order, requestedDeliveryDate: String? = null) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                val deliveryDate = requestedDeliveryDate
                    ?: order.deliverBefore
                    ?: order.autoConfirmAt
                    ?: ""
                val lineItems = order.items.map { item ->
                    mapOf(
                        "sku" to item.productId.ifBlank { item.id },
                        "name" to item.productName,
                        "quantity" to item.quantity,
                        "unit_price_minor" to item.unitPrice.toLong(),
                    )
                }
                api.editPreorder(
                    body = mapOf(
                        "order_id" to order.id,
                        "line_items" to lineItems,
                        "requested_delivery_date" to deliveryDate,
                    ),
                    idempotencyKey = RetailerIdempotencyKeys.editPreorder(order.id),
                )
                refresh()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue)
                }
            } finally {
                _uiState.update { it.copy(orderActionPending = false) }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): OrdersLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> OrdersLoadIssue.RESTRICTED
            error is IOException -> OrdersLoadIssue.OFFLINE
            else -> OrdersLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: OrdersLoadIssue): String {
        return when (issue) {
            OrdersLoadIssue.RESTRICTED -> "Orders access is restricted for this account"
            OrdersLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            OrdersLoadIssue.ERROR -> error.message ?: "Orders request failed"
        }
    }

    private companion object {
        const val ORDER_RECONNECT_RECOVERY_HINT =
            "Connection restored — verify order status before retrying."
    }
}
