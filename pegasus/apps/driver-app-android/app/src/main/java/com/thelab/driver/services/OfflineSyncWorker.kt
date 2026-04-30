package com.pegasus.driver.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasus.driver.data.local.PendingMutationDao
import com.pegasus.driver.data.remote.DriverApi
import com.pegasus.driver.data.model.DeliverySubmitRequest
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.serialization.json.Json

/**
 * Drains the pending_mutations Room table when network reconnects.
 * Enqueued with a NetworkType.CONNECTED constraint so it only fires online.
 *
 * For each mutation:
 *  - POST to the stored endpoint with the original payload + Idempotency-Key header
 *  - On 200/409 (success or idempotent duplicate) → delete from Room
 *  - On 5xx / network error → leave in queue for the next retry cycle
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

        var failures = 0
        for (mutation in pending) {
            try {
                when (mutation.endpoint) {
                    "v1/order/deliver" -> {
                        val req = json.decodeFromString<DeliverySubmitRequest>(mutation.payloadJson)
                        api.submitDelivery(req, idempotencyKey = mutation.id)
                    }
                    else -> {
                        Log.w(TAG, "Unknown endpoint: ${mutation.endpoint}, skipping")
                        continue
                    }
                }
                // Success — purge from queue
                pendingDao.deleteById(mutation.id)
                Log.d(TAG, "Synced mutation ${mutation.id} → ${mutation.endpoint}")
            } catch (e: retrofit2.HttpException) {
                if (e.code() == 409) {
                    // Idempotent duplicate — safe to discard
                    pendingDao.deleteById(mutation.id)
                    Log.d(TAG, "409 idempotent duplicate for ${mutation.id}, purged")
                } else if (e.code() in 500..599) {
                    // Server error — retry later
                    failures++
                    Log.w(TAG, "Server error ${e.code()} for ${mutation.id}, will retry")
                } else {
                    // 4xx (except 409) — discard to prevent infinite retries
                    pendingDao.deleteById(mutation.id)
                    Log.w(TAG, "Client error ${e.code()} for ${mutation.id}, discarded")
                }
            } catch (e: Exception) {
                // Network still down or unexpected failure — retry
                failures++
                Log.e(TAG, "Failed to sync ${mutation.id}: ${e.message}")
            }
        }

        return if (failures > 0) Result.retry() else Result.success()
    }
}
