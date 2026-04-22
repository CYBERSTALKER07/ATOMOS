package com.thelab.retailer.ui.screens.suppliers

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.retailer.data.api.LabApi
import com.thelab.retailer.data.model.Product
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class SupplierCatalogUiState(
    val isLoading: Boolean = true,
    val products: List<Product> = emptyList(),
    val error: String? = null,
)

@HiltViewModel
class SupplierCatalogViewModel @Inject constructor(
    private val api: LabApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SupplierCatalogUiState())
    val uiState: StateFlow<SupplierCatalogUiState> = _uiState.asStateFlow()

    fun load(supplierId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val products = api.getCatalogProducts(supplierId = supplierId)
                _uiState.update { it.copy(isLoading = false, products = products) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, products = emptyList(), error = e.message) }
            }
        }
    }
}