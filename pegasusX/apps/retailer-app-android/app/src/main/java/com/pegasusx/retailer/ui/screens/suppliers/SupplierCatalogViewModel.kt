package com.pegasusx.retailer.ui.screens.suppliers

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.local.TokenManager
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

enum class SupplierCatalogLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class SupplierCatalogUiState(
    val isLoading: Boolean = true,
    val products: List<Product> = emptyList(),
    val error: String? = null,
    val isFavorite: Boolean = false,
    val loadIssue: SupplierCatalogLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            SupplierCatalogLoadIssue.RESTRICTED -> "Supplier catalog access is restricted for this account"
            SupplierCatalogLoadIssue.OFFLINE -> "Offline mode active. Showing latest supplier catalog"
            SupplierCatalogLoadIssue.ERROR -> "Supplier catalog sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class SupplierCatalogViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SupplierCatalogUiState())
    val uiState: StateFlow<SupplierCatalogUiState> = _uiState.asStateFlow()
    private var activeSupplierId: String = ""

    init {
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events
                .filter {
                    it.type == "PROMOTION_CHANGED" &&
                        (it.supplierId.isBlank() || it.supplierId == activeSupplierId)
                }
                .collect {
                    val supplierId = activeSupplierId
                    if (supplierId.isNotBlank()) {
                        load(supplierId)
                    }
                }
        }
    }

    fun load(supplierId: String) {
        activeSupplierId = supplierId
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null, loadIssue = null) }
            try {
                // Actually, there's no single supplier GET to check favorite state...
                // But we can fetch mySuppliers and see if it's there
                val retailerId = tokenManager.getUserId()
                runCatching {
                    api.watchSupplierPromotions(mapOf("supplier_id" to supplierId))
                }
                val products = api.getCatalogProducts(
                    supplierId = supplierId,
                    retailerId = retailerId,
                )
                val mySuppliers = api.getMySuppliers()
                val isFav = mySuppliers.any { it.id == supplierId }
                _uiState.update { it.copy(isLoading = false, products = products, isFavorite = isFav, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue, "Could not load supplier catalog"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun toggleFavorite(supplierId: String) {
        viewModelScope.launch {
            try {
                val isCurrentlyFav = _uiState.value.isFavorite
                if (isCurrentlyFav) {
                    api.removeSupplier(supplierId)
                    _uiState.update { it.copy(isFavorite = false, error = null, loadIssue = null) }
                } else {
                    api.addSupplier(supplierId)
                    _uiState.update { it.copy(isFavorite = true, error = null, loadIssue = null) }
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        error = resolveErrorMessage(e, issue, "Could not update favorite supplier"),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): SupplierCatalogLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> SupplierCatalogLoadIssue.RESTRICTED
            error is IOException -> SupplierCatalogLoadIssue.OFFLINE
            else -> SupplierCatalogLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: SupplierCatalogLoadIssue, fallback: String): String {
        return when (issue) {
            SupplierCatalogLoadIssue.RESTRICTED -> "Supplier catalog access is restricted for this account"
            SupplierCatalogLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            SupplierCatalogLoadIssue.ERROR -> error.message ?: fallback
        }
    }
}