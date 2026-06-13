package com.pegasusx.retailer.ui.screens.procurement

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.ProcurementOrderItem
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.Product
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import retrofit2.HttpException

enum class ProcurementLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class ProcurementUiState(
    val isLoading: Boolean = true,
    val forecasts: List<DemandForecast> = emptyList(),
    val products: List<Product> = emptyList(),
    val selectedIds: Set<String> = emptySet(),
    val quantities: Map<String, Int> = emptyMap(),
    val isSubmitting: Boolean = false,
    val showSuccess: Boolean = false,
    val submitError: String? = null,
    val loadIssue: ProcurementLoadIssue? = null,
) {
    val selectedCount: Int get() = selectedIds.size
    val selectedUnits: Int get() = selectedIds.sumOf { quantities[it] ?: 0 }
    val syncMessage: String?
        get() = when (loadIssue) {
            ProcurementLoadIssue.RESTRICTED -> "Procurement access is restricted for this account"
            ProcurementLoadIssue.OFFLINE -> "Offline mode active. Showing latest suggestions"
            ProcurementLoadIssue.ERROR -> "Procurement sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class ProcurementViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val pendingOrderDao: PendingOrderDao,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ProcurementUiState())
    val uiState: StateFlow<ProcurementUiState> = _uiState.asStateFlow()

    init {
        loadPredictions()
    }

    fun loadPredictions() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, submitError = null) }
            val retailerId = tokenManager.getUserId().orEmpty()
            var issue: ProcurementLoadIssue? = null

            val forecasts = try {
                api.getPredictions(retailerId)
            } catch (e: Exception) {
                issue = resolveLoadIssue(e)
                emptyList()
            }

            val products = try {
                api.getCatalogProducts(retailerId = retailerId.takeIf { it.isNotBlank() })
            } catch (e: Exception) {
                if (issue == null) {
                    issue = resolveLoadIssue(e)
                }
                _uiState.value.products
            }

            _uiState.update {
                it.copy(
                    isLoading = false,
                    forecasts = forecasts,
                    products = products,
                    loadIssue = issue,
                )
            }
        }
    }

    fun toggleSelection(forecast: DemandForecast) {
        _uiState.update { state ->
            val nextSelected = state.selectedIds.toMutableSet()
            val nextQuantities = state.quantities.toMutableMap()
            if (nextSelected.contains(forecast.id)) {
                nextSelected.remove(forecast.id)
                nextQuantities.remove(forecast.id)
            } else {
                nextSelected.add(forecast.id)
                nextQuantities[forecast.id] = forecast.predictedQuantity
            }
            state.copy(selectedIds = nextSelected, quantities = nextQuantities)
        }
    }

    fun toggleSelectAll() {
        _uiState.update { state ->
            if (state.selectedIds.size == state.forecasts.size) {
                state.copy(selectedIds = emptySet(), quantities = emptyMap())
            } else {
                val quantities = state.forecasts.associate { it.id to it.predictedQuantity }
                state.copy(
                    selectedIds = state.forecasts.map { it.id }.toSet(),
                    quantities = quantities,
                )
            }
        }
    }

    fun updateQuantity(forecastId: String, quantity: Int) {
        if (quantity <= 0) return
        _uiState.update { state ->
            state.copy(quantities = state.quantities + (forecastId to quantity))
        }
    }

    fun dismissSuccess() {
        _uiState.update { it.copy(showSuccess = false, selectedIds = emptySet(), quantities = emptyMap()) }
    }

    fun clearSelections() {
        _uiState.update { it.copy(selectedIds = emptySet(), quantities = emptyMap()) }
    }

    fun clearSubmitError() {
        _uiState.update { it.copy(submitError = null) }
    }

    fun createOrder() {
        val state = _uiState.value
        if (state.selectedIds.isEmpty() || state.isSubmitting) return

        viewModelScope.launch {
            _uiState.update { it.copy(isSubmitting = true, submitError = null) }
            val retailerId = tokenManager.getUserId().orEmpty()
            val orderItems = state.forecasts
                .filter { state.selectedIds.contains(it.id) }
                .map { forecast ->
                    ProcurementOrderItem(
                        productId = forecast.productId,
                        quantity = state.quantities[forecast.id] ?: forecast.predictedQuantity,
                    )
                }
            val request = ProcurementOrderRequest(retailerId = retailerId, items = orderItems)
            val idempotencyKey = buildIdempotencyKey(orderItems)

            try {
                api.createOrder(request, idempotencyKey)
                _uiState.update {
                    it.copy(
                        isSubmitting = false,
                        showSuccess = true,
                        selectedIds = emptySet(),
                        quantities = emptyMap(),
                        loadIssue = null,
                    )
                }
                loadPredictions()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                if (issue == ProcurementLoadIssue.OFFLINE) {
                    queuePendingOrder(request, idempotencyKey)
                }
                _uiState.update {
                    it.copy(
                        isSubmitting = false,
                        submitError = resolveErrorMessage(e, issue, submitting = true),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun selectedProducts(): List<Pair<Product, Int>> {
        val state = _uiState.value
        return state.forecasts
            .filter { state.selectedIds.contains(it.id) }
            .mapNotNull { forecast ->
                val product = state.products.firstOrNull { it.id == forecast.productId } ?: return@mapNotNull null
                val qty = state.quantities[forecast.id] ?: forecast.predictedQuantity
                product to qty
            }
    }

    private suspend fun queuePendingOrder(request: ProcurementOrderRequest, idempotencyKey: String) {
        pendingOrderDao.insert(
            PendingOrderEntity(
                endpoint = "/v1/order/create",
                method = "POST",
                payloadJson = Json.encodeToString(request),
                idempotencyKey = idempotencyKey,
            ),
        )
    }

    private fun buildIdempotencyKey(items: List<ProcurementOrderItem>): String {
        return "retailer-procurement:" + items
            .map { "${it.productId}:${it.quantity}" }
            .sorted()
            .joinToString("|")
    }

    private fun resolveLoadIssue(error: Exception): ProcurementLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> ProcurementLoadIssue.RESTRICTED
            error is IOException -> ProcurementLoadIssue.OFFLINE
            else -> ProcurementLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(
        error: Exception,
        issue: ProcurementLoadIssue,
        submitting: Boolean,
    ): String {
        return when (issue) {
            ProcurementLoadIssue.RESTRICTED ->
                "Procurement order access is restricted for this account"
            ProcurementLoadIssue.OFFLINE ->
                if (submitting) {
                    "Saved for retry. Procurement submission is degraded."
                } else {
                    "Offline mode active. Reconnect and retry procurement."
                }
            ProcurementLoadIssue.ERROR ->
                error.message ?: "Procurement order could not be submitted. Please try again."
        }
    }
}
