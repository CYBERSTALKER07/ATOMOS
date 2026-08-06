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
        capturedLat: Double? = null,
        capturedLng: Double? = null,
        capturedAtMs: Long? = null,
    ): Boolean {
        val ep = DriverOfflineActionCatalog.normalize(endpoint)
        if (!DriverOfflineActionCatalog.isOfflineEligible(ep)) {
            Log.w(TAG, "Endpoint not offline-eligible: $ep")
            return false
        }
        val fromPayload = extractCoords(payloadJson)
        val lat = capturedLat ?: fromPayload.first
        val lng = capturedLng ?: fromPayload.second
        val at = capturedAtMs ?: if (lat != null && lng != null) System.currentTimeMillis() else null
        val entity = PendingMutationEntity(
            id = idempotencyKey.ifBlank { UUID.randomUUID().toString() },
            endpoint = ep,
            payloadJson = ensureCoordsInPayload(payloadJson, lat, lng),
            idempotencyKey = idempotencyKey,
            method = method,
            priority = DriverOfflineActionCatalog.priorityFor(ep),
            clientTimestampIso = clientTimestampIso.ifBlank { Instant.now().toString() },
            orderId = orderId.ifBlank { extractOrderId(payloadJson) },
            status = DriverOfflineActionCatalog.STATUS_PENDING,
            capturedLat = lat,
            capturedLng = lng,
            capturedAtMs = at,
        )
        dao.insert(entity)
        Log.i(
            TAG,
            "Enqueued ${entity.endpoint} order=${entity.orderId} key=${entity.idempotencyKey} lat=$lat lng=$lng",
        )
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
        capturedLat: Double? = null,
        capturedLng: Double? = null,
        capturedAtMs: Long? = null,
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
        val lat = capturedLat ?: numberFrom(obj, "latitude")
        val lng = capturedLng ?: numberFrom(obj, "longitude")
        return enqueue(
            endpoint = endpoint,
            payloadJson = obj.toString(),
            idempotencyKey = idempotencyKey,
            orderId = orderId.ifBlank { obj.optString("order_id", "") },
            clientTimestampIso = clientTimestampIso,
            capturedLat = lat,
            capturedLng = lng,
            capturedAtMs = capturedAtMs,
        )
    }

    fun nowIso(): String = Instant.now().toString()

    private fun extractOrderId(payloadJson: String): String = runCatching {
        JSONObject(payloadJson).optString("order_id", "")
    }.getOrDefault("")

    private fun extractCoords(payloadJson: String): Pair<Double?, Double?> = runCatching {
        val obj = JSONObject(payloadJson)
        numberFrom(obj, "latitude") to numberFrom(obj, "longitude")
    }.getOrDefault(null to null)

    private fun numberFrom(obj: JSONObject, key: String): Double? {
        if (!obj.has(key) || obj.isNull(key)) return null
        return when (val v = obj.get(key)) {
            is Number -> v.toDouble()
            is String -> v.toDoubleOrNull()
            else -> null
        }
    }

    private fun ensureCoordsInPayload(payloadJson: String, lat: Double?, lng: Double?): String {
        if (lat == null || lng == null) return payloadJson
        return runCatching {
            val obj = JSONObject(payloadJson)
            if (!obj.has("latitude") || obj.isNull("latitude")) obj.put("latitude", lat)
            if (!obj.has("longitude") || obj.isNull("longitude")) obj.put("longitude", lng)
            obj.toString()
        }.getOrDefault(payloadJson)
    }
}
