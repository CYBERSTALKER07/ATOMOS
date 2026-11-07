package com.pegasusx.mobilekit.offline

import android.content.Context
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import java.util.UUID

/**
 * SharedPreferences-backed offline queue implementing the §8.8 mutation contract.
 * Suitable for warehouse/factory until they adopt Room; preserves lat/lng/attempts.
 */
class PrefsOfflineQueueStore(
    context: Context,
    prefsName: String,
    private val catalog: OfflineEndpointCatalog? = null,
) {
    private val prefs = context.applicationContext.getSharedPreferences(prefsName, Context.MODE_PRIVATE)
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    fun readPending(): List<QueuedMutation> =
        readAll().filter { it.status == OfflineHttpSemantics.STATUS_PENDING }

    fun readAll(): List<QueuedMutation> {
        val raw = prefs.getString(KEY, null) ?: return emptyList()
        return runCatching {
            json.decodeFromString(ListSerializer(QueuedMutation.serializer()), raw)
        }.getOrDefault(emptyList())
    }

    fun writeAll(items: List<QueuedMutation>) {
        prefs.edit()
            .putString(
                KEY,
                if (items.isEmpty()) null
                else json.encodeToString(ListSerializer(QueuedMutation.serializer()), items),
            )
            .apply()
    }

    fun enqueue(mutation: QueuedMutation): Boolean {
        val ep = OfflineHttpSemantics.normalizeEndpoint(mutation.endpoint)
        if (catalog != null && !catalog.isOfflineEligible(ep)) {
            return false
        }
        val row = mutation.copy(
            endpoint = ep,
            id = mutation.id.ifBlank { mutation.idempotencyKey.ifBlank { UUID.randomUUID().toString() } },
            priority = if (mutation.priority != 40 || catalog == null) {
                mutation.priority
            } else {
                catalog.priorityFor(ep)
            },
            status = OfflineHttpSemantics.STATUS_PENDING,
        )
        writeAll(readAll() + row)
        return true
    }

    fun ack(id: String) {
        writeAll(readAll().filterNot { it.id == id })
    }

    fun markDead(id: String, error: String) {
        writeAll(
            readAll().map {
                if (it.id == id) {
                    it.copy(
                        status = OfflineHttpSemantics.STATUS_DEAD,
                        lastError = error,
                        attemptCount = it.attemptCount + 1,
                    )
                } else {
                    it
                }
            },
        )
    }

    fun bumpAttempt(id: String, error: String) {
        writeAll(
            readAll().map {
                if (it.id == id) {
                    it.copy(attemptCount = it.attemptCount + 1, lastError = error)
                } else {
                    it
                }
            },
        )
    }

    companion object {
        private const val KEY = "queued_mutations_v1"
    }
}
