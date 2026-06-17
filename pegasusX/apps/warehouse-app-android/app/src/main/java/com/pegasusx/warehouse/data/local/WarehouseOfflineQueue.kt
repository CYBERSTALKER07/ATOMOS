package com.pegasusx.warehouse.data.local

import android.content.Context
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Serializable
data class WarehouseQueuedAction(
    val id: String,
    val method: String,
    val endpoint: String,
    val body: String,
)

object WarehouseOfflineQueue {
    private const val PREFS = "warehouse_offline_queue"
    private const val KEY = "queue_json"
    private val json = Json { ignoreUnknownKeys = true }

    fun read(context: Context): List<WarehouseQueuedAction> {
        val raw = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getString(KEY, null)
            ?: return emptyList()
        return runCatching { json.decodeFromString<List<WarehouseQueuedAction>>(raw) }.getOrDefault(emptyList())
    }

    fun write(context: Context, items: List<WarehouseQueuedAction>) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY, if (items.isEmpty()) null else json.encodeToString(items))
            .apply()
    }

    fun enqueue(context: Context, action: WarehouseQueuedAction) {
        write(context, read(context) + action)
    }
}
