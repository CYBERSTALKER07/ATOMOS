package com.pegasusx.warehouse.data.remote

import kotlin.math.min
import kotlin.random.Random
import okhttp3.Response

object ReconnectBackoff {
    fun delayMs(
        attempt: Int,
        baseMs: Long = 1_000L,
        maxMs: Long = 30_000L,
        retryAfterMs: Long? = null,
    ): Long {
        val capped = (attempt - 1).coerceIn(0, 4)
        val exp = min(baseMs * (1L shl capped), maxMs)
        val jittered = exp + Random.nextLong(0, exp / 2 + 1)
        return maxOf(jittered, retryAfterMs ?: 0L)
    }

    fun retryAfterMs(response: Response?): Long? {
        val raw = response?.header("Retry-After") ?: return null
        return raw.toLongOrNull()?.times(1_000)
    }
}
