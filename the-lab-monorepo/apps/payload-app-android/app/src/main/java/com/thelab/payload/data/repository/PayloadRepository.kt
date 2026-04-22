package com.thelab.payload.data.repository

import com.thelab.payload.data.local.SecureStore
import com.thelab.payload.data.model.DeviceTokenRequest
import com.thelab.payload.data.model.FleetReassignRequest
import com.thelab.payload.data.model.FleetReassignResponse
import com.thelab.payload.data.model.InjectOrderRequest
import com.thelab.payload.data.model.LiveOrder
import com.thelab.payload.data.model.Manifest
import com.thelab.payload.data.model.ManifestExceptionRequest
import com.thelab.payload.data.model.ManifestExceptionResponse
import com.thelab.payload.data.model.MarkReadRequest
import com.thelab.payload.data.model.NotificationsResponse
import com.thelab.payload.data.model.QueuedAction
import com.thelab.payload.data.model.RecommendReassignRequest
import com.thelab.payload.data.model.RecommendReassignResponse
import com.thelab.payload.data.model.SealManifestResponse
import com.thelab.payload.data.model.SealOrderRequest
import com.thelab.payload.data.model.SealOrderResponse
import com.thelab.payload.data.model.StatusResponse
import com.thelab.payload.data.model.Truck
import com.thelab.payload.data.remote.PayloadApi
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Phase 3 + 4 + 5 + 6 repo. Read paths (trucks, manifest, orders), loading
 * workflow mutations (start-loading, per-order seal, manifest seal), Phase 5
 * mid-load operations (exception, inject-order, re-dispatch), and Phase 6
 * (notifications inbox + offline action queue + FCM token registration).
 */
@Singleton
class PayloadRepository @Inject constructor(
    private val api: PayloadApi,
    private val secureStore: SecureStore,
    private val json: Json,
    private val okHttp: OkHttpClient,
) {
    suspend fun loadTrucks(): List<Truck> = api.trucks()

    /** Draft OR currently-loading manifest for the selected truck, or null. */
    suspend fun loadOpenManifest(truckId: String): Manifest? {
        val draft = api.manifests(state = "DRAFT", truckId = truckId).manifests.firstOrNull()
        if (draft != null) return draft
        return api.manifests(state = "LOADING", truckId = truckId).manifests.firstOrNull()
    }

    suspend fun loadManifestDetail(manifestId: String): Manifest =
        api.manifestDetail(manifestId)

    /** Live orders (with line items) for the selected vehicle. */
    suspend fun loadOrders(vehicleId: String, state: String = "LOADED"): List<LiveOrder> =
        api.orders(vehicleId = vehicleId, state = state)

    suspend fun startLoading(manifestId: String): StatusResponse =
        api.startLoading(manifestId)

    suspend fun sealOrder(orderId: String, terminalId: String): SealOrderResponse =
        api.sealOrder(SealOrderRequest(orderId = orderId, terminalId = terminalId, manifestCleared = true))

    suspend fun sealManifest(manifestId: String): SealManifestResponse =
        api.sealManifest(manifestId)

    // ── Phase 5 ──────────────────────────────────────────────────────────────

    /** Reasons: OVERFLOW | DAMAGED | MANUAL. 3+ OVERFLOW → DLQ escalation. */
    suspend fun manifestException(
        manifestId: String,
        orderId: String,
        reason: String,
        metadata: String = "",
    ): ManifestExceptionResponse = api.manifestException(
        ManifestExceptionRequest(manifestId = manifestId, orderId = orderId, reason = reason, metadata = metadata)
    )

    suspend fun injectOrder(manifestId: String, orderId: String): StatusResponse =
        api.injectOrder(manifestId, InjectOrderRequest(orderId = orderId))

    suspend fun recommendReassign(orderId: String): RecommendReassignResponse =
        api.recommendReassign(RecommendReassignRequest(orderId = orderId))

    /**
     * Move an order to a new route. In this codebase RouteId == DriverId, so
     * pass the recommended driver_id as [newRouteId].
     */
    suspend fun fleetReassign(orderIds: List<String>, newRouteId: String): FleetReassignResponse =
        api.fleetReassign(FleetReassignRequest(orderIds = orderIds, newRouteId = newRouteId))

    // ── Phase 6: notifications ───────────────────────────────────────────────

    suspend fun loadNotifications(limit: Int = 50): NotificationsResponse =
        api.notifications(limit = limit)

    suspend fun markRead(id: String): StatusResponse =
        api.markRead(MarkReadRequest(notificationIds = listOf(id)))

    suspend fun markAllRead(): StatusResponse =
        api.markRead(MarkReadRequest(all = true))

    // ── Phase 6: FCM token lifecycle ─────────────────────────────────────────

    suspend fun registerDeviceToken(token: String): StatusResponse =
        api.registerDeviceToken(DeviceTokenRequest(token = token, platform = "ANDROID"))

    suspend fun unregisterDeviceToken(): StatusResponse =
        api.unregisterDeviceToken(platform = "ANDROID")

    // ── Phase 6: offline action queue ────────────────────────────────────────
    // Persists a small queue of write actions (currently only inject-order)
    // in EncryptedSharedPreferences. Drained on WS reconnect via [flushQueue].

    private val queueSerializer = ListSerializer(QueuedAction.serializer())

    fun readQueue(): List<QueuedAction> =
        secureStore.offlineQueueJson?.let {
            runCatching { json.decodeFromString(queueSerializer, it) }.getOrDefault(emptyList())
        } ?: emptyList()

    fun writeQueue(items: List<QueuedAction>) {
        secureStore.offlineQueueJson = if (items.isEmpty()) null
        else json.encodeToString(queueSerializer, items)
    }

    fun enqueue(action: QueuedAction) {
        writeQueue(readQueue() + action)
    }

    /**
     * Drain the persisted offline queue. Returns (sent, kept) pair. Kept items
     * are re-persisted for next reconnect attempt.
     */
    suspend fun flushQueue(baseUrl: String): Pair<Int, Int> {
        val current = readQueue()
        if (current.isEmpty()) return 0 to 0
        val token = secureStore.token ?: return 0 to current.size
        val remaining = mutableListOf<QueuedAction>()
        var sent = 0
        for (action in current) {
            val req = Request.Builder()
                .url("${baseUrl.trimEnd('/')}${action.endpoint}")
                .header("Authorization", "Bearer $token")
                .header("Content-Type", "application/json")
                .method(action.method, action.body.toRequestBody("application/json".toMediaType()))
                .build()
            val ok = runCatching { okHttp.newCall(req).execute().use { it.isSuccessful } }
                .getOrDefault(false)
            if (ok) sent++ else remaining.add(action)
        }
        writeQueue(remaining)
        return sent to remaining.size
    }
}


