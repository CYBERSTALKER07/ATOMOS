package com.pegasus.retailer.ui.navigation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.api.RetailerWSMessage
import com.pegasus.retailer.data.api.RetailerWebSocket
import com.pegasus.retailer.data.local.TokenManager
import com.pegasus.retailer.data.model.CardCheckoutRequest
import com.pegasus.retailer.data.model.CashCheckoutRequest
import com.pegasus.retailer.data.model.formatRetailerAmount
import com.pegasus.retailer.data.model.Order
import com.pegasus.retailer.data.model.PendingPaymentSession
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CardCheckoutResult(
    val paymentUrl: String? = null,
    val sessionId: String? = null,
    val attemptId: String? = null,
    val attemptNo: Int = 0,
)

data class NavigationUiState(
    val activeOrders: List<Order> = emptyList(),
    val approachingOrderIds: Set<String> = emptySet(),
    val userName: String = "",
    val companyName: String = "",
    val avatarInitial: String = "?",
    val paymentEvent: RetailerWSMessage? = null,
    val orderCompleted: Boolean = false,
    val paymentFailed: Boolean = false,
    val paymentFailureMessage: String = "",
) {
    val activeOrderCount: Int get() = activeOrders.size
    val floatingStatusText: String
        get() = activeOrders.firstOrNull()?.status?.displayName ?: ""
    val floatingTotalDisplay: String
        get() {
            val total = activeOrders.sumOf { it.totalAmount }
            val currency = activeOrders.firstOrNull()?.currency ?: "UZS"
            return if (total > 0) formatRetailerAmount(total, currency) else ""
        }
    val floatingCountdownIso: String?
        get() = activeOrders.firstOrNull()?.estimatedDelivery
}

private fun PendingPaymentSession.toRetailerPaymentEvent(): RetailerWSMessage {
    val normalizedGateway = gateway.ifBlank { "GLOBAL_PAY" }
    return RetailerWSMessage(
        type = "PAYMENT_REQUIRED",
        orderId = orderId,
        invoiceId = invoiceId ?: "",
        sessionId = sessionId,
        amount = lockedAmount,
        originalAmount = lockedAmount,
        availableCardGateways = if (normalizedGateway == "CASH") emptyList() else listOf(normalizedGateway),
        message = "Pending payment requires completion.",
        paymentMethod = if (normalizedGateway == "CASH") "CASH" else "CARD",
        gateway = normalizedGateway,
    )
}

@HiltViewModel
class NavigationViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
) : ViewModel() {

    private val _uiState = MutableStateFlow(NavigationUiState())
    val uiState: StateFlow<NavigationUiState> = _uiState.asStateFlow()

    init {
        val name = tokenManager.getUserName() ?: ""
        _uiState.update {
            it.copy(
                userName = name,
                companyName = "Pegasus",
                avatarInitial = name.firstOrNull()?.uppercase() ?: "?",
            )
        }
        loadActiveOrders()
        loadPendingPayments()
        connectWebSocket()
    }

    fun loadActiveOrders() {
        viewModelScope.launch {
            try {
                val rid = tokenManager.getUserId() ?: return@launch
                val orders = api.getOrders(rid)
                val active = orders.filter { it.status.isActive }
                _uiState.update { it.copy(activeOrders = active) }
            } catch (_: Exception) {
                _uiState.update { it.copy(activeOrders = emptyList()) }
            }
        }
    }

    fun clearPaymentEvent() {
        _uiState.update { it.copy(paymentEvent = null, orderCompleted = false, paymentFailed = false, paymentFailureMessage = "") }
        loadActiveOrders()
        loadPendingPayments()
    }

    fun loadPendingPayments() {
        viewModelScope.launch {
            try {
                val response = api.getPendingPayments()
                val pending = response.pendingPayments.firstOrNull()
                if (pending != null) {
                    _uiState.update { it.copy(paymentEvent = pending.toRetailerPaymentEvent()) }
                }
            } catch (_: Exception) {
                // WebSocket delivery remains the primary realtime path.
            }
        }
    }

    suspend fun cashCheckout(orderId: String): Result<Unit> {
        return try {
            api.cashCheckout(CashCheckoutRequest(orderId = orderId), "retailer-cash-checkout:$orderId")
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun cardCheckout(orderId: String, gateway: String): Result<CardCheckoutResult> {
        return try {
            val resp = api.cardCheckout(
                CardCheckoutRequest(orderId = orderId, gateway = gateway),
                "retailer-card-checkout:$orderId:$gateway",
            )
            val result = CardCheckoutResult(
                paymentUrl = resp.paymentUrl,
                sessionId = resp.sessionId,
                attemptId = resp.attemptId,
                attemptNo = resp.attemptNo ?: 0,
            )
            Result.success(result)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun connectWebSocket() {
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "PAYMENT_REQUIRED" ||
                        it.type == "GLOBAL_PAYNT_REQUIRED" ||
                        it.type == "SETTLEMENT_REQUIRED"
                }
                .collect { msg ->
                    _uiState.update { it.copy(paymentEvent = msg) }
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "DELIVERY_SESSION_UPDATED" }
                .collect { msg ->
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        val updatedAmount = if (msg.adjustedAmount > 0) msg.adjustedAmount else current.amount
                        val updatedOriginalAmount = when {
                            msg.originalAmount > 0 -> msg.originalAmount
                            current.originalAmount > 0 -> current.originalAmount
                            else -> updatedAmount
                        }
                        val updatedState = if (msg.state.isNotBlank()) msg.state else current.state
                        _uiState.update {
                            it.copy(
                                paymentEvent = current.copy(
                                    amount = updatedAmount,
                                    originalAmount = updatedOriginalAmount,
                                    state = updatedState,
                                ),
                            )
                        }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "ORDER_COMPLETED" }
                .collect { msg ->
                    // If this completion matches the active payment event, signal success
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update { it.copy(orderCompleted = true) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "DRIVER_APPROACHING" }
                .collect { msg ->
                    if (msg.orderId.isNotBlank()) {
                        _uiState.update { it.copy(
                            approachingOrderIds = it.approachingOrderIds + msg.orderId
                        ) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "PAYMENT_SETTLED" || it.type == "GLOBAL_PAYNT_SETTLED" }
                .collect { msg ->
                    // If this settlement matches the active payment event, signal success
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update { it.copy(orderCompleted = true) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "PAYMENT_FAILED" ||
                        it.type == "PAYMENT_EXPIRED" ||
                        it.type == "GLOBAL_PAYNT_FAILED" ||
                        it.type == "GLOBAL_PAYNT_EXPIRED"
                }
                .collect { msg ->
                    // If this failure/expiry matches the active payment event, signal failure
                    val current = _uiState.value.paymentEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update {
                            it.copy(
                                paymentFailed = true,
                                paymentFailureMessage = msg.message.ifBlank {
                                    if (msg.type == "PAYMENT_EXPIRED" || msg.type == "GLOBAL_PAYNT_EXPIRED") {
                                        "Payment session expired"
                                    } else {
                                        "Payment failed"
                                    }
                                },
                            )
                        }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "ORDER_STATUS_CHANGED" ||
                        it.type == "ORDER_AMENDED" ||
                        it.type == "ORDER_REASSIGNED"
                }
                .collect { loadActiveOrders() }
        }
    }

    override fun onCleared() {
        super.onCleared()
        retailerWebSocket.disconnect()
    }
}
