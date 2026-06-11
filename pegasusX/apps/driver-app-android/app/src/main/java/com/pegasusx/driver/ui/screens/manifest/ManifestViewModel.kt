package com.pegasusx.driver.ui.screens.manifest

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.local.OrderDao
import com.pegasusx.driver.data.model.AvailabilityRequest
import com.pegasusx.driver.data.model.DepartRequest
import com.pegasusx.driver.data.model.EarlyCompletePayload
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderEntity
import com.pegasusx.driver.data.model.OrderLineItem
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.model.PendingCollection
import com.pegasusx.driver.data.model.ReorderStopsRequest
import com.pegasusx.driver.data.model.ReturnCompleteRequest
import com.pegasusx.driver.data.remote.ConnectionState
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.data.remote.shouldRefreshManifestOnWsEvent
import com.pegasusx.driver.data.repository.ProfileRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import javax.inject.Inject

data class ManifestUiState(
    val orders: List<Order> = emptyList(),
    val isLoading: Boolean = true,
    val error: String? = null,
    val totalStops: Int = 0,
    val truckStatus: String = "AVAILABLE",
    val isReturning: Boolean = false,
    val isEndingSession: Boolean = false,
    val endSessionError: String? = null,
    val sessionEnded: Boolean = false,
    // LEO: Ghost Stop Prevention
    val manifestId: String? = null,
    val manifestSealed: Boolean = false,
    val manifestState: String? = null, // DRAFT | LOADING | SEALED | DISPATCHED
    val awaitingSeal: Boolean = false,  // true when manifest exists but not SEALED
    val wsConnectionState: ConnectionState = ConnectionState.DISCONNECTED,
    val lastWsRefreshAt: Long? = null,
    val pendingCollections: List<PendingCollection> = emptyList(),
)

