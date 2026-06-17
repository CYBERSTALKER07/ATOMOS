package com.pegasusx.driver.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.DeliverySubmitRequest
import com.pegasusx.driver.data.model.OfflineDeliveryPayload
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.data.model.SyncBatchDelivery
import com.pegasusx.driver.data.model.SyncBatchRequest
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.util.DriverIdempotencyKeys
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.serialization.json.Json

/**
 * Drains pending_mutations when network reconnects.
 * Offline verifier deliveries use POST /v1/sync/batch first; unresolved rows fall back to direct deliver.
 */
@HiltWorker
class OfflineSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: DriverApi,
    private val pendingDao: PendingMutationDao,
    private val json: Json,
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
        val unknown = pending.filter { it.endpoint != "v1/order/deliver" }
        unknown.forEach { mutation ->
            Log.w(TAG, "Unknown endpoint: ${mutation.endpoint}, discarding")
            pendingDao.deleteById(mutation.id)
        }

        if (deliverMutations.isEmpty()) return Result.success()

        trySyncBatch(deliverMutations)
        val remaining = pendingDao.getAll().filter { it.endpoint == "v1/order/deliver" }
        val failures = flushDirectDeliveries(remaining)
        return if (failures > 0) Result.retry() else Result.success()
    }

    private suspend fun trySyncBatch(mutations: List<PendingMutationEntity>) {
        val driverId = TokenHolder.userId?.takeIf { it.isNotBlank() } ?: return
        val parsed = mutations.mapNotNull { mutation -> parsePayload(mutation)?.let { mutation to it } }
        if (parsed.isEmpty()) return

        runCatching {
            val deliveries = parsed.map { (_, payload) ->
                SyncBatchDelivery(
                    orderId = payload.orderId,
                    signature = payload.signature,
                    timestamp = System.currentTimeMillis() / 1000.0,
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

    private suspend fun flushDirectDeliveries(mutations: List<PendingMutationEntity>): Int {
        var failures = 0
        for (mutation in mutations) {
            val payload = parsePayload(mutation) ?: run {
                pendingDao.deleteById(mutation.id)
                continue
            }
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
                Log.d(TAG, "Direct synced ${payload.orderId}")
            } catch (e: retrofit2.HttpException) {
                when (e.code()) {
                    409 -> {
                        pendingDao.deleteById(mutation.id)
                        Log.d(TAG, "409 idempotent duplicate for ${mutation.id}, purged")
                    }
                    in 500..599 -> {
                        failures++
                        Log.w(TAG, "Server error ${e.code()} for ${mutation.id}, will retry")
                    }
                    else -> {
                        pendingDao.deleteById(mutation.id)
                        Log.w(TAG, "Client error ${e.code()} for ${mutation.id}, discarded")
                    }
                }
            } catch (e: Exception) {
                failures++
                Log.e(TAG, "Failed to sync ${mutation.id}: ${e.message}")
            }
        }
        return failures
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
}
