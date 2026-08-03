package com.pegasusx.driver.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasusx.driver.BuildConfig
import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.DeliverySubmitRequest
import com.pegasusx.driver.data.model.OfflineDeliveryPayload
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.data.model.SyncBatchDelivery
import com.pegasusx.driver.data.model.SyncBatchRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.offline.DriverOfflineActionCatalog
import com.pegasusx.driver.util.DriverIdempotencyKeys
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.time.Instant
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject

/**
 * Drains pending_mutations when network reconnects.
 * Protocol order (4.1): proximity → shop-closed/partial/deliver → settlement.
 */
@HiltWorker
class OfflineSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: DriverApi,
    private val pendingDao: PendingMutationDao,
    private val json: Json,
    private val httpClient: OkHttpClient,
) : CoroutineWorker(appContext, params) {

    companion object {
        const val TAG = "OfflineSyncWorker"
        const val WORK_NAME = "offline_sync"
        private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()
    }

    override suspend fun doWork(): Result {
        val pending = pendingDao.getPending()
        if (pending.isEmpty()) return Result.success()

        Log.d(TAG, "Draining ${pending.size} queued mutation(s)")

        val ordered = pending.sortedWith(
            compareBy<PendingMutationEntity> { it.priority }.thenBy { it.createdAt },
        )

        val delivers = ordered.filter {
            DriverOfflineActionCatalog.normalize(it.endpoint) == DriverOfflineActionCatalog.ENDPOINT_DELIVER
        }
        if (delivers.isNotEmpty()) {
            trySyncBatch(delivers)
        }

        var retryableFailures = 0
        val remaining = pendingDao.getPending().sortedWith(
            compareBy<PendingMutationEntity> { it.priority }.thenBy { it.createdAt },
        )
        for (mutation in remaining) {
            when (flushOne(mutation)) {
                FlushOutcome.RETRY -> retryableFailures++
                FlushOutcome.SKIP_KEEP -> retryableFailures++ // proximity too stale — retry later
                FlushOutcome.ACK, FlushOutcome.DEAD -> Unit
            }
        }

        return if (retryableFailures > 0) Result.retry() else Result.success()
    }

    private suspend fun flushOne(mutation: PendingMutationEntity): FlushOutcome {
        val ep = DriverOfflineActionCatalog.normalize(mutation.endpoint)

        if (ep == DriverOfflineActionCatalog.ENDPOINT_PROXIMITY) {
            val ageMs = ageMillis(mutation)
            if (ageMs > DriverOfflineActionCatalog.PROXIMITY_MAX_AGE_MS) {
                pendingDao.recordAttempt(
                    mutation.id,
                    "proximity_timestamp_stale ageMs=$ageMs",
                )
                Log.w(TAG, "Skip proximity ${mutation.orderId}: GPS/timestamp too stale")
                return FlushOutcome.SKIP_KEEP
            }
        }

        return when (ep) {
            DriverOfflineActionCatalog.ENDPOINT_DELIVER -> flushDeliverDirect(mutation)
            else -> flushGeneric(mutation)
        }
    }

    private suspend fun trySyncBatch(mutations: List<PendingMutationEntity>) {
        val driverId = TokenHolder.userId?.takeIf { it.isNotBlank() } ?: return
        val parsed = mutations.mapNotNull { mutation -> parsePayload(mutation)?.let { mutation to it } }
        if (parsed.isEmpty()) return

        runCatching {
            val deliveries = parsed.map { (mutation, payload) ->
                SyncBatchDelivery(
                    orderId = payload.orderId,
                    signature = payload.signature,
                    timestamp = storedEpochSeconds(mutation),
                    status = "DELIVERED",
                )
            }
            val batchKey = DriverIdempotencyKeys.syncBatch(
                deliveries.map { "${it.orderId}:${it.signature}" },
            )
            val result = api.syncBatch(
                SyncBatchRequest(
                    driverId = driverId,
                    deliveries = deliveries,
                ),
                idempotencyKey = batchKey,
            )
            val processed = result.processed.toSet()
            for ((mutation, payload) in parsed) {
                if (payload.orderId in processed) {
                    pendingDao.deleteById(mutation.id)
                    Log.d(TAG, "Batch synced ${payload.orderId}")
                }
            }
        }.onFailure { err ->
            Log.w(TAG, "Batch sync failed, will retry direct deliver: ${err.message}")
        }
    }

    private suspend fun flushDeliverDirect(mutation: PendingMutationEntity): FlushOutcome {
        val payload = parsePayload(mutation) ?: run {
            pendingDao.markDead(mutation.id, "invalid_deliver_payload")
            return FlushOutcome.DEAD
        }
        return try {
            api.submitDelivery(
                DeliverySubmitRequest(
                    orderId = payload.orderId,
                    scannedToken = payload.scannedToken,
                ),
                idempotencyKey = mutation.idempotencyKey.ifBlank {
                    DriverIdempotencyKeys.deliver(payload.orderId)
                },
            )
            pendingDao.deleteById(mutation.id)
            Log.d(TAG, "Direct synced ${payload.orderId}")
            FlushOutcome.ACK
        } catch (e: retrofit2.HttpException) {
            handleHttp(mutation, e.code(), e.message())
        } catch (e: Exception) {
            pendingDao.recordAttempt(mutation.id, e.message ?: "network_error")
            maybeDead(mutation, e.message ?: "network_error")
        }
    }

    private suspend fun flushGeneric(mutation: PendingMutationEntity): FlushOutcome {
        val token = TokenHolder.token?.takeIf { it.isNotBlank() } ?: run {
            pendingDao.recordAttempt(mutation.id, "missing_auth_token")
            return FlushOutcome.RETRY
        }
        val url = BuildConfig.API_BASE_URL.trimEnd('/') + "/" +
            DriverOfflineActionCatalog.normalize(mutation.endpoint)
        val body = mutation.payloadJson.toRequestBody(JSON_MEDIA)
        val request = Request.Builder()
            .url(url)
            .method(mutation.method.ifBlank { "POST" }, body)
            .header("Authorization", "Bearer $token")
            .header("Idempotency-Key", mutation.idempotencyKey)
            .header("Content-Type", "application/json")
            .build()

        return try {
            httpClient.newCall(request).execute().use { resp ->
                handleHttp(mutation, resp.code, resp.body?.string().orEmpty())
            }
        } catch (e: Exception) {
            pendingDao.recordAttempt(mutation.id, e.message ?: "network_error")
            maybeDead(mutation, e.message ?: "network_error")
        }
    }

    private suspend fun handleHttp(mutation: PendingMutationEntity, code: Int, detail: String): FlushOutcome {
        return when {
            DriverOfflineActionCatalog.isSuccessHttp(code) -> {
                pendingDao.deleteById(mutation.id)
                Log.d(TAG, "Ack ${mutation.endpoint} ${mutation.orderId} code=$code")
                FlushOutcome.ACK
            }
            DriverOfflineActionCatalog.isRetryableHttp(code) -> {
                pendingDao.recordAttempt(mutation.id, "http_$code $detail")
                maybeDead(mutation, "http_$code")
            }
            else -> {
                pendingDao.markDead(mutation.id, "http_$code $detail")
                Log.w(TAG, "Dead-letter ${mutation.endpoint} ${mutation.orderId} code=$code")
                FlushOutcome.DEAD
            }
        }
    }

    private suspend fun maybeDead(mutation: PendingMutationEntity, error: String): FlushOutcome {
        val nextAttempts = mutation.attemptCount + 1
        return if (nextAttempts >= DriverOfflineActionCatalog.MAX_ATTEMPTS) {
            pendingDao.markDead(mutation.id, error)
            FlushOutcome.DEAD
        } else {
            FlushOutcome.RETRY
        }
    }

    private fun storedEpochSeconds(mutation: PendingMutationEntity): Double {
        val fromIso = runCatching {
            Instant.parse(mutation.clientTimestampIso).epochSecond.toDouble()
        }.getOrNull()
        if (fromIso != null) return fromIso
        val fromPayload = runCatching {
            val obj = JSONObject(mutation.payloadJson)
            when {
                obj.has("timestamp") -> obj.getDouble("timestamp").let { if (it > 1e12) it / 1000.0 else it }
                obj.has("client_timestamp") -> Instant.parse(obj.getString("client_timestamp")).epochSecond.toDouble()
                else -> null
            }
        }.getOrNull()
        if (fromPayload != null) return fromPayload
        return mutation.createdAt / 1000.0
    }

    private fun ageMillis(mutation: PendingMutationEntity): Long {
        val ts = runCatching { Instant.parse(mutation.clientTimestampIso).toEpochMilli() }.getOrNull()
            ?: mutation.createdAt
        return System.currentTimeMillis() - ts
    }

    private fun parsePayload(mutation: PendingMutationEntity): OfflineDeliveryPayload? {
        val offline = runCatching {
            json.decodeFromString<OfflineDeliveryPayload>(mutation.payloadJson)
        }.getOrNull()
        if (offline != null) return offline

        val legacy = runCatching {
            json.decodeFromString<DeliverySubmitRequest>(mutation.payloadJson)
        }.getOrNull() ?: return null
        return OfflineDeliveryPayload(
            orderId = legacy.orderId,
            scannedToken = legacy.scannedToken.orEmpty(),
            signature = legacy.scannedToken.orEmpty(),
        )
    }

    private enum class FlushOutcome { ACK, RETRY, DEAD, SKIP_KEEP }
}
