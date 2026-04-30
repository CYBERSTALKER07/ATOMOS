package com.thelab.payload.data.local

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Encrypted secure-store wrapper around AES256_GCM EncryptedSharedPreferences.
 * Mirrors the Keychain wrapper used on the iOS sibling app.
 *
 * Stores only short-lived auth/session tuples (token, worker_id, supplier_id,
 * warehouse_id/name, firebase_token). Larger artefacts (manifest cache,
 * offline action queue) live in Room.
 */
@Singleton
class SecureStore @Inject constructor(@ApplicationContext context: Context) {

    private val prefs = EncryptedSharedPreferences.create(
        context,
        "payload_secure_prefs",
        MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    var token: String?
        get() = prefs.getString(K_TOKEN, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_TOKEN) else putString(K_TOKEN, value); apply() }

    var name: String?
        get() = prefs.getString(K_NAME, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_NAME) else putString(K_NAME, value); apply() }

    var supplierId: String?
        get() = prefs.getString(K_SUPPLIER, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_SUPPLIER) else putString(K_SUPPLIER, value); apply() }

    var warehouseId: String?
        get() = prefs.getString(K_WH_ID, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_WH_ID) else putString(K_WH_ID, value); apply() }

    var warehouseName: String?
        get() = prefs.getString(K_WH_NAME, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_WH_NAME) else putString(K_WH_NAME, value); apply() }

    var firebaseToken: String?
        get() = prefs.getString(K_FB_TOKEN, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_FB_TOKEN) else putString(K_FB_TOKEN, value); apply() }

    /** JSON-serialized List<QueuedAction>. Mirrors Expo's `offline_queue` SecureStore key. */
    var offlineQueueJson: String?
        get() = prefs.getString(K_OFFLINE_QUEUE, null)
        set(value) = prefs.edit().run { if (value == null) remove(K_OFFLINE_QUEUE) else putString(K_OFFLINE_QUEUE, value); apply() }

    fun clear() = prefs.edit().clear().apply()

    private companion object {
        const val K_TOKEN = "payloader_token"
        const val K_NAME = "payloader_name"
        const val K_SUPPLIER = "payloader_supplier_id"
        const val K_WH_ID = "payloader_warehouse_id"
        const val K_WH_NAME = "payloader_warehouse_name"
        const val K_FB_TOKEN = "payloader_firebase_token"
        const val K_OFFLINE_QUEUE = "payloader_offline_queue"
    }
}
