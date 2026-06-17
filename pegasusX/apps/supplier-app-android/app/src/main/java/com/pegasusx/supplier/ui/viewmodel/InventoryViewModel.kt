package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.InventoryItem
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

data class InventoryUiState(
    val items: List<InventoryItem> = emptyList(),
    val loading: Boolean = true,
    val error: String? = null,
    val adjustingSku: String? = null,
    val adjustBusy: Boolean = false,
)

@HiltViewModel
class InventoryViewModel @Inject constructor(
    private val api: SupplierApi,
    private val ops: SupplierOperationsRepository,
) : ViewModel() {
    private val _state = MutableStateFlow(InventoryUiState())
    val state: StateFlow<InventoryUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, error = null) }
            try {
                val resp = api.getInventory()
                if (resp.isSuccessful) {
                    _state.update {
                        it.copy(items = resp.body()?.items.orEmpty(), loading = false)
                    }
                } else {
                    _state.update {
                        it.copy(error = "Failed (${resp.code()})", loading = false)
                    }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message, loading = false) }
            }
        }
    }

    fun showAdjust(sku: String?) {
        _state.update { it.copy(adjustingSku = sku) }
    }

    fun adjustQuantity(sku: String, delta: Long) {
        viewModelScope.launch {
            _state.update { it.copy(adjustBusy = true, error = null) }
            try {
                val body = buildJsonObject {
                    put("sku", JsonPrimitive(sku))
                    put("quantity_delta", JsonPrimitive(delta))
                }
                val resp = ops.updateInventory(body)
                if (resp.isSuccessful) {
                    showAdjust(null)
                    load()
                } else {
                    _state.update { it.copy(error = "Adjust failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message) }
            } finally {
                _state.update { it.copy(adjustBusy = false) }
            }
        }
    }
}
