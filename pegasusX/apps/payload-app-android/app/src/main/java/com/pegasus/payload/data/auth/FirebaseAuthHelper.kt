package com.pegasus.payload.data.auth

import android.app.Activity
import android.content.Context
import android.util.Log
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseException
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.auth.PhoneAuthCredential
import com.google.firebase.auth.PhoneAuthOptions
import com.google.firebase.auth.PhoneAuthProvider
import com.pegasus.payload.BuildConfig
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.tasks.await
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * Firebase Auth helper for dual-mode authentication.
 * Uses google-services.json; Auth emulator only when local.properties
 * sets firebase.auth.emulator=true.
 * All methods degrade gracefully — legacy JWT still works when Firebase is unavailable.
 */
object FirebaseAuthHelper {
    private const val TAG = "FirebaseAuth"
    private var initialized = false

    @Volatile
    private var pendingVerificationId: String? = null

    @Volatile
    private var autoCredential: PhoneAuthCredential? = null

    fun init(context: Context) {
        if (initialized) return
        try {
            if (FirebaseApp.getApps(context).isEmpty()) {
                FirebaseApp.initializeApp(context)
            }
            if (BuildConfig.FIREBASE_AUTH_EMULATOR) {
                FirebaseAuth.getInstance().useEmulator(BuildConfig.FIREBASE_AUTH_EMULATOR_HOST, 9099)
            }
            initialized = true
            Log.d(
                TAG,
                "Firebase Auth initialized (emulator=${BuildConfig.FIREBASE_AUTH_EMULATOR})",
            )
        } catch (e: Exception) {
            if (!BuildConfig.DEBUG) {
                throw IllegalStateException("Firebase configuration is missing in release build", e)
            }
            Log.w(TAG, "Firebase Auth init failed (non-fatal in dev): ${e.message}")
        }
    }

    suspend fun sendPhoneVerification(activity: Activity, phone: String): Unit =
        suspendCancellableCoroutine { cont ->
            val normalized = phone.trim()
            if (normalized.isBlank()) {
                cont.resumeWithException(IllegalArgumentException("Phone number required"))
                return@suspendCancellableCoroutine
            }
            val callbacks = object : PhoneAuthProvider.OnVerificationStateChangedCallbacks() {
                override fun onVerificationCompleted(credential: PhoneAuthCredential) {
                    pendingVerificationId = null
                    autoCredential = credential
                    if (cont.isActive) cont.resume(Unit)
                }

                override fun onVerificationFailed(e: FirebaseException) {
                    pendingVerificationId = null
                    Log.w(TAG, "Phone verification failed: ${e.message}")
                    if (cont.isActive) cont.resumeWithException(e)
                }

                override fun onCodeSent(verificationId: String, token: PhoneAuthProvider.ForceResendingToken) {
                    pendingVerificationId = verificationId
                    if (cont.isActive) cont.resume(Unit)
                }
            }
            val options = PhoneAuthOptions.newBuilder(FirebaseAuth.getInstance())
                .setPhoneNumber(normalized)
                .setTimeout(60L, TimeUnit.SECONDS)
                .setActivity(activity)
                .setCallbacks(callbacks)
                .build()
            PhoneAuthProvider.verifyPhoneNumber(options)
        }

    suspend fun verifySmsCode(code: String): String {
        val credential = autoCredential ?: run {
            val verificationId = pendingVerificationId
                ?: throw IllegalStateException("No verification in progress; request a code first")
            PhoneAuthProvider.getCredential(verificationId, code.trim())
        }
        autoCredential = null
        pendingVerificationId = null
        val result = FirebaseAuth.getInstance().signInWithCredential(credential).await()
        return result.user?.getIdToken(false)?.await()?.token
            ?: throw IllegalStateException("Firebase sign-in succeeded but id_token was missing")
    }

    fun hasAutoCredential(): Boolean = autoCredential != null

    suspend fun currentIdToken(): String? {
        return try {
            FirebaseAuth.getInstance().currentUser?.getIdToken(false)?.await()?.token
        } catch (e: Exception) {
            Log.w(TAG, "currentIdToken failed (non-fatal): ${e.message}")
            null
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
        pendingVerificationId = null
        autoCredential = null
        try {
            FirebaseAuth.getInstance().signOut()
        } catch (_: Exception) {
        }
    }
}
