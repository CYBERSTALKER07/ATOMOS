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
import com.pegasus.payload.util.PayloadIdempotencyKeys
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import com.pegasus.payload.data.local.QueuedActionDao
import com.pegasus.payload.data.local.QueuedActionEntity
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
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
    private val queuedActionDao: QueuedActionDao,
) {
    private fun deterministicIdempotencyKey(action: String, entityId: String): String =
        PayloadIdempotencyKeys.key(action, entityId)

    /** Parallel refetch of authoritative payload snapshots after WS reconnect. */
    suspend fun reconcileSession(baseUrl: String) {
        val token = secureStore.token ?: return
        val endpoints = listOf(
            "/v1/payloader/trucks",
            "/v1/payloader/manifests",
            "/v1/factory/manifests",
        )
        for (endpoint in endpoints) {
            runCatching {
                okHttp.newCall(
                    Request.Builder()
                        .url("${baseUrl.trimEnd('/')}$endpoint")
                        .header("Authorization", "Bearer $token")
                        .get()
                        .build(),
                ).execute().use { /* drain */ }
            }
        }
    }

    suspend fun loadTrucks(): List<Truck> = api.trucks()

    /** All board-state manifests (DRAFT/LOADING/SEALED/DISPATCHED). VU lives here — never GET /v1/payload/capacity/. */
    suspend fun listBoardManifests(): List<Manifest> {
        val out = linkedMapOf<String, Manifest>()
        for (state in listOf("DRAFT", "LOADING", "SEALED", "DISPATCHED")) {
            for (m in listLoadingBayManifests(state)) {
                out.putIfAbsent(m.manifestId, m)
            }
        }
        return out.values.toList()
    }

    /**
     * Payloader + factory loading-bay manifests (P1-18 / P2-25 Class A bridge).
     * Dedupes by manifest_id; payloader wins on collision.
     */
    suspend fun listLoadingBayManifests(state: String = "DRAFT"): List<Manifest> {
        val out = linkedMapOf<String, Manifest>()
        val payloader = runCatching { api.manifests(state = state).manifests }
            .getOrElse {
                runCatching { api.supplierManifests(state = state).manifests }.getOrDefault(emptyList())
            }
        for (m in payloader) {
            out.putIfAbsent(m.manifestId, m.copy(source = Manifest.SOURCE_PAYLOADER))
        }
        val factory = runCatching { api.factoryManifests(state = state).manifests }.getOrDefault(emptyList())
        for (m in factory) {
            out.putIfAbsent(m.manifestId, m.copy(source = Manifest.SOURCE_FACTORY))
        }
        return out.values.toList()
    }

    /** Current board-state manifest for the selected truck, or null. */
    suspend fun loadOpenManifest(truckId: String): Manifest? {
        for (state in listOf("DRAFT", "LOADING", "SEALED", "DISPATCHED")) {
            listLoadingBayManifests(state).firstOrNull { it.matchesTruck(truckId) }?.let { return it }
        }
        return null
    }

    suspend fun loadSupplierManifestDetail(manifestId: String, source: String = Manifest.SOURCE_PAYLOADER): Manifest =
        when (source) {
            Manifest.SOURCE_FACTORY -> api.factoryManifestDetail(manifestId).copy(source = Manifest.SOURCE_FACTORY)
            else -> runCatching { api.manifestDetail(manifestId).copy(source = Manifest.SOURCE_PAYLOADER) }
                .getOrElse {
                    runCatching { api.supplierManifestDetail(manifestId).copy(source = Manifest.SOURCE_PAYLOADER) }
                        .getOrElse {
                            api.factoryManifestDetail(manifestId).copy(source = Manifest.SOURCE_FACTORY)
                        }
                }
        }

    /** Live orders (with line items) for the selected vehicle. */
    suspend fun loadOrders(vehicleId: String, state: String = "LOADED"): List<LiveOrder> =
        api.orders(vehicleId = vehicleId, state = state)

    suspend fun startLoading(manifestId: String, source: String = Manifest.SOURCE_PAYLOADER): StatusResponse {
        val key = deterministicIdempotencyKey("supplier-start-loading", manifestId)
        if (source == Manifest.SOURCE_FACTORY) {
            return api.factoryStartLoading(manifestId = manifestId, idempotencyKey = key)
        }
        return runCatching {
            api.supplierStartLoading(manifestId = manifestId, idempotencyKey = key)
        }.getOrElse {
            api.factoryStartLoading(manifestId = manifestId, idempotencyKey = key)
        }
    }

    suspend fun sealOrder(orderId: String, terminalId: String): SealOrderResponse =
        api.sealOrder(
            req = SealOrderRequest(orderId = orderId, terminalId = terminalId, manifestCleared = true),
            idempotencyKey = deterministicIdempotencyKey("payload-seal", orderId),
        )

    suspend fun loadLoadingManifests(): List<Manifest> =
        listLoadingBayManifests("LOADING")

    suspend fun sealCompletedManifests(manifestIds: List<String>): SealCompletedManifestsResponse {
        val ids = manifestIds.map { it.trim() }.filter { it.isNotEmpty() }.distinct()
        require(ids.isNotEmpty()) { "manifest_ids required" }
        return api.sealCompletedManifests(
            req = SealCompletedManifestsRequest(manifestIds = ids),
            idempotencyKey = deterministicIdempotencyKey("seal-completed", ids.sorted().joinToString(",")),
        )
    }

    suspend fun sealAllManifests(): SealCompletedManifestsResponse {
        return api.sealAllManifests(
            idempotencyKey = deterministicIdempotencyKey("seal-all", "payloader"),
        )
    }

    suspend fun sealManifest(manifestId: String, source: String = Manifest.SOURCE_PAYLOADER): SealManifestResponse {
        if (source == Manifest.SOURCE_FACTORY) {
            return api.factorySealManifest(
                manifestId = manifestId,
                idempotencyKey = deterministicIdempotencyKey("seal-completed", manifestId),
            )
        }
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
            idempotencyKey = PayloadIdempotencyKeys.recommendReassign(orderId),
        )

    /**
     * Move an order to a new route. In this codebase RouteId == DriverId, so
     * pass the recommended driver_id as [newRouteId].
     */
    suspend fun fleetReassign(orderIds: List<String>, newRouteId: String): FleetReassignResponse =
        api.fleetReassign(
            req = FleetReassignRequest(orderIds = orderIds, newRouteId = newRouteId),
            idempotencyKey = PayloadIdempotencyKeys.fleetReassign(orderIds),
        )

    suspend fun applyReassignOrder(orderId: String, toDriverId: String, reason: String = "payload-redispatch", isPartial: Boolean = false): StatusResponse {
        return api.reassignOrder(
            req = ReassignOrderRequest(orderId = orderId, toDriverId = toDriverId, reason = reason, isPartial = isPartial),
            idempotencyKey = PayloadIdempotencyKeys.applyReassign(orderId, toDriverId) + (if (isPartial) "-partial" else ""),
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
    // in Room Database. Drained on WS reconnect via WorkManager or flushQueue.

    suspend fun queuedActionsCount(): Int = queuedActionDao.count()

    fun queuedScanCountFlow(pathContains: String): Flow<Int> {
        return queuedActionDao.countByEndpointFlow(pathContains)
    }

    suspend fun enqueue(action: QueuedAction) {
        queuedActionDao.insert(
            QueuedActionEntity(
                id = action.id,
                endpoint = action.endpoint,
                method = action.method,
                bodyJson = action.body,
                timestamp = action.createdAt
            )
        )
    }

    /**
     * Drain the persisted offline queue. Returns (sent, kept) pair.
     */
    suspend fun flushQueue(baseUrl: String): Pair<Int, Int> {
        val current = queuedActionDao.getAll()
        if (current.isEmpty()) return 0 to 0
        val token = secureStore.token ?: return 0 to current.size
        
        var sent = 0
        var remaining = 0
        for (action in current) {
            val req = Request.Builder()
                .url("${baseUrl.trimEnd('/')}${action.endpoint}")
                .header("Authorization", "Bearer $token")
                .header("Content-Type", "application/json")
                .header("Idempotency-Key", action.id)
                .method(action.method, action.bodyJson?.toRequestBody("application/json".toMediaType()) ?: okhttp3.internal.EMPTY_REQUEST)
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
                QueueReplayOutcome.Sent, QueueReplayOutcome.Drop -> {
                    queuedActionDao.deleteById(action.id)
                    sent++
                }
                QueueReplayOutcome.Retry -> {
                    remaining++
                }
            }
        }
        return sent to remaining
    }

    private enum class QueueReplayOutcome { Sent, Retry, Drop }

    data class CatalogBarcodeLookup(
        val skuId: String,
        val name: String,
    )

    suspend fun lookupBarcode(ean: String): CatalogBarcodeLookup {
        val resp = api.lookupBarcode(ean)
        if (!resp.isSuccessful) {
            val msg = resp.errorBody()?.string()?.takeIf { it.isNotBlank() }
            throw IllegalStateException(msg ?: "Barcode lookup failed (${resp.code()})")
        }
        val body = resp.body() ?: throw IllegalStateException("Empty barcode lookup response")
        return CatalogBarcodeLookup(
            skuId = body["sku_id"]?.toString().orEmpty().ifBlank { body["product_id"]?.toString().orEmpty() },
            name = body["name"]?.toString().orEmpty(),
        )
    }
}
