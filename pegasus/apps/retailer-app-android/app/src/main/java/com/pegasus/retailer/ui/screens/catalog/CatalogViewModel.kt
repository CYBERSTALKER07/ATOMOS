package com.pegasus.retailer.ui.screens.catalog

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.Product
import com.pegasus.retailer.data.model.ProductCategory
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CatalogUiState(
    val isLoading: Boolean = false,
    val categories: List<ProductCategory> = emptyList(),
    val products: List<Product> = emptyList(),
    val filteredProducts: List<Product> = emptyList(),
    val searchQuery: String = "",
    val isSearching: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class CatalogViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CatalogUiState())
    val uiState: StateFlow<CatalogUiState> = _uiState.asStateFlow()

    init {
        loadCategories()
    }

    private fun loadCategories() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val categories = api.getCategories()
                _uiState.update { it.copy(isLoading = false, categories = categories) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, categories = emptyList()) }
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
                val products = api.getCatalogProducts()
                val filtered = products.filter { it.name.contains(query, ignoreCase = true) }
                _uiState.update { it.copy(isSearching = false, filteredProducts = filtered) }
            } catch (e: Exception) {
                val filtered = emptyList<Product>()
                _uiState.update { it.copy(isSearching = false, filteredProducts = filtered) }
            }
        }
    }
}
