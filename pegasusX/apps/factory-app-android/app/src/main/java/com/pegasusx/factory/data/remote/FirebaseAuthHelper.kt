package com.pegasusx.factory.data.remote

import android.app.Activity
import android.content.Context
import android.util.Log
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseException
import com.google.firebase.FirebaseOptions
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.auth.PhoneAuthCredential
import com.google.firebase.auth.PhoneAuthOptions
import com.google.firebase.auth.PhoneAuthProvider
import com.pegasusx.factory.BuildConfig
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.tasks.await
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * Firebase Auth helper for factory phone OTP login.
 * Connects to Firebase Auth Emulator in debug builds.
 */
object FirebaseAuthHelper {
    private const val TAG = "FactoryFirebaseAuth"
    private var initialized = false

    @Volatile
    private var pendingVerificationId: String? = null

    @Volatile
    private var autoCredential: PhoneAuthCredential? = null

    fun init(context: Context) {
        if (initialized) return
        try {
            if (FirebaseApp.getApps(context).isEmpty()) {
                val options = FirebaseOptions.Builder()
                    .setProjectId("demo-pegasus")
                    .setApplicationId("1:000000000000:android:0000000000000000")
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

    suspend fun sendPhoneVerification(activity: Activity, phone: String): Unit =
        suspendCancellableCoroutine { cont ->
            init(activity)
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

    fun resetFlow() {
        pendingVerificationId = null
        autoCredential = null
    }

    fun signOut() {
        resetFlow()
        try {
            FirebaseAuth.getInstance().signOut()
        } catch (_: Exception) {
        }
    }
}
