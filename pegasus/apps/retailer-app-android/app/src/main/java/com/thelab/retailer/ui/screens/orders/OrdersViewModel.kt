package com.pegasus.retailer.ui.screens.orders

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.LabApi
import com.pegasus.retailer.data.api.RetailerWebSocket
import com.pegasus.retailer.data.local.TokenManager
import com.pegasus.retailer.data.model.DemandForecast
import com.pegasus.retailer.data.model.Order
import com.pegasus.retailer.data.model.OrderStatus
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class OrdersUiState(
    val isLoading: Boolean = false,
    val allOrders: List<Order> = emptyList(),
    val predictions: List<DemandForecast> = emptyList(),
    val error: String? = null,
) {
    val activeOrders: List<Order> get() = allOrders.filter {
        it.status == OrderStatus.LOADED || it.status == OrderStatus.DISPATCHED || it.status == OrderStatus.IN_TRANSIT || it.status == OrderStatus.ARRIVED
    }
    val pendingOrders: List<Order> get() = allOrders.filter {
        it.status == OrderStatus.PENDING
    }
}

@HiltViewModel
class OrdersViewModel @Inject constructor(
    private val api: LabApi,
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
    }

    private fun connectWebSocket() {
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events.collect { msg ->
                when (msg.type) {
                    "ORDER_STATUS_CHANGED", "ORDER_DISPATCHED", "ORDER_DELIVERED",
                    "ORDER_ARRIVING", "DELIVERY_TOKEN",
                    "GLOBAL_PAYNT_REQUIRED", "GLOBAL_PAYNT_SETTLED", "GLOBAL_PAYNT_FAILED",
                    "GLOBAL_PAYNT_EXPIRED", "ORDER_AMENDED", "ORDER_COMPLETED",
                    "PRE_ORDER_AUTO_ACCEPTED", "PRE_ORDER_CONFIRMED", "PRE_ORDER_EDITED" -> refresh()
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
            _uiState.update { it.copy(isLoading = true) }
            try {
                val orders = api.getOrders(retailerId)
                _uiState.update { it.copy(allOrders = orders) }
            } catch (_: Exception) {}

            try {
                val forecasts = api.getPredictions(retailerId)
                _uiState.update { it.copy(predictions = forecasts) }
            } catch (_: Exception) {}

            _uiState.update { it.copy(isLoading = false) }
        }
    }

    fun cancelOrder(orderId: String) {
        if (!cancellingIds.add(orderId)) return
        viewModelScope.launch {
            try {
                api.cancelOrder(mapOf("order_id" to orderId, "retailer_id" to retailerId))
                refresh()
            } catch (_: Exception) {
            } finally {
                cancellingIds.remove(orderId)
            }
        }
    }

    fun requestPreorder(forecast: DemandForecast) {
        viewModelScope.launch {
            try {
                api.aiPreorder(
                    mapOf(
                        "product_id" to forecast.productId,
                        "quantity" to forecast.predictedQuantity,
                    ),
                )
                refresh()
            } catch (_: Exception) {}
        }
    }

    fun correctPrediction(predictionId: String, amount: Long) {
        viewModelScope.launch {
            try {
                api.correctPrediction(
                    predictionId = predictionId,
                    body = mapOf("amount" to amount),
                )
                refresh()
            } catch (_: Exception) {}
        }
    }

    fun rejectPrediction(predictionId: String) {
        viewModelScope.launch {
            try {
                api.correctPrediction(
                    predictionId = predictionId,
                    body = mapOf("status" to "REJECTED"),
                )
                refresh()
            } catch (_: Exception) {}
        }
    }
}
