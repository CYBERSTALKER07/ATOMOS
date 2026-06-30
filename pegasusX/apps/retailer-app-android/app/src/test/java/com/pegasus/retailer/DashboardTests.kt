package com.pegasus.retailer

import com.pegasusx.retailer.data.model.RetailerDetailedAnalytics
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test

class DashboardTests {
    private val json = Json { ignoreUnknownKeys = true }

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
}
