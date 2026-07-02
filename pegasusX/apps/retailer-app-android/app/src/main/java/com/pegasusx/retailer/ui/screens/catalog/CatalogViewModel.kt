package com.pegasusx.retailer.ui.screens.catalog

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.ProductCategory
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

enum class CatalogBrowseMode {
    CATEGORIES,
    ALL_PRODUCTS,
}

data class CatalogSupplierFilter(
    val id: String,
    val name: String,
)

data class CatalogUiState(
    val isLoading: Boolean = false,
    val isLoadingProducts: Boolean = false,
    val categories: List<ProductCategory> = emptyList(),
    val products: List<Product> = emptyList(),
    val filteredProducts: List<Product> = emptyList(),
    val searchQuery: String = "",
    val isSearching: Boolean = false,
    val browseMode: CatalogBrowseMode = CatalogBrowseMode.CATEGORIES,
    val selectedSupplierId: String? = null,
    val supplierFilters: List<CatalogSupplierFilter> = emptyList(),
    val error: String? = null,
) {
    val displayedProducts: List<Product>
        get() {
            val base = if (searchQuery.length >= 2) filteredProducts else products
            val supplierId = selectedSupplierId?.takeIf { it.isNotBlank() } ?: return base
            return base.filter { it.supplierId == supplierId }
        }
}

@HiltViewModel
class CatalogViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    private val retailerWebSocket: RetailerWebSocket,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CatalogUiState())
    val uiState: StateFlow<CatalogUiState> = _uiState.asStateFlow()
    private val retailerId: String get() = tokenManager.getUserId().orEmpty()

    init {
        loadCategories()
        retailerWebSocket.connect()
        viewModelScope.launch {
            retailerWebSocket.events
                .filter { it.type == "PROMOTION_CHANGED" }
                .collect { msg ->
                    val selected = _uiState.value.selectedSupplierId
                    if (
                        _uiState.value.browseMode == CatalogBrowseMode.ALL_PRODUCTS &&
                        (msg.supplierId.isBlank() || selected.isNullOrBlank() || msg.supplierId == selected)
                    ) {
                        loadAllProducts()
                    }
                }
        }
        viewModelScope.launch {
            retailerWebSocket.reconnects.collect {
                refresh()
            }
        }
    }

    fun refresh() {
        when (_uiState.value.browseMode) {
            CatalogBrowseMode.ALL_PRODUCTS -> loadAllProducts()
            CatalogBrowseMode.CATEGORIES -> loadCategories()
        }
    }

    fun onSupplierFilterSelected(supplierId: String?) {
        val normalized = supplierId?.takeIf { it.isNotBlank() }
        _uiState.update { it.copy(selectedSupplierId = normalized) }
        if (!normalized.isNullOrBlank()) {
            viewModelScope.launch {
                runCatching {
                    api.watchSupplierPromotions(mapOf("supplier_id" to normalized))
                }
            }
        }
    }

    private fun loadCategories() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val categories = api.getCategories()
                _uiState.update { it.copy(isLoading = false, categories = categories, error = null) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, categories = emptyList(), error = e.message) }
            }
        }
    }

    fun onBrowseModeSelected(mode: CatalogBrowseMode) {
        _uiState.update {
            it.copy(
                browseMode = mode,
                selectedSupplierId = if (mode == CatalogBrowseMode.ALL_PRODUCTS) it.selectedSupplierId else null,
            )
        }
        if (mode == CatalogBrowseMode.ALL_PRODUCTS && _uiState.value.products.isEmpty()) {
            loadAllProducts()
        }
    }

    private fun loadAllProducts() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingProducts = true) }
            try {
                val products = api.getCatalogProducts(retailerId = retailerId.takeIf { it.isNotBlank() })
                val supplierMap = LinkedHashMap<String, String>()
                for (product in products) {
                    val supplierId = product.supplierId?.takeIf { it.isNotBlank() } ?: continue
                    supplierMap[supplierId] = product.supplierName?.takeIf { it.isNotBlank() }
                        ?: supplierMap[supplierId]
                        ?: supplierId.take(8)
                }
                val filters = supplierMap.map { CatalogSupplierFilter(it.key, it.value) }
                    .sortedBy { it.name.lowercase() }
                val retainedSupplierId = _uiState.value.selectedSupplierId?.takeIf { id -> id in supplierMap }
                _uiState.update {
                    it.copy(
                        isLoadingProducts = false,
                        products = products,
                        supplierFilters = filters,
                        selectedSupplierId = retainedSupplierId,
                        error = null,
                    )
                }
                retainedSupplierId?.let { supplierId ->
                    runCatching {
                        api.watchSupplierPromotions(mapOf("supplier_id" to supplierId))
                    }
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoadingProducts = false, products = emptyList(), error = e.message) }
            }
        }
    }

    fun onSearchChanged(query: String) {
        _uiState.update { it.copy(searchQuery = query) }
        if (query.length >= 2) {
            searchProducts(query)
        } else {
            _uiState.update { it.copy(isSearching = false, filteredProducts = emptyList()) }
        }
    }

    private fun searchProducts(query: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isSearching = true) }
            try {
                val products = if (_uiState.value.products.isNotEmpty()) {
                    _uiState.value.products
                } else {
                    api.getCatalogProducts(retailerId = retailerId.takeIf { it.isNotBlank() })
                }
                val filtered = products.filter { product ->
                    product.name.contains(query, ignoreCase = true) ||
                        product.description.contains(query, ignoreCase = true) ||
                        product.supplierName?.contains(query, ignoreCase = true) == true ||
                        product.categoryName?.contains(query, ignoreCase = true) == true
                }
                _uiState.update {
                    it.copy(
                        isSearching = false,
                        filteredProducts = filtered,
                        products = if (it.products.isEmpty()) products else it.products,
                    )
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(isSearching = false, filteredProducts = emptyList(), error = e.message) }
            }
        }
    }
}
