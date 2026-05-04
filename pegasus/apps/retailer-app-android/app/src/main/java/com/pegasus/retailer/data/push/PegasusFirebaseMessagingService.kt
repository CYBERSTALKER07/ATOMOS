package com.pegasus.retailer.data.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.pegasus.retailer.R
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.local.TokenManager
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import javax.inject.Inject

@AndroidEntryPoint
class PegasusFirebaseMessagingService : FirebaseMessagingService() {

    @Inject lateinit var api: PegasusApi
    @Inject lateinit var tokenManager: TokenManager

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        Log.d("FCM", "New token: $token")
        scope.launch {
            try {
                api.registerDeviceToken(mapOf("token" to token, "platform" to "android"))
            } catch (e: Exception) {
                Log.e("FCM", "Failed to register device token", e)
            }
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        super.onMessageReceived(message)
        val data = message.data
        Log.d("FCM", "Data message: $data")

        val type = data["type"] ?: return
        val title = data["title"] ?: typeTitle(type)
        val body = data["body"] ?: typeBody(type, data)

        showLocalNotification(title = title, body = body)
    }

    private fun typeTitle(type: String): String = when (type) {
        "ORDER_DISPATCHED" -> "Order Dispatched"
        "DRIVER_APPROACHING" -> "Delivery Arriving"
        "DRIVER_ARRIVED" -> "Driver Has Arrived"
        "ORDER_STATUS_CHANGED" -> "Order Status Updated"
        "PAYMENT_SETTLED", "GLOBAL_PAYNT_SETTLED" -> "Payment Received"
        "PAYMENT_FAILED", "PAYMENT_EXPIRED", "GLOBAL_PAYNT_FAILED", "GLOBAL_PAYNT_EXPIRED" -> "Payment Failed"
        "ORDER_COMPLETED" -> "Order Completed"
        else -> "Notification"
    }

    private fun typeBody(type: String, data: Map<String, String>): String {
        val orderId = data["order_id"]?.takeLast(6) ?: ""
        return when (type) {
            "ORDER_DISPATCHED" -> "Your order #$orderId has been dispatched"
            "DRIVER_APPROACHING" -> "Driver from ${data["supplier_name"] ?: "a supplier"} is approaching"
            "DRIVER_ARRIVED" -> "Driver has arrived for order #$orderId"
            "ORDER_STATUS_CHANGED" -> "Order #$orderId is now ${data["new_state"] ?: "updated"}"
            "PAYMENT_SETTLED", "GLOBAL_PAYNT_SETTLED" -> "Payment confirmed for order #$orderId"
            "PAYMENT_FAILED", "GLOBAL_PAYNT_FAILED" -> "Payment failed for order #$orderId"
            "PAYMENT_EXPIRED", "GLOBAL_PAYNT_EXPIRED" -> "Payment session expired for order #$orderId"
            "ORDER_COMPLETED" -> "Order #$orderId was completed"
            else -> "You have a new notification"
        }
    }

    private fun showLocalNotification(title: String, body: String) {
        val channelId = "deliveries"
        val nm = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            nm.createNotificationChannel(
                NotificationChannel(channelId, "Deliveries", NotificationManager.IMPORTANCE_HIGH)
            )
        }
        val notification = NotificationCompat.Builder(this, channelId)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .build()
        nm.notify(System.currentTimeMillis().toInt(), notification)
    }
}
