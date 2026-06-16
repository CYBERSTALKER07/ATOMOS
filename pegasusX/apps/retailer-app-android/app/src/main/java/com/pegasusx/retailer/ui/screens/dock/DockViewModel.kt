package com.pegasusx.retailer.ui.screens.dock

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.RetailerWebSocket
import com.pegasusx.retailer.data.model.TrackingOrder
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import javax.inject.Inject
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException

private val DOCK_STATES = setOf(
    "DISPATCHED",
    "IN_TRANSIT",
    "ARRIVING",
    "ARRIVED",
    "AWAITING_PAYMENT",
)

data class SupplierDockGroup(
    val supplierId: String,
    val supplierName: String,
    val orders: List<TrackingOrder>,
    val totalAmount: Long,
    val hasApproaching: Boolean,
    val hasArrived: Boolean,
)

enum class DockLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class DockUiState(
    val orders: List<TrackingOrder> = emptyList(),
    val expandedSupplierIds: Set<String> = emptySet(),
    val revealedTokenOrderIds: Set<String> = emptySet(),
    val activeQrOrderId: String? = null,
    val autoOpenedOrderIds: Set<String> = emptySet(),
    val isLoading: Boolean = true,
    val isRefreshing: Boolean = false,
    val error: String? = null,
    val loadIssue: DockLoadIssue? = null,
) {
    val activeOrders: List<TrackingOrder>
        get() = orders.filter { it.state in DOCK_STATES }

    val supplierGroups: List<SupplierDockGroup>
        get() {
            val map = LinkedHashMap<String, SupplierDockGroup>()
            for (order in activeOrders) {
                val existing = map[order.supplierId]
                if (existing == null) {
                    map[order.supplierId] = SupplierDockGroup(
                        supplierId = order.supplierId,
                        supplierName = order.supplierName.ifBlank { order.supplierId.take(8) },
                        orders = listOf(order),
                        totalAmount = order.totalAmount,
                        hasApproaching = order.isApproaching || order.state == "ARRIVING",
                        hasArrived = order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT",
                    )
                } else {
                    map[order.supplierId] = existing.copy(
                        orders = existing.orders + order,
                        totalAmount = existing.totalAmount + order.totalAmount,
                        hasApproaching = existing.hasApproaching ||
                            order.isApproaching ||
                            order.state == "ARRIVING",
                        hasArrived = existing.hasArrived ||
                            order.state == "ARRIVED" ||
                            order.state == "AWAITING_PAYMENT",
                    )
                }
            }
            return map.values.sortedByDescending { it.totalAmount }
        }

    val arrivedCount: Int
        get() = activeOrders.count { it.state == "ARRIVED" }

    val approachingCount: Int
        get() = activeOrders.count { it.isApproaching || it.state == "ARRIVING" }

    val activeQrOrder: TrackingOrder?
        get() = activeQrOrderId?.let { id -> orders.find { it.orderId == id } }

    val syncMessage: String?
        get() = when {
            loadIssue == DockLoadIssue.OFFLINE ->
                "Live tracking is offline. Showing the latest cached dock queue."
            loadIssue == DockLoadIssue.RESTRICTED ->
                "Your account cannot view the dock queue right now."
            loadIssue == DockLoadIssue.ERROR && orders.isEmpty() ->
                error ?: "Queue refresh failed. Pull to retry."
            isRefreshing && !isLoading ->
                "Syncing live dock updates..."
            else -> null
        }
}

