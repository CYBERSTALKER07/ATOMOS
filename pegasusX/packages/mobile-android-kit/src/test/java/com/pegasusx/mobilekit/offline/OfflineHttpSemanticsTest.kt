package com.pegasusx.mobilekit.offline

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class OfflineHttpSemanticsTest {
    @Test
    fun successIncludesConflict() {
        assertTrue(OfflineHttpSemantics.isSuccessHttp(200))
        assertTrue(OfflineHttpSemantics.isSuccessHttp(409))
        assertEquals(OfflineHttpSemantics.FlushOutcome.ACK, OfflineHttpSemantics.outcomeForHttp(409))
    }

    @Test
    fun retryableServerErrors() {
        assertTrue(OfflineHttpSemantics.isRetryableHttp(503))
        assertTrue(OfflineHttpSemantics.isRetryableHttp(429))
        assertEquals(OfflineHttpSemantics.FlushOutcome.RETRY, OfflineHttpSemantics.outcomeForHttp(500))
    }

    @Test
    fun clientErrorsAreDead() {
        assertTrue(OfflineHttpSemantics.isDeadHttp(422))
        assertFalse(OfflineHttpSemantics.isDeadHttp(409))
        assertEquals(OfflineHttpSemantics.FlushOutcome.DEAD, OfflineHttpSemantics.outcomeForHttp(400))
    }

    @Test
    fun normalizeStripsPrefixes() {
        assertEquals("v1/order/deliver", OfflineHttpSemantics.normalizeEndpoint("/api/v1/order/deliver"))
    }
}
