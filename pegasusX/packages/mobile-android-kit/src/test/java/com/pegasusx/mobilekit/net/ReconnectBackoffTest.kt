package com.pegasusx.mobilekit.net

import org.junit.Assert.assertTrue
import org.junit.Test

class ReconnectBackoffTest {

    @Test
    fun delayCapsAtMaxEvenForHighAttemptCounts() {
        val maxMs = 60_000L
        for (attempt in listOf(0, 5, 10, 50, 100, 1_000)) {
            val delay = ReconnectBackoff.delayMs(attempt, baseMs = 5_000L, maxMs = maxMs)
            assertTrue(
                "attempt=$attempt delay=$delay should be <= ${maxMs + maxMs / 2}",
                delay <= maxMs + maxMs / 2 + 1,
            )
            assertTrue(delay > 0)
        }
    }
}
