package com.pegasus.payload.data.auth

import android.content.Context
import android.util.Log
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.google.firebase.auth.FirebaseAuth
import com.pegasus.payload.BuildConfig
import kotlinx.coroutines.tasks.await

/**
 * Firebase Auth helper for dual-mode authentication.
 * Connects to Firebase Auth Emulator in debug builds.
 * All methods degrade gracefully — legacy JWT still works when Firebase is unavailable.
 */
object FirebaseAuthHelper {
    private const val TAG = "FirebaseAuth"
    private var initialized = false

    fun init(context: Context) {
        if (initialized) return
        try {
            if (FirebaseApp.getApps(context).isEmpty()) {
                val options = FirebaseOptions.Builder()
                    .setProjectId("demo-pegasus")
                    .setApplicationId("1:000000000000:android:0000000000000001")
                    .setApiKey("demo-key")
                    .build()
                FirebaseApp.initializeApp(context, options)
            }
            if (BuildConfig.DEBUG) {
                FirebaseAuth.getInstance().useEmulator("10.0.2.2", 9099)
            }
            initialized = true
            Log.d(TAG, "Firebase Auth initialized (debug=${BuildConfig.DEBUG})")
        } catch (e: Exception) {
            Log.w(TAG, "Firebase Auth init failed (non-fatal): ${e.message}")
        }
    }

    suspend fun exchangeCustomToken(customToken: String): String? {
        if (customToken.isBlank()) return null
        return try {
            val result = FirebaseAuth.getInstance().signInWithCustomToken(customToken).await()
            result.user?.getIdToken(false)?.await()?.token
        } catch (e: Exception) {
            Log.w(TAG, "Custom token exchange failed (non-fatal): ${e.message}")
            null
        }
    }

    fun signOut() {
        try {
            FirebaseAuth.getInstance().signOut()
        } catch (_: Exception) {
        }
    }
}