@HiltViewModel
class DockViewModel @Inject constructor(
    private val api: PegasusApi,
    private val ws: RetailerWebSocket,
) : ViewModel() {

    private val _state = MutableStateFlow(DockUiState())
    val state: StateFlow<DockUiState> = _state.asStateFlow()

    init {
        startPolling()
        observeWebSocket()
        observeReconnects()
    }

    fun refresh() {
        viewModelScope.launch { fetchTracking(refreshing = true) }
    }

    fun toggleSupplier(supplierId: String) {
        _state.update { current ->
            val next = current.expandedSupplierIds.toMutableSet()
            if (supplierId in next) next.remove(supplierId) else next.add(supplierId)
            current.copy(expandedSupplierIds = next)
        }
    }

    fun toggleTokenReveal(orderId: String) {
        _state.update { current ->
            val next = current.revealedTokenOrderIds.toMutableSet()
            if (orderId in next) next.remove(orderId) else next.add(orderId)
            current.copy(revealedTokenOrderIds = next)
        }
    }

    fun showQr(orderId: String) {
        _state.update { it.copy(activeQrOrderId = orderId) }
    }

    fun dismissQr() {
        _state.update { it.copy(activeQrOrderId = null) }
    }

    private fun startPolling() {
        viewModelScope.launch {
            while (true) {
                fetchTracking()
                delay(15_000)
            }
        }
    }

    private suspend fun fetchTracking(refreshing: Boolean = false) {
        _state.update {
            it.copy(
                isLoading = it.orders.isEmpty(),
                isRefreshing = refreshing,
            )
        }
        try {
            val response = api.getTrackingOrders()
            val active = response.orders.filter { it.state !in listOf("COMPLETED", "CANCELLED") }
            _state.update { current ->
                val groups = buildGroups(active.filter { it.state in DOCK_STATES })
                val availableSupplierIds = groups.map { it.supplierId }.toSet()
                val expanded = when {
                    current.expandedSupplierIds.isEmpty() && availableSupplierIds.isNotEmpty() ->
                        availableSupplierIds
                    else -> current.expandedSupplierIds.filter { it in availableSupplierIds }.toSet()
                }
                val tokenEligible = active
                    .filter { it.state == "ARRIVED" || it.state == "AWAITING_PAYMENT" }
                    .map { it.orderId }
                    .toSet()
                val revealed = current.revealedTokenOrderIds.filter { it in tokenEligible }.toSet()
                current.copy(
                    orders = active,
                    expandedSupplierIds = expanded,
                    revealedTokenOrderIds = revealed,
                    isLoading = false,
                    isRefreshing = false,
                    error = null,
                    loadIssue = null,
                ).let { next -> maybeAutoOpenQr(next) }
            }
        } catch (e: Exception) {
            _state.update {
                it.copy(
                    isLoading = false,
                    isRefreshing = false,
                    loadIssue = resolveLoadIssue(e),
                    error = if (it.orders.isEmpty()) "Failed to load dock queue: ${e.message}" else null,
                )
            }
        }
    }

    private fun buildGroups(orders: List<TrackingOrder>): List<SupplierDockGroup> {
        val map = LinkedHashMap<String, SupplierDockGroup>()
        for (order in orders) {
            val existing = map[order.supplierId]
            if (existing == null) {
                map[order.supplierId] = SupplierDockGroup(
                    supplierId = order.supplierId,
                    supplierName = order.supplierName.ifBlank { order.supplierId.take(8) },
                    orders = listOf(order),
                    totalAmount = order.totalAmount,
                    hasApproaching = order.isApproaching || order.state == "ARRIVING",
                    hasArrived = order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT",
                )
            } else {
                map[order.supplierId] = existing.copy(
                    orders = existing.orders + order,
                    totalAmount = existing.totalAmount + order.totalAmount,
                    hasApproaching = existing.hasApproaching ||
                        order.isApproaching ||
                        order.state == "ARRIVING",
                    hasArrived = existing.hasArrived ||
                        order.state == "ARRIVED" ||
                        order.state == "AWAITING_PAYMENT",
                )
            }
        }
        return map.values.sortedByDescending { it.totalAmount }
    }

    private fun maybeAutoOpenQr(state: DockUiState): DockUiState {
        val candidate = state.activeOrders.find { order ->
            order.deliveryToken.isNotBlank() &&
                (order.isApproaching || order.state == "ARRIVED") &&
                order.orderId !in state.autoOpenedOrderIds
        } ?: return state

        return state.copy(
            expandedSupplierIds = state.expandedSupplierIds + candidate.supplierId,
            revealedTokenOrderIds = state.revealedTokenOrderIds + candidate.orderId,
            autoOpenedOrderIds = state.autoOpenedOrderIds + candidate.orderId,
            activeQrOrderId = candidate.orderId,
        )
    }

    private fun observeReconnects() {
        viewModelScope.launch {
            ws.reconnects.collect { fetchTracking(refreshing = true) }
        }
    }

    private fun observeWebSocket() {
        viewModelScope.launch {
            ws.events.collect { msg ->
                when (msg.type) {
                    "DRIVER_APPROACHING" -> {
                        val orderId = msg.orderId
                        if (orderId.isEmpty()) return@collect
                        _state.update { current ->
                            val updated = current.orders.map { order ->
                                if (order.orderId == orderId) {
                                    order.copy(
                                        isApproaching = true,
                                        deliveryToken = msg.deliveryToken?.takeIf { it.isNotBlank() }
                                            ?: order.deliveryToken,
                                        state = if (order.state == "IN_TRANSIT") "ARRIVING" else order.state,
                                    )
                                } else {
                                    order
                                }
                            }
                            maybeAutoOpenQr(current.copy(orders = updated))
                        }
                    }
                    "ORDER_STATUS_CHANGED" -> {
                        val orderId = msg.orderId
                        val newState = msg.state
                        if (orderId.isEmpty() || newState.isBlank()) return@collect
                        _state.update { current ->
                            current.copy(
                                orders = current.orders.map { order ->
                                    if (order.orderId == orderId) order.copy(state = newState) else order
                                },
                            )
                        }
                    }
                    "ORDER_COMPLETED" -> {
                        val orderId = msg.orderId
                        if (orderId.isEmpty()) return@collect
                        _state.update { current ->
                            current.copy(
                                orders = current.orders.filter { it.orderId != orderId },
                                activeQrOrderId = if (current.activeQrOrderId == orderId) null else current.activeQrOrderId,
                            )
                        }
                    }
                    "ORDER_REASSIGNED", "ORDER_AMENDED" -> fetchTracking(refreshing = true)
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): DockLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> DockLoadIssue.RESTRICTED
            error is IOException -> DockLoadIssue.OFFLINE
            else -> DockLoadIssue.ERROR
        }
    }
}
