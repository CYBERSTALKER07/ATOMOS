package com.pegasusx.retailer.data.local

import com.pegasusx.retailer.data.api.PegasusApi
import java.util.UUID
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.json.JSONArray
import org.json.JSONObject

@Singleton
class PendingPosSaleSync @Inject constructor(
    private val dao: PendingPosSaleDao,
    private val api: PegasusApi,
) {
    private val json = Json { ignoreUnknownKeys = true }

    suspend fun enqueue(
        sessionId: String,
        lines: List<Map<String, Any>>,
        totalMinor: Long,
    ): PendingPosSaleEntity {
        val clientSaleId = UUID.randomUUID().toString()
        val receipt = "OFF-${System.currentTimeMillis().toString(36).uppercase()}-${(1..999).random()}"
        val linesJson = JSONArray()
        for (line in lines) {
            linesJson.put(
                JSONObject()
                    .put("sku", line["sku"]?.toString() ?: "")
                    .put("name", line["name"]?.toString() ?: line["sku"]?.toString() ?: "")
                    .put("qty", (line["qty"] as? Number)?.toLong() ?: 1L)
                    .put("unit_price_minor", (line["unit_price_minor"] as? Number)?.toLong() ?: 0L),
            )
        }
        val tendersJson = JSONArray().put(
            JSONObject().put("method", "CASH").put("amount_minor", totalMinor),
        )
        val payload = JSONObject()
            .put("session_id", sessionId)
            .put("stock_bin", "FLOOR")
            .put("origin", "offline")
            .put("client_sale_id", clientSaleId)
            .put("client_created_at", java.time.Instant.now().toString())
            .put("lines", linesJson)
            .put("tenders", tendersJson)
        val entity = PendingPosSaleEntity(
            id = clientSaleId,
            clientSaleId = clientSaleId,
            clientReceipt = receipt,
            sessionId = sessionId,
            payloadJson = payload.toString(),
            idempotencyKey = "pos-sale:$clientSaleId",
            status = "PENDING",
        )
        dao.upsert(entity)
        return entity
    }

    suspend fun countActiveForSession(sessionId: String): Int =
        dao.countActiveForSession(sessionId)

    suspend fun listActive(): List<PendingPosSaleEntity> = dao.getActive()

    suspend fun flush(): Pair<Int, Int> {
        var flushed = 0
        var failed = 0
        for (entry in dao.getActive()) {
            dao.updateStatus(entry.id, "SYNCING", entry.retryCount, null, null, null)
            try {
                val body = jsonObjectToAnyMap(JSONObject(entry.payloadJson))
                val saleEl = api.createPosSale(body = body, idempotencyKey = entry.idempotencyKey)
                val sale = saleEl.jsonObject
                val saleId = sale["sale_id"]?.jsonPrimitive?.contentOrNull
                val receipt = sale["receipt_number"]?.jsonPrimitive?.contentOrNull
                dao.updateStatus(entry.id, "SYNCED", entry.retryCount, null, saleId, receipt)
                dao.delete(entry.id)
                flushed++
            } catch (e: Exception) {
                failed++
                dao.updateStatus(
                    entry.id,
                    "FAILED",
                    entry.retryCount + 1,
                    e.message,
                    null,
                    null,
                )
            }
        }
        return flushed to failed
    }

    /** Convert org.json tree to Map/List for Retrofit Map body. */
    private fun jsonObjectToAnyMap(obj: JSONObject): Map<String, @JvmSuppressWildcards Any> {
        val map = linkedMapOf<String, Any>()
        val keys = obj.keys()
        while (keys.hasNext()) {
            val key = keys.next()
            map[key] = jsonValueToAny(obj.get(key))
        }
        return map
    }

    private fun jsonValueToAny(value: Any?): Any {
        return when (value) {
            null, JSONObject.NULL -> ""
            is JSONObject -> jsonObjectToAnyMap(value)
            is JSONArray -> {
                val list = ArrayList<Any>(value.length())
                for (i in 0 until value.length()) {
                    list.add(jsonValueToAny(value.get(i)))
                }
                list
            }
            is Number, is Boolean, is String -> value
            else -> value.toString()
        }
    }
}
