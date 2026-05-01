package com.pegasus.retailer.ui.screens.product

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.AutoOrderSettings
import com.pegasus.retailer.data.model.Product
import com.pegasus.retailer.data.model.UpdateSettingsRequest
import com.pegasus.retailer.ui.screens.autoorder.EnableTarget
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ProductDetailUiState(
    val isLoading: Boolean = true,
    val product: Product? = null,
    val settings: AutoOrderSettings? = null,
    val pendingEnableTarget: EnableTarget? = null,
    val error: String? = null,
)

@HiltViewModel
class ProductDetailViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ProductDetailUiState())
    val uiState: StateFlow<ProductDetailUiState> = _uiState.asStateFlow()

    fun load(productId: String) {
        if (_uiState.value.product != null) return
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val products = api.getCatalogProducts()
                val product = products.firstOrNull { it.id == productId }
                val settings = try { api.getAutoOrderSettings() } catch (_: Exception) { null }
                _uiState.update { it.copy(isLoading = false, product = product, settings = settings) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun onToggleProduct(productId: String, enabled: Boolean) {
        if (!enabled) {
            disableEntity(EnableTarget.Product(productId))
            return
        }
        val hasHistory = _uiState.value.settings
            ?.productOverrides?.firstOrNull { it.productId == productId }
            ?.hasHistory
            ?: (_uiState.value.settings?.hasAnyHistory == true)
        if (hasHistory) {
            _uiState.update { it.copy(pendingEnableTarget = EnableTarget.Product(productId)) }
        } else {
            enableEntity(EnableTarget.Product(productId), useHistory = false)
        }
    }

    fun onToggleVariant(skuId: String, enabled: Boolean) {
        if (!enabled) {
            disableEntity(EnableTarget.Variant(skuId))
            return
        }
        val hasHistory = _uiState.value.settings
            ?.variantOverrides?.firstOrNull { it.skuId == skuId }
            ?.hasHistory
            ?: (_uiState.value.settings?.hasAnyHistory == true)
        if (hasHistory) {
            _uiState.update { it.copy(pendingEnableTarget = EnableTarget.Variant(skuId)) }
        } else {
            enableEntity(EnableTarget.Variant(skuId), useHistory = false)
        }
    }

    fun confirmEnable(useHistory: Boolean) {
        val target = _uiState.value.pendingEnableTarget ?: return
        _uiState.update { it.copy(pendingEnableTarget = null) }
        enableEntity(target, useHistory)
    }

    fun dismissEnableDialog() {
        _uiState.update { it.copy(pendingEnableTarget = null) }
    }

    private fun enableEntity(target: EnableTarget, useHistory: Boolean) {
        viewModelScope.launch {
            try {
                when (target) {
                    is EnableTarget.Product ->
                        api.updateProductAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    is EnableTarget.Variant ->
                        api.updateVariantAutoOrder(target.id, UpdateSettingsRequest(enabled = true, useHistory = useHistory))
                    else -> Unit
                }
                val refreshed = api.getAutoOrderSettings()
                _uiState.update { it.copy(settings = refreshed) }
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }

    private fun disableEntity(target: EnableTarget) {
        viewModelScope.launch {
            try {
                when (target) {
                    is EnableTarget.Product ->
                        api.updateProductAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    is EnableTarget.Variant ->
                        api.updateVariantAutoOrder(target.id, UpdateSettingsRequest(enabled = false))
                    else -> Unit
                }
                val refreshed = api.getAutoOrderSettings()
                _uiState.update { it.copy(settings = refreshed) }
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
            }
        }
    }
}
