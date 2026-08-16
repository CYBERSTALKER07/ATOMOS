package com.pegasus.payload.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.firebase.messaging.FirebaseMessaging
import com.pegasus.payload.BuildConfig
import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.model.InjectOrderRequest
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.ManifestExceptionRow
import com.pegasus.payload.data.model.NotificationItem
import com.pegasus.payload.data.model.PulseEvent
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.parseApiErrorPayload
import com.pegasus.payload.data.model.QueuedAction
import com.pegasus.payload.data.model.SealCompletedManifestResult
import com.pegasus.payload.data.model.StatusExplain
import com.pegasus.payload.ui.navigation.HandoffDestination
import com.pegasus.payload.ui.navigation.HandoffPathResolver
import com.pegasus.payload.data.model.Truck
import com.pegasus.payload.data.remote.PayloadApi
import com.pegasus.payload.data.repository.PayloadRepository
import com.pegasus.payload.services.PayloadWebSocket
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.launchIn
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
    /**
     * lineItemId → checked (local UX). G2.B: when PAYLOAD_LOAD_LEDGER_ENABLED,
     * seal is backend-gated via load-ledger scan API — local checks alone must not
     * be treated as durable truth. Prefer POST .../load-ledger/scan then seal.
     */
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
    val reportingMissingItems: Boolean = false,
    val missingItemsReportedMessage: String? = null,
    val showExceptionsPanel: Boolean = false,
    val loadingExceptions: Boolean = false,
    val manifestExceptions: List<ManifestExceptionRow> = emptyList(),
    /** Per-truck sealed order ids so multi-truck batch seal survives truck switches. */
    val sealedOrdersByTruck: Map<String, Set<String>> = emptyMap(),
    /** LOADING manifests with every order sealed — eligible for seal-completed batch. */
    val batchReadyManifestIds: List<String> = emptyList(),
    val batchSealing: Boolean = false,
    val batchSealFailures: List<SealCompletedManifestResult> = emptyList(),
    val handoffNavigationMessage: String? = null,
    val barcodeScanMessage: String? = null,
    val pulseEvents: List<PulseEvent> = emptyList(),
    val pulseLoading: Boolean = false,
    val error: String? = null,
    val errorExplain: StatusExplain? = null,
    val pack: com.pegasus.design.MarketPack? = null,
)

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val repository: PayloadRepository,
    private val api: PayloadApi,
    private val secureStore: SecureStore,
    private val webSocket: PayloadWebSocket,
    private val notificationBus: com.pegasus.payload.services.NotificationBus,
    private val json: Json,
) : ViewModel() {
    private companion object {
        const val MAX_LIVE_NOTIFICATIONS = 200
    }

    private fun deterministicQueueActionId(action: String, entityId: String): String =
        "payload-$action-$entityId"


    private val _state = MutableStateFlow(HomeUiState())
    val state: StateFlow<HomeUiState> = _state.asStateFlow()

    private var countdownJob: Job? = null

    init {
        refreshTrucks()
        refreshPulse()
        bootstrapPhase6()
        observeNotificationBus()
        viewModelScope.launch {
            val pack = com.pegasus.design.MarketPackBinder.fetch(
                BuildConfig.API_BASE_URL,
                secureStore.token.orEmpty(),
            )?.pack
            _state.update { it.copy(pack = pack) }
        }
    }

    fun refreshPulse() {
        viewModelScope.launch {
            _state.update { it.copy(pulseLoading = true) }
            try {
                val response = api.getPulse()
                _state.update { it.copy(pulseEvents = response.events, pulseLoading = false) }
            } catch (_: Exception) {
                _state.update { it.copy(pulseEvents = emptyList(), pulseLoading = false) }
            }
        }
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
        viewModelScope.launch {
            val qCount = repository.queuedActionsCount()
            _state.update { it.copy(queuedActions = qCount) }
        }
        loadNotifications()
        registerFcmToken()
        // Connect WebSocket and observe its frames + reconnects.
        webSocket.connect(token)
        webSocket.online
            .onEach { online -> _state.update { it.copy(online = online) } }
            .launchInVm()
        webSocket.onReconnect
            .onEach {
                viewModelScope.launch {
                    runCatching { repository.reconcileSession(BuildConfig.API_BASE_URL) }
                    recoverInFlightMutations()
                    refreshTrucks(silent = _state.value.trucks.isNotEmpty())
                    refreshManifest(silent = _state.value.manifest != null || _state.value.orders.isNotEmpty())
                    loadNotifications()
                    flushQueueAndNotify()
                }
            }
            .launchInVm()
        webSocket.frames
            .onEach { frame ->
                if (frame.type == "PAYLOAD_SYNC") {
                    refreshManifest(silent = _state.value.manifest != null || _state.value.orders.isNotEmpty())
                    return@onEach
                }

                // Surface the live frame instantly; full inbox refresh follows on reconnect.
                val item = NotificationItem(
                    notificationId = "live-" + System.currentTimeMillis(),
                    type = frame.type,
                    title = frame.title.orEmpty(),
                    body = frame.body.orEmpty(),
                    channel = frame.channel.orEmpty(),
                    createdAt = "",
                )
                _state.update {
                    val next = (listOf(item) + it.notifications).take(MAX_LIVE_NOTIFICATIONS)
                    it.copy(notifications = next, unreadCount = it.unreadCount + 1)
                }
            }
            .launchInVm()
    }

    private fun <T> kotlinx.coroutines.flow.Flow<T>.launchInVm() {
        launchIn(viewModelScope)
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
                .getOrDefault(0 to repository.queuedActionsCount())
            _state.update {
                it.copy(
                    queuedActions = kept,
                    syncCompleteMessage = if (sent > 0) "Synced $sent queued action${if (sent == 1) "" else "s"}." else it.syncCompleteMessage,
                )
            }
        }
    }

    private fun recoverInFlightMutations() {
        _state.update {
            val hadInFlight = it.startingLoading || it.sealingManifest || it.injectingOrder || it.sealingOrderId != null
            it.copy(
                startingLoading = false,
                sealingManifest = false,
                sealingOrderId = null,
                injectingOrder = false,
                syncCompleteMessage = if (hadInFlight) {
                    "Connection restored — loading workflow refreshed from server."
                } else {
                    it.syncCompleteMessage
                },
            )
        }
    }

    fun clearSyncCompleteMessage() { _state.update { it.copy(syncCompleteMessage = null) } }
    fun clearQueuedNoticeMessage() { _state.update { it.copy(queuedNoticeMessage = null) } }

    // ── Truck list ──────────────────────────────────────────────────────────
    fun refreshTrucks(silent: Boolean = false) {
        if (!silent) {
            _state.update { it.copy(loadingTrucks = true, error = null) }
        }
        viewModelScope.launch {
            runCatching { repository.loadTrucks() }
                .onSuccess { trucks ->
                    _state.update { it.copy(trucks = trucks, loadingTrucks = if (silent) it.loadingTrucks else false) }
                    if (_state.value.selectedTruckId == null) {
                        trucks.firstOrNull()?.id?.let { selectTruck(it) }
                    }
                }
                .onFailure { e ->
                    _state.update { state ->
                        state.copy(
                            loadingTrucks = if (silent) state.loadingTrucks else false,
                            error = if (silent) state.error else (e.message ?: "Failed to load vehicles"),
                        )
                    }
                }
        }
    }

    fun selectTruck(truckId: String) {
        if (_state.value.selectedTruckId == truckId && _state.value.manifest != null) return
        cancelCountdown()
        val previousTruckId = _state.value.selectedTruckId
        val sealedSnapshot = _state.value.sealedOrderIds
        val sealedByTruck = if (previousTruckId != null && sealedSnapshot.isNotEmpty()) {
            _state.value.sealedOrdersByTruck + (previousTruckId to sealedSnapshot)
        } else {
            _state.value.sealedOrdersByTruck
        }
        val restoredSealed = sealedByTruck[truckId].orEmpty()
        _state.update {
            it.copy(
                selectedTruckId = truckId,
                manifest = null,
                orders = emptyList(),
                selectedOrderId = null,
                checkedItems = emptySet(),
                sealedOrderIds = restoredSealed,
                sealedOrdersByTruck = sealedByTruck,
                dispatchCodes = emptyMap(),
                postSealOrderId = null,
                postSealCountdown = 0,
                loadingManifest = true,
                loadingOrders = true,
                manifestSealed = false,
                batchReadyManifestIds = emptyList(),
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
        refreshBatchReadyManifests()
    }

    fun refreshBatchReadyManifests() {
        viewModelScope.launch {
            val snapshot = _state.value
            val truckId = snapshot.selectedTruckId ?: return@launch
            val sealedByTruck = snapshot.sealedOrdersByTruck.toMutableMap()
            if (snapshot.sealedOrderIds.isNotEmpty()) {
                sealedByTruck[truckId] = snapshot.sealedOrderIds
            }
            val loading = runCatching { repository.loadLoadingManifests() }.getOrElse { return@launch }
            val ready = mutableListOf<String>()
            for (manifest in loading) {
                // Batch seal-completed is payloader-only; factory seals via /v1/factory/.../seal.
                if (manifest.source == Manifest.SOURCE_FACTORY) continue
                val manifestTruckId = when {
                    manifest.truckId.isNotEmpty() -> manifest.truckId
                    manifest.vehicleId.isNotEmpty() -> manifest.vehicleId
                    else -> continue
                }
                val orders = if (manifestTruckId == truckId) {
                    snapshot.orders
                } else {
                    runCatching { repository.loadOrders(manifestTruckId) }.getOrElse { continue }
                }
                val sealed = if (manifestTruckId == truckId) {
                    snapshot.sealedOrderIds
                } else {
                    sealedByTruck[manifestTruckId].orEmpty()
                }
                if (orders.isNotEmpty() && orders.all { it.orderId in sealed }) {
                    ready.add(manifest.manifestId)
                }
            }
            _state.update {
                it.copy(
                    batchReadyManifestIds = ready.distinct(),
                    sealedOrdersByTruck = sealedByTruck,
                )
            }
        }
    }

    fun finalizeBatchSeal() {
        val manifestIds = _state.value.batchReadyManifestIds
        if (manifestIds.size < 2 || _state.value.batchSealing) return
        _state.update { it.copy(batchSealing = true, error = null, errorExplain = null, batchSealFailures = emptyList()) }
        viewModelScope.launch {
            runCatching { repository.sealAllManifests() }
                .onSuccess { response ->
                    val failures = response.results.filter { it.status.isNotBlank() && it.status != "sealed" }
                    if (failures.isNotEmpty()) {
                        _state.update {
                            it.copy(
                                batchSealing = false,
                                batchSealFailures = failures,
                                error = failures.joinToString("\n") { row ->
                                    row.explain?.title ?: "${row.manifestId}: ${row.status}"
                                },
                                errorExplain = failures.firstNotNullOfOrNull { row -> row.explain },
                            )
                        }
                        return@launch
                    }
                    _state.update {
                        it.copy(
                            batchSealing = false,
                            manifestSealed = true,
                            batchReadyManifestIds = emptyList(),
                            batchSealFailures = emptyList(),
                            sealedOrdersByTruck = emptyMap(),
                            manifest = it.manifest?.copy(state = "SEALED"),
                        )
                    }
                }
                .onFailure { e ->
                    val parsed = parseApiErrorPayload(e)
                    _state.update {
                        it.copy(
                            batchSealing = false,
                            error = parsed.message,
                            errorExplain = parsed.explain,
                        )
                    }
                }
        }
    }

    fun handleHandoffLink(link: String) {
        when (val dest = HandoffPathResolver.resolve(link)) {
            HandoffDestination.TruckList -> {
                _state.update { it.copy(showNotificationsPanel = false) }
            }
            is HandoffDestination.ManifestDetail -> viewModelScope.launch {
                runCatching { repository.loadSupplierManifestDetail(dest.manifestId) }
                    .onSuccess { manifest ->
                        val truckId = manifest.truckId.takeIf { it.isNotBlank() }
                        if (truckId == null) {
                            _state.update {
                                it.copy(
                                    showNotificationsPanel = false,
                                    handoffNavigationMessage = "Open in portal — no native route for this link",
                                )
                            }
                            return@onSuccess
                        }
                        _state.update { it.copy(showNotificationsPanel = false) }
                        selectTruck(truckId)
                    }
                    .onFailure {
                        _state.update {
                            it.copy(
                                showNotificationsPanel = false,
                                handoffNavigationMessage = "Open in portal — no native route for this link",
                            )
                        }
                    }
            }
            is HandoffDestination.OrderDetail -> {
                _state.update { it.copy(showNotificationsPanel = false) }
                if (_state.value.orders.any { it.orderId == dest.orderId }) {
                    selectOrder(dest.orderId)
                } else {
                    _state.update { it.copy(handoffNavigationMessage = "Open in portal — order not on this truck") }
                }
            }
            HandoffDestination.Unresolved -> {
                _state.update {
                    it.copy(
                        showNotificationsPanel = false,
                        handoffNavigationMessage = "Open in portal — no native route for this link",
                    )
                }
            }
        }
    }

    fun clearHandoffNavigationMessage() {
        _state.update { it.copy(handoffNavigationMessage = null) }
    }

    fun refreshManifest(silent: Boolean = false) {
        val truckId = _state.value.selectedTruckId ?: return
        if (!silent) {
            _state.update { it.copy(loadingManifest = true, loadingOrders = true, error = null) }
        }
        viewModelScope.launch {
            runCatching { repository.loadOpenManifest(truckId) }
                .onSuccess { manifest ->
                    _state.update { state ->
                        state.copy(
                            manifest = manifest,
                            loadingManifest = if (silent) state.loadingManifest else false,
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { state ->
                        state.copy(
                            loadingManifest = if (silent) state.loadingManifest else false,
                            error = if (silent) state.error else (e.message ?: "Failed to load manifest"),
                        )
                    }
                }
        }
        viewModelScope.launch {
            runCatching { repository.loadOrders(truckId) }
                .onSuccess { orders ->
                    _state.update { state ->
                        val selectedOrderId = state.selectedOrderId?.takeIf { selected -> orders.any { it.orderId == selected } }
                            ?: orders.firstOrNull()?.orderId
                        state.copy(
                            orders = orders,
                            loadingOrders = if (silent) state.loadingOrders else false,
                            selectedOrderId = selectedOrderId,
                        )
                    }
                }
                .onFailure { e ->
                    _state.update { state ->
                        state.copy(
                            loadingOrders = if (silent) state.loadingOrders else false,
                            error = if (silent) state.error else (e.message ?: "Failed to load orders"),
                        )
                    }
                }
        }
        refreshBatchReadyManifests()
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

    fun clearBarcodeScanMessage() {
        _state.update { it.copy(barcodeScanMessage = null) }
    }

    fun onBarcodeScanned(ean: String) {
        val trimmed = ean.trim()
        if (trimmed.isEmpty()) return
        val orderId = _state.value.selectedOrderId
        if (orderId == null) {
            _state.update { it.copy(barcodeScanMessage = "Select an order first") }
            return
        }
        val order = _state.value.orders.firstOrNull { it.orderId == orderId }
        if (order == null) {
            _state.update { it.copy(barcodeScanMessage = "Order not found") }
            return
        }
        viewModelScope.launch {
            runCatching { repository.lookupBarcode(trimmed) }
                .onSuccess { product ->
                    val match = order.items.firstOrNull { it.skuId == product.skuId }
                    if (match == null) {
                        _state.update { it.copy(barcodeScanMessage = "SKU not on this order") }
                    } else {
                        toggleItem(match.lineItemId)
                        val label = product.name.ifBlank { product.skuId }
                        _state.update { it.copy(barcodeScanMessage = "Checked $label") }
                    }
                }
                .onFailure { e ->
                    _state.update { it.copy(barcodeScanMessage = e.message ?: "Barcode lookup failed") }
                }
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
                    refreshBatchReadyManifests()
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
        val manifest = _state.value.manifest ?: return
        if (_state.value.startingLoading) return
        _state.update { it.copy(startingLoading = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.startLoading(manifest.manifestId, manifest.source) }
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
        val manifest = _state.value.manifest ?: return
        val manifestId = manifest.manifestId
        if (_state.value.sealingManifest) return
        val factoryOnly = manifest.source == Manifest.SOURCE_FACTORY
        val batchIds = _state.value.batchReadyManifestIds
        val manifestIds = if (!factoryOnly && batchIds.size > 1 && manifestId in batchIds) {
            batchIds
        } else {
            listOf(manifestId)
        }
        _state.update { it.copy(sealingManifest = true, error = null, errorExplain = null) }
        viewModelScope.launch {
            runCatching {
                if (factoryOnly) {
                    repository.sealManifest(manifestId, Manifest.SOURCE_FACTORY)
                } else {
                    repository.sealCompletedManifests(manifestIds)
                }
            }
                .onSuccess {
                    _state.update {
                        it.copy(
                            sealingManifest = false,
                            manifestSealed = true,
                            batchReadyManifestIds = if (!factoryOnly && manifestIds.size > 1) emptyList() else it.batchReadyManifestIds,
                            manifest = it.manifest?.copy(state = "SEALED"),
                        )
                    }
                    if (!factoryOnly && manifestIds.size > 1) {
                        refreshBatchReadyManifests()
                    }
                }
                .onFailure { e ->
                    val payload = parseApiErrorPayload(e)
                    _state.update {
                        it.copy(
                            sealingManifest = false,
                            error = payload.message,
                            errorExplain = payload.explain,
                        )
                    }
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
        _state.update { it.copy(error = null, errorExplain = null) }
    }

    fun clearEscalatedMessage() {
        _state.update { it.copy(escalatedMessage = null) }
    }

    fun clearMissingItemsReportedMessage() {
        _state.update { it.copy(missingItemsReportedMessage = null) }
    }

    fun toggleExceptionsPanel() {
        val opening = !_state.value.showExceptionsPanel
        _state.update { it.copy(showExceptionsPanel = opening) }
        if (opening) loadManifestExceptions()
    }

    fun loadManifestExceptions() {
        if (_state.value.loadingExceptions) return
        _state.update { it.copy(loadingExceptions = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.loadManifestExceptions() }
                .onSuccess { rows ->
                    _state.update { it.copy(loadingExceptions = false, manifestExceptions = rows) }
                }
                .onFailure { e ->
                    _state.update {
                        it.copy(
                            loadingExceptions = false,
                            error = e.message ?: "Failed to load manifest exceptions",
                        )
                    }
                }
        }
    }

    /** Edge 33: flag sealed order for missing-item review during post-seal window. */
    fun reportMissingItems(orderId: String) {
        if (_state.value.reportingMissingItems) return
        _state.update { it.copy(reportingMissingItems = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.reportMissingItems(orderId, emptyList()) }
                .onSuccess {
                    _state.update {
                        it.copy(
                            reportingMissingItems = false,
                            missingItemsReportedMessage = "Missing items flagged for review.",
                        )
                    }
                }
                .onFailure { e ->
                    _state.update {
                        it.copy(
                            reportingMissingItems = false,
                            error = e.message ?: "Failed to report missing items",
                        )
                    }
                }
        }
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
            viewModelScope.launch {
                repository.enqueue(
                    QueuedAction(
                        id = deterministicQueueActionId("inject-order", "$manifestId-$trimmed"),
                        endpoint = "/v1/supplier/manifests/$manifestId/inject-order",
                        method = "POST",
                        body = body,
                        createdAt = System.currentTimeMillis(),
                    )
                )
                val count = repository.queuedActionsCount()
                _state.update {
                    it.copy(
                        queuedActions = count,
                        queuedNoticeMessage = "Queued offline. Will sync when connection restores.",
                    )
                }
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
    fun reassignTo(newDriverId: String, isPartial: Boolean) {
        val orderId = _state.value.reDispatchOrderId ?: return
        if (_state.value.reassigning) return
        _state.update { it.copy(reassigning = true, error = null) }
        viewModelScope.launch {
            runCatching { repository.applyReassignOrder(orderId, newDriverId, isPartial = isPartial) }
                .onSuccess { _ ->
                    _state.update { s ->
                        val nextOrders = if (isPartial) s.orders else s.orders.filterNot { it.orderId == orderId }
                        val nextSelected = if (!isPartial && s.selectedOrderId == orderId) nextOrders.firstOrNull()?.orderId else s.selectedOrderId
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
