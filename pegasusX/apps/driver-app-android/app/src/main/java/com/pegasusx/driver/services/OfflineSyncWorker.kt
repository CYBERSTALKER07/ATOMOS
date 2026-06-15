package com.pegasusx.driver.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.DeliverySubmitRequest
import com.pegasusx.driver.data.model.OfflineDeliveryPayload
import com.pegasusx.driver.data.model.SyncBatchDelivery
import com.pegasusx.driver.data.model.SyncBatchRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.util.DriverIdempotencyKeys
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.serialization.json.Json

/**
 * Drains the pending_mutations Room table when network reconnects.
 * Offline verifier deliveries prefer POST /v1/sync/batch; legacy rows fall back to direct deliver.
 */
@HiltWorker
class OfflineSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: DriverApi,
    private val pendingDao: PendingMutationDao,
    private val json: Json
) : CoroutineWorker(appContext, params) {

    companion object {
        const val TAG = "OfflineSyncWorker"
        const val WORK_NAME = "offline_sync"
    }

    override suspend fun doWork(): Result {
        val pending = pendingDao.getAll()
        if (pending.isEmpty()) return Result.success()

        Log.d(TAG, "Draining ${pending.size} queued mutation(s)")

        val deliverMutations = pending.filter { it.endpoint == "v1/order/deliver" }
        val otherMutations = pending.filter { it.endpoint != "v1/order/deliver" }

        var failures = 0

        if (deliverMutations.isNotEmpty()) {
            val batchFailures = syncDeliveries(deliverMutations)
            failures += batchFailures
        }

        for (mutation in otherMutations) {
            Log.w(TAG, "Unknown endpoint: ${mutation.endpoint}, skipping")
            pendingDao.deleteById(mutation.id)
        }

        return if (failures > 0) Result.retry() else Result.success()
    }

    private suspend fun syncDeliveries(mutations: List<com.pegasusx.driver.data.model.PendingMutationEntity>): Int {
        val driverId = TokenHolder.userId?.takeIf { it.isNotBlank() } ?: return mutations.size

        val parsed = mutations.mapNotNull { mutation ->
            val offline = runCatching {
                json.decodeFromString<OfflineDeliveryPayload>(mutation.payloadJson)
            }.getOrNull()
            if (offline != null) {
                mutation to offline
            } else {
                val legacy = runCatching {
                    json.decodeFromString<DeliverySubmitRequest>(mutation.payloadJson)
                }.getOrNull() ?: return@mapNotNull null
                mutation to OfflineDeliveryPayload(
                    orderId = legacy.orderId,
                    scannedToken = legacy.scannedToken,
                    signature = legacy.scannedToken,
                )
            }
        }

        if (parsed.isEmpty()) return mutations.size

        val batchDeliveries = parsed.map { (_, payload) ->
            SyncBatchDelivery(
                orderId = payload.orderId,
                signature = payload.signature,
                timestamp = System.currentTimeMillis() / 1000.0,
                status = "DELIVERED",
            )
        }

        return try {
            val result = api.syncBatch(
                SyncBatchRequest(
                    driverId = driverId,
                    deliveries = batchDeliveries,
                ),
            )
            val processed = result.processed.toSet()
            for ((mutation, payload) in parsed) {
                if (payload.orderId in processed) {
                    pendingDao.deleteById(mutation.id)
                    Log.d(TAG, "Batch synced ${payload.orderId}")
                }
            }
            val remaining = parsed.count { (mutation, payload) ->
                payload.orderId !in processed && pendingDao.getAll().any { it.id == mutation.id }
            }
            if (remaining > 0) flushDirectDeliveries(parsed.filter { it.first.id !in processed.map { m -> m } })
            remaining
        } catch (e: retrofit2.HttpException) {
            if (e.code() in 500..599) mutations.size else flushDirectDeliveries(parsed)
        } catch (e: Exception) {
            Log.e(TAG, "Batch sync failed: ${e.message}")
            flushDirectDeliveries(parsed)
        }
    }

    private suspend fun flushDirectDeliveries(
        parsed: List<Pair<com.pegasusx.driver.data.model.PendingMutationEntity, OfflineDeliveryPayload>>,
    ): Int {
        var failures = 0
        for ((mutation, payload) in parsed) {
            try {
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
            } catch (e: retrofit2.HttpException) {
                when (e.code()) {
                    409 -> pendingDao.deleteById(mutation.id)
                    in 500..599 -> failures++
                    else -> pendingDao.deleteById(mutation.id)
                }
            } catch (e: Exception) {
                failures++
            }
        }
        return failures
    }
}
