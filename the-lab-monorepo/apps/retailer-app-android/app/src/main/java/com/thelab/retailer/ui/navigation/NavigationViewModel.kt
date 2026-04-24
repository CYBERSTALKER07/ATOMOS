package com.thelab.retailer.ui.navigation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.retailer.data.api.LabApi
import com.thelab.retailer.data.api.RetailerWSMessage
import com.thelab.retailer.data.api.RetailerWebSocket
import com.thelab.retailer.data.local.TokenManager
import com.thelab.retailer.data.model.Order
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CardCheckoutResult(
    val global_payntUrl: String? = null,
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
    val global_payntEvent: RetailerWSMessage? = null,
    val orderCompleted: Boolean = false,
    val global_payntFailed: Boolean = false,
    val global_payntFailureMessage: String = "",
) {
    val activeOrderCount: Int get() = activeOrders.size
    val floatingStatusText: String
        get() = activeOrders.firstOrNull()?.status?.displayName ?: ""
    val floatingTotalDisplay: String
        get() {
            val total = activeOrders.sumOf { it.totalAmount }
            return if (total > 0) String.format("$%.2f", total) else ""
        }
    val floatingCountdownIso: String?
        get() = activeOrders.firstOrNull()?.estimatedDelivery
}

@HiltViewModel
class NavigationViewModel @Inject constructor(
    private val api: LabApi,
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
                companyName = "The Lab Industries",
                avatarInitial = name.firstOrNull()?.uppercase() ?: "?",
            )
        }
        loadActiveOrders()
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

    fun clearGlobalPayntEvent() {
        _uiState.update { it.copy(global_payntEvent = null, orderCompleted = false, global_payntFailed = false, global_payntFailureMessage = "") }
        loadActiveOrders()
    }

    suspend fun cashCheckout(orderId: String): Result<Unit> {
        return try {
            api.cashCheckout(mapOf("order_id" to orderId))
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun cardCheckout(orderId: String, gateway: String): Result<CardCheckoutResult> {
        return try {
            val resp = api.cardCheckout(mapOf("order_id" to orderId, "gateway" to gateway))
            val result = CardCheckoutResult(
                global_payntUrl = resp["global_paynt_url"] as? String,
                sessionId = resp["session_id"] as? String,
                attemptId = resp["attempt_id"] as? String,
                attemptNo = (resp["attempt_no"] as? Number)?.toInt() ?: 0,
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
                .filter { it.type == "GLOBAL_PAYNT_REQUIRED" }
                .collect { msg ->
                    _uiState.update { it.copy(global_payntEvent = msg) }
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "ORDER_COMPLETED" }
                .collect { msg ->
                    // If this completion matches the active global_paynt event, signal success
                    val current = _uiState.value.global_payntEvent
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
                .filter { it.type == "GLOBAL_PAYNT_SETTLED" }
                .collect { msg ->
                    // If this settlement matches the active global_paynt event, signal success
                    val current = _uiState.value.global_payntEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update { it.copy(orderCompleted = true) }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "GLOBAL_PAYNT_FAILED" || it.type == "GLOBAL_PAYNT_EXPIRED" }
                .collect { msg ->
                    // If this failure/expiry matches the active global_paynt event, signal failure
                    val current = _uiState.value.global_payntEvent
                    if (current != null && current.orderId == msg.orderId) {
                        _uiState.update {
                            it.copy(
                                global_payntFailed = true,
                                global_payntFailureMessage = msg.message.ifBlank {
                                    if (msg.type == "GLOBAL_PAYNT_EXPIRED") "GlobalPaynt session expired" else "GlobalPaynt failed"
                                },
                            )
                        }
                    }
                    loadActiveOrders()
                }
        }
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "ORDER_STATUS_CHANGED" || it.type == "ORDER_AMENDED" }
                .collect { loadActiveOrders() }
        }
    }

    override fun onCleared() {
        super.onCleared()
        retailerWebSocket.disconnect()
    }
}
