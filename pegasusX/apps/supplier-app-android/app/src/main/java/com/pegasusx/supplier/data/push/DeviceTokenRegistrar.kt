package com.pegasusx.supplier.data.push

import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import com.pegasusx.supplier.data.model.DeviceTokenRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.TokenHolder
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

/** POSTs FCM registration tokens. Admin Messaging send rejects APNs hex. JWT is HTTP SoT. */
object DeviceTokenRegistrar {
    private const val TAG = "SupplierFCM"
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    fun uploadBestEffort(api: SupplierApi) {
        if (TokenHolder.token.isNullOrBlank()) return
        scope.launch { upload(api, token = null) }
    }

    fun uploadBestEffort(api: SupplierApi, token: String) {
        if (TokenHolder.token.isNullOrBlank()) return
        scope.launch { upload(api, token) }
    }

    private suspend fun upload(api: SupplierApi, token: String?) {
        val fcm = token?.trim().orEmpty().ifEmpty {
            runCatching { FirebaseMessaging.getInstance().token.await().trim() }.getOrElse { "" }
        }
        if (fcm.isEmpty()) return
        runCatching {
            api.registerDeviceToken(DeviceTokenRequest(token = fcm, platform = "android"))
        }.onFailure { Log.e(TAG, "registerDeviceToken failed", it) }
    }
}
