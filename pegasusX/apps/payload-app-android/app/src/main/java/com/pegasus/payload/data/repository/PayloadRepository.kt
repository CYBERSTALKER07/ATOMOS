package com.pegasus.payload.data.repository

import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.model.DeviceTokenRequest
import com.pegasus.payload.data.model.NotificationItem
import com.pegasus.payload.data.model.ReassignOrderRequest
import com.pegasus.payload.data.model.FleetReassignRequest
import com.pegasus.payload.data.model.FleetReassignResponse
import com.pegasus.payload.data.model.InjectOrderRequest
import com.pegasus.payload.data.model.LiveOrder
import com.pegasus.payload.data.model.Manifest
import com.pegasus.payload.data.model.ManifestExceptionRequest
import com.pegasus.payload.data.model.ManifestExceptionResponse
import com.pegasus.payload.data.model.ManifestExceptionRow
import com.pegasus.payload.data.model.ManifestExceptionsResponse
import com.pegasus.payload.data.model.MissingItemEntry
import com.pegasus.payload.data.model.MissingItemsRequest
import com.pegasus.payload.data.model.MarkReadRequest
import com.pegasus.payload.data.model.NotificationsResponse
import com.pegasus.payload.data.model.QueuedAction
import com.pegasus.payload.data.model.RecommendReassignRequest
import com.pegasus.payload.data.model.RecommendReassignResponse
import com.pegasus.payload.data.model.SealCompletedManifestsRequest
import com.pegasus.payload.data.model.SealCompletedManifestsResponse
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
        val draft = api.supplierManifests(state = "DRAFT").manifests.firstOrNull { it.truckId == truckId }
        if (draft != null) return draft
        return api.supplierManifests(state = "LOADING").manifests.firstOrNull { it.truckId == truckId }
    }

    suspend fun loadSupplierManifestDetail(manifestId: String): Manifest =
        api.supplierManifestDetail(manifestId)

    /** Live orders (with line items) for the selected vehicle. */
    suspend fun loadOrders(vehicleId: String, state: String = "LOADED"): List<LiveOrder> =
        api.orders(vehicleId = vehicleId, state = state)

    suspend fun startLoading(manifestId: String): StatusResponse =
        api.supplierStartLoading(
            manifestId = manifestId,
            idempotencyKey = deterministicIdempotencyKey("supplier-start-loading", manifestId),
        )

    suspend fun sealOrder(orderId: String, terminalId: String): SealOrderResponse =
        api.sealOrder(
            req = SealOrderRequest(orderId = orderId, terminalId = terminalId, manifestCleared = true),
            idempotencyKey = deterministicIdempotencyKey("payload-seal", orderId),
        )

    suspend fun loadLoadingManifests(): List<Manifest> =
        api.supplierManifests(state = "LOADING").manifests

    suspend fun sealCompletedManifests(manifestIds: List<String>): SealCompletedManifestsResponse {
        val ids = manifestIds.map { it.trim() }.filter { it.isNotEmpty() }.distinct()
        require(ids.isNotEmpty()) { "manifest_ids required" }
        return api.sealCompletedManifests(
            req = SealCompletedManifestsRequest(manifestIds = ids),
            idempotencyKey = deterministicIdempotencyKey("seal-completed", ids.sorted().joinToString(",")),
        )
    }

    suspend fun sealManifest(manifestId: String): SealManifestResponse {
        val batch = sealCompletedManifests(listOf(manifestId))
        return SealManifestResponse(
            status = batch.status,
            stopCount = batch.sealedCount,
        )
    }

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
        api.supplierInjectOrder(
            manifestId = manifestId,
            req = InjectOrderRequest(orderId = orderId),
            idempotencyKey = deterministicIdempotencyKey("supplier-inject-order", "$manifestId-$orderId"),
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

    suspend fun applyReassignOrder(orderId: String, toDriverId: String, reason: String = "payload-redispatch"): StatusResponse {
        return api.reassignOrder(
            req = ReassignOrderRequest(orderId = orderId, toDriverId = toDriverId, reason = reason),
            idempotencyKey = deterministicIdempotencyKey("reassign-order", "$orderId-$toDriverId"),
        )
    }

    suspend fun reportMissingItems(orderId: String, items: List<MissingItemEntry> = emptyList()): StatusResponse =
        api.missingItems(
            req = MissingItemsRequest(orderId = orderId, missingItems = items),
            idempotencyKey = deterministicIdempotencyKey("missing-items", orderId),
        )

    suspend fun loadManifestExceptions(limit: Int = 50, offset: Int = 0): List<ManifestExceptionRow> =
        api.manifestExceptionsList(limit = limit, offset = offset).exceptions

    // ── Phase 6: notifications ───────────────────────────────────────────────

    suspend fun loadNotifications(limit: Int = 100): NotificationsResponse {
        val pageSize = limit.coerceIn(1, 100)
        var offset = 0
        val items = mutableListOf<NotificationItem>()
        var unread = 0L
        var hasMore = true
        while (hasMore && offset < 2500) {
            val page = api.notifications(limit = pageSize, offset = offset)
            items.addAll(page.notifications)
            unread = page.unreadCount
            hasMore = page.hasMore
            offset += pageSize
        }
        return NotificationsResponse(items, unread, hasMore = false)
    }

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
