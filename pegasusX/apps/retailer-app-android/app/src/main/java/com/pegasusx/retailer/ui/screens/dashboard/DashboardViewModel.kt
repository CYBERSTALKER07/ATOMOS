package com.pegasusx.retailer.ui.screens.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.Product
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
    val predictions: List<DemandForecast> = emptyList(),
    val recentProducts: List<Product> = emptyList(),
    val error: String? = null,
    val loadIssue: DashboardLoadIssue? = null,
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
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

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
                val forecasts = api.getPredictions(retailerId)
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
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
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
}
