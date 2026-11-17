package com.pegasusx.warehouse.offline

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import java.util.UUID

/**
 * EncryptedSharedPreferences-backed offline queue implementing the §8.8 mutation contract.
 * Encrypts queued offline mutations at rest via Android Keystore AES256-GCM.
 */
class PrefsOfflineQueueStore(
    context: Context,
    prefsName: String,
    private val catalog: OfflineEndpointCatalog? = null,
) {
    private val appContext = context.applicationContext
    private val prefs: SharedPreferences = openEncryptedPrefs(appContext, prefsName)
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    private fun openEncryptedPrefs(ctx: Context, baseName: String): SharedPreferences {
        val masterKey = MasterKey.Builder(ctx)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        val encrypted = EncryptedSharedPreferences.create(
            ctx,
            "${baseName}_encrypted",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
        // Migrate and purge any pre-hardening plaintext queue entries once.
        val legacy = ctx.getSharedPreferences(baseName, Context.MODE_PRIVATE)
        val legacyRaw = legacy.getString(KEY, null)
        if (!legacyRaw.isNullOrBlank()) {
            if (!encrypted.contains(KEY)) {
                encrypted.edit().putString(KEY, legacyRaw).apply()
            }
            legacy.edit().remove(KEY).apply()
        }
        return encrypted
    }

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
