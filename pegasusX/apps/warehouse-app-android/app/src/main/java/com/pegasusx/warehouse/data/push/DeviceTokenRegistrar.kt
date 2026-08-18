package com.pegasusx.warehouse.data.push

import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import com.pegasusx.warehouse.data.model.DeviceTokenRequest
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

/** POSTs FCM registration tokens. Admin Messaging send rejects APNs hex. JWT is HTTP SoT. */
object DeviceTokenRegistrar {
    private const val TAG = "WarehouseFCM"
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    fun uploadBestEffort(api: WarehouseApi) {
        if (TokenHolder.token.isNullOrBlank()) return
        scope.launch { upload(api, token = null) }
    }

    fun uploadBestEffort(api: WarehouseApi, token: String) {
        if (TokenHolder.token.isNullOrBlank()) return
        scope.launch { upload(api, token) }
    }

    private suspend fun upload(api: WarehouseApi, token: String?) {
        val fcm = token?.trim().orEmpty().ifEmpty {
            runCatching { FirebaseMessaging.getInstance().token.await().trim() }.getOrElse { "" }
        }
        if (fcm.isEmpty()) return
        runCatching {
            api.registerDeviceToken(DeviceTokenRequest(token = fcm, platform = "android"))
        }.onFailure { Log.e(TAG, "registerDeviceToken failed", it) }
    }
}
