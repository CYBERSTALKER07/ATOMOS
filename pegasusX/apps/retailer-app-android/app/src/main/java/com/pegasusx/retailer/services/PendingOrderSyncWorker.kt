package com.pegasusx.retailer.services

import android.content.Context
import android.util.Log
import androidx.hilt.work.HiltWorker
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.PendingOrderDao
import com.pegasusx.retailer.data.model.ProcurementOrderRequest
import com.pegasusx.retailer.data.model.UnifiedCheckoutRequest
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import kotlinx.serialization.json.Json
import retrofit2.HttpException

/**
 * Drains pending_orders when network reconnects (checkout + procurement).
 * Idempotency keys prevent double-apply if the original request succeeded during the outage.
 */
@HiltWorker
class PendingOrderSyncWorker @AssistedInject constructor(
    @Assisted appContext: Context,
    @Assisted params: WorkerParameters,
    private val api: PegasusApi,
    private val pendingOrderDao: PendingOrderDao,
    private val json: Json,
) : CoroutineWorker(appContext, params) {

    companion object {
        const val TAG = "PendingOrderSyncWorker"
        const val WORK_NAME = "retailer_pending_order_sync"
    }

    override suspend fun doWork(): Result {
        val pending = pendingOrderDao.getAll()
        if (pending.isEmpty()) return Result.success()

        Log.d(TAG, "Draining ${pending.size} queued order(s)")

        var failures = 0
        for (order in pending) {
            try {
                when (order.endpoint) {
                    "/v1/checkout/unified" -> {
                        val request = json.decodeFromString<UnifiedCheckoutRequest>(order.payloadJson)
                        api.unifiedCheckout(request, order.idempotencyKey)
                    }
                    "/v1/order/create" -> {
                        val request = json.decodeFromString<ProcurementOrderRequest>(order.payloadJson)
                        api.createOrder(request, order.idempotencyKey)
                    }
                    else -> {
                        Log.w(TAG, "Unknown endpoint: ${order.endpoint}, skipping")
                        continue
                    }
                }
                pendingOrderDao.deleteById(order.id)
                Log.d(TAG, "Synced pending order ${order.id} → ${order.endpoint}")
            } catch (e: HttpException) {
                when (e.code()) {
                    409 -> {
                        pendingOrderDao.deleteById(order.id)
                        Log.d(TAG, "409 idempotent duplicate for ${order.id}, purged")
                    }
                    in 500..599 -> {
                        failures++
                        pendingOrderDao.incrementRetry(order.id, "Server error ${e.code()}")
                    }
                    else -> {
                        pendingOrderDao.deleteById(order.id)
                        Log.w(TAG, "Client error ${e.code()} for ${order.id}, discarded")
                    }
                }
            } catch (e: Exception) {
                failures++
                pendingOrderDao.incrementRetry(order.id, e.message ?: "sync failed")
                Log.e(TAG, "Failed to sync ${order.id}: ${e.message}")
            }
        }

        return if (failures > 0) Result.retry() else Result.success()
    }
}
