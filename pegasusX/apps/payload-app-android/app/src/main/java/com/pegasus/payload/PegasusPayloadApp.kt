package com.pegasus.payload

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import com.pegasus.payload.services.NotificationBus
import dagger.hilt.android.HiltAndroidApp

/**
 * PegasusPayloadApp — Hilt application root for the native iPad/Android Payload Terminal.
 *
 * Mirrors the role-app sibling pattern of [com.pegasus.driver.PegasusDriverApp].
 * All long-lived singletons (Retrofit, OkHttp, Room, WebSocket, OfflineQueue,
 * NotificationsHub) are bound via Hilt modules under [com.pegasus.payload.di].
 */
@HiltAndroidApp
class PegasusPayloadApp : Application() {
    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val mgr = getSystemService(NotificationManager::class.java) ?: return
        if (mgr.getNotificationChannel(NotificationBus.CHANNEL_ID) != null) return
        mgr.createNotificationChannel(
            NotificationChannel(
                NotificationBus.CHANNEL_ID,
                NotificationBus.CHANNEL_NAME,
                NotificationManager.IMPORTANCE_HIGH,
            ).apply {
                description = "Load-out, dispatch, and exception alerts"
                enableVibration(true)
            },
        )
    }
}

