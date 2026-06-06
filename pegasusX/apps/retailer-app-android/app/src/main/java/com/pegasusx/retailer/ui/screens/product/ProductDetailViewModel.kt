package com.pegasusx.retailer.ui.screens.product

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.AutoOrderSettings
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.UpdateSettingsRequest
import com.pegasusx.retailer.ui.screens.autoorder.EnableTarget
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class ProductDetailLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class ProductDetailUiState(
    val isLoading: Boolean = true,
    val product: Product? = null,
    val settings: AutoOrderSettings? = null,
    val pendingEnableTarget: EnableTarget? = null,
    val error: String? = null,
    val loadIssue: ProductDetailLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            ProductDetailLoadIssue.RESTRICTED -> "Product detail access is restricted for this account"
            ProductDetailLoadIssue.OFFLINE -> "Offline mode active. Showing latest product settings"
            ProductDetailLoadIssue.ERROR -> "Product detail sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class ProductDetailViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ProductDetailUiState())
    val uiState: StateFlow<ProductDetailUiState> = _uiState.asStateFlow()

    fun load(productId: String) {
        if (
            _uiState.value.product != null &&
            _uiState.value.settings != null &&
            _uiState.value.loadIssue == null &&
            _uiState.value.error == null
        ) return

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = it.product == null, error = null) }

            var nextIssue: ProductDetailLoadIssue? = null
            var nextError: String? = null
            var nextProduct: Product? = null
            var nextSettings: AutoOrderSettings? = null

            try {
                val products = api.getCatalogProducts()
                nextProduct = products.firstOrNull { it.id == productId }
                if (nextProduct == null && _uiState.value.product == null) {
                    nextIssue = ProductDetailLoadIssue.ERROR
                    nextError = "Product not found"
                }
            } catch (e: Exception) {
                nextIssue = resolveLoadIssue(e)
                nextError = resolveErrorMessage(e, nextIssue)
            }

            try {
                nextSettings = api.getAutoOrderSettings()
            } catch (e: Exception) {
                if (nextIssue == null) {
                    nextIssue = resolveLoadIssue(e)
                    nextError = resolveErrorMessage(e, nextIssue)
                }
            }

            _uiState.update {
                it.copy(
                    isLoading = false,
                    product = nextProduct ?: it.product,
                    settings = nextSettings ?: it.settings,
                    error = nextError,
                    loadIssue = nextIssue,
                )
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
                _uiState.update { it.copy(settings = refreshed, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update { it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue) }
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
                _uiState.update { it.copy(settings = refreshed, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update { it.copy(error = resolveErrorMessage(e, issue), loadIssue = issue) }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): ProductDetailLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> ProductDetailLoadIssue.RESTRICTED
            error is IOException -> ProductDetailLoadIssue.OFFLINE
            else -> ProductDetailLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: ProductDetailLoadIssue): String {
        return when (issue) {
            ProductDetailLoadIssue.RESTRICTED -> "Product detail access is restricted for this account"
            ProductDetailLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            ProductDetailLoadIssue.ERROR -> error.message ?: "Product detail request failed"
        }
    }
}
