package com.pegasusx.factory.data.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.pegasusx.factory.R
import com.pegasusx.factory.data.remote.FactoryApi
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

/** Receives FCM and refreshes `POST /v1/user/device-token` when a pegasus JWT exists. */
@AndroidEntryPoint
class FactoryFirebaseMessagingService : FirebaseMessagingService() {

    @Inject lateinit var api: FactoryApi

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        DeviceTokenRegistrar.uploadBestEffort(api, token)
    }

    override fun onMessageReceived(message: RemoteMessage) {
        super.onMessageReceived(message)
        val title = message.data["title"] ?: message.notification?.title ?: "Pegasus Factory"
        val body = message.data["body"] ?: message.notification?.body ?: message.data["type"].orEmpty()
        if (body.isBlank()) return
        showLocalNotification(title, body)
    }

    private fun showLocalNotification(title: String, body: String) {
        val channelId = "pegasus_factory_ops"
        val nm = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            nm.createNotificationChannel(
                NotificationChannel(channelId, "Factory ops", NotificationManager.IMPORTANCE_HIGH),
            )
        }
        val notification = androidx.core.app.NotificationCompat.Builder(this, channelId)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .build()
        nm.notify((System.currentTimeMillis() % Int.MAX_VALUE).toInt(), notification)
    }
}
