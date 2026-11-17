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
        val safeBase = baseMs.coerceAtLeast(100L)
        val safeMax = maxMs.coerceAtLeast(safeBase)
        val capped = attempt.coerceIn(0, 10)
        val exp = min(safeBase * (1L shl capped), safeMax)
        val jitterRange = maxOf(1L, exp / 2)
        val jittered = min(exp + Random.nextLong(0, jitterRange), safeMax)
        return maxOf(jittered, retryAfterMs ?: 0L)
    }

    fun retryAfterHeaderSeconds(raw: String?): Long? =
        raw?.toLongOrNull()?.times(1_000)
}
