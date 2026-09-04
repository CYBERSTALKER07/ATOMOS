package com.pegasusx.driver.data.remote

import com.pegasusx.mobilekit.net.ReconnectBackoff as KitReconnectBackoff
import okhttp3.Response

/** Desert Protocol reconnect scheduling — delegates to mobile-android-kit. */
object ReconnectBackoff {
    fun delayMs(
        attempt: Int,
        baseMs: Long = 2_000L,
        maxMs: Long = 60_000L,
        retryAfterMs: Long? = null,
    ): Long = KitReconnectBackoff.delayMs(attempt, baseMs, maxMs, retryAfterMs)

    fun retryAfterMs(response: Response?): Long? {
        val raw = response?.header("Retry-After") ?: return null
        return KitReconnectBackoff.retryAfterHeaderSeconds(raw)
    }
}
