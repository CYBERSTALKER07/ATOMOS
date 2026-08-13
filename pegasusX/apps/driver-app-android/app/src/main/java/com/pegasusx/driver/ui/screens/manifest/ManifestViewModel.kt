package com.pegasusx.driver.ui.screens.manifest

import android.annotation.SuppressLint
import android.app.Application
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.driver.data.local.OrderDao
import com.pegasusx.driver.data.model.AvailabilityRequest
import com.pegasusx.driver.data.model.DepartRequest
import com.pegasusx.driver.data.model.DriverEarningsResponse
import com.pegasusx.driver.data.model.DriverHistoryRow
import com.pegasusx.driver.data.model.EarlyCompletePayload
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderEntity
import com.pegasusx.driver.data.model.OrderLineItem
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.model.PendingCollection
import com.pegasusx.driver.data.model.RouteCoordinate
import com.pegasusx.driver.data.model.RouteStep
import com.pegasusx.driver.data.model.StatusExplain
import com.pegasusx.driver.data.model.ReorderStopsRequest
import com.pegasusx.driver.data.telemetry.NavigationCue
import com.pegasusx.driver.data.telemetry.NavigationCueAnnouncer
import com.pegasusx.driver.data.telemetry.RouteDeviationAction
import com.pegasusx.driver.data.telemetry.RouteDeviationTracker
import com.pegasusx.driver.data.telemetry.advanceNavigationStepIndex
import com.pegasusx.driver.data.telemetry.resolveNavigationCue
import com.pegasusx.driver.data.telemetry.shouldAnnounceManeuverAdvance
import com.pegasusx.driver.data.model.ReturnCompleteRequest
import com.pegasusx.driver.data.model.SubmitCashReconciliationRequest
import com.pegasusx.driver.data.model.UpdateOrderDuringDeliveryRequest
import com.pegasusx.driver.data.remote.ConnectionState
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.DriverWebSocket
import com.pegasusx.driver.data.remote.DRIVER_RECONNECT_RECOVERY_HINT
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.util.DriverIdempotencyKeys
import com.pegasusx.driver.data.remote.reconcileDriverSession
import com.pegasusx.driver.services.OfflineSyncScheduler
import com.pegasusx.driver.data.repository.ProfileRepository
import com.pegasusx.driver.ui.screens.map.resolveActiveOrder
import com.pegasusx.driver.data.remote.shouldRefreshManifestOnWsEvent
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
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
    val gateExplain: StatusExplain? = null,
    val wsConnectionState: ConnectionState = ConnectionState.DISCONNECTED,
    val lastWsRefreshAt: Long? = null,
    val pendingCollections: List<PendingCollection> = emptyList(),
    /** Phase 6: open fiscal soft-freezes cash bag / return-complete. */
    val openFiscalCount: Long = 0,
    val cashBagFrozen: Boolean = false,
    val showCashRecon: Boolean = false,
    val declaredCashMinor: String = "",
    val routeGeometry: List<RouteCoordinate> = emptyList(),
    val routeSteps: List<RouteStep> = emptyList(),
    val navigationCue: NavigationCue? = null,
    val deliveryEdgeMessage: String? = null,
    val isRequestingEarlyComplete: Boolean = false,
    val historyRows: List<DriverHistoryRow> = emptyList(),
    val earnings: DriverEarningsResponse? = null,
)

