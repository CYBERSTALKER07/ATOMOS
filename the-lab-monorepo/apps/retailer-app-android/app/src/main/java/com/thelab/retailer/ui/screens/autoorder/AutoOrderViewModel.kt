package com.thelab.retailer.ui.screens.autoorder

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.retailer.data.api.LabApi
import com.thelab.retailer.data.model.AutoOrderSettings
import com.thelab.retailer.data.model.DemandForecast
import com.thelab.retailer.data.model.UpdateGlobalSettingsRequest
import com.thelab.retailer.data.model.UpdateSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

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
)

@HiltViewModel
class AutoOrderViewModel @Inject constructor(
    private val api: LabApi,
    private val tokenManager: com.thelab.retailer.data.local.TokenManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AutoOrderUiState())
    val uiState: StateFlow<AutoOrderUiState> = _uiState.asStateFlow()

    init {
        loadAll()
    }

    fun loadAll() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val settings = api.getAutoOrderSettings()
                val rid = tokenManager.getUserId() ?: ""
                val forecasts = try { api.getPredictions(rid) } catch (_: Exception) { emptyList() }
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        settings = settings,
                        forecasts = forecasts,
                        globalEnabled = settings.globalEnabled,
                    )
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
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
                if (target is EnableTarget.Global) _uiState.update { it.copy(globalEnabled = false) }
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }

    private fun disableTarget(target: EnableTarget) {
        viewModelScope.launch {
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
                if (target is EnableTarget.Global) _uiState.update { it.copy(globalEnabled = true) }
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }
}

