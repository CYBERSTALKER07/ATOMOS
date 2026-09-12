package com.pegasusx.driver.services

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import com.pegasusx.driver.data.remote.TokenHolder

/**
 * Resumes transit telemetry after device reboot when the driver had active tracking (§8.8).
 */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent?) {
        val action = intent?.action ?: return
        if (action != Intent.ACTION_BOOT_COMPLETED) {
            return
        }
        if (!TelemetryTrackingPrefs.isActive(context)) {
            Log.d(TAG, "Boot: telemetry not active — skip")
            return
        }
        val token = TokenHolder.token
        if (token.isNullOrBlank()) {
            Log.w(TAG, "Boot: tracking flagged but no auth token — skip")
            return
        }
        val serviceIntent = Intent(context, TelemetryService::class.java).apply {
            this.action = TelemetryService.ACTION_START
        }
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(serviceIntent)
            } else {
                context.startService(serviceIntent)
            }
            Log.d(TAG, "Boot: resumed TelemetryService")
        } catch (e: Exception) {
            Log.e(TAG, "Boot: failed to start TelemetryService", e)
        }
    }

    companion object {
        private const val TAG = "BootReceiver"
    }
}
