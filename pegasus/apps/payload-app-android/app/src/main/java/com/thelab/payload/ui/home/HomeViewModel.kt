package com.thelab.payload.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.firebase.messaging.FirebaseMessaging
import com.pegasus.payload.BuildConfig
import com.thelab.payload.data.local.SecureStore
import com.thelab.payload.data.model.InjectOrderRequest
import com.thelab.payload.data.model.LiveOrder
import com.thelab.payload.data.model.Manifest
import com.thelab.payload.data.model.NotificationItem
import com.thelab.payload.data.model.QueuedAction
import com.thelab.payload.data.model.RecommendReassignResponse
import com.thelab.payload.data.model.Truck
import com.thelab.payload.data.repository.PayloadRepository
import com.thelab.payload.services.PayloadWebSocket
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.onEach
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import javax.inject.Inject

/**
 * UI state for the master-detail home screen:
 *  - sidebar     (truck list)
 *  - detail      (manifest summary + per-order checklist + seal flow)
 *
 * Phase 4 owns the loading workflow:
 *   DRAFT  ─▶ Start Loading ─▶ LOADING ─▶ tap-check items ─▶ per-order Seal
 *          ─▶ 60-second double-check countdown ─▶ all sealed ─▶ Manifest Seal
 *          ─▶ SEALED success screen.
 */
