package com.pegasus.payload.services

import android.app.PendingIntent
import android.content.Intent
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.pegasus.payload.MainActivity
import com.pegasus.payload.R
import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.repository.PayloadRepository
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import javax.inject.Inject
import kotlin.random.Random

/**
 * Receives FCM messages while the app is backgrounded/foreground and forwards
 * token refreshes to `POST /v1/user/device-token`. Foreground display for
 * data-only messages is handled here; the WebSocket path delivers in-app
 * notifications when the socket is open. Tap-action deep-links into the
 * notifications panel via [NotificationBus.EXTRA_OPEN_NOTIFICATIONS].
 */
@AndroidEntryPoint
class PayloadFirebaseMessagingService : FirebaseMessagingService() {

    @Inject lateinit var secureStore: SecureStore
    @Inject lateinit var repo: PayloadRepository

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onNewToken(token: String) {
        secureStore.firebaseToken = token
        if (secureStore.token.isNullOrEmpty()) return
        scope.launch { runCatching { repo.registerDeviceToken(token) } }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val title = message.notification?.title
            ?: message.data["title"]
            ?: return
        val body = message.notification?.body ?: message.data["body"].orEmpty()
        showNotification(title, body)
    }

    private fun showNotification(title: String, body: String) {
        val tapIntent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP
            putExtra(NotificationBus.EXTRA_OPEN_NOTIFICATIONS, true)
        }
        val pi = PendingIntent.getActivity(
            this,
            Random.nextInt(),
            tapIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(this, NotificationBus.CHANNEL_ID)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setAutoCancel(true)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setContentIntent(pi)
            .build()
        runCatching {
            NotificationManagerCompat.from(this).notify(Random.nextInt(), notification)
        }
    }
}