@HiltViewModel
class ManifestViewModel @Inject constructor(
    private val api: DriverApi,
    private val orderDao: OrderDao,
    private val json: Json,
    private val profileRepository: ProfileRepository,
    private val driverWebSocket: DriverWebSocket,
) : ViewModel() {

    private val _state = MutableStateFlow(ManifestUiState())
    val state: StateFlow<ManifestUiState> = _state.asStateFlow()

    init {
        loadManifest()
        startProfilePolling()
        observeRealtime()
    }

    private fun observeRealtime() {
        viewModelScope.launch {
            driverWebSocket.connectionState.collect { connection ->
                _state.value = _state.value.copy(wsConnectionState = connection)
            }
        }
        viewModelScope.launch {
            driverWebSocket.messages.collect { message ->
                if (message.type == "SYSTEM_APP_OUTDATED") return@collect
                if (shouldRefreshManifestOnWsEvent(message.type)) {
                    loadManifest()
                    _state.value = _state.value.copy(lastWsRefreshAt = System.currentTimeMillis())
                }
            }
        }
    }

    private fun startProfilePolling() {
        viewModelScope.launch {
            profileRepository.pollProfile().collect { /* TokenHolder updated by repo */ }
        }
    }

    fun loadManifest() {
        viewModelScope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            try {
                val orders = api.getAssignedOrders()
                val pendingCollections = runCatching { api.getPendingCollections() }.getOrDefault(emptyList())

                // Cache to Room
                val entities = orders.map { it.toEntity() }
                orderDao.upsertAll(entities)

                val allComplete = orders.isNotEmpty() && orders.all {
                    it.state == OrderState.COMPLETED || it.state == OrderState.CANCELLED
                }
                val hasInTransit = orders.any {
                    it.state == OrderState.IN_TRANSIT || it.state == OrderState.ARRIVING ||
                    it.state == OrderState.ARRIVED || it.state == OrderState.AWAITING_PAYMENT ||
                    it.state == OrderState.PENDING_CASH_COLLECTION || it.state == OrderState.DISPATCHED
                }
                val derivedStatus = when {
                    _state.value.truckStatus == "RETURNING" -> "RETURNING"
                    allComplete -> "RETURNING"
                    hasInTransit -> "IN_TRANSIT"
                    else -> _state.value.truckStatus
                }

                _state.value = _state.value.copy(
                    orders = orders,
                    pendingCollections = pendingCollections,
                    isLoading = false,
                    totalStops = orders.count { it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED },
                    truckStatus = derivedStatus,
                    isReturning = derivedStatus == "RETURNING",
                    error = null,
                )
            } catch (e: Exception) {
                // Fallback to Room cache
                try {
                    val entities = orderDao.observeAll().first()
                    val orders = entities.map { it.toDomain() }
                    _state.value = _state.value.copy(
                        orders = orders,
                        isLoading = false,
                        totalStops = orders.count { it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED },
                        error = if (orders.isEmpty()) "No internet and no cached data" else null,
                    )
                } catch (_: Exception) {
                    _state.value = _state.value.copy(
                        isLoading = false,
                        error = "Network error: ${e.message}",
                    )
                }
            }
        }
    }

    fun transitionOrder(orderId: String, newState: OrderState) {
        viewModelScope.launch {
            try {
                api.transitionState(orderId, mapOf("state" to newState.name))
                loadManifest()
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun departRoute() {
        val truckId = TokenHolder.vehicleId ?: return
        viewModelScope.launch {
            // LEO: Ghost Stop Prevention — check manifest seal gate before depart
            val manifestId = _state.value.manifestId
            if (manifestId != null && !_state.value.manifestSealed) {
                try {
                    val gate = api.checkManifestGate(manifestId)
                    val cleared = gate.cleared
                    if (!cleared) {
                        val mState = gate.state
                        _state.value = _state.value.copy(
                            error = "Cannot depart: manifest is $mState. Wait for Payloader to seal.",
                            awaitingSeal = true,
                            manifestState = mState
                        )
                        return@launch
                    }
                    _state.value = _state.value.copy(manifestSealed = true, awaitingSeal = false)
                } catch (e: Exception) {
                    // If gate check fails, allow depart (graceful degradation)
                }
            }
            try {
                api.depart(DepartRequest(truckId = truckId))
                _state.value = _state.value.copy(truckStatus = "IN_TRANSIT")
                loadManifest()
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun returnComplete() {
        val truckId = TokenHolder.vehicleId ?: return
        viewModelScope.launch {
            try {
                api.returnComplete(ReturnCompleteRequest(truckId = truckId))
                _state.value = _state.value.copy(truckStatus = "AVAILABLE", isReturning = false)
                loadManifest()
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun moveOrder(fromIndex: Int, toIndex: Int) {
        val currentOrders = _state.value.orders.toMutableList()
        val pendingOrders = currentOrders.filter {
            it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
        }.toMutableList()

        if (fromIndex !in pendingOrders.indices || toIndex !in pendingOrders.indices) return

        val moved = pendingOrders.removeAt(fromIndex)
        pendingOrders.add(toIndex, moved)

        // Rebuild full list: reordered pending + completed/cancelled
        val completedOrders = currentOrders.filter {
            it.state == OrderState.COMPLETED || it.state == OrderState.CANCELLED
        }
        _state.value = _state.value.copy(orders = pendingOrders + completedOrders)

        // Persist to backend
        val routeId = moved.routeId ?: return
        val orderSequence = pendingOrders.map { it.id }
        viewModelScope.launch {
            try {
                api.reorderStops(ReorderStopsRequest(routeId = routeId, orderSequence = orderSequence))
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = "Reorder failed: ${e.message}")
                loadManifest() // Revert to server state
            }
        }
    }

    fun requestEarlyComplete(reason: String, note: String) {
        viewModelScope.launch {
            try {
                api.requestEarlyComplete(EarlyCompletePayload(reason = reason, note = note))
                loadManifest()
            } catch (e: Exception) {
                _state.value = _state.value.copy(
                    error = e.message ?: "Early complete request failed",
                )
            }
        }
    }

    fun endSession(reason: String, note: String? = null) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isEndingSession = true, endSessionError = null)
            try {
                api.setAvailability(AvailabilityRequest(
                    available = false,
                    reason = reason,
                    note = note
                ))
                _state.value = _state.value.copy(isEndingSession = false, sessionEnded = true)
                TokenHolder.clear()
            } catch (e: Exception) {
                _state.value = _state.value.copy(
                    isEndingSession = false,
                    endSessionError = "Failed to end session: ${e.message}"
                )
            }
        }
    }

    val hasActiveOrders: Boolean
        get() = _state.value.orders.any {
            it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
        }

    private fun Order.toEntity() = OrderEntity(
        id = id,
        retailerId = retailerId,
        retailerName = retailerName,
        state = state.name,
        totalAmount = totalAmount,
        deliveryAddress = deliveryAddress,
        latitude = latitude,
        longitude = longitude,
        qrToken = qrToken,
        createdAt = createdAt,
        updatedAt = updatedAt,
        itemsJson = json.encodeToString(items)
    )

    private fun OrderEntity.toDomain() = Order(
        id = id,
        retailerId = retailerId,
        retailerName = retailerName,
        state = try { OrderState.valueOf(state) } catch (_: Exception) { OrderState.PENDING },
        totalAmount = totalAmount,
        deliveryAddress = deliveryAddress,
        latitude = latitude,
        longitude = longitude,
        qrToken = qrToken,
        createdAt = createdAt,
        updatedAt = updatedAt,
        items = try { json.decodeFromString(itemsJson) } catch (_: Exception) { emptyList() },
    )
}
