package com.pegasusx.retailer.ui.screens.procurement

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.local.PendingOrderEntity
import com.pegasusx.retailer.data.local.RetailerUserIdProvider
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.ProcurementOrderItem
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.services.PendingOrderSyncEnqueuer
import com.pegasusx.retailer.util.OrderLineContractHelper
import com.pegasusx.retailer.util.OwnerApprovalRequiredDetail
import com.pegasusx.retailer.util.PackMultipleSuggestion
import com.pegasusx.retailer.util.PackSnapMode
import com.pegasusx.retailer.util.PendingOrderConflictKind
import com.pegasusx.retailer.util.RetailerW2ContractHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import javax.inject.Inject
import kotlinx.coroutines.CancellationException
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
    val mustHaveByForecastId: Map<String, Boolean> = emptyMap(),
    val allowSubstitutesByForecastId: Map<String, Boolean> = emptyMap(),
    val productIdByForecastId: Map<String, String> = emptyMap(),
    val autoConfirmOptIn: Boolean = false,
    val packSuggestions: List<PackMultipleSuggestion> = emptyList(),
    val isSubmitting: Boolean = false,
    val showSuccess: Boolean = false,
    val queuedForSyncMessage: String? = null,
    val submitError: String? = null,
    val ownerApprovalRequired: OwnerApprovalRequiredDetail? = null,
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
    private val tokenManager: RetailerUserIdProvider,
    private val pendingOrderDao: PendingOrderDao,
    private val syncEnqueuer: PendingOrderSyncEnqueuer = PendingOrderSyncEnqueuer {},
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

            val products = try {
                api.getCatalogProducts(retailerId = retailerId.takeIf { it.isNotBlank() })
            } catch (e: Exception) {
                issue = resolveLoadIssue(e)
                _uiState.value.products
            }

            _uiState.update {
                it.copy(
                    isLoading = false,
                    forecasts = emptyList(),
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
            state.copy(
                selectedIds = nextSelected,
                quantities = nextQuantities,
                productIdByForecastId = state.productIdByForecastId + (forecast.id to forecast.productId),
            )
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

    fun effectivePackSizeFor(productId: String): Int {
        val product = _uiState.value.products.firstOrNull { it.id == productId } ?: return 1
        val variant = product.defaultVariant ?: return (product.unitsPerBlock ?: 1).takeIf { product.sellByBlock } ?: 1
        return OrderLineContractHelper.effectivePackSize(product, variant)
    }

    fun snapQuantityToPackMultiple(forecastId: String, mode: PackSnapMode = PackSnapMode.NEAREST) {
        val state = _uiState.value
        val forecast = state.forecasts.firstOrNull { it.id == forecastId }
        val productId = forecast?.productId ?: state.productIdByForecastId[forecastId] ?: forecastId
        val packSize = effectivePackSizeFor(productId)
        val currentQty = state.quantities[forecastId] ?: forecast?.predictedQuantity ?: 1
        val snapped = OrderLineContractHelper.snapToPackMultiple(currentQty, packSize, mode)
        if (snapped > 0) {
            updateQuantity(forecastId, snapped)
        }
        _uiState.update { s ->
            val remaining = s.packSuggestions.filterNot { it.skuId == productId || it.skuId == forecastId }
            s.copy(
                packSuggestions = remaining,
                submitError = if (remaining.isEmpty()) null else s.submitError,
            )
        }
    }

    fun applyPackSuggestion(skuId: String, suggestedQty: Int) {
        val stateSnapshot = _uiState.value
        val matchedForecastId = stateSnapshot.forecasts
            .firstOrNull { it.productId == skuId || it.id == skuId }
            ?.id
            ?: stateSnapshot.productIdByForecastId.entries
                .firstOrNull { it.value == skuId || it.key == skuId }
                ?.key
            ?: skuId
        if (suggestedQty > 0) {
            updateQuantity(matchedForecastId, suggestedQty)
        }
        _uiState.update { state ->
            val remaining = state.packSuggestions.filterNot { it.skuId == skuId || it.skuId == matchedForecastId }
            state.copy(
                packSuggestions = remaining,
                submitError = if (remaining.isEmpty()) null else state.submitError,
            )
        }
    }

    fun toggleMustHave(forecastId: String) {
        _uiState.update { state ->
            val next = !(state.mustHaveByForecastId[forecastId] ?: false)
            state.copy(mustHaveByForecastId = state.mustHaveByForecastId + (forecastId to next))
        }
    }

    fun toggleAllowSubstitutes(forecastId: String) {
        _uiState.update { state ->
            val next = !(state.allowSubstitutesByForecastId[forecastId] ?: false)
            state.copy(allowSubstitutesByForecastId = state.allowSubstitutesByForecastId + (forecastId to next))
        }
    }

    fun setAutoConfirmOptIn(enabled: Boolean) {
        _uiState.update { it.copy(autoConfirmOptIn = enabled) }
    }

    fun dismissPackSuggestions() {
        _uiState.update { it.copy(packSuggestions = emptyList()) }
    }

    fun dismissQueuedMessage() {
        _uiState.update { it.copy(queuedForSyncMessage = null) }
    }

    fun dismissSuccess() {
        _uiState.update {
            it.copy(
                showSuccess = false,
                selectedIds = emptySet(),
                quantities = emptyMap(),
                mustHaveByForecastId = emptyMap(),
                allowSubstitutesByForecastId = emptyMap(),
            )
        }
    }

    fun clearSelections() {
        _uiState.update {
            it.copy(
                selectedIds = emptySet(),
                quantities = emptyMap(),
                mustHaveByForecastId = emptyMap(),
                allowSubstitutesByForecastId = emptyMap(),
                packSuggestions = emptyList(),
            )
        }
    }

    fun clearSubmitError() {
        _uiState.update { it.copy(submitError = null) }
    }

    fun createOrder() {
        submitOrder(paymentMethodOverride = null)
    }

    fun dismissOwnerApprovalRequired() {
        _uiState.update { it.copy(ownerApprovalRequired = null) }
    }

    fun retryWithFallbackPaymentMethod(method: String) {
        val normalized = method.trim().uppercase()
        if (normalized.isEmpty()) return
        submitOrder(paymentMethodOverride = normalized)
    }

    private fun submitOrder(paymentMethodOverride: String?) {
        val state = _uiState.value
        if (state.selectedIds.isEmpty() || state.isSubmitting) return

        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    isSubmitting = true,
                    submitError = null,
                    queuedForSyncMessage = null,
                    packSuggestions = emptyList(),
                    ownerApprovalRequired = if (paymentMethodOverride != null) null else it.ownerApprovalRequired,
                )
            }
            if (paymentMethodOverride != null) {
                clearAwaitingOwnerApprovalProcurementOrders()
            }
            val retailerId = tokenManager.getUserId().orEmpty()
            val selectedForecasts = state.forecasts.filter { state.selectedIds.contains(it.id) }
            val orderItems = if (selectedForecasts.isNotEmpty()) {
                selectedForecasts.map { forecast ->
                    ProcurementOrderItem(
                        productId = forecast.productId,
                        quantity = state.quantities[forecast.id] ?: forecast.predictedQuantity,
                        mustHave = state.mustHaveByForecastId[forecast.id] ?: false,
                        allowSubstitutes = state.allowSubstitutesByForecastId[forecast.id] ?: false,
                    )
                }
            } else {
                state.selectedIds.sorted().map { selectedId ->
                    ProcurementOrderItem(
                        productId = state.productIdByForecastId[selectedId] ?: selectedId,
                        quantity = state.quantities[selectedId] ?: 1,
                        mustHave = state.mustHaveByForecastId[selectedId] ?: false,
                        allowSubstitutes = state.allowSubstitutesByForecastId[selectedId] ?: false,
                    )
                }
            }
            val fallbackGateway = when (paymentMethodOverride) {
                "CASH" -> "CASH"
                "CARD" -> "GLOBAL_PAY"
                else -> null
            }
            val request = ProcurementOrderRequest(
                retailerId = retailerId,
                items = orderItems,
                autoConfirmOptIn = state.autoConfirmOptIn,
                paymentMethod = paymentMethodOverride,
                paymentGateway = fallbackGateway,
            )
            val baseKey = RetailerIdempotencyKeys.orderCreate(
                items = orderItems,
                autoConfirmOptIn = state.autoConfirmOptIn,
            )
            val idempotencyKey = if (paymentMethodOverride != null) {
                "$baseKey:fallback:$paymentMethodOverride"
            } else {
                baseKey
            }

            try {
                api.createOrder(request, idempotencyKey)
                _uiState.update {
                    it.copy(
                        isSubmitting = false,
                        showSuccess = true,
                        selectedIds = emptySet(),
                        quantities = emptyMap(),
                        mustHaveByForecastId = emptyMap(),
                        allowSubstitutesByForecastId = emptyMap(),
                        packSuggestions = emptyList(),
                        ownerApprovalRequired = null,
                        loadIssue = null,
                    )
                }
                loadPredictions()
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                val packFixes = OrderLineContractHelper.parsePackMultipleSuggestions(
                    error = e,
                    fallbackSku = orderItems.firstOrNull()?.productId,
                )
                if (packFixes.isNotEmpty()) {
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            packSuggestions = packFixes,
                            submitError = packFixes.firstOrNull()?.message
                                ?: "Adjust order quantities to match required pack size.",
                        )
                    }
                    return@launch
                }

                val ownerApproval = RetailerW2ContractHelper.parseOwnerApprovalRequired(e)
                val assessment = OrderLineContractHelper.assessPendingOrderError(e)
                val shouldQueue = e is IOException ||
                    ownerApproval != null ||
                    assessment.kind == PendingOrderConflictKind.OWNER_APPROVAL_REQUIRED ||
                    assessment.kind == PendingOrderConflictKind.REQUEST_IN_PROGRESS ||
                    assessment.kind == PendingOrderConflictKind.RATE_LIMITED

                if (shouldQueue) {
                    val queueStatus = when {
                        ownerApproval != null || assessment.kind == PendingOrderConflictKind.OWNER_APPROVAL_REQUIRED ->
                            PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL
                        assessment.kind == PendingOrderConflictKind.REQUEST_IN_PROGRESS ->
                            PendingOrderEntity.STATUS_IN_PROGRESS
                        assessment.kind == PendingOrderConflictKind.RATE_LIMITED ->
                            PendingOrderEntity.STATUS_RATE_LIMITED
                        else -> PendingOrderEntity.STATUS_PENDING
                    }
                    val message = if (e is IOException) {
                        "Saved for retry. Procurement submission is queued and will auto-sync."
                    } else {
                        ownerApproval?.message ?: assessment.message
                    }
                    queuePendingOrder(request, idempotencyKey, status = queueStatus, lastError = message)
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            queuedForSyncMessage = message,
                            submitError = message,
                            ownerApprovalRequired = ownerApproval,
                            loadIssue = if (e is IOException) ProcurementLoadIssue.OFFLINE else null,
                        )
                    }
                    return@launch
                }

                val issue = resolveLoadIssue(e)
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

    private suspend fun clearAwaitingOwnerApprovalProcurementOrders() {
        runCatching {
            pendingOrderDao.getAll()
                .filter {
                    it.status == PendingOrderEntity.STATUS_AWAITING_OWNER_APPROVAL &&
                        it.endpoint == "/v1/order/create"
                }
                .forEach { pendingOrderDao.deleteById(it.id) }
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

    private suspend fun queuePendingOrder(
        request: ProcurementOrderRequest,
        idempotencyKey: String,
        status: String = PendingOrderEntity.STATUS_PENDING,
        lastError: String? = null,
    ) {
        pendingOrderDao.insert(
            PendingOrderEntity(
                endpoint = "/v1/order/create",
                method = "POST",
                payloadJson = Json.encodeToString(request),
                idempotencyKey = idempotencyKey,
                lastError = lastError,
                status = status,
            ),
        )
        syncEnqueuer.enqueue()
    }

    private fun resolveLoadIssue(error: Exception): ProcurementLoadIssue {
        val code = when (error) {
            is ProblemDetailException -> error.problem.status
            is HttpException -> error.code()
            else -> null
        }
        return when {
            code == 401 || code == 403 -> ProcurementLoadIssue.RESTRICTED
            error is IOException -> ProcurementLoadIssue.OFFLINE
            else -> ProcurementLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(
        error: Exception,
        issue: ProcurementLoadIssue,
        submitting: Boolean,
    ): String {
        val problemMessage = (error as? ProblemDetailException)?.let {
            it.problem.detail ?: it.problem.title ?: it.message
        }
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
                problemMessage ?: error.message ?: "Procurement order could not be submitted. Please try again."
        }
    }
}
