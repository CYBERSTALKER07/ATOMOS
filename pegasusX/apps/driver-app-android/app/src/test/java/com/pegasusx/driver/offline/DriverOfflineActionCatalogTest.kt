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
        assertTrue(DriverOfflineActionCatalog.isSuccessHttp(409))
        assertTrue(DriverOfflineActionCatalog.isRetryableHttp(500))
        assertTrue(DriverOfflineActionCatalog.isRetryableHttp(429))
        assertFalse(DriverOfflineActionCatalog.isRetryableHttp(400))
        assertFalse(DriverOfflineActionCatalog.isSuccessHttp(400))
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
