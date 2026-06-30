package com.pegasus.driver.util

import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager

object BatteryGuard {
    const val WARN_THRESHOLD = 20
    const val BLOCK_THRESHOLD = 10

    enum class DepartGate { OK, WARN_LOW, BLOCK_CRITICAL }

    fun currentLevelPercent(context: Context): Int {
        val batteryStatus = context.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val level = batteryStatus?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
        val scale = batteryStatus?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
        if (level < 0 || scale <= 0) return 100
        return (level * 100) / scale
    }

    fun departGate(context: Context): DepartGate {
        val level = currentLevelPercent(context)
        return when {
            level < BLOCK_THRESHOLD -> DepartGate.BLOCK_CRITICAL
            level < WARN_THRESHOLD -> DepartGate.WARN_LOW
            else -> DepartGate.OK
        }
    }
}
