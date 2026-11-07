package com.pegasusx.driver.data.remote

import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * §8.8: reconnect must never hard-stop — backoff stays capped at max delay.
 */
class ReconnectBackoffTest {

    @Test
    fun delayCapsAtMaxEvenForHighAttemptCounts() {
        val maxMs = 60_000L
        // Kit caps attempt index at 10 for exponent; still must stay ≤ maxMs (+ jitter ≤ maxMs + maxMs/2).
        for (attempt in listOf(0, 5, 10, 50, 100, 1_000)) {
            val delay = ReconnectBackoff.delayMs(attempt, baseMs = 5_000L, maxMs = maxMs)
            assertTrue(
                "attempt=$attempt delay=$delay should be <= ${maxMs + maxMs / 2}",
                delay <= maxMs + maxMs / 2 + 1,
            )
            assertTrue("attempt=$attempt delay=$delay should be > 0", delay > 0)
        }
    }

    @Test
    fun highAttemptsStillProduceFiniteDelay() {
        // Regression: previously MAX_RECONNECT_ATTEMPTS=10 aborted forever.
        val d10 = ReconnectBackoff.delayMs(10, baseMs = 2_000L, maxMs = 60_000L)
        val d100 = ReconnectBackoff.delayMs(100, baseMs = 2_000L, maxMs = 60_000L)
        assertTrue(d10 in 1..(60_000L + 30_000L))
        assertTrue(d100 in 1..(60_000L + 30_000L))
    }
}
