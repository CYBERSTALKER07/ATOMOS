package com.pegasusx.retailer.ui.screens.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasus.design.MarketPack
import com.pegasus.design.MarketPackBinder
import com.pegasus.design.PulseHonesty
import com.pegasusx.retailer.BuildConfig
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.ControlTowerPulse
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.PulseEvent
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class DashboardLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class DashboardUiState(
    val isLoading: Boolean = false,
    val activeOrders: List<Order> = emptyList(),
    val predictions: List<RetailerAIPrediction> = emptyList(),
    val recentProducts: List<Product> = emptyList(),
    val pulseEvents: List<PulseEvent> = emptyList(),
    val pulseLoading: Boolean = false,
    val pulseError: String? = null,
    val commandPulse: ControlTowerPulse? = null,
    val commandPulseError: String? = null,
    val error: String? = null,
    val loadIssue: DashboardLoadIssue? = null,
    val orderActionPending: Boolean = false,
    val pack: MarketPack? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            DashboardLoadIssue.RESTRICTED -> "Dashboard access is restricted for this account"
            DashboardLoadIssue.OFFLINE -> "Offline mode active. Showing latest dashboard data"
            DashboardLoadIssue.ERROR -> "Dashboard sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class DashboardViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
) : ViewModel() {

    private val _uiState = MutableStateFlow(DashboardUiState())
    val uiState: StateFlow<DashboardUiState> = _uiState.asStateFlow()

    private val retailerId: String get() = tokenManager.getUserId() ?: ""

    init {
        refresh()
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "PROMOTION_CHANGED" }
                .collect { refresh() }
        }
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
                }
                refresh()
            }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, pulseLoading = true, pulseError = null, error = null) }
            val pack = MarketPackBinder.fetch(BuildConfig.BASE_URL, tokenManager.getPreferredToken().orEmpty())?.pack
            _uiState.update { it.copy(pack = pack) }

            var nextIssue: DashboardLoadIssue? = null
            var nextError: String? = null

            try {
                val orders = api.getOrders(retailerId)
                val active = orders.filter { it.status.isActive }
                _uiState.update { it.copy(activeOrders = active) }
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            try {
                val forecasts = api.getRetailerAIPredictions().items
                _uiState.update { it.copy(predictions = forecasts) }
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            try {
                val products = api.getCatalogProducts(retailerId = retailerId.takeIf { it.isNotBlank() })
                _uiState.update { it.copy(recentProducts = products.take(6)) }
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            try {
                val pulse = api.getRetailerPulse()
                _uiState.update { it.copy(pulseEvents = pulse.events, pulseLoading = false, pulseError = null) }
            } catch (_: Exception) {
                _uiState.update { it.copy(pulseLoading = false, pulseError = PulseHonesty.FAILED) }
            }

            try {
                val command = api.getControlTowerPulse()
                val applied = PulseHonesty.applyObject(
                    ok = true,
                    incoming = command,
                    previous = _uiState.value.commandPulse,
                )
                _uiState.update {
                    it.copy(commandPulse = applied.value, commandPulseError = applied.error)
                }
            } catch (_: Exception) {
                val applied = PulseHonesty.applyObject(
                    ok = false,
                    incoming = null,
                    previous = _uiState.value.commandPulse,
                )
                _uiState.update {
                    it.copy(commandPulse = applied.value, commandPulseError = applied.error)
                }
            }

            _uiState.update {
                it.copy(
                    isLoading = false,
                    loadIssue = nextIssue,
                    error = nextError,
                )
            }
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null, loadIssue = null) }
    }

    fun confirmAiOrder(orderId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.confirmAiOrder(
                    body = mapOf("order_id" to orderId),
                    idempotencyKey = RetailerIdempotencyKeys.confirmAI(orderId),
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

    fun rejectAiOrder(orderId: String, reason: String = "Retailer rejected") {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.rejectAiOrder(
                    body = mapOf("order_id" to orderId, "reason" to reason),
                    idempotencyKey = RetailerIdempotencyKeys.rejectAI(orderId, reason),
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

    fun confirmPreorder(orderId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(orderActionPending = true, error = null) }
            try {
                api.confirmPreorder(
                    body = mapOf("order_id" to orderId),
                    idempotencyKey = "retailer-confirm-preorder:$orderId",
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

    private fun resolveLoadIssue(error: Exception): DashboardLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> DashboardLoadIssue.RESTRICTED
            error is IOException -> DashboardLoadIssue.OFFLINE
            else -> DashboardLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: DashboardLoadIssue): String {
        return when (issue) {
            DashboardLoadIssue.RESTRICTED -> "Dashboard access is restricted for this account"
            DashboardLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            DashboardLoadIssue.ERROR -> error.message ?: "Dashboard request failed"
        }
    }

    private companion object {
        const val ORDER_RECONNECT_RECOVERY_HINT =
            "Connection restored — verify order status before retrying."
    }
}
