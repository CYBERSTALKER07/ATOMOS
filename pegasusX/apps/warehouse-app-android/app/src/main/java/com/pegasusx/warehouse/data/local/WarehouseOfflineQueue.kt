package com.pegasusx.warehouse.data.local

import android.content.Context
import com.pegasusx.mobilekit.offline.PrefsOfflineQueueStore
import com.pegasusx.mobilekit.offline.QueuedMutation
import java.util.UUID

/** @deprecated Prefer [QueuedMutation]; kept as alias for call sites. */
typealias WarehouseQueuedAction = QueuedMutation

/**
 * Warehouse offline queue on shared kit contract (§8.8).
 * Migrates legacy prefs `{id,method,endpoint,body}` on first read.
 */
object WarehouseOfflineQueue {
    private const val PREFS = "warehouse_offline_queue_v2"
    private const val LEGACY_PREFS = "warehouse_offline_queue"
    private const val LEGACY_KEY = "queue_json"

    private fun store(context: Context) = PrefsOfflineQueueStore(context, PREFS)

    fun read(context: Context): List<QueuedMutation> {
        migrateLegacyIfNeeded(context)
        return store(context).readAll()
    }

    fun write(context: Context, items: List<QueuedMutation>) {
        store(context).writeAll(items)
    }

    fun enqueue(context: Context, action: QueuedMutation) {
        store(context).enqueue(action)
    }

    fun enqueue(
        context: Context,
        method: String,
        endpoint: String,
        body: String,
        idempotencyKey: String = UUID.randomUUID().toString(),
        capturedLat: Double? = null,
        capturedLng: Double? = null,
    ) {
        enqueue(
            context,
            QueuedMutation(
                id = idempotencyKey,
                endpoint = endpoint,
                method = method,
                payloadJson = body,
                idempotencyKey = idempotencyKey,
                capturedLat = capturedLat,
                capturedLng = capturedLng,
                capturedAtMs = if (capturedLat != null && capturedLng != null) {
                    System.currentTimeMillis()
                } else {
                    null
                },
            ),
        )
    }

    private fun migrateLegacyIfNeeded(context: Context) {
        val legacy = context.getSharedPreferences(LEGACY_PREFS, Context.MODE_PRIVATE)
        val raw = legacy.getString(LEGACY_KEY, null) ?: return
        if (store(context).readAll().isNotEmpty()) {
            legacy.edit().remove(LEGACY_KEY).apply()
            return
        }
        // Best-effort: treat legacy JSON array of {id,method,endpoint,body}
        runCatching {
            val arr = org.json.JSONArray(raw)
            val migrated = mutableListOf<QueuedMutation>()
            for (i in 0 until arr.length()) {
                val o = arr.getJSONObject(i)
                val id = o.optString("id").ifBlank { UUID.randomUUID().toString() }
                migrated += QueuedMutation(
                    id = id,
                    endpoint = o.optString("endpoint"),
                    method = o.optString("method", "POST"),
                    payloadJson = o.optString("body"),
                    idempotencyKey = id,
                )
            }
            store(context).writeAll(migrated)
            legacy.edit().remove(LEGACY_KEY).apply()
        }
    }
}
