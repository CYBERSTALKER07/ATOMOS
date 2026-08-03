package com.pegasusx.driver.offline

import android.content.Context
import android.util.Log
import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.services.OfflineSyncScheduler
import dagger.hilt.android.qualifiers.ApplicationContext
import java.time.Instant
import java.util.UUID
import javax.inject.Inject
import javax.inject.Singleton
import org.json.JSONArray
import org.json.JSONObject

@Singleton
class DriverOfflineQueue @Inject constructor(
    @ApplicationContext private val appContext: Context,
    private val dao: PendingMutationDao,
) {
    companion object {
        private const val TAG = "DriverOfflineQueue"
    }

    suspend fun enqueue(
        endpoint: String,
        payloadJson: String,
        idempotencyKey: String,
        orderId: String = "",
        clientTimestampIso: String = Instant.now().toString(),
        method: String = "POST",
        scheduleFlush: Boolean = true,
    ): Boolean {
        val ep = DriverOfflineActionCatalog.normalize(endpoint)
        if (!DriverOfflineActionCatalog.isOfflineEligible(ep)) {
            Log.w(TAG, "Endpoint not offline-eligible: $ep")
            return false
        }
        val entity = PendingMutationEntity(
            id = idempotencyKey.ifBlank { UUID.randomUUID().toString() },
            endpoint = ep,
            payloadJson = payloadJson,
            idempotencyKey = idempotencyKey,
            method = method,
            priority = DriverOfflineActionCatalog.priorityFor(ep),
            clientTimestampIso = clientTimestampIso.ifBlank { Instant.now().toString() },
            orderId = orderId.ifBlank { extractOrderId(payloadJson) },
            status = DriverOfflineActionCatalog.STATUS_PENDING,
        )
        dao.insert(entity)
        Log.i(TAG, "Enqueued ${entity.endpoint} order=${entity.orderId} key=${entity.idempotencyKey}")
        if (scheduleFlush) {
            OfflineSyncScheduler.enqueue(appContext)
        }
        return true
    }

    suspend fun enqueueMap(
        endpoint: String,
        body: Map<String, Any?>,
        idempotencyKey: String,
        orderId: String = "",
        clientTimestampIso: String = Instant.now().toString(),
    ): Boolean {
        val obj = JSONObject()
        for ((k, v) in body) {
            if (v == null) continue
            when (v) {
                is Map<*, *> -> obj.put(k, JSONObject(v))
                is List<*> -> obj.put(k, JSONArray(v))
                else -> obj.put(k, v)
            }
        }
        if (!obj.has("client_timestamp") && clientTimestampIso.isNotBlank()) {
            obj.put("client_timestamp", clientTimestampIso)
        }
        return enqueue(
            endpoint = endpoint,
            payloadJson = obj.toString(),
            idempotencyKey = idempotencyKey,
            orderId = orderId.ifBlank { obj.optString("order_id", "") },
            clientTimestampIso = clientTimestampIso,
        )
    }

    fun nowIso(): String = Instant.now().toString()

    private fun extractOrderId(payloadJson: String): String = runCatching {
        JSONObject(payloadJson).optString("order_id", "")
    }.getOrDefault("")
}
