package com.pegasusx.driver.offline

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DriverOfflineActionCatalogTest {

    @Test
    fun priority_ordersProtocolFlush() {
        assertTrue(
            DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_PROXIMITY)
                < DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_SHOP_CLOSED),
        )
        assertTrue(
            DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_DELIVER)
                < DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_COLLECT_CASH),
        )
        assertEquals(10, DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_PROXIMITY))
        assertEquals(20, DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_DELIVER))
        assertEquals(30, DriverOfflineActionCatalog.priorityFor(DriverOfflineActionCatalog.ENDPOINT_CREDIT))
    }

    @Test
    fun httpClassification_poisonVsRetry() {
        assertTrue(DriverOfflineActionCatalog.isSuccessHttp(200))
        assertFalse(DriverOfflineActionCatalog.isSuccessHttp(409))
        assertTrue(DriverOfflineActionCatalog.isConflictHttp(409))
        assertFalse(DriverOfflineActionCatalog.isConflictHttp(200))
        assertTrue(DriverOfflineActionCatalog.isRetryableHttp(500))
        assertTrue(DriverOfflineActionCatalog.isRetryableHttp(429))
        assertFalse(DriverOfflineActionCatalog.isRetryableHttp(400))
        assertFalse(DriverOfflineActionCatalog.isSuccessHttp(400))
        assertFalse(DriverOfflineActionCatalog.isSuccessHttp(422))
        assertFalse(DriverOfflineActionCatalog.isSuccessHttp(503))
    }

    @Test
    fun idempotentReplay409_distinguishesDuplicateFromConflict() {
        // Idempotent duplicate replay markers
        assertTrue(DriverOfflineActionCatalog.isIdempotentReplay409(409, "{\"error\":\"duplicate_mutation\"}"))
        assertTrue(DriverOfflineActionCatalog.isIdempotentReplay409(409, "{\"code\":\"already_exists\"}"))
        assertTrue(DriverOfflineActionCatalog.isIdempotentReplay409(409, "idempotent transaction already recorded"))
        assertTrue(DriverOfflineActionCatalog.isIdempotentReplay409(409, "{\"status\":\"already_processed\"}"))

        // Business state conflicts
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(409, "{\"error\":\"order_cancelled\"}"))
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(409, "{\"error\":\"invalid_state_transition\"}"))
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(409, "{\"error\":\"manifest_reassigned\"}"))
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(409, ""))
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(409, null))

        // Non-409 codes
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(200, "duplicate"))
        assertFalse(DriverOfflineActionCatalog.isIdempotentReplay409(422, "duplicate"))

        // Outcome mapping
        assertEquals(OfflineHttpSemantics.FlushOutcome.ACK, OfflineHttpSemantics.outcomeForHttp(200))
        assertEquals(OfflineHttpSemantics.FlushOutcome.ACK, OfflineHttpSemantics.outcomeForHttp(409, "idempotent duplicate"))
        assertEquals(OfflineHttpSemantics.FlushOutcome.DEAD, OfflineHttpSemantics.outcomeForHttp(409, "order_cancelled"))
        assertEquals(OfflineHttpSemantics.FlushOutcome.RETRY, OfflineHttpSemantics.outcomeForHttp(503))
        assertEquals(OfflineHttpSemantics.FlushOutcome.DEAD, OfflineHttpSemantics.outcomeForHttp(422))
    }

    @Test
    fun offlineEligible_coversDoorstepAndSettlement() {
        assertTrue(DriverOfflineActionCatalog.isOfflineEligible("v1/delivery/shop-closed"))
        assertTrue(DriverOfflineActionCatalog.isOfflineEligible("/v1/order/deliver"))
        assertTrue(DriverOfflineActionCatalog.isOfflineEligible("v1/order/collect-cash"))
        assertFalse(DriverOfflineActionCatalog.isOfflineEligible("v1/auth/driver/login"))
        assertFalse(DriverOfflineActionCatalog.isOfflineEligible("v1/fleet/orders"))
    }
}
