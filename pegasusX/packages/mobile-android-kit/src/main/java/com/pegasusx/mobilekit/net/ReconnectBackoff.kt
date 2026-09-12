package com.pegasusx.mobilekit.net

import kotlin.math.min
import kotlin.random.Random

/** Exponential backoff + jitter for WS / telemetry reconnect (§8.8). */
object ReconnectBackoff {
    fun delayMs(
        attempt: Int,
        baseMs: Long = 2_000L,
        maxMs: Long = 60_000L,
        retryAfterMs: Long? = null,
    ): Long {
        val capped = attempt.coerceIn(0, 10)
        val exp = min(baseMs * (1L shl capped), maxMs)
        val jittered = exp + Random.nextLong(0, exp / 2 + 1)
        return maxOf(jittered, retryAfterMs ?: 0L)
    }

    fun retryAfterHeaderSeconds(raw: String?): Long? =
        raw?.toLongOrNull()?.times(1_000)
}