data class HomeUiState(
    val trucks: List<Truck> = emptyList(),
    val selectedTruckId: String? = null,
    val manifest: Manifest? = null,
    val orders: List<LiveOrder> = emptyList(),
    val selectedOrderId: String? = null,
    /** lineItemId → checked. Local state only; persisted nowhere by design. */
    val checkedItems: Set<String> = emptySet(),
    val sealedOrderIds: Set<String> = emptySet(),
    val dispatchCodes: Map<String, String> = emptyMap(),
    /** Order currently inside the 60s post-seal double-check window. */
    val postSealOrderId: String? = null,
    val postSealCountdown: Int = 0,
    val loadingTrucks: Boolean = false,
    val loadingManifest: Boolean = false,
    val loadingOrders: Boolean = false,
    val startingLoading: Boolean = false,
    val sealingOrderId: String? = null,
    val sealingManifest: Boolean = false,
    val manifestSealed: Boolean = false,
    // ── Phase 5 ──
    /** Order currently being removed via manifest-exception. */
    val exceptionLoadingOrderId: String? = null,
    /** DLQ-escalated message to surface as a one-shot banner. */
    val escalatedMessage: String? = null,
    /** True while POST inject-order is in flight. */
    val injectingOrder: Boolean = false,
    /** Order id currently being re-dispatched (drives Re-Dispatch dialog). */
    val reDispatchOrderId: String? = null,
    val loadingRecommendations: Boolean = false,
    val recommendations: RecommendReassignResponse? = null,
    val reassigning: Boolean = false,
    // ── Phase 6: notifications / WS / offline ──
    val notifications: List<NotificationItem> = emptyList(),
    val unreadCount: Int = 0,
    val online: Boolean = false,
    val showNotificationsPanel: Boolean = false,
    val queuedActions: Int = 0,
    val syncCompleteMessage: String? = null,
    val queuedNoticeMessage: String? = null,
    val error: String? = null,
)

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val repository: PayloadRepository,
    private val secureStore: SecureStore,
    private val webSocket: PayloadWebSocket,
    private val notificationBus: com.thelab.payload.services.NotificationBus,
    private val json: Json,
) : ViewModel() {

    private val _state = MutableStateFlow(HomeUiState())
    val state: StateFlow<HomeUiState> = _state.asStateFlow()

    private var countdownJob: Job? = null

    init {
        refreshTrucks()
        bootstrapPhase6()
        observeNotificationBus()
    }

    private fun observeNotificationBus() {
        notificationBus.openPanel
            .onEach { _state.update { s -> if (s.showNotificationsPanel) s else s.copy(showNotificationsPanel = true) } }
            .launchInVm()
    }

    // ── Phase 6: WebSocket + notifications + FCM bootstrap ──────────────────
    private fun bootstrapPhase6() {
        val token = secureStore.token ?: return
        // Initial inbox fetch + queue restore.
        _state.update { it.copy(queuedActions = repository.readQueue().size) }
        loadNotifications()
        registerFcmToken()
        // Connect WebSocket and observe its frames + reconnects.
        webSocket.connect(token)
        webSocket.online
            .onEach { online -> _state.update { it.copy(online = online) } }
            .launchInVm()
        webSocket.onReconnect
            .onEach {
                loadNotifications()
                flushQueueAndNotify()
            }
            .launchInVm()
        webSocket.frames
            .onEach { frame ->
                // Surface the live frame instantly; full inbox refresh follows on reconnect.
                val item = NotificationItem(
                    notificationId = "live-" + System.currentTimeMillis(),
                    type = frame.type,
                    title = frame.title.orEmpty(),
                    body = frame.body.orEmpty(),
                    channel = frame.channel.orEmpty(),
                    createdAt = "",
                )
                _state.update { it.copy(notifications = listOf(item) + it.notifications, unreadCount = it.unreadCount + 1) }
            }
            .launchInVm()
    }

    private fun <T> kotlinx.coroutines.flow.Flow<T>.launchInVm() {
        viewModelScope.launch { collect {} }
    }

    private fun registerFcmToken() {
        FirebaseMessaging.getInstance().token.addOnSuccessListener { token ->
            if (token.isNullOrEmpty()) return@addOnSuccessListener
            secureStore.firebaseToken = token
            viewModelScope.launch { runCatching { repository.registerDeviceToken(token) } }
        }
    }

    fun loadNotifications() {
        viewModelScope.launch {
            runCatching { repository.loadNotifications() }
                .onSuccess { resp ->
                    _state.update {
                        it.copy(
                            notifications = resp.notifications,
                            unreadCount = resp.unreadCount.toInt(),
                        )
                    }
                }
        }
    }

    fun toggleNotificationsPanel() {
        _state.update { it.copy(showNotificationsPanel = !it.showNotificationsPanel) }
    }

    fun markNotificationRead(id: String) {
        _state.update { s ->
            s.copy(
                notifications = s.notifications.map { if (it.notificationId == id && it.readAt.isNullOrEmpty()) it.copy(readAt = nowIso()) else it },
                unreadCount = (s.unreadCount - 1).coerceAtLeast(0),
            )
        }
        viewModelScope.launch { runCatching { repository.markRead(id) } }
    }

    fun markAllNotificationsRead() {
        _state.update { s ->
            s.copy(
                notifications = s.notifications.map { if (it.readAt.isNullOrEmpty()) it.copy(readAt = nowIso()) else it },
                unreadCount = 0,
            )
        }
        viewModelScope.launch { runCatching { repository.markAllRead() } }
    }

    private fun nowIso(): String = java.time.OffsetDateTime.now().toString()

    private fun flushQueueAndNotify() {
        viewModelScope.launch {
            val (sent, kept) = runCatching { repository.flushQueue(BuildConfig.API_BASE_URL) }
                .getOrDefault(0 to repository.readQueue().size)
            _state.update {
                it.copy(
                    queuedActions = kept,
                    syncCompleteMessage = if (sent > 0) "Synced $sent queued action${if (sent == 1) "" else "s"}." else it.syncCompleteMessage,
                )
            }
        }
    }

    fun clearSyncCompleteMessage() { _state.update { it.copy(syncCompleteMessage = null) } }
    fun clearQueuedNoticeMessage() { _state.update { it.copy(queuedNoticeMessage = null) } }

    // ── Truck list ──────────────────────────────────────────────────────────
    fun refreshTrucks() {
        _state.update { it.copy(loadingTrucks = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.loadTrucks() }
                .onSuccess { trucks ->
                    _state.update { it.copy(trucks = trucks, loadingTrucks = false) }
                    if (_state.value.selectedTruckId == null) {
                        trucks.firstOrNull()?.id?.let { selectTruck(it) }
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(loadingTrucks = false, error = e.message ?: "Failed to load vehicles") }
                }
        }
    }

    fun selectTruck(truckId: String) {
        if (_state.value.selectedTruckId == truckId && _state.value.manifest != null) return
        cancelCountdown()
        _state.update {
            it.copy(
                selectedTruckId = truckId,
                manifest = null,
                orders = emptyList(),
                selectedOrderId = null,
                checkedItems = emptySet(),
                sealedOrderIds = emptySet(),
                dispatchCodes = emptyMap(),
                postSealOrderId = null,
                postSealCountdown = 0,
                loadingManifest = true,
                loadingOrders = true,
                manifestSealed = false,
                error = null,
            )
        }
        viewModelScope.launch {
            runCatching { repository.loadOpenManifest(truckId) }
                .onSuccess { m -> _state.update { it.copy(manifest = m, loadingManifest = false) } }
                .onFailure { e -> _state.update { it.copy(loadingManifest = false, error = e.message ?: "Failed to load manifest") } }
        }
        viewModelScope.launch {
            runCatching { repository.loadOrders(truckId) }
                .onSuccess { orders ->
                    _state.update {
                        it.copy(
                            orders = orders,
                            loadingOrders = false,
                            selectedOrderId = orders.firstOrNull()?.orderId,
                        )
                    }
                }
                .onFailure { e -> _state.update { it.copy(loadingOrders = false, error = e.message ?: "Failed to load orders") } }
        }
    }

    fun refreshManifest() {
        _state.value.selectedTruckId?.let { selectTruck(it) }
    }

    // ── Per-order checklist + seal ──────────────────────────────────────────
    fun selectOrder(orderId: String) {
        _state.update { it.copy(selectedOrderId = orderId) }
    }

    fun toggleItem(lineItemId: String) {
        _state.update {
            val next = it.checkedItems.toMutableSet()
            if (!next.add(lineItemId)) next.remove(lineItemId)
            it.copy(checkedItems = next)
        }
    }

    /** True when every line item of [orderId] is checked AND it isn't sealed yet. */
    fun canSealOrder(orderId: String): Boolean {
        val s = _state.value
        if (orderId in s.sealedOrderIds) return false
        val order = s.orders.firstOrNull { it.orderId == orderId } ?: return false
        if (order.items.isEmpty()) return false
        return order.items.all { it.lineItemId in s.checkedItems }
    }

    fun sealSelectedOrder() {
        val s = _state.value
        val orderId = s.selectedOrderId ?: return
        val truckId = s.selectedTruckId ?: return
        if (!canSealOrder(orderId)) return
        _state.update { it.copy(sealingOrderId = orderId, error = null) }
        viewModelScope.launch {
            runCatching { repository.sealOrder(orderId, truckId) }
                .onSuccess { resp ->
                    _state.update {
                        it.copy(
                            sealingOrderId = null,
                            sealedOrderIds = it.sealedOrderIds + orderId,
                            dispatchCodes = it.dispatchCodes + (orderId to resp.dispatchCode),
                            postSealOrderId = orderId,
                            postSealCountdown = 60,
                        )
                    }
                    startCountdown()
                }
                .onFailure { e ->
                    _state.update { it.copy(sealingOrderId = null, error = e.message ?: "Seal failed") }
                }
        }
    }

    private fun startCountdown() {
        cancelCountdown()
        countdownJob = viewModelScope.launch {
            while (_state.value.postSealCountdown > 0) {
                delay(1_000)
                _state.update { it.copy(postSealCountdown = (it.postSealCountdown - 1).coerceAtLeast(0)) }
            }
            // Countdown done — auto-advance to next unsealed order if any.
            _state.update { s ->
                val nextOrder = s.orders.firstOrNull { it.orderId !in s.sealedOrderIds }
                s.copy(
                    postSealOrderId = null,
                    selectedOrderId = nextOrder?.orderId ?: s.selectedOrderId,
                )
            }
        }
    }

    fun dismissCountdown() {
        cancelCountdown()
        _state.update { s ->
            val nextOrder = s.orders.firstOrNull { it.orderId !in s.sealedOrderIds }
            s.copy(
                postSealOrderId = null,
                postSealCountdown = 0,
                selectedOrderId = nextOrder?.orderId ?: s.selectedOrderId,
            )
        }
    }

    private fun cancelCountdown() {
        countdownJob?.cancel()
        countdownJob = null
    }

    // ── Manifest-level transitions ──────────────────────────────────────────
    fun startLoading() {
        val manifestId = _state.value.manifest?.manifestId ?: return
        if (_state.value.startingLoading) return
        _state.update { it.copy(startingLoading = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.startLoading(manifestId) }
                .onSuccess {
                    _state.update {
                        it.copy(
                            startingLoading = false,
                            manifest = it.manifest?.copy(state = "LOADING"),
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(startingLoading = false, error = e.message ?: "Start loading failed") }
                }
        }
    }

    /** True when every loaded order has been sealed. */
    val allOrdersSealed: Boolean
        get() {
            val s = _state.value
            return s.orders.isNotEmpty() && s.orders.all { it.orderId in s.sealedOrderIds }
        }

    fun sealManifest() {
        val manifestId = _state.value.manifest?.manifestId ?: return
        if (_state.value.sealingManifest) return
        _state.update { it.copy(sealingManifest = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.sealManifest(manifestId) }
                .onSuccess {
                    _state.update {
                        it.copy(
                            sealingManifest = false,
                            manifestSealed = true,
                            manifest = it.manifest?.copy(state = "SEALED"),
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(sealingManifest = false, error = e.message ?: "Manifest seal failed") }
                }
        }
    }

    /** Reset to a fresh state and reload trucks (after All Sealed). */
    fun startNewManifest() {
        cancelCountdown()
        _state.update { HomeUiState() }
        refreshTrucks()
    }

    fun clearError() {
        _state.update { it.copy(error = null) }
    }

    fun clearEscalatedMessage() {
        _state.update { it.copy(escalatedMessage = null) }
    }

    // ── Phase 5: Exception (remove order from manifest) ─────────────────────
    /** Reasons: OVERFLOW | DAMAGED | MANUAL. 3+ OVERFLOW → DLQ escalation. */
    fun reportException(orderId: String, reason: String) {
        val manifestId = _state.value.manifest?.manifestId ?: return
        if (_state.value.exceptionLoadingOrderId != null) return
        _state.update { it.copy(exceptionLoadingOrderId = orderId, error = null) }
        viewModelScope.launch {
            runCatching { repository.manifestException(manifestId, orderId, reason) }
                .onSuccess { resp ->
                    _state.update { s ->
                        val nextOrders = s.orders.filterNot { it.orderId == orderId }
                        val nextSelected = if (s.selectedOrderId == orderId) nextOrders.firstOrNull()?.orderId else s.selectedOrderId
                        s.copy(
                            exceptionLoadingOrderId = null,
                            orders = nextOrders,
                            selectedOrderId = nextSelected,
                            escalatedMessage = if (resp.escalated)
                                "DLQ ESCALATION: order ${orderId.take(8)} escalated after ${resp.overflowCount} overflow attempts."
                            else null,
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(exceptionLoadingOrderId = null, error = e.message ?: "Exception failed") }
                }
        }
    }

    // ── Phase 5: Mid-load order injection ───────────────────────────────────
    fun injectOrder(orderId: String) {
        val trimmed = orderId.trim()
        if (trimmed.isEmpty()) return
        val manifestId = _state.value.manifest?.manifestId ?: return
        val truckId = _state.value.selectedTruckId ?: return
        if (_state.value.injectingOrder) return
        // Phase 6: when offline, persist to the queue and surface a notice.
        if (!_state.value.online) {
            val body = json.encodeToString(InjectOrderRequest.serializer(), InjectOrderRequest(orderId = trimmed))
            repository.enqueue(
                QueuedAction(
                    id = System.currentTimeMillis().toString(),
                    endpoint = "/v1/supplier/manifests/$manifestId/inject-order",
                    method = "POST",
                    body = body,
                    createdAt = System.currentTimeMillis(),
                )
            )
            _state.update {
                it.copy(
                    queuedActions = repository.readQueue().size,
                    queuedNoticeMessage = "Queued offline. Will sync when connection restores.",
                )
            }
            return
        }
        _state.update { it.copy(injectingOrder = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.injectOrder(manifestId, trimmed) }
                .onSuccess {
                    val refreshedManifest = runCatching { repository.loadOpenManifest(truckId) }.getOrNull()
                    val refreshedOrders = runCatching { repository.loadOrders(truckId) }.getOrNull()
                    _state.update { s ->
                        s.copy(
                            injectingOrder = false,
                            manifest = refreshedManifest ?: s.manifest,
                            orders = refreshedOrders ?: s.orders,
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(injectingOrder = false, error = e.message ?: "Inject failed") }
                }
        }
    }

    // ── Phase 5: Re-dispatch (recommend + reassign) ─────────────────────────
    fun openReDispatch(orderId: String) {
        _state.update { it.copy(reDispatchOrderId = orderId, loadingRecommendations = true, recommendations = null, error = null) }
        viewModelScope.launch {
            runCatching { repository.recommendReassign(orderId) }
                .onSuccess { resp ->
                    _state.update { it.copy(loadingRecommendations = false, recommendations = resp) }
                }
                .onFailure { e ->
                    _state.update { it.copy(loadingRecommendations = false, error = e.message ?: "Recommendation failed") }
                }
        }
    }

    fun closeReDispatch() {
        _state.update { it.copy(reDispatchOrderId = null, recommendations = null, loadingRecommendations = false) }
    }

    /** [newDriverId] is the chosen recommendation's driver_id (RouteId == DriverId). */
    fun reassignTo(newDriverId: String) {
        val orderId = _state.value.reDispatchOrderId ?: return
        if (_state.value.reassigning) return
        _state.update { it.copy(reassigning = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.fleetReassign(listOf(orderId), newDriverId) }
                .onSuccess { resp ->
                    val conflict = resp.conflicts.firstOrNull { it.orderId == orderId }
                    if (conflict != null) {
                        _state.update { it.copy(reassigning = false, error = "Reassign conflict: ${conflict.reason}") }
                        return@onSuccess
                    }
                    _state.update { s ->
                        val nextOrders = s.orders.filterNot { it.orderId == orderId }
                        val nextSelected = if (s.selectedOrderId == orderId) nextOrders.firstOrNull()?.orderId else s.selectedOrderId
                        s.copy(
                            reassigning = false,
                            reDispatchOrderId = null,
                            recommendations = null,
                            orders = nextOrders,
                            selectedOrderId = nextSelected,
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(reassigning = false, error = e.message ?: "Reassign failed") }
                }
        }
    }

    override fun onCleared() {
        cancelCountdown()
        webSocket.disconnect()
        super.onCleared()
    }
}
