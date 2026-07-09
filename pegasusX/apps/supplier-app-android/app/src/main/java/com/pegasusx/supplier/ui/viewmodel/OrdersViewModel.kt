package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

enum class OrderFilterTab { ACTIVE, SCHEDULED, COMPLETED, CANCELLED }

data class OrdersUiState(
    val filter: OrderFilterTab = OrderFilterTab.ACTIVE,
    val orders: List<SupplierOrder> = emptyList(),
    val loading: Boolean = true,
    val error: String? = null,
    val vettingId: String? = null,
    val reassignTarget: String? = null,
    val reassignRecommendations: com.pegasusx.supplier.data.model.RecommendReassignResponse? = null,
    val reassignMessage: String? = null,
    val isReassigning: Boolean = false,
)

@HiltViewModel
class OrdersViewModel @Inject constructor(
    private val ops: SupplierOperationsRepository,
    private val realtimeSignals: SupplierRealtimeSignals,
) : ViewModel() {
    private val _state = MutableStateFlow(OrdersUiState())
    val state: StateFlow<OrdersUiState> = _state.asStateFlow()

    init {
        load()
        viewModelScope.launch {
            realtimeSignals.refreshTick.collect { load(silent = true) }
        }
        viewModelScope.launch {
            realtimeSignals.reconnectTick.collect { load(silent = true) }
        }
    }

    fun setFilter(filter: OrderFilterTab) {
        _state.update { it.copy(filter = filter) }
        load()
    }

    fun load(silent: Boolean = false) {
        viewModelScope.launch {
            if (!silent) {
                _state.update { it.copy(loading = true, error = null) }
            } else {
                _state.update { it.copy(error = null) }
            }
            try {
                val filter = _state.value.filter
                val resp = when (filter) {
                    OrderFilterTab.SCHEDULED -> ops.getOrders(status = "SCHEDULED", limit = 500)
                    else -> ops.getOrders(filter = filter.name, limit = 500)
                }
                if (resp.isSuccessful) {
                    _state.update {
                        it.copy(orders = resp.body()?.orders.orEmpty(), loading = false)
                    }
                } else if (!silent) {
                    _state.update {
                        it.copy(error = "Failed (${resp.code()})", loading = false, orders = emptyList())
                    }
                } else {
                    _state.update { it.copy(loading = false) }
                }
            } catch (e: Exception) {
                if (!silent) {
                    _state.update { it.copy(error = e.message, loading = false) }
                } else {
                    _state.update { it.copy(loading = false) }
                }
            }
        }
    }

    fun canWarehouseOps(order: SupplierOrder): Boolean {
        val warehouseId = order.warehouseId ?: return false
        if (warehouseId.isBlank()) return false
        val filter = _state.value.filter
        return filter == OrderFilterTab.ACTIVE || filter == OrderFilterTab.SCHEDULED
    }

    fun proposeWarehouseOrder(order: SupplierOrder, proposedDeliveryDate: String, reason: String) {
        val warehouseId = order.warehouseId ?: return
        viewModelScope.launch {
            _state.update { it.copy(vettingId = order.orderId) }
            try {
                val resp = ops.proposeWarehouseOrder(
                    order.orderId,
                    warehouseId,
                    proposedDeliveryDate,
                    reason,
                    SupplierIdempotencyKeys.warehouseOrderPropose(order.orderId, proposedDeliveryDate, reason),
                )
                if (resp.isSuccessful) load(silent = true) else {
                    _state.update { it.copy(error = "Propose failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message) }
            } finally {
                _state.update { it.copy(vettingId = null) }
            }
        }
    }

    fun rejectWarehouseOrder(order: SupplierOrder, reason: String) {
        val warehouseId = order.warehouseId ?: return
        viewModelScope.launch {
            _state.update { it.copy(vettingId = order.orderId) }
            try {
                val resp = ops.rejectWarehouseOrder(
                    order.orderId,
                    warehouseId,
                    reason,
                    SupplierIdempotencyKeys.warehouseOrderReject(order.orderId, reason),
                )
                if (resp.isSuccessful) load(silent = true) else {
                    _state.update { it.copy(error = "Reject failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message) }
            } finally {
                _state.update { it.copy(vettingId = null) }
            }
        }
    }

    fun dismissReassignMessage() {
        _state.update { it.copy(reassignMessage = null) }
    }

    fun openReassignDialog(orderId: String) {
        _state.update { it.copy(reassignTarget = orderId, reassignRecommendations = null, reassignMessage = null) }
        viewModelScope.launch {
            try {
                val resp = ops.recommendReassign(orderId)
                if (resp.isSuccessful) {
                    _state.update { it.copy(reassignRecommendations = resp.body()) }
                } else {
                    _state.update {
                        it.copy(
                            reassignMessage = "Failed to load recommendations (${resp.code()})",
                            reassignTarget = null
                        )
                    }
                }
            } catch (e: Exception) {
                _state.update { it.copy(reassignMessage = e.message ?: "Network error", reassignTarget = null) }
            }
        }
    }

    fun closeReassignDialog() {
        if (!_state.value.isReassigning) {
            _state.update { it.copy(reassignTarget = null, reassignRecommendations = null) }
        }
    }

    fun applyReassign(orderId: String, driverId: String, partial: Boolean) {
        _state.update { it.copy(isReassigning = true, reassignMessage = null) }
        viewModelScope.launch {
            try {
                val resp = ops.applyReassign(orderId, driverId, partial)
                if (resp.isSuccessful) {
                    _state.update {
                        it.copy(
                            reassignMessage = if (partial) "Reassigned (Partial)" else "Reassigned (Full)",
                            reassignTarget = null
                        )
                    }
                    load(silent = true)
                } else {
                    _state.update { it.copy(reassignMessage = "Failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(reassignMessage = e.message ?: "Network error") }
            } finally {
                _state.update { it.copy(isReassigning = false) }
            }
        }
    }
}
