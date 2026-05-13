package com.pegasus.retailer.ui.screens.autoorder

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.AutoOrderSettings
import com.pegasus.retailer.data.model.DemandForecast
import com.pegasus.retailer.data.model.UpdateGlobalSettingsRequest
import com.pegasus.retailer.data.model.UpdateSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class AutoOrderLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

/** Represents which entity the pending enable/history-choice dialog is for. */
sealed class EnableTarget {
    object Global : EnableTarget()
    data class Supplier(val id: String) : EnableTarget()
    data class Category(val id: String) : EnableTarget()
    data class Product(val id: String) : EnableTarget()
    data class Variant(val id: String) : EnableTarget()
}

data class AutoOrderUiState(
    val isLoading: Boolean = true,
    val settings: AutoOrderSettings? = null,
    val forecasts: List<DemandForecast> = emptyList(),
    val globalEnabled: Boolean = false,
    /** Non-null when the "Use past history or start fresh?" dialog should be shown. */
    val pendingEnableTarget: EnableTarget? = null,
    val error: String? = null,
    val loadIssue: AutoOrderLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            AutoOrderLoadIssue.RESTRICTED -> "Auto-order access is restricted for this account"
            AutoOrderLoadIssue.OFFLINE -> "Offline mode active. Showing latest auto-order settings"
            AutoOrderLoadIssue.ERROR -> "Auto-order sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class AutoOrderViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: com.pegasus.retailer.data.local.TokenManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AutoOrderUiState())
    val uiState: StateFlow<AutoOrderUiState> = _uiState.asStateFlow()

    init {
        loadAll()
    }

    fun loadAll() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            var nextIssue: AutoOrderLoadIssue? = null
            var nextError: String? = null
            var nextSettings: AutoOrderSettings? = null

            try {
                nextSettings = api.getAutoOrderSettings()
            } catch (e: Exception) {
                nextIssue = resolveLoadIssue(e)
                nextError = resolveErrorMessage(e, nextIssue)
            }

            val rid = tokenManager.getUserId().orEmpty()
            val nextForecasts = try {
                api.getPredictions(rid)
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
                _uiState.value.forecasts
            }

            _uiState.update {
                val effectiveSettings = nextSettings ?: it.settings
                it.copy(
                    isLoading = false,
                    settings = effectiveSettings,
                    forecasts = nextForecasts,
                    globalEnabled = effectiveSettings?.globalEnabled ?: it.globalEnabled,
                    error = nextError,
                    loadIssue = nextIssue,
                )
            }
        }
    }

    /**
     * Called when any entity toggle is switched ON.
     * If the entity has order history, show the "Continue/Fresh" dialog.
     * If no history exists, silently enable with useHistory=false (start fresh).
     */
    fun onToggle(target: EnableTarget, enabled: Boolean) {
        if (!enabled) {
            disableTarget(target)
            return
        }
        val hasHistory = entityHasHistory(target)
        if (hasHistory) {
            _uiState.update { it.copy(pendingEnableTarget = target) }
        } else {
            // No history — start fresh without prompting
            enableTarget(target, useHistory = false)
        }
    }

    /** Called from the dialog when the user picks "Use history" or "Start fresh". */
    fun confirmEnable(useHistory: Boolean) {
        val target = _uiState.value.pendingEnableTarget ?: return
        _uiState.update { it.copy(pendingEnableTarget = null) }
        enableTarget(target, useHistory)
    }

    fun dismissEnableDialog() {
        _uiState.update { it.copy(pendingEnableTarget = null) }
    }

    // ── Legacy entry-points kept for backward-compat call-sites ──────────────

    fun onGlobalToggle(enabled: Boolean) = onToggle(EnableTarget.Global, enabled)

    fun toggleSupplier(supplierId: String, enabled: Boolean) =
        onToggle(EnableTarget.Supplier(supplierId), enabled)

    fun toggleCategory(categoryId: String, enabled: Boolean) =
        onToggle(EnableTarget.Category(categoryId), enabled)

    fun toggleProduct(productId: String, enabled: Boolean) =
        onToggle(EnableTarget.Product(productId), enabled)

    fun toggleVariant(skuId: String, enabled: Boolean) =
        onToggle(EnableTarget.Variant(skuId), enabled)

    // ── Private helpers ───────────────────────────────────────────────────────

    private fun entityHasHistory(target: EnableTarget): Boolean {
        val s = _uiState.value.settings ?: return false
        return when (target) {
            is EnableTarget.Global -> s.hasAnyHistory
            is EnableTarget.Supplier -> s.supplierOverrides.firstOrNull { it.supplierId == target.id }?.hasHistory ?: s.hasAnyHistory
            is EnableTarget.Category -> s.categoryOverrides.firstOrNull { it.categoryId == target.id }?.hasHistory ?: s.hasAnyHistory
            is EnableTarget.Product -> s.productOverrides.firstOrNull { it.productId == target.id }?.hasHistory ?: s.hasAnyHistory
            is EnableTarget.Variant -> s.variantOverrides.firstOrNull { it.skuId == target.id }?.hasHistory ?: s.hasAnyHistory
        }
    }

    private fun enableTarget(target: EnableTarget, useHistory: Boolean) {
        viewModelScope.launch {
            _uiState.update { it.copy(error = null) }
            try {
                when (target) {
                    is EnableTarget.Global -> {
                        _uiState.update { it.copy(globalEnabled = true) }
                        api.updateGlobalAutoOrder(
                            UpdateGlobalSettingsRequest(globalAutoOrderEnabled = true, useHistory = useHistory)
                        )
                    }
                    is EnableTarget.Supplier ->
                        api.updateSupplierAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Category ->
                        api.updateCategoryAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Product ->
                        api.updateProductAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Variant ->
                        api.updateVariantAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                }
                loadAll()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                if (target is EnableTarget.Global) _uiState.update { it.copy(globalEnabled = false) }
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun disableTarget(target: EnableTarget) {
        viewModelScope.launch {
            _uiState.update { it.copy(error = null) }
            try {
                when (target) {
                    is EnableTarget.Global -> {
                        _uiState.update { it.copy(globalEnabled = false) }
                        api.updateGlobalAutoOrder(UpdateGlobalSettingsRequest(globalAutoOrderEnabled = false))
                    }
                    is EnableTarget.Supplier ->
                        api.updateSupplierAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Category ->
                        api.updateCategoryAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Product ->
                        api.updateProductAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Variant ->
                        api.updateVariantAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                }
                loadAll()
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                if (target is EnableTarget.Global) _uiState.update { it.copy(globalEnabled = true) }
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): AutoOrderLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> AutoOrderLoadIssue.RESTRICTED
            error is IOException -> AutoOrderLoadIssue.OFFLINE
            else -> AutoOrderLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: AutoOrderLoadIssue): String {
        return when (issue) {
            AutoOrderLoadIssue.RESTRICTED -> "Auto-order access is restricted for this account"
            AutoOrderLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AutoOrderLoadIssue.ERROR -> error.message ?: "Auto-order request failed"
        }
    }
}

