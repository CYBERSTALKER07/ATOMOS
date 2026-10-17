package com.pegasusx.driver.services

import android.content.Context

/**
 * Persists whether transit telemetry should resume after process death / reboot (§8.8).
 */
object TelemetryTrackingPrefs {
    private const val PREFS = "driver_telemetry_tracking"
    private const val KEY_ACTIVE = "telemetry_tracking_active"

    fun setActive(context: Context, active: Boolean) {
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY_ACTIVE, active)
            .apply()
    }

    fun isActive(context: Context): Boolean =
        context.applicationContext
            .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .getBoolean(KEY_ACTIVE, false)
}