@HiltViewModel
class ManifestViewModel @Inject constructor(
    @ApplicationContext private val appContext: android.content.Context,
    private val app: Application,
    private val api: DriverApi,
    private val orderDao: OrderDao,
    private val json: Json,
    private val profileRepository: ProfileRepository,
    private val driverWebSocket: DriverWebSocket,
) : ViewModel() {

    private val _state = MutableStateFlow(ManifestUiState())
    val state: StateFlow<ManifestUiState> = _state.asStateFlow()
    private var navigationStepIndex = 0
    private val navigationCueAnnouncer = NavigationCueAnnouncer(app)
    private val routeDeviationTracker = RouteDeviationTracker()
    private var isReroutingRoute = false
    private val fusedClient = LocationServices.getFusedLocationProviderClient(app)

    init {
        loadManifest()
        startProfilePolling()
        observeRealtime()
        loadEarningsAndHistory()
    }

    fun loadEarningsAndHistory() {
        viewModelScope.launch {
            try {
                val earnings = api.getEarnings()
                _state.update { it.copy(earnings = earnings) }
            } catch (_: Exception) { }
            try {
                val history = api.getHistory()
                _state.update { it.copy(historyRows = history.rows) }
            } catch (_: Exception) { }
        }
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
                    loadManifest(silent = true)
                    _state.value = _state.value.copy(lastWsRefreshAt = System.currentTimeMillis())
                }
            }
        }
        viewModelScope.launch {
            driverWebSocket.onReconnect.collect {
                reconcileAfterReconnect()
            }
        }
    }

    private fun reconcileAfterReconnect() {
        viewModelScope.launch { recoverInFlightMutation() }
    }

    private suspend fun recoverInFlightMutation() {
        val hadInFlight = _state.value.isRequestingEarlyComplete
        runCatching { reconcileDriverSession(api) }
        loadManifest(silent = true)
        OfflineSyncScheduler.enqueue(appContext)
        if (hadInFlight) {
            _state.update {
                it.copy(
                    isRequestingEarlyComplete = false,
                    error = DRIVER_RECONNECT_RECOVERY_HINT,
                )
            }
        }
    }

    private fun startProfilePolling() {
        viewModelScope.launch {
            profileRepository.pollProfile().collect { /* TokenHolder updated by repo */ }
        }
    }

    fun loadManifest(silent: Boolean = false) {
        viewModelScope.launch {
            if (!silent) {
                _state.value = _state.value.copy(isLoading = true, error = null)
            } else {
                _state.value = _state.value.copy(error = null)
            }
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
                loadRouteGeometry(resolveActiveOrder(orders)?.routeId)
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

    fun loadRouteGeometry(routeId: String?) {
        if (routeId.isNullOrBlank()) {
            navigationStepIndex = 0
            navigationCueAnnouncer.stop()
            routeDeviationTracker.reset()
            _state.value = _state.value.copy(
                routeGeometry = emptyList(),
                routeSteps = emptyList(),
                navigationCue = null,
            )
            return
        }
        viewModelScope.launch {
            runCatching { api.getRouteGeometry(routeId) }
                .onSuccess { response ->
                    navigationStepIndex = 0
                    navigationCueAnnouncer.stop()
                    routeDeviationTracker.reset()
                    _state.value = _state.value.copy(
                        routeGeometry = response.coordinates,
                        routeSteps = response.steps,
                        navigationCue = null,
                    )
                }
                .onFailure {
                    navigationStepIndex = 0
                    navigationCueAnnouncer.stop()
                    routeDeviationTracker.reset()
                    _state.value = _state.value.copy(
                        routeGeometry = emptyList(),
                        routeSteps = emptyList(),
                        navigationCue = null,
                    )
                }
        }
    }

    fun checkRouteDeviation(routeId: String?, lat: Double, lng: Double) {
        if (routeId.isNullOrBlank() || isReroutingRoute) {
            return
        }
        val action = routeDeviationTracker.evaluate(
            nowMs = System.currentTimeMillis(),
            lat = lat,
            lng = lng,
            polyline = _state.value.routeGeometry,
        )
        if (action == RouteDeviationAction.Reroute) {
            rerouteRouteGeometry(routeId, lat, lng)
        }
    }

    private fun rerouteRouteGeometry(routeId: String, lat: Double, lng: Double) {
        isReroutingRoute = true
        viewModelScope.launch {
            runCatching {
                api.getRouteGeometry(
                    routeId = routeId,
                    includeSteps = true,
                    fromLat = lat,
                    fromLng = lng,
                    reroute = true,
                )
            }.onSuccess { response ->
                navigationStepIndex = 0
                navigationCueAnnouncer.stop()
                routeDeviationTracker.reset()
                _state.value = _state.value.copy(
                    routeGeometry = response.coordinates,
                    routeSteps = response.steps,
                    navigationCue = null,
                )
            }.onFailure {
                routeDeviationTracker.reset()
            }
            isReroutingRoute = false
        }
    }

    fun updateNavigationCue(lat: Double?, lng: Double?) {
        val steps = _state.value.routeSteps
        if (lat == null || lng == null || steps.isEmpty()) {
            _state.value = _state.value.copy(navigationCue = null)
            return
        }
        val previousIndex = navigationStepIndex
        navigationStepIndex = advanceNavigationStepIndex(navigationStepIndex, steps, lat, lng)
        val cue = resolveNavigationCue(steps, navigationStepIndex, lat, lng)
        if (cue != null && shouldAnnounceManeuverAdvance(previousIndex, navigationStepIndex)) {
            navigationCueAnnouncer.onManeuverAdvanced(cue)
        }
        _state.value = _state.value.copy(navigationCue = cue)
    }

    override fun onCleared() {
        navigationCueAnnouncer.shutdown()
        super.onCleared()
    }

    fun transitionOrder(orderId: String, newState: OrderState) {
        // G1.C: PATCH /v1/orders/{id}/state is always 501 — route to real edges only.
        viewModelScope.launch {
            try {
                when (newState) {
                    OrderState.ARRIVED -> {
                        api.markArrived(
                            mapOf("order_id" to orderId),
                            DriverIdempotencyKeys.markArrived(orderId),
                        )
                        loadManifest(silent = true)
                    }
                    else -> {
                        _state.value = _state.value.copy(
                            error = "Use delivery edges (arrive, depart, cash/card/credit) — not generic state patch (${newState.name})",
                        )
                    }
                }
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
                    val response = api.checkManifestGate(manifestId)
                    val gate = response.body()
                    if (response.isSuccessful && gate?.cleared == true) {
                        _state.value = _state.value.copy(manifestSealed = true, awaitingSeal = false, gateExplain = null)
                    } else if (gate != null && !gate.cleared) {
                        _state.value = _state.value.copy(
                            error = gate.message ?: gate.explain?.summary ?: "Cannot depart: manifest is ${gate.state}. Wait for Payloader to seal.",
                            awaitingSeal = true,
                            manifestState = gate.state,
                            gateExplain = gate.explain,
                        )
                        return@launch
                    }
                } catch (e: Exception) {
                    // If gate check fails, allow depart (graceful degradation)
                }
            }
            try {
                api.depart(DepartRequest(truckId = truckId), DriverIdempotencyKeys.depart(truckId))
                _state.value = _state.value.copy(truckStatus = "IN_TRANSIT")
                loadManifest(silent = true)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun refreshOpenFiscal() {
        viewModelScope.launch {
            try {
                val snap = api.getOpenFiscal()
                _state.update {
                    it.copy(
                        openFiscalCount = snap.openFiscalCount,
                        cashBagFrozen = snap.cashBagFrozen || snap.openFiscalCount > 0,
                    )
                }
            } catch (_: Exception) {
                // non-fatal — return-complete still enforces server-side
            }
        }
    }

    fun returnComplete() {
        val truckId = TokenHolder.vehicleId ?: return
        viewModelScope.launch {
            try {
                // Soft-freeze: surface open fiscal before hitting the server.
                try {
                    val snap = api.getOpenFiscal()
                    if (snap.cashBagFrozen || snap.openFiscalCount > 0) {
                        _state.update {
                            it.copy(
                                openFiscalCount = snap.openFiscalCount,
                                cashBagFrozen = true,
                                error = "Cash bag frozen: ${snap.openFiscalCount} order(s) still fiscalizing. Retry fiscal or call supervisor.",
                            )
                        }
                        return@launch
                    }
                } catch (_: Exception) { /* proceed; server is source of truth */ }

                api.returnComplete(
                    ReturnCompleteRequest(truckId = truckId),
                    DriverIdempotencyKeys.returnComplete(truckId),
                )
                _state.value = _state.value.copy(
                    truckStatus = "AVAILABLE",
                    isReturning = false,
                    cashBagFrozen = false,
                    openFiscalCount = 0,
                )
                loadManifest(silent = true)
            } catch (e: Exception) {
                val msg = e.message.orEmpty()
                val frozen = msg.contains("open_fiscal_block", ignoreCase = true)
                val cashRecon = msg.contains("cash_reconciliation_required", ignoreCase = true)
                _state.value = _state.value.copy(
                    error = when {
                        frozen -> "Cash bag frozen: clear fiscalizing orders before ending shift."
                        cashRecon -> "Submit cash reconciliation before ending shift."
                        else -> e.message
                    },
                    cashBagFrozen = frozen || _state.value.cashBagFrozen,
                    showCashRecon = cashRecon || _state.value.showCashRecon,
                )
            }
        }
    }

    fun submitCashReconciliation() {
        viewModelScope.launch {
            val minor = _state.value.declaredCashMinor.filter { it.isDigit() }.toLongOrNull() ?: 0L
            try {
                api.submitCashReconciliation(
                    SubmitCashReconciliationRequest(declaredCashMinor = minor),
                    DriverIdempotencyKeys.cashReconciliation(minor),
                )
                _state.value = _state.value.copy(
                    showCashRecon = false,
                    declaredCashMinor = "",
                    deliveryEdgeMessage = "Cash reconciliation submitted",
                    error = null,
                )
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = e.message)
            }
        }
    }

    fun updateDeclaredCash(value: String) {
        _state.value = _state.value.copy(declaredCashMinor = value)
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
                api.reorderStops(
                    ReorderStopsRequest(routeId = routeId, orderSequence = orderSequence),
                    DriverIdempotencyKeys.routeReorder(routeId, orderSequence),
                )
                loadRouteGeometry(routeId)
            } catch (e: Exception) {
                _state.value = _state.value.copy(error = "Reorder failed: ${e.message}")
                loadManifest(silent = true) // Revert to server state
            }
        }
    }

    fun requestEarlyComplete(reason: String, note: String) {
        viewModelScope.launch {
            _state.update { it.copy(isRequestingEarlyComplete = true, error = null) }
            try {
                api.requestEarlyComplete(
                    EarlyCompletePayload(reason = reason, note = note),
                    DriverIdempotencyKeys.requestEarlyComplete(reason),
                )
                loadManifest(silent = true)
            } catch (e: Exception) {
                _state.update {
                    it.copy(error = e.message ?: "Early complete request failed")
                }
            } finally {
                _state.update { it.copy(isRequestingEarlyComplete = false) }
            }
        }
    }

    fun updateOrderDuringDelivery(orderId: String) {
        // G1.C: mid-delivery endpoint has no durable writer — use delivery correction / amend.
        _state.value = _state.value.copy(
            deliveryEdgeMessage = null,
            error = "Use delivery correction (amend / missing items) — mid-delivery update is not implemented",
        )
    }

    fun endSession(reason: String, note: String? = null) {
        viewModelScope.launch {
            _state.value = _state.value.copy(isEndingSession = true, endSessionError = null)
            try {
                api.setAvailability(
                    AvailabilityRequest(
                        available = false,
                        reason = reason,
                        note = note,
                    ),
                    DriverIdempotencyKeys.availability(onShift = false, reason = reason, note = note),
                )
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
