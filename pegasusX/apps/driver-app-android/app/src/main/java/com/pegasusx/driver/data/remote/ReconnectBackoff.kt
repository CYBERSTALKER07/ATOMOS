package com.pegasusx.driver.data.remote

import okhttp3.Response
import kotlin.math.min
import kotlin.random.Random

/** Desert Protocol reconnect scheduling — local self-contained implementation. */
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

    fun retryAfterMs(response: Response?): Long? {
        val raw = response?.header("Retry-After") ?: return null
        return retryAfterHeaderSeconds(raw)
    }
}
