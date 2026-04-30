package com.thelab.driver.data.remote

import android.content.Context
import android.util.Log
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.google.firebase.auth.FirebaseAuth
import kotlinx.coroutines.tasks.await

/**
 * Firebase Auth helper for dual-mode authentication.
 * Connects to Firebase Auth Emulator in debug builds.
 * All methods degrade gracefully — if Firebase is unavailable, legacy JWT still works.
 */
object FirebaseAuthHelper {
    private const val TAG = "FirebaseAuth"
    private var initialized = false

    /**
     * Initialize Firebase with programmatic config (no google-services.json needed).
     * Call once from Application.onCreate() or login flow.
     */
    fun init(context: Context) {
        if (initialized) return
        try {
            if (FirebaseApp.getApps(context).isEmpty()) {
                val options = FirebaseOptions.Builder()
                    .setProjectId("demo-thelab")
                    .setApplicationId("1:000000000000:android:0000000000000000")
                    .setApiKey("demo-key")
                    .build()
                FirebaseApp.initializeApp(context, options)
            }
            // Connect to emulator in debug builds
            if (com.pegasus.driver.BuildConfig.DEBUG) {
                val emulatorHost = "10.0.2.2" // Android emulator localhost
                FirebaseAuth.getInstance().useEmulator(emulatorHost, 9099)
            }
            initialized = true
            Log.d(TAG, "Firebase Auth initialized (debug=${com.pegasus.driver.BuildConfig.DEBUG})")
        } catch (e: Exception) {
            Log.w(TAG, "Firebase Auth init failed (non-fatal): ${e.message}")
        }
    }

    /**
     * Exchange a Firebase Custom Token from the backend for a Firebase session.
     * Returns the Firebase ID token string, or null on failure.
     */
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

    /**
     * Get a fresh Firebase ID token for the currently signed-in user.
     * Returns null if no Firebase session exists.
     */
    suspend fun getIdToken(): String? {
        return try {
            FirebaseAuth.getInstance().currentUser
                ?.getIdToken(false)
                ?.await()
                ?.token
        } catch (e: Exception) {
            null
        }
    }

    /** Sign out of Firebase Auth. */
    fun signOut() {
        try {
            FirebaseAuth.getInstance().signOut()
        } catch (_: Exception) { }
    }
}
