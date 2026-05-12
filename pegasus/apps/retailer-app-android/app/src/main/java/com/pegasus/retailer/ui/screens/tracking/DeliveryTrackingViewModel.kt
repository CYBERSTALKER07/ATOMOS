package com.pegasus.retailer.ui.screens.tracking

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.api.RetailerWebSocket
import com.pegasus.retailer.data.model.TrackingOrder
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

data class SupplierFilter(
    val supplierId: String,
    val supplierName: String,
)

enum class TrackingLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class TrackingUiState(
    val orders: List<TrackingOrder> = emptyList(),
    val suppliers: List<SupplierFilter> = emptyList(),
    val selectedSupplierIds: Set<String> = emptySet(),
    val isLoading: Boolean = true,
    val error: String? = null,
    val activeFulfillmentCount: Int = 0,
    val loadIssue: TrackingLoadIssue? = null,
) {
    /** Orders filtered by selected suppliers. If none selected, show all. */
    val visibleOrders: List<TrackingOrder>
        get() = if (selectedSupplierIds.isEmpty()) orders
        else orders.filter { it.supplierId in selectedSupplierIds }

    val activeDeliveryCount: Int
        get() = if (activeFulfillmentCount > 0 || orders.isEmpty()) activeFulfillmentCount else orders.size

    val emptyStateMessage: String
        get() = when (loadIssue) {
            TrackingLoadIssue.RESTRICTED -> "Your account cannot view retailer tracking right now"
            TrackingLoadIssue.OFFLINE -> "Live tracking is offline. Reconnect to refresh driver positions"
            TrackingLoadIssue.ERROR -> "Tracking data could not be loaded right now"
            null -> if (activeDeliveryCount > 0) {
                "Active deliveries exist, but live driver location is not available yet"
            } else {
                "No active deliveries with driver location"
            }
        }
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
        viewModelScope.launch { refreshTrackingState() }
    }

    private fun startPolling() {
        viewModelScope.launch {
            while (true) {
                refreshTrackingState()
                delay(15_000) // 15-second polling interval
            }
        }
    }

    private suspend fun refreshTrackingState() {
        fetchTracking()
        fetchActiveFulfillmentCount()
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
                loadIssue = null,
            )
        } catch (e: Exception) {
            _state.value = _state.value.copy(
                isLoading = false,
                loadIssue = resolveLoadIssue(e),
                error = if (_state.value.orders.isEmpty()) "Failed to load: ${e.message}" else null,
            )
        }
    }

    private suspend fun fetchActiveFulfillmentCount() {
        try {
            val response = api.getActiveFulfillments()
            _state.value = _state.value.copy(
                activeFulfillmentCount = response.count,
                loadIssue = null,
            )
        } catch (e: Exception) {
            if (_state.value.orders.isEmpty() && _state.value.activeFulfillmentCount == 0) {
                _state.value = _state.value.copy(loadIssue = resolveLoadIssue(e))
            }
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
                        fetchActiveFulfillmentCount()
                    }
                    "ORDER_STATUS_CHANGED", "ORDER_AMENDED", "ORDER_REASSIGNED" -> {
                        refreshTrackingState()
                    }
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): TrackingLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> TrackingLoadIssue.RESTRICTED
            error is IOException -> TrackingLoadIssue.OFFLINE
            else -> TrackingLoadIssue.ERROR
        }
    }
}
