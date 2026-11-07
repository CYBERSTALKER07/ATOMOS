package com.pegasusx.factory.data.push

import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import com.pegasusx.factory.data.model.DeviceTokenRequest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.TokenHolder
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

/** POSTs FCM registration tokens. Admin Messaging send rejects APNs hex. JWT is HTTP SoT. */
object DeviceTokenRegistrar {
    private const val TAG = "FactoryFCM"
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    fun uploadBestEffort(api: FactoryApi) {
        if (TokenHolder.token.isNullOrBlank()) return
        scope.launch { upload(api, token = null) }
    }

    fun uploadBestEffort(api: FactoryApi, token: String) {
        if (TokenHolder.token.isNullOrBlank()) return
        scope.launch { upload(api, token) }
    }

    private suspend fun upload(api: FactoryApi, token: String?) {
        val fcm = token?.trim().orEmpty().ifEmpty {
            runCatching { FirebaseMessaging.getInstance().token.await().trim() }.getOrElse { "" }
        }
        if (fcm.isEmpty()) return
        runCatching {
            api.registerDeviceToken(DeviceTokenRequest(token = fcm, platform = "android"))
        }.onFailure { Log.e(TAG, "registerDeviceToken failed", it) }
    }
}
