package com.pegasus.retailer.ui.screens.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.local.TokenManager
import com.pegasus.retailer.data.model.DemandForecast
import com.pegasus.retailer.data.model.Order
import com.pegasus.retailer.data.model.Product
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class DashboardUiState(
    val isLoading: Boolean = false,
    val activeOrders: List<Order> = emptyList(),
    val predictions: List<DemandForecast> = emptyList(),
    val recentProducts: List<Product> = emptyList(),
    val error: String? = null,
)

@HiltViewModel
class DashboardViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(DashboardUiState())
    val uiState: StateFlow<DashboardUiState> = _uiState.asStateFlow()

    private val retailerId: String get() = tokenManager.getUserId() ?: ""

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val orders = api.getOrders(retailerId)
                val active = orders.filter { it.status.isActive }
                _uiState.update { it.copy(activeOrders = active) }
            } catch (e: Exception) {
                _uiState.update { it.copy(error = "Could not load orders. Pull to refresh.") }
            }

            try {
                val forecasts = api.getPredictions(retailerId)
                _uiState.update { it.copy(predictions = forecasts) }
            } catch (_: Exception) {}

            try {
                val products = api.getCatalogProducts()
                _uiState.update { it.copy(recentProducts = products.take(6)) }
            } catch (_: Exception) {}

            _uiState.update { it.copy(isLoading = false) }
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
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
}
