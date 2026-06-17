package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

enum class OrderFilterTab { ACTIVE, REVIEW, COMPLETED, CANCELLED }

data class OrdersUiState(
    val filter: OrderFilterTab = OrderFilterTab.ACTIVE,
    val orders: List<SupplierOrder> = emptyList(),
    val loading: Boolean = true,
    val error: String? = null,
    val vettingId: String? = null,
)

@HiltViewModel
class OrdersViewModel @Inject constructor(
    private val ops: SupplierOperationsRepository,
) : ViewModel() {
    private val _state = MutableStateFlow(OrdersUiState())
    val state: StateFlow<OrdersUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun setFilter(filter: OrderFilterTab) {
        _state.update { it.copy(filter = filter) }
        load()
    }

    fun load() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, error = null) }
            try {
                val filter = _state.value.filter
                val resp = when (filter) {
                    OrderFilterTab.REVIEW -> ops.getOrders(status = "AWAITING_REVIEW", limit = 500)
                    else -> ops.getOrders(filter = filter.name, limit = 500)
                }
                if (resp.isSuccessful) {
                    _state.update {
                        it.copy(orders = resp.body()?.orders.orEmpty(), loading = false)
                    }
                } else {
                    _state.update {
                        it.copy(error = "Failed (${resp.code()})", loading = false, orders = emptyList())
                    }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message, loading = false) }
            }
        }
    }

    fun vetOrder(order: SupplierOrder, decision: String) {
        viewModelScope.launch {
            _state.update { it.copy(vettingId = order.orderId) }
            try {
                val body = buildJsonObject {
                    put("order_id", JsonPrimitive(order.orderId))
                    put("retailer_id", JsonPrimitive(order.retailerId))
                    put("decision", JsonPrimitive(decision))
                    order.note?.takeIf { it.isNotBlank() }?.let { put("note", JsonPrimitive(it)) }
                    if (order.totalMinor > 0) put("total_minor", JsonPrimitive(order.totalMinor))
                    if (order.currency.isNotBlank()) put("currency", JsonPrimitive(order.currency))
                }
                val key = SupplierIdempotencyKeys.vetOrder(order.orderId, decision)
                val resp = ops.vetOrder(body, key)
                if (resp.isSuccessful) load() else {
                    _state.update { it.copy(error = "Vet failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message) }
            } finally {
                _state.update { it.copy(vettingId = null) }
            }
        }
    }
}
