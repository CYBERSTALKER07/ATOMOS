package com.pegasus.retailer

import com.pegasus.design.ORDER_STATUS_FUNNEL
import com.pegasus.design.statusStackModel
import com.pegasusx.retailer.data.model.ControlTowerPulse
import com.pegasusx.retailer.ui.retailerOrderMatchesCommand
import com.pegasusx.retailer.data.model.RetailerDetailedAnalytics
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardTests {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testControlTowerPulseFiscalFailedFacet() {
        val raw = """
            {
              "source": "spanner",
              "empty": false,
              "open_orders": 1,
              "loyalty": { "enrolled": false },
              "orders_by_status": { "FISCAL_FAILED": 1, "COMPLETED": 1 },
              "orders_by_supplier": [
                { "supplier_id": "sup-a", "orders_by_status": { "FISCAL_FAILED": 1 } },
                { "supplier_id": "sup-b", "orders_by_status": { "COMPLETED": 1 } }
              ]
            }
        """.trimIndent()
        val pulse = json.decodeFromString(ControlTowerPulse.serializer(), raw)
        assertEquals("spanner", pulse.source)
        assertFalse(pulse.empty)
        assertEquals(1, pulse.ordersByStatus["FISCAL_FAILED"])
        assertEquals(1, pulse.ordersBySupplier.first { it.supplierId == "sup-a" }.ordersByStatus["FISCAL_FAILED"])
        assertEquals(1, pulse.ordersBySupplier.first { it.supplierId == "sup-b" }.ordersByStatus["COMPLETED"])
        assertFalse(pulse.loyalty.enrolled)
        assertTrue(ORDER_STATUS_FUNNEL.contains("FISCAL_FAILED"))
        assertEquals(17, ORDER_STATUS_FUNNEL.size)
        assertEquals(17, statusStackModel(ORDER_STATUS_FUNNEL, pulse.ordersByStatus).rows.size)
    }

    @Test
    fun testRetailerDetailedAnalyticsSerialization() {
        val rawJson = """
            {
                "total_spent": 1500000,
                "total_orders": 15,
                "avg_order_value": 100000,
                "orders_by_state": [
                    {
                        "state": "PENDING",
                        "count": 2
                    }
                ]
            }
        """.trimIndent()

        val analytics = json.decodeFromString<RetailerDetailedAnalytics>(rawJson)
        assertEquals(1500000L, analytics.totalSpent)
        assertEquals(15L, analytics.totalOrders)
        assertEquals(100000L, analytics.avgOrderValue)
        assertEquals(1, analytics.ordersByState.size)
        assertEquals("PENDING", analytics.ordersByState.first().state)
        assertEquals(2L, analytics.ordersByState.first().count)
    }

    @Test
    fun testCommandChipMatchesCanonicalStatusAndSupplier() {
        assertTrue(retailerOrderMatchesCommand("FISCAL_FAILED", "sup-a", "FISCAL_FAILED", null))
        assertTrue(retailerOrderMatchesCommand("DISPATCHED", "sup-a", "LOADED", "sup-a"))
        assertFalse(retailerOrderMatchesCommand("FISCAL_FAILED", "sup-a", "FISCAL_FAILED", "sup-b"))
    }
}
