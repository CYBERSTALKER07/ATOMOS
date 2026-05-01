package com.pegasus.retailer.ui.screens.tracking

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.api.RetailerWebSocket
import com.pegasus.retailer.data.model.TrackingOrder
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class SupplierFilter(
    val supplierId: String,
    val supplierName: String,
)

data class TrackingUiState(
    val orders: List<TrackingOrder> = emptyList(),
    val suppliers: List<SupplierFilter> = emptyList(),
    val selectedSupplierIds: Set<String> = emptySet(),
    val isLoading: Boolean = true,
    val error: String? = null,
) {
    /** Orders filtered by selected suppliers. If none selected, show all. */
    val visibleOrders: List<TrackingOrder>
        get() = if (selectedSupplierIds.isEmpty()) orders
        else orders.filter { it.supplierId in selectedSupplierIds }
}

@HiltViewModel
class DeliveryTrackingViewModel @Inject constructor(
    private val api: PegasusApi,
    private val ws: RetailerWebSocket,
) : ViewModel() {

    private val _state = MutableStateFlow(TrackingUiState())
    val state: StateFlow<TrackingUiState> = _state.asStateFlow()

    init {
        startPolling()
        observeWebSocket()
    }

    fun toggleSupplier(supplierId: String) {
        val current = _state.value.selectedSupplierIds
        _state.value = _state.value.copy(
            selectedSupplierIds = if (supplierId in current) current - supplierId else current + supplierId,
        )
    }

    fun refresh() {
        viewModelScope.launch { fetchTracking() }
    }

    private fun startPolling() {
        viewModelScope.launch {
            while (true) {
                fetchTracking()
                delay(15_000) // 15-second polling interval
            }
        }
    }

    private suspend fun fetchTracking() {
        try {
            val response = api.getTrackingOrders()
            // Filter out COMPLETED/CANCELLED — backend already does this, but belt-and-suspenders
            val active = response.orders.filter { it.state !in listOf("COMPLETED", "CANCELLED") }

            // Extract unique suppliers
            val supplierMap = LinkedHashMap<String, String>()
            for (o in active) {
                if (o.supplierId.isNotEmpty()) {
                    supplierMap[o.supplierId] = o.supplierName
                }
            }
            val suppliers = supplierMap.map { SupplierFilter(it.key, it.value) }

            _state.value = _state.value.copy(
                orders = active,
                suppliers = suppliers,
                isLoading = false,
                error = null,
            )
        } catch (e: Exception) {
            _state.value = _state.value.copy(
                isLoading = false,
                error = if (_state.value.orders.isEmpty()) "Failed to load: ${e.message}" else null,
            )
        }
    }

    private fun observeWebSocket() {
        viewModelScope.launch {
            ws.events.collect { msg ->
                when (msg.type) {
                    "DRIVER_APPROACHING" -> {
                        if (msg.orderId.isNotEmpty()) {
                            val updated = _state.value.orders.map { order ->
                                if (order.orderId == msg.orderId) {
                                    order.copy(
                                        isApproaching = true,
                                        driverLatitude = msg.driverLatitude ?: order.driverLatitude,
                                        driverLongitude = msg.driverLongitude ?: order.driverLongitude,
                                    )
                                } else order
                            }
                            _state.value = _state.value.copy(orders = updated)
                        }
                    }
                    "ORDER_COMPLETED" -> {
                        if (msg.orderId.isNotEmpty()) {
                            val filtered = _state.value.orders.filter { it.orderId != msg.orderId }
                            _state.value = _state.value.copy(orders = filtered)
                        }
                    }
                    "ORDER_STATUS_CHANGED", "ORDER_AMENDED" -> {
                        fetchTracking()
                    }
                }
            }
        }
    }
}
