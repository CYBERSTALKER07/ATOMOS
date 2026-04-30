package com.thelab.retailer.ui.screens.catalog

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.thelab.retailer.data.api.LabApi
import com.thelab.retailer.data.model.Supplier
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CategorySuppliersUiState(
    val isLoading: Boolean = true,
    val suppliers: List<Supplier> = emptyList(),
    val error: String? = null,
)

@HiltViewModel
class CategorySuppliersViewModel @Inject constructor(
    private val api: LabApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CategorySuppliersUiState())
    val uiState: StateFlow<CategorySuppliersUiState> = _uiState.asStateFlow()

    fun load(categoryId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val suppliers = api.getCategorySuppliers(categoryId)
                _uiState.update { it.copy(isLoading = false, suppliers = suppliers) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, suppliers = emptyList(), error = e.message) }
            }
        }
    }
}