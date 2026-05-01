package com.pegasus.retailer.ui.screens.suppliers

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.Supplier
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class MySuppliersUiState(
    val isLoading: Boolean = false,
    val suppliers: List<Supplier> = emptyList(),
    val error: String? = null,
)

@HiltViewModel
class MySuppliersViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(MySuppliersUiState())
    val uiState: StateFlow<MySuppliersUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val suppliers = api.getMySuppliers()
                _uiState.update { it.copy(isLoading = false, suppliers = suppliers, error = null) }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        suppliers = emptyList(),
                        error = e.message ?: "Supplier list unavailable. Check your connection.",
                    )
                }
            }
        }
    }

    fun addSupplier(supplierId: String) {
        viewModelScope.launch {
            try {
                api.addSupplier(supplierId)
                refresh()
            } catch (_: Exception) {}
        }
    }

    fun removeSupplier(supplierId: String) {
        viewModelScope.launch {
            try {
                api.removeSupplier(supplierId)
                refresh()
            } catch (_: Exception) {}
        }
    }
}
