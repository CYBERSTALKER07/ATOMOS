package com.pegasus.payload.data.repository

import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.model.DeviceTokenRequest
import com.pegasus.payload.data.model.FleetReassignRequest
import com.pegasus.payload.data.model.FleetReassignResponse
import com.pegasus.payload.data.model.InjectOrderRequest
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.ManifestExceptionRequest
import com.pegasus.payload.data.model.ManifestExceptionResponse
import com.pegasus.payload.data.model.MarkReadRequest
import com.pegasus.payload.data.model.NotificationsResponse
import com.pegasus.payload.data.model.QueuedAction
import com.pegasus.payload.data.model.RecommendReassignRequest
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.SealManifestResponse
import com.pegasus.payload.data.model.SealOrderRequest
import com.pegasus.payload.data.model.SealOrderResponse
import com.pegasus.payload.data.model.StatusResponse
import com.pegasus.payload.data.model.Truck
import com.pegasus.payload.data.remote.PayloadApi
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
    private fun deterministicIdempotencyKey(action: String, entityId: String): String =
        "payload-$action-$entityId"

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
        api.startLoading(
            manifestId = manifestId,
            idempotencyKey = deterministicIdempotencyKey("start-loading", manifestId),
        )

    suspend fun sealOrder(orderId: String, terminalId: String): SealOrderResponse =
        api.sealOrder(
            req = SealOrderRequest(orderId = orderId, terminalId = terminalId, manifestCleared = true),
            idempotencyKey = deterministicIdempotencyKey("payload-seal", orderId),
        )

    suspend fun sealManifest(manifestId: String): SealManifestResponse =
        api.sealManifest(
            manifestId = manifestId,
            idempotencyKey = deterministicIdempotencyKey("seal-manifest", manifestId),
        )

    // ── Phase 5 ──────────────────────────────────────────────────────────────

    /** Reasons: OVERFLOW | DAMAGED | MANUAL. 3+ OVERFLOW → DLQ escalation. */
    suspend fun manifestException(
        manifestId: String,
        orderId: String,
        reason: String,
        metadata: String = "",
    ): ManifestExceptionResponse = api.manifestException(
        req = ManifestExceptionRequest(manifestId = manifestId, orderId = orderId, reason = reason, metadata = metadata),
        idempotencyKey = deterministicIdempotencyKey("manifest-exception", "$manifestId-$orderId"),
    )

    suspend fun injectOrder(manifestId: String, orderId: String): StatusResponse =
        api.injectOrder(
            manifestId = manifestId,
            req = InjectOrderRequest(orderId = orderId),
            idempotencyKey = deterministicIdempotencyKey("inject-order", "$manifestId-$orderId"),
        )

    suspend fun recommendReassign(orderId: String): RecommendReassignResponse =
        api.recommendReassign(
            req = RecommendReassignRequest(orderId = orderId),
            idempotencyKey = deterministicIdempotencyKey("recommend-reassign", orderId),
        )

    /**
     * Move an order to a new route. In this codebase RouteId == DriverId, so
     * pass the recommended driver_id as [newRouteId].
     */
    suspend fun fleetReassign(orderIds: List<String>, newRouteId: String): FleetReassignResponse =
        api.fleetReassign(
            req = FleetReassignRequest(orderIds = orderIds, newRouteId = newRouteId),
            idempotencyKey = deterministicIdempotencyKey("fleet-reassign", orderIds.sorted().joinToString(",")),
        )

    // ── Phase 6: notifications ───────────────────────────────────────────────

    suspend fun loadNotifications(limit: Int = 50): NotificationsResponse =
        api.notifications(limit = limit)

    suspend fun markRead(id: String): StatusResponse =
        api.markRead(MarkReadRequest(notificationIds = listOf(id)))

    suspend fun markAllRead(): StatusResponse =
        api.markRead(MarkReadRequest(markAll = true))

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
                .header("Idempotency-Key", action.id)
                .method(action.method, action.body.toRequestBody("application/json".toMediaType()))
                .build()
            val outcome = runCatching {
                okHttp.newCall(req).execute().use { response ->
                    val status = response.code
                    when {
                        response.isSuccessful || status == 409 -> QueueReplayOutcome.Sent
                        status == 408 || status == 429 || status >= 500 -> QueueReplayOutcome.Retry
                        else -> QueueReplayOutcome.Drop
                    }
                }
            }.getOrElse { QueueReplayOutcome.Retry }
            when (outcome) {
                QueueReplayOutcome.Sent,
                QueueReplayOutcome.Drop -> sent++
                QueueReplayOutcome.Retry -> remaining.add(action)
            }
        }
        writeQueue(remaining)
        return sent to remaining.size
    }

    private enum class QueueReplayOutcome { Sent, Retry, Drop }
}
