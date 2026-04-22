package com.thelab.retailer.data.local

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class TokenManager @Inject constructor(@ApplicationContext context: Context) {

    private val prefs: SharedPreferences by lazy {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "lab_auth_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    fun saveToken(token: String) {
        prefs.edit().putString(KEY_JWT, token).apply()
    }

    fun getToken(): String? = prefs.getString(KEY_JWT, null)

    /** Firebase ID token — preferred over legacy JWT when non-null */
    fun saveFirebaseIdToken(token: String) {
        prefs.edit().putString(KEY_FIREBASE_ID_TOKEN, token).apply()
    }

    fun getFirebaseIdToken(): String? = prefs.getString(KEY_FIREBASE_ID_TOKEN, null)

    /** Returns Firebase ID token if available, otherwise legacy JWT */
    fun getPreferredToken(): String? = getFirebaseIdToken() ?: getToken()

    fun saveUserId(userId: String) {
        prefs.edit().putString(KEY_USER_ID, userId).apply()
    }

    fun getUserId(): String? = prefs.getString(KEY_USER_ID, null)

    fun saveUserName(name: String) {
        prefs.edit().putString(KEY_USER_NAME, name).apply()
    }

    fun getUserName(): String? = prefs.getString(KEY_USER_NAME, null)

    fun clearToken() {
        prefs.edit()
            .remove(KEY_JWT)
            .remove(KEY_FIREBASE_ID_TOKEN)
            .remove(KEY_USER_ID)
            .remove(KEY_USER_NAME)
            .apply()
    }

    companion object {
        private const val KEY_JWT = "jwt_token"
        private const val KEY_FIREBASE_ID_TOKEN = "firebase_id_token"
        private const val KEY_USER_ID = "user_id"
        private const val KEY_USER_NAME = "user_name"
    }
}
